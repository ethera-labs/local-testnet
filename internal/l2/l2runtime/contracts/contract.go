package contracts

import "github.com/ethereum/go-ethereum/accounts/abi"

const (
	contractsFileName  = "contracts.json"
	entryPointFileName = "entrypoint.json"
)

type (
	ContractName     string
	CompiledContract struct {
		ABI              abi.ABI
		RawABI           string
		Bytecode         []byte
		DeployedBytecode []byte // optional; populated for contracts whose runtime code is needed at deploy time.
	}
)

const (
	ContractNameUniversalBridgeMailbox = "UniversalBridgeMailbox"
	ContractNameCetFactory             = "CetFactory"
	ContractNameComposeETHLiquidity    = "ComposeETHLiquidity"
	ContractNameComposeL2ToL2Bridge    = "ComposeL2ToL2Bridge"
	ContractNameTestToken              = "MockL2ERC20"

	// ERC-4337 v0.7 - compiled from the account-abstraction repository,
	// vendored separately in compiled/entrypoint.json.
	ContractNameEntryPoint            ContractName = "EntryPoint"
	ContractNameEntryPointSimulations ContractName = "EntryPointSimulations"
	// SimpleAccount + SimpleAccountFactory are the canonical ERC-4337 reference
	// implementation used by the localnet console for bundler tests.
	ContractNameSimpleAccount        ContractName = "SimpleAccount"
	ContractNameSimpleAccountFactory ContractName = "SimpleAccountFactory"
)

var Contracts = map[ContractName]struct{}{
	ContractNameUniversalBridgeMailbox: {},
	ContractNameCetFactory:             {},
	ContractNameComposeETHLiquidity:    {},
	ContractNameComposeL2ToL2Bridge:    {},
	ContractNameTestToken:              {},
}

// EntryPointContracts is the allowlist for ERC-4337 v0.7 artefacts compiled
// from the eth-infinitism/account-abstraction repository.
var EntryPointContracts = map[ContractName]struct{}{
	ContractNameEntryPoint:            {},
	ContractNameEntryPointSimulations: {},
	ContractNameSimpleAccount:         {},
	ContractNameSimpleAccountFactory:  {},
}
