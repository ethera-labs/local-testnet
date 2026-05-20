package contracts

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

//go:embed compiled/contracts.json compiled/entrypoint.json
var compiledContractsFS embed.FS

// LoadCompiledContracts loads compiled Ethera Labs L2 contracts.
func LoadCompiledContracts() (map[ContractName]CompiledContract, error) {
	data, err := compiledContractsFS.ReadFile("compiled/contracts.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded contracts: %w", err)
	}

	return parseContracts(data, Contracts)
}

// LoadCompiledEntryPoint loads the ERC-4337 v0.7 EntryPoint artefact. Returns
// an empty map (without error) when the artefact has not been generated yet,
// so callers can treat EntryPoint deployment as optional.
func LoadCompiledEntryPoint() (map[ContractName]CompiledContract, error) {
	data, err := compiledContractsFS.ReadFile("compiled/entrypoint.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded EntryPoint contracts: %w", err)
	}
	if len(strings.TrimSpace(string(data))) <= len("{}") {
		return map[ContractName]CompiledContract{}, nil
	}
	return parseContracts(data, EntryPointContracts)
}

func parseContracts(data []byte, allow map[ContractName]struct{}) (map[ContractName]CompiledContract, error) {
	var result map[string]struct {
		ABI              json.RawMessage `json:"abi"`
		Bytecode         string          `json:"bytecode"`
		DeployedBytecode string          `json:"deployedBytecode,omitempty"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to parse compiled contracts: %w", err)
	}

	loadedContracts := make(map[ContractName]CompiledContract)

	for name, contract := range result {
		if _, ok := allow[ContractName(name)]; !ok {
			continue
		}

		parsedABI, err := abi.JSON(strings.NewReader(string(contract.ABI)))
		if err != nil {
			return nil, fmt.Errorf("failed to parse ABI for %s: %w", name, err)
		}

		bytecode := common.Hex2Bytes(strings.TrimPrefix(contract.Bytecode, "0x"))
		var deployed []byte
		if contract.DeployedBytecode != "" {
			deployed = common.Hex2Bytes(strings.TrimPrefix(contract.DeployedBytecode, "0x"))
		}

		loadedContracts[ContractName(name)] = CompiledContract{
			ABI:              parsedABI,
			RawABI:           string(contract.ABI),
			Bytecode:         bytecode,
			DeployedBytecode: deployed,
		}
	}

	return loadedContracts, nil
}
