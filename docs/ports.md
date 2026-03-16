# L2 Service Ports

All ports are bound to `localhost` unless otherwise noted.

## Core Services

| Service     | Port (Chain A) | Port (Chain B) | Description           |
|-------------|----------------|----------------|-----------------------|
| op-geth RPC | 18545          | 28545          | Execution layer RPC   |
| op-node RPC | 19545          | 29545          | Rollup node RPC       |
| op-batcher  | 18548          | 28548          | Batcher admin RPC     |
| op-proposer | 18560          | 28560          | Proposer admin RPC    |
| Publisher   | 18080 (QUIC)   | —              | Sequencer connections |
| Publisher   | 18081 (HTTP)   | —              | Health, metrics, API  |

## Flashblocks (--flashblocks-enabled)

| Service      | Port (Chain A) | Port (Chain B) | Description                  |
|--------------|----------------|----------------|------------------------------|
| op-rbuilder  | 17545 (RPC)    | 27545 (RPC)    | Builder RPC for transactions |
| op-rbuilder  | 17552 (Engine) | 27552 (Engine) | Engine API                   |
| op-rbuilder  | 17111 (WS)     | 27111 (WS)     | Flashblocks WebSocket        |
| rollup-boost | 17551 (Engine) | 27551 (Engine) | Engine multiplexer           |
| rollup-boost | 17999 (SSE)    | 27999 (SSE)    | Flashblocks SSE stream       |

## Sidecar (--sidecar-enabled, requires flashblocks)

| Service | Port (Chain A) | Port (Chain B) | Description    |
|---------|----------------|----------------|----------------|
| Sidecar | 17090          | 27090          | XT API, health |

## Blockscout (--blockscout-enabled)

| Service    | Port (Chain A) | Port (Chain B) | Description    |
|------------|----------------|----------------|----------------|
| Blockscout | 19000          | 29000          | Block explorer |

## Compose Network Console (--frontend-enabled, requires flashblocks and sidecar)

| Service         | Port | Description                    |
|-----------------|------|--------------------------------|
| Compose Console | 3000 | Web UI for cross-chain testing |

## Summary by Command

```bash
# Minimal (no optional services)
make run-l2
# Ports: 18545, 28545, 19545, 29545, 18080, 18081, ...

# With flashblocks
make run-l2 L2_ARGS="--flashblocks-enabled"
# + 17545, 27545, 17552, 27552, 17111, 27111, 17551, 27551, 17999, 27999

# With sidecar (requires flashblocks)
make run-l2 L2_ARGS="--flashblocks-enabled --sidecar-enabled"
# + 17090, 27090

# With Blockscout
make run-l2 L2_ARGS="--blockscout-enabled"
# + 19000, 29000

# Full stack including Compose Network Console
make run-l2 L2_ARGS="--flashblocks-enabled --blockscout-enabled --sidecar-enabled --frontend-enabled"
# + 3000 (Compose Console at http://localhost:3000)
```
