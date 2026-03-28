// internal/queue/client.go
package queue

import (
	"encoding/json"
	"os"

	"github.com/hibiken/asynq"
)

type RunPayload struct {
	RunID string `json:"run_id"`
	Repo  string `json:"repo"`
}

func NewTask(runID, repo string) *asynq.Task {
	payload, _ := json.Marshal(RunPayload{
		RunID: runID,
		Repo:  repo,
	})
	return asynq.NewTask("run", payload)
}

func NewClient() *asynq.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "redis:6379"
	}
	return asynq.NewClient(asynq.RedisClientOpt{Addr: addr})
}

func RedisOpt() asynq.RedisClientOpt {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "redis:6379"
	}
	return asynq.RedisClientOpt{Addr: addr}
}
