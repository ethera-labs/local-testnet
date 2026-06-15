# Ethera Labs Console

Web UI for testing cross-chain transactions against the local-testnet stack.

## Running with L2 (Docker)

Start the full stack including the console:

```bash
make run-l2 L2_ARGS="--flashblocks-enabled --blockscout-enabled --sidecar-enabled --frontend-enabled"
```

Open http://localhost:3000

## Hot-Reload in Docker

`--frontend-dev-enabled` builds the console against the Vite dev server and
mounts `frontend/src`, `index.html`, the Tailwind/PostCSS/TypeScript config,
and `vite.config.ts` into the container. Edits to those files reload live
in the browser, no container rebuild required.

```bash
make run-l2 L2_ARGS="--flashblocks-enabled --sidecar-enabled --frontend-dev-enabled"
```

The flag is mutually exclusive in spirit with `--frontend-enabled` (use one or
the other) and still requires flashblocks and sidecar.

## Running Locally (Outside Docker)

1. Start L2 with flashblocks and sidecar:
   ```bash
   make run-l2 L2_ARGS="--flashblocks-enabled --sidecar-enabled"
   ```

2. Run the frontend:
   ```bash
   make run-frontend
   # or: cd frontend && bun install && bun run dev
   ```

Open http://localhost:5173

## Configuration

`.env` should match your local-testnet deployment. Contract addresses come from `output.yaml` after L2 deployment.
See [docs/ports.md](../docs/ports.md) for the full port reference.

The bridge UI reads native token balances from `VITE_CHAIN_A_TOKEN_ADDRESS` and
`VITE_CHAIN_B_TOKEN_ADDRESS`. Bridged balances are CET balances derived through
`VITE_CET_FACTORY_ADDRESS`, so that value must be set when running the frontend
outside the local-testnet orchestrator.

The System Architecture diagram and the header status indicators are driven by
the `localnet-health` HTTP service, which inspects every container in the L2
stack and reports its status. Point the UI at it with `VITE_HEALTH_API_URL`
(defaults to `http://localhost:8090`); when the URL is unreachable every node
falls back to a "missing" indicator and optional services (AltDA, op-succinct,
sidecar, flashblocks) are hidden from the diagram.

| Service         | Chain A | Chain B |
|-----------------|---------|---------|
| Builder RPC     | 17545   | 27545   |
| Sidecar API     | 17090   | 27090   |
| Bundler         | 17082   | 27082   |
| Blockscout      | 19000   | 29000   |
| localnet-health | 8090    | -       |

## Bundler Test Tab

When the L2 stack is started with `--bundler-enabled` (which requires
`--flashblocks-enabled`), Phase 3 deploys `EntryPoint` v0.7 and
`SimpleAccountFactory` to each rollup and the docker-compose overlay forwards
the resulting addresses to the console as:

- `VITE_BUNDLER_A_URL`, `VITE_BUNDLER_B_URL`
- `VITE_ENTRYPOINT_A`, `VITE_ENTRYPOINT_B`
- `VITE_SIMPLE_ACCOUNT_FACTORY_A`, `VITE_SIMPLE_ACCOUNT_FACTORY_B`

With those set, the console renders an extra **Bundler** tab that derives a
`SimpleAccount` address from your `VITE_WALLET_PRIVATE_KEY`, tops up its
`EntryPoint` deposit if empty, builds + signs a no-op `UserOperation`, asks
the bundler for the sequencer-signed `handleOps` transaction
(`ethera_buildSignedUserOpsTx`), broadcasts it to op-rbuilder, and decodes the
`UserOperationEvent` from the receipt.

When running the frontend outside Docker, copy those values into `.env` from
`.localnet/networks/<chain>/contracts.json` after `make run-l2 L2_ARGS="--flashblocks-enabled --bundler-enabled"`
brings the stack up.
