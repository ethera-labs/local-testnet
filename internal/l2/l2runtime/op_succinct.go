package l2runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethereum/go-ethereum/common"
)

var strictAddressRegex = regexp.MustCompile(`(?i)\b0x[0-9a-f]{40}\b`)
var labeledAddressRegex = regexp.MustCompile(`(?im)(?:contract\s+address|address)\s*:?\s*(0x[0-9a-f]{40})\b`)

const opSuccinctPrebuiltBinaryRelativePath = ".localnet-prebuilt/validity-proposer"

type opSuccinctInstance struct {
	chainName configs.L2ChainName
	envFile   string
}

func (o *Orchestrator) prepareOpSuccinctPrebuiltBinary(ctx context.Context, cfg configs.L2, opSuccinctPath string) error {
	feature := opSuccinctOracleFeature(cfg)
	targetDir := filepath.Join(o.localnetDir, "op-succinct", "cargo-target")
	dstPath := filepath.Join(opSuccinctPath, opSuccinctPrebuiltBinaryRelativePath)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create op-succinct cargo target dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create op-succinct prebuilt dir: %w", err)
	}

	args := []string{"build", "--bin", "validity", "--features", feature}
	profile := "debug"
	if cfg.OpSuccinct.ReleaseBuild {
		args = append(args, "--release")
		profile = "release"
	}
	cmd := exec.CommandContext(ctx, "cargo", args...)
	cmd.Dir = opSuccinctPath
	cmd.Env = append(os.Environ(),
		"CARGO_TARGET_DIR="+targetDir,
		"RUSTFLAGS=-A warnings",
	)

	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &output)

	o.logger.With("path", opSuccinctPath, "feature", feature, "profile", profile).Info("building host op-succinct validity binary")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build op-succinct validity binary in %s: %w", opSuccinctPath, err)
	}

	srcPath := filepath.Join(targetDir, profile, "validity")
	if err := copyFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to stage prebuilt op-succinct validity binary: %w", err)
	}
	if err := os.Chmod(dstPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod prebuilt op-succinct validity binary: %w", err)
	}

	o.logger.With("path", dstPath).Info("prepared prebuilt op-succinct validity binary")
	return nil
}

func (o *Orchestrator) prepareOpSuccinctEnvFiles(
	cfg configs.L2,
	composeEnv map[string]string,
	disputeGameFactoryAddresses map[configs.L2ChainName]common.Address,
	anchorStateRegistryAddresses map[configs.L2ChainName]common.Address,
	opSuccinctPath string,
) error {
	baseEnvPath := filepath.Join(opSuccinctPath, ".env")
	baseEnv, err := loadEnvFile(baseEnvPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", baseEnvPath, err)
	}

	instances := make([]struct {
		chainName     configs.L2ChainName
		envFileVar    string
		opNodeRPCPort int
	}, 0, 2)
	if cfg.IsOpSuccinctChainEnabled(configs.L2ChainNameRollupA) {
		instances = append(instances, struct {
			chainName     configs.L2ChainName
			envFileVar    string
			opNodeRPCPort int
		}{
			chainName:     configs.L2ChainNameRollupA,
			envFileVar:    "OP_SUCCINCT_ENV_FILE_A",
			opNodeRPCPort: 19545,
		})
	}
	if cfg.IsOpSuccinctChainEnabled(configs.L2ChainNameRollupB) {
		instances = append(instances, struct {
			chainName     configs.L2ChainName
			envFileVar    string
			opNodeRPCPort int
		}{
			chainName:     configs.L2ChainNameRollupB,
			envFileVar:    "OP_SUCCINCT_ENV_FILE_B",
			opNodeRPCPort: 29545,
		})
	}

	for _, instance := range instances {
		chainCfg, ok := cfg.ChainConfigs[instance.chainName]
		if !ok {
			return fmt.Errorf("missing chain config for %s", instance.chainName)
		}

		disputeGameFactory, ok := disputeGameFactoryAddresses[instance.chainName]
		if !ok || disputeGameFactory == (common.Address{}) {
			return fmt.Errorf("missing dispute game factory proxy address for %s", instance.chainName)
		}

		envFilePath := composeEnv[instance.envFileVar]
		if strings.TrimSpace(envFilePath) == "" {
			return fmt.Errorf("compose env %s is required", instance.envFileVar)
		}
		if err := os.MkdirAll(filepath.Dir(envFilePath), 0755); err != nil {
			return fmt.Errorf("failed to create op-succinct env directory for %s: %w", instance.chainName, err)
		}

		envVars := cloneEnvMap(baseEnv)
		sender := resolveRollupSender(cfg, instance.chainName)
		envVars["L1_RPC"] = hostAccessibleRPCURL(cfg.L1ElURL)
		envVars["L1_BEACON_RPC"] = hostAccessibleRPCURL(cfg.L1ClURL)
		envVars["L2_RPC"] = fmt.Sprintf("http://127.0.0.1:%d", chainCfg.RPCPort)
		envVars["L2_NODE_RPC"] = fmt.Sprintf("http://127.0.0.1:%d", instance.opNodeRPCPort)
		// Always force fresh per-run contract setup to avoid stale addresses copied from base .env.
		envVars["L2OO_ADDRESS"] = ""
		envVars["VERIFIER_ADDRESS"] = ""
		// Always use the resolved sender key to avoid nonce conflicts
		// with batcher/proposer when a dedicated deployer key is configured.
		envVars["PRIVATE_KEY"] = sender.PrivateKey
		envVars["PROPOSER_ADDRESSES"] = sender.Address
		if rustLog := strings.TrimSpace(envVars["RUST_LOG"]); rustLog == "" || strings.EqualFold(rustLog, "info") {
			envVars["RUST_LOG"] = "debug"
		}
		if strings.TrimSpace(envVars["RUST_BACKTRACE"]) == "" {
			envVars["RUST_BACKTRACE"] = "1"
		}
		// Reduce peak memory on constrained Docker Desktop environments.
		// SP1's program cache can be several GB; disabling it avoids startup OOMs.
		if strings.TrimSpace(envVars["SP1_DISABLE_PROGRAM_CACHE"]) == "" {
			envVars["SP1_DISABLE_PROGRAM_CACHE"] = "true"
		}
		// Force CPU prover path in local fake-proof mode; avoids accidental CUDA/network usage.
		if strings.TrimSpace(envVars["SP1_PROVER"]) == "" {
			envVars["SP1_PROVER"] = "cpu"
		}
		if strings.TrimSpace(envVars["CUDA_VISIBLE_DEVICES"]) == "" {
			envVars["CUDA_VISIBLE_DEVICES"] = "-1"
		}
		if strings.TrimSpace(envVars["SAFE_DB_FALLBACK"]) == "" {
			envVars["SAFE_DB_FALLBACK"] = "true"
		}
		envVars["DGF_ADDRESS"] = disputeGameFactory.Hex()
		if anchorStateRegistry, ok := anchorStateRegistryAddresses[instance.chainName]; ok && anchorStateRegistry != (common.Address{}) {
			envVars["ANCHOR_STATE_REGISTRY"] = anchorStateRegistry.Hex()
		}
		// Allow deploy-oracle to succeed immediately after rollup startup
		// without waiting for L2 finalization (default requires 1800 finalized blocks).
		if strings.TrimSpace(envVars["STARTING_BLOCK_NUMBER"]) == "" {
			envVars["STARTING_BLOCK_NUMBER"] = "1"
		}
		// Mailbox addresses are only known after L2 helper contracts are deployed.
		// Populate them early when available so repeated runs and contract-only flows
		// do not depend on later mutation.
		mailboxAddress, err := lookupOpSuccinctMailboxAddress(instance.chainName, composeEnv)
		if err != nil {
			return err
		}
		if mailboxAddress != "" {
			envVars["MAILBOX_ADDRESS"] = mailboxAddress
		}
		if cfg.IsLocalOpAltDAEnabled() {
			altDAServer, err := opSuccinctHostAltDAServerURL(instance.chainName, cfg.AltDA.DAServer)
			if err != nil {
				return err
			}
			envVars["ALTDA_DA_SERVER"] = altDAServer
			envVars["ALTDA_SERVER_URL"] = altDAServer
			envVars["ALTDA_VERIFY_ON_READ"] = strconv.FormatBool(cfg.AltDA.VerifyOnRead)
		}

		if err := writeEnvFile(envFilePath, envVars); err != nil {
			return fmt.Errorf("failed to write op-succinct env file for %s: %w", instance.chainName, err)
		}
	}

	return nil
}

func (o *Orchestrator) setupOpSuccinct(ctx context.Context, cfg configs.L2, opSuccinctPath string, composeEnv map[string]string) error {
	recipes, err := o.listJustRecipes(ctx, opSuccinctPath)
	if err != nil {
		return err
	}

	useSetDisputeGameCalls := recipes["set-dispute-game-impl"] && recipes["set-dispute-game-factory"]
	useDeployDisputeGameFactory := recipes["deploy-dispute-game-factory"]
	if !useSetDisputeGameCalls && !useDeployDisputeGameFactory {
		return fmt.Errorf("required op-succinct just recipes are missing; expected either set-dispute-game-impl + set-dispute-game-factory, or deploy-dispute-game-factory")
	}

	instances := make([]opSuccinctInstance, 0, 2)
	if cfg.IsOpSuccinctChainEnabled(configs.L2ChainNameRollupA) {
		instances = append(instances, opSuccinctInstance{
			chainName: configs.L2ChainNameRollupA,
			envFile:   composeEnv["OP_SUCCINCT_ENV_FILE_A"],
		})
	}
	if cfg.IsOpSuccinctChainEnabled(configs.L2ChainNameRollupB) {
		instances = append(instances, opSuccinctInstance{
			chainName: configs.L2ChainNameRollupB,
			envFile:   composeEnv["OP_SUCCINCT_ENV_FILE_B"],
		})
	}

	workEnvPath := filepath.Join(opSuccinctPath, ".env")
	originalWorkEnv, err := os.ReadFile(workEnvPath)
	if err != nil {
		return fmt.Errorf("failed to read op-succinct base env file %s: %w", workEnvPath, err)
	}
	defer func() {
		if restoreErr := os.WriteFile(workEnvPath, originalWorkEnv, 0600); restoreErr != nil {
			o.logger.With("error", restoreErr).Warn("failed to restore original op-succinct .env")
		}
	}()

	for _, instance := range instances {
		if strings.TrimSpace(instance.envFile) == "" {
			return fmt.Errorf("op-succinct env file path is empty for %s", instance.chainName)
		}

		mailboxAddress, err := lookupOpSuccinctMailboxAddress(instance.chainName, composeEnv)
		if err != nil {
			return err
		}
		if mailboxAddress == "" {
			key, err := opSuccinctMailboxComposeKey(instance.chainName)
			if err != nil {
				return err
			}
			return fmt.Errorf("MAILBOX_ADDRESS is not available for %s: compose env %s is empty", instance.chainName, key)
		}
		if err := setEnvValue(instance.envFile, "MAILBOX_ADDRESS", mailboxAddress); err != nil {
			return fmt.Errorf("failed to set MAILBOX_ADDRESS for %s: %w", instance.chainName, err)
		}

		o.logger.With("chain", instance.chainName, "env_file", instance.envFile).Info("running op-succinct setup calls")
		if err := copyFile(instance.envFile, workEnvPath); err != nil {
			return fmt.Errorf("failed to activate op-succinct env for %s: %w", instance.chainName, err)
		}

		mockVerifierOutput, err := o.runJustCommand(ctx, opSuccinctPath, "deploy-mock-verifier")
		if err != nil {
			return fmt.Errorf("deploy-mock-verifier failed for %s: %w", instance.chainName, err)
		}
		if verifierAddress, ok := extractLastAddress(mockVerifierOutput); ok {
			if err := setEnvValue(instance.envFile, "VERIFIER_ADDRESS", verifierAddress); err != nil {
				return fmt.Errorf("failed to set VERIFIER_ADDRESS for %s: %w", instance.chainName, err)
			}
			if err := setEnvValue(workEnvPath, "VERIFIER_ADDRESS", verifierAddress); err != nil {
				return fmt.Errorf("failed to update active VERIFIER_ADDRESS for %s: %w", instance.chainName, err)
			}
		} else {
			existingVerifier := mustGetEnvValue(instance.envFile, "VERIFIER_ADDRESS")
			if existingVerifier == "" {
				return fmt.Errorf("could not determine VERIFIER_ADDRESS for %s", instance.chainName)
			}
		}

		// Always pass an explicit feature so deploy-oracle rebuilds fetch-l2oo-config for the
		// currently selected DA path instead of reusing a stale binary from a previous mode.
		oracleArgs := []string{"deploy-oracle", ".env", opSuccinctOracleFeature(cfg)}
		oracleOutput, err := o.runJustCommand(ctx, opSuccinctPath, oracleArgs...)
		if err != nil {
			return fmt.Errorf("deploy-oracle failed for %s: %w", instance.chainName, err)
		}
		if l2ooAddress, ok := extractLastAddress(oracleOutput); ok {
			if err := setEnvValue(instance.envFile, "L2OO_ADDRESS", l2ooAddress); err != nil {
				return fmt.Errorf("failed to set L2OO_ADDRESS for %s: %w", instance.chainName, err)
			}
			if err := setEnvValue(workEnvPath, "L2OO_ADDRESS", l2ooAddress); err != nil {
				return fmt.Errorf("failed to update active L2OO_ADDRESS for %s: %w", instance.chainName, err)
			}
		} else {
			existingL2OO := mustGetEnvValue(instance.envFile, "L2OO_ADDRESS")
			if existingL2OO == "" {
				return fmt.Errorf("could not determine L2OO_ADDRESS for %s", instance.chainName)
			}
		}

		if useSetDisputeGameCalls {
			// set-dispute-game-impl calls factory.setImplementation on the DGF,
			// which is owned by the wallet key (deployer from Phase 1).
			// Temporarily swap PRIVATE_KEY to the wallet key for this call.
			if err := setEnvValue(workEnvPath, "PRIVATE_KEY", cfg.Wallet.PrivateKey); err != nil {
				return fmt.Errorf("failed to set wallet key for set-dispute-game-impl (%s): %w", instance.chainName, err)
			}
			if _, err := o.runJustCommand(ctx, opSuccinctPath, "set-dispute-game-impl"); err != nil {
				return fmt.Errorf("set-dispute-game-impl failed for %s: %w", instance.chainName, err)
			}

			// set-dispute-game-factory calls setDisputeGameFactory on the L2OO,
			// which is owned by the deployer key (deployed in deploy-oracle above).
			sender := resolveRollupSender(cfg, instance.chainName)
			if err := setEnvValue(workEnvPath, "PRIVATE_KEY", sender.PrivateKey); err != nil {
				return fmt.Errorf("failed to restore deployer key for set-dispute-game-factory (%s): %w", instance.chainName, err)
			}
			if _, err := o.runJustCommand(ctx, opSuccinctPath, "set-dispute-game-factory"); err != nil {
				return fmt.Errorf("set-dispute-game-factory failed for %s: %w", instance.chainName, err)
			}
			continue
		}

		disputeGameFactoryOutput, err := o.runJustCommand(ctx, opSuccinctPath, "deploy-dispute-game-factory")
		if err != nil {
			return fmt.Errorf("deploy-dispute-game-factory failed for %s: %w", instance.chainName, err)
		}
		if disputeGameFactoryAddress, ok := extractLastAddress(disputeGameFactoryOutput); ok {
			if err := setEnvValue(instance.envFile, "DGF_ADDRESS", disputeGameFactoryAddress); err != nil {
				return fmt.Errorf("failed to set DGF_ADDRESS for %s: %w", instance.chainName, err)
			}
			if err := setEnvValue(workEnvPath, "DGF_ADDRESS", disputeGameFactoryAddress); err != nil {
				return fmt.Errorf("failed to update active DGF_ADDRESS for %s: %w", instance.chainName, err)
			}
		} else {
			existingDGF := mustGetEnvValue(instance.envFile, "DGF_ADDRESS")
			if existingDGF == "" {
				return fmt.Errorf("could not determine DGF_ADDRESS for %s", instance.chainName)
			}
		}
	}

	return nil
}

func (o *Orchestrator) finalizeOpSuccinctRuntimeEnvFiles(cfg configs.L2, composeEnv map[string]string) error {
	instances := make([]opSuccinctInstance, 0, 2)
	if cfg.IsOpSuccinctChainEnabled(configs.L2ChainNameRollupA) {
		instances = append(instances, opSuccinctInstance{
			chainName: configs.L2ChainNameRollupA,
			envFile:   composeEnv["OP_SUCCINCT_ENV_FILE_A"],
		})
	}
	if cfg.IsOpSuccinctChainEnabled(configs.L2ChainNameRollupB) {
		instances = append(instances, opSuccinctInstance{
			chainName: configs.L2ChainNameRollupB,
			envFile:   composeEnv["OP_SUCCINCT_ENV_FILE_B"],
		})
	}

	for _, instance := range instances {
		if strings.TrimSpace(instance.envFile) == "" {
			return fmt.Errorf("op-succinct env file path is empty for %s", instance.chainName)
		}

		envVars, err := loadEnvFile(instance.envFile)
		if err != nil {
			return fmt.Errorf("failed to read op-succinct env file %s: %w", instance.envFile, err)
		}

		l2RPC, l2NodeRPC, err := opSuccinctRuntimeRPCURLs(instance.chainName)
		if err != nil {
			return err
		}

		envVars["L1_RPC"] = cfg.L1ElURL
		envVars["L1_BEACON_RPC"] = cfg.L1ClURL
		envVars["L2_RPC"] = l2RPC
		envVars["L2_NODE_RPC"] = l2NodeRPC
		if rustLog := strings.TrimSpace(envVars["RUST_LOG"]); rustLog == "" || strings.EqualFold(rustLog, "info") {
			envVars["RUST_LOG"] = "debug"
		}
		if strings.TrimSpace(envVars["RUST_BACKTRACE"]) == "" {
			envVars["RUST_BACKTRACE"] = "1"
		}
		if strings.TrimSpace(envVars["SP1_DISABLE_PROGRAM_CACHE"]) == "" {
			envVars["SP1_DISABLE_PROGRAM_CACHE"] = "true"
		}
		if strings.TrimSpace(envVars["SP1_PROVER"]) == "" {
			envVars["SP1_PROVER"] = "cpu"
		}
		if strings.TrimSpace(envVars["CUDA_VISIBLE_DEVICES"]) == "" {
			envVars["CUDA_VISIBLE_DEVICES"] = "-1"
		}
		mailboxAddress, err := lookupOpSuccinctMailboxAddress(instance.chainName, composeEnv)
		if err != nil {
			return err
		}
		if mailboxAddress == "" {
			key, err := opSuccinctMailboxComposeKey(instance.chainName)
			if err != nil {
				return err
			}
			return fmt.Errorf("MAILBOX_ADDRESS is not available for %s: compose env %s is empty", instance.chainName, key)
		}
		envVars["MAILBOX_ADDRESS"] = mailboxAddress
		if cfg.IsLocalOpAltDAEnabled() {
			altDAServer, err := opSuccinctRuntimeAltDAServerURL(instance.chainName)
			if err != nil {
				return err
			}
			envVars["ALTDA_DA_SERVER"] = altDAServer
			envVars["ALTDA_SERVER_URL"] = altDAServer
			envVars["ALTDA_VERIFY_ON_READ"] = strconv.FormatBool(cfg.AltDA.VerifyOnRead)
		}

		if err := writeEnvFile(instance.envFile, envVars); err != nil {
			return fmt.Errorf("failed to write op-succinct runtime env file for %s: %w", instance.chainName, err)
		}
	}

	return nil
}

func (o *Orchestrator) syncOpSuccinctMultiEnvFiles(cfg configs.L2, composeEnv map[string]string) error {
	instances := make([]opSuccinctInstance, 0, 2)
	if cfg.IsOpSuccinctChainEnabled(configs.L2ChainNameRollupA) {
		instances = append(instances, opSuccinctInstance{
			chainName: configs.L2ChainNameRollupA,
			envFile:   composeEnv["OP_SUCCINCT_ENV_FILE_A"],
		})
	}
	if cfg.IsOpSuccinctChainEnabled(configs.L2ChainNameRollupB) {
		instances = append(instances, opSuccinctInstance{
			chainName: configs.L2ChainNameRollupB,
			envFile:   composeEnv["OP_SUCCINCT_ENV_FILE_B"],
		})
	}

	for _, instance := range instances {
		if strings.TrimSpace(instance.envFile) == "" {
			return fmt.Errorf("op-succinct env file path is empty for %s", instance.chainName)
		}

		sourceEnv, err := loadEnvFile(instance.envFile)
		if err != nil {
			return fmt.Errorf("failed to read source op-succinct env file %s: %w", instance.envFile, err)
		}

		multiEnvPath := strings.TrimSuffix(instance.envFile, ".env") + ".multi.env"
		multiEnv := make(map[string]string)
		existingMultiEnv, err := loadEnvFile(multiEnvPath)
		switch {
		case err == nil:
			multiEnv = existingMultiEnv
		case errors.Is(err, os.ErrNotExist):
		default:
			return fmt.Errorf("failed to read existing op-succinct multi env file %s: %w", multiEnvPath, err)
		}

		for key, value := range sourceEnv {
			multiEnv[key] = value
		}

		if err := writeEnvFile(multiEnvPath, multiEnv); err != nil {
			return fmt.Errorf("failed to write op-succinct multi env file for %s: %w", instance.chainName, err)
		}
	}

	return nil
}

func (o *Orchestrator) runJustCommand(ctx context.Context, workingDir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "just", args...)
	cmd.Dir = workingDir
	// Set a high priority fee so forge script transactions land quickly
	// on congested L1 networks (e.g. Hoodi with a large mempool backlog).
	env := os.Environ()
	env = appendEnvIfUnset(env, "ETH_GAS_PRICE", "10gwei")
	env = appendEnvIfUnset(env, "ETH_PRIORITY_GAS_PRICE", "3gwei")
	cmd.Env = env

	var output bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &output)
	cmd.Stderr = io.MultiWriter(os.Stderr, &output)

	if err := cmd.Run(); err != nil {
		return output.String(), fmt.Errorf("command 'just %s' failed in %s: %w", strings.Join(args, " "), workingDir, err)
	}

	return output.String(), nil
}

func appendEnvIfUnset(environ []string, key, value string) []string {
	prefix := key + "="
	for _, e := range environ {
		if strings.HasPrefix(e, prefix) {
			return environ
		}
	}
	return append(environ, prefix+value)
}

func (o *Orchestrator) listJustRecipes(ctx context.Context, workingDir string) (map[string]bool, error) {
	cmd := exec.CommandContext(ctx, "just", "--summary")
	cmd.Dir = workingDir

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to list just recipes in %s: %w", workingDir, err)
	}

	recipes := make(map[string]bool)
	for _, token := range strings.Fields(output.String()) {
		recipes[token] = true
	}

	return recipes, nil
}

func loadEnvFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	env := make(map[string]string)
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if strings.HasPrefix(value, "#") {
			value = ""
		} else if idx := strings.Index(value, " #"); idx >= 0 {
			value = strings.TrimSpace(value[:idx])
		}
		value = strings.Trim(value, `"'`)
		env[key] = value
	}

	return env, nil
}

func writeEnvFile(path string, env map[string]string) error {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	var builder strings.Builder
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(env[key])
		builder.WriteByte('\n')
	}

	return os.WriteFile(path, []byte(builder.String()), 0600)
}

func setEnvValue(path, key, value string) error {
	env, err := loadEnvFile(path)
	if err != nil {
		return err
	}
	env[key] = value
	return writeEnvFile(path, env)
}

func mustGetEnvValue(path, key string) string {
	env, err := loadEnvFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(env[key])
}

func extractLastAddress(text string) (string, bool) {
	labeledMatches := labeledAddressRegex.FindAllStringSubmatch(text, -1)
	if len(labeledMatches) > 0 {
		return labeledMatches[len(labeledMatches)-1][1], true
	}

	addresses := strictAddressRegex.FindAllString(text, -1)
	if len(addresses) > 0 {
		return addresses[len(addresses)-1], true
	}

	return "", false
}

func resolveRollupSender(cfg configs.L2, chainName configs.L2ChainName) configs.Wallet {
	// Prefer the dedicated op-succinct deployer wallet to avoid nonce
	// conflicts with the batcher/proposer which share the main wallet key.
	if cfg.OpSuccinct.Deployer.PrivateKey != "" {
		return cfg.OpSuccinct.Deployer
	}

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

func opSuccinctMailboxComposeKey(chainName configs.L2ChainName) (string, error) {
	switch chainName {
	case configs.L2ChainNameRollupA:
		return "MAILBOX_A", nil
	case configs.L2ChainNameRollupB:
		return "MAILBOX_B", nil
	default:
		return "", fmt.Errorf("unsupported chain for op-succinct mailbox address: %s", chainName)
	}
}

func lookupOpSuccinctMailboxAddress(chainName configs.L2ChainName, composeEnv map[string]string) (string, error) {
	key, err := opSuccinctMailboxComposeKey(chainName)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(composeEnv[key]), nil
}

func cloneEnvMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func copyFile(srcPath, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, data, 0600)
}

func opSuccinctHostAltDAServerURL(chainName configs.L2ChainName, configuredURL string) (string, error) {
	var targetPort string
	switch chainName {
	case configs.L2ChainNameRollupA:
		targetPort = "3100"
	case configs.L2ChainNameRollupB:
		targetPort = "3101"
	default:
		return "", fmt.Errorf("unsupported chain for op-succinct host AltDA URL: %s", chainName)
	}

	parsed, err := url.Parse(strings.TrimSpace(configuredURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Sprintf("http://127.0.0.1:%s", targetPort), nil
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		hostname = "127.0.0.1"
	}

	parsed.Host = net.JoinHostPort(hostname, targetPort)
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func opSuccinctRuntimeAltDAServerURL(chainName configs.L2ChainName) (string, error) {
	switch chainName {
	case configs.L2ChainNameRollupA:
		return "http://op-alt-da-a:3100", nil
	case configs.L2ChainNameRollupB:
		return "http://op-alt-da-b:3100", nil
	default:
		return "", fmt.Errorf("unsupported chain for op-succinct runtime AltDA URL: %s", chainName)
	}
}

func opSuccinctOracleFeature(cfg configs.L2) string {
	if cfg.IsLocalOpAltDAEnabled() {
		return "altda"
	}
	return "ethereum"
}

func opSuccinctRuntimeRPCURLs(chainName configs.L2ChainName) (string, string, error) {
	switch chainName {
	case configs.L2ChainNameRollupA:
		return "http://op-geth-a:8545", "http://op-node-a:9545", nil
	case configs.L2ChainNameRollupB:
		return "http://op-geth-b:8545", "http://op-node-b:9545", nil
	default:
		return "", "", fmt.Errorf("unsupported chain for op-succinct runtime env: %s", chainName)
	}
}

func hostAccessibleRPCURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if parsed.Hostname() != "host.docker.internal" {
		return raw
	}

	port := parsed.Port()
	if port == "" {
		parsed.Host = "127.0.0.1"
		return parsed.String()
	}

	parsed.Host = net.JoinHostPort("127.0.0.1", port)
	return parsed.String()
}

func isTruthyEnvValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
