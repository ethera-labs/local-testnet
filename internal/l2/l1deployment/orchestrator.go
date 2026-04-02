package l1deployment

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/docker"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem/json"
	"github.com/ethera-labs/local-testnet/internal/l2/l1deployment/deployer"
	"github.com/ethera-labs/local-testnet/internal/l2/l1deployment/dispute"
	"github.com/ethera-labs/local-testnet/internal/l2/l2config/crypto"
	"github.com/ethera-labs/local-testnet/internal/logger"
	"github.com/ethereum/go-ethereum/common"
)

/*
Orchestrator coordinates Phase 1: L1 deployment
  - Initializes op-deployer state and writes intent.toml
  - Deploys OP Stack L1 contracts to the L1 chain
  - Outputs state.json with contract addresses
*/
type (
	DeploymentState struct {
		DisputeGameFactoryAddress       common.Address
		ComposeL2OutputOracleAddress    common.Address
		DisputeGameFactoryImplAddressOP common.Address //TODO: Determine the necessity of this variable's usage.
		StartBlocks                     map[configs.L2ChainName]StartBlock
		SystemConfigProxyAddresses      map[configs.L2ChainName]common.Address
	}

	StartBlock struct {
		Hash   string
		Number string
	}

	Orchestrator struct {
		rootDir     string
		stateDir    string
		servicesDir string
		logger      *slog.Logger
	}
)

// NewOrchestrator creates a new Phase 1 orchestrator
func NewOrchestrator(rootDir, stateDir, servicesDir string) *Orchestrator {
	return &Orchestrator{
		rootDir:     rootDir,
		stateDir:    stateDir,
		servicesDir: servicesDir,
		logger:      logger.Named("l1_orchestrator"),
	}
}

// Execute runs Phase 1: Deploy L1 contracts and dispute contracts
// Returns deployment state with DisputeGameFactory address
func (o *Orchestrator) Execute(ctx context.Context, cfg configs.L2) (DeploymentState, error) {
	o.logger.Info("Phase 1: Starting L1 deployment")

	var deploymentState DeploymentState
	stateManager := deployer.NewStateManager(o.stateDir, json.NewReader())

	o.logger.Info("ensuring state directory created")
	if err := stateManager.EnsureStateDir(); err != nil {
		return deploymentState, fmt.Errorf("failed to ensure state directory: %w", err)
	}

	o.logger.Info("instantiating Docker client")
	dockerClient, err := docker.New()
	if err != nil {
		return deploymentState, fmt.Errorf("failed to create docker client: %w", err)
	}
	defer dockerClient.Close()

	o.logger.Info("instantiating Deployer")
	opDeployer := deployer.NewDeployer(o.rootDir, o.stateDir, cfg.Images[configs.ImageNameOpDeployer].Tag, dockerClient)

	o.logger.Info("initializing Deployer")
	if err := opDeployer.Init(ctx, cfg.L1ChainID, cfg.ChainConfigs); err != nil {
		return deploymentState, fmt.Errorf("failed to initialize op-deployer state: %w", err)
	}

	o.logger.Info("converting coordinator PK to address")
	coordinatorAddress, err := crypto.AddressFromPrivateKey(cfg.CoordinatorPrivateKey)
	if err != nil {
		return deploymentState, fmt.Errorf("failed to derive coordinator address: %w", err)
	}

	o.logger.Info("generating intent file")
	intentWriter := deployer.NewIntentWriter(o.stateDir, json.NewWriter())
	if err := intentWriter.WriteIntent(
		cfg.Wallet.Address,
		coordinatorAddress,
		cfg.L1ChainID,
		cfg.ChainConfigs,
		cfg.AltDA,
	); err != nil {
		return deploymentState, fmt.Errorf("failed to write intent: %w", err)
	}

	if err := opDeployer.Apply(ctx, cfg.L1ElURL, cfg.Wallet.PrivateKey, cfg.DeploymentTarget); err != nil {
		return deploymentState, fmt.Errorf("failed to deploy L1 contracts: %w", err)
	}

	if cfg.AltDA.SkipL1Deploy && cfg.AltDA.ChallengeProxyAddress != "" {
		if err := stateManager.PatchAltDAState(cfg.AltDA); err != nil {
			return deploymentState, fmt.Errorf("failed to patch AltDA state: %w", err)
		}
	}

	opState, err := stateManager.Load()
	if err != nil {
		return deploymentState, fmt.Errorf("failed to load OP deployment state: %w", err)
	}

	o.logger.Info("deploying dispute contracts")
	disputeService := dispute.NewService(o.rootDir, o.servicesDir, cfg)
	disputeContracts, err := disputeService.Deploy(ctx)
	if err != nil {
		return deploymentState, fmt.Errorf("failed to deploy dispute contracts: %w", err)
	}

	o.logger.With(
		"game_factory_address", disputeContracts.DisputeGameFactoryAddress,
		"compose_l2_output_oracle_address", disputeContracts.ComposeL2OutputOracleAddress,
	).Info("Phase 1: L1 deployment completed successfully")

	startBlocks := make(map[configs.L2ChainName]StartBlock)
	systemConfigProxyAddresses := make(map[configs.L2ChainName]common.Address)
	for _, opChain := range opState.OpChainDeployments {
		chainIDInt, err := strconv.ParseInt(opChain.ID, 0, 64)
		if err != nil {
			return deploymentState, fmt.Errorf("failed to parse chain ID %s: %w", opChain.ID, err)
		}

		for chainName, chainConfig := range cfg.ChainConfigs {
			if int64(chainConfig.ID) == chainIDInt {
				startBlocks[chainName] = StartBlock{
					Hash:   opChain.StartBlock.Hash,
					Number: opChain.StartBlock.Number,
				}
				systemConfigProxyAddresses[chainName] = common.HexToAddress(opChain.SystemConfigProxy)
				break
			}
		}
	}

	deploymentState = DeploymentState{
		DisputeGameFactoryAddress:       disputeContracts.DisputeGameFactoryAddress,
		ComposeL2OutputOracleAddress:    disputeContracts.ComposeL2OutputOracleAddress,
		DisputeGameFactoryImplAddressOP: common.HexToAddress(opState.ImplementationsDeployment.DisputeGameFactoryImplAddress),
		StartBlocks:                     startBlocks,
		SystemConfigProxyAddresses:      systemConfigProxyAddresses,
	}

	return deploymentState, nil
}
