#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

L2_ARGS="${L2_ARGS:-$*}"

echo "[1/6] Stopping existing L2 containers"
docker ps -aq --filter "label=stack=localnet-l2" | xargs -r docker rm -f

echo "[2/6] Removing stale L2 chain-state volumes"
docker volume rm \
  localnet_rollup-a-geth \
  localnet_rollup-b-geth \
  localnet_rollup-a-opnode \
  localnet_rollup-b-opnode 2>/dev/null || true

echo "[3/6] Removing generated L2 runtime files (preserving compiled contracts/images)"
rm -rf \
  ./.localnet/state \
  ./.localnet/networks \
  ./.localnet/docker-compose.yml \
  ./.localnet/docker-compose.blockscout.yml \
  ./.localnet/.tmp \
  ./.localnet/registry

echo "[4/6] Restarting L1 Kurtosis enclave"
kurtosis enclave stop localnet >/dev/null 2>&1 || true
kurtosis enclave rm localnet >/dev/null 2>&1 || true
make run-l1

echo "[5/6] Starting L2"
if [[ -n "${L2_ARGS}" ]]; then
  make run-l2 L2_ARGS="${L2_ARGS}"
else
  make run-l2
fi

echo "[6/6] Done"
