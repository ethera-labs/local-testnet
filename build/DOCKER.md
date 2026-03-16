# Docker Usage

## Building the Image

```bash
make docker-build
```

## Running the Container

### Configuration Methods

There are two ways to configure the container:

#### 1. Using config.yaml (Recommended)

Edit `configs/config.yaml` with your settings, then run:

```bash
./build/docker-run-example.sh
```

The config file is automatically loaded from the mounted workspace. This is the simplest approach and keeps all settings in one place.

**Required settings in config.yaml:**
- `l2.l1-chain-id` - L1 network chain ID
- `l2.l1-el-url` - L1 execution client RPC URL
- `l2.l1-cl-url` - L1 consensus client REST URL
- `l2.wallet.private-key` - Private key for deployment
- `l2.wallet.address` - Wallet address
- `l2.coordinator-private-key` - Coordinator private key
- `l2.repositories` - Repository URLs/branches OR local paths for development
- `l2.dispute.*` - Dispute game configuration

See `configs/config.example.yaml` for all available options.

#### 2. Using CLI Flags (Alternative)

Pass configuration directly via command-line flags:

```bash
docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(PWD):/workspace \
  -w /workspace \
  -e HOST_PROJECT_PATH=$(PWD) \
  ethera-labs/local-testnet:latest \
  l2 \
  --l1-chain-id 560048 \
  --l1-el-url http://your-l1-node:8545 \
  --l1-cl-url http://your-l1-node:5052 \
  # ... (see --help for all flags)
```

This approach is useful for:
- CI/CD pipelines
- Automated deployments
- Overriding specific config values without modifying config.yaml

**Note:** CLI flags override config.yaml values when both are present.

### Local Development Workflow

For iterating on local changes to repositories (op-geth, publisher):

1. Configure local paths in `configs/config.yaml`:
```yaml
repositories:
  op-geth:
    local-path: ../op-geth  # Path to your local clone
```

2. Make your changes in the local repository

3. The container will use your local repository instead of cloning

**Note:** Local paths in Docker require the repository to be within or adjacent to the project directory so it can be accessed via the workspace mount.


## Installed Tools

The image includes:
- **localnet** binary at `/usr/local/bin/localnet`
- **Docker CLI** with compose plugin
- **Git** for repository cloning
- **Foundry** (forge, cast, anvil) for Solidity compilation
- **just** - Command runner for contract setup scripts
- **jq** - JSON processor for contract deployment scripts
