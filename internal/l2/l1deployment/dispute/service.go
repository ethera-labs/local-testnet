package dispute

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/url"
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
	rootDir      string
	contractsDir string // Path to cloned ethera-contracts repo
	deployerPK   string
	cfg          configs.L2
	logger       *slog.Logger
}

// NewService creates a new dispute deployment service.
// contractsRootDir must point at the root of the ethera-contracts checkout;
// L1-settlement is the subdir actually used by this service.
func NewService(rootDir, contractsRootDir string, cfg configs.L2) *Service {
	return &Service{
		rootDir:      rootDir,
		contractsDir: filepath.Join(contractsRootDir, "L1-settlement"),
		deployerPK:   cfg.Wallet.PrivateKey,
		cfg:          cfg,
		logger:       logger.Named("dispute_deployer"),
	}
}

// Deploy executes the full deployment workflow and returns DisputeGameFactory proxy address
func (s *Service) Deploy(ctx context.Context) (common.Address, error) {
	s.logger.Info("starting dispute contracts deployment")

	if _, err := os.Stat(s.contractsDir); os.IsNotExist(err) {
		return common.Address{}, fmt.Errorf("L1-settlement directory not found at %s. Make sure ethera-contracts repository is cloned first", s.contractsDir)
	}

	s.logger.Info("generating networks.toml")
	if err := s.generateNetworksToml(); err != nil {
		return common.Address{}, fmt.Errorf("failed to generate networks.toml: %w", err)
	}

	s.logger.Info("generating .env file")
	if err := s.generateEnvFile(); err != nil {
		return common.Address{}, fmt.Errorf("failed to generate .env file: %w", err)
	}

	s.logger.Info("running just setup")
	if err := s.runJustCommand(ctx, "setup"); err != nil {
		return common.Address{}, fmt.Errorf("failed to run just setup: %w", err)
	}

	s.logger.Info("running just build")
	if err := s.runJustCommand(ctx, "build"); err != nil {
		return common.Address{}, fmt.Errorf("failed to run just build: %w", err)
	}

	s.logger.Info("running just deploy")
	if err := s.runJustCommand(ctx, "deploy-network", s.cfg.Dispute.NetworkName); err != nil {
		return common.Address{}, fmt.Errorf("failed to deploy network '%s': %w", s.cfg.Dispute.NetworkName, err)
	}

	s.logger.Info("parsing deployments.json")
	addr, err := s.parseDisputeGameFactoryAddress()
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to parse DisputeGameFactory address: %w", err)
	}

	s.logger.With("address", addr).Info("dispute contracts deployed successfully")

	return addr, nil
}

// generateNetworksToml creates networks.toml from template and config
func (s *Service) generateNetworksToml() error {
	tmplContent, err := templatesFS.ReadFile("networks.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read template file: %w", err)
	}

	tmpl, err := template.New("networks").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	type templateData struct {
		NetworkName                     string
		RpcURL                          string
		ChainID                         int
		ExplorerURL                     string
		ExplorerAPIURL                  string
		VerifierAddress                 string
		OwnerAddress                    string
		ProposerAddress                 string
		AggregationVkey                 string
		GuardianAddress                 string
		ProofMaturityDelaySeconds       int
		DisputeGameFinalityDelaySeconds int
		DisputeGameInitBond             string
	}

	data := templateData{
		NetworkName:                     s.cfg.Dispute.NetworkName,
		RpcURL:                          hostAccessibleRPCURL(s.cfg.L1ElURL),
		ChainID:                         s.cfg.L1ChainID,
		ExplorerURL:                     s.cfg.Dispute.ExplorerURL,
		ExplorerAPIURL:                  s.cfg.Dispute.ExplorerAPIURL,
		VerifierAddress:                 s.cfg.Dispute.VerifierAddress,
		OwnerAddress:                    s.cfg.Dispute.OwnerAddress,
		ProposerAddress:                 s.cfg.Dispute.ProposerAddress,
		AggregationVkey:                 s.cfg.Dispute.AggregationVkey,
		GuardianAddress:                 s.cfg.Dispute.GuardianAddress,
		ProofMaturityDelaySeconds:       s.cfg.Dispute.ProofMaturityDelaySeconds,
		DisputeGameFinalityDelaySeconds: s.cfg.Dispute.DisputeGameFinalityDelaySeconds,
		DisputeGameInitBond:             s.cfg.Dispute.DisputeGameInitBond,
	}

	outputPath := filepath.Join(s.contractsDir, "networks.toml")
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create networks.toml: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// hostAccessibleRPCURL rewrites Docker host aliases to localhost for commands
// that execute directly on the host machine (e.g. just/forge in dispute deploy).
func hostAccessibleRPCURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if parsed.Hostname() != "host.docker.internal" {
		return raw
	}

	port := parsed.Port()
	if port == "" {
		parsed.Host = "127.0.0.1"
		return parsed.String()
	}

	parsed.Host = net.JoinHostPort("127.0.0.1", port)
	return parsed.String()
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
		DeployerPrivateKey string
	}{
		DeployerPrivateKey: s.deployerPK,
	}

	envPath := filepath.Join(s.contractsDir, ".env")
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
	cmd.Dir = s.contractsDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	s.logger.
		With("command", fmt.Sprintf("just %s", strings.Join(args, " "))).
		With("working_dir", s.contractsDir).
		Info("executing just command")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("command 'just %s' failed in directory %s: %w", strings.Join(args, " "), s.contractsDir, err)
	}

	return nil
}

// parseDisputeGameFactoryAddress reads deployments.json and extracts DisputeGameFactory proxy address
func (s *Service) parseDisputeGameFactoryAddress() (common.Address, error) {
	deploymentsPath := filepath.Join(s.contractsDir, "deployments.json")

	if data, err := os.ReadFile(deploymentsPath); err == nil {
		var deployments map[string]struct {
			DisputeGameFactory struct {
				Proxy string `json:"proxy"`
			} `json:"DisputeGameFactory"`
		}

		if err := json.Unmarshal(data, &deployments); err == nil {
			if network, ok := deployments[s.cfg.Dispute.NetworkName]; ok {
				if network.DisputeGameFactory.Proxy != "" {
					return common.HexToAddress(network.DisputeGameFactory.Proxy), nil
				}
				return common.Address{}, fmt.Errorf("DisputeGameFactory proxy address is empty")
			}
		}
	}

	// Fallback to ethera deployment layout: deployments/ethera/<network>.json
	etheraPath := filepath.Join(s.contractsDir, "deployments", "compose", s.cfg.Dispute.NetworkName+".json")
	data, err := os.ReadFile(etheraPath)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to read deployments.json or ethera deployments: %w", err)
	}

	var etheraDeployments map[string]struct {
		Contracts struct {
			DisputeGameFactory struct {
				ProxyAddress string `json:"proxyAddress"`
			} `json:"DisputeGameFactory"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(data, &etheraDeployments); err != nil {
		return common.Address{}, fmt.Errorf("failed to parse ethera deployments file: %w", err)
	}

	network, ok := etheraDeployments[s.cfg.Dispute.NetworkName]
	if !ok {
		return common.Address{}, fmt.Errorf("%s deployment not found in ethera deployments file", s.cfg.Dispute.NetworkName)
	}

	if network.Contracts.DisputeGameFactory.ProxyAddress == "" {
		return common.Address{}, fmt.Errorf("DisputeGameFactory proxy address is empty")
	}

	return common.HexToAddress(network.Contracts.DisputeGameFactory.ProxyAddress), nil
}
