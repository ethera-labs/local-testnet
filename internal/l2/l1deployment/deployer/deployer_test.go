package deployer

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	fsjson "github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem/json"
)

func TestDecorateApplyErrorIncludesRecoveryForPartialState(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	writer := fsjson.NewWriter()
	if err := writer.WriteJSON(filepath.Join(stateDir, stateFile), OPDeploymentState{
		SuperchainContracts: SuperchainContracts{
			ProtocolVersionsProxy: "0x0000000000000000000000000000000000000001",
		},
		OpChainDeployments: []OpChainDeployment{
			{ID: "0x1"},
		},
	}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}

	deployer := &Deployer{stateDir: stateDir}
	err := deployer.decorateApplyError(errors.New("execution reverted: DeployUtils: contract already deployed"))
	if err == nil {
		t.Fatal("decorateApplyError() returned nil")
	}

	msg := err.Error()
	if !strings.Contains(msg, "partially applied L1 deployment state") {
		t.Fatalf("expected partial-state explanation, got %q", msg)
	}
	if !strings.Contains(msg, "make clean-l1 && make clean-l2") {
		t.Fatalf("expected cleanup hint, got %q", msg)
	}
}

func TestDecorateApplyErrorFallsBackToGenericAlreadyDeployedHint(t *testing.T) {
	t.Parallel()

	deployer := &Deployer{stateDir: t.TempDir()}
	err := deployer.decorateApplyError(errors.New("execution reverted: DeployUtils: contract already deployed"))
	if err == nil {
		t.Fatal("decorateApplyError() returned nil")
	}

	msg := err.Error()
	if strings.Contains(msg, "partially applied L1 deployment state") {
		t.Fatalf("did not expect partial-state explanation, got %q", msg)
	}
	if !strings.Contains(msg, "already has code") {
		t.Fatalf("expected generic already-deployed explanation, got %q", msg)
	}
}

func TestDecorateApplyErrorPassesThroughOtherFailures(t *testing.T) {
	t.Parallel()

	deployer := &Deployer{stateDir: t.TempDir()}
	err := deployer.decorateApplyError(errors.New("rpc dial failed"))
	if err == nil {
		t.Fatal("decorateApplyError() returned nil")
	}

	msg := err.Error()
	if msg != "failed to run op-deployer apply: rpc dial failed" {
		t.Fatalf("unexpected error message %q", msg)
	}
}
