package deployer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	fsjson "github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem/json"
)

func TestPrepareStateForAltDASkipReusesExistingDeployment(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, stateFile)

	err := fsjson.NewWriter().WriteJSON(statePath, OPDeploymentState{
		OpChainDeployments: []OpChainDeployment{
			{
				ID:                  "0x1",
				AltDAChallengeProxy: "0xproxy",
				AltDAChallengeImpl:  "0ximpl",
			},
			{
				ID: "0x2",
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	cfg := configs.L2{
		AltDA: configs.AltDAConfig{
			Enabled:          true,
			SkipL1Deploy:     true,
			DACommitmentType: configs.AltDACommitmentTypeKeccak,
		},
	}

	if err := PrepareStateForAltDASkip(stateDir, cfg); err != nil {
		t.Fatalf("PrepareStateForAltDASkip() error = %v", err)
	}

	var state OPDeploymentState
	if err := fsjson.NewReader().ReadJSON(statePath, &state); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	if got := state.OpChainDeployments[1].AltDAChallengeProxy; got != "0xproxy" {
		t.Fatalf("expected reused proxy %q, got %q", "0xproxy", got)
	}
	if got := state.OpChainDeployments[1].AltDAChallengeImpl; got != "0ximpl" {
		t.Fatalf("expected reused impl %q, got %q", "0ximpl", got)
	}
}

func TestPrepareStateForAltDASkipErrorsWithoutExistingDeployment(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, stateFile)

	err := fsjson.NewWriter().WriteJSON(statePath, OPDeploymentState{
		OpChainDeployments: []OpChainDeployment{
			{ID: "0x1"},
			{ID: "0x2"},
		},
	})
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	cfg := configs.L2{
		AltDA: configs.AltDAConfig{
			Enabled:          true,
			SkipL1Deploy:     true,
			DACommitmentType: configs.AltDACommitmentTypeKeccak,
		},
	}

	err = PrepareStateForAltDASkip(stateDir, cfg)
	if err == nil {
		t.Fatal("expected error when no alt-da deployment exists")
	}
}

func TestPrepareStateForAltDASkipUsesConfiguredAddresses(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, stateFile)

	err := fsjson.NewWriter().WriteJSON(statePath, OPDeploymentState{
		OpChainDeployments: []OpChainDeployment{
			{ID: "0x1"},
			{ID: "0x2"},
		},
	})
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	cfg := configs.L2{
		AltDA: configs.AltDAConfig{
			Enabled:               true,
			SkipL1Deploy:          true,
			DACommitmentType:      configs.AltDACommitmentTypeKeccak,
			ChallengeProxyAddress: "0x1111111111111111111111111111111111111111",
			ChallengeImplAddress:  "0x2222222222222222222222222222222222222222",
		},
	}

	if err := PrepareStateForAltDASkip(stateDir, cfg); err != nil {
		t.Fatalf("PrepareStateForAltDASkip() error = %v", err)
	}

	var state OPDeploymentState
	if err := fsjson.NewReader().ReadJSON(statePath, &state); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	for _, deployment := range state.OpChainDeployments {
		if deployment.AltDAChallengeProxy != cfg.AltDA.ChallengeProxyAddress {
			t.Fatalf("expected configured proxy %q, got %q", cfg.AltDA.ChallengeProxyAddress, deployment.AltDAChallengeProxy)
		}
		if deployment.AltDAChallengeImpl != cfg.AltDA.ChallengeImplAddress {
			t.Fatalf("expected configured impl %q, got %q", cfg.AltDA.ChallengeImplAddress, deployment.AltDAChallengeImpl)
		}
	}
}

func TestPrepareStateForAltDASkipTreatsZeroAddressesAsUnset(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, stateFile)

	err := fsjson.NewWriter().WriteJSON(statePath, OPDeploymentState{
		OpChainDeployments: []OpChainDeployment{
			{
				ID:                  "0x1",
				AltDAChallengeProxy: "0x0000000000000000000000000000000000000000",
				AltDAChallengeImpl:  "0x0000000000000000000000000000000000000000",
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	cfg := configs.L2{
		AltDA: configs.AltDAConfig{
			Enabled:               true,
			SkipL1Deploy:          true,
			DACommitmentType:      configs.AltDACommitmentTypeKeccak,
			ChallengeProxyAddress: "0x1111111111111111111111111111111111111111",
			ChallengeImplAddress:  "0x2222222222222222222222222222222222222222",
		},
	}

	if err := PrepareStateForAltDASkip(stateDir, cfg); err != nil {
		t.Fatalf("PrepareStateForAltDASkip() error = %v", err)
	}

	var state OPDeploymentState
	if err := fsjson.NewReader().ReadJSON(statePath, &state); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	got := state.OpChainDeployments[0]
	if got.AltDAChallengeProxy != cfg.AltDA.ChallengeProxyAddress {
		t.Fatalf("expected configured proxy %q, got %q", cfg.AltDA.ChallengeProxyAddress, got.AltDAChallengeProxy)
	}
	if got.AltDAChallengeImpl != cfg.AltDA.ChallengeImplAddress {
		t.Fatalf("expected configured impl %q, got %q", cfg.AltDA.ChallengeImplAddress, got.AltDAChallengeImpl)
	}
}

func TestFinalizeStateForAltDASkipUpdatesAppliedIntent(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, stateFile)

	appliedIntentRaw, err := json.Marshal(map[string]any{
		"chains": []map[string]any{
			{
				"id": "0x1",
			},
		},
	})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	err = fsjson.NewWriter().WriteJSON(statePath, OPDeploymentState{
		AppliedIntent: appliedIntentRaw,
		OpChainDeployments: []OpChainDeployment{
			{
				ID: "0x1",
			},
		},
	})
	if err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	cfg := configs.L2{
		AltDA: configs.AltDAConfig{
			Enabled:                    true,
			SkipL1Deploy:               true,
			DACommitmentType:           configs.AltDACommitmentTypeKeccak,
			DAChallengeWindow:          1,
			DAResolveWindow:            2,
			DABondSize:                 3,
			DAResolverRefundPercentage: 4,
			ChallengeProxyAddress:      "0x1111111111111111111111111111111111111111",
			ChallengeImplAddress:       "0x2222222222222222222222222222222222222222",
		},
	}

	if err := FinalizeStateForAltDASkip(stateDir, cfg); err != nil {
		t.Fatalf("FinalizeStateForAltDASkip() error = %v", err)
	}

	var state OPDeploymentState
	if err := fsjson.NewReader().ReadJSON(statePath, &state); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	if got := state.OpChainDeployments[0].AltDAChallengeProxy; got != cfg.AltDA.ChallengeProxyAddress {
		t.Fatalf("expected finalized proxy %q, got %q", cfg.AltDA.ChallengeProxyAddress, got)
	}
	if got := state.OpChainDeployments[0].AltDAChallengeImpl; got != cfg.AltDA.ChallengeImplAddress {
		t.Fatalf("expected finalized impl %q, got %q", cfg.AltDA.ChallengeImplAddress, got)
	}

	var appliedIntent struct {
		Chains []struct {
			DangerousAltDAConfig struct {
				UseAltDA                   bool   `json:"useAltDA"`
				DACommitmentType           string `json:"daCommitmentType"`
				DAChallengeWindow          uint64 `json:"daChallengeWindow"`
				DAResolveWindow            uint64 `json:"daResolveWindow"`
				DABondSize                 uint64 `json:"daBondSize"`
				DAResolverRefundPercentage uint64 `json:"daResolverRefundPercentage"`
			} `json:"dangerousAltDAConfig"`
		} `json:"chains"`
	}
	if err := json.Unmarshal(state.AppliedIntent, &appliedIntent); err != nil {
		t.Fatalf("Unmarshal(AppliedIntent) error = %v", err)
	}
	if len(appliedIntent.Chains) != 1 || !appliedIntent.Chains[0].DangerousAltDAConfig.UseAltDA {
		t.Fatalf("expected applied intent to include alt-da config, got %+v", appliedIntent)
	}
	if got := appliedIntent.Chains[0].DangerousAltDAConfig.DACommitmentType; got != cfg.AltDA.DACommitmentType {
		t.Fatalf("expected commitment type %q, got %q", cfg.AltDA.DACommitmentType, got)
	}
}

func TestFinalizeStateForAltDASkipPreservesOpDeployerFields(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	statePath := filepath.Join(stateDir, stateFile)

	rawState := []byte(`{
  "version": 1,
  "appliedIntent": {
    "chains": [
      {
        "id": "0x1"
      }
    ]
  },
  "superchainContracts": {
    "SuperchainConfigProxy": "0xabc"
  },
  "opChainDeployments": [
    {
      "id": "0x1",
      "allocs": {
        "data": {
          "accounts": {
            "0x1111111111111111111111111111111111111111": {}
          }
        }
      },
      "startBlock": {
        "hash": "0xabc",
        "number": "0x1"
      }
    }
  ]
}`)
	if err := os.WriteFile(statePath, rawState, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := configs.L2{
		AltDA: configs.AltDAConfig{
			Enabled:               true,
			SkipL1Deploy:          true,
			DACommitmentType:      configs.AltDACommitmentTypeKeccak,
			ChallengeProxyAddress: "0x1111111111111111111111111111111111111111",
			ChallengeImplAddress:  "0x2222222222222222222222222222222222222222",
		},
	}

	if err := FinalizeStateForAltDASkip(stateDir, cfg); err != nil {
		t.Fatalf("FinalizeStateForAltDASkip() error = %v", err)
	}

	var rawDoc map[string]json.RawMessage
	if err := fsjson.NewReader().ReadJSON(statePath, &rawDoc); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	var version int
	if err := json.Unmarshal(rawDoc["version"], &version); err != nil {
		t.Fatalf("Unmarshal(version) error = %v", err)
	}
	if version != 1 {
		t.Fatalf("expected version 1, got %d", version)
	}

	var chains []map[string]json.RawMessage
	if err := json.Unmarshal(rawDoc["opChainDeployments"], &chains); err != nil {
		t.Fatalf("Unmarshal(opChainDeployments) error = %v", err)
	}
	if len(chains) != 1 {
		t.Fatalf("expected 1 chain deployment, got %d", len(chains))
	}
	if _, ok := chains[0]["allocs"]; !ok {
		t.Fatal("expected allocs field to be preserved")
	}

	var proxy string
	if err := json.Unmarshal(chains[0]["AltDAChallengeProxy"], &proxy); err != nil {
		t.Fatalf("Unmarshal(AltDAChallengeProxy) error = %v", err)
	}
	if proxy != cfg.AltDA.ChallengeProxyAddress {
		t.Fatalf("expected proxy %q, got %q", cfg.AltDA.ChallengeProxyAddress, proxy)
	}
}
