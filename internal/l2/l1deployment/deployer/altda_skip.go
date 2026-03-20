package deployer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethera-labs/local-testnet/configs"
	fsjson "github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem/json"
)

func AltDAConfigForApply(cfg configs.AltDAConfig) configs.AltDAConfig {
	if shouldSkipAltDAApply(cfg) {
		return configs.AltDAConfig{}
	}

	return cfg
}

func FinalizeStateForAltDASkip(stateDir string, cfg configs.L2) error {
	if !shouldSkipAltDAL1Deploy(cfg) {
		return nil
	}

	statePath := filepath.Join(stateDir, stateFile)
	state, rawDoc, chainDocsByID, err := loadAltDASkipStateDocument(statePath)
	if err != nil {
		return fmt.Errorf("failed to read %s for alt-da skip finalization: %w", statePath, err)
	}

	proxy, impl, err := configuredOrReusableAltDAContracts(cfg.AltDA, state)
	if err != nil {
		return err
	}

	updated := false
	for i := range state.OpChainDeployments {
		deployment := &state.OpChainDeployments[i]
		proxyUnset := isUnsetAltDAAddress(deployment.AltDAChallengeProxy)
		implUnset := isUnsetAltDAAddress(deployment.AltDAChallengeImpl)
		switch {
		case proxyUnset && implUnset:
			deployment.AltDAChallengeProxy = proxy
			deployment.AltDAChallengeImpl = impl
			rawChain, ok := chainDocsByID[strings.ToLower(strings.TrimSpace(deployment.ID))]
			if !ok {
				return fmt.Errorf("failed to find raw chain deployment for %s in %s", deployment.ID, statePath)
			}
			rawChain["AltDAChallengeProxy"], err = marshalRawJSON(proxy)
			if err != nil {
				return fmt.Errorf("failed to encode AltDAChallengeProxy for chain %s: %w", deployment.ID, err)
			}
			rawChain["AltDAChallengeImpl"], err = marshalRawJSON(impl)
			if err != nil {
				return fmt.Errorf("failed to encode AltDAChallengeImpl for chain %s: %w", deployment.ID, err)
			}
			updated = true
		case proxyUnset || implUnset:
			return fmt.Errorf(
				"found incomplete alt-da state for chain %s in %s: both AltDAChallengeProxy and AltDAChallengeImpl must be set together",
				deployment.ID,
				statePath,
			)
		case deployment.AltDAChallengeProxy != proxy || deployment.AltDAChallengeImpl != impl:
			return fmt.Errorf(
				"found conflicting alt-da deployment for chain %s in %s: expected proxy=%s impl=%s, got proxy=%s impl=%s",
				deployment.ID,
				statePath,
				proxy,
				impl,
				deployment.AltDAChallengeProxy,
				deployment.AltDAChallengeImpl,
			)
		}
	}

	intentUpdated, err := updateAppliedIntentAltDAConfig(&state, cfg.AltDA)
	if err != nil {
		return fmt.Errorf("failed to update applied intent alt-da config in %s: %w", statePath, err)
	}

	if !updated && !intentUpdated {
		return nil
	}

	if updated {
		chainDocsRaw, err := marshalRawJSON(rawDocOpChainDeployments(chainDocsByID, state.OpChainDeployments))
		if err != nil {
			return fmt.Errorf("failed to encode opChainDeployments for %s: %w", statePath, err)
		}
		rawDoc["opChainDeployments"] = chainDocsRaw
	}
	if intentUpdated {
		rawDoc["appliedIntent"] = append(json.RawMessage(nil), state.AppliedIntent...)
	}

	if err := fsjson.NewWriter().WriteJSON(statePath, rawDoc); err != nil {
		return fmt.Errorf("failed to write finalized alt-da skip state to %s: %w", statePath, err)
	}

	return nil
}

func PrepareStateForAltDASkip(stateDir string, cfg configs.L2) error {
	return FinalizeStateForAltDASkip(stateDir, cfg)
}

func shouldSkipAltDAL1Deploy(cfg configs.L2) bool {
	return cfg.AltDA.Enabled &&
		cfg.AltDA.SkipL1Deploy &&
		cfg.AltDA.DACommitmentType == configs.AltDACommitmentTypeKeccak
}

func shouldSkipAltDAApply(cfg configs.AltDAConfig) bool {
	return cfg.Enabled &&
		cfg.SkipL1Deploy &&
		cfg.DACommitmentType == configs.AltDACommitmentTypeKeccak
}

func configuredOrReusableAltDAContracts(cfg configs.AltDAConfig, state OPDeploymentState) (string, string, error) {
	proxy := strings.TrimSpace(cfg.ChallengeProxyAddress)
	impl := strings.TrimSpace(cfg.ChallengeImplAddress)
	if proxy != "" || impl != "" {
		if proxy == "" || impl == "" {
			return "", "", fmt.Errorf(
				"l2.alt-da.challenge-proxy-address and l2.alt-da.challenge-impl-address must be set together",
			)
		}
		return proxy, impl, nil
	}

	return reusableAltDAContracts(state)
}

func reusableAltDAContracts(state OPDeploymentState) (string, string, error) {
	var proxy string
	var impl string

	for _, deployment := range state.OpChainDeployments {
		proxyUnset := isUnsetAltDAAddress(deployment.AltDAChallengeProxy)
		implUnset := isUnsetAltDAAddress(deployment.AltDAChallengeImpl)
		if proxyUnset && implUnset {
			continue
		}
		if proxyUnset || implUnset {
			return "", "", fmt.Errorf(
				"found incomplete alt-da state for chain %s: both AltDAChallengeProxy and AltDAChallengeImpl must be set together",
				deployment.ID,
			)
		}
		if proxy == "" && impl == "" {
			proxy = deployment.AltDAChallengeProxy
			impl = deployment.AltDAChallengeImpl
			continue
		}
		if deployment.AltDAChallengeProxy != proxy || deployment.AltDAChallengeImpl != impl {
			return "", "", fmt.Errorf(
				"found multiple alt-da deployments in state.json; skip-l1-deploy requires one reusable deployment",
			)
		}
	}

	if proxy == "" || impl == "" {
		return "", "", fmt.Errorf(
			"l2.alt-da.skip-l1-deploy is true, but state.json does not contain an existing Alt-DA deployment to reuse",
		)
	}

	return proxy, impl, nil
}

func isUnsetAltDAAddress(addr string) bool {
	trimmed := strings.TrimSpace(addr)
	return trimmed == "" || strings.EqualFold(trimmed, "0x0000000000000000000000000000000000000000")
}

func updateAppliedIntentAltDAConfig(state *OPDeploymentState, cfg configs.AltDAConfig) (bool, error) {
	if len(state.AppliedIntent) == 0 || strings.EqualFold(strings.TrimSpace(string(state.AppliedIntent)), "null") {
		return false, nil
	}

	var applied map[string]json.RawMessage
	if err := json.Unmarshal(state.AppliedIntent, &applied); err != nil {
		return false, fmt.Errorf("failed to decode appliedIntent: %w", err)
	}

	chainsRaw, ok := applied["chains"]
	if !ok {
		return false, fmt.Errorf("appliedIntent.chains is missing")
	}

	var chains []map[string]json.RawMessage
	if err := json.Unmarshal(chainsRaw, &chains); err != nil {
		return false, fmt.Errorf("failed to decode appliedIntent.chains: %w", err)
	}

	altDAConfigRaw, err := json.Marshal(appliedIntentAltDAConfig{
		UseAltDA:                   true,
		DACommitmentType:           cfg.DACommitmentType,
		DAChallengeWindow:          cfg.DAChallengeWindow,
		DAResolveWindow:            cfg.DAResolveWindow,
		DABondSize:                 cfg.DABondSize,
		DAResolverRefundPercentage: cfg.DAResolverRefundPercentage,
	})
	if err != nil {
		return false, fmt.Errorf("failed to encode dangerousAltDAConfig: %w", err)
	}

	updated := false
	for _, chain := range chains {
		currentRaw, exists := chain["dangerousAltDAConfig"]
		if exists && jsonEqualRaw(currentRaw, altDAConfigRaw) {
			continue
		}
		chain["dangerousAltDAConfig"] = altDAConfigRaw
		updated = true
	}

	if !updated {
		return false, nil
	}

	updatedChainsRaw, err := json.Marshal(chains)
	if err != nil {
		return false, fmt.Errorf("failed to encode appliedIntent.chains: %w", err)
	}
	applied["chains"] = updatedChainsRaw

	updatedIntentRaw, err := json.Marshal(applied)
	if err != nil {
		return false, fmt.Errorf("failed to encode appliedIntent: %w", err)
	}
	state.AppliedIntent = updatedIntentRaw

	return true, nil
}

func jsonEqualRaw(a json.RawMessage, b []byte) bool {
	return strings.TrimSpace(string(a)) == strings.TrimSpace(string(b))
}

type appliedIntentAltDAConfig struct {
	UseAltDA                   bool   `json:"useAltDA"`
	DACommitmentType           string `json:"daCommitmentType"`
	DAChallengeWindow          uint64 `json:"daChallengeWindow"`
	DAResolveWindow            uint64 `json:"daResolveWindow"`
	DABondSize                 uint64 `json:"daBondSize"`
	DAResolverRefundPercentage uint64 `json:"daResolverRefundPercentage"`
}

func loadAltDASkipStateDocument(statePath string) (OPDeploymentState, map[string]json.RawMessage, map[string]map[string]json.RawMessage, error) {
	data, err := os.ReadFile(statePath)
	if err != nil {
		return OPDeploymentState{}, nil, nil, err
	}

	var state OPDeploymentState
	if err := json.Unmarshal(data, &state); err != nil {
		return OPDeploymentState{}, nil, nil, err
	}

	var rawDoc map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawDoc); err != nil {
		return OPDeploymentState{}, nil, nil, err
	}

	var chainDocs []map[string]json.RawMessage
	if rawChains, ok := rawDoc["opChainDeployments"]; ok && len(rawChains) > 0 {
		if err := json.Unmarshal(rawChains, &chainDocs); err != nil {
			return OPDeploymentState{}, nil, nil, err
		}
	}

	chainDocsByID := make(map[string]map[string]json.RawMessage, len(chainDocs))
	for _, chainDoc := range chainDocs {
		var id string
		if rawID, ok := chainDoc["id"]; !ok || json.Unmarshal(rawID, &id) != nil || strings.TrimSpace(id) == "" {
			return OPDeploymentState{}, nil, nil, fmt.Errorf("failed to decode opChainDeployments id in %s", statePath)
		}
		chainDocsByID[strings.ToLower(strings.TrimSpace(id))] = chainDoc
	}

	return state, rawDoc, chainDocsByID, nil
}

func rawDocOpChainDeployments(chainDocsByID map[string]map[string]json.RawMessage, deployments []OpChainDeployment) []map[string]json.RawMessage {
	chainDocs := make([]map[string]json.RawMessage, 0, len(deployments))
	for _, deployment := range deployments {
		if rawChain, ok := chainDocsByID[strings.ToLower(strings.TrimSpace(deployment.ID))]; ok {
			chainDocs = append(chainDocs, rawChain)
		}
	}
	return chainDocs
}

func marshalRawJSON(v any) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}
