package docker

import (
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethereum/go-ethereum/common"
)

func TestBuildComposeEnvIncludesFeatureRepositoryPathsWhenEnabled(t *testing.T) {
	t.Setenv("HOST_PROJECT_PATH", "")

	rootDir := t.TempDir()
	networksDir := filepath.Join(rootDir, ".localnet", "networks")
	servicesDir := filepath.Join(rootDir, ".localnet", "services")

	builder := NewEnvBuilder(rootDir, networksDir, servicesDir)
	cfg := validL2ConfigForEnv()
	cfg.Flashblocks.Enabled = true
	cfg.Sidecar.Enabled = true
	cfg.Repositories[configs.RepositoryNameOpRbuilder] = configs.Repository{LocalPath: "../op-rbuilder"}
	cfg.Repositories[configs.RepositoryNameSidecar] = configs.Repository{LocalPath: "../sidecar"}

	env, err := builder.BuildComposeEnv(cfg, common.HexToAddress("0x1234"))
	if err != nil {
		t.Fatalf("expected env build to succeed, got: %v", err)
	}

	if got := env["OP_RBUILDER_PATH"]; got != filepath.Join(rootDir, "..", "op-rbuilder") {
		t.Fatalf("unexpected op-rbuilder path: %q", got)
	}
	if got := env["SIDECAR_PATH"]; got != filepath.Join(rootDir, "..", "sidecar") {
		t.Fatalf("unexpected sidecar path: %q", got)
	}
}

func TestBuildComposeEnvSkipsFeatureRepositoryPathsWhenDisabled(t *testing.T) {
	t.Setenv("HOST_PROJECT_PATH", "")

	rootDir := t.TempDir()
	networksDir := filepath.Join(rootDir, ".localnet", "networks")
	servicesDir := filepath.Join(rootDir, ".localnet", "services")

	builder := NewEnvBuilder(rootDir, networksDir, servicesDir)
	cfg := validL2ConfigForEnv()

	env, err := builder.BuildComposeEnv(cfg, common.Address{})
	if err != nil {
		t.Fatalf("expected env build to succeed, got: %v", err)
	}

	if _, ok := env["OP_RBUILDER_PATH"]; ok {
		t.Fatal("expected op-rbuilder path to be omitted when flashblocks is disabled")
	}
	if _, ok := env["SIDECAR_PATH"]; ok {
		t.Fatal("expected sidecar path to be omitted when sidecar is disabled")
	}
}

func validL2ConfigForEnv() configs.L2 {
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
			configs.RepositoryNameOpGeth: {
				LocalPath: "../op-geth",
			},
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
		Images: map[configs.ImageName]configs.Image{
			configs.ImageNameOpBatcher: {
				Tag: "v1",
			},
			configs.ImageNameOpNode: {
				Tag: "v1",
			},
			configs.ImageNameOpProposer: {
				Tag: "v1",
			},
		},
		Flashblocks: configs.FlashblocksConfig{
			RollupARPCPort: 17545,
			RollupBRPCPort: 27545,
		},
		Sidecar: configs.SidecarConfig{
			RollupAAPIPort: 17090,
			RollupBAPIPort: 27090,
		},
		Frontend: configs.FrontendConfig{
			Port: 3000,
		},
	}
}
