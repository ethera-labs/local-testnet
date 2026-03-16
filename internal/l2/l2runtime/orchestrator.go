package l2runtime

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/docker"
	"github.com/ethera-labs/local-testnet/internal/l2/l2config/genesis"
	"github.com/ethera-labs/local-testnet/internal/l2/l2config/secrets"
	"github.com/ethera-labs/local-testnet/internal/l2/l2runtime/contracts"
	"github.com/ethera-labs/local-testnet/internal/l2/l2runtime/registry"
	"github.com/ethera-labs/local-testnet/internal/l2/l2runtime/services"
	"github.com/ethera-labs/local-testnet/internal/logger"
	"github.com/ethereum/go-ethereum/common"
)

// Orchestrator coordinates Phase 3: L2 runtime operations
//   - Builds Docker images via docker-compose
//   - Starts initial services (publisher, op-geth)
//   - Deploys L2 helper contracts
//   - Restarts services to pick up contract addresses
//   - Starts final services (op-node, batcher, proposer)
type Orchestrator struct {
	rootDir     string
	localnetDir string
	networksDir string
	servicesDir string
	logger      *slog.Logger
}

// NewOrchestrator creates a new Phase 3 orchestrator
func NewOrchestrator(rootDir, localnetDir, networksDir, servicesDir string) *Orchestrator {
	return &Orchestrator{
		rootDir:     rootDir,
		localnetDir: localnetDir,
		networksDir: networksDir,
		servicesDir: servicesDir,
		logger:      logger.Named("l2_runtime_orchestrator"),
	}
}

// Execute runs Phase 3: Build images, start services, deploy contracts
func (o *Orchestrator) Execute(ctx context.Context, cfg configs.L2, gameFactoryAddr common.Address) (map[configs.L2ChainName]map[contracts.ContractName]common.Address, error) {
	o.logger.Info("Phase 3: Starting L2 runtime operations")

	publisherConfig := registry.NewConfigurator()
	if err := publisherConfig.SetupRegistry(o.localnetDir, cfg, gameFactoryAddr); err != nil {
		return nil, fmt.Errorf("failed to setup publisher registry: %w", err)
	}

	composePath, err := docker.EnsureComposeFile(o.localnetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare docker-compose file: %w", err)
	}

	envBuilder := docker.NewEnvBuilder(o.rootDir, o.networksDir, o.servicesDir)
	envVars, err := envBuilder.BuildComposeEnv(cfg, gameFactoryAddr)
	if err != nil {
		return nil, err
	}

	o.logger.With("env", envVars).Info("environment variables were constructed. Building compose services")
	if err := o.buildComposeServices(ctx, composePath, envVars, cfg); err != nil {
		return nil, fmt.Errorf("failed to build compose services: %w", err)
	}

	o.logger.Info("docker-compose services built successfully")
	serviceManager := services.NewManager(o.rootDir, composePath)

	var flashblocksComposePath string
	var sidecarComposePath string

	if cfg.Flashblocks.Enabled {
		o.logger.Info("flashblocks enabled, configuring services to use rollup-boost")
		flashblocksComposePath, err = docker.EnsureFlashblocksComposeFile(o.localnetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare flashblocks compose file: %w", err)
		}
		serviceManager.WithFlashblocks(flashblocksComposePath)

		if cfg.Flashblocks.OpRbuilderImageTag != "" {
			envVars["OP_RBUILDER_IMAGE_TAG"] = cfg.Flashblocks.OpRbuilderImageTag
		}
		if cfg.Flashblocks.RollupBoostImageTag != "" {
			envVars["ROLLUP_BOOST_IMAGE_TAG"] = cfg.Flashblocks.RollupBoostImageTag
		}
	}

	if cfg.Sidecar.Enabled {
		if !cfg.Flashblocks.Enabled {
			return nil, fmt.Errorf("sidecar requires flashblocks to be enabled")
		}
		o.logger.Info("sidecar enabled, configuring sidecar services")
		sidecarComposePath, err = docker.EnsureSidecarComposeFile(o.localnetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare sidecar compose file: %w", err)
		}
		serviceManager.WithSidecar(sidecarComposePath)
	}

	if cfg.Frontend.Enabled {
		if !cfg.Flashblocks.Enabled || !cfg.Sidecar.Enabled {
			return nil, fmt.Errorf("frontend requires flashblocks and sidecar to be enabled")
		}
		o.logger.Info("frontend enabled, configuring Compose Network Console")
		frontendComposePath, err := docker.EnsureFrontendComposeFile(o.localnetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare frontend compose file: %w", err)
		}
		serviceManager.WithFrontend(frontendComposePath)
	}

	if err := o.waitForNetworkFiles(); err != nil {
		return nil, fmt.Errorf("required network files not ready: %w", err)
	}

	if err := serviceManager.StartAll(ctx, envVars); err != nil {
		return nil, fmt.Errorf("failed to start L2 services: %w", err)
	}

	// When flashblocks is enabled, use op-rbuilder RPC ports for contract deployment
	effectiveChainConfigs := cfg.ChainConfigs
	if cfg.Flashblocks.Enabled {
		effectiveChainConfigs = o.getFlashblocksChainConfigs(cfg)
		o.logger.Info("using flashblocks RPC ports for contract deployment",
			"rollup_a_port", effectiveChainConfigs[configs.L2ChainNameRollupA].RPCPort,
			"rollup_b_port", effectiveChainConfigs[configs.L2ChainNameRollupB].RPCPort)
	}

	contractDeployer := contracts.NewDeployer(o.networksDir)
	deployedContracts, err := contractDeployer.Deploy(ctx, effectiveChainConfigs, cfg.CoordinatorPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy contracts: %w", err)
	}

	o.logger.Info("restarting op-geth services to apply mailbox configuration")
	if err := o.restartOpGeth(ctx, composePath, envVars, deployedContracts); err != nil {
		return nil, fmt.Errorf("failed to restart op-geth services after contract deployment. Error: '%w'", err)
	}

	if cfg.Sidecar.Enabled {
		o.logger.Info("restarting sidecar services to apply mailbox configuration")
		if err := o.restartSidecar(ctx, composePath, flashblocksComposePath, sidecarComposePath, envVars); err != nil {
			return nil, fmt.Errorf("failed to restart sidecar services after contract deployment: %w", err)
		}
	}

	if cfg.Frontend.Enabled {
		o.logger.Info("building and starting Compose Network Console")
		chainContracts := deployedContracts[configs.L2ChainNameRollupA]
		envVars["CONTRACT_BRIDGE_ADDRESS"] = chainContracts[contracts.ContractNameBridge].Hex()
		envVars["CONTRACT_TOKEN_ADDRESS"] = chainContracts[contracts.ContractNameBridgeableToken].Hex()

		composeFiles := []string{composePath}
		if flashblocksComposePath != "" {
			composeFiles = append(composeFiles, flashblocksComposePath)
		}
		if sidecarComposePath != "" {
			composeFiles = append(composeFiles, sidecarComposePath)
		}

		if err := serviceManager.StartFrontend(ctx, composeFiles, envVars); err != nil {
			return nil, fmt.Errorf("failed to start Compose Network Console: %w", err)
		}
	}

	o.logger.Info("Phase 3: L2 runtime operations completed successfully")

	return deployedContracts, nil
}

func (o *Orchestrator) waitForNetworkFiles() error {
	type fileSpec struct {
		path  string
		label string
	}
	files := []fileSpec{
		{
			path:  filepath.Join(o.networksDir, string(configs.L2ChainNameRollupA), genesis.GenesisFileName),
			label: "rollup-a genesis",
		},
		{
			path:  filepath.Join(o.networksDir, string(configs.L2ChainNameRollupA), secrets.JWTFileName),
			label: "rollup-a jwt",
		},
		{
			path:  filepath.Join(o.networksDir, string(configs.L2ChainNameRollupB), genesis.GenesisFileName),
			label: "rollup-b genesis",
		},
		{
			path:  filepath.Join(o.networksDir, string(configs.L2ChainNameRollupB), secrets.JWTFileName),
			label: "rollup-b jwt",
		},
	}

	deadline := time.Now().Add(120 * time.Second)
	for {
		missing := make([]fileSpec, 0, len(files))
		for _, f := range files {
			info, err := os.Stat(f.path)
			if err != nil || info.Size() == 0 {
				missing = append(missing, f)
			}
		}

		if len(missing) == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			parts := make([]string, 0, len(missing))
			for _, f := range missing {
				parts = append(parts, fmt.Sprintf("%s(%s)", f.label, f.path))
			}
			return fmt.Errorf("missing files: %s", strings.Join(parts, " "))
		}

		time.Sleep(1 * time.Second)
	}
}

func (o *Orchestrator) restartOpGeth(ctx context.Context, composeFilePath string, env map[string]string, deployedContracts map[configs.L2ChainName]map[contracts.ContractName]common.Address) error {
	mailboxA := deployedContracts[configs.L2ChainNameRollupA][contracts.ContractNameMailbox]
	mailboxB := deployedContracts[configs.L2ChainNameRollupB][contracts.ContractNameMailbox]

	if mailboxA == (common.Address{}) || mailboxB == (common.Address{}) {
		return fmt.Errorf("mailbox addresses not found in deployed contracts")
	}

	env["MAILBOX_A"] = mailboxA.Hex()
	env["MAILBOX_B"] = mailboxB.Hex()

	o.logger.Info("restarting op-geth with mailbox addresses",
		"mailbox_a", mailboxA.Hex(),
		"mailbox_b", mailboxB.Hex())

	services := []string{"op-geth-a", "op-geth-b"}
	if err := docker.ComposeRestart(ctx, composeFilePath, env, services...); err != nil {
		return fmt.Errorf("failed to restart op-geth: %w", err)
	}

	o.logger.Info("op-geth services restarted successfully, waiting for them to be ready")

	return nil
}

func (o *Orchestrator) restartSidecar(ctx context.Context, composeFilePath, flashblocksComposePath, sidecarComposePath string, env map[string]string) error {
	if sidecarComposePath == "" {
		return fmt.Errorf("sidecar compose file path is empty")
	}

	composeFiles := []string{composeFilePath}
	if flashblocksComposePath != "" {
		composeFiles = append(composeFiles, flashblocksComposePath)
	}
	composeFiles = append(composeFiles, sidecarComposePath)

	services := []string{"sidecar-a", "sidecar-b"}
	if err := docker.ComposeRestartMultiFile(ctx, composeFiles, env, services...); err != nil {
		return fmt.Errorf("failed to restart sidecar: %w", err)
	}

	return nil
}

// buildComposeServices builds services using docker-compose
func (o *Orchestrator) buildComposeServices(ctx context.Context, composeFilePath string, env map[string]string, cfg configs.L2) error {
	services := []string{
		"publisher",
		"op-geth-a",
		"op-geth-b",
	}

	composeFiles := []string{composeFilePath}

	// Sidecar requires flashblocks, so add flashblocks compose file first
	if cfg.Sidecar.Enabled {
		flashblocksComposePath, err := docker.EnsureFlashblocksComposeFile(o.localnetDir)
		if err != nil {
			return fmt.Errorf("failed to prepare flashblocks compose file for build: %w", err)
		}
		composeFiles = append(composeFiles, flashblocksComposePath)

		sidecarComposePath, err := docker.EnsureSidecarComposeFile(o.localnetDir)
		if err != nil {
			return fmt.Errorf("failed to prepare sidecar compose file for build: %w", err)
		}
		composeFiles = append(composeFiles, sidecarComposePath)
		services = append(services, "sidecar-a", "sidecar-b")
	}

	if len(composeFiles) > 1 {
		if err := docker.ComposeBuildMultiFile(ctx, composeFiles, env, services...); err != nil {
			return fmt.Errorf("failed to build compose services: %w", err)
		}
	} else {
		if err := docker.ComposeBuild(ctx, composeFilePath, env, services...); err != nil {
			return fmt.Errorf("failed to build compose services: %w", err)
		}
	}

	return nil
}

// getFlashblocksChainConfigs returns chain configs with op-rbuilder RPC ports.
func (o *Orchestrator) getFlashblocksChainConfigs(cfg configs.L2) map[configs.L2ChainName]configs.Chain {
	result := make(map[configs.L2ChainName]configs.Chain)

	for chainName, chainCfg := range cfg.ChainConfigs {
		modifiedCfg := chainCfg
		switch chainName {
		case configs.L2ChainNameRollupA:
			if cfg.Flashblocks.RollupARPCPort > 0 {
				modifiedCfg.RPCPort = cfg.Flashblocks.RollupARPCPort
			}
		case configs.L2ChainNameRollupB:
			if cfg.Flashblocks.RollupBRPCPort > 0 {
				modifiedCfg.RPCPort = cfg.Flashblocks.RollupBRPCPort
			}
		}
		result[chainName] = modifiedCfg
	}

	return result
}
