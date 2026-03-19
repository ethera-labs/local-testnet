package deployer

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/docker"
	fsjson "github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem/json"
	"github.com/ethera-labs/local-testnet/internal/l2/path"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

const (
	publicImageName      = "us-docker.pkg.dev/oplabs-tools-artifacts/images/op-deployer"
	kurtosisL1Network    = "kt-localnet"
	kurtosisL1ELPort     = 8545
	kurtosisL1ELNameHint = "el-1-geth-lighthouse--"
)

// Deployer wraps the op-deployer tool
type Deployer struct {
	rootDir         string
	stateDir        string
	imageWithTag    string
	imageEntrypoint string
	docker          *docker.Client
	logger          *slog.Logger
}

// NewDeployer creates a new op-deployer wrapper
// imageTag should be the version tag (e.g., "v0.3.3")
func NewDeployer(rootDir, stateDir, imageTag string, dockerClient *docker.Client) *Deployer {
	return &Deployer{
		rootDir:         rootDir,
		stateDir:        stateDir,
		imageWithTag:    fmt.Sprintf("%s:%s", publicImageName, imageTag),
		imageEntrypoint: "/usr/local/bin/op-deployer",
		docker:          dockerClient,
		logger:          logger.Named("deployer"),
	}
}

// Init initializes the op-deployer state
func (o *Deployer) Init(ctx context.Context, l1ChainID int, l2Chains map[configs.L2ChainName]configs.Chain) error {
	o.logger.
		With("state_dir", o.stateDir).
		Info("initializing deployer state. Ensuring image exists")

	if err := o.ensureImage(ctx); err != nil {
		return fmt.Errorf("failed to ensure op-deployer image: %w", err)
	}

	stateFile := filepath.Join(o.stateDir, stateFile)
	if _, err := os.Stat(stateFile); err == nil {
		o.logger.
			With("file_name", stateFile).
			Info("state already exists, skipping init")

		return nil
	}

	// When running in Docker, we need to use the host's path for volume mounts
	// Otherwise Docker daemon won't recognize the path
	absStateDir, err := path.GetHostPath(o.stateDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	var chainIDsStr []string
	for _, chainConfig := range l2Chains {
		chainIDsStr = append(chainIDsStr, fmt.Sprintf("%d", chainConfig.ID))
	}

	o.logger.Info("running docker image")
	_, err = o.docker.Run(ctx, docker.RunOptions{
		Image:      o.imageWithTag,
		Entrypoint: []string{o.imageEntrypoint},
		Cmd: []string{
			"init",
			"--intent-type", "custom",
			"--l1-chain-id", strconv.Itoa(l1ChainID),
			"--l2-chain-ids", strings.Join(chainIDsStr, ","),
		},
		Env: []string{
			"HOME=/work",
			"DEPLOYER_CACHE_DIR=/work/.cache",
		},
		Volumes: map[string]string{
			absStateDir: "/work",
		},
		WorkDir:    "/work",
		User:       fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		AutoRemove: true,
		StreamLogs: true,
	})

	if err != nil {
		return fmt.Errorf("failed to run op-deployer init: %w", err)
	}

	o.logger.Info("deployer state initialized successfully")

	return nil
}

// EnsureImage ensures the op-deployer public image exists
func (o *Deployer) ensureImage(ctx context.Context) error {
	logger := o.logger.With("image", o.imageWithTag)

	exists, err := o.docker.ImageExists(ctx, o.imageWithTag)
	if err != nil {
		return fmt.Errorf("failed to check image: '%s' existence: %w", o.imageWithTag, err)
	}

	if exists {
		logger.Info("image already exists")
		return nil
	}

	logger.Info("pulling public image")
	if err := o.docker.PullImage(ctx, o.imageWithTag); err != nil {
		return fmt.Errorf("failed to pull image: '%s', %w", o.imageWithTag, err)
	}

	logger.Info("image pulled successfully")
	return nil
}

// Apply runs op-deployer apply to deploy L1 contracts
func (o *Deployer) Apply(ctx context.Context, l1RpcURL, deployerPrivateKey, deploymentTarget string) error {
	o.logger.
		With("deployment_target", deploymentTarget).
		Info("running deployer apply")

	absStateDir, err := path.GetHostPath(o.stateDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	containerL1RPCURL, network, err := o.containerAccessibleL1RPCURL(ctx, l1RpcURL)
	if err != nil {
		return fmt.Errorf("failed to resolve container-accessible L1 RPC URL: %w", err)
	}

	_, err = o.docker.Run(ctx, docker.RunOptions{
		Image:      o.imageWithTag,
		Entrypoint: []string{o.imageEntrypoint},
		Cmd: []string{
			"apply",
			"--deployment-target", deploymentTarget,
		},
		Env: []string{
			"HOME=/work",
			"DEPLOYER_CACHE_DIR=/work/.cache",
			fmt.Sprintf("L1_RPC_URL=%s", containerL1RPCURL),
			fmt.Sprintf("DEPLOYER_PRIVATE_KEY=%s", deployerPrivateKey),
		},
		Network: network,
		Volumes: map[string]string{
			absStateDir: "/work",
		},
		WorkDir:    "/work",
		User:       fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		AutoRemove: true,
		StreamLogs: true,
	})

	if err != nil {
		return o.decorateApplyError(err)
	}

	o.logger.Info("deployer apply completed successfully")

	return nil
}

func (o *Deployer) decorateApplyError(runErr error) error {
	if !looksLikeAlreadyDeployedError(runErr) {
		return fmt.Errorf("failed to run op-deployer apply: %w", runErr)
	}

	const recoveryHint = "If this is the local Kurtosis L1, run `make clean-l1 && make clean-l2` before retrying. Otherwise, retry against a fresh L1."

	stateManager := NewStateManager(o.stateDir, fsjson.NewReader())
	state, err := stateManager.Load()
	if err == nil && state.HasIncompleteProgress() {
		return fmt.Errorf(
			"failed to run op-deployer apply: detected partially applied L1 deployment state in %s. Some contracts were already deployed on L1, but the local op-deployer state never reached a full apply, so the retry cannot resume safely. %s: %w",
			filepath.Join(o.stateDir, stateFile),
			recoveryHint,
			runErr,
		)
	}

	return fmt.Errorf(
		"failed to run op-deployer apply: op-deployer tried to deploy a contract to an address that already has code. %s: %w",
		recoveryHint,
		runErr,
	)
}

func looksLikeAlreadyDeployedError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "contract already deployed")
}

func (o *Deployer) containerAccessibleL1RPCURL(ctx context.Context, raw string) (string, string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw, "", nil
	}
	if parsed.Hostname() != "host.docker.internal" {
		return raw, "", nil
	}
	if parsed.Port() != "63816" {
		return raw, "", nil
	}

	containerName, err := o.docker.FindContainerByNamePrefix(ctx, kurtosisL1ELNameHint)
	if err != nil {
		return "", "", err
	}
	ip, err := o.docker.ContainerNetworkIPv4(ctx, containerName, kurtosisL1Network)
	if err != nil {
		return "", "", err
	}
	return fmt.Sprintf("http://%s:%d", ip, kurtosisL1ELPort), kurtosisL1Network, nil
}

// InspectGenesis exports genesis JSON for a chain
func (o *Deployer) InspectGenesis(ctx context.Context, chainID int) (string, error) {
	absStateDir, err := path.GetHostPath(o.stateDir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	output, err := o.docker.Run(ctx, docker.RunOptions{
		Image:      o.imageWithTag,
		Entrypoint: []string{o.imageEntrypoint},
		Cmd: []string{
			"inspect",
			"genesis",
			fmt.Sprintf("%d", chainID),
		},
		Env: []string{
			"HOME=/work",
		},
		Volumes: map[string]string{
			absStateDir: "/work",
		},
		WorkDir:    "/work",
		User:       fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		AutoRemove: true,
		CaptureOut: true,
	})

	if err != nil {
		return "", fmt.Errorf("failed to run op-deployer inspect genesis: %w", err)
	}

	return output, nil
}

// InspectRollup exports rollup config for a chain
func (o *Deployer) InspectRollup(ctx context.Context, chainID int, outputPath string) error {
	absStateDir, err := path.GetHostPath(o.stateDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	outputDir := filepath.Dir(outputPath)
	absOutputDir, err := path.GetHostPath(outputDir)
	if err != nil {
		return fmt.Errorf("failed to get host path for output dir: %w", err)
	}
	outputFile := filepath.Base(outputPath)

	_, err = o.docker.Run(ctx, docker.RunOptions{
		Image:      o.imageWithTag,
		Entrypoint: []string{o.imageEntrypoint},
		Cmd: []string{
			"inspect",
			"rollup",
			"--outfile", fmt.Sprintf("/output/%s", outputFile),
			fmt.Sprintf("%d", chainID),
		},
		Env: []string{
			"HOME=/work",
		},
		Volumes: map[string]string{
			absStateDir:  "/work",
			absOutputDir: "/output",
		},
		WorkDir:    "/work",
		User:       fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		AutoRemove: true,
	})

	if err != nil {
		return fmt.Errorf("failed to run op-deployer inspect rollup: %w", err)
	}

	return nil
}
