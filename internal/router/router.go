package router

import (
	"github.com/sdnxshu/cascoon/internal/handlers"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/", handlers.RootHandler)
	r.GET("/health", handlers.HealthHandler)

	return r
}
