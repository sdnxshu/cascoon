// internal/middleware/logger.go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sdnxshu/cascoon/pkg/logger"
	"go.uber.org/zap"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)

		logger.Log.Info("request",
			// structured logs
			// key-value pairs
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", duration),
			zap.String("ip", c.ClientIP()),
		)
	}
}
