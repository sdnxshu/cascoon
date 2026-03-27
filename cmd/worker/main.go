// cmd/worker/main.go
package main

import (
	"log"

	"github.com/hibiken/asynq"
	"github.com/sdnxshu/cascoon/internal/worker"
)

func main() {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: "redis:6379"},
		asynq.Config{Concurrency: 10},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc("run", worker.HandleTask)

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
