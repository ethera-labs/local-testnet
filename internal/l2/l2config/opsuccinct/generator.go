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

const fileName = "opsuccinct.env"

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

func (g *Generator) Generate(composeL2OutputOracleAddr, disputeGameFactoryAddr common.Address, path string) error {
	if composeL2OutputOracleAddr == (common.Address{}) {
		return fmt.Errorf("could not generate %s, composeL2OutputOracleAddr cannot be empty", fileName)
	}
	if disputeGameFactoryAddr == (common.Address{}) {
		return fmt.Errorf("could not generate %s, disputeGameFactoryAddr cannot be empty", fileName)
	}

	opsuccinctFilePath := filepath.Join(path, fileName)

	lines := []string{
		fmt.Sprintf("COMPOSE_L2_OUTPUT_ORACLE_ADDRESS=%s", composeL2OutputOracleAddr.Hex()),
		fmt.Sprintf("DGF_ADDRESS=%s", disputeGameFactoryAddr.Hex()),
	}
	content := strings.Join(lines, "\n") + "\n"

	if err := os.WriteFile(opsuccinctFilePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", fileName, err)
	}

	g.logger.With("path", opsuccinctFilePath).Info("file was successfully written")

	return nil
}
