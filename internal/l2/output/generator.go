package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/ethera-labs/local-testnet/configs"
	"github.com/ethera-labs/local-testnet/internal/l2/l2runtime/contracts"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v3"
)

const fileName = "output.yaml"

type Generator struct {
}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) Generate(_ context.Context, deployedContracts map[configs.L2ChainName]map[contracts.ContractName]common.Address) error {
	compiledContracts, err := contracts.LoadCompiledContracts()
	if err != nil {
		return fmt.Errorf("could not load compiled contracts. Err: '%w'", err)
	}

	//NOTE: contracts on both rollups have the same address, so we can just take from one of them
	chainContracts := deployedContracts[configs.L2ChainNameRollupA]
	model := &Model{
		L2: L2{
			ChainConfigs: map[configs.L2ChainName]ChainConfig{
				configs.L2ChainNameRollupA: {
					ID:     configs.Values.L2.ChainConfigs[configs.L2ChainNameRollupA].ID,
					RPCURL: buildURL("http", "localhost", configs.Values.L2.ChainConfigs[configs.L2ChainNameRollupA].RPCPort),
					PK:     configs.Values.L2.Wallet.PrivateKey,
				},
				configs.L2ChainNameRollupB: {
					ID:     configs.Values.L2.ChainConfigs[configs.L2ChainNameRollupB].ID,
					RPCURL: buildURL("http", "localhost", configs.Values.L2.ChainConfigs[configs.L2ChainNameRollupB].RPCPort),
					PK:     configs.Values.L2.Wallet.PrivateKey,
				},
			},
			Contracts: map[string]ContractConfig{
				strings.ToLower(string(contracts.ContractNameBridge)): {
					Address: chainContracts[contracts.ContractNameBridge],
					ABI:     SingleQuotedString(compactJSON(compiledContracts[contracts.ContractNameBridge].RawABI)),
				},
				strings.ToLower(string(contracts.ContractNameMailbox)): {
					Address: chainContracts[contracts.ContractNameMailbox],
					ABI:     SingleQuotedString(compactJSON(compiledContracts[contracts.ContractNameMailbox].RawABI)),
				},
				strings.ToLower(string(contracts.ContractNameBridgeableToken)): {
					Address: chainContracts[contracts.ContractNameBridgeableToken],
					ABI:     SingleQuotedString(compactJSON(compiledContracts[contracts.ContractNameBridgeableToken].RawABI)),
				},
			},
		},
	}

	data, err := yaml.Marshal(model)
	if err != nil {
		return fmt.Errorf("could not marshal output model. Err: '%w'", err)
	}

	if err := os.WriteFile(fileName, data, 0644); err != nil {
		return fmt.Errorf("could not write output file. Err: '%w'", err)
	}

	return nil
}

func buildURL(scheme, host string, port int) string {
	addr := url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", host, port),
	}
	return addr.String()
}

func compactJSON(jsonStr string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(jsonStr)); err != nil {
		return jsonStr
	}
	return buf.String()
}
