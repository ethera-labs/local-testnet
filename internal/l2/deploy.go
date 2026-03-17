package l2

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/docker"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [op-geth|publisher|all]",
	Short: "Build and restart selected L2 services for rapid local development",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := strings.ToLower(args[0])
		allowed := map[string]bool{"op-geth": true, "publisher": true, "all": true}
		if !allowed[target] {
			return fmt.Errorf("invalid service '%s' (expected: op-geth|publisher|all)", target)
		}

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
		envVars, err := envBuilder.BuildComposeEnv(configs.Values.L2, common.Address{})
		if err != nil {
			return err
		}

		services := mapServices(target)
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

func mapServices(target string) []string {
	switch target {
	case "op-geth":
		return []string{"op-geth-a", "op-geth-b"}
	case "publisher":
		return []string{"publisher"}
	default:
		return []string{"publisher", "op-geth-a", "op-geth-b"}
	}
}
