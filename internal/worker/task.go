package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hibiken/asynq"
)

type Payload struct {
	Repo string `json:"repo"`
}

func HandleTask(ctx context.Context, t *asynq.Task) error {
	var p Payload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Println("Processing repo:", p.Repo)
	return nil
}
