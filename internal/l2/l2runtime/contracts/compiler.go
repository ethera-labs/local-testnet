package contracts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethera-labs/local-testnet/internal/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

// Compiler compiles Solidity L2 contracts
type Compiler struct {
	contractsRootDir string
	outputDir        string
	logger           *slog.Logger
}

// NewCompiler creates a new contract compiler
func NewCompiler(contractsRootDir, outputDir string) *Compiler {
	return &Compiler{
		contractsRootDir: contractsRootDir,
		outputDir:        outputDir,
		logger:           logger.Named("contracts_compiler"),
	}
}

// Compile compiles Solidity contracts and persists the output to contracts.json.
func (c *Compiler) Compile(ctx context.Context, contractNames []string) error {
	return c.compileTo(ctx, contractNames, contractsFileName, false)
}

// CompileEntryPoint compiles ERC-4337 v0.7 EntryPoint + EntryPointSimulations
// from the configured account-abstraction source tree and writes the result to
// entrypoint.json. The output includes deployedBytecode for every compiled
// contract because the bundler needs the runtime bytecode for state-override
// eth_call simulation.
func (c *Compiler) CompileEntryPoint(ctx context.Context, contractNames []string) error {
	return c.compileTo(ctx, contractNames, entryPointFileName, true)
}

func (c *Compiler) compileTo(ctx context.Context, contractNames []string, outputFile string, includeDeployedBytecode bool) error {
	c.logger.
		With("contracts_dir", c.contractsRootDir).
		With("output", outputFile).
		Info("starting contract compilation")

	c.logger.Info("installing forge dependencies")
	if err := c.installDependencies(ctx); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	jsonContracts := make(map[string]map[string]any)
	for _, name := range contractNames {
		c.logger.With("name", name).Info("compiling contract")

		abiJSON, bytecodeHex, err := c.compileContractRaw(ctx, name)
		if err != nil {
			return fmt.Errorf("failed to compile %s: %w", name, err)
		}

		entry := map[string]any{
			"abi":      json.RawMessage(abiJSON),
			"bytecode": bytecodeHex,
		}

		if includeDeployedBytecode {
			deployedHex, err := c.inspectDeployedBytecode(ctx, name)
			if err != nil {
				return fmt.Errorf("failed to inspect deployedBytecode for %s: %w", name, err)
			}
			entry["deployedBytecode"] = deployedHex
		}

		jsonContracts[name] = entry
	}

	if err := c.writeJSON(jsonContracts, outputFile); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputFile, err)
	}

	c.logger.With("output", outputFile).Info("contracts compiled successfully")

	return nil
}

func (c *Compiler) installDependencies(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "forge", "install")
	cmd.Dir = c.contractsRootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("forge install failed: %w", err)
	}

	return nil
}

// compileContractRaw compiles a contract and returns raw JSON ABI and hex bytecode
func (c *Compiler) compileContractRaw(ctx context.Context, contractName string) ([]byte, string, error) {
	abiCmd := exec.CommandContext(ctx, "forge", "inspect", contractName, "abi", "--json")
	// Forge automatically looks for contracts in src/ subdirectory relative to the working directory
	abiCmd.Dir = c.contractsRootDir

	abiOutput, err := abiCmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get ABI for %s: %w", contractName, err)
	}

	// Validate that the ABI is valid JSON and parseable
	if _, err := abi.JSON(strings.NewReader(string(abiOutput))); err != nil {
		return nil, "", fmt.Errorf("failed to parse ABI for %s: %w", contractName, err)
	}

	bytecodeCmd := exec.CommandContext(ctx, "forge", "inspect", contractName, "bytecode")
	bytecodeCmd.Dir = c.contractsRootDir

	bytecodeOutput, err := bytecodeCmd.Output()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get bytecode for %s: %w", contractName, err)
	}

	bytecodeStr := strings.TrimSpace(string(bytecodeOutput))

	// Return raw ABI JSON and hex string with 0x prefix
	return abiOutput, bytecodeStr, nil
}

// inspectDeployedBytecode returns the runtime (deployed) bytecode for a contract.
func (c *Compiler) inspectDeployedBytecode(ctx context.Context, contractName string) (string, error) {
	cmd := exec.CommandContext(ctx, "forge", "inspect", contractName, "deployedBytecode")
	cmd.Dir = c.contractsRootDir

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get deployedBytecode for %s: %w", contractName, err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (c *Compiler) writeJSON(contracts map[string]map[string]any, outputFile string) error {
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(c.outputDir, outputFile)

	data, err := json.MarshalIndent(contracts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal contracts: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputFile, err)
	}

	return nil
}
