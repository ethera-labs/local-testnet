# L1 Network Command

The `localnet l1` command manages a local Layer 1 Ethereum test network using Kurtosis.

## Architecture

![L1 Architecture](../../docs/l1-architecture.png)

## What It Does

The L1 command deploys a complete Ethereum network with:
- **Execution Layer**: Geth clients
- **Consensus Layer**: Lighthouse clients
- **SSV Nodes**: Distributed validator technology nodes via `github.com/ssvlabs/ssv-mini` package
- **MEV Infrastructure**: Relay, builder, and boost components

## Prerequisites

- [Kurtosis](https://docs.kurtosis.com/install) - Container orchestration platform
- Docker - For running containerized services

## Configuration

Network parameters are embedded from `internal/l1/params.yaml` at build time:
- Ethereum fork configurations (Capella, Deneb, etc.)
- Network participants (execution and consensus clients)
- Validator counts
- SSV and Anchor node configurations
- MEV settings

The observability stack runs independently - see [`internal/observability`](../observability/README.md).

## Usage

```bash
# Start L1 network
make run-l1

# Or run directly
./cmd/localnet/bin/localnet l1

# Inspect the running network
make show-l1

# Clean up
make clean-l1
```

## Implementation Details

The L1 network is orchestrated through Kurtosis:
- Creates an enclave named `localnet`
- Runs the `github.com/ssvlabs/ssv-mini` Starlark package
- Parameters are embedded from `params.yaml`
- Real-time progress logging via structured output streams

## Stopping Services

```bash
# Stop the enclave (preserves state for restart)
make stop-l1

# Remove everything (full cleanup)
make clean-l1
```

## Viewing Logs

L1 services run via Kurtosis. Use Kurtosis commands to view logs:

```bash
# List all services in the enclave
kurtosis enclave inspect localnet

# View logs for a specific service
kurtosis service logs localnet <service-name>

# Follow logs in real-time
kurtosis service logs localnet <service-name> -f
```

## Troubleshooting

**Issue:** Kurtosis enclave already exists
**Solution:** Clean existing enclaves with `make clean-l1` or `kurtosis clean -a`

For more details, see the [main README](../../README.md).
