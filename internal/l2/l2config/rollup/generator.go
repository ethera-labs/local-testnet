package rollup

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"log/slog"
	"math"
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

	if err := normalizeHoloceneEIP1559Params(rollup); err != nil {
		return fmt.Errorf("failed to normalize holocene eip1559 params: %w", err)
	}

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

func normalizeHoloceneEIP1559Params(rollup map[string]any) error {
	holoceneTime, hasHoloceneTime, err := readUint64Field(rollup, "holocene_time")
	if err != nil {
		return fmt.Errorf("invalid holocene_time: %w", err)
	}
	if !hasHoloceneTime || holoceneTime != 0 {
		return nil
	}

	chainOpConfig, ok := rollup["chain_op_config"].(map[string]any)
	if !ok {
		return fmt.Errorf("missing chain_op_config for holocene-at-genesis rollup")
	}

	elasticity, hasElasticity, err := readUint32Field(chainOpConfig, "eip1559Elasticity")
	if err != nil {
		return fmt.Errorf("invalid chain_op_config.eip1559Elasticity: %w", err)
	}
	if !hasElasticity {
		return fmt.Errorf("missing chain_op_config.eip1559Elasticity for holocene-at-genesis rollup")
	}

	denominator, hasDenominator, err := readUint32Field(chainOpConfig, "eip1559DenominatorCanyon")
	if err != nil {
		return fmt.Errorf("invalid chain_op_config.eip1559DenominatorCanyon: %w", err)
	}
	if !hasDenominator {
		denominator, hasDenominator, err = readUint32Field(chainOpConfig, "eip1559Denominator")
		if err != nil {
			return fmt.Errorf("invalid chain_op_config.eip1559Denominator: %w", err)
		}
	}
	if !hasDenominator {
		return fmt.Errorf("missing chain_op_config eip1559 denominator for holocene-at-genesis rollup")
	}

	genesis := ensureMap(rollup, "genesis")
	systemConfig := ensureMap(genesis, "system_config")
	systemConfig["eip1559Params"] = encodeHoloceneEIP1559Params(denominator, elasticity)

	return nil
}

func ensureMap(parent map[string]any, key string) map[string]any {
	if m, ok := parent[key].(map[string]any); ok {
		return m
	}

	m := make(map[string]any)
	parent[key] = m

	return m
}

func encodeHoloceneEIP1559Params(denominator, elasticity uint32) string {
	packed := (uint64(denominator) << 32) | uint64(elasticity)
	return fmt.Sprintf("0x%016x", packed)
}

func readUint32Field(object map[string]any, key string) (uint32, bool, error) {
	value, ok, err := readUint64Field(object, key)
	if err != nil {
		return 0, ok, err
	}
	if !ok {
		return 0, false, nil
	}
	if value > math.MaxUint32 {
		return 0, true, fmt.Errorf("value %d exceeds uint32", value)
	}

	return uint32(value), true, nil
}

func readUint64Field(object map[string]any, key string) (uint64, bool, error) {
	rawValue, ok := object[key]
	if !ok || rawValue == nil {
		return 0, false, nil
	}

	value, err := parseUint64(rawValue)
	if err != nil {
		return 0, true, err
	}

	return value, true, nil
}

func parseUint64(value any) (uint64, error) {
	switch v := value.(type) {
	case uint8:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case uint:
		return uint64(v), nil
	case int8:
		if v < 0 {
			return 0, fmt.Errorf("negative integer value: %d", v)
		}
		return uint64(v), nil
	case int16:
		if v < 0 {
			return 0, fmt.Errorf("negative integer value: %d", v)
		}
		return uint64(v), nil
	case int32:
		if v < 0 {
			return 0, fmt.Errorf("negative integer value: %d", v)
		}
		return uint64(v), nil
	case int64:
		if v < 0 {
			return 0, fmt.Errorf("negative integer value: %d", v)
		}
		return uint64(v), nil
	case int:
		if v < 0 {
			return 0, fmt.Errorf("negative integer value: %d", v)
		}
		return uint64(v), nil
	case float64:
		if v < 0 {
			return 0, fmt.Errorf("negative numeric value: %v", v)
		}
		if math.Trunc(v) != v {
			return 0, fmt.Errorf("non-integer numeric value: %v", v)
		}
		if v > float64(math.MaxUint64) {
			return 0, fmt.Errorf("numeric value out of uint64 range: %v", v)
		}
		return uint64(v), nil
	case stdjson.Number:
		if i, err := v.Int64(); err == nil {
			if i < 0 {
				return 0, fmt.Errorf("negative numeric value: %d", i)
			}
			return uint64(i), nil
		}
		return parseUint64(v.String())
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return 0, fmt.Errorf("empty numeric string")
		}

		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			num, err := strconv.ParseUint(s[2:], 16, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid hex number %q: %w", s, err)
			}
			return num, nil
		}

		num, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid decimal number %q: %w", s, err)
		}
		return num, nil
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}
