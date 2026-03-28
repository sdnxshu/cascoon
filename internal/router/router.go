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

	r.Use(gin.Recovery())
	r.Use(middleware.LoggerMiddleware())

	r.GET("/", handlers.RootHandler)
	r.GET("/health", handlers.HealthHandler)

	// /run has its own longer timeout since it hits Redis
	run := r.Group("/")
	run.Use(middleware.TimeoutMiddleware(5 * time.Second))
	run.POST("/run", handlers.RunHandler)

	// Runs - no tight timeout needed, just DB reads
	runs := r.Group("/runs")
	runs.Use(middleware.TimeoutMiddleware(10 * time.Second))
	runs.GET("", handlers.ListRunsHandler)
	runs.GET("/:id", handlers.GetRunHandler)

	return r
}
