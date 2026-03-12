package registry

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

//go:embed *.tmpl
var templatesFS embed.FS

// Configurator creates the custom registry structure that prevents
// Publisher and OP-geth from loading embedded chain definitions
type Configurator struct {
	logger *slog.Logger
}

func NewConfigurator() *Configurator {
	return &Configurator{
		logger: logger.Named("registry_configurator"),
	}
}

// SetupRegistry creates the complete registry directory structure
// This includes the network-level ethera.toml and individual rollup.toml for each chain
func (c *Configurator) SetupRegistry(localnetDir string, cfg configs.L2, gameFactoryAddr common.Address) error {
	registryNetworkDir := filepath.Join(localnetDir, "registry", "networks", cfg.EtheraNetworkName)
	if err := os.MkdirAll(registryNetworkDir, 0755); err != nil {
		return fmt.Errorf("failed to create registry network directory: %w", err)
	}

	c.logger.Info("created registry network directory", "path", registryNetworkDir)

	if err := c.generateEtheraToml(registryNetworkDir, cfg, gameFactoryAddr); err != nil {
		return fmt.Errorf("failed to generate ethera.toml: %w", err)
	}

	for chainName, chainCfg := range cfg.ChainConfigs {
		if err := c.generateRollupToml(registryNetworkDir, string(chainName), chainCfg); err != nil {
			return fmt.Errorf("failed to generate rollup.toml for %s: %w", chainName, err)
		}
	}

	return nil
}

func (c *Configurator) generateEtheraToml(registryNetworkDir string, cfg configs.L2, gameFactoryAddr common.Address) error {
	const etheraFileName = "ethera.toml"

	tmplContent, err := templatesFS.ReadFile("ethera.toml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read %s template: %w", etheraFileName, err)
	}

	tmpl, err := template.New(etheraFileName).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse %s template: %w", etheraFileName, err)
	}

	etheraTomlPath := filepath.Join(registryNetworkDir, etheraFileName)
	file, err := os.Create(etheraTomlPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", etheraFileName, err)
	}
	defer file.Close()

	data := struct {
		NetworkName        string
		L1ELURL            string
		L1ChainID          uint64
		ExplorerURL        string
		DisputeGameFactory string
	}{
		NetworkName:        cfg.EtheraNetworkName,
		L1ELURL:            cfg.L1ElURL,
		L1ChainID:          uint64(cfg.L1ChainID),
		ExplorerURL:        cfg.Dispute.ExplorerURL,
		DisputeGameFactory: gameFactoryAddr.Hex(),
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute %s template: %w", etheraFileName, err)
	}

	c.logger.Info("created network registry configuration", "path", etheraTomlPath)

	return nil
}

// UpdateRollupMailboxAddresses updates Mailbox values in generated rollup TOML files.
// This must run after L2 contracts are deployed, so chain IDs resolve to real mailbox
// addresses (instead of zero placeholders) in the registry.
func (c *Configurator) UpdateRollupMailboxAddresses(
	localnetDir string,
	networkName string,
	mailboxByChain map[configs.L2ChainName]common.Address,
) error {
	registryNetworkDir := filepath.Join(localnetDir, "registry", "networks", networkName)

	for chainName, mailboxAddr := range mailboxByChain {
		if mailboxAddr == (common.Address{}) {
			return fmt.Errorf("mailbox address is zero for chain %s", chainName)
		}

		rollupTomlPath := filepath.Join(registryNetworkDir, fmt.Sprintf("%s.toml", chainName))
		raw, err := os.ReadFile(rollupTomlPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rollupTomlPath, err)
		}

		updated, err := replaceMailboxLine(string(raw), mailboxAddr.Hex())
		if err != nil {
			return fmt.Errorf("failed to update mailbox in %s: %w", rollupTomlPath, err)
		}

		if err := os.WriteFile(rollupTomlPath, []byte(updated), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", rollupTomlPath, err)
		}

		c.logger.Info("updated chain registry mailbox address",
			"chain", chainName,
			"mailbox", mailboxAddr.Hex(),
			"path", rollupTomlPath,
		)
	}

	return nil
}

func (c *Configurator) generateRollupToml(registryNetworkDir, chainName string, chainCfg configs.Chain) error {
	rollupFileName := chainName + ".toml"

	tmplContent, err := templatesFS.ReadFile("rollup.toml.tmpl")
	if err != nil {
		return fmt.Errorf("failed to read rollup.toml template: %w", err)
	}

	tmpl, err := template.New("rollup.toml").Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse rollup.toml template: %w", err)
	}

	rollupTomlPath := filepath.Join(registryNetworkDir, rollupFileName)
	file, err := os.Create(rollupTomlPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", rollupFileName, err)
	}
	defer file.Close()

	// Extract suffix from chain name (e.g., "rollup-a" -> "a")
	// The sequencer host is "op-geth-a" not "op-geth-rollup-a"
	suffix := chainName
	if len(chainName) > 7 && chainName[:7] == "rollup-" {
		suffix = chainName[7:]
	}

	data := struct {
		ChainName      string
		ChainID        uint64
		RPCPort        int
		SequencerHost  string
		MailboxAddress string
		L2GenesisTime  uint64
	}{
		ChainName:      chainName,
		ChainID:        uint64(chainCfg.ID),
		RPCPort:        chainCfg.RPCPort,
		SequencerHost:  "op-geth-" + suffix,
		MailboxAddress: "0x0000000000000000000000000000000000000000", // Placeholder: contracts not deployed yet
		L2GenesisTime:  0,                                            // Use 0 for testnet genesis time
	}

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("failed to execute rollup.toml template: %w", err)
	}

	c.logger.Info("created chain registry configuration", "chain", chainName, "chain_id", chainCfg.ID, "rpc_port", chainCfg.RPCPort, "path", rollupTomlPath)

	return nil
}

func replaceMailboxLine(content, mailboxAddress string) (string, error) {
	lines := strings.Split(content, "\n")
	updated := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "Mailbox = ") {
			continue
		}

		indentLen := len(line) - len(strings.TrimLeft(line, " \t"))
		indent := line[:indentLen]
		lines[i] = fmt.Sprintf(`%sMailbox = "%s"`, indent, mailboxAddress)
		updated = true
		break
	}

	if !updated {
		return "", fmt.Errorf("Mailbox entry not found")
	}

	return strings.Join(lines, "\n"), nil
}
