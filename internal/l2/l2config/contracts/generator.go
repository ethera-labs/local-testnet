package contracts

import (
	"fmt"
	"path/filepath"

	"github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem"
)

type Generator struct {
	writer filesystem.Writer
}

func NewGenerator(writer filesystem.Writer) *Generator {
	return &Generator{
		writer: writer,
	}
}

func (g *Generator) GeneratePlaceholders(path string, chainID int) error {
	// These will be updated in Phase 3 after deploying the actual contracts
	type contracts struct {
		ChainInfo map[string]any    `json:"chainInfo,omitempty"`
		Addresses map[string]string `json:"addresses,omitempty"`
	}

	contractsJSON := contracts{
		ChainInfo: map[string]any{
			"chainId": chainID,
		},
		Addresses: map[string]string{
			"UniversalBridgeMailbox": "0x0000000000000000000000000000000000000000",
			"CetFactory":             "0x0000000000000000000000000000000000000000",
			"ComposeETHLiquidity":    "0x0000000000000000000000000000000000000000",
			"ComposeL2ToL2Bridge":    "0x0000000000000000000000000000000000000000",
			"MockL2ERC20":            "0x0000000000000000000000000000000000000000",
		},
	}

	const fileName = "contracts.json"
	contractsPath := filepath.Join(path, fileName)

	if err := g.writer.WriteJSON(contractsPath, contractsJSON); err != nil {
		return fmt.Errorf("failed to write '%s' for chain %d: %w", fileName, chainID, err)
	}

	return nil
}
