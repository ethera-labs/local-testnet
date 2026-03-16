# AltDA Production Upgrade Guide

## 1. Add New `op-alt-da` Services

Add two new services under `gitops-production/environments/aws/hoodi/compose/`:

- `op-alt-da-a`
- `op-alt-da-b`

Use these service URLs:

| Service | URL |
| --- | --- |
| `op-alt-da-a` | `http://hoodi-op-alt-da-a.compose.svc:3100` |
| `op-alt-da-b` | `http://hoodi-op-alt-da-b.compose.svc:3100` |

Each service expected to be launched with flags below:

```sh
da-server \
  --addr=0.0.0.0 \
  --port=3100 \
  --file.path=/data \
  --log.level=info
```

For each service:

- mount persistent storage at `/data`
- expose port `3100`

Reference links:

- `op-alt-da` source: `https://github.com/ethereum-optimism/optimism/tree/develop/op-alt-da`
- Dockerfile used in this repo: `https://github.com/ethera-labs/local-testnet/blob/alt-da-support/internal/l2/infra/docker/op-alt-da.Dockerfile#L1-L17`
- `op-alt-da-a` docker-compose example: `https://github.com/ethera-labs/local-testnet/blob/alt-da-support/internal/l2/infra/docker/docker-compose.yml#L159-L186`

## 2. Update `op-stack-a.yaml` And `op-stack-b.yaml`

Update these files:

- `gitops-production/environments/aws/hoodi/compose/op-stack-a.yaml`
- `gitops-production/environments/aws/hoodi/compose/op-stack-b.yaml`

In both files:

- update `op-node.image.tag` to `v1.16.2`
- add the AltDA flags below to both `node.extraArgs` and `batcher.extraArgs`

Use this `--altda.da-server` value per file:

| File | `--altda.da-server` |
| --- | --- |
| `op-stack-a.yaml` | `http://hoodi-op-alt-da-a.compose.svc:3100` |
| `op-stack-b.yaml` | `http://hoodi-op-alt-da-b.compose.svc:3100` |

Flags to add:

```yaml
- --altda.enabled=true
- --altda.da-server=<stack-specific URL from the table above>
- --altda.da-service=true
```

## 3. Update `source/intent.toml`

Update `gitops-production/environments/aws/hoodi/compose/source/intent.toml`.

Add this block for each chain before running `op-deployer`:

```toml
[chains.dangerousAltDAConfig]
useAltDA = true
daCommitmentType = "KeccakCommitment"
daChallengeWindow = 1
daResolveWindow = 1
daBondSize = 1
daResolverRefundPercentage = 0
```

## 4. Run `op-deployer` And Replace Generated Files

Run the existing AltDA-enabled `op-deployer` flow with `gitops-production/environments/aws/hoodi/compose/source/intent.toml` as input.

Replace these files with the outputs from that run:

1. `gitops-production/environments/aws/hoodi/compose/source/state.json`
2. `gitops-production/environments/aws/hoodi/compose/source/rollup-a.json`
3. `gitops-production/environments/aws/hoodi/compose/source/rollup-b.json`
4. `gitops-production/environments/aws/hoodi/compose/config-resources/rollup-a-config.yaml`
5. `gitops-production/environments/aws/hoodi/compose/config-resources/rollup-b-config.yaml`
6. `gitops-production/environments/aws/hoodi/compose/config-resources/rollup-a-genesis.yaml`
7. `gitops-production/environments/aws/hoodi/compose/config-resources/rollup-b-genesis.yaml`
