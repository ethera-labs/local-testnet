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

func TestNormalizePrivateKeyStripsAtMostOnePrefix(t *testing.T) {
	t.Parallel()

	if got := normalizePrivateKey(" 0XAbC123 "); got != "abc123" {
		t.Fatalf("expected normalized key %q, got %q", "abc123", got)
	}

	if got := normalizePrivateKey("0x0X123"); got != "0x123" {
		t.Fatalf("expected malformed double-prefix key to normalize to %q, got %q", "0x123", got)
	}
}

func TestL2ValidateRequiresOpSuccinctRepositoryWhenEnabled(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.OPSuccinct.Enabled = true

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when op-succinct is enabled without repository config")
	}

	if got := err.Error(); !strings.Contains(got, "l2.repositories.op-succinct is required") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestL2ValidateAllowsOpSuccinctRepositoryWhenEnabled(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.OPSuccinct.Enabled = true
	cfg.Repositories[RepositoryNameOPSuccinct] = Repository{
		LocalPath: "../op-succinct",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validation to succeed, got: %v", err)
	}
}

func TestL2ValidateRequiresOpRbuilderRepositoryWhenFlashblocksEnabled(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.Flashblocks.Enabled = true

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing op-rbuilder repository")
	}

	if got := err.Error(); !strings.Contains(got, "l2.repositories.op-rbuilder is required") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestL2ValidateRequiresSidecarRepositoryWhenSidecarEnabled(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.Flashblocks.Enabled = true
	cfg.Sidecar.Enabled = true
	cfg.Repositories[RepositoryNameOpRbuilder] = Repository{LocalPath: "../op-rbuilder"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing sidecar repository")
	}

	if got := err.Error(); !strings.Contains(got, "l2.repositories.sidecar is required") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestL2ValidateAllowsFeatureRepositoriesWhenEnabled(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.Flashblocks.Enabled = true
	cfg.Sidecar.Enabled = true
	cfg.Repositories[RepositoryNameOpRbuilder] = Repository{LocalPath: "../op-rbuilder"}
	cfg.Repositories[RepositoryNameSidecar] = Repository{LocalPath: "../sidecar"}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validation to succeed, got: %v", err)
	}
}

func TestL2ValidateRejectsSidecarWithoutFlashblocks(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.Sidecar.Enabled = true
	cfg.Repositories[RepositoryNameSidecar] = Repository{LocalPath: "../sidecar"}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error when sidecar is enabled without flashblocks")
	}

	if got := err.Error(); !strings.Contains(got, "l2.sidecar.enabled requires l2.flashblocks.enabled") {
		t.Fatalf("unexpected validation error: %v", err)
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
			RepositoryNameEtheraContracts: {
				LocalPath: "../ethera-contracts",
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
