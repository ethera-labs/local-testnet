const env = import.meta.env

function requireEnv(name: keyof ImportMetaEnv): string {
  const value = env[name]
  if (!value || typeof value !== 'string' || !value.trim()) {
    throw new Error(`Missing required env var: ${name}`)
  }
  return value.trim()
}

function requireNumber(name: keyof ImportMetaEnv): number {
  const raw = requireEnv(name)
  const parsed = Number(raw)
  if (!Number.isFinite(parsed)) {
    throw new Error(`Invalid number for env var: ${name}`)
  }
  return parsed
}

function normalizePrivateKey(value: string | undefined): string {
  if (!value) {
    return ''
  }
  const trimmed = value.trim()
  if (!trimmed) {
    return ''
  }
  return trimmed.startsWith('0x') ? trimmed : `0x${trimmed}`
}

export const CHAIN_A_ID = requireNumber('VITE_CHAIN_A_ID')
export const CHAIN_B_ID = requireNumber('VITE_CHAIN_B_ID')

export const FLASHBLOCKS_ENABLED = env.VITE_FLASHBLOCKS_ENABLED === 'true'

export const CHAIN_A_BUILDER_RPC = requireEnv('VITE_CHAIN_A_BUILDER_RPC')
export const CHAIN_A_GETH_RPC = requireEnv('VITE_CHAIN_A_GETH_RPC')

export const CHAIN_B_BUILDER_RPC = requireEnv('VITE_CHAIN_B_BUILDER_RPC')
export const CHAIN_B_GETH_RPC = requireEnv('VITE_CHAIN_B_GETH_RPC')

export const CHAIN_A_RPC = FLASHBLOCKS_ENABLED ? CHAIN_A_BUILDER_RPC : CHAIN_A_GETH_RPC
export const CHAIN_B_RPC = FLASHBLOCKS_ENABLED ? CHAIN_B_BUILDER_RPC : CHAIN_B_GETH_RPC

export const SIDECAR_A_URL = requireEnv('VITE_SIDECAR_A_URL')
export const SIDECAR_B_URL = requireEnv('VITE_SIDECAR_B_URL')

export const CHAIN_A_BLOCKSCOUT = 'http://localhost:19000'
export const CHAIN_B_BLOCKSCOUT = 'http://localhost:29000'

export const CHAIN_A_BRIDGE_ADDRESS =
  env.VITE_CHAIN_A_BRIDGE_ADDRESS || env.VITE_BRIDGE_ADDRESS || ''
export const CHAIN_B_BRIDGE_ADDRESS =
  env.VITE_CHAIN_B_BRIDGE_ADDRESS || env.VITE_BRIDGE_ADDRESS || ''

export const CHAIN_A_TOKEN_ADDRESS =
  env.VITE_CHAIN_A_TOKEN_ADDRESS || env.VITE_TOKEN_ADDRESS || ''
export const CHAIN_B_TOKEN_ADDRESS =
  env.VITE_CHAIN_B_TOKEN_ADDRESS || env.VITE_TOKEN_ADDRESS || ''

export const CHAIN_A_PRIVATE_KEY = normalizePrivateKey(
  env.VITE_CHAIN_A_PRIVATE_KEY || env.VITE_WALLET_PRIVATE_KEY
)
export const CHAIN_B_PRIVATE_KEY = normalizePrivateKey(
  env.VITE_CHAIN_B_PRIVATE_KEY || env.VITE_WALLET_PRIVATE_KEY
)
