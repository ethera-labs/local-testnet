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
	if !strings.Contains(content, "DGF_ADDRESS=0x2222222222222222222222222222222222222222") {
		t.Fatalf("missing dispute game factory address in file: %s", content)
	}
	if strings.Contains(content, "L2OO_ADDRESS=") {
		t.Fatalf("unexpected L2 output oracle address in file: %s", content)
	}
}

func TestSetMailboxAddressUpdatesOpsuccinctEnv(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	generator := NewGenerator()

	if err := generator.Generate(
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		dir,
	); err != nil {
		t.Fatalf("expected generate to succeed, got: %v", err)
	}

	if err := generator.SetMailboxAddress(common.HexToAddress("0x3333333333333333333333333333333333333333"), dir); err != nil {
		t.Fatalf("expected mailbox update to succeed, got: %v", err)
	}

	if err := generator.SetMailboxAddress(common.HexToAddress("0x4444444444444444444444444444444444444444"), dir); err != nil {
		t.Fatalf("expected mailbox update to replace previous value, got: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "opsuccinct.env"))
	if err != nil {
		t.Fatalf("failed to read generated file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "MAILBOX_ADDRESS=0x4444444444444444444444444444444444444444") {
		t.Fatalf("missing mailbox address in file: %s", content)
	}
	if strings.Count(content, "MAILBOX_ADDRESS=") != 1 {
		t.Fatalf("expected exactly one mailbox address entry, got: %s", content)
	}
}
