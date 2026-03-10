package rollup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem/json"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

const rollupConfigFileName = "rollup.json"

// Generator generates rollup.json files for L2 chains
type (
	deployer interface {
		InspectRollup(ctx context.Context, chainID int, outputPath string) error
	}

	Generator struct {
		reader      filesystem.Reader
		deployer    deployer
		writer      filesystem.Writer
		localnetDir string
		logger      *slog.Logger
	}
)

// NewGenerator creates a new rollup generator
func NewGenerator(reader filesystem.Reader, deployer deployer, writer filesystem.Writer, localnetDir string) *Generator {
	return &Generator{
		reader:      json.NewReader(),
		deployer:    deployer,
		writer:      writer,
		localnetDir: localnetDir,
		logger:      logger.Named("rollup_generator"),
	}
}

// Generate generates rollup config for a chain
func (g *Generator) Generate(ctx context.Context, chainID int, path string, genesisHash, l1BlockHash, l1BlockNumber string) error {
	logger := g.logger.With("chain_id", chainID)
	logger.Info("generating rollup config for chain")

	// Create temp directories under .localnet/.tmp/ to make them accessible when running in Docker
	tmpBaseDir := filepath.Join(g.localnetDir, ".tmp")
	if err := os.MkdirAll(tmpBaseDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp base dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp(tmpBaseDir, "rollup-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	rollupPath := filepath.Join(tmpDir, fmt.Sprintf("rollup-%d.json", chainID))
	if err := g.deployer.InspectRollup(ctx, chainID, rollupPath); err != nil {
		return fmt.Errorf("failed to inspect rollup: %w", err)
	}

	var rollup map[string]any
	if err := g.reader.ReadJSON(rollupPath, &rollup); err != nil {
		return fmt.Errorf("failed to read rollup config: %w", err)
	}

	l1BlockNum, err := parseHexNumber(l1BlockNumber)
	if err != nil {
		return fmt.Errorf("failed to parse L1 block number %s: %w", l1BlockNumber, err)
	}

	genesis, ok := rollup["genesis"].(map[string]any)
	if !ok {
		genesis = make(map[string]any)
		rollup["genesis"] = genesis
	}

	l1, ok := genesis["l1"].(map[string]any)
	if !ok {
		l1 = make(map[string]any)
		genesis["l1"] = l1
	}
	l1["hash"] = l1BlockHash
	l1["number"] = l1BlockNum

	l2, ok := genesis["l2"].(map[string]any)
	if !ok {
		l2 = make(map[string]any)
		genesis["l2"] = l2
	}
	l2["hash"] = genesisHash
	l2["number"] = 0

	rollup["isthmus_time"] = 0

	rollupPath = filepath.Join(path, rollupConfigFileName)

	logger.With("file_path", rollupPath).Info("rollup config generated successfully. Persisting file")

	if err := g.writer.WriteJSON(rollupPath, rollup); err != nil {
		return fmt.Errorf("failed to write '%s' for chain %d: %w", rollupConfigFileName, chainID, err)
	}

	return nil
}

func parseHexNumber(hexStr string) (uint64, error) {
	hexStr = strings.TrimPrefix(hexStr, "0x")

	num, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid hex number: %w", err)
	}

	return num, nil
}
