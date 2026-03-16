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

// Compile compiles Solidity contracts and persists the output
func (c *Compiler) Compile(ctx context.Context, contractNames []string) error {
	c.logger.
		With("contracts_dir", c.contractsRootDir).
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

		jsonContracts[string(name)] = map[string]any{
			"abi":      json.RawMessage(abiJSON),
			"bytecode": bytecodeHex,
		}
	}

	if err := c.writeContractsJSON(jsonContracts); err != nil {
		return fmt.Errorf("failed to write %s: %w", contractsFileName, err)
	}

	c.logger.Info("contracts compiled successfully")

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

func (c *Compiler) writeContractsJSON(contracts map[string]map[string]any) error {
	if err := os.MkdirAll(c.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	outputPath := filepath.Join(c.outputDir, contractsFileName)

	data, err := json.MarshalIndent(contracts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal contracts: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", contractsFileName, err)
	}

	return nil
}
