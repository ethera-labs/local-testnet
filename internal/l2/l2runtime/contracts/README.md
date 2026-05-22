# Precompiled L2 Contracts

This package embeds and deploys the L2 contracts that the rollup pair requires
for cross-chain composability.

## Layout

- `compiled/contracts.json`- embedded ABI + bytecode for every Ethera
  Labs contract listed in `contract.go`.
- `compiled/entrypoint.json`- embedded ABI + bytecode + `deployedBytecode`
  for the ERC-4337 v0.7 contracts compiled from the upstream
  `eth-infinitism/account-abstraction` repository.
- `loader.go`- loads both embedded artefacts at runtime.
- `deployer.go`- deploys the contracts to each rollup and writes the
  resulting `<networks-dir>/<chain>/contracts.json`.
- `compiler.go`- regenerates the embedded artefacts via `forge inspect`.

## Contracts

| Name                     | Source repository                     | Purpose                                                                   |
|--------------------------|---------------------------------------|---------------------------------------------------------------------------|
| `UniversalBridgeMailbox` | `l2.repositories.ethera-contracts`    | Cross-rollup message inbox/outbox                                         |
| `CetFactory`             | `l2.repositories.ethera-contracts`    | Composable ERC-20 token factory                                           |
| `ComposeETHLiquidity`    | `l2.repositories.ethera-contracts`    | ETH liquidity helper used by the L2-to-L2 bridge                          |
| `ComposeL2ToL2Bridge`    | `l2.repositories.ethera-contracts`    | L2-to-L2 bridge composing the contracts above                             |
| `MockL2ERC20`            | `l2.repositories.ethera-contracts`    | ERC-20 token used by integration tests                                    |
| `EntryPoint`             | `l2.repositories.account-abstraction` | ERC-4337 v0.7 EntryPoint                                                  |
| `SimpleAccount`          | `l2.repositories.account-abstraction` | Reference ERC-4337 v0.7 smart account (deployed on demand by the factory) |
| `SimpleAccountFactory`   | `l2.repositories.account-abstraction` | Deterministic factory for `SimpleAccount`; constructor takes EntryPoint   |

`EntryPointSimulations` is intentionally absent from the deployment list. Its
runtime bytecode is embedded directly in the bundler binary (see
[`crates/bundler/assets/entrypoint_simulations_v07.bin`][asset] in the bundler
repo) and applied as an `eth_call` state override during per-op validation
nothing needs to be written on-chain or recorded in `contracts.json`.

[asset]: https://github.com/ethera-labs/ethera-bundler/tree/main/crates/bundler/assets

## Regenerating the embedded artefacts

```bash
make run-l2-compile
```

`localnet l2 compile` builds the contracts checked out under
`l2.repositories.ethera-contracts` and, when configured, also builds
`EntryPoint` + `EntryPointSimulations` from
`l2.repositories.account-abstraction`. Outputs are written to
`.localnet/compiled-contracts/`.

To update the embedded copies used by the binary, copy the regenerated files
into `compiled/` and commit:

```bash
cp .localnet/compiled-contracts/contracts.json  internal/l2/l2runtime/contracts/compiled/contracts.json
cp .localnet/compiled-contracts/entrypoint.json internal/l2/l2runtime/contracts/compiled/entrypoint.json
```

## Deployment Flow

1. `Deployer.Deploy` waits for each rollup RPC to be reachable and producing
   blocks.
2. Contracts deploy to both rollups using the coordinator key. `EntryPoint` is
   included whenever its artefact is present in `compiled/entrypoint.json`.
3. Bridges are authorised against `UniversalBridgeMailbox`, `CetFactory`, and
   `ComposeETHLiquidity` so `ComposeL2ToL2Bridge` can move state.
4. Deployed addresses are validated to match across rollups (CREATE2-style
   determinism is required) and written to `contracts.json` per chain.
