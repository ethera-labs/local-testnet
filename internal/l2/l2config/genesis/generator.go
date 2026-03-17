package genesis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"

	"github.com/ethera-labs/local-testnet/internal/l2/infra/docker"
	"github.com/ethera-labs/local-testnet/internal/l2/infra/filesystem"
	"github.com/ethera-labs/local-testnet/internal/l2/path"
	"github.com/ethera-labs/local-testnet/internal/logger"
)

const GenesisFileName = "genesis.json"

// Generator generates genesis.json files for L2 chains
type (
	deployer interface {
		InspectGenesis(ctx context.Context, chainID int) (string, error)
	}

	Generator struct {
		deployer    deployer
		docker      *docker.Client
		writer      filesystem.Writer
		rootDir     string
		localnetDir string
		servicesDir string
		networksDir string
		opGethPath  string
		logger      *slog.Logger
	}
)

// NewGenerator creates a new genesis generator
func NewGenerator(deployer deployer, docker *docker.Client, writer filesystem.Writer, rootDir, localnetDir, servicesDir, networksDir, opGethPath string) *Generator {
	return &Generator{
		deployer:    deployer,
		docker:      docker,
		writer:      writer,
		rootDir:     rootDir,
		localnetDir: localnetDir,
		servicesDir: servicesDir,
		networksDir: networksDir,
		opGethPath:  opGethPath,
		logger:      logger.Named("genesis_generator"),
	}
}

// Generate generates genesis config for a chain
func (g *Generator) Generate(ctx context.Context, chainID int, path string, walletAddress, sequencerAddress, genesisBalanceWei, coordinatorPrivateKey string) (string, error) {
	logger := g.logger.With("chain_id", chainID)

	logger.Info("inspecting genesis")
	output, err := g.deployer.InspectGenesis(ctx, chainID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect genesis: %w", err)
	}

	logger.With("output_len", len(output)).Info("unmarshalling output")
	var genesis map[string]any
	if err := json.Unmarshal([]byte(output), &genesis); err != nil {
		return "", fmt.Errorf("failed to parse genesis JSON: %w", err)
	}

	alloc, ok := genesis["alloc"].(map[string]any)
	if !ok {
		alloc = make(map[string]any)
		genesis["alloc"] = alloc
	}

	balanceWei, success := new(big.Int).SetString(genesisBalanceWei, 10)
	if !success {
		return "", fmt.Errorf("invalid genesis balance: %s", genesisBalanceWei)
	}

	for _, addr := range []string{walletAddress, sequencerAddress} {
		if addr == "" {
			return "", fmt.Errorf("wallet or sequencer address cannot be empty")
		}

		cleanAddr := addr
		if len(cleanAddr) > 2 && cleanAddr[:2] == "0x" {
			cleanAddr = cleanAddr[2:]
		}
		cleanAddr = "0x" + cleanAddr

		accountData, ok := alloc[cleanAddr].(map[string]any)
		if !ok {
			accountData = make(map[string]any)
			alloc[cleanAddr] = accountData
		}
		accountData["balance"] = fmt.Sprintf("0x%x", balanceWei)
	}

	config, ok := genesis["config"].(map[string]any)
	if !ok {
		config = make(map[string]any)
		genesis["config"] = config
	}
	config["pragueTime"] = 0
	config["isthmusTime"] = 0

	g.logger.Info("computing genesis hash")
	hash, err := g.computeGenesisHash(ctx, chainID, genesis, coordinatorPrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to compute genesis hash: %w", err)
	}

	genesisPath := filepath.Join(path, GenesisFileName)

	g.logger.
		With("hash", hash).
		With("file_path", genesisPath).
		Info("genesis generated successfully. Persisting file")
	if err := g.writer.WriteJSON(genesisPath, genesis); err != nil {
		return "", fmt.Errorf("failed to write genesis.json for chain %d: %w", chainID, err)
	}

	return hash, nil
}

// computeGenesisHash computes the genesis block hash
func (g *Generator) computeGenesisHash(ctx context.Context, chainID int, genesis map[string]any, coordinatorPrivateKey string) (string, error) {
	// Create temp directories under .localnet/.tmp/ to make them accessible when running in Docker
	tmpBaseDir := filepath.Join(g.localnetDir, ".tmp")
	if err := os.MkdirAll(tmpBaseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temp base dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp(tmpBaseDir, "genesis-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	genesisPath := filepath.Join(tmpDir, GenesisFileName)
	genesisJSON, err := json.Marshal(genesis)
	if err != nil {
		return "", fmt.Errorf("failed to marshal genesis: %w", err)
	}

	if err := os.WriteFile(genesisPath, genesisJSON, 0644); err != nil {
		return "", fmt.Errorf("failed to write genesis file: %w", err)
	}

	tmpDataDir, err := os.MkdirTemp(tmpBaseDir, "geth-init-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp datadir: %w", err)
	}
	defer os.RemoveAll(tmpDataDir)

	const opGethImage = "local/op-geth:dev"

	if err := g.ensureOpGethImage(ctx, opGethImage); err != nil {
		return "", fmt.Errorf("failed to ensure op-geth image: %w", err)
	}

	hostTmpDir, err := path.GetHostPath(tmpDir)
	if err != nil {
		return "", fmt.Errorf("failed to get host path for tmpDir: %w", err)
	}

	hostTmpDataDir, err := path.GetHostPath(tmpDataDir)
	if err != nil {
		return "", fmt.Errorf("failed to get host path for tmpDataDir: %w", err)
	}

	g.logger.With("image", opGethImage).Info("running geth init")
	_, err = g.docker.Run(ctx, docker.RunOptions{
		Image: opGethImage,
		Cmd: []string{
			fmt.Sprintf("--networkid=%d", chainID),
			"init",
			"--state.scheme=hash",
			"--datadir=/datadir",
			fmt.Sprintf("/genesis/%s", GenesisFileName),
		},
		Env: []string{
			fmt.Sprintf("GETH_COORDINATOR_KEY=%s", coordinatorPrivateKey),
		},
		Volumes: map[string]string{
			hostTmpDir:     "/genesis",
			hostTmpDataDir: "/datadir",
		},
		User:       fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()),
		AutoRemove: true,
		StreamLogs: false,
		CaptureOut: false,
	})

	if err != nil {
		slog.Error("geth init failed", "error", err)
		return "", fmt.Errorf("failed to run geth init: %w", err)
	}

	genesisBlockPath := filepath.Join(tmpDataDir, "geth", "chaindata")
	if _, err := os.Stat(genesisBlockPath); os.IsNotExist(err) {
		return "", fmt.Errorf("genesis database not created")
	}

	var kvStore ethdb.KeyValueStore
	kvStore, err = pebble.New(genesisBlockPath, 16, 16, "", true)
	if err != nil {
		return "", fmt.Errorf("failed to open database: %w", err)
	}
	defer kvStore.Close()

	db, err := rawdb.Open(kvStore, rawdb.OpenOptions{
		Ancient:  filepath.Join(tmpDataDir, "geth", "chaindata", "ancient"),
		ReadOnly: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to open database with freezer: %w", err)
	}
	defer db.Close()

	genesisHash := rawdb.ReadCanonicalHash(db, 0)
	if genesisHash == (common.Hash{}) {
		return "", fmt.Errorf("genesis hash not found in database")
	}

	return genesisHash.Hex(), nil
}

// ensureOpGethImage checks if op-geth image exists, and builds it if not
func (g *Generator) ensureOpGethImage(ctx context.Context, imageName string) error {
	exists, err := g.docker.ImageExists(ctx, imageName)
	if err != nil {
		return fmt.Errorf("failed to check if image exists: %w", err)
	}

	if exists {
		g.logger.Info("op-geth image already exists")
		return nil
	}

	g.logger.Info("op-geth image not found, building it using docker compose")

	dockerPath, err := docker.EnsureDockerFile(g.localnetDir)
	if err != nil {
		return fmt.Errorf("failed to ensure docker file: %w", err)
	}

	g.logger.With("docker_path", dockerPath).Info("docker file created")

	// Environment variables need host paths (for docker daemon to access build contexts)
	// but docker file path stays as container path (docker compose CLI reads it from container)
	rootHostPath, err := path.GetHostPath(g.rootDir)
	if err != nil {
		return fmt.Errorf("failed to resolve host path for root dir: %w", err)
	}

	env := map[string]string{
		"ROOT_DIR":     rootHostPath,
		"OP_GETH_PATH": g.opGethPath,
	}

	g.logger.With("op_geth_path", g.opGethPath, "root_dir", rootHostPath, "docker_file", dockerPath).Info("building op-geth image")

	if err := docker.DockerBuild(ctx, dockerPath, env, "op-geth-a"); err != nil {
		return fmt.Errorf("failed to build op-geth image: %w", err)
	}

	g.logger.Info("op-geth image built successfully")

	return nil
}
