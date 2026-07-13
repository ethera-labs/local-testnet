package docker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethereum/go-ethereum/common"
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

func TestBuildComposeEnvOmitsCrossScoutWhenDisabled(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cfg := composeEnvTestConfig()
	builder := NewEnvBuilder(rootDir, filepath.Join(rootDir, "networks"), filepath.Join(rootDir, ".localnet", "services"))

	gameFactory := common.HexToAddress("0x1111111111111111111111111111111111111111")
	anchorStateRegistry := common.HexToAddress("0x2222222222222222222222222222222222222222")
	env, err := builder.BuildComposeEnv(cfg, gameFactory, anchorStateRegistry)
	if err != nil {
		t.Fatalf("BuildComposeEnv() error = %v", err)
	}

	if _, ok := env["CROSS_SCOUT_URL"]; ok {
		t.Fatal("expected CROSS_SCOUT_URL to be omitted when cross-scout is disabled")
	}
	if got := env["SP_L1_DISPUTE_GAME_FACTORY"]; got != gameFactory.Hex() {
		t.Fatalf("SP_L1_DISPUTE_GAME_FACTORY = %q, want %q", got, gameFactory.Hex())
	}
	if got := env["SP_L1_ANCHOR_STATE_REGISTRY"]; got != anchorStateRegistry.Hex() {
		t.Fatalf("SP_L1_ANCHOR_STATE_REGISTRY = %q, want %q", got, anchorStateRegistry.Hex())
	}
	if _, ok := env["SP_L1_SUPERBLOCK_CONTRACT"]; ok {
		t.Fatal("expected SP_L1_SUPERBLOCK_CONTRACT to be omitted")
	}
}

func TestBuildComposeEnvAddsCrossScoutWhenEnabled(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	cfg := composeEnvTestConfig()
	cfg.CrossScout = configs.CrossScoutConfig{
		Enabled:      true,
		APIPort:      3001,
		ExplorerPort: 3002,
		PostgresPort: 15432,
	}
	cfg.Repositories[configs.RepositoryNameCrossScout] = configs.Repository{LocalPath: "../cross-scout"}

	builder := NewEnvBuilder(rootDir, filepath.Join(rootDir, "networks"), filepath.Join(rootDir, ".localnet", "services"))
	anchorStateRegistry := common.HexToAddress("0x2222222222222222222222222222222222222222")
	env, err := builder.BuildComposeEnv(cfg, common.Address{}, anchorStateRegistry)
	if err != nil {
		t.Fatalf("BuildComposeEnv() error = %v", err)
	}

	if got, want := env["CROSS_SCOUT_URL"], "http://localhost:3002"; got != want {
		t.Fatalf("CROSS_SCOUT_URL = %q, want %q", got, want)
	}
	if got, want := env["CROSS_SCOUT_EL_RPC_URLS"], "11111=http://validator-el-a:8545,22222=http://validator-el-b:8545"; got != want {
		t.Fatalf("CROSS_SCOUT_EL_RPC_URLS = %q, want %q", got, want)
	}
	if got, want := env["CROSS_SCOUT_CHAIN_NAMES"], "11111=Rollup A,22222=Rollup B"; got != want {
		t.Fatalf("CROSS_SCOUT_CHAIN_NAMES = %q, want %q", got, want)
	}
	if got := env["CROSS_SCOUT_ANCHOR_STATE_REGISTRY_ADDRESS"]; got != anchorStateRegistry.Hex() {
		t.Fatalf("CROSS_SCOUT_ANCHOR_STATE_REGISTRY_ADDRESS = %q, want %q", got, anchorStateRegistry.Hex())
	}
}

func TestMergePostDeployEnvCrossScoutAddresses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		contractsA  string
		contractsB  string
		wantMailbox string
		wantBridges string
	}{
		{
			name:        "distinct bridges from both chains are joined",
			contractsA:  `{"addresses":{"UniversalBridgeMailbox":"0x1111111111111111111111111111111111111111","ComposeL2ToL2Bridge":"0x3333333333333333333333333333333333333333"}}`,
			contractsB:  `{"addresses":{"UniversalBridgeMailbox":"0x1111111111111111111111111111111111111111","ComposeL2ToL2Bridge":"0x4444444444444444444444444444444444444444"}}`,
			wantMailbox: "0x1111111111111111111111111111111111111111",
			wantBridges: "0x3333333333333333333333333333333333333333,0x4444444444444444444444444444444444444444",
		},
		{
			name:        "identical bridges are deduplicated",
			contractsA:  `{"addresses":{"UniversalBridgeMailbox":"0x1111111111111111111111111111111111111111","ComposeL2ToL2Bridge":"0x3333333333333333333333333333333333333333"}}`,
			contractsB:  `{"addresses":{"UniversalBridgeMailbox":"0x1111111111111111111111111111111111111111","ComposeL2ToL2Bridge":"0x3333333333333333333333333333333333333333"}}`,
			wantMailbox: "0x1111111111111111111111111111111111111111",
			wantBridges: "0x3333333333333333333333333333333333333333",
		},
		{
			name:        "rollup B fills in when rollup A contracts are missing",
			contractsB:  `{"addresses":{"UniversalBridgeMailbox":"0x2222222222222222222222222222222222222222","ComposeL2ToL2Bridge":"0x4444444444444444444444444444444444444444"}}`,
			wantMailbox: "0x2222222222222222222222222222222222222222",
			wantBridges: "0x4444444444444444444444444444444444444444",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			networksDir := t.TempDir()
			writeChainContracts := func(chain configs.L2ChainName, content string) {
				if content == "" {
					return
				}
				chainDir := filepath.Join(networksDir, string(chain))
				if err := os.MkdirAll(chainDir, 0755); err != nil {
					t.Fatalf("failed to create chain dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(chainDir, "contracts.json"), []byte(content), 0644); err != nil {
					t.Fatalf("failed to write contracts.json: %v", err)
				}
			}
			writeChainContracts(configs.L2ChainNameRollupA, tc.contractsA)
			writeChainContracts(configs.L2ChainNameRollupB, tc.contractsB)

			env := map[string]string{}
			NewEnvBuilder("", networksDir, "").MergePostDeployEnv(env)

			if got := env["CROSS_SCOUT_MAILBOX_ADDRESS"]; got != tc.wantMailbox {
				t.Fatalf("CROSS_SCOUT_MAILBOX_ADDRESS = %q, want %q", got, tc.wantMailbox)
			}
			if got := env["CROSS_SCOUT_BRIDGE_ADDRESSES"]; got != tc.wantBridges {
				t.Fatalf("CROSS_SCOUT_BRIDGE_ADDRESSES = %q, want %q", got, tc.wantBridges)
			}
		})
	}
}

func composeEnvTestConfig() configs.L2 {
	return configs.L2{
		L1ChainID:         1,
		L1ElURL:           "http://localhost:8545",
		L1ClURL:           "http://localhost:5052",
		EtheraNetworkName: "compose",
		Wallet: configs.Wallet{
			PrivateKey: "wallet-key",
			Address:    "0x123",
		},
		CoordinatorPrivateKey: "coordinator-key",
		Repositories: map[configs.RepositoryName]configs.Repository{
			configs.RepositoryNamePublisher: {
				LocalPath: "../publisher",
			},
		},
		ChainConfigs: map[configs.L2ChainName]configs.Chain{
			configs.L2ChainNameRollupA: {
				ID:      11111,
				RPCPort: 18545,
			},
			configs.L2ChainNameRollupB: {
				ID:      22222,
				RPCPort: 28545,
			},
		},
		Flashblocks: configs.FlashblocksConfig{
			RollupAP2PSecretKeyHex: "0101010101010101010101010101010101010101010101010101010101010101",
			RollupBP2PSecretKeyHex: "0202020202020202020202020202020202020202020202020202020202020202",
		},
		Images: map[configs.ImageName]configs.Image{
			configs.ImageNameOpNode: {
				Tag: "v1",
			},
			configs.ImageNameOpProposer: {
				Tag: "v1",
			},
			configs.ImageNameOpBatcher: {
				Tag: "v1",
			},
			configs.ImageNameOpReth: {
				Tag: "v1",
			},
		},
	}
}
