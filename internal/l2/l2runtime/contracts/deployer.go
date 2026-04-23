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
	"github.com/ethereum/go-ethereum/accounts/abi"
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
		testWalletAddress             common.Address
		waitForDeploymentConfirmation bool
		logger                        *slog.Logger
	}
)

// NewDeployer creates a new contract deployer. testWalletAddress, when
// non-zero, receives an initial mint of the test ERC-20 used by xbridge.
func NewDeployer(networksDir string, testWalletAddress common.Address) *Deployer {
	return &Deployer{
		networksDir:                   networksDir,
		testWalletAddress:             testWalletAddress,
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

func (d *Deployer) deployToChain(ctx context.Context, rpcURL, coordinatorPrivateKey string, contractSet map[ContractName]CompiledContract) (map[ContractName]string, error) {
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

	coordinatorPubKey, ok := privateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")
	}
	coordinatorAddr := crypto.PubkeyToAddress(*coordinatorPubKey)

	addresses := make(map[ContractName]string)
	d.logger.Info("deploying Compose L2↔L2 bridge stack")

	// 1. CetFactory — no ctor args. Must be first because the bridge ctor
	//    takes a reference to it.
	cetFactoryAddr, err := d.deployContract(ctx, client, privateKey, chainID, contractSet[ContractNameCETFactory])
	if err != nil {
		return nil, fmt.Errorf("failed to deploy CetFactory: %w", err)
	}
	addresses[ContractNameCETFactory] = cetFactoryAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameCETFactory, "address", cetFactoryAddr.Hex())

	// 2. UniversalBridgeMailbox(coordinator). Coordinator is the off-chain
	//    relayer — same key as the deployer for local-testnet.
	mailboxAddr, err := d.deployContract(ctx, client, privateKey, chainID, contractSet[ContractNameMailbox], coordinatorAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy UniversalBridgeMailbox: %w", err)
	}
	addresses[ContractNameMailbox] = mailboxAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameMailbox, "address", mailboxAddr.Hex())

	// 3. ComposeETHLiquidity(owner). Owner is the deployer so we can wire
	//    authorizations below.
	ethLiquidityAddr, err := d.deployContract(ctx, client, privateKey, chainID, contractSet[ContractNameETHLiquidity], coordinatorAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy ComposeETHLiquidity: %w", err)
	}
	addresses[ContractNameETHLiquidity] = ethLiquidityAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameETHLiquidity, "address", ethLiquidityAddr.Hex())

	// 4. ComposeL2ToL2Bridge(mailbox, factory, ethLiquidity).
	bridgeAddr, err := d.deployContract(ctx, client, privateKey, chainID, contractSet[ContractNameBridge], mailboxAddr, cetFactoryAddr, ethLiquidityAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy ComposeL2ToL2Bridge: %w", err)
	}
	addresses[ContractNameBridge] = bridgeAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameBridge, "address", bridgeAddr.Hex())

	// 5. USDCMintable — the test token bridged by xbridge-client.
	tokenAddr, err := d.deployContract(ctx, client, privateKey, chainID, contractSet[ContractNameBridgeableToken])
	if err != nil {
		return nil, fmt.Errorf("failed to deploy USDCMintable: %w", err)
	}
	addresses[ContractNameBridgeableToken] = tokenAddr.Hex()
	d.logger.Info("deployed", "contract", ContractNameBridgeableToken, "address", tokenAddr.Hex())

	// 6. Authorize the bridge on mailbox / factory / ethLiquidity so it can
	//    write/read messages and mint CET wrappers.
	if err := d.wireBridgeAuthorizations(ctx, client, privateKey, chainID, contractSet, mailboxAddr, cetFactoryAddr, ethLiquidityAddr, bridgeAddr); err != nil {
		return nil, fmt.Errorf("failed to wire bridge authorizations: %w", err)
	}

	// 7. Seed the test wallet with USDCMintable tokens + approval for the bridge
	//    so xbridge-client can call bridgeERC20To immediately.
	walletAddr, walletAmount, mintEnabled := d.testWalletSeed()
	if mintEnabled {
		mintInput, err := contractSet[ContractNameBridgeableToken].ABI.Pack("mint", walletAddr, walletAmount)
		if err != nil {
			return nil, fmt.Errorf("pack mint: %w", err)
		}
		if err := d.sendTx(ctx, client, privateKey, chainID, tokenAddr, mintInput); err != nil {
			return nil, fmt.Errorf("mint USDCMintable to test wallet: %w", err)
		}
		d.logger.Info("minted test token balance", "token", tokenAddr.Hex(), "to", walletAddr.Hex(), "amount", walletAmount.String())
	}

	return addresses, nil
}

// testWalletSeed returns the L2 test wallet address and an initial USDC mint
// amount. When no wallet address was configured, the seed step is skipped.
func (d *Deployer) testWalletSeed() (common.Address, *big.Int, bool) {
	if (d.testWalletAddress == common.Address{}) {
		return common.Address{}, nil, false
	}
	// 1_000_000 * 10^18 — generous supply for stress tests.
	amount := new(big.Int).Mul(big.NewInt(1_000_000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
	return d.testWalletAddress, amount, true
}

// sendTx builds, signs and submits a transaction to `to` with `data`, waits
// for inclusion, and surfaces revert receipts as errors.
func (d *Deployer) sendTx(
	ctx context.Context,
	client *ethclient.Client,
	privateKey *ecdsa.PrivateKey,
	chainID *big.Int,
	to common.Address,
	data []byte,
) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*3)
	defer cancel()

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return fmt.Errorf("transactor: %w", err)
	}
	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return fmt.Errorf("suggest gas: %w", err)
	}
	nonce, err := client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		return fmt.Errorf("nonce: %w", err)
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    big.NewInt(0),
		Gas:      uint64(1_000_000),
		GasPrice: gasPrice,
		Data:     data,
	})
	signed, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	if err := client.SendTransaction(ctx, signed); err != nil {
		return fmt.Errorf("send: %w", err)
	}

	if !d.waitForDeploymentConfirmation {
		return nil
	}
	receipt, err := bind.WaitMined(ctx, client, signed)
	if err != nil {
		return fmt.Errorf("wait mined: %w", err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("tx reverted (hash=%s)", signed.Hash().Hex())
	}
	return nil
}

// wireBridgeAuthorizations calls authorizeBridge on the three gated contracts
// so the newly deployed ComposeL2ToL2Bridge can write to the mailbox, deploy
// CETs, and lock/unlock ETH. Mirrors contracts/L2/script/bridge/DeployComposeBridge.s.sol.
func (d *Deployer) wireBridgeAuthorizations(
	ctx context.Context,
	client *ethclient.Client,
	privateKey *ecdsa.PrivateKey,
	chainID *big.Int,
	contractSet map[ContractName]CompiledContract,
	mailbox, factory, ethLiquidity, bridge common.Address,
) error {
	for _, step := range []struct {
		name     string
		target   common.Address
		abi      abi.ABI
		method   string
		argument common.Address
	}{
		{"mailbox.authorizeBridge", mailbox, contractSet[ContractNameMailbox].ABI, "authorizeBridge", bridge},
		{"cetFactory.authorizeBridge", factory, contractSet[ContractNameCETFactory].ABI, "authorizeBridge", bridge},
		{"ethLiquidity.authorizeBridge", ethLiquidity, contractSet[ContractNameETHLiquidity].ABI, "authorizeBridge", bridge},
	} {
		input, err := step.abi.Pack(step.method, step.argument)
		if err != nil {
			return fmt.Errorf("pack %s: %w", step.name, err)
		}
		if err := d.sendTx(ctx, client, privateKey, chainID, step.target, input); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
		d.logger.Info("authorized bridge", "on", step.name, "bridge", bridge.Hex())
	}
	return nil
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
