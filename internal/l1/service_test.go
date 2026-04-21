package l1

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEmbeddedParamsDisablePerfectPeerDASForSmallNetworks(t *testing.T) {
	t.Parallel()

	var cfg struct {
		Network struct {
			Participants []struct {
				Count int `yaml:"count"`
			} `yaml:"participants"`
			NetworkParams struct {
				PerfectPeerDASEnabled bool `yaml:"perfect_peerdas_enabled"`
			} `yaml:"network_params"`
		} `yaml:"network"`
	}

	if err := yaml.Unmarshal(params, &cfg); err != nil {
		t.Fatalf("Unmarshal(params) error = %v", err)
	}

	totalParticipants := 0
	for _, participant := range cfg.Network.Participants {
		totalParticipants += participant.Count
	}

	if totalParticipants >= 16 {
		t.Fatalf("test assumption broken: total participants = %d, want < 16", totalParticipants)
	}

	if cfg.Network.NetworkParams.PerfectPeerDASEnabled {
		t.Fatalf("perfect_peerdas_enabled must be false when total participants = %d", totalParticipants)
	}
}
