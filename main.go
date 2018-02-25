package main

import (
	"github.com/gin-gonic/gin"
	"github.com/kpacha/load-test/db"
)

func main() {
	store := db.NewFS(".")
	server := Server{
		Engine:   gin.New(),
		DB:       store,
		Executor: NewExecutor(store),
	}
	server.Run()
}
