package queue

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hibiken/asynq"
)

const (
	TypeRunWorkflow  = "workflow:run"
	DefaultRedisAddr = "redis:6379"
)

// TaskPayload is the data attached to every workflow:run task.
type TaskPayload struct {
	Repo string `json:"repo"`
}

// Client wraps asynq.Client so the rest of the app stays decoupled from asynq.
type Client struct {
	inner *asynq.Client
}

// NewClient creates a queue client. Set REDIS_ADDR env var to override the default.
func NewClient() *Client {
	addr := redisAddr()
	return &Client{inner: asynq.NewClient(asynq.RedisClientOpt{Addr: addr})}
}

func (c *Client) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	return c.inner.Enqueue(task, opts...)
}

func (c *Client) Close() error {
	return c.inner.Close()
}

// NewTask creates an asynq.Task for running workflows against a repo.
func NewTask(repo string) *asynq.Task {
	payload, _ := json.Marshal(TaskPayload{Repo: repo})
	return asynq.NewTask(TypeRunWorkflow, payload)
}

// ParsePayload extracts the TaskPayload from a raw asynq.Task.
func ParsePayload(task *asynq.Task) (*TaskPayload, error) {
	var p TaskPayload
	if err := json.Unmarshal(task.Payload(), &p); err != nil {
		return nil, fmt.Errorf("failed to parse task payload: %w", err)
	}
	return &p, nil
}

func redisAddr() string {
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		return addr
	}
	return DefaultRedisAddr
}
