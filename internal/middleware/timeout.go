// internal/middleware/timeout.go
package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		done := make(chan struct{})

		go func() {
			c.Next()
			close(done)
		}()

		select {
		case <-done:
			return
		case <-time.After(timeout):
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
				"error": "request timed out",
			})
		}
	}
}
