package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
)

type RunOptions struct {
	Image      string
	Entrypoint []string
	Cmd        []string
	Env        []string
	Network    string
	Volumes    map[string]string // host:container
	WorkDir    string
	User       string
	AutoRemove bool
	StreamLogs bool
	CaptureOut bool
}

// Run runs a Docker container and waits for it to complete.
func (c *Client) Run(ctx context.Context, opts RunOptions) (string, error) {
	config := &container.Config{
		Image:      opts.Image,
		Entrypoint: opts.Entrypoint,
		Cmd:        opts.Cmd,
		Env:        opts.Env,
		WorkingDir: opts.WorkDir,
		User:       opts.User,
	}

	hostConfig := &container.HostConfig{
		AutoRemove: opts.AutoRemove,
	}
	var networkingConfig *network.NetworkingConfig
	if opts.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(opts.Network)
		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				opts.Network: {},
			},
		}
	}

	if len(opts.Volumes) > 0 {
		binds := make([]string, 0, len(opts.Volumes))
		for host, containerPath := range opts.Volumes {
			binds = append(binds, fmt.Sprintf("%s:%s", host, containerPath))
		}
		hostConfig.Binds = binds
	}

	resp, err := c.cli.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	containerID := resp.ID

	defer func() {
		if err != nil && !opts.AutoRemove {
			_ = c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true})
		}
	}()

	// Attach to container logs before starting (needed for AutoRemove containers)
	var stdout, stderr bytes.Buffer
	var attachDone chan error

	if opts.CaptureOut || opts.StreamLogs || opts.AutoRemove {
		attachResp, err := c.cli.ContainerAttach(ctx, containerID, container.AttachOptions{
			Stream: true,
			Stdout: true,
			Stderr: true,
		})
		if err != nil {
			return "", fmt.Errorf("failed to attach to container: %w", err)
		}
		defer attachResp.Close()

		// Start copying output in background
		attachDone = make(chan error, 1)
		go func() {
			var copyErr error
			if opts.StreamLogs {
				// Stream to console and capture
				outWriter := io.MultiWriter(os.Stdout, &stdout)
				errWriter := io.MultiWriter(os.Stderr, &stderr)
				_, copyErr = stdcopy.StdCopy(outWriter, errWriter, attachResp.Reader)
			} else {
				// Just capture
				_, copyErr = stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader)
			}
			attachDone <- copyErr
		}()
	}

	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	statusCh, errCh := c.cli.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		if err := waitForAttachDone(ctx, attachDone); err != nil {
			return "", err
		}

		if status.StatusCode != 0 {
			errorOutput := stdout.String() + stderr.String()
			if errorOutput != "" {
				return "", fmt.Errorf("container exited with code %d: %s", status.StatusCode, errorOutput)
			}
			return "", fmt.Errorf("container exited with code %d", status.StatusCode)
		}
	}

	// Return captured output
	if opts.CaptureOut {
		return stdout.String(), nil
	}

	return "", nil
}

func waitForAttachDone(ctx context.Context, attachDone <-chan error) error {
	if attachDone == nil {
		return nil
	}

	select {
	case copyErr := <-attachDone:
		if copyErr != nil {
			return fmt.Errorf("failed to copy container output: %w", copyErr)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context done while waiting for container output: %w", ctx.Err())
	}
}
