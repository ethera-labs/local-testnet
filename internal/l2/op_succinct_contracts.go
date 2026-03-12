package l2

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem/json"
	"github.com/ethera-labs/local-testnet/internal/l2/l1deployment/deployer"
	"github.com/ethera-labs/local-testnet/internal/l2/l2runtime"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

var opSuccinctContractsCmd = &cobra.Command{
	Use:   "op-succinct-contracts",
	Short: "Deploy op-succinct contracts and refresh env files without starting op-succinct services",
	RunE: func(cmd *cobra.Command, _ []string) error {
		slog.Info("starting op-succinct contract-only deployment. Validating config", slog.Any("config", configs.Values.L2))

		if err := configs.Values.L2.Validate(); err != nil {
			return err
		}

		resolvedCfg, err := resolveManagedL1Endpoints(cmd.Context(), configs.Values.L2, slog.Default())
		if err != nil {
			return fmt.Errorf("failed to resolve local L1 endpoints: %w", err)
		}

		rootDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		localnetDir := filepath.Join(rootDir, localnetDirName)
		stateDir := filepath.Join(localnetDir, stateDirName)
		networksDir := filepath.Join(localnetDir, networksDirName)
		servicesDir := filepath.Join(localnetDir, servicesDirName)

		disputeGameFactoryProxyAddresses, err := loadDisputeGameFactoryProxyAddresses(stateDir, resolvedCfg)
		if err != nil {
			return err
		}

		runtimeOrchestrator := l2runtime.NewOrchestrator(rootDir, localnetDir, networksDir, servicesDir)
		if err := runtimeOrchestrator.DeployOpSuccinctContractsOnly(
			cmd.Context(),
			resolvedCfg,
			disputeGameFactoryProxyAddresses,
		); err != nil {
			return fmt.Errorf("op-succinct contract-only deployment failed: %w", err)
		}

		slog.Info("op-succinct contract-only deployment completed successfully")
		return nil
	},
}

func loadDisputeGameFactoryProxyAddresses(stateDir string, cfg configs.L2) (map[configs.L2ChainName]common.Address, error) {
	stateManager := deployer.NewStateManager(stateDir, json.NewReader())
	state, err := stateManager.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load state from %s/state.json: %w", stateDir, err)
	}

	chainIDToName := make(map[int64]configs.L2ChainName, len(cfg.ChainConfigs))
	for chainName, chainCfg := range cfg.ChainConfigs {
		chainIDToName[int64(chainCfg.ID)] = chainName
	}

	disputeGameFactoryProxyAddresses := make(map[configs.L2ChainName]common.Address, len(cfg.ChainConfigs))
	for _, opChain := range state.OpChainDeployments {
		chainID, err := strconv.ParseInt(opChain.ID, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse chain ID %q from state.json: %w", opChain.ID, err)
		}

		chainName, ok := chainIDToName[chainID]
		if !ok {
			continue
		}

		address := common.HexToAddress(opChain.DisputeGameFactoryProxy)
		if address == (common.Address{}) {
			return nil, fmt.Errorf("state.json has empty DisputeGameFactoryProxy for chain %s", chainName)
		}
		disputeGameFactoryProxyAddresses[chainName] = address
	}

	for _, chainName := range cfg.EnabledOpSuccinctChains() {
		if _, ok := disputeGameFactoryProxyAddresses[chainName]; !ok {
			return nil, fmt.Errorf("state.json is missing DisputeGameFactoryProxy for enabled op-succinct chain %s", chainName)
		}
	}

	return disputeGameFactoryProxyAddresses, nil
}
