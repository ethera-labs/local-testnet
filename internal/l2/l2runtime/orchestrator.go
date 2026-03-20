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
	"github.com/ethera-labs/local-testnet/internal/l2/l1deployment"
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
//   - Runs op-succinct setup calls and starts op-succinct services
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

// Execute runs Phase 3: Build images, start services, deploy contracts.
func (o *Orchestrator) Execute(ctx context.Context, cfg configs.L2, deploymentState l1deployment.DeploymentState) (map[configs.L2ChainName]map[contracts.ContractName]common.Address, error) {
	o.logger.Info("Phase 3: Starting L2 runtime operations")
	gameFactoryAddr := deploymentState.DisputeGameFactoryAddress

	publisherConfig := registry.NewConfigurator()
	if err := publisherConfig.SetupRegistry(o.localnetDir, cfg, gameFactoryAddr); err != nil {
		return nil, fmt.Errorf("failed to setup publisher registry: %w", err)
	}

	dockerPath, err := docker.EnsureComposeFile(o.localnetDir)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare docker file: %w", err)
	}

	envBuilder := docker.NewEnvBuilder(o.rootDir, o.networksDir, o.servicesDir)
	enabledOpSuccinctChains := cfg.EnabledOpSuccinctChains()
	opSuccinctEnabled := isOpSuccinctEnabled(cfg, enabledOpSuccinctChains)
	if opSuccinctEnabled && cfg.IsLocalOpAltDAEnabled() && docker.CanUseOpSuccinctPrebuiltBinary() {
		opSuccinctPath, err := envBuilder.ResolveRepoPath(
			cfg.Repositories[configs.RepositoryNameOpSuccinct],
			configs.RepositoryNameOpSuccinct,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve op-succinct path: %w", err)
		}
		if err := o.prepareOpSuccinctPrebuiltBinary(ctx, cfg, opSuccinctPath); err != nil {
			return nil, fmt.Errorf("failed to prepare op-succinct prebuilt binary: %w", err)
		}
	} else if opSuccinctEnabled && cfg.IsLocalOpAltDAEnabled() {
		o.logger.Info("skipping host-built op-succinct prebuilt binary; Docker will build a Linux binary for this platform")
	}

	envVars, err := envBuilder.BuildComposeEnv(cfg, gameFactoryAddr)
	if err != nil {
		return nil, err
	}

	opSuccinctPath := envVars["OP_SUCCINCT_PATH"]
	if opSuccinctEnabled {
		if opSuccinctPath == "" {
			return nil, fmt.Errorf("OP_SUCCINCT_PATH is empty")
		}
		if err := o.prepareOpSuccinctEnvFiles(cfg, envVars, deploymentState.DisputeGameFactoryProxyAddresses, deploymentState.AnchorStateRegistryProxyAddresses, opSuccinctPath); err != nil {
			return nil, fmt.Errorf("failed to prepare op-succinct env files: %w", err)
		}
	} else {
		o.logger.With("enabled_chains", enabledOpSuccinctChains).Info("op-succinct is disabled or repository is not configured; skipping op-succinct setup and services")
	}

	o.logger.With("env", envVars).Info("environment variables were constructed. Building compose services")
	if err := o.buildComposeServices(
		ctx,
		dockerPath,
		envVars,
		cfg,
		opSuccinctBuildServiceName(enabledOpSuccinctChains, opSuccinctEnabled),
		cfg.IsLocalOpAltDAEnabled(),
	); err != nil {
		return nil, fmt.Errorf("failed to build compose services: %w", err)
	}

	o.logger.Info("docker services built successfully")
	serviceManager := services.NewManager(o.rootDir, dockerPath)

	var flashblocksDockerPath string
	var sidecarDockerPath string

	if cfg.Flashblocks.Enabled {
		o.logger.Info("flashblocks enabled, configuring services to use rollup-boost")
		flashblocksDockerPath, err = docker.EnsureFlashblocksComposeFile(o.localnetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare flashblocks docker file: %w", err)
		}
		serviceManager.WithFlashblocks(flashblocksDockerPath)

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
		sidecarDockerPath, err = docker.EnsureSidecarComposeFile(o.localnetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare sidecar docker file: %w", err)
		}
		serviceManager.WithSidecar(sidecarDockerPath)
	}

	if cfg.Frontend.Enabled {
		if !cfg.Flashblocks.Enabled || !cfg.Sidecar.Enabled {
			return nil, fmt.Errorf("frontend requires flashblocks and sidecar to be enabled")
		}
		o.logger.Info("frontend enabled, configuring Ethera Labs Console")
		frontendDockerPath, err := docker.EnsureFrontendComposeFile(o.localnetDir)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare frontend docker file: %w", err)
		}
		serviceManager.WithFrontend(frontendDockerPath)
	}

	if err := o.waitForNetworkFiles(); err != nil {
		return nil, fmt.Errorf("required network files not ready: %w", err)
	}

	if err := serviceManager.StartAll(ctx, envVars, cfg.IsLocalOpAltDAEnabled()); err != nil {
		return nil, fmt.Errorf("failed to start base L2 services: %w", err)
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

	if err := publisherConfig.UpdateRollupMailboxAddresses(
		o.localnetDir,
		cfg.EtheraNetworkName,
		map[configs.L2ChainName]common.Address{
			configs.L2ChainNameRollupA: deployedContracts[configs.L2ChainNameRollupA][contracts.ContractNameMailbox],
			configs.L2ChainNameRollupB: deployedContracts[configs.L2ChainNameRollupB][contracts.ContractNameMailbox],
		},
	); err != nil {
		return nil, fmt.Errorf("failed to update registry mailbox addresses: %w", err)
	}

	o.logger.Info("restarting op-geth services to apply mailbox configuration")
	if err := o.restartOpGeth(ctx, dockerPath, envVars, deployedContracts); err != nil {
		return nil, fmt.Errorf("failed to restart op-geth services after contract deployment. Error: '%w'", err)
	}

	if cfg.Sidecar.Enabled {
		o.logger.Info("restarting sidecar services to apply mailbox configuration")
		if err := o.restartSidecar(ctx, dockerPath, flashblocksDockerPath, sidecarDockerPath, envVars); err != nil {
			return nil, fmt.Errorf("failed to restart sidecar services after contract deployment: %w", err)
		}
	}

	if opSuccinctEnabled {
		if err := o.setupOpSuccinct(ctx, cfg, opSuccinctPath, envVars); err != nil {
			return nil, fmt.Errorf("failed to run op-succinct setup calls: %w", err)
		}
		if err := o.finalizeOpSuccinctRuntimeEnvFiles(cfg, envVars); err != nil {
			return nil, fmt.Errorf("failed to finalize op-succinct runtime env files: %w", err)
		}

		if err := serviceManager.StartOpSuccinct(
			ctx,
			envVars,
			enabledOpSuccinctChains,
		); err != nil {
			return nil, fmt.Errorf("failed to start op-succinct services: %w", err)
		}
	}

	if cfg.Frontend.Enabled {
		o.logger.Info("building and starting Ethera Labs Console")
		chainContracts := deployedContracts[configs.L2ChainNameRollupA]
		envVars["CONTRACT_BRIDGE_ADDRESS"] = chainContracts[contracts.ContractNameBridge].Hex()
		envVars["CONTRACT_TOKEN_ADDRESS"] = chainContracts[contracts.ContractNameBridgeableToken].Hex()

		dockerFiles := []string{dockerPath}
		if flashblocksDockerPath != "" {
			dockerFiles = append(dockerFiles, flashblocksDockerPath)
		}
		if sidecarDockerPath != "" {
			dockerFiles = append(dockerFiles, sidecarDockerPath)
		}

		if err := serviceManager.StartFrontend(ctx, dockerFiles, envVars); err != nil {
			return nil, fmt.Errorf("failed to start Ethera Labs Console: %w", err)
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

// DeployOpSuccinctContractsOnly deploys op-succinct contracts and updates env files
// without starting op-succinct runtime services.
func (o *Orchestrator) DeployOpSuccinctContractsOnly(
	ctx context.Context,
	cfg configs.L2,
	disputeGameFactoryProxyAddresses map[configs.L2ChainName]common.Address,
	anchorStateRegistryProxyAddresses map[configs.L2ChainName]common.Address,
) error {
	enabledOpSuccinctChains := cfg.EnabledOpSuccinctChains()
	opSuccinctEnabled := isOpSuccinctEnabled(cfg, enabledOpSuccinctChains)
	if !opSuccinctEnabled {
		return fmt.Errorf("op-succinct is disabled or repository is not configured")
	}

	envBuilder := docker.NewEnvBuilder(o.rootDir, o.networksDir, o.servicesDir)
	envVars, err := envBuilder.BuildComposeEnv(cfg, common.Address{})
	if err != nil {
		return err
	}

	opSuccinctPath := envVars["OP_SUCCINCT_PATH"]
	if opSuccinctPath == "" {
		return fmt.Errorf("OP_SUCCINCT_PATH is empty")
	}

	if err := o.prepareOpSuccinctEnvFiles(cfg, envVars, disputeGameFactoryProxyAddresses, anchorStateRegistryProxyAddresses, opSuccinctPath); err != nil {
		return fmt.Errorf("failed to prepare op-succinct env files: %w", err)
	}
	if err := o.setupOpSuccinct(ctx, cfg, opSuccinctPath, envVars); err != nil {
		return fmt.Errorf("failed to run op-succinct setup calls: %w", err)
	}
	if err := o.syncOpSuccinctMultiEnvFiles(cfg, envVars); err != nil {
		return fmt.Errorf("failed to synchronize op-succinct multi env files: %w", err)
	}

	o.logger.Info("op-succinct contract deployment completed without starting op-succinct services")
	return nil
}

func (o *Orchestrator) restartOpGeth(ctx context.Context, dockerFilePath string, env map[string]string, deployedContracts map[configs.L2ChainName]map[contracts.ContractName]common.Address) error {
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
	if err := docker.ComposeRestart(ctx, dockerFilePath, env, services...); err != nil {
		return fmt.Errorf("failed to restart op-geth: %w", err)
	}

	o.logger.Info("op-geth services restarted successfully, waiting for them to be ready")

	return nil
}

func (o *Orchestrator) restartSidecar(ctx context.Context, dockerFilePath, flashblocksDockerPath, sidecarDockerPath string, env map[string]string) error {
	if sidecarDockerPath == "" {
		return fmt.Errorf("sidecar docker file path is empty")
	}

	dockerFiles := []string{dockerFilePath}
	if flashblocksDockerPath != "" {
		dockerFiles = append(dockerFiles, flashblocksDockerPath)
	}
	dockerFiles = append(dockerFiles, sidecarDockerPath)

	services := []string{"sidecar-a", "sidecar-b"}
	if err := docker.ComposeRestartMultiFile(ctx, dockerFiles, env, services...); err != nil {
		return fmt.Errorf("failed to restart sidecar: %w", err)
	}

	return nil
}

// buildComposeServices builds services that are sourced locally.
func (o *Orchestrator) buildComposeServices(
	ctx context.Context,
	composeFilePath string,
	env map[string]string,
	cfg configs.L2,
	opSuccinctBuildService string,
	localAltDAEnabled bool,
) error {
	services := []string{
		"publisher",
		"op-geth-a",
		"op-geth-b",
	}
	if localAltDAEnabled {
		services = append(services, "op-alt-da-a", "op-alt-da-b")
	}
	if opSuccinctBuildService != "" {
		services = append(services, opSuccinctBuildService)
	}

	dockerFiles := []string{composeFilePath}

	// Sidecar requires flashblocks, so add flashblocks docker file first
	if cfg.Sidecar.Enabled {
		flashblocksDockerPath, err := docker.EnsureFlashblocksComposeFile(o.localnetDir)
		if err != nil {
			return fmt.Errorf("failed to prepare flashblocks docker file for build: %w", err)
		}
		dockerFiles = append(dockerFiles, flashblocksDockerPath)

		sidecarDockerPath, err := docker.EnsureSidecarComposeFile(o.localnetDir)
		if err != nil {
			return fmt.Errorf("failed to prepare sidecar docker file for build: %w", err)
		}
		dockerFiles = append(dockerFiles, sidecarDockerPath)
		services = append(services, "sidecar-a", "sidecar-b")
	}

	if len(dockerFiles) > 1 {
		if err := docker.ComposeBuildMultiFile(ctx, dockerFiles, env, services...); err != nil {
			return fmt.Errorf("failed to build docker services: %w", err)
		}
	} else {
		if err := docker.ComposeBuild(ctx, composeFilePath, env, services...); err != nil {
			return fmt.Errorf("failed to build docker services: %w", err)
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

func isOpSuccinctEnabled(cfg configs.L2, enabledChains []configs.L2ChainName) bool {
	if len(enabledChains) == 0 {
		return false
	}

	repo, exists := cfg.Repositories[configs.RepositoryNameOpSuccinct]
	if !exists {
		return false
	}

	return repo.LocalPath != "" || repo.URL != "" || repo.Branch != ""
}

func opSuccinctBuildServiceName(enabledChains []configs.L2ChainName, opSuccinctEnabled bool) string {
	if !opSuccinctEnabled {
		return ""
	}

	for _, chain := range enabledChains {
		switch chain {
		case configs.L2ChainNameRollupA:
			return "op-succinct-a"
		case configs.L2ChainNameRollupB:
			return "op-succinct-b"
		}
	}

	return ""
}
