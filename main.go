package main

import (
	"flag"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/kpacha/load-test/db"
)

func main() {
	storePath := flag.String("f", ".", "path to use as store")
	port := flag.Int("p", 7879, "port to expose the html ui")
	flag.Parse()

	store := db.NewFS(*storePath)
	server := Server{
		Engine:   gin.New(),
		DB:       store,
		Executor: NewExecutor(store),
		Addr:     fmt.Sprintf(":%d", *port),
	}
	server.Run()
}
