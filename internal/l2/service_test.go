package l2

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/blockscout"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/git"
	"github.com/ethera-labs/local-testnet/internal/l2/l1deployment"
	"github.com/ethera-labs/local-testnet/internal/l2/l2runtime/contracts"
	"github.com/ethereum/go-ethereum/common"
)

type stubCloner struct {
	baseDir string
	repos   []git.Repository
}

func (s *stubCloner) CloneAll(_ context.Context, baseDir string, repos []git.Repository) error {
	s.baseDir = baseDir
	s.repos = append([]git.Repository(nil), repos...)
	return nil
}

type stubL1Orchestrator struct {
	state l1deployment.DeploymentState
	err   error
}

func (s stubL1Orchestrator) Execute(context.Context, configs.L2) (l1deployment.DeploymentState, error) {
	return s.state, s.err
}

type stubL2ConfigOrchestrator struct {
	err error
}

func (s stubL2ConfigOrchestrator) Execute(context.Context, configs.L2, l1deployment.DeploymentState) error {
	return s.err
}

type stubL2RuntimeOrchestrator struct {
	contracts map[configs.L2ChainName]map[contracts.ContractName]common.Address
	err       error
}

func (s stubL2RuntimeOrchestrator) Execute(context.Context, configs.L2, l1deployment.DeploymentState) (map[configs.L2ChainName]map[contracts.ContractName]common.Address, error) {
	return s.contracts, s.err
}

type stubBlockscoutService struct {
	called bool
}

func (s *stubBlockscoutService) Run(context.Context, []blockscout.RollupConfig, string, string) error {
	s.called = true
	return nil
}

type stubOutputGenerator struct {
	called    bool
	contracts map[configs.L2ChainName]map[contracts.ContractName]common.Address
}

func (s *stubOutputGenerator) Generate(_ context.Context, deployed map[configs.L2ChainName]map[contracts.ContractName]common.Address) error {
	s.called = true
	s.contracts = deployed
	return nil
}

func TestServiceDeployDoesNotRestartOpGethAfterPhase3(t *testing.T) {
	t.Parallel()

	rootDir := t.TempDir()
	opGethDir := filepath.Join(rootDir, "op-geth")
	publisherDir := filepath.Join(rootDir, "publisher")

	cfg := configs.L2{
		L1ChainID:         1,
		L1ElURL:           "https://l1-rpc.example.test",
		L1ClURL:           "https://l1-beacon.example.test",
		EtheraNetworkName: "compose-local",
		Wallet: configs.Wallet{
			PrivateKey: "0x01",
			Address:    "0x0000000000000000000000000000000000000001",
		},
		CoordinatorPrivateKey: "0x02",
		Repositories: map[configs.RepositoryName]configs.Repository{
			configs.RepositoryNameOpGeth: {
				LocalPath: opGethDir,
			},
			configs.RepositoryNamePublisher: {
				LocalPath: publisherDir,
			},
		},
	}

	deploymentState := l1deployment.DeploymentState{
		DisputeGameFactoryAddress: common.HexToAddress("0x0000000000000000000000000000000000000010"),
	}

	deployedContracts := map[configs.L2ChainName]map[contracts.ContractName]common.Address{
		configs.L2ChainNameRollupA: {
			contracts.ContractNameMailbox: common.HexToAddress("0x0000000000000000000000000000000000000020"),
		},
		configs.L2ChainNameRollupB: {
			contracts.ContractNameMailbox: common.HexToAddress("0x0000000000000000000000000000000000000030"),
		},
	}

	cloner := &stubCloner{}
	blockscoutSvc := &stubBlockscoutService{}
	outputGen := &stubOutputGenerator{}

	svc := NewService(
		rootDir,
		cloner,
		stubL1Orchestrator{state: deploymentState},
		stubL2ConfigOrchestrator{},
		stubL2RuntimeOrchestrator{contracts: deployedContracts},
		blockscoutSvc,
		outputGen,
	)

	if err := svc.Deploy(context.Background(), cfg); err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	if blockscoutSvc.called {
		t.Fatalf("Blockscout should not start when disabled")
	}

	if !outputGen.called {
		t.Fatalf("output generator was not called")
	}

	if got, want := outputGen.contracts, deployedContracts; len(got) != len(want) {
		t.Fatalf("output generator received %d chains, want %d", len(got), len(want))
	}

	if got, want := cloner.baseDir, filepath.Join(rootDir, localnetDirName, servicesDirName); got != want {
		t.Fatalf("CloneAll() baseDir = %q, want %q", got, want)
	}

	if len(cloner.repos) != 0 {
		t.Fatalf("CloneAll() received %d repos, want 0 for local-path repos", len(cloner.repos))
	}
}
