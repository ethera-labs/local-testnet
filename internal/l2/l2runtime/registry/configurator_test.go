package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethera-labs/local-testnet/configs"
)

func TestSetupRegistryWritesComposeToml(t *testing.T) {
	t.Parallel()

	localnetDir := t.TempDir()
	cfg := configs.L2{
		L1ChainID:         560048,
		L1ElURL:           "http://l1.example",
		EtheraNetworkName: "hoodi",
		Dispute: configs.DisputeConfig{
			ExplorerURL: "https://explorer.example",
		},
		ChainConfigs: map[configs.L2ChainName]configs.Chain{
			configs.L2ChainNameRollupA: {
				ID:      11113,
				RPCPort: 18545,
			},
		},
	}

	err := NewConfigurator().SetupRegistry(
		localnetDir,
		cfg,
		common.HexToAddress("0x00000000000000000000000000000000000000aa"),
	)
	if err != nil {
		t.Fatalf("SetupRegistry() error = %v", err)
	}

	registryDir := filepath.Join(localnetDir, "registry", "networks", "hoodi")
	composePath := filepath.Join(registryDir, "compose.toml")
	raw, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", composePath, err)
	}

	if strings.Contains(string(raw), `name = ""`) {
		t.Fatalf("compose.toml contains empty network name: %s", raw)
	}

	if _, err := os.Stat(filepath.Join(registryDir, "ethera.toml")); !os.IsNotExist(err) {
		t.Fatalf("ethera.toml should not be generated, stat err = %v", err)
	}
}
