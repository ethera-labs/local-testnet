/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_CHAIN_A_ID: string
  readonly VITE_CHAIN_B_ID: string
  readonly VITE_CHAIN_A_RPC: string
  readonly VITE_CHAIN_B_RPC: string
  readonly VITE_SIDECAR_A_URL: string
  readonly VITE_SIDECAR_B_URL: string
  readonly VITE_CHAIN_A_BRIDGE_ADDRESS?: string
  readonly VITE_CHAIN_B_BRIDGE_ADDRESS?: string
  readonly VITE_BRIDGE_ADDRESS?: string
  readonly VITE_CHAIN_A_TOKEN_ADDRESS?: string
  readonly VITE_CHAIN_B_TOKEN_ADDRESS?: string
  readonly VITE_TOKEN_ADDRESS?: string
  readonly VITE_CHAIN_A_PRIVATE_KEY?: string
  readonly VITE_CHAIN_B_PRIVATE_KEY?: string
  readonly VITE_WALLET_PRIVATE_KEY?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
