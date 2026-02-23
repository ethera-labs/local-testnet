#!/bin/bash
set -euo pipefail

NODE_STORE="${NODE_STORE:-/home/celestia}"
P2P_NETWORK="${P2P_NETWORK:-private}"
CORE_IP="${CORE_IP:-celestia}"
CORE_PORT="${CORE_PORT:-9090}"
RPC_ADDR="${RPC_ADDR:-0.0.0.0}"
RPC_PORT="${RPC_PORT:-26658}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"

echo "Waiting for core endpoint ${CORE_IP}:${CORE_PORT}..."
for _ in $(seq 1 120); do
  if (echo >"/dev/tcp/${CORE_IP}/${CORE_PORT}") >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if [ ! -f "${NODE_STORE}/config.toml" ]; then
  echo "Initializing local bridge store at ${NODE_STORE}..."
  celestia bridge init \
    --node.store "${NODE_STORE}" \
    --p2p.network "${P2P_NETWORK}" \
    --core.ip "${CORE_IP}" \
    --core.port "${CORE_PORT}" \
    --rpc.addr "${RPC_ADDR}" \
    --rpc.port "${RPC_PORT}" \
    --rpc.skip-auth \
    --keyring.backend "${KEYRING_BACKEND}"
else
  echo "Reusing existing bridge store at ${NODE_STORE}"
fi

exec celestia bridge start \
  --node.store "${NODE_STORE}" \
  --p2p.network "${P2P_NETWORK}" \
  --core.ip "${CORE_IP}" \
  --core.port "${CORE_PORT}" \
  --rpc.addr "${RPC_ADDR}" \
  --rpc.port "${RPC_PORT}" \
  --rpc.skip-auth \
  --keyring.backend "${KEYRING_BACKEND}"

