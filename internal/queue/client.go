// internal/queue/client.go
package queue

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

func NewTask(repo string) *asynq.Task {
	payload, _ := json.Marshal(map[string]string{
		"repo": repo,
	})

	return asynq.NewTask("run", payload)
}

func NewClient() *asynq.Client {
	return asynq.NewClient(asynq.RedisClientOpt{
		Addr: "redis:6379",
	})
}
