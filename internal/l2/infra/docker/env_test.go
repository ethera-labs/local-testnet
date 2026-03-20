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
		})
	}
}
