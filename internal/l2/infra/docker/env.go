package docker

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/path"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EnvBuilder constructs environment variables for docker-compose operations.
// It handles path resolution for both operator-supplied local checkouts
// (`local-path`) and workspace-managed clones under `.localnet/services`.
type EnvBuilder struct {
	rootDir     string
	networksDir string
	servicesDir string
}

func NewEnvBuilder(rootDir, networksDir, servicesDir string) *EnvBuilder {
	return &EnvBuilder{
		rootDir:     rootDir,
		networksDir: networksDir,
		servicesDir: servicesDir,
	}
}

// BuildComposeEnv builds environment variables for docker-compose.
// Settlement contract addresses may be zero during bootstrap before dispute
// contracts are known.
func (b *EnvBuilder) BuildComposeEnv(cfg configs.L2, gameFactoryAddr common.Address, anchorStateRegistryAddr common.Address) (map[string]string, error) {
	env := make(map[string]string)

	publisherPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNamePublisher], configs.RepositoryNamePublisher)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve publisher path: %w", err)
	}

	rollupAConfigPath := filepath.Join(b.networksDir, string(configs.L2ChainNameRollupA))
	rollupBConfigPath := filepath.Join(b.networksDir, string(configs.L2ChainNameRollupB))

	rollupAHost, err := path.GetHostPath(rollupAConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host path for rollup-a config: %w", err)
	}
	rollupBHost, err := path.GetHostPath(rollupBConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host path for rollup-b config: %w", err)
	}
	rootHost, err := path.GetHostPath(b.rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve host path for rootDir: %w", err)
	}

	env["ROOT_DIR"] = rootHost
	env["WALLET_PRIVATE_KEY"] = cfg.Wallet.PrivateKey
	env["WALLET_ADDRESS"] = cfg.Wallet.Address
	env["L1_EL_URL"] = cfg.L1ElURL
	env["L1_CL_URL"] = cfg.L1ClURL
	env["L1_CHAIN_ID"] = fmt.Sprintf("%d", cfg.L1ChainID)
	env["ETHERA_NETWORK_NAME"] = cfg.EtheraNetworkName
	env["COORDINATOR_PRIVATE_KEY"] = cfg.CoordinatorPrivateKey
	env["SEQUENCER_PRIVATE_KEY"] = cfg.CoordinatorPrivateKey

	env["PUBLISHER_PATH"] = publisherPath

	if cfg.OPSuccinct.Enabled {
		opSuccinctPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNameOPSuccinct], configs.RepositoryNameOPSuccinct)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve op-succinct path: %w", err)
		}
		env["OP_SUCCINCT_PATH"] = opSuccinctPath

		env["OP_SUCCINCT_DOCKERFILE"] = filepath.Join(opSuccinctPath, "validity", "Dockerfile.ethera")
		if cfg.AltDA.Enabled {
			env["OP_SUCCINCT_BUILD_FEATURES"] = "altda"
			env["OP_SUCCINCT_ALTDA_SERVER_A"] = "http://op-alt-da-a:3100"
			env["OP_SUCCINCT_ALTDA_SERVER_B"] = "http://op-alt-da-b:3100"
		} else {
			env["OP_SUCCINCT_BUILD_FEATURES"] = ""
		}
	}

	if cfg.Flashblocks.Enabled {
		opRbuilderPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNameOpRbuilder], configs.RepositoryNameOpRbuilder)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve op-rbuilder path: %w", err)
		}
		env["OP_RBUILDER_PATH"] = opRbuilderPath
	}

	// The validator EL's P2P secret + matching enode pubkey are always populated so the
	// flashblocks trusted-peer wiring has a value regardless of feature flags. Both op-reth
	// (--p2p-secret-key-hex) and op-besu (--node-private-key-file) boot with this same key, so
	// the enode op-rbuilder dials is stable across either client. Derived from the secret; one
	// source of truth.
	rethASK, rethAEnode, err := derivePeerKeys(cfg.Flashblocks.RollupAP2PSecretKeyHex)
	if err != nil {
		return nil, fmt.Errorf("rollup-a flashblocks p2p key: %w", err)
	}
	env["VALIDATOR_EL_A_P2P_SECRET_KEY_HEX"] = rethASK
	env["VALIDATOR_EL_A_ENODE_PUBKEY"] = rethAEnode

	rethBSK, rethBEnode, err := derivePeerKeys(cfg.Flashblocks.RollupBP2PSecretKeyHex)
	if err != nil {
		return nil, fmt.Errorf("rollup-b flashblocks p2p key: %w", err)
	}
	env["VALIDATOR_EL_B_P2P_SECRET_KEY_HEX"] = rethBSK
	env["VALIDATOR_EL_B_ENODE_PUBKEY"] = rethBEnode

	if cfg.Sidecar.Enabled {
		sidecarPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNameSidecar], configs.RepositoryNameSidecar)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve sidecar path: %w", err)
		}
		env["SIDECAR_PATH"] = sidecarPath
	}
	if cfg.Bundler.Enabled {
		bundlerPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNameBundler], configs.RepositoryNameBundler)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve bundler path: %w", err)
		}
		env["BUNDLER_PATH"] = bundlerPath
	}
	if cfg.CrossScout.Enabled {
		crossScoutPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNameCrossScout], configs.RepositoryNameCrossScout)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve cross-scout path: %w", err)
		}
		env["CROSS_SCOUT_PATH"] = crossScoutPath
		env["CROSS_SCOUT_API_PORT"] = fmt.Sprintf("%d", cfg.CrossScout.APIPort)
		env["CROSS_SCOUT_EXPLORER_PORT"] = fmt.Sprintf("%d", cfg.CrossScout.ExplorerPort)
		env["CROSS_SCOUT_POSTGRES_PORT"] = fmt.Sprintf("%d", cfg.CrossScout.PostgresPort)
		env["CROSS_SCOUT_REDIS_PORT"] = fmt.Sprintf("%d", cfg.CrossScout.RedisPort)
		env["CROSS_SCOUT_URL"] = fmt.Sprintf("http://localhost:%d", cfg.CrossScout.ExplorerPort)
		env["CROSS_SCOUT_DATABASE_URL"] = "postgres://crossscout:crossscout@cross-scout-postgres:5432/crossscout"
		env["CROSS_SCOUT_REDIS_URL"] = "redis://cross-scout-redis:6379"
		env["CROSS_SCOUT_MAILBOX_ADDRESS"] = (common.Address{}).Hex()
		env["CROSS_SCOUT_BRIDGE_ADDRESSES"] = (common.Address{}).Hex()
		env["CROSS_SCOUT_ANCHOR_STATE_REGISTRY_ADDRESS"] = anchorStateRegistryAddr.Hex()
		env["CROSS_SCOUT_CHAIN_NAMES"] = fmt.Sprintf(
			"%d=Rollup A,%d=Rollup B",
			cfg.ChainConfigs[configs.L2ChainNameRollupA].ID,
			cfg.ChainConfigs[configs.L2ChainNameRollupB].ID,
		)
		env["CROSS_SCOUT_EL_RPC_URLS"] = b.crossScoutELRPCURLs(cfg)
		env["CROSS_SCOUT_FLASHBLOCKS_WS_URLS"] = b.crossScoutFlashblocksWSURLs(cfg)
	}
	if cfg.AltDA.Enabled {
		composeContractsPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNameEtheraContracts], configs.RepositoryNameEtheraContracts)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve ethera-contracts path for AltDA DA server build: %w", err)
		}
		env["COMPOSE_CONTRACTS_PATH"] = composeContractsPath
	}
	env["ALTDA_USE_GENERIC_COMMITMENT"] = fmt.Sprintf("%t", cfg.AltDA.CommitmentType() == configs.AltDACommitmentTypeGeneric)

	env["ROLLUP_A_CHAIN_ID"] = fmt.Sprintf("%d", cfg.ChainConfigs[configs.L2ChainNameRollupA].ID)
	env["ROLLUP_A_RPC_PORT"] = fmt.Sprintf("%d", cfg.ChainConfigs[configs.L2ChainNameRollupA].RPCPort)
	env["ROLLUP_A_CONFIG_PATH"] = rollupAHost
	env["ROLLUP_A_CONFIG_PATH_CONTAINER"] = rollupAConfigPath

	env["ROLLUP_B_CHAIN_ID"] = fmt.Sprintf("%d", cfg.ChainConfigs[configs.L2ChainNameRollupB].ID)
	env["ROLLUP_B_RPC_PORT"] = fmt.Sprintf("%d", cfg.ChainConfigs[configs.L2ChainNameRollupB].RPCPort)
	env["ROLLUP_B_CONFIG_PATH"] = rollupBHost
	env["ROLLUP_B_CONFIG_PATH_CONTAINER"] = rollupBConfigPath

	env["FLASHBLOCKS_ROLLUP_A_RPC_PORT"] = fmt.Sprintf("%d", cfg.Flashblocks.RollupARPCPort)
	env["FLASHBLOCKS_ROLLUP_B_RPC_PORT"] = fmt.Sprintf("%d", cfg.Flashblocks.RollupBRPCPort)

	env["SIDECAR_ROLLUP_A_API_PORT"] = fmt.Sprintf("%d", cfg.Sidecar.RollupAAPIPort)
	env["SIDECAR_ROLLUP_B_API_PORT"] = fmt.Sprintf("%d", cfg.Sidecar.RollupBAPIPort)

	env["BUNDLER_ROLLUP_A_API_PORT"] = fmt.Sprintf("%d", cfg.Bundler.RollupAAPIPort)
	env["BUNDLER_ROLLUP_B_API_PORT"] = fmt.Sprintf("%d", cfg.Bundler.RollupBAPIPort)

	frontendPath, err := path.GetHostPath(filepath.Join(b.rootDir, "frontend"))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve frontend path: %w", err)
	}
	env["FRONTEND_PATH"] = frontendPath

	if cfg.Frontend.Active() && cfg.Frontend.Port > 0 {
		env["CONSOLE_PORT"] = fmt.Sprintf("%d", cfg.Frontend.Port)
	}

	env["SP_L1_DISPUTE_GAME_FACTORY"] = gameFactoryAddr.Hex()
	env["SP_L1_ANCHOR_STATE_REGISTRY"] = anchorStateRegistryAddr.Hex()

	env["OP_BATCHER_IMAGE_TAG"] = cfg.Images[configs.ImageNameOpBatcher].Tag
	env["OP_NODE_IMAGE_TAG"] = cfg.Images[configs.ImageNameOpNode].Tag
	env["OP_PROPOSER_IMAGE_TAG"] = cfg.Images[configs.ImageNameOpProposer].Tag
	env["OP_RETH_IMAGE_TAG"] = cfg.Images[configs.ImageNameOpReth].Tag

	// Contract-derived values (MAILBOX, ENTRYPOINT, SIMPLE_ACCOUNT_FACTORY) are
	// populated via MergePostDeployEnv. Called here so the first env snapshot
	// reflects any contracts.json that already exists on disk from a previous
	// run; the orchestrator calls MergePostDeployEnv again after Phase 3.
	b.MergePostDeployEnv(env)

	return env, nil
}

// ResolveRepoPath resolves the repository path for a given repository configuration.
// This is exported so other packages can resolve paths consistently.
// Config validation ensures URL and local-path are mutually exclusive.
// When URL is set, uses cloned repository path (.localnet/services/<name>).
// When local-path is set, uses the specified local path (for development).
// When running in Docker:
//   - Cloned paths stay as container paths (accessible via workspace mount)
//   - Local paths get translated to host paths (outside workspace mount)
func (b *EnvBuilder) ResolveRepoPath(repo configs.Repository, name configs.RepositoryName) (string, error) {
	// If URL is provided (via CLI or config), use cloned path
	// This ensures CLI flags like --op-reth-url override local-path from config
	// Cloned paths are inside the workspace mount, so they stay as container paths
	if repo.URL != "" {
		return filepath.Join(b.servicesDir, string(name)), nil
	}

	if repo.LocalPath != "" {
		expanded := expandUserHome(repo.LocalPath)

		var resolvedPath string
		if filepath.IsAbs(expanded) {
			resolvedPath = expanded
		} else {
			resolvedPath = filepath.Clean(filepath.Join(b.rootDir, expanded))
		}

		hostPath, err := path.GetHostPath(resolvedPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve host path for %s: %w", resolvedPath, err)
		}
		return hostPath, nil
	}

	return "", fmt.Errorf("repository %s has neither URL nor local-path set", name)
}

// MergePostDeployEnv re-reads the per-chain contracts.json files and injects
// the resulting addresses into the env map. Idempotent: safe to call before
// deployment (no-op when contracts.json is missing) and again after
// deployment to pick up the freshly-written values.
func (b *EnvBuilder) MergePostDeployEnv(env map[string]string) {
	mailboxA := b.readUniversalBridgeMailboxAddress(configs.L2ChainNameRollupA)
	mailboxB := b.readUniversalBridgeMailboxAddress(configs.L2ChainNameRollupB)
	if mailboxA != "" {
		env["MAILBOX_A"] = mailboxA
	}
	if mailboxB != "" {
		env["MAILBOX_B"] = mailboxB
	}
	// CrossScout takes a single mailbox address and applies it to every rollup
	// it ingests; localnet deploys the mailbox at the same address on both
	// chains, so either chain's value works.
	if mailboxA != "" {
		env["CROSS_SCOUT_MAILBOX_ADDRESS"] = mailboxA
	} else if mailboxB != "" {
		env["CROSS_SCOUT_MAILBOX_ADDRESS"] = mailboxB
	}
	if bridges := joinUniqueAddresses(
		b.readContractAddress(configs.L2ChainNameRollupA, "ComposeL2ToL2Bridge"),
		b.readContractAddress(configs.L2ChainNameRollupB, "ComposeL2ToL2Bridge"),
	); bridges != "" {
		env["CROSS_SCOUT_BRIDGE_ADDRESSES"] = bridges
	}
	if ep := b.readContractAddress(configs.L2ChainNameRollupA, "EntryPoint"); ep != "" {
		env["ENTRYPOINT_A"] = ep
	}
	if ep := b.readContractAddress(configs.L2ChainNameRollupB, "EntryPoint"); ep != "" {
		env["ENTRYPOINT_B"] = ep
	}
	if f := b.readContractAddress(configs.L2ChainNameRollupA, "SimpleAccountFactory"); f != "" {
		env["SIMPLE_ACCOUNT_FACTORY_A"] = f
	}
	if f := b.readContractAddress(configs.L2ChainNameRollupB, "SimpleAccountFactory"); f != "" {
		env["SIMPLE_ACCOUNT_FACTORY_B"] = f
	}
}

// joinUniqueAddresses joins non-empty addresses into a comma-separated list,
// dropping case-insensitive duplicates.
func joinUniqueAddresses(addrs ...string) string {
	seen := make(map[string]struct{}, len(addrs))
	unique := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr == "" {
			continue
		}
		key := strings.ToLower(addr)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, addr)
	}
	return strings.Join(unique, ",")
}

// readUniversalBridgeMailboxAddress reads the deployed UniversalBridgeMailbox
// address from the chain's contracts.json. Returns an empty string before the
// file is written or if the address is missing.
func (b *EnvBuilder) readUniversalBridgeMailboxAddress(chainName configs.L2ChainName) string {
	path := filepath.Join(b.networksDir, string(chainName), "contracts.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	var cf struct {
		Addresses map[string]string `json:"addresses"`
	}
	if err := json.Unmarshal(data, &cf); err != nil {
		return ""
	}

	return strings.TrimSpace(cf.Addresses["UniversalBridgeMailbox"])
}

func (b *EnvBuilder) crossScoutELRPCURLs(cfg configs.L2) string {
	rollupA := cfg.ChainConfigs[configs.L2ChainNameRollupA]
	rollupB := cfg.ChainConfigs[configs.L2ChainNameRollupB]
	rollupAURL := "http://validator-el-a:8545"
	rollupBURL := "http://validator-el-b:8545"
	if cfg.Flashblocks.Enabled {
		rollupAURL = "http://op-rbuilder-a:8545"
		rollupBURL = "http://op-rbuilder-b:8545"
	}
	return fmt.Sprintf("%d=%s,%d=%s", rollupA.ID, rollupAURL, rollupB.ID, rollupBURL)
}

func (b *EnvBuilder) crossScoutFlashblocksWSURLs(cfg configs.L2) string {
	if !cfg.Flashblocks.Enabled {
		return ""
	}
	rollupA := cfg.ChainConfigs[configs.L2ChainNameRollupA]
	rollupB := cfg.ChainConfigs[configs.L2ChainNameRollupB]
	return fmt.Sprintf("%d=ws://op-rbuilder-a:1111,%d=ws://op-rbuilder-b:1111", rollupA.ID, rollupB.ID)
}

// readContractAddress reads a named contract address from the chain's
// contracts.json. Returns "" when the file or entry is missing so callers can
// treat the address as optional.
func (b *EnvBuilder) readContractAddress(chainName configs.L2ChainName, contractName string) string {
	data, err := os.ReadFile(filepath.Join(b.networksDir, string(chainName), "contracts.json"))
	if err != nil {
		return ""
	}
	var cf struct {
		Addresses map[string]string `json:"addresses"`
	}
	if err := json.Unmarshal(data, &cf); err != nil {
		return ""
	}
	return strings.TrimSpace(cf.Addresses[contractName])
}

// derivePeerKeys takes a 32-byte hex-encoded secp256k1 secret and returns the
// normalized hex secret (no 0x prefix) plus the 64-byte uncompressed enode
// pubkey (no 0x04 prefix), as expected by `--p2p-secret-key-hex` and the enode
// URL format respectively.
func derivePeerKeys(secretHex string) (string, string, error) {
	sk := strings.TrimPrefix(secretHex, "0x")
	priv, err := crypto.HexToECDSA(sk)
	if err != nil {
		return "", "", fmt.Errorf("invalid secret: %w", err)
	}
	pub := crypto.FromECDSAPub(&priv.PublicKey) // 65 bytes: 0x04 || X || Y
	return sk, hex.EncodeToString(pub[1:]), nil
}

// expandUserHome expands a leading ~ to the current user's home directory.
// Returns the original path if expansion fails or is not needed.
func expandUserHome(p string) string {
	if p == "" || p[0] != '~' {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	return filepath.Join(home, p[2:])
}
