// internal/worker/task.go
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v6"
	"github.com/hibiken/asynq"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/sdnxshu/cascoon/internal/store"
	"github.com/sdnxshu/cascoon/pkg/db"
)

func RunInContainer(ctx context.Context, runID, repoURL string, workflowName, dockerImage string, commands []string) error {
	s := store.NewRunStore(db.DB)

	repoDir, err := os.MkdirTemp("", "repo-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(repoDir)

	fmt.Printf("Cloning %s into %s\n", repoURL, repoDir)
	_, err = git.PlainClone(repoDir, &git.CloneOptions{
		URL:   repoURL,
		Bare:  false,
		Depth: 1,
	})
	if err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	absRepoDir, err := filepath.Abs(repoDir)
	if err != nil {
		return fmt.Errorf("resolve abs path: %w", err)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer cli.Close()

	fmt.Printf("Pulling image %s\n", dockerImage)
	reader, err := cli.ImagePull(ctx, dockerImage, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("image pull: %w", err)
	}
	io.Copy(io.Discard, reader)
	reader.Close()

	resp, err := cli.ContainerCreate(ctx,
		&container.Config{
			Image:      dockerImage,
			WorkingDir: "/workspace",
			Cmd:        []string{"sleep", "infinity"},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: absRepoDir,
					Target: "/workspace",
				},
			},
		},
		nil, nil, "",
	)
	if err != nil {
		return fmt.Errorf("container create: %w", err)
	}
	containerID := resp.ID

	defer func() {
		fmt.Println("Removing container...")
		cli.ContainerRemove(context.Background(), containerID, container.RemoveOptions{Force: true})
	}()

	if err := cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("container start: %w", err)
	}
	fmt.Printf("Container %s started\n", containerID[:12])

	// Run each step, capture output, save to DB
	for _, command := range commands {
		output, exitCode, err := execInContainer(ctx, cli, containerID, command)

		// Always save the log, even on failure
		_ = s.InsertLog(runID, workflowName, command, output, exitCode)

		if err != nil {
			return fmt.Errorf("exec %q: %w", command, err)
		}
	}

	return nil
}

// execInContainer runs a command and returns (output, exitCode, error)
func execInContainer(ctx context.Context, cli *client.Client, containerID, command string) (string, int, error) {
	fmt.Printf("\n--- Running: %s ---\n", command)

	execID, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/workspace",
	})
	if err != nil {
		return "", -1, fmt.Errorf("exec create: %w", err)
	}

	attach, err := cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return "", -1, fmt.Errorf("exec attach: %w", err)
	}
	defer attach.Close()

	// Capture output AND stream to stdout
	var buf bytes.Buffer
	w := io.MultiWriter(os.Stdout, &buf)
	if _, err := io.Copy(w, attach.Reader); err != nil && err != io.EOF {
		return "", -1, fmt.Errorf("stream output: %w", err)
	}

	inspect, err := cli.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return buf.String(), -1, fmt.Errorf("exec inspect: %w", err)
	}

	if inspect.ExitCode != 0 {
		return buf.String(), inspect.ExitCode, fmt.Errorf("exited with code %d", inspect.ExitCode)
	}

	return buf.String(), 0, nil
}

type Payload struct {
	RunID string `json:"run_id"`
	Repo  string `json:"repo"`
}

func HandleTask(ctx context.Context, t *asynq.Task) error {
	var p Payload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Println("Processing repo:", p.Repo, "run:", p.RunID)

	s := store.NewRunStore(db.DB)
	_ = s.UpdateStatus(p.RunID, "running")

	// TODO: replace with workflow YAML loading from repo
	// For now runs a single default workflow
	err := RunInContainer(
		ctx,
		p.RunID,
		p.Repo,
		"default",
		"oven/bun:alpine",
		[]string{
			"bun install",
			"bun run build",
		},
	)
	if err != nil {
		_ = s.UpdateStatus(p.RunID, "failed")
		return err // let asynq handle retries, don't os.Exit
	}

	_ = s.UpdateStatus(p.RunID, "success")
	return nil
}
