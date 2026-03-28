// internal/router/router.go
package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdnxshu/cascoon/internal/handlers"
	"github.com/sdnxshu/cascoon/internal/middleware"
)

func SetupRouter() *gin.Engine {
	r := gin.New()

	// Middleware
	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.TimeoutMiddleware(3 * time.Second))

	// Routes
	r.GET("/", handlers.RootHandler)

	r.GET("/health", handlers.HealthHandler)

	r.POST("/run", handlers.RunHandler)
	r.GET("/runs", handlers.RunHandler)
	r.GET("/runs/:id", handlers.RunHandler)

	return r
}
