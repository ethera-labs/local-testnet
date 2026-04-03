# L2 Network Command

The `localnet l2` command manages Layer 2 rollup networks built on the OP Stack.

## Architecture

![L2 Architecture](../../docs/l2-architecture.png)

## What It Does

The L2 command orchestrates a complete rollup deployment in three phases:

### Phase 1: L1 Contract Deployment

Deploys OP Stack contracts to L1 using `op-deployer`:

- System Config
- L1 Standard Bridge
- L1 Cross Domain Messenger
- OptimismPortal
- DisputeGameFactory
- And other core contracts

### Phase 2: Configuration Generation

Generates configuration files for each L2 chain:

- `genesis.json` - Initial blockchain state
- `rollup.json` - Rollup configuration
- `jwt-secret.txt` - Authentication between services

### Phase 3: Runtime Deployment

Starts L2 services using Docker:

- **op-geth**: Execution client for each rollup
- **op-node**: Consensus/derivation client
- **op-batcher**: Batches transactions to L1
- **op-proposer**: Proposes output roots to L1
- **Publisher**: Publishes superblocks to L1

**Optional services / modes** (enabled via CLI flags):

- **AltDA**: Alternative data availability mode with per-rollup DA servers (`--alt-da-enabled`)
- **op-rbuilder**: External block builder for flashblocks (`--flashblocks-enabled`)
- **rollup-boost**: Engine API multiplexer for flashblocks (`--flashblocks-enabled`)
- **blockscout**: Block explorer UI (`--blockscout-enabled`)
- **sidecar**: Cross-chain coordination (`--sidecar-enabled`, requires flashblocks)
- **Ethera Labs Console**: Web UI for XT testing (`--frontend-enabled`, requires flashblocks and sidecar)

Deploys Ethera Labs contracts to L2:

- Dispute settlement contracts
- Verification contracts

## Prerequisites

- **Foundry/Forge**: For Solidity compilation
- **just**: Command runner for contract scripts
- **jq**: JSON processor for deployment scripts
- **Docker**: For running L2 services

## Configuration

All L2 settings are configured in `configs/config.yaml`. See [example config](../../configs/config.example.yaml) for all
available options.

**Required settings:**

- L1 connection (chain ID, EL URL, CL URL)
- Wallet credentials (private key, address)
- Coordinator credentials — must use a **different private key** than the wallet to avoid nonce collisions (both submit
  L1 transactions independently)
- Ethera Labs network name
- Dispute game settings (addresses, vkeys, explorer URLs)

## Usage

### Running L2 Networks

```bash
# Start L2 deployment (all phases)
make run-l2

# With optional features
make run-l2 L2_ARGS="--alt-da-enabled"                   # Route batches through AltDA servers
make run-l2 L2_ARGS="--flashblocks-enabled"              # Enable flashblocks
make run-l2 L2_ARGS="--blockscout-enabled"               # Enable block explorer
make run-l2 L2_ARGS="--flashblocks-enabled --blockscout-enabled"  # Both
make run-l2 L2_ARGS="--flashblocks-enabled --blockscout-enabled --sidecar-enabled --frontend-enabled"  # Full stack

# Or run directly
./cmd/localnet/bin/localnet l2
./cmd/localnet/bin/localnet l2 --flashblocks-enabled --blockscout-enabled

# Show running services
make show-l2

# Clean up
make clean-l2
```

For flashblocks documentation, see [docs/flashblocks.md](../../docs/flashblocks.md).

### Flashblocks and Sidecar Sources

Flashblocks and sidecar build sources are configured through `l2.repositories`.
Each source-built repository must set exactly one of:
- `local-path` for a checked-out repository
- `url` and `branch` for a cloned repository source

Use `local-path` when building from repositories already present on the machine running `local-testnet`:

```yaml
# configs/config.yaml
l2:
  repositories:
    op-rbuilder:
      local-path: ../op-rbuilder
    sidecar:
      local-path: ../sidecar
```

When using remote `url` and `branch` values, Docker BuildKit must be able to access those repositories:

```bash
eval "$(ssh-agent -s)" && ssh-add ~/.ssh/id_ed25519
```

### AltDA

`--alt-da-enabled` switches each rollup from calldata publication to a dedicated AltDA server.

Operational guidance:

- Leave `l2.alt-da.skip-l1-deploy=false` for ephemeral environments where localnet should deploy the AltDA challenge
  contracts.
- Set `l2.alt-da.skip-l1-deploy=true` only in long-lived environments that already have pinned `challenge-proxy-address`
  and `challenge-impl-address` values.
- Ensure `repositories.ethera-contracts` resolves to a checkout that contains the `op-alt-da` sources used to build the
  DA server image.
### Local Development

Point source-built services at checked-out repositories through `l2.repositories.*.local-path`:

```yaml
# configs/config.yaml
l2:
  repositories:
    op-geth:
      local-path: ../op-geth  # Relative path
    op-rbuilder:
      local-path: ../op-rbuilder
    publisher:
      local-path: ~/projects/publisher  # Absolute path with ~
    sidecar:
      local-path: ~/projects/sidecar
```

Rebuild and restart specific services after code changes:

```bash
# Rebuild and restart publisher service only
make run-l2-deploy SERVICE=publisher

# Rebuild and restart op-geth services only
make run-l2-deploy SERVICE=op-geth

# Rebuild and restart all services
make run-l2-deploy SERVICE=all
```

This skips full redeployment (Phase 1-2) and only rebuilds Docker images + restarts containers.

### Compiling Contracts

```bash
# Compile contracts from ethera-contracts repository
make run-l2-compile

# Or run directly
./cmd/localnet/bin/localnet l2 compile
```

This generates `contracts.json` in `.localnet/compiled-contracts/`. To embed in binary, copy to
`internal/l2/l2runtime/contracts/compiled/` and commit.

### Docker Usage

For running in Docker, see the [Docker documentation](../../build/DOCKER.md).

**Quick start:**

```bash
# Edit configs/config.yaml with your settings
./build/docker-run-example.sh
```

The container automatically uses your `configs/config.yaml`. You can also override settings with CLI flags.

For more details, see the [Docker documentation](../../build/DOCKER.md).

## Stopping Services

```bash
# Stop containers without removing configs (preserves .localnet/ files)
make stop-l2

# Stop and remove everything (containers + volumes + generated configs)
make clean-l2

# Also remove locally built L2 images
make clean-l2-full
```

## Viewing Logs

L2 services run as Docker containers. View logs using standard Docker commands:

```bash
# Core services
docker logs publisher -f
docker logs op-geth-a -f
docker logs op-geth-b -f
docker logs op-node-a -f
docker logs op-node-b -f
docker logs op-batcher-a -f
docker logs op-batcher-b -f
docker logs op-proposer-a -f
docker logs op-proposer-b -f

# Flashblocks services (when --flashblocks-enabled)
docker logs op-rbuilder-a -f
docker logs op-rbuilder-b -f
docker logs rollup-boost-a -f
docker logs rollup-boost-b -f

# Blockscout services (when --blockscout-enabled)
docker logs blockscout-a -f
docker logs blockscout-b -f

# View last N lines
docker logs op-geth-a --tail 100

# View logs with timestamps
docker logs op-geth-a -t

# View all L2 logs aggregated via docker-compose
docker compose -f .localnet/docker-compose.yml logs -f

# View specific services via docker-compose
docker compose -f .localnet/docker-compose.yml logs -f publisher op-geth-a op-geth-b
```

## Service Ports

| Service         | Chain A | Chain B | Description                 |
|-----------------|---------|---------|-----------------------------|
| op-geth RPC     | 18545   | 28545   | Execution RPC               |
| op-rbuilder RPC | 17545   | 27545   | Flashblocks RPC             |
| sidecar         | 17090   | 27090   | Sidecar API                 |
| Blockscout      | 19000   | 29000   | Block explorer UI           |
| Ethera Console  | 3000    | —       | Web UI (--frontend-enabled) |

## Sidecar Mode

The sidecar handles cross-chain transaction coordination as a standalone service.

### Running with Sidecar Mode

```bash
# Requires flashblocks
make run-l2 L2_ARGS="--flashblocks-enabled --sidecar-enabled"

# View logs
docker logs sidecar-a -f
docker logs sidecar-b -f
```

### Configuration

Configure sidecar build source in `l2.repositories.sidecar`.
