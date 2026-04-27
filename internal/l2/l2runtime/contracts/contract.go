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

const (
	ContractNameUniversalBridgeMailbox = "UniversalBridgeMailbox"
	ContractNameStagedMailbox          = "StagedMailbox"
	ContractNameCetFactory             = "CetFactory"
	ContractNameComposeETHLiquidity    = "ComposeETHLiquidity"
	ContractNameComposeL2ToL2Bridge    = "ComposeL2ToL2Bridge"
	ContractNameTestToken              = "MockL2ERC20"
)

var Contracts = map[ContractName]struct{}{
	ContractNameUniversalBridgeMailbox: {},
	ContractNameStagedMailbox:          {},
	ContractNameCetFactory:             {},
	ContractNameComposeETHLiquidity:    {},
	ContractNameComposeL2ToL2Bridge:    {},
	ContractNameTestToken:              {},
}
