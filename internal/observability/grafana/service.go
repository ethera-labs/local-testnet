package grafana

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
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/ethera-labs/local-testnet/internal/observability/shared"
)

const dockerImage = "grafana/grafana:12.0.1"

func Start(ctx context.Context, client *client.Client) error {
	logger := slog.With("observability_service_name", "grafana")

	reader, err := client.ImagePull(ctx, dockerImage, image.PullOptions{})
	if err != nil {
		return errors.Join(err, errors.New("failed to pull Grafana image"))
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logger.Debug(scanner.Text())
	}

	logger.Info("grafana image was pulled. Creating container")

	configAbsPath, err := filepath.Abs("configs/grafana")
	if err != nil {
		return errors.Join(err, errors.New("failed build absolute path for Grafana configuration file"))
	}

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: dockerImage,
		Env: []string{
			"GF_AUTH_ANONYMOUS_ENABLED=true",
			"GF_AUTH_ANONYMOUS_ORG_ROLE=Admin",
		},
		ExposedPorts: nat.PortSet{
			"3000/tcp": struct{}{},
		},
		Labels: shared.Labels,
	},
		&container.HostConfig{
			NetworkMode: shared.NetworkMode,
			PortBindings: nat.PortMap{
				"3000/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "3000"}},
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: configAbsPath,
					Target: "/etc/grafana/provisioning/datasources/",
				},
			},
		},
		&network.NetworkingConfig{},
		nil,
		"grafana")
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
