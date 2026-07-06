package dispute

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/logger"
	"github.com/ethereum/go-ethereum/common"
)

//go:embed *.tmpl
var templatesFS embed.FS

// Service handles dispute game factory deployment
type Service struct {
	contractsRoot string
	deployerPK    string
	cfg           configs.L2
	logger        *slog.Logger
}

// NewService creates a new dispute deployment service.
// etheraContractsDir is the resolved path to the ethera-contracts repository root,
// honoring both clone (.localnet/services/...) and local-path configurations.
func NewService(etheraContractsDir string, cfg configs.L2) *Service {
	return &Service{
		contractsRoot: etheraContractsDir,
		deployerPK:    cfg.Wallet.PrivateKey,
		cfg:           cfg,
		logger:        logger.Named("dispute_deployer"),
	}
}

type DeploymentContracts struct {
	DisputeGameFactoryAddress  common.Address
	AnchorStateRegistryAddress common.Address
}

// Deploy executes the dispute-contract deployment workflow and returns the
// deployed L1 contract addresses localnet needs in later phases.
func (s *Service) Deploy(ctx context.Context) (DeploymentContracts, error) {
	s.logger.Info("starting dispute contracts deployment")

	if _, err := os.Stat(filepath.Join(s.contractsRoot, "justfile")); err != nil {
		return DeploymentContracts{}, fmt.Errorf("ethera-contracts root not found at %s: %w", s.contractsRoot, err)
	}

	s.logger.Info("generating contracts .env file")
	if err := s.generateEnvFile(); err != nil {
		return DeploymentContracts{}, fmt.Errorf("failed to generate .env file: %w", err)
	}

	s.logger.Info("running just build")
	if err := s.runJustCommand(ctx, "build"); err != nil {
		return DeploymentContracts{}, fmt.Errorf("failed to run just build: %w", err)
	}

	s.logger.Info("generating contracts config.json")
	if err := s.generateContractsConfig(); err != nil {
		return DeploymentContracts{}, fmt.Errorf("failed to generate contracts config.json: %w", err)
	}

	if s.cfg.OPSuccinct.Enabled {
		s.logger.Info("running just l1-deploy-sp1-mock-verifier")
		if err := s.runJustCommand(ctx, "l1-deploy-sp1-mock-verifier"); err != nil {
			return DeploymentContracts{}, fmt.Errorf("failed to deploy SP1 mock verifier: %w", err)
		}
	}

	s.logger.Info("running just l1-deploy-shared")
	if err := s.runJustCommand(ctx, "l1-deploy-shared"); err != nil {
		return DeploymentContracts{}, fmt.Errorf("failed to deploy shared L1 contracts: %w", err)
	}

	s.logger.Info("parsing contracts config.json")
	contracts, err := s.parseDeploymentContracts()
	if err != nil {
		return DeploymentContracts{}, fmt.Errorf("failed to parse deployment contracts: %w", err)
	}

	s.logger.With(
		"dispute_game_factory", contracts.DisputeGameFactoryAddress,
		"anchor_state_registry", contracts.AnchorStateRegistryAddress,
	).Info("dispute contracts deployed successfully")

	return contracts, nil
}

// LoadDeploymentContracts reads the latest shared settlement deployment output.
func (s *Service) LoadDeploymentContracts() (DeploymentContracts, error) {
	return s.parseDeploymentContracts()
}

// generateEnvFile creates .env file from template
func (s *Service) generateEnvFile() error {
	tmplContent, err := templatesFS.ReadFile("env.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read env template file: %w", err)
	}

	tmpl, err := template.New("env").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse env template: %w", err)
	}

	data := struct {
		RpcURL             string
		DeployerPrivateKey string
	}{
		RpcURL:             s.cfg.L1ElURL,
		DeployerPrivateKey: normalizePrivateKey(s.deployerPK),
	}

	envPath := filepath.Join(s.contractsRoot, ".env")
	file, err := os.Create(envPath)
	if err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute env template: %w", err)
	}

	if err := os.Chmod(envPath, 0600); err != nil {
		return fmt.Errorf("failed to set .env file permissions: %w", err)
	}

	return nil
}

// runJustCommand executes a just command in the contracts directory
func (s *Service) runJustCommand(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "just", args...)
	cmd.Dir = s.contractsRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	s.logger.
		With("command", fmt.Sprintf("just %s", strings.Join(args, " "))).
		With("working_dir", s.contractsRoot).
		Info("executing just command")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command 'just %s' failed in directory %s: %w", strings.Join(args, " "), s.contractsRoot, err)
	}

	return nil
}

const (
	zeroAddress = "0x0000000000000000000000000000000000000000"
	// Non-zero stand-in accepted by l1-deploy-shared when no SP1 mock verifier
	// is deployed; nothing submits proofs in that mode.
	inertSP1Verifier = "0x000000000000000000000000000000000000dEaD"
)

func (s *Service) generateContractsConfig() error {
	// l1-deploy-shared rejects a zero verifier. With op-succinct enabled the SP1
	// mock verifier deployment overwrites this seed before l1-deploy-shared reads
	// it; otherwise no proof producer runs and an inert placeholder stands in.
	sp1Verifier := inertSP1Verifier
	if s.cfg.OPSuccinct.Enabled {
		sp1Verifier = zeroAddress
	}

	config := map[string]any{
		"l1": map[string]any{
			"guardian":                        s.cfg.Dispute.GuardianAddress,
			"proxyAdminOwner":                 s.cfg.Dispute.OwnerAddress,
			"defaultAdmin":                    s.cfg.Dispute.OwnerAddress,
			"depositWhitelistAdmin":           s.cfg.Dispute.OwnerAddress,
			"authorizedProposer":              s.cfg.Dispute.ProposerAddress,
			"sp1Verifier":                     sp1Verifier,
			"aggregationVkey":                 s.cfg.Dispute.AggregationVkey,
			"proofMaturityDelaySeconds":       s.cfg.Dispute.ProofMaturityDelaySeconds,
			"disputeGameFinalityDelaySeconds": s.cfg.Dispute.DisputeGameFinalityDelaySeconds,
			"disputeGameInitBond":             s.cfg.Dispute.DisputeGameInitBond,
			"deployed": map[string]any{
				"l1ChainId":           0,
				"proxyAdmin":          zeroAddress,
				"superchainConfig":    zeroAddress,
				"disputeGameFactory":  zeroAddress,
				"anchorStateRegistry": zeroAddress,
				"ethLockbox":          zeroAddress,
				"depositWhitelist":    zeroAddress,
				"composeDisputeGame":  zeroAddress,
				"erc20LockboxProxy":   zeroAddress,
			},
		},
		"rollups": map[string]any{},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal contracts config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(s.configPath(), data, 0644); err != nil {
		return fmt.Errorf("failed to write contracts config: %w", err)
	}
	return nil
}

// parseDeploymentContracts reads contracts/config.json after l1-deploy-shared writes l1.deployed.
func (s *Service) parseDeploymentContracts() (DeploymentContracts, error) {
	data, err := os.ReadFile(s.configPath())
	if err != nil {
		return DeploymentContracts{}, fmt.Errorf("failed to read contracts config.json: %w", err)
	}

	var cfg struct {
		L1 struct {
			Deployed struct {
				DisputeGameFactory  string `json:"disputeGameFactory"`
				AnchorStateRegistry string `json:"anchorStateRegistry"`
			} `json:"deployed"`
		} `json:"l1"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DeploymentContracts{}, fmt.Errorf("failed to parse contracts config.json: %w", err)
	}

	disputeGameFactory, err := requiredAddress("l1.deployed.disputeGameFactory", cfg.L1.Deployed.DisputeGameFactory)
	if err != nil {
		return DeploymentContracts{}, err
	}
	anchorStateRegistry, err := requiredAddress("l1.deployed.anchorStateRegistry", cfg.L1.Deployed.AnchorStateRegistry)
	if err != nil {
		return DeploymentContracts{}, err
	}

	return DeploymentContracts{
		DisputeGameFactoryAddress:  disputeGameFactory,
		AnchorStateRegistryAddress: anchorStateRegistry,
	}, nil
}

func (s *Service) configPath() string {
	return filepath.Join(s.contractsRoot, "config.json")
}

func requiredAddress(name, value string) (common.Address, error) {
	if !common.IsHexAddress(value) {
		return common.Address{}, fmt.Errorf("%s is not a valid address: %q", name, value)
	}
	address := common.HexToAddress(value)
	if address == (common.Address{}) {
		return common.Address{}, fmt.Errorf("%s is zero", name)
	}
	return address, nil
}

func normalizePrivateKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "0x") {
		return trimmed
	}
	return "0x" + trimmed
}
