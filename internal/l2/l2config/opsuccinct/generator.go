package opsuccinct

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethera-labs/local-testnet/internal/logger"
	"github.com/ethereum/go-ethereum/common"
)

const (
	fileName   = "opsuccinct.env"
	dgfKey     = "DGF_ADDRESS"
	mailboxKey = "MAILBOX_ADDRESS"
)

type envConfig struct {
	dgfAddress     string
	mailboxAddress string
}

// Generator writes the per-chain contract metadata needed by later
// op-succinct runtime wiring.
type Generator struct {
	logger *slog.Logger
}

func NewGenerator() *Generator {
	return &Generator{
		logger: logger.Named("opsuccinct.env_generator"),
	}
}

// Generate writes opsuccinct.env to path with the dispute game factory address.
func (g *Generator) Generate(disputeGameFactoryAddr common.Address, path string) error {
	if disputeGameFactoryAddr == (common.Address{}) {
		return fmt.Errorf("could not generate %s, disputeGameFactoryAddr cannot be empty", fileName)
	}

	opsuccinctFilePath := filepath.Join(path, fileName)
	cfg := envConfig{
		dgfAddress: disputeGameFactoryAddr.Hex(),
	}
	if err := writeFile(opsuccinctFilePath, cfg); err != nil {
		return fmt.Errorf("failed to write %s: %w", fileName, err)
	}

	g.logger.With("path", opsuccinctFilePath).Info("file was successfully written")

	return nil
}

// SetMailboxAddress updates MAILBOX_ADDRESS in the opsuccinct.env file at path.
// The file must already exist; call Generate before calling this.
func (g *Generator) SetMailboxAddress(mailboxAddr common.Address, path string) error {
	if mailboxAddr == (common.Address{}) {
		return fmt.Errorf("could not update %s, mailboxAddr cannot be empty", fileName)
	}

	opsuccinctFilePath := filepath.Join(path, fileName)
	cfg, err := readFile(opsuccinctFilePath)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", fileName, err)
	}
	cfg.mailboxAddress = mailboxAddr.Hex()
	if err := writeFile(opsuccinctFilePath, cfg); err != nil {
		return fmt.Errorf("failed to update %s: %w", fileName, err)
	}

	g.logger.With("path", opsuccinctFilePath, "mailbox_address", mailboxAddr.Hex()).Info("mailbox address updated successfully")

	return nil
}

func readFile(path string) (envConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return envConfig{}, err
	}

	var cfg envConfig
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case dgfKey:
			cfg.dgfAddress = value
		case mailboxKey:
			cfg.mailboxAddress = value
		}
	}

	return cfg, nil
}

func writeFile(path string, cfg envConfig) error {
	lines := make([]string, 0, 2)
	if cfg.dgfAddress != "" {
		lines = append(lines, fmt.Sprintf("%s=%s", dgfKey, cfg.dgfAddress))
	}
	if cfg.mailboxAddress != "" {
		lines = append(lines, fmt.Sprintf("%s=%s", mailboxKey, cfg.mailboxAddress))
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0644)
}
