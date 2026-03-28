// internal/handlers/runs.go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sdnxshu/cascoon/internal/store"
	"github.com/sdnxshu/cascoon/pkg/db"
)

func ListRunsHandler(c *gin.Context) {
	s := store.NewRunStore(db.DB)
	runs, err := s.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch runs"})
		return
	}
	if runs == nil {
		runs = []*store.Run{}
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs})
}

func GetRunHandler(c *gin.Context) {
	id := c.Param("id")
	s := store.NewRunStore(db.DB)

	run, err := s.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "run not found"})
		return
	}

	logs, err := s.GetLogs(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch logs"})
		return
	}
	if logs == nil {
		logs = []*store.RunLog{}
	}

	c.JSON(http.StatusOK, gin.H{
		"run":  run,
		"logs": logs,
	})
}
