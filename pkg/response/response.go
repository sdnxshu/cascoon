// pkg/response/response.go
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func JSON(c *gin.Context, status int, payload interface{}) {
	c.JSON(status, payload)
}

func OK(c *gin.Context, payload interface{}) {
	c.JSON(http.StatusOK, payload)
}
