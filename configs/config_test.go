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

func TestAltDAProviderDefaultsToLocalOpAltDA(t *testing.T) {
	t.Parallel()

	var cfg AltDAConfig
	if got := cfg.ProviderName(); got != AltDAProviderOpAltDALocal {
		t.Fatalf("expected default provider %q, got %q", AltDAProviderOpAltDALocal, got)
	}
}

func TestAltDASkipL1DeployDefaultsFalse(t *testing.T) {
	t.Parallel()

	var cfg AltDAConfig
	if cfg.SkipL1Deploy {
		t.Fatal("expected skip-l1-deploy to default to false")
	}
}

func TestL2ValidateRejectsIncompleteAltDAConfiguredAddresses(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.AltDA.Enabled = true
	cfg.AltDA.Provider = AltDAProviderOpAltDALocal
	cfg.AltDA.DAServer = "http://localhost:3100"
	cfg.AltDA.DACommitmentType = AltDACommitmentTypeKeccak
	cfg.AltDA.DAChallengeWindow = 1
	cfg.AltDA.DAResolveWindow = 1
	cfg.AltDA.MaxConcurrentDARequests = 1
	cfg.AltDA.ChallengeProxyAddress = "0x1111111111111111111111111111111111111111"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for incomplete alt-da configured addresses")
	}

	if got := err.Error(); !strings.Contains(got, "l2.alt-da.challenge-proxy-address and l2.alt-da.challenge-impl-address must be set together") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestL2ValidateRejectsInvalidAltDAConfiguredAddresses(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.AltDA.Enabled = true
	cfg.AltDA.Provider = AltDAProviderOpAltDALocal
	cfg.AltDA.DAServer = "http://localhost:3100"
	cfg.AltDA.DACommitmentType = AltDACommitmentTypeKeccak
	cfg.AltDA.DAChallengeWindow = 1
	cfg.AltDA.DAResolveWindow = 1
	cfg.AltDA.MaxConcurrentDARequests = 1
	cfg.AltDA.ChallengeProxyAddress = "proxy"
	cfg.AltDA.ChallengeImplAddress = "0x2222222222222222222222222222222222222222"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid alt-da proxy address")
	}

	if got := err.Error(); !strings.Contains(got, "l2.alt-da.challenge-proxy-address must be a valid hex address") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestL2ValidateRejectsRemovedCelestiaProvider(t *testing.T) {
	t.Parallel()

	cfg := validL2Config()
	cfg.AltDA.Enabled = true
	cfg.AltDA.Provider = "celestia"
	cfg.AltDA.DAServer = "http://localhost:3100"
	cfg.AltDA.DACommitmentType = AltDACommitmentTypeKeccak
	cfg.AltDA.DAChallengeWindow = 1
	cfg.AltDA.DAResolveWindow = 1
	cfg.AltDA.MaxConcurrentDARequests = 1

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for removed Celestia provider")
	}

	if got := err.Error(); !strings.Contains(got, `l2.alt-da.provider must be "op-alt-da-local" when l2.alt-da.enabled is true`) {
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
