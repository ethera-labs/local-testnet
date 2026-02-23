package docker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/path"
	"github.com/ethereum/go-ethereum/common"
)

// EnvBuilder constructs environment variables for docker-compose operations.
// It handles path resolution for both local development (local-path) and
// production (cloned repositories) scenarios.
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
// The gameFactoryAddr parameter can be empty (zero address) for dev deployments.
func (b *EnvBuilder) BuildComposeEnv(cfg configs.L2, gameFactoryAddr common.Address) (map[string]string, error) {
	env := make(map[string]string)

	publisherPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNamePublisher], configs.RepositoryNamePublisher)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve publisher path: %w", err)
	}

	opGethPath, err := b.ResolveRepoPath(cfg.Repositories[configs.RepositoryNameOpGeth], configs.RepositoryNameOpGeth)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve op-geth path: %w", err)
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
	env["SP_L1_SUPERBLOCK_CONTRACT"] = ""

	env["PUBLISHER_PATH"] = publisherPath
	env["OP_GETH_PATH"] = opGethPath

	env["ROLLUP_A_CHAIN_ID"] = fmt.Sprintf("%d", cfg.ChainConfigs[configs.L2ChainNameRollupA].ID)
	env["ROLLUP_A_RPC_PORT"] = fmt.Sprintf("%d", cfg.ChainConfigs[configs.L2ChainNameRollupA].RPCPort)
	env["ROLLUP_A_CONFIG_PATH"] = rollupAHost
	env["ROLLUP_A_CONFIG_PATH_CONTAINER"] = rollupAConfigPath

	env["ROLLUP_B_CHAIN_ID"] = fmt.Sprintf("%d", cfg.ChainConfigs[configs.L2ChainNameRollupB].ID)
	env["ROLLUP_B_RPC_PORT"] = fmt.Sprintf("%d", cfg.ChainConfigs[configs.L2ChainNameRollupB].RPCPort)
	env["ROLLUP_B_CONFIG_PATH"] = rollupBHost
	env["ROLLUP_B_CONFIG_PATH_CONTAINER"] = rollupBConfigPath

	rollupAL1Sender := resolveRollupL1Sender(cfg, configs.L2ChainNameRollupA)
	rollupBL1Sender := resolveRollupL1Sender(cfg, configs.L2ChainNameRollupB)
	env["ROLLUP_A_L1_SENDER_PRIVATE_KEY"] = rollupAL1Sender.PrivateKey
	env["ROLLUP_A_L1_SENDER_ADDRESS"] = rollupAL1Sender.Address
	env["ROLLUP_B_L1_SENDER_PRIVATE_KEY"] = rollupBL1Sender.PrivateKey
	env["ROLLUP_B_L1_SENDER_ADDRESS"] = rollupBL1Sender.Address

	env["FLASHBLOCKS_ROLLUP_A_RPC_PORT"] = fmt.Sprintf("%d", cfg.Flashblocks.RollupARPCPort)
	env["FLASHBLOCKS_ROLLUP_B_RPC_PORT"] = fmt.Sprintf("%d", cfg.Flashblocks.RollupBRPCPort)

	env["SIDECAR_ROLLUP_A_API_PORT"] = fmt.Sprintf("%d", cfg.Sidecar.RollupAAPIPort)
	env["SIDECAR_ROLLUP_B_API_PORT"] = fmt.Sprintf("%d", cfg.Sidecar.RollupBAPIPort)

	frontendPath, err := path.GetHostPath(filepath.Join(b.rootDir, "frontend"))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve frontend path: %w", err)
	}
	env["FRONTEND_PATH"] = frontendPath

	if cfg.Frontend.Enabled && cfg.Frontend.Port > 0 {
		env["CONSOLE_PORT"] = fmt.Sprintf("%d", cfg.Frontend.Port)
	}

	env["SP_L1_DISPUTE_GAME_FACTORY"] = gameFactoryAddr.Hex()

	env["OP_BATCHER_IMAGE_TAG"] = cfg.Images[configs.ImageNameOpBatcher].Tag
	env["OP_NODE_IMAGE_TAG"] = cfg.Images[configs.ImageNameOpNode].Tag
	env["OP_PROPOSER_IMAGE_TAG"] = cfg.Images[configs.ImageNameOpProposer].Tag

	env["ALTDA_ENABLED"] = fmt.Sprintf("%t", cfg.AltDA.Enabled)
	env["ALTDA_DA_SERVER"] = cfg.AltDA.DAServer
	env["ALTDA_VERIFY_ON_READ"] = fmt.Sprintf("%t", cfg.AltDA.VerifyOnRead)
	env["ALTDA_DA_SERVICE"] = fmt.Sprintf("%t", cfg.AltDA.DAService)
	env["ALTDA_PUT_TIMEOUT"] = cfg.AltDA.PutTimeout
	if env["ALTDA_PUT_TIMEOUT"] == "" {
		env["ALTDA_PUT_TIMEOUT"] = "0s"
	}
	env["ALTDA_GET_TIMEOUT"] = cfg.AltDA.GetTimeout
	if env["ALTDA_GET_TIMEOUT"] == "" {
		env["ALTDA_GET_TIMEOUT"] = "0s"
	}
	maxConcurrentRequests := cfg.AltDA.MaxConcurrentDARequests
	if maxConcurrentRequests == 0 {
		maxConcurrentRequests = 1
	}
	env["ALTDA_MAX_CONCURRENT_DA_REQUESTS"] = fmt.Sprintf("%d", maxConcurrentRequests)

	if ma := b.readMailboxAddress(configs.L2ChainNameRollupA); ma != "" {
		env["MAILBOX_A"] = ma
	}
	if mb := b.readMailboxAddress(configs.L2ChainNameRollupB); mb != "" {
		env["MAILBOX_B"] = mb
	}

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
	// This ensures CLI flags like --op-geth-url override local-path from config
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

// readMailboxAddress reads the mailbox address from contracts.json for a given chain.
// Returns empty string if file doesn't exist or address not found (best-effort).
func (b *EnvBuilder) readMailboxAddress(chainName configs.L2ChainName) string {
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

	if v := cf.Addresses["Mailbox"]; strings.TrimSpace(v) != "" {
		return v
	}

	return ""
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

func resolveRollupL1Sender(cfg configs.L2, chainName configs.L2ChainName) configs.Wallet {
	chainCfg, ok := cfg.ChainConfigs[chainName]
	if !ok {
		return cfg.Wallet
	}
	sender := cfg.Wallet
	if chainCfg.L1Sender.PrivateKey != "" {
		sender.PrivateKey = chainCfg.L1Sender.PrivateKey
	}
	if chainCfg.L1Sender.Address != "" {
		sender.Address = chainCfg.L1Sender.Address
	}
	return sender
}
