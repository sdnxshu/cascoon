package main

import (
	"log"

	"github.com/sdnxshu/cascoon/internal/router"
)

func main() {
	r := router.SetupRouter()

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
