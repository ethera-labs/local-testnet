package opsuccinct

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestGenerateWritesOpsuccinctEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	generator := NewGenerator()

	err := generator.Generate(
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		dir,
	)
	if err != nil {
		t.Fatalf("expected generate to succeed, got: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "opsuccinct.env"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "COMPOSE_L2_OUTPUT_ORACLE_ADDRESS=0x1111111111111111111111111111111111111111") {
		t.Fatalf("missing compose oracle address in file: %s", content)
	}
	if !strings.Contains(content, "DGF_ADDRESS=0x2222222222222222222222222222222222222222") {
		t.Fatalf("missing dispute game factory address in file: %s", content)
	}
}
