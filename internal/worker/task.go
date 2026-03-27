package worker

import (
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
	"github.com/docker/go-connections/nat"
)

func RunInContainer(ctx context.Context, repoURL string, commands []string) error {
	repoDir, err := os.MkdirTemp("", "repo-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(repoDir)

	fmt.Printf("Cloning %s into %s\n", repoURL, repoDir)

	_, err = git.PlainClone(repoDir, &git.CloneOptions{
		URL:      repoURL,
		Bare:     false,
		Depth:    1,
		Progress: nil,
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

	const dockerImage = "oven/bun:alpine"
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
			ExposedPorts: nat.PortSet{
				"3000/tcp": struct{}{},
			},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: absRepoDir,
					Target: "/workspace",
				},
			},
			PortBindings: nat.PortMap{
				"3000/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "3000"}},
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

	for _, command := range commands {
		if err := execInContainer(ctx, cli, containerID, command); err != nil {
			return fmt.Errorf("exec %q: %w", command, err)
		}
	}

	return nil
}

func execInContainer(ctx context.Context, cli *client.Client, containerID, command string) error {
	fmt.Printf("\n--- Running: %s ---\n", command)

	execID, err := cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          []string{"/bin/sh", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		WorkingDir:   "/workspace",
	})
	if err != nil {
		return fmt.Errorf("exec create: %w", err)
	}

	attach, err := cli.ContainerExecAttach(ctx, execID.ID, container.ExecStartOptions{})
	if err != nil {
		return fmt.Errorf("exec attach: %w", err)
	}
	defer attach.Close()

	if _, err := io.Copy(os.Stdout, attach.Reader); err != nil && err != io.EOF {
		return fmt.Errorf("stream output: %w", err)
	}
	inspect, err := cli.ContainerExecInspect(ctx, execID.ID)
	if err != nil {
		return fmt.Errorf("exec inspect: %w", err)
	}
	if inspect.ExitCode != 0 {
		return fmt.Errorf("exited with code %d", inspect.ExitCode)
	}

	return nil
}

type Payload struct {
	Repo string `json:"repo"`
}

func HandleTask(ctx context.Context, t *asynq.Task) error {
	var p Payload

	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	fmt.Println("Processing repo:", p.Repo)

	err := RunInContainer(
		context.Background(),
		p.Repo,
		[]string{
			"bun install",
			"bun run dev",
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	return nil
}
