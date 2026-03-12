package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/build"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/ethera-labs/local-testnet/internal/logger"
	"github.com/moby/go-archive"
)

type Client struct {
	cli    *client.Client
	logger *slog.Logger
}

// New creates a new Docker client.
func New() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Client{cli: cli, logger: logger.Named("docker_client")}, nil
}

// Close closes the Docker client connection.
func (c *Client) Close() error {
	return c.cli.Close()
}

// ImageExists checks if a Docker image exists locally.
func (c *Client) ImageExists(ctx context.Context, imageName string) (bool, error) {
	_, err := c.cli.ImageInspect(ctx, imageName)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// PullImage pulls a Docker image from a registry.
func (c *Client) PullImage(ctx context.Context, imageName string) error {
	c.logger.With("image", imageName).Info("pulling docker image")

	resp, err := c.cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer resp.Close()

	scanner := bufio.NewScanner(resp)
	var pullError error
	for scanner.Scan() {
		line := scanner.Text()
		c.logger.Debug(line)

		var msg struct {
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			if msg.Error != "" {
				pullError = fmt.Errorf("pull failed: %s", msg.Error)
				c.logger.Error("docker pull error", "error", msg.Error)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading pull output: %w", err)
	}

	if pullError != nil {
		return pullError
	}

	c.logger.With("image", imageName).Info("docker image pulled successfully")
	return nil
}

// BuildImage builds a Docker image from a Dockerfile.
func (c *Client) BuildImage(ctx context.Context, dockerfilePath, contextPath, tag string, buildArgs map[string]*string) error {
	buildContext, err := archive.TarWithOptions(contextPath, &archive.TarOptions{})
	if err != nil {
		return fmt.Errorf("failed to create build context: %w", err)
	}
	defer buildContext.Close()

	buildOptions := build.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: dockerfilePath,
		Remove:     true,
		BuildArgs:  buildArgs,
	}

	resp, err := c.cli.ImageBuild(ctx, buildContext, buildOptions)
	if err != nil {
		return fmt.Errorf("failed to build image: %w", err)
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var buildError error
	for scanner.Scan() {
		line := scanner.Text()
		c.logger.Debug(line)

		var msg struct {
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err == nil {
			if msg.Error != "" {
				buildError = fmt.Errorf("build failed: %s", msg.Error)
				c.logger.Error("docker build error", "error", msg.Error)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading build output: %w", err)
	}

	if buildError != nil {
		return buildError
	}

	c.logger.With("tag", tag).Info("docker image built successfully")
	return nil
}

// FindContainerByNamePrefix returns the first container name that matches the prefix.
func (c *Client) FindContainerByNamePrefix(ctx context.Context, prefix string) (string, error) {
	containers, err := c.cli.ContainerList(ctx, containertypes.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	for _, container := range containers {
		for _, name := range container.Names {
			name = strings.TrimPrefix(name, "/")
			if strings.HasPrefix(name, prefix) {
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("no container found with prefix %q", prefix)
}

// FindRunningContainerByPrefixes returns the first running container name that matches any prefix.
func (c *Client) FindRunningContainerByPrefixes(ctx context.Context, prefixes ...string) (string, error) {
	containers, err := c.cli.ContainerList(ctx, containertypes.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list running containers: %w", err)
	}

	matches := make([]string, 0)
	for _, container := range containers {
		for _, name := range container.Names {
			name = strings.TrimPrefix(name, "/")
			for _, prefix := range prefixes {
				if strings.HasPrefix(name, prefix) {
					matches = append(matches, name)
					break
				}
			}
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no running container found with prefixes %q", prefixes)
	}

	sort.Strings(matches)
	return matches[0], nil
}

// ContainerNetworkIPv4 returns the IPv4 address of a container on a specific Docker network.
func (c *Client) ContainerNetworkIPv4(ctx context.Context, containerName, network string) (string, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}

	endpoint, ok := inspect.NetworkSettings.Networks[network]
	if !ok || endpoint == nil {
		return "", fmt.Errorf("container %s is not attached to network %s", containerName, network)
	}
	if endpoint.IPAddress != "" {
		return endpoint.IPAddress, nil
	}
	if endpoint.IPAMConfig != nil && endpoint.IPAMConfig.IPv4Address != "" {
		return endpoint.IPAMConfig.IPv4Address, nil
	}

	return "", fmt.Errorf("container %s has no IPv4 address on network %s", containerName, network)
}

// ContainerPublishedHostPort returns the published host port for a container port (for example "8545/tcp").
func (c *Client) ContainerPublishedHostPort(ctx context.Context, containerName, containerPort string) (string, error) {
	inspect, err := c.cli.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", containerName, err)
	}

	portBindings := inspect.NetworkSettings.Ports
	if portBindings == nil {
		return "", fmt.Errorf("container %s has no published ports", containerName)
	}

	bindings, ok := portBindings[nat.Port(containerPort)]
	if !ok || len(bindings) == 0 {
		return "", fmt.Errorf("container %s does not publish port %s", containerName, containerPort)
	}

	return bindings[0].HostPort, nil
}
