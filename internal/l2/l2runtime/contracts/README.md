# Precompiled L2 Contracts

This package embeds and deploys the L2 contracts that the rollup pair requires
for cross-chain composability.

## Layout

- `compiled/contracts.json` — embedded ABI + bytecode for every contract listed
  in `contract.go`.
- `loader.go` — loads `compiled/contracts.json` at runtime.
- `deployer.go` — deploys the contracts to each rollup and writes
  `<networks-dir>/<chain>/contracts.json`.
- `compiler.go` — invokes the contracts repository to regenerate
  `compiled/contracts.json`.

## Contracts

| Name                     | Purpose                                           |
|--------------------------|---------------------------------------------------|
| `UniversalBridgeMailbox` | Cross-rollup message inbox/outbox                 |
| `CetFactory`             | Composable ERC-20 token factory                   |
| `ComposeETHLiquidity`    | ETH liquidity helper used by the L2-to-L2 bridge  |
| `ComposeL2ToL2Bridge`    | L2-to-L2 bridge that composes the contracts above |
| `MockL2ERC20`            | ERC-20 token used by integration tests            |

## Regenerating `contracts.json`

```bash
make run-l2-compile
```

This runs `localnet l2 compile`, which builds the contracts repository checked
out in `l2.repositories.ethera-contracts` and writes
`.localnet/compiled-contracts/contracts.json`. To embed the new artefact in the
binary, copy it into `compiled/` and commit:

```bash
cp .localnet/compiled-contracts/contracts.json \
   internal/l2/l2runtime/contracts/compiled/contracts.json
```

## Deployment Flow

1. `Deployer.Deploy` waits for each rollup RPC to be reachable and producing
   blocks.
2. Contracts deploy to both rollups using the coordinator key.
3. Bridges are authorised against `UniversalBridgeMailbox`, `CetFactory`, and
   `ComposeETHLiquidity` so `ComposeL2ToL2Bridge` can move state.
4. Deployed addresses are validated to match across rollups (CREATE2-style
   determinism is required) and written to `contracts.json` per chain.
