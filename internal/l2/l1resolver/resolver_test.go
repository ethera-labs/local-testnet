package l1resolver

import (
	"context"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l1"
)

type stubBootstrapper struct {
	ensureCalls  int
	resolveCalls int
	endpoints    l1.Endpoints
}

func (s *stubBootstrapper) EnsureRunning(context.Context) error {
	s.ensureCalls++
	return nil
}

func (s *stubBootstrapper) ResolvePublicEndpoints(context.Context) (l1.Endpoints, error) {
	s.resolveCalls++
	return s.endpoints, nil
}

func TestResolveFallsBackToLocalKurtosisForBundledHoodiConfig(t *testing.T) {
	bootstrapper := &stubBootstrapper{
		endpoints: l1.Endpoints{
			ExecutionURL: "http://127.0.0.1:61234",
			ConsensusURL: "http://127.0.0.1:61235",
			ChainID:      l1.LocalChainID,
		},
	}
	resolver := &Resolver{
		bootstrapper: bootstrapper,
		executionProbe: func(context.Context, string) bool {
			return false
		},
		consensusProbe: func(context.Context, string) bool {
			return false
		},
	}

	resolved, err := resolver.Resolve(context.Background(), bundledHoodiConfig())
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if bootstrapper.ensureCalls != 1 {
		t.Fatalf("expected EnsureRunning to be called once, got %d", bootstrapper.ensureCalls)
	}
	if bootstrapper.resolveCalls != 1 {
		t.Fatalf("expected ResolvePublicEndpoints to be called once, got %d", bootstrapper.resolveCalls)
	}
	if resolved.L1ElURL != bootstrapper.endpoints.ExecutionURL {
		t.Fatalf("expected execution URL %q, got %q", bootstrapper.endpoints.ExecutionURL, resolved.L1ElURL)
	}
	if resolved.L1ClURL != bootstrapper.endpoints.ConsensusURL {
		t.Fatalf("expected consensus URL %q, got %q", bootstrapper.endpoints.ConsensusURL, resolved.L1ClURL)
	}
	if resolved.L1ChainID != bootstrapper.endpoints.ChainID {
		t.Fatalf("expected chain ID %d, got %d", bootstrapper.endpoints.ChainID, resolved.L1ChainID)
	}
}

func TestResolveLeavesReachableBundledHoodiConfigUntouched(t *testing.T) {
	bootstrapper := &stubBootstrapper{}
	resolver := &Resolver{
		bootstrapper: bootstrapper,
		executionProbe: func(context.Context, string) bool {
			return true
		},
		consensusProbe: func(context.Context, string) bool {
			return true
		},
	}

	cfg := bundledHoodiConfig()
	resolved, err := resolver.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if bootstrapper.ensureCalls != 0 || bootstrapper.resolveCalls != 0 {
		t.Fatalf("expected no local bootstrap calls, got ensure=%d resolve=%d", bootstrapper.ensureCalls, bootstrapper.resolveCalls)
	}
	assertSameL1Config(t, resolved, cfg)
}

func TestResolveSkipsFallbackForNonBundledConfig(t *testing.T) {
	bootstrapper := &stubBootstrapper{}
	resolver := &Resolver{
		bootstrapper: bootstrapper,
		executionProbe: func(context.Context, string) bool {
			return false
		},
		consensusProbe: func(context.Context, string) bool {
			return false
		},
	}

	cfg := bundledHoodiConfig()
	cfg.L1ElURL = "https://ethereum-hoodi-rpc.publicnode.com/example"

	resolved, err := resolver.Resolve(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}

	if bootstrapper.ensureCalls != 0 || bootstrapper.resolveCalls != 0 {
		t.Fatalf("expected no local bootstrap calls, got ensure=%d resolve=%d", bootstrapper.ensureCalls, bootstrapper.resolveCalls)
	}
	assertSameL1Config(t, resolved, cfg)
}

func TestDockerAccessibleURL(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "localhost",
			input:    "http://127.0.0.1:8545",
			expected: "http://host.docker.internal:8545",
		},
		{
			name:     "named localhost",
			input:    "http://localhost:5052",
			expected: "http://host.docker.internal:5052",
		},
		{
			name:     "remote host",
			input:    "https://rpc.example.test:8545",
			expected: "https://rpc.example.test:8545",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := DockerAccessibleURL(tc.input); actual != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func bundledHoodiConfig() configs.L2 {
	return configs.L2{
		L1ChainID: bundledHoodiChainID,
		L1ElURL:   bundledHoodiELURL,
		L1ClURL:   bundledHoodiCLURL,
	}
}

func assertSameL1Config(t *testing.T, actual, expected configs.L2) {
	t.Helper()

	if actual.L1ChainID != expected.L1ChainID {
		t.Fatalf("expected chain ID %d, got %d", expected.L1ChainID, actual.L1ChainID)
	}
	if actual.L1ElURL != expected.L1ElURL {
		t.Fatalf("expected execution URL %q, got %q", expected.L1ElURL, actual.L1ElURL)
	}
	if actual.L1ClURL != expected.L1ClURL {
		t.Fatalf("expected consensus URL %q, got %q", expected.L1ClURL, actual.L1ClURL)
	}
}
