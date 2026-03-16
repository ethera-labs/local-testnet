package l2

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/blockscout"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/git"
	"github.com/ethera-labs/local-testnet/internal/l2/l1deployment"
	"github.com/ethera-labs/local-testnet/internal/l2/l2config"
	"github.com/ethera-labs/local-testnet/internal/l2/l2runtime"
	"github.com/ethera-labs/local-testnet/internal/l2/output"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var CMD = &cobra.Command{
	Use:   "l2",
	Short: "Commands for running L2 network",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Re-unmarshal config to pick up flag overrides
		// Flags have higher precedence than config file values
		if err := viper.Unmarshal(&configs.Values); err != nil {
			return fmt.Errorf("failed to unmarshal config with flag overrides: %w", err)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		slog.Info("starting l2 command. Validating config", slog.Any("config", configs.Values.L2))

		if err := configs.Values.L2.Validate(); err != nil {
			return err
		}

		slog.Info("config validation successful. Starting l2 deployment...")

		rootDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		localnetDir := filepath.Join(rootDir, localnetDirName)
		stateDir := filepath.Join(localnetDir, stateDirName)
		networksDir := filepath.Join(localnetDir, networksDirName)
		servicesDir := filepath.Join(localnetDir, servicesDirName)

		l1Orchestrator := l1deployment.NewOrchestrator(rootDir, stateDir, servicesDir)
		l2ConfigOrchestrator := l2config.NewOrchestrator(rootDir, localnetDir, stateDir, networksDir, servicesDir)
		runtimeOrchestrator := l2runtime.NewOrchestrator(rootDir, localnetDir, networksDir, servicesDir)

		service := NewService(rootDir, git.NewCloner(), l1Orchestrator, l2ConfigOrchestrator, runtimeOrchestrator, blockscout.New(localnetDir, networksDir), output.NewGenerator())

		if err := service.Deploy(cmd.Context(), configs.Values.L2); err != nil {
			return fmt.Errorf("l2 deployment failed: %w", err)
		}

		slog.Info("l2 deployment completed successfully")

		return nil
	},
}
