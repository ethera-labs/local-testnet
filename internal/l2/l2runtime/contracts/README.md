# Precompiled Smart Contracts

This directory contains precompiled Solidity smart contracts for the Ethera Labs rollup system.

## Contents

- `compiled/contracts.json` - Precompiled contract ABIs and bytecode

## Contracts

The following contracts are included:

1. **Mailbox** - Cross-chain message passing contract
2. **PingPong** - Example cross-chain ping-pong application
3. **Bridge** - Token bridging contract
4. **MyToken** - ERC20 token (MTK) for testing

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
