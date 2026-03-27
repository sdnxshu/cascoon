// internal/handlers/run.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdnxshu/cascoon/internal/queue"
)

type RequestBody struct {
	Repo string `json:"repo"`
}

func RunHandler(c *gin.Context) {
	var body RequestBody

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	client := queue.NewClient()
	defer client.Close()

	task := queue.NewTask(body.Repo)

	if _, err := client.Enqueue(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to enqueue task"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Queued successfully 🚀",
	})
}
