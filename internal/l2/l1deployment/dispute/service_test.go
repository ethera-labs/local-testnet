package dispute

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethereum/go-ethereum/common"
)

func TestParseDeploymentContractsFromDeploymentsJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := `{
  "hoodi": {
    "ComposeL2OutputOracle": {
      "proxy": "0x1111111111111111111111111111111111111111"
    },
    "DisputeGameFactory": {
      "proxy": "0x2222222222222222222222222222222222222222"
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "deployments.json"), []byte(data), 0644); err != nil {
		t.Fatalf("failed to write deployments.json: %v", err)
	}

	svc := &Service{
		contractsDir: dir,
		cfg: configs.L2{
			Dispute: configs.DisputeConfig{
				NetworkName: "hoodi",
			},
		},
	}

	got, err := svc.parseDeploymentContracts()
	if err != nil {
		t.Fatalf("expected parse to succeed, got: %v", err)
	}

	if got.ComposeL2OutputOracleAddress != common.HexToAddress("0x1111111111111111111111111111111111111111") {
		t.Fatalf("unexpected L2 output oracle address: %s", got.ComposeL2OutputOracleAddress)
	}
	if got.DisputeGameFactoryAddress != common.HexToAddress("0x2222222222222222222222222222222222222222") {
		t.Fatalf("unexpected dispute game factory address: %s", got.DisputeGameFactoryAddress)
	}
}

func TestParseDeploymentContractsFromComposeFallback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	composeDir := filepath.Join(dir, "deployments", "compose")
	if err := os.MkdirAll(composeDir, 0755); err != nil {
		t.Fatalf("failed to create fallback directory: %v", err)
	}

	data := `{
  "hoodi": {
    "contracts": {
      "ComposeL2OutputOracle": {
        "proxyAddress": "0x3333333333333333333333333333333333333333"
      },
      "DisputeGameFactory": {
        "proxyAddress": "0x4444444444444444444444444444444444444444"
      }
    }
  }
}`
	if err := os.WriteFile(filepath.Join(composeDir, "hoodi.json"), []byte(data), 0644); err != nil {
		t.Fatalf("failed to write fallback deployment file: %v", err)
	}

	svc := &Service{
		contractsDir: dir,
		cfg: configs.L2{
			Dispute: configs.DisputeConfig{
				NetworkName: "hoodi",
			},
		},
	}

	got, err := svc.parseDeploymentContracts()
	if err != nil {
		t.Fatalf("expected parse to succeed, got: %v", err)
	}

	if got.ComposeL2OutputOracleAddress != common.HexToAddress("0x3333333333333333333333333333333333333333") {
		t.Fatalf("unexpected L2 output oracle address: %s", got.ComposeL2OutputOracleAddress)
	}
	if got.DisputeGameFactoryAddress != common.HexToAddress("0x4444444444444444444444444444444444444444") {
		t.Fatalf("unexpected dispute game factory address: %s", got.DisputeGameFactoryAddress)
	}
}
