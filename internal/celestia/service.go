package celestia

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethera-labs/local-testnet/configs"
)

const (
	l2NetworkName = "localnet-l2"
)

type preparedRuntime struct {
	ProjectName string
	ComposePath string
	RuntimeDir  string
	DataDir     string
}

func start(ctx context.Context, cfg configs.Celestia) error {
	applyDefaults(&cfg)
	if err := cfg.Validate(); err != nil {
		return err
	}

	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	runtime, err := prepareRuntime(ctx, rootDir, cfg)
	if err != nil {
		return err
	}

	args := []string{"up", "-d", "--remove-orphans"}
	if cfg.CeleniumEnabled {
		args = append(args, "--build")
	}

	slog.With("compose_file", runtime.ComposePath, "project_name", runtime.ProjectName).Info("starting Celestia compose stack")
	if err := composeRun(ctx, runtime.ComposePath, runtime.ProjectName, args...); err != nil {
		return fmt.Errorf("failed to start Celestia services: %w", err)
	}

	return nil
}

func stop(ctx context.Context, cfg configs.Celestia) error {
	applyDefaults(&cfg)

	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}
	runtimeDir, err := resolvePath(rootDir, cfg.RuntimeDir)
	if err != nil {
		return fmt.Errorf("failed to resolve runtime directory: %w", err)
	}
	composePath := filepath.Join(runtimeDir, composeFileName)
	if !fileExists(composePath) {
		slog.With("compose_file", composePath).Info("Celestia compose file not found, nothing to stop")
		return nil
	}

	if err := composeRun(ctx, composePath, cfg.ProjectName, "stop"); err != nil {
		return fmt.Errorf("failed to stop Celestia services: %w", err)
	}

	return nil
}

func clean(ctx context.Context, cfg configs.Celestia) error {
	applyDefaults(&cfg)

	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	runtimeDir, err := resolvePath(rootDir, cfg.RuntimeDir)
	if err != nil {
		return fmt.Errorf("failed to resolve runtime directory: %w", err)
	}
	dataDir, err := resolvePath(rootDir, cfg.DataDir)
	if err != nil {
		return fmt.Errorf("failed to resolve data directory: %w", err)
	}
	composePath := filepath.Join(runtimeDir, composeFileName)
	if fileExists(composePath) {
		if err := composeRun(ctx, composePath, cfg.ProjectName, "down", "-v", "--remove-orphans"); err != nil {
			return fmt.Errorf("failed to clean Celestia services: %w", err)
		}
	}

	if err := os.RemoveAll(runtimeDir); err != nil {
		return fmt.Errorf("failed to remove runtime directory %q: %w", runtimeDir, err)
	}
	if dataDir != runtimeDir {
		if err := os.RemoveAll(dataDir); err != nil {
			return fmt.Errorf("failed to remove data directory %q: %w", dataDir, err)
		}
	}

	servicesDir := filepath.Join(rootDir, ".localnet", "services")
	if cfg.CeleniumIndexer.LocalPath == "" {
		_ = os.RemoveAll(filepath.Join(servicesDir, "celenium-indexer"))
	}
	if cfg.CeleniumInterface.LocalPath == "" {
		_ = os.RemoveAll(filepath.Join(servicesDir, "celenium-interface"))
	}

	return nil
}

func show(ctx context.Context, cfg configs.Celestia) error {
	applyDefaults(&cfg)

	filter := fmt.Sprintf("label=com.docker.compose.project=%s", cfg.ProjectName)
	cmd := exec.CommandContext(ctx, "docker", "ps", "-a", "--filter", filter)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to list Celestia containers: %w", err)
	}
	return nil
}

func prepareRuntime(ctx context.Context, rootDir string, cfg configs.Celestia) (preparedRuntime, error) {
	runtimeDir, err := resolvePath(rootDir, cfg.RuntimeDir)
	if err != nil {
		return preparedRuntime{}, fmt.Errorf("failed to resolve runtime directory: %w", err)
	}
	dataDir, err := resolvePath(rootDir, cfg.DataDir)
	if err != nil {
		return preparedRuntime{}, fmt.Errorf("failed to resolve data directory: %w", err)
	}

	runtimeDirHost, err := toHostPath(runtimeDir)
	if err != nil {
		return preparedRuntime{}, fmt.Errorf("failed to resolve runtime host path: %w", err)
	}
	dataDirHost, err := toHostPath(dataDir)
	if err != nil {
		return preparedRuntime{}, fmt.Errorf("failed to resolve data host path: %w", err)
	}

	layout := runtimeLayout{
		RuntimeDir: runtimeDir,
		ConfigsDir: filepath.Join(runtimeDir, "configs"),
		ScriptsDir: filepath.Join(runtimeDir, "scripts"),
		DataDir:    dataDir,
	}

	attachToL2 := false
	if cfg.AttachToL2Network {
		exists, err := dockerNetworkExists(ctx, l2NetworkName)
		if err != nil {
			return preparedRuntime{}, fmt.Errorf("failed to inspect docker network %q: %w", l2NetworkName, err)
		}
		if exists {
			attachToL2 = true
		} else {
			slog.With("network", l2NetworkName).Warn("localnet L2 network not found, op-alt-da will not be attached to l2 network")
		}
	}

	indexerRepoHostPath := ""
	interfaceRepoHostPath := ""
	if cfg.CeleniumEnabled {
		servicesDir := filepath.Join(rootDir, ".localnet", "services")

		indexerRepo, err := prepareRepository(
			ctx,
			rootDir,
			servicesDir,
			cfg.CeleniumIndexer,
			defaultCeleniumIndexerURL,
			defaultCeleniumIndexerRef,
			"celenium-indexer",
		)
		if err != nil {
			return preparedRuntime{}, err
		}

		interfaceRepo, err := prepareRepository(
			ctx,
			rootDir,
			servicesDir,
			cfg.CeleniumInterface,
			defaultCeleniumInterfaceURL,
			defaultCeleniumInterfaceRef,
			"celenium-interface",
		)
		if err != nil {
			return preparedRuntime{}, err
		}

		indexerRepoHostPath = indexerRepo.HostPath
		interfaceRepoHostPath = interfaceRepo.HostPath
	}

	composeData := composeTemplateData{
		ProjectName:       cfg.ProjectName,
		ChainID:           cfg.ChainID,
		AttachToL2Network: attachToL2,
		CeleniumEnabled:   cfg.CeleniumEnabled,
		Images: composeImages{
			CelestiaApp:  cfg.Images.CelestiaApp,
			CelestiaNode: cfg.Images.CelestiaNode,
			OpAltDA:      cfg.Images.OpAltDA,
			CeleniumDB:   cfg.Images.CeleniumDB,
		},
		DataDir:               dataDirHost,
		ConfigsDir:            filepath.Join(runtimeDirHost, "configs"),
		ScriptsDir:            filepath.Join(runtimeDirHost, "scripts"),
		CeleniumIndexerPath:   indexerRepoHostPath,
		CeleniumInterfacePath: interfaceRepoHostPath,
	}

	opAltDAData := opAltDAConfigData{
		Namespace: defaultNamespace,
		ChainID:   cfg.ChainID,
	}

	celeniumData := celeniumEnvData{
		IndexerStartLevel: cfg.CeleniumIndexerStartHeight,
		ChainID:           cfg.ChainID,
	}

	composePath, err := ensureRuntimeAssets(layout, composeData, opAltDAData, celeniumData)
	if err != nil {
		return preparedRuntime{}, err
	}

	return preparedRuntime{
		ProjectName: cfg.ProjectName,
		ComposePath: composePath,
		RuntimeDir:  runtimeDir,
		DataDir:     dataDir,
	}, nil
}

func composeRun(ctx context.Context, composeFilePath, projectName string, args ...string) error {
	fullArgs := append([]string{"compose", "-f", composeFilePath}, args...)
	cmd := exec.CommandContext(ctx, "docker", fullArgs...)
	cmd.Dir = filepath.Dir(composeFilePath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", projectName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker compose %s failed: %w", strings.Join(args, " "), err)
	}
	return nil
}

func applyDefaults(cfg *configs.Celestia) {
	if cfg.ProjectName == "" {
		cfg.ProjectName = defaultProjectName
	}
	if cfg.RuntimeDir == "" {
		cfg.RuntimeDir = defaultRuntimeDir
	}
	if cfg.DataDir == "" {
		cfg.DataDir = defaultDataDir
	}
	if cfg.ChainID == "" {
		cfg.ChainID = defaultChainID
	}

	if cfg.Images.CelestiaApp == "" {
		cfg.Images.CelestiaApp = defaultCelestiaAppImage
	}
	if cfg.Images.CelestiaNode == "" {
		cfg.Images.CelestiaNode = defaultCelestiaNodeImage
	}
	if cfg.Images.OpAltDA == "" {
		cfg.Images.OpAltDA = defaultOpAltDAImage
	}
	if cfg.Images.CeleniumDB == "" {
		cfg.Images.CeleniumDB = defaultCeleniumDBImage
	}

	if cfg.CeleniumIndexerStartHeight == 0 {
		cfg.CeleniumIndexerStartHeight = defaultCeleniumIndexerStartHeight
	}

	if cfg.CeleniumEnabled && cfg.CeleniumIndexer.URL == "" && cfg.CeleniumIndexer.LocalPath == "" {
		cfg.CeleniumIndexer.URL = defaultCeleniumIndexerURL
		cfg.CeleniumIndexer.Branch = defaultCeleniumIndexerRef
	}
	if cfg.CeleniumEnabled && cfg.CeleniumInterface.URL == "" && cfg.CeleniumInterface.LocalPath == "" {
		cfg.CeleniumInterface.URL = defaultCeleniumInterfaceURL
		cfg.CeleniumInterface.Branch = defaultCeleniumInterfaceRef
	}
}

func resolvePath(rootDir, pathValue string) (string, error) {
	if filepath.IsAbs(pathValue) {
		return filepath.Clean(pathValue), nil
	}
	return filepath.Clean(filepath.Join(rootDir, pathValue)), nil
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

func dockerNetworkExists(ctx context.Context, networkName string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "network", "inspect", networkName)
	cmd.Stdout = nil
	cmd.Stderr = nil
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() != 0 {
		return false, nil
	}
	return false, err
}

func toHostPath(containerPath string) (string, error) {
	hostProjectPath := os.Getenv("HOST_PROJECT_PATH")
	if hostProjectPath == "" {
		return filepath.Abs(containerPath)
	}

	absPath, err := filepath.Abs(containerPath)
	if err != nil {
		return "", err
	}

	if after, ok := strings.CutPrefix(absPath, "/workspace/"); ok {
		return filepath.Join(hostProjectPath, after), nil
	}
	if absPath == "/workspace" {
		return hostProjectPath, nil
	}

	return absPath, nil
}
