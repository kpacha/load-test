package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/gin-gonic/gin"
	"github.com/kpacha/load-test/db"
)

//go:embed templates/browse.html
//go:embed templates/index.html
//go:embed templates/partials.html
var fs embed.FS

func main() {
	storePath := flag.String("f", ".", "path to use as store")
	port := flag.Int("p", 7879, "port to expose the html ui")
	isDevel := flag.Bool("d", false, "devel mode enabled")
	inMemory := flag.Bool("m", false, "use in-memory store instead of the fs persistent one")
	flag.Parse()

	var store db.DB
	if *inMemory {
		store = db.NewInMemory()
	} else {
		s, err := session.NewSession(&aws.Config{Region: aws.String(os.Getenv("S3_REGION"))})
		if err != nil {
			log.Fatal(err)
		}
		store, err = db.NewFS(*storePath, s, os.Getenv("S3_BUCKET"))
		if err != nil {
			log.Fatal(err)
		}
	}

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-quit
		cancel()
	}()

	server, err := NewServer(gin.Default(), store, NewExecutor(store), *isDevel)
	if err != nil {
		fmt.Println("error building the server:", err.Error())
		return
	}

	fmt.Println(server.Run(ctx, fmt.Sprintf(":%d", *port)))
}
