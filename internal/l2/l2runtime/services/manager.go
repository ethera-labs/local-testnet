package services

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/ethera-labs/local-testnet/configs"
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

// StartAll starts the core L2 services (excluding op-succinct).
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

		if err := docker.ComposeUpMultiFile(ctx, dockerFiles, env, services...); err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}
	} else {
		m.logger.With("services", services).Info("starting L2 services")

		if err := docker.ComposeUp(ctx, m.dockerFilePath, env, services...); err != nil {
			return fmt.Errorf("failed to start services: %w", err)
		}
	}

	m.logger.Info("L2 services started successfully")
	return nil
}

// StartOpSuccinct starts selected op-succinct services after the rest of L2 is running.
func (m *Manager) StartOpSuccinct(ctx context.Context, env map[string]string, enabledChains []configs.L2ChainName) error {
	if len(enabledChains) == 0 {
		m.logger.Info("op-succinct has no enabled chains; skipping service startup")
		return nil
	}

	dbServices := make([]string, 0, len(enabledChains))
	proposerServices := make([]string, 0, len(enabledChains))

	for _, chain := range enabledChains {
		switch chain {
		case configs.L2ChainNameRollupA:
			dbServices = append(dbServices, "op-succinct-db-a")
			proposerServices = append(proposerServices, "op-succinct-a")
		case configs.L2ChainNameRollupB:
			dbServices = append(dbServices, "op-succinct-db-b")
			proposerServices = append(proposerServices, "op-succinct-b")
		default:
			m.logger.With("chain", chain).Warn("unknown op-succinct chain; skipping")
		}
	}

	if len(proposerServices) == 0 {
		m.logger.Info("op-succinct has no valid enabled chains; skipping service startup")
		return nil
	}

	m.logger.With("services", dbServices).Info("starting op-succinct database services")
	if err := docker.ComposeUp(ctx, m.dockerFilePath, env, dbServices...); err != nil {
		return fmt.Errorf("failed to start op-succinct database services: %w", err)
	}

	if err := m.clearOpSuccinctLocks(ctx, dbServices); err != nil {
		return err
	}

	m.logger.With("services", proposerServices).Info("starting op-succinct proposer services")
	if err := docker.ComposeUp(ctx, m.dockerFilePath, env, proposerServices...); err != nil {
		return fmt.Errorf("failed to start op-succinct proposer services: %w", err)
	}

	m.logger.Info("op-succinct services started successfully")
	return nil
}

func (m *Manager) clearOpSuccinctLocks(ctx context.Context, dbContainers []string) error {
	for _, container := range dbContainers {
		if err := waitForPostgresReady(ctx, container); err != nil {
			return fmt.Errorf("op-succinct database %s is not ready: %w", container, err)
		}
		if err := clearChainLocks(ctx, container); err != nil {
			return fmt.Errorf("failed to clear op-succinct chain locks in %s: %w", container, err)
		}
	}

	m.logger.Info("cleared stale op-succinct database chain locks")
	return nil
}

func waitForPostgresReady(ctx context.Context, container string) error {
	deadline := time.Now().Add(60 * time.Second)
	var lastErr error

	for {
		cmd := exec.CommandContext(ctx, "docker", "exec", container, "pg_isready", "-U", "op-succinct", "-d", "op-succinct")
		if err := cmd.Run(); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if time.Now().After(deadline) {
			if lastErr == nil {
				lastErr = fmt.Errorf("timed out waiting for postgres readiness")
			}
			return lastErr
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func clearChainLocks(ctx context.Context, container string) error {
	const cleanupSQL = "DO $$ BEGIN IF to_regclass('public.chain_locks') IS NOT NULL THEN DELETE FROM chain_locks; END IF; END $$;"

	cmd := exec.CommandContext(
		ctx,
		"docker", "exec",
		container,
		"psql", "-U", "op-succinct", "-d", "op-succinct",
		"-v", "ON_ERROR_STOP=1",
		"-c", cleanupSQL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("psql cleanup failed: %w: %s", err, strings.TrimSpace(string(output)))
	}

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

	if err := docker.ComposeUpMultiFile(ctx, []string{m.dockerFilePath, m.flashblocksDockerFilePath}, env, services...); err != nil {
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

	if err := docker.ComposeBuildMultiFile(ctx, allFiles, env, "ethera-console"); err != nil {
		return fmt.Errorf("failed to build ethera-console: %w", err)
	}

	if err := docker.ComposeUpMultiFile(ctx, allFiles, env, "ethera-console"); err != nil {
		return fmt.Errorf("failed to start ethera-console: %w", err)
	}

	m.logger.Info("Ethera Labs Console started successfully")
	return nil
}
