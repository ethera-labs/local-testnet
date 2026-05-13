package l2

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/docker"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Rebuild and restart the publisher for rapid local development",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		rootDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		localnetDir := filepath.Join(rootDir, localnetDirName)
		networksDir := filepath.Join(localnetDir, networksDirName)
		servicesDir := filepath.Join(localnetDir, servicesDirName)

		dockerPath, err := docker.EnsureComposeFile(localnetDir)
		if err != nil {
			return fmt.Errorf("failed to prepare docker file: %w", err)
		}

		envBuilder := docker.NewEnvBuilder(rootDir, networksDir, servicesDir)
		envVars, err := envBuilder.BuildComposeEnv(configs.Values.L2, common.Address{}, common.Address{})
		if err != nil {
			return err
		}

		services := []string{"publisher"}
		ctx := cmd.Context()
		slog.With("services", services).Info("building services from local sources")
		if err := docker.ComposeBuild(ctx, dockerPath, envVars, services...); err != nil {
			return fmt.Errorf("failed to build services: %w", err)
		}

		slog.Info("restarting services to apply new images")
		if err := docker.ComposeRestart(ctx, dockerPath, envVars, services...); err != nil {
			return fmt.Errorf("failed to restart services: %w", err)
		}

		slog.Info("deploy completed successfully")
		return nil
	},
}
