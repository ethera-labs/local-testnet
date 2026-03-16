package output

import (
	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

type (
	Model struct {
		L2 L2 `yaml:"l2"`
	}
	L2 struct {
		ChainConfigs map[configs.L2ChainName]ChainConfig `yaml:"chain-configs"`
		Contracts    map[string]ContractConfig           `yaml:"contracts"`
	}
	ChainConfig struct {
		ID     int    `yaml:"id"`
		RPCURL string `yaml:"rpc-url"`
		PK     string `yaml:"pk"`
	}

	ContractConfig struct {
		Address common.Address     `yaml:"address"`
		ABI     SingleQuotedString `yaml:"abi"`
	}

	SingleQuotedString string
)

func (s SingleQuotedString) MarshalYAML() (any, error) {
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Style: yaml.SingleQuotedStyle,
		Value: string(s),
	}
	return node, nil
}
