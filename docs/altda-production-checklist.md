# AltDA Production Verification Checklist

Before rollout:

- `op-stack-a.yaml` and `op-stack-b.yaml` both contain `--altda.enabled=true`, the stack-specific `--altda.da-server=...:3100`, and `--altda.da-service=true` for `node` and `batcher`
- two AltDA server services exist and their DNS matches the guide
- each AltDA server exposes port `3100`, has its own persistent storage, and keeps generic commitments disabled

After replacing artifacts from the AltDA-enabled `op-deployer` run:

- `source/intent.toml` contains `[chains.dangerousAltDAConfig]` for both chains with `useAltDA = true`, `daCommitmentType = "KeccakCommitment"`, `daChallengeWindow = 1`, `daResolveWindow = 1`, `daBondSize = 1`, and `daResolverRefundPercentage = 0`
- `source/state.json` shows `useAltDA=true`, `daCommitmentType="KeccakCommitment"`, `daChallengeWindow=1`, `daResolveWindow=1`, `daBondSize=1`, `daResolverRefundPercentage=0`, and non-zero DAC proxy and implementation addresses for both chains
- `source/rollup-a.json` and `source/rollup-b.json` include `alt_da` with `da_commitment_type="KeccakCommitment"`, `da_challenge_window=1`, `da_resolve_window=1`, and a non-zero `da_challenge_contract_address`
- `config-resources/rollup-a-config.yaml` and `rollup-b-config.yaml` embed the new rollup JSON
- `config-resources/rollup-a-genesis.yaml` and `rollup-b-genesis.yaml` were replaced from the same `op-deployer` run

After deployment:

- `op-node` starts without AltDA config validation errors
- `op-batcher` can reach its AltDA server
- the AltDA services store data successfully
