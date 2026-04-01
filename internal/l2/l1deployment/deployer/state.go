package deployer

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

const stateFile = "state.json"

// StateManager manages the deployment state (state.json)
type StateManager struct {
	stateDir string
	reader   filesystem.Reader
	logger   *slog.Logger
}

// NewStateManager creates a new state manager
func NewStateManager(stateDir string, reader filesystem.Reader) *StateManager {
	return &StateManager{
		stateDir: stateDir,
		reader:   reader,
		logger:   logger.Named("state_manager"),
	}
}

// EnsureStateDir ensures the state directory and cache exist
func (s *StateManager) EnsureStateDir() error {
	if err := os.MkdirAll(s.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	cacheDir := filepath.Join(s.stateDir, ".cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	return nil
}

// Load reads the OP deployment state from state.json
func (s *StateManager) Load() (*OPDeploymentState, error) {
	statePath := filepath.Join(s.stateDir, stateFile)

	var state OPDeploymentState
	if err := s.reader.ReadJSON(statePath, &state); err != nil {
		return nil, fmt.Errorf("failed to read '%s': %w", stateFile, err)
	}

	return &state, nil
}

// PatchAltDAState reconciles state.json for environments that reuse predeployed
// AltDA challenge contracts. It updates both the per-chain contract addresses and
// the applied intent's dangerousAltDAConfig so later inspect/apply steps emit
// AltDA-enabled configs without trying to redeploy those contracts.
func (s *StateManager) PatchAltDAState(altDA configs.AltDAConfig) error {
	statePath := filepath.Join(s.stateDir, stateFile)

	data, err := os.ReadFile(statePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	chainsRaw, ok := raw["opChainDeployments"]
	if !ok {
		return nil
	}

	var chains []map[string]json.RawMessage
	if err := json.Unmarshal(chainsRaw, &chains); err != nil {
		return fmt.Errorf("failed to parse opChainDeployments: %w", err)
	}

	patched := false
	for _, chain := range chains {
		b, _ := json.Marshal(altDA.ChallengeProxyAddress)
		chain["AltDAChallengeProxy"] = b
		b, _ = json.Marshal(altDA.ChallengeImplAddress)
		chain["AltDAChallengeImpl"] = b
		patched = true
	}

	if patched {
		raw["opChainDeployments"], err = json.Marshal(chains)
		if err != nil {
			return fmt.Errorf("failed to marshal opChainDeployments: %w", err)
		}
	}

	if appliedIntentRaw, ok := raw["appliedIntent"]; ok && len(appliedIntentRaw) > 0 && string(appliedIntentRaw) != "null" {
		var appliedIntent map[string]json.RawMessage
		if err := json.Unmarshal(appliedIntentRaw, &appliedIntent); err != nil {
			return fmt.Errorf("failed to parse appliedIntent: %w", err)
		}

		intentChainsRaw, ok := appliedIntent["chains"]
		if ok {
			var intentChains []map[string]json.RawMessage
			if err := json.Unmarshal(intentChainsRaw, &intentChains); err != nil {
				return fmt.Errorf("failed to parse appliedIntent.chains: %w", err)
			}

			altDAConfig := map[string]any{
				"useAltDA":                   true,
				"daCommitmentType":           altDA.CommitmentType(),
				"daChallengeWindow":          altDA.ChallengeWindow(),
				"daResolveWindow":            altDA.ResolveWindow(),
				"daBondSize":                 altDA.BondSize(),
				"daResolverRefundPercentage": altDA.DAResolverRefundPercentage,
			}
			encodedAltDAConfig, err := json.Marshal(altDAConfig)
			if err != nil {
				return fmt.Errorf("failed to marshal dangerousAltDAConfig: %w", err)
			}

			for _, chain := range intentChains {
				chain["dangerousAltDAConfig"] = encodedAltDAConfig
			}

			appliedIntent["chains"], err = json.Marshal(intentChains)
			if err != nil {
				return fmt.Errorf("failed to marshal appliedIntent.chains: %w", err)
			}
		}

		raw["appliedIntent"], err = json.Marshal(appliedIntent)
		if err != nil {
			return fmt.Errorf("failed to marshal appliedIntent: %w", err)
		}
	}

	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	s.logger.Info(
		"patched AltDA state in state.json",
		"proxy", altDA.ChallengeProxyAddress,
		"impl", altDA.ChallengeImplAddress,
		"commitment_type", altDA.CommitmentType(),
	)
	return nil
}
