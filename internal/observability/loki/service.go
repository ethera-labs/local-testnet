package loki

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/ethera-labs/local-testnet/internal/observability/shared"
)

const dockerImage = "grafana/loki:3.5.1"

func Start(ctx context.Context, client *client.Client) error {
	logger := slog.With("observability_service_name", "loki")

	reader, err := client.ImagePull(ctx, dockerImage, image.PullOptions{})
	if err != nil {
		return errors.Join(err, errors.New("failed to pull Loki image"))
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logger.Debug(scanner.Text())
	}

	logger.Info("image was pulled. Creating container")

	configAbsPath, err := filepath.Abs("configs/loki")
	if err != nil {
		return errors.Join(err, errors.New("failed build absolute path for Loki configuration file"))
	}

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: dockerImage,
		ExposedPorts: nat.PortSet{
			"3100/tcp": struct{}{},
		},
		Labels: shared.Labels,
		Cmd:    strslice.StrSlice{"-config.file=/etc/loki/config.yaml"},
	},
		&container.HostConfig{
			NetworkMode: shared.NetworkMode,
			PortBindings: nat.PortMap{
				"3100/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "3100"}},
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: configAbsPath,
					Target: "/etc/loki",
				},
			},
		},
		&network.NetworkingConfig{},
		nil,
		"loki")
	if err != nil {
		return errors.Join(err, errors.New("failed to create container"))
	}

	logger.With("ID", resp.ID).Info("container created. Starting...")

	if err := client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return errors.Join(err, errors.New("failed to start container"))
	}

	logger.Info("container started")

	if err := shared.AttachToLocalnetNetworks(ctx, client, resp.ID); err != nil {
		return errors.Join(err, errors.New("failed to attach container to localnet networks"))
	}

	logger.Info("container attached to localnet networks")

	return nil
}
