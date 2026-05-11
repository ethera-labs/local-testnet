# Flashblocks Support

Flashblocks enables external block building with sub-second latency
using [op-rbuilder](https://github.com/flashbots/op-rbuilder)
and [rollup-boost](https://github.com/flashbots/rollup-boost).

## Architecture

```
┌─────────────┐     ┌───────────────┐     ┌──────────────┐
│   op-node   │────▶│ rollup-boost  │────▶│ op-rbuilder  │  (builder)
└─────────────┘     └───────────────┘     └──────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   op-geth    │  (fallback)
                    └──────────────┘
```

- **op-rbuilder**: Block builder that produces flashblocks (sub-block updates)
- **rollup-boost**: Multiplexer that routes between op-geth (fallback) and op-rbuilder (builder)

## Quick Start

```bash
# Run L2 with flashblocks enabled
make run-l2 L2_ARGS="--flashblocks-enabled"
```

Image tags are configured under `l2.flashblocks` in `configs/config.yaml`.

## Builder Source

`op-rbuilder` build source is configured through `l2.repositories.op-rbuilder`.
Set either:
- `local-path` for a checked-out repository
- `url` and `branch` for a cloned repository source

Example with a checked-out repository:

```yaml
# configs/config.yaml
l2:
  repositories:
    op-rbuilder:
      local-path: ../op-rbuilder
```

When using remote `url` and `branch` values, Docker BuildKit must be able to access that repository source.

## Services

When flashblocks is enabled, 4 additional services start:

| Service        | Port                                                | Description               |
|----------------|-----------------------------------------------------|---------------------------|
| op-rbuilder-a  | 17545 (RPC), 17552 (Engine), 17111 (Flashblocks WS) | Block builder for Chain A |
| op-rbuilder-b  | 27545 (RPC), 27552 (Engine), 27111 (Flashblocks WS) | Block builder for Chain B |
| rollup-boost-a | 17551 (Engine), 17999 (Flashblocks SSE)             | Multiplexer for Chain A   |
| rollup-boost-b | 27551 (Engine), 27999 (Flashblocks SSE)             | Multiplexer for Chain B   |

## Transaction Submission

With flashblocks enabled, send transactions to **op-rbuilder's RPC** (not op-geth):

| Chain    | RPC Endpoint             |
|----------|--------------------------|
| Rollup A | `http://localhost:17545` |
| Rollup B | `http://localhost:27545` |

This is required because op-rbuilder maintains its own mempool and builds blocks.

## Flashblocks Streaming

Subscribe to flashblocks via WebSocket:

```bash
# Rollup A flashblocks stream
wscat -c ws://localhost:17111

# Rollup B flashblocks stream  
wscat -c ws://localhost:27111
```

Or via SSE through rollup-boost:

```bash
# Rollup A SSE
curl http://localhost:17999/flashblocks

# Rollup B SSE
curl http://localhost:27999/flashblocks
```

## Testing

```bash
# eth_call simulation through op-rbuilder
curl -s http://localhost:17545 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_call","params":[{"to":"0xe5D5d610fb9767Df117f4076444B45404201a097","data":"0x3b2bcbf1"},"latest"],"id":1}'

# debug_traceCall for transaction simulation
curl -s http://localhost:17545 \
  -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"debug_traceCall","params":[{"to":"0xe5D5d610fb9767Df117f4076444B45404201a097","data":"0x3b2bcbf1"},"latest",{"tracer":"prestateTracer","tracerConfig":{"diffMode":true}}],"id":1}'
```

## Configuration

In `config.yaml`:

```yaml
l2:
  flashblocks:
    enabled: true
    op-rbuilder-image-tag: "latest"
    rollup-boost-image-tag: "latest"
    rollup-a-rpc-port: 17545  # op-rbuilder RPC port for Chain A
    rollup-b-rpc-port: 27545  # op-rbuilder RPC port for Chain B
```

Or via CLI flags:

```bash
--flashblocks-enabled                    # Enable flashblocks
--flashblocks-rollup-a-rpc-port=17545   # Rollup A op-rbuilder RPC port
--flashblocks-rollup-b-rpc-port=27545   # Rollup B op-rbuilder RPC port
```
