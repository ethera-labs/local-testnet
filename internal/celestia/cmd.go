package celestia

import (
	"fmt"
	"log/slog"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	CMD = &cobra.Command{
		Use:   "celestia",
		Short: "Commands for running Celestia DA services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			slog.Info("starting Celestia services")
			if err := start(cmd.Context(), cfg); err != nil {
				return fmt.Errorf("error occurred starting Celestia services: %w", err)
			}
			return nil
		},
	}

	stopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop Celestia services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if err := stop(cmd.Context(), cfg); err != nil {
				return fmt.Errorf("error occurred stopping Celestia services: %w", err)
			}
			return nil
		},
	}

	cleanCmd = &cobra.Command{
		Use:   "clean",
		Short: "Stop Celestia services and remove generated assets/state",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if err := clean(cmd.Context(), cfg); err != nil {
				return fmt.Errorf("error occurred cleaning Celestia services: %w", err)
			}
			return nil
		},
	}

	showCmd = &cobra.Command{
		Use:   "show",
		Short: "Show Celestia containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig()
			if err != nil {
				return err
			}

			if err := show(cmd.Context(), cfg); err != nil {
				return fmt.Errorf("error occurred showing Celestia services: %w", err)
			}
			return nil
		},
	}
)

func init() {
	CMD.AddCommand(stopCmd)
	CMD.AddCommand(cleanCmd)
	CMD.AddCommand(showCmd)
}

func loadConfig() (configs.Celestia, error) {
	// Re-unmarshal to include flag overrides.
	if err := viper.Unmarshal(&configs.Values); err != nil {
		return configs.Celestia{}, fmt.Errorf("failed to unmarshal config with flag overrides: %w", err)
	}
	return configs.Values.Celestia, nil
}
