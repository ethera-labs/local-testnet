package alloy

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/ethera-labs/local-testnet/internal/observability/shared"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const dockerImage = "grafana/alloy:v1.9.1"

func Start(ctx context.Context, client *client.Client) error {
	logger := slog.With("observability_service_name", "alloy")

	reader, err := client.ImagePull(ctx, dockerImage, image.PullOptions{})
	if err != nil {
		return errors.Join(err, errors.New("failed to pull Alloy image"))
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		logger.Debug(scanner.Text())
	}

	logger.Info("image was pulled. Creating container")

	configAbsPath, err := filepath.Abs("configs/alloy")
	if err != nil {
		return errors.Join(err, errors.New("failed build absolute path for Alloy configuration file"))
	}

	resp, err := client.ContainerCreate(ctx, &container.Config{
		Image: dockerImage,
		ExposedPorts: nat.PortSet{
			"12345/tcp": {},
			"4317/tcp":  {},
			"4318/tcp":  {},
		},
		Labels: shared.Labels,
		Cmd:    strslice.StrSlice{"run", "--server.http.listen-addr=0.0.0.0:12345", "--storage.path=/var/lib/alloy/data", "/etc/alloy/config.alloy"},
	},
		&container.HostConfig{
			NetworkMode: shared.NetworkMode,
			PortBindings: nat.PortMap{
				"12345/tcp": []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "12345"}},
				"4317/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "4317"}},
				"4318/tcp":  []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "4318"}},
			},
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: configAbsPath,
					Target: "/etc/alloy",
				},
				{
					Type:   mount.TypeBind,
					Source: "/var/run/docker.sock",
					Target: "/var/run/docker.sock",
				},
			},
		},
		&network.NetworkingConfig{},
		nil,
		"alloy")
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
