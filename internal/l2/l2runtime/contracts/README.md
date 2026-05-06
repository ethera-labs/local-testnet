# Precompiled Smart Contracts

This directory contains precompiled Solidity smart contracts for the Ethera Labs rollup system.

## Contents

- `compiled/contracts.json` - Precompiled contract ABIs and bytecode

## Contracts

The following contracts are included:

1. **UniversalBridgeMailbox** - Universal bridge message passing contract
2. **CetFactory** - Composable ERC20 factory
3. **ComposeETHLiquidity** - ETH liquidity helper for L2-to-L2 bridging
4. **ComposeL2ToL2Bridge** - Universal L2-to-L2 bridge
5. **MockL2ERC20** - ERC20 token for local testing

## Deployment

These contracts are deployed automatically by the Go deployment code in `../shared/contracts/deploy.go`. The deployment process:

1. Loads precompiled contracts from `compiled/contracts.json`
2. Deploys to both Rollup A and Rollup B
3. Writes contract addresses to configuration files

## Recompiling Contracts

If you need to modify the contracts and recompile them:

1. Get the source files from the parent repository or git history
2. Install dependencies: `forge install`
3. Compile: `forge build` or use solc directly
4. Extract ABI and bytecode to `compiled/contracts.json` in this format:

```json
{
  "ContractName": {
    "abi": "[...]",
    "bytecode": "0x..."
  }
}
```

The original source was located at:
- Solidity files: `src/*.sol`
- Dependencies: `lib/openzeppelin-contracts`, `lib/forge-std`
- Remappings: `remappings.txt`
