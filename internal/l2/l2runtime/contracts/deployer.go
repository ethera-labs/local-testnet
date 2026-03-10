package contracts

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/logger"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type (
	// Deployer deploys L2 contracts
	Deployer struct {
		networksDir                   string
		waitForDeploymentConfirmation bool
		logger                        *slog.Logger
	}
)

// NewDeployer creates a new contract deployer
func NewDeployer(networksDir string) *Deployer {
	return &Deployer{
		networksDir:                   networksDir,
		waitForDeploymentConfirmation: true,
		logger:                        logger.Named("contracts_deployer"),
	}
}

// Deploy deploys L2 contracts and returns the deployed addresses
func (d *Deployer) Deploy(ctx context.Context, chainConfigs map[configs.L2ChainName]configs.Chain, coordinatorPK string) (map[configs.L2ChainName]map[ContractName]common.Address, error) {
	d.logger.Info("deploying L2 contracts")

	deployments, err := d.deployContracts(ctx, chainConfigs, coordinatorPK)
	if err != nil {
		d.logger.With("err", err.Error()).Error("contract deployment failed or timed out")
		return nil, err
	}

	d.logger.Info("L2 contracts deployed successfully")

	return deployments, nil
}

// deployContracts deploys contracts to rollups using go-ethereum.
func (d *Deployer) deployContracts(ctx context.Context, chainConfigs map[configs.L2ChainName]configs.Chain, coordinatorPK string) (map[configs.L2ChainName]map[ContractName]common.Address, error) {
	d.logger.Info("loading precompiled contracts")
	compiledContracts, err := LoadCompiledContracts()
	if err != nil {
		return nil, fmt.Errorf("failed to load compiled contracts: %w", err)
	}

	d.logger.With("len", len(compiledContracts)).Info("precompiled contracts loaded")

	deployments := make(map[configs.L2ChainName]map[ContractName]common.Address)
	for chainName, chainConfig := range chainConfigs {
		// When running in Docker, use host.docker.internal to access host services
		// Otherwise use localhost for native execution
		hostname := "localhost"
		if os.Getenv("HOST_PROJECT_PATH") != "" {
			hostname = "host.docker.internal"
		}
		url := fmt.Sprintf("http://%s:%d", hostname, chainConfig.RPCPort)
		d.logger.With("chain_name", chainName).With("url", url).Info("waiting for rollup RPC")
		if err := waitForRPC(ctx, url); err != nil {
			return nil, err
		}

		d.logger.With("chain_name", chainName).Info("waiting for block production")
		if err := waitForBlockProduction(ctx, url, d.logger); err != nil {
			return nil, fmt.Errorf("block production not started for %s: %w", chainName, err)
		}

		d.logger.Info("deploying contracts to L2")
		addressStrings, err := d.deployToChain(ctx, url, coordinatorPK, compiledContracts)
		if err != nil {
			return nil, fmt.Errorf("failed to deploy to %s: %w", chainName, err)
		}

		// Convert string addresses to common.Address
		addressMap := make(map[ContractName]common.Address)
		for contractName, addrStr := range addressStrings {
			addressMap[contractName] = common.HexToAddress(addrStr)
		}
		deployments[chainName] = addressMap
	}

	if !addressesMatchAcrossChains(deployments) {
		return nil, fmt.Errorf("contract addresses differ between rollups")
	}

	for chainName, addresses := range deployments {
		addressStrings := make(map[ContractName]string)
		for contractName, addr := range addresses {
			addressStrings[contractName] = addr.Hex()
		}

		directory := filepath.Join(d.networksDir, string(chainName))
		if err := writeContractJSON(filepath.Join(directory, contractsFileName), addressStrings, uint64(chainConfigs[chainName].ID)); err != nil {
			return nil, fmt.Errorf("failed to write %s for %s: %w", contractsFileName, chainName, err)
		}
	}

	d.logger.Info("contracts deployed successfully")

	return deployments, nil
}

func waitForRPC(ctx context.Context, url string) error {
	for range 120 {
		client, err := ethclient.DialContext(ctx, url)
		if err == nil {
			_, err := client.BlockNumber(ctx)
			client.Close()
			if err == nil {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	return fmt.Errorf("timed out waiting for RPC at %s", url)
}

// waitForBlockProduction waits until block number increases, ensuring L2 is producing blocks.
func waitForBlockProduction(ctx context.Context, url string, logger *slog.Logger) error {
	client, err := ethclient.DialContext(ctx, url)
	if err != nil {
		return fmt.Errorf("failed to connect to RPC: %w", err)
	}
	defer client.Close()

	initialBlock, err := client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial block number: %w", err)
	}

	logger.With("initial_block", initialBlock, "url", url).Info("waiting for block production to start")

	// Wait up to 5 minutes for block production to start
	const maxWaitTime = 5 * time.Minute
	const pollInterval = 2 * time.Second

	deadline := time.Now().Add(maxWaitTime)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			currentBlock, err := client.BlockNumber(ctx)
			if err != nil {
				logger.With("err", err.Error()).Warn("failed to get block number, retrying")
				continue
			}

			if currentBlock > initialBlock {
				logger.With("initial_block", initialBlock, "current_block", currentBlock).
					Info("block production confirmed")
				return nil
			}
		}
	}

	return fmt.Errorf("timed out waiting for block production at %s (stuck at block %d)", url, initialBlock)
}

func (d *Deployer) deployToChain(ctx context.Context, rpcURL, coordinatorPrivateKey string, contracts map[ContractName]CompiledContract) (map[ContractName]string, error) {
	d.logger.With("url", rpcURL).Info("dialing the L2 RPC")
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", rpcURL, err)
	}
	defer client.Close()

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(coordinatorPrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	d.logger.Info("fetching chain ID")
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}
	d.logger.With("chain_id", chainID).Info("chain ID was fetched")

	addresses := make(map[ContractName]string)

	d.logger.Info("deploying contracts")

	coordinatorPubKey, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}

	mailboxAddr, err := d.deployContract(ctx, client, privateKey, chainID, contracts[ContractNameMailbox], crypto.PubkeyToAddress(*coordinatorPubKey))
	if err != nil {
		return nil, fmt.Errorf("failed to deploy Mailbox: %w", err)
	}
	addresses[ContractNameMailbox] = mailboxAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameMailbox, "address", mailboxAddr.Hex())

	pingPongAddr, err := d.deployContract(ctx, client, privateKey, chainID, contracts[ContractNamePingPong], mailboxAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy PingPong: %w", err)
	}
	addresses[ContractNamePingPong] = pingPongAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNamePingPong, "address", pingPongAddr.Hex())

	bridgeAddr, err := d.deployContract(ctx, client, privateKey, chainID, contracts[ContractNameBridge], mailboxAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy Bridge: %w", err)
	}
	addresses[ContractNameBridge] = bridgeAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameBridge, "address", bridgeAddr.Hex())

	tokenAddr, err := d.deployContract(ctx, client, privateKey, chainID, contracts[ContractNameBridgeableToken], bridgeAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy BridgeableToken: %w", err)
	}
	addresses[ContractNameBridgeableToken] = tokenAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameBridgeableToken, "address", tokenAddr.Hex())

	stagedMailboxAddr, err := d.deployContract(ctx, client, privateKey, chainID, contracts[ContractNameStagedMailbox], crypto.PubkeyToAddress(*coordinatorPubKey))
	if err != nil {
		return nil, fmt.Errorf("failed to deploy StagedMailbox: %w", err)
	}
	addresses[ContractNameStagedMailbox] = stagedMailboxAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameStagedMailbox, "address", stagedMailboxAddr.Hex())

	return addresses, nil
}

func (d *Deployer) deployContract(ctx context.Context, client *ethclient.Client, privateKey *ecdsa.PrivateKey, chainID *big.Int, contract CompiledContract, constructorArgs ...any) (common.Address, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*3)
	defer cancel()

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to create transactor: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to get gas price: %w", err)
	}

	auth.Context = ctx
	auth.GasLimit = uint64(10_000_000)
	auth.GasPrice = gasPrice

	address, tx, _, err := bind.DeployContract(auth, contract.ABI, contract.Bytecode, client, constructorArgs...)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to deploy contract: %w", err)
	}

	d.logger.
		With("address", address).
		With("tx_hash", tx.Hash().Hex()).
		Info("contract deployment transaction sent")

	if d.waitForDeploymentConfirmation {
		receipt, err := bind.WaitMined(ctx, client, tx)
		if err != nil {
			return common.Address{}, fmt.Errorf("failed to wait for transaction: %w", err)
		}

		if receipt.Status != types.ReceiptStatusSuccessful {
			return common.Address{}, fmt.Errorf("contract deployment failed with status %d", receipt.Status)
		}
	}

	return address, nil
}

func writeContractJSON(path string, addresses map[ContractName]string, chainID uint64) error {
	payload := map[string]any{
		"chainInfo": map[string]any{
			"chainId": chainID,
		},
		"addresses": addresses,
	}

	if err := writeJSON(path, payload); err != nil {
		return err
	}

	return nil
}

func writeJSON(path string, data any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", path, err)
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for %s: %w", path, err)
	}

	if err := os.WriteFile(path, append(content, '\n'), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", path, err)
	}

	return nil
}

// addressesMatchAcrossChains verifies that all chains deployed the same contracts at the same addresses
func addressesMatchAcrossChains(deployments map[configs.L2ChainName]map[ContractName]common.Address) bool {
	if len(deployments) < 2 {
		return true
	}

	var firstChain configs.L2ChainName
	var firstDeployment map[ContractName]common.Address
	for chainName, deployment := range deployments {
		firstChain = chainName
		firstDeployment = deployment
		break
	}

	for chainName, deployment := range deployments {
		if chainName == firstChain {
			continue
		}

		if len(firstDeployment) != len(deployment) {
			return false
		}

		for contractName, firstAddr := range firstDeployment {
			otherAddr, ok := deployment[contractName]
			if !ok {
				return false
			}
			if firstAddr != otherAddr {
				return false
			}
		}
	}

	return true
}
