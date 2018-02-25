package main

import (
	"flag"

	"github.com/gin-gonic/gin"
	"github.com/kpacha/load-test/db"
)

func main() {
	storePath := flag.String("f", ".", "path to use as store")
	flag.Parse()

	store := db.NewFS(*storePath)
	server := Server{
		Engine:   gin.New(),
		DB:       store,
		Executor: NewExecutor(store),
	}
	server.Run()
}
