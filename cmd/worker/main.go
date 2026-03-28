// cmd/worker/main.go
package main

import (
	"log"

	"github.com/hibiken/asynq"
	"github.com/sdnxshu/cascoon/internal/queue"
	"github.com/sdnxshu/cascoon/internal/worker"
	"github.com/sdnxshu/cascoon/pkg/db"
)

func main() {
	if err := db.Init(); err != nil {
		log.Fatalf("db init: %v", err)
	}

	srv := asynq.NewServer(
		queue.RedisOpt(),
		asynq.Config{Concurrency: 10},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc("run", worker.HandleTask)

	if err := srv.Run(mux); err != nil {
		log.Fatal(err)
	}
}
