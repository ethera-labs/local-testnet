#!/bin/sh
set -eu

HOME_DIR="${CELESTIA_APP_HOME:-/home/celestia/.celestia-app}"
CHAIN_ID="${CHAIN_ID:-private}"
MONIKER="${MONIKER:-local-validator}"
KEY_NAME="${KEY_NAME:-validator}"
KEYRING_BACKEND="${KEYRING_BACKEND:-test}"
GENESIS_BALANCE="${GENESIS_BALANCE:-1000000000000utia}"
SELF_DELEGATION="${SELF_DELEGATION:-500000000utia}"
GENTX_FEES="${GENTX_FEES:-1utia}"

if [ ! -f "${HOME_DIR}/config/genesis.json" ]; then
  echo "Initializing local Celestia chain from genesis..."
  celestia-appd init "${MONIKER}" --chain-id "${CHAIN_ID}" --home "${HOME_DIR}" >/dev/null
  celestia-appd keys add "${KEY_NAME}" --keyring-backend "${KEYRING_BACKEND}" --home "${HOME_DIR}"

  ADDRESS="$(celestia-appd keys show "${KEY_NAME}" -a --keyring-backend "${KEYRING_BACKEND}" --home "${HOME_DIR}" | tail -n 1)"

  celestia-appd genesis add-genesis-account "${ADDRESS}" "${GENESIS_BALANCE}" --home "${HOME_DIR}" >/dev/null
  celestia-appd genesis gentx \
    "${KEY_NAME}" \
    "${SELF_DELEGATION}" \
    --chain-id "${CHAIN_ID}" \
    --fees "${GENTX_FEES}" \
    --keyring-backend "${KEYRING_BACKEND}" \
    --home "${HOME_DIR}" >/dev/null
  celestia-appd genesis collect-gentxs --home "${HOME_DIR}" >/dev/null
  celestia-appd genesis validate --home "${HOME_DIR}" >/dev/null

  echo "Initialized ${CHAIN_ID} with validator address ${ADDRESS}"
else
  echo "Reusing existing chain data at ${HOME_DIR}"
fi

# Celenium needs block_results, which requires persisted ABCI responses.
CONFIG_TOML="${HOME_DIR}/config/config.toml"
if [ -f "${CONFIG_TOML}" ]; then
  sed -i 's/^discard_abci_responses = true/discard_abci_responses = false/' "${CONFIG_TOML}"
fi

exec celestia-appd start \
  --home "${HOME_DIR}" \
  --force-no-bbr \
  --minimum-gas-prices "0.000001utia" \
  --rpc.laddr "tcp://0.0.0.0:26657" \
  --rpc.grpc_laddr "tcp://0.0.0.0:9098" \
  --p2p.laddr "tcp://0.0.0.0:26656" \
  --grpc.enable \
  --grpc.address "0.0.0.0:9090" \
  --grpc-web.enable \
  --api.enable \
  --api.address "tcp://0.0.0.0:1317"

