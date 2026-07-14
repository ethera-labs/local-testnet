package dispute

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethereum/go-ethereum/common"
)

func TestParseDeploymentContractsFromContractsConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := `{
  "l1": {
    "deployed": {
      "disputeGameFactory": "0x1111111111111111111111111111111111111111",
      "anchorStateRegistry": "0x2222222222222222222222222222222222222222"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(data), 0644); err != nil {
		t.Fatalf("failed to write deployment file: %v", err)
	}

	svc := &Service{contractsRoot: dir}

	got, err := svc.parseDeploymentContracts()
	if err != nil {
		t.Fatalf("expected parse to succeed, got: %v", err)
	}

	if got.DisputeGameFactoryAddress != common.HexToAddress("0x1111111111111111111111111111111111111111") {
		t.Fatalf("unexpected dispute game factory address: %s", got.DisputeGameFactoryAddress)
	}
	if got.AnchorStateRegistryAddress != common.HexToAddress("0x2222222222222222222222222222222222222222") {
		t.Fatalf("unexpected anchor state registry address: %s", got.AnchorStateRegistryAddress)
	}
}

func TestGenerateContractsConfigSeedsSharedInfraConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		opSuccinctEnabled bool
		wantSP1Verifier   string
	}{
		{
			name:              "op-succinct enabled seeds zero verifier for mock deploy to overwrite",
			opSuccinctEnabled: true,
			wantSP1Verifier:   zeroAddress,
		},
		{
			name:              "op-succinct disabled seeds inert placeholder verifier",
			opSuccinctEnabled: false,
			wantSP1Verifier:   inertSP1Verifier,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			svc := &Service{
				contractsRoot: dir,
				cfg: configs.L2{
					OPSuccinct: configs.OPSuccinctConfig{Enabled: tc.opSuccinctEnabled},
					Dispute: configs.DisputeConfig{
						GuardianAddress:                 "0x1111111111111111111111111111111111111111",
						OwnerAddress:                    "0x2222222222222222222222222222222222222222",
						ProposerAddress:                 "0x3333333333333333333333333333333333333333",
						AggregationVkey:                 "0x0059ae2f8c8ad61a6af02594067148b58dbecff2e3352170923efda8ea603f1e",
						ProofMaturityDelaySeconds:       604800,
						DisputeGameFinalityDelaySeconds: 302400,
						DisputeGameInitBond:             "80000000000000000",
					},
				},
			}

			if err := svc.generateContractsConfig(); err != nil {
				t.Fatalf("expected config generation to succeed, got: %v", err)
			}

			data, err := os.ReadFile(filepath.Join(dir, "config.json"))
			if err != nil {
				t.Fatalf("failed to read config.json: %v", err)
			}

			var got struct {
				L1 struct {
					Guardian              string `json:"guardian"`
					ProxyAdminOwner       string `json:"proxyAdminOwner"`
					DefaultAdmin          string `json:"defaultAdmin"`
					DepositWhitelistAdmin string `json:"depositWhitelistAdmin"`
					AuthorizedProposer    string `json:"authorizedProposer"`
					SP1Verifier           string `json:"sp1Verifier"`
				} `json:"l1"`
			}
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("failed to parse generated config: %v", err)
			}

			if got.L1.SP1Verifier != tc.wantSP1Verifier {
				t.Fatalf("unexpected verifier address: %s", got.L1.SP1Verifier)
			}
			if got.L1.DefaultAdmin != got.L1.ProxyAdminOwner {
				t.Fatalf("default admin should use owner address")
			}
			if got.L1.DepositWhitelistAdmin != got.L1.ProxyAdminOwner {
				t.Fatalf("deposit whitelist admin should use owner address")
			}
			if got.L1.AuthorizedProposer != "0x3333333333333333333333333333333333333333" {
				t.Fatalf("unexpected proposer address: %s", got.L1.AuthorizedProposer)
			}
		})
	}
}
