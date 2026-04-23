package contracts

import "github.com/ethereum/go-ethereum/accounts/abi"

const contractsFileName = "contracts.json"

type (
	ContractName     string
	CompiledContract struct {
		ABI      abi.ABI
		RawABI   string
		Bytecode []byte
	}
)

// Contract names match the Solidity contract identifiers so `forge inspect`
// resolves them directly. Variable names kept generic (Mailbox, Bridge, Token)
// so orchestrator call sites read naturally — the Solidity types they map to
// are the Compose L2↔L2 stack (UniversalBridgeMailbox + ComposeL2ToL2Bridge).
const (
	ContractNameMailbox         ContractName = "UniversalBridgeMailbox"
	ContractNameBridge          ContractName = "ComposeL2ToL2Bridge"
	ContractNameCETFactory      ContractName = "CetFactory"
	ContractNameETHLiquidity    ContractName = "ComposeETHLiquidity"
	ContractNameBridgeableToken ContractName = "USDCMintable"
)

var Contracts = map[ContractName]struct{}{
	ContractNameMailbox:         {},
	ContractNameBridge:          {},
	ContractNameCETFactory:      {},
	ContractNameETHLiquidity:    {},
	ContractNameBridgeableToken: {},
}
