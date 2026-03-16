#!/bin/bash
# Docker run command for L2 deployment
#
# Configuration:
# 1. Using config.yaml (recommended):
#    - Edit configs/config.yaml to customize all settings
#    - Config is automatically loaded from the mounted workspace
#
# 2. Using CLI flags (alternative):
#    - Pass flags directly: docker run ... l2 --l1-chain-id 1 --l1-el-url http://...
#    - Useful for CI/CD or when you don't want to modify config.yaml
#    - Flags override config.yaml values
#    - See available flags: docker run ... l2 --help
#
# Note: The workspace mount (-v $(PWD):/workspace) is required for:
# - Reading configs/config.yaml
# - Cloning repositories to .localnet/
# - Accessing local repositories (if using local-path in config)

docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $(PWD):/workspace \
  -w /workspace \
  -e HOST_PROJECT_PATH=$(PWD) \
  ethera-labs/local-testnet:latest \
  l2
