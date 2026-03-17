package configs

import (
	"strings"
	"testing"
)

func TestL2ValidateRejectsEqualCoordinatorAndWalletKeys(t *testing.T) {
	cfg := validL2Config()
	cfg.CoordinatorPrivateKey = "0xABC123"
	cfg.Wallet.PrivateKey = "abc123"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for equal coordinator and wallet private keys")
	}

	if got := err.Error(); got == "" || !strings.Contains(got, "l2.coordinator-private-key must differ from l2.wallet.private-key to avoid nonce collisions") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestL2ValidateAllowsDifferentCoordinatorAndWalletKeys(t *testing.T) {
	cfg := validL2Config()
	cfg.CoordinatorPrivateKey = "0xabc123"
	cfg.Wallet.PrivateKey = "def456"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validation to succeed, got: %v", err)
	}
}

func validL2Config() L2 {
	return L2{
		L1ChainID:         1,
		L1ElURL:           "http://localhost:8545",
		L1ClURL:           "http://localhost:5052",
		EtheraNetworkName: "compose",
		Wallet: Wallet{
			PrivateKey: "wallet-key",
			Address:    "0x123",
		},
		CoordinatorPrivateKey: "coordinator-key",
		Repositories: map[RepositoryName]Repository{
			RepositoryNameOpGeth: {
				LocalPath: "../op-geth",
			},
			RepositoryNamePublisher: {
				LocalPath: "../publisher",
			},
		},
		ChainConfigs: map[L2ChainName]Chain{
			L2ChainNameRollupA: {
				ID:      11111,
				RPCPort: 18545,
			},
			L2ChainNameRollupB: {
				ID:      22222,
				RPCPort: 28545,
			},
		},
		Images: map[ImageName]Image{
			ImageNameOpDeployer: {
				Tag: "v1",
			},
			ImageNameOpNode: {
				Tag: "v1",
			},
			ImageNameOpProposer: {
				Tag: "v1",
			},
			ImageNameOpBatcher: {
				Tag: "v1",
			},
		},
		DeploymentTarget:  "live",
		GenesisBalanceWei: "1000000000000000000",
		Dispute: DisputeConfig{
			NetworkName:                     "hoodi",
			ExplorerURL:                     "https://explorer.example",
			ExplorerAPIURL:                  "https://explorer.example/api",
			VerifierAddress:                 "0x456",
			OwnerAddress:                    "0x789",
			ProposerAddress:                 "0xabc",
			AggregationVkey:                 "0xdef",
			GuardianAddress:                 "0x987",
			ProofMaturityDelaySeconds:       1,
			DisputeGameFinalityDelaySeconds: 1,
			DisputeGameInitBond:             "1",
		},
	}
}
