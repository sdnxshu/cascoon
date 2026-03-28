// internal/handlers/run.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdnxshu/cascoon/internal/queue"
	"github.com/sdnxshu/cascoon/internal/store"
	"github.com/sdnxshu/cascoon/pkg/db"
)

type RunRequest struct {
	Repo string `json:"repo" binding:"required"`
}

func RunHandler(c *gin.Context) {
	var body RunRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo is required"})
		return
	}

	// 1. Create a run record in DB with status=pending
	s := store.NewRunStore(db.DB)
	run, err := s.Create(body.Repo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create run"})
		return
	}

	// 2. Enqueue the job with the run ID so the worker can update it
	client := queue.NewClient()
	defer client.Close()

	task := queue.NewTask(run.ID, body.Repo)
	if _, err := client.Enqueue(task); err != nil {
		// Mark run as failed if we can't enqueue
		_ = s.UpdateStatus(run.ID, "failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue task"})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"run_id":  run.ID,
		"repo":    run.Repo,
		"status":  run.Status,
		"message": "Run queued successfully",
	})
}
