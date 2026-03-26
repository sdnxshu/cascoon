package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/hibiken/asynq"
	"github.com/sdnxshu/cascoon/internal/handlers"
	"github.com/sdnxshu/cascoon/internal/queue"
)

const workflowsDir = ".jennings/workflows"

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "redis:6379"
	}

	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 4,
			Queues:      map[string]int{"default": 1},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(queue.TypeRunWorkflow, handleRunWorkflow)

	log.Printf("Worker started. Redis: %s", redisAddr)

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down worker...")
		srv.Shutdown()
	}()

	if err := srv.Run(mux); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

// handleRunWorkflow is the asynq handler for TypeRunWorkflow tasks.
func handleRunWorkflow(ctx context.Context, t *asynq.Task) error {
	payload, err := queue.ParsePayload(t)
	if err != nil {
		return err
	}

	log.Printf("[task] Running workflows for repo: %s", payload.Repo)

	workflows, err := handlers.LoadWorkflows(workflowsDir)
	if err != nil {
		return fmt.Errorf("load workflows: %w", err)
	}

	if len(workflows) == 0 {
		log.Printf("[task] No workflows found in %s, nothing to do", workflowsDir)
		return nil
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer dockerClient.Close()

	for _, wf := range workflows {
		log.Printf("[workflow] Starting: %s (image: %s)", wf.Name, wf.Image)

		if err := runWorkflow(ctx, dockerClient, wf, payload.Repo); err != nil {
			// Log the failure but continue running other workflows
			log.Printf("[workflow] FAILED %s: %v", wf.Name, err)
			continue
		}

		log.Printf("[workflow] Completed: %s", wf.Name)
	}

	return nil
}

// runWorkflow pulls the image (if needed) and runs each step sequentially inside Docker.
func runWorkflow(ctx context.Context, dockerClient *client.Client, wf *handlers.Workflow, repo string) error {
	log.Printf("[workflow:%s] Pulling image: %s", wf.Name, wf.Image)

	reader, err := dockerClient.ImagePull(ctx, wf.Image, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", wf.Image, err)
	}
	defer reader.Close()
	// Drain pull output so the pull completes
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return fmt.Errorf("drain image pull: %w", err)
	}

	for i, step := range wf.Steps {
		log.Printf("[workflow:%s] Step %d/%d: %s", wf.Name, i+1, len(wf.Steps), step.Name)

		if err := runStep(ctx, dockerClient, wf, step, repo); err != nil {
			return fmt.Errorf("step %q failed: %w", step.Name, err)
		}
	}

	return nil
}

// runStep runs a single workflow step inside a fresh Docker container.
func runStep(ctx context.Context, dockerClient *client.Client, wf *handlers.Workflow, step handlers.Step, repo string) error {
	// Build env slice: REPO is always injected, then any step-level env vars
	env := []string{fmt.Sprintf("REPO=%s", repo)}
	for k, v := range step.Env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	// Apply per-step timeout if specified
	stepCtx := ctx
	if step.Timeout != "" {
		d, err := time.ParseDuration(step.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout %q: %w", step.Timeout, err)
		}
		var cancel context.CancelFunc
		stepCtx, cancel = context.WithTimeout(ctx, d)
		defer cancel()
	}

	resp, err := dockerClient.ContainerCreate(stepCtx, &container.Config{
		Image: wf.Image,
		Cmd:   []string{"/bin/sh", "-c", step.Run},
		Env:   env,
	}, nil, nil, nil, "")
	if err != nil {
		return fmt.Errorf("create container: %w", err)
	}

	containerID := resp.ID

	// Always remove the container when done
	defer func() {
		_ = dockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
	}()

	if err := dockerClient.ContainerStart(stepCtx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}

	// Stream logs to stdout
	logReader, err := dockerClient.ContainerLogs(stepCtx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return fmt.Errorf("container logs: %w", err)
	}
	defer logReader.Close()

	// stdcopy demuxes Docker's multiplexed stdout/stderr stream
	if _, err := stdcopy.StdCopy(os.Stdout, os.Stderr, logReader); err != nil {
		return fmt.Errorf("stream logs: %w", err)
	}

	// Wait for the container to exit and check its exit code
	statusCh, errCh := dockerClient.ContainerWait(stepCtx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("container wait: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("step exited with code %d", status.StatusCode)
		}
	}

	return nil
}
