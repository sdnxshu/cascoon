package queue

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

// const TypeWebhook = "webhook:trigger"

func NewTask(repo string) *asynq.Task {
	payload, _ := json.Marshal(map[string]string{
		"repo": repo,
	})

	return asynq.NewTask("run", payload)
}
