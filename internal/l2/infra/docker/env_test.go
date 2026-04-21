package docker

import (
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethereum/go-ethereum/common"
)

func TestBuildComposeEnvSequencerPrivateKeyFallbacks(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name             string
		sequencerKey     string
		wantSequencerKey string
	}{
		{
			name:             "fallbacks to coordinator key",
			sequencerKey:     "",
			wantSequencerKey: "coordinator-key",
		},
		{
			name:             "uses explicit sequencer key",
			sequencerKey:     "sequencer-key",
			wantSequencerKey: "sequencer-key",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			rootDir := t.TempDir()
			networksDir := filepath.Join(rootDir, ".localnet", "networks")
			servicesDir := filepath.Join(rootDir, ".localnet", "services")

			cfg := configs.L2{
				L1ChainID:         1,
				L1ElURL:           "http://localhost:8545",
				L1ClURL:           "http://localhost:5052",
				EtheraNetworkName: "compose",
				Wallet: configs.Wallet{
					PrivateKey: "wallet-key",
					Address:    "0x123",
				},
				CoordinatorPrivateKey: "coordinator-key",
				SequencerPrivateKey:   tc.sequencerKey,
				Repositories: map[configs.RepositoryName]configs.Repository{
					configs.RepositoryNameOpGeth: {
						LocalPath: filepath.Join(rootDir, "op-geth"),
					},
					configs.RepositoryNamePublisher: {
						LocalPath: filepath.Join(rootDir, "publisher"),
					},
					configs.RepositoryNameEtheraContracts: {
						LocalPath: filepath.Join(rootDir, "ethera-contracts"),
					},
				},
				ChainConfigs: map[configs.L2ChainName]configs.Chain{
					configs.L2ChainNameRollupA: {ID: 11111, RPCPort: 18545},
					configs.L2ChainNameRollupB: {ID: 22222, RPCPort: 28545},
				},
			}

			builder := NewEnvBuilder(rootDir, networksDir, servicesDir)
			env, err := builder.BuildComposeEnv(cfg, common.Address{})
			if err != nil {
				t.Fatalf("BuildComposeEnv() error = %v", err)
			}

			if got := env["SEQUENCER_PRIVATE_KEY"]; got != tc.wantSequencerKey {
				t.Fatalf("SEQUENCER_PRIVATE_KEY = %q, want %q", got, tc.wantSequencerKey)
			}
			if got := env["COORDINATOR_PRIVATE_KEY"]; got != "coordinator-key" {
				t.Fatalf("COORDINATOR_PRIVATE_KEY = %q, want %q", got, "coordinator-key")
			}
			if got := env["L1_COMPOSE_NETWORK_NAME"]; got != "compose" {
				t.Fatalf("L1_COMPOSE_NETWORK_NAME = %q, want %q", got, "compose")
			}
			if got := env["SP_L1_SUPERBLOCK_CONTRACT"]; got != "" {
				t.Fatalf("SP_L1_SUPERBLOCK_CONTRACT = %q, want empty", got)
			}
		})
	}
}

func TestBuildComposeEnvSetsLegacySuperblockContractAlias(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	networksDir := filepath.Join(rootDir, ".localnet", "networks")
	servicesDir := filepath.Join(rootDir, ".localnet", "services")

	cfg := configs.L2{
		L1ChainID:         1,
		L1ElURL:           "http://localhost:8545",
		L1ClURL:           "http://localhost:5052",
		EtheraNetworkName: "compose",
		Wallet: configs.Wallet{
			PrivateKey: "wallet-key",
			Address:    "0x123",
		},
		CoordinatorPrivateKey: "coordinator-key",
		SequencerPrivateKey:   "sequencer-key",
		Repositories: map[configs.RepositoryName]configs.Repository{
			configs.RepositoryNameOpGeth: {
				LocalPath: filepath.Join(rootDir, "op-geth"),
			},
			configs.RepositoryNamePublisher: {
				LocalPath: filepath.Join(rootDir, "publisher"),
			},
			configs.RepositoryNameEtheraContracts: {
				LocalPath: filepath.Join(rootDir, "ethera-contracts"),
			},
		},
		ChainConfigs: map[configs.L2ChainName]configs.Chain{
			configs.L2ChainNameRollupA: {ID: 11111, RPCPort: 18545},
			configs.L2ChainNameRollupB: {ID: 22222, RPCPort: 28545},
		},
	}

	gameFactoryAddr := common.HexToAddress("0x00000000000000000000000000000000000000aa")

	builder := NewEnvBuilder(rootDir, networksDir, servicesDir)
	env, err := builder.BuildComposeEnv(cfg, gameFactoryAddr)
	if err != nil {
		t.Fatalf("BuildComposeEnv() error = %v", err)
	}

	if got := env["SP_L1_DISPUTE_GAME_FACTORY"]; got != gameFactoryAddr.Hex() {
		t.Fatalf("SP_L1_DISPUTE_GAME_FACTORY = %q, want %q", got, gameFactoryAddr.Hex())
	}
	if got := env["SP_L1_SUPERBLOCK_CONTRACT"]; got != gameFactoryAddr.Hex() {
		t.Fatalf("SP_L1_SUPERBLOCK_CONTRACT = %q, want %q", got, gameFactoryAddr.Hex())
	}
}

func TestBuildComposeEnvSetsSuperblockProverNetworkKey(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	networksDir := filepath.Join(rootDir, ".localnet", "networks")
	servicesDir := filepath.Join(rootDir, ".localnet", "services")

	cfg := configs.L2{
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
			configs.RepositoryNameOpGeth: {
				LocalPath: filepath.Join(rootDir, "op-geth"),
			},
			configs.RepositoryNamePublisher: {
				LocalPath: filepath.Join(rootDir, "publisher"),
			},
			configs.RepositoryNameEtheraContracts: {
				LocalPath: filepath.Join(rootDir, "ethera-contracts"),
			},
			configs.RepositoryNameSuperblockProver: {
				LocalPath: filepath.Join(rootDir, "superblock-prover"),
			},
		},
		ChainConfigs: map[configs.L2ChainName]configs.Chain{
			configs.L2ChainNameRollupA: {ID: 11111, RPCPort: 18545},
			configs.L2ChainNameRollupB: {ID: 22222, RPCPort: 28545},
		},
		SuperblockProver: configs.SuperblockProverConfig{
			Enabled:           true,
			SP1Prover:         "network",
			NetworkPrivateKey: "network-key",
			MinAuctionPeriod:  "1",
			CycleLimit:        "123",
			GasLimit:          "456",
		},
	}

	builder := NewEnvBuilder(rootDir, networksDir, servicesDir)
	env, err := builder.BuildComposeEnv(cfg, common.Address{})
	if err != nil {
		t.Fatalf("BuildComposeEnv() error = %v", err)
	}

	if got := env["SUPERBLOCK_PROVER_SP1_PROVER"]; got != "network" {
		t.Fatalf("SUPERBLOCK_PROVER_SP1_PROVER = %q, want %q", got, "network")
	}
	if got := env["NETWORK_PRIVATE_KEY"]; got != "network-key" {
		t.Fatalf("NETWORK_PRIVATE_KEY = %q, want %q", got, "network-key")
	}
	if got := env["SUPERBLOCK_PROVER_MIN_AUCTION_PERIOD"]; got != "1" {
		t.Fatalf("SUPERBLOCK_PROVER_MIN_AUCTION_PERIOD = %q, want %q", got, "1")
	}
	if got := env["SUPERBLOCK_PROVER_CYCLE_LIMIT"]; got != "123" {
		t.Fatalf("SUPERBLOCK_PROVER_CYCLE_LIMIT = %q, want %q", got, "123")
	}
	if got := env["SUPERBLOCK_PROVER_GAS_LIMIT"]; got != "456" {
		t.Fatalf("SUPERBLOCK_PROVER_GAS_LIMIT = %q, want %q", got, "456")
	}
	if got := env["PROOFS_PROVER_BASE_URL"]; got != "http://superblock-prover:5000" {
		t.Fatalf("PROOFS_PROVER_BASE_URL = %q, want %q", got, "http://superblock-prover:5000")
	}
}
