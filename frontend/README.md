# Compose Network Console

Web UI for testing cross-chain transactions against the local-testnet stack.

## Running with L2 (Docker)

Start the full stack including the console:

```bash
make run-l2 L2_ARGS="--flashblocks-enabled --blockscout-enabled --sidecar-enabled --frontend-enabled"
```

Open http://localhost:3000

## Running Locally (Development)

1. Start L2 with flashblocks and sidecar:
   ```bash
   make run-l2 L2_ARGS="--flashblocks-enabled --sidecar-enabled"
   ```

2. Run the frontend:
   ```bash
   make run-frontend
   # or: cd frontend && npm install && npm run dev
   ```

Open http://localhost:5173

## Configuration

`.env` should match your local-testnet deployment. Contract addresses come from `output.yaml` after L2 deployment.
See [docs/ports.md](../docs/ports.md) for the full port reference.

| Service     | Chain A | Chain B |
|-------------|---------|---------|
| Builder RPC | 17545   | 27545   |
| Sidecar API | 17090   | 27090   |
| Blockscout  | 19000   | 29000   |
