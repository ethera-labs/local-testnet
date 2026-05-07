package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
)

func TestReadUniversalBridgeMailboxAddress(t *testing.T) {
	t.Parallel()

	networksDir := t.TempDir()
	chainDir := filepath.Join(networksDir, string(configs.L2ChainNameRollupA))
	if err := os.MkdirAll(chainDir, 0755); err != nil {
		t.Fatalf("failed to create chain dir: %v", err)
	}

	const mailbox = "0x1111111111111111111111111111111111111111"
	content := []byte(`{"addresses":{"UniversalBridgeMailbox":"` + mailbox + `"}}`)
	if err := os.WriteFile(filepath.Join(chainDir, "contracts.json"), content, 0644); err != nil {
		t.Fatalf("failed to write contracts.json: %v", err)
	}

	builder := NewEnvBuilder("", networksDir, "")
	if got := builder.readUniversalBridgeMailboxAddress(configs.L2ChainNameRollupA); got != mailbox {
		t.Fatalf("readUniversalBridgeMailboxAddress() = %q, want %q", got, mailbox)
	}
}
