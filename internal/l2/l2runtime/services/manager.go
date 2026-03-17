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
	rootDir                   string
	dockerFilePath            string
	flashblocksDockerFilePath string
	sidecarDockerFilePath     string
	frontendDockerFilePath    string
	flashblocksEnabled        bool
	sidecarEnabled            bool
	frontendEnabled           bool
	logger                    *slog.Logger
}

// NewManager creates a new service manager
func NewManager(rootDir, dockerFilePath string) *Manager {
	return &Manager{
		rootDir:        rootDir,
		dockerFilePath: dockerFilePath,
		logger:         logger.Named("service_manager"),
	}
}

// WithFlashblocks enables flashblocks support with the specified docker file
func (m *Manager) WithFlashblocks(flashblocksDockerFilePath string) *Manager {
	m.flashblocksDockerFilePath = flashblocksDockerFilePath
	m.flashblocksEnabled = true
	return m
}

// WithSidecar enables sidecar support with the specified docker file
func (m *Manager) WithSidecar(sidecarDockerFilePath string) *Manager {
	m.sidecarDockerFilePath = sidecarDockerFilePath
	m.sidecarEnabled = true
	return m
}

// WithFrontend enables Ethera Labs Console with the specified docker file
func (m *Manager) WithFrontend(frontendDockerFilePath string) *Manager {
	m.frontendDockerFilePath = frontendDockerFilePath
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

	dockerFiles := []string{m.dockerFilePath}

	if m.flashblocksEnabled && m.flashblocksDockerFilePath != "" {
		dockerFiles = append(dockerFiles, m.flashblocksDockerFilePath)
		services = append(services,
			"op-rbuilder-a",
			"op-rbuilder-b",
			"rollup-boost-a",
			"rollup-boost-b",
		)
	}

	if m.sidecarEnabled && m.sidecarDockerFilePath != "" {
		dockerFiles = append(dockerFiles, m.sidecarDockerFilePath)
		services = append(services,
			"sidecar-a",
			"sidecar-b",
		)
	}

	// Frontend (ethera-console) is started separately after contract deployment

	if len(dockerFiles) > 1 {
		m.logger.With("services", services, "flashblocks", m.flashblocksEnabled, "sidecar", m.sidecarEnabled).Info("starting L2 services")

		if err := docker.DockerUpMultiFile(ctx, dockerFiles, env, services...); err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}
	} else {
		m.logger.With("services", services).Info("starting L2 services")

		if err := docker.DockerUp(ctx, m.dockerFilePath, env, services...); err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}
	}

	m.logger.Info("L2 services started successfully")
	return nil
}

// StartFlashblocks starts flashblocks services (op-rbuilder and rollup-boost)
func (m *Manager) StartFlashblocks(ctx context.Context, env map[string]string) error {
	if !m.flashblocksEnabled || m.flashblocksDockerFilePath == "" {
		return fmt.Errorf("flashblocks not enabled or docker file not set")
	}

	services := []string{
		"op-rbuilder-a",
		"op-rbuilder-b",
		"rollup-boost-a",
		"rollup-boost-b",
	}

	m.logger.With("services", services).Info("starting flashblocks services")

	if err := docker.DockerUpMultiFile(ctx, []string{m.dockerFilePath, m.flashblocksDockerFilePath}, env, services...); err != nil {
		return fmt.Errorf("failed to start flashblocks services: %w", err)
	}

	m.logger.Info("flashblocks services started successfully")
	return nil
}

// StartFrontend builds and starts the Ethera Labs Console. Must be called after contract deployment
// so that env contains CONTRACT_BRIDGE_ADDRESS and CONTRACT_TOKEN_ADDRESS.
func (m *Manager) StartFrontend(ctx context.Context, dockerFiles []string, env map[string]string) error {
	if !m.frontendEnabled || m.frontendDockerFilePath == "" {
		return fmt.Errorf("frontend not enabled or docker file not set")
	}

	allFiles := append(dockerFiles, m.frontendDockerFilePath)
	m.logger.Info("building and starting Ethera Labs Console")

	if err := docker.DockerBuildMultiFile(ctx, allFiles, env, "ethera-console"); err != nil {
		return fmt.Errorf("failed to build ethera-console: %w", err)
	}

	if err := docker.DockerUpMultiFile(ctx, allFiles, env, "ethera-console"); err != nil {
		return fmt.Errorf("failed to start ethera-console: %w", err)
	}

	m.logger.Info("Ethera Labs Console started successfully")
	return nil
}
