package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/signal"

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
		store = db.NewFS(*storePath)
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
