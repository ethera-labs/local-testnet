package deployer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	jsonfs "github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem/json"
)

func TestPatchAltDAStateUpdatesContractsAndAppliedIntent(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, stateFile)
	stateJSON := `{
  "appliedIntent": {
    "chains": [
      {
        "id": "0x1",
        "dangerousAltDAConfig": {
          "useAltDA": false,
          "daCommitmentType": "",
          "daChallengeWindow": 0,
          "daResolveWindow": 0,
          "daBondSize": 0,
          "daResolverRefundPercentage": 0
        }
      }
    ]
  },
  "opChainDeployments": [
    {
      "id": "0x1",
      "AltDAChallengeProxy": "0x0000000000000000000000000000000000000000",
      "AltDAChallengeImpl": "0x0000000000000000000000000000000000000000"
    }
  ]
}
`
	if err := (&jsonfs.Writer{}).WriteBytes(statePath, []byte(stateJSON)); err != nil {
		t.Fatalf("WriteBytes() error = %v", err)
	}

	manager := NewStateManager(stateDir, jsonfs.NewReader())
	err := manager.PatchAltDAState(configs.AltDAConfig{
		Enabled:                    true,
		DACommitmentType:           configs.AltDACommitmentTypeKeccak,
		ChallengeProxyAddress:      "0x3616ff26f428ca6e4b994b8bce275977a2dea4d5",
		ChallengeImplAddress:       "0xf0bd4fdf7b568de8354bfe9a52c60789749d365b",
		DAChallengeWindow:          100,
		DAResolveWindow:            100,
		DABondSize:                 1,
		DAResolverRefundPercentage: 0,
	})
	if err != nil {
		t.Fatalf("PatchAltDAState() error = %v", err)
	}

	stateBytes, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(stateBytes)

	for _, want := range []string{
		`"AltDAChallengeProxy": "0x3616ff26f428ca6e4b994b8bce275977a2dea4d5"`,
		`"AltDAChallengeImpl": "0xf0bd4fdf7b568de8354bfe9a52c60789749d365b"`,
		`"useAltDA": true`,
		`"daCommitmentType": "KeccakCommitment"`,
		`"daChallengeWindow": 100`,
		`"daResolveWindow": 100`,
		`"daBondSize": 1`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("patched state missing %q\n%s", want, got)
		}
	}
}
