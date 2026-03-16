package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ethera-labs/local-testnet/internal/l2/infra/docker"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

// Manager manages L2 service lifecycle via docker-compose
type Manager struct {
	rootDir                    string
	composeFilePath            string
	flashblocksComposeFilePath string
	sidecarComposeFilePath     string
	frontendComposeFilePath    string
	flashblocksEnabled         bool
	sidecarEnabled             bool
	frontendEnabled            bool
	logger                     *slog.Logger
}

// NewManager creates a new service manager
func NewManager(rootDir, composeFilePath string) *Manager {
	return &Manager{
		rootDir:         rootDir,
		composeFilePath: composeFilePath,
		logger:          logger.Named("service_manager"),
	}
}

// WithFlashblocks enables flashblocks support with the specified compose file
func (m *Manager) WithFlashblocks(flashblocksComposeFilePath string) *Manager {
	m.flashblocksComposeFilePath = flashblocksComposeFilePath
	m.flashblocksEnabled = true
	return m
}

// WithSidecar enables sidecar support with the specified compose file
func (m *Manager) WithSidecar(sidecarComposeFilePath string) *Manager {
	m.sidecarComposeFilePath = sidecarComposeFilePath
	m.sidecarEnabled = true
	return m
}

// WithFrontend enables Compose Network Console with the specified compose file
func (m *Manager) WithFrontend(frontendComposeFilePath string) *Manager {
	m.frontendComposeFilePath = frontendComposeFilePath
	m.frontendEnabled = true
	return m
}

// StartAll starts all L2 services
func (m *Manager) StartAll(ctx context.Context, env map[string]string) error {
	services := []string{
		"publisher",
		"op-geth-a",
		"op-geth-b",
		"op-node-a",
		"op-node-b",
		"op-batcher-a",
		"op-batcher-b",
		"op-proposer-a",
		"op-proposer-b",
	}

	composeFiles := []string{m.composeFilePath}

	if m.flashblocksEnabled && m.flashblocksComposeFilePath != "" {
		composeFiles = append(composeFiles, m.flashblocksComposeFilePath)
		services = append(services,
			"op-rbuilder-a",
			"op-rbuilder-b",
			"rollup-boost-a",
			"rollup-boost-b",
		)
	}

	if m.sidecarEnabled && m.sidecarComposeFilePath != "" {
		composeFiles = append(composeFiles, m.sidecarComposeFilePath)
		services = append(services,
			"sidecar-a",
			"sidecar-b",
		)
	}

	// Frontend (compose-console) is started separately after contract deployment

	if len(composeFiles) > 1 {
		m.logger.With("services", services, "flashblocks", m.flashblocksEnabled, "sidecar", m.sidecarEnabled).Info("starting L2 services")

		if err := docker.ComposeUpMultiFile(ctx, composeFiles, env, services...); err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}
	} else {
		m.logger.With("services", services).Info("starting L2 services")

		if err := docker.ComposeUp(ctx, m.composeFilePath, env, services...); err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}
	}

	m.logger.Info("L2 services started successfully")
	return nil
}

// StartFlashblocks starts flashblocks services (op-rbuilder and rollup-boost)
func (m *Manager) StartFlashblocks(ctx context.Context, env map[string]string) error {
	if !m.flashblocksEnabled || m.flashblocksComposeFilePath == "" {
		return fmt.Errorf("flashblocks not enabled or compose file not set")
	}

	services := []string{
		"op-rbuilder-a",
		"op-rbuilder-b",
		"rollup-boost-a",
		"rollup-boost-b",
	}

	m.logger.With("services", services).Info("starting flashblocks services")

	if err := docker.ComposeUpMultiFile(ctx, []string{m.composeFilePath, m.flashblocksComposeFilePath}, env, services...); err != nil {
		return fmt.Errorf("failed to start flashblocks services: %w", err)
	}

	m.logger.Info("flashblocks services started successfully")
	return nil
}

// StartFrontend builds and starts the Compose Network Console. Must be called after contract deployment
// so that env contains CONTRACT_BRIDGE_ADDRESS and CONTRACT_TOKEN_ADDRESS.
func (m *Manager) StartFrontend(ctx context.Context, composeFiles []string, env map[string]string) error {
	if !m.frontendEnabled || m.frontendComposeFilePath == "" {
		return fmt.Errorf("frontend not enabled or compose file not set")
	}

	allFiles := append(composeFiles, m.frontendComposeFilePath)
	m.logger.Info("building and starting Compose Network Console")

	if err := docker.ComposeBuildMultiFile(ctx, allFiles, env, "compose-console"); err != nil {
		return fmt.Errorf("failed to build compose-console: %w", err)
	}

	if err := docker.ComposeUpMultiFile(ctx, allFiles, env, "compose-console"); err != nil {
		return fmt.Errorf("failed to start compose-console: %w", err)
	}

	m.logger.Info("Compose Network Console started successfully")
	return nil
}
