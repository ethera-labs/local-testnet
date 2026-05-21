import {
  BUNDLER_A_URL,
  BUNDLER_B_URL,
  ENTRYPOINT_A,
  ENTRYPOINT_B,
  SIMPLE_ACCOUNT_FACTORY_A,
  SIMPLE_ACCOUNT_FACTORY_B,
} from '../config/chains'

export type Chain = 'A' | 'B'

export interface UserOpV07 {
  sender: string
  nonce: string
  factory?: string
  factoryData?: string
  callData: string
  callGasLimit: string
  verificationGasLimit: string
  preVerificationGas: string
  maxFeePerGas: string
  maxPriorityFeePerGas: string
  paymaster?: string
  paymasterVerificationGasLimit?: string
  paymasterPostOpGasLimit?: string
  paymasterData?: string
  signature: string
}

export interface SignedTxResp {
  raw: string
  hash: string
  to: string
  chainId: number
  gas: string
  maxFeePerGas: string
  maxPriorityFeePerGas: string
  userOpHashes: string[]
}

export interface BundlerErrorData {
  reason?: string
  opIndex?: number
  [key: string]: unknown
}

export class BundlerError extends Error {
  constructor(
    public readonly code: number,
    message: string,
    public readonly data?: BundlerErrorData
  ) {
    super(message)
    this.name = 'BundlerError'
  }
}

export function getBundlerUrl(chain: Chain): string {
  const url = chain === 'A' ? BUNDLER_A_URL : BUNDLER_B_URL
  if (!url) {
    throw new Error(
      `Bundler URL missing for chain ${chain}. Set VITE_BUNDLER_${chain}_URL.`
    )
  }
  return url
}

export function getEntryPointAddress(chain: Chain): string {
  const addr = chain === 'A' ? ENTRYPOINT_A : ENTRYPOINT_B
  if (!addr) {
    throw new Error(
      `EntryPoint address missing for chain ${chain}. Set VITE_ENTRYPOINT_${chain}.`
    )
  }
  return addr
}

export function getSimpleAccountFactoryAddress(chain: Chain): string {
  const addr =
    chain === 'A' ? SIMPLE_ACCOUNT_FACTORY_A : SIMPLE_ACCOUNT_FACTORY_B
  if (!addr) {
    throw new Error(
      `SimpleAccountFactory address missing for chain ${chain}. ` +
        `Set VITE_SIMPLE_ACCOUNT_FACTORY_${chain}.`
    )
  }
  return addr
}

interface JsonRpcResponse<T> {
  jsonrpc: '2.0'
  id: number
  result?: T
  error?: { code: number; message: string; data?: BundlerErrorData }
}

let jsonRpcId = 0

// buildSignedUserOpsTx calls `ethera_buildSignedUserOpsTx` against the bundler
// for the given chain. The bundler validates each op, runs simulateValidation,
// builds and signs the outer EIP-1559 handleOps transaction, and returns the
// raw bytes ready for `eth_sendRawTransaction`.
export async function buildSignedUserOpsTx(
  chain: Chain,
  userOps: UserOpV07[],
  chainId: number
): Promise<SignedTxResp> {
  const url = getBundlerUrl(chain)
  const body = {
    jsonrpc: '2.0' as const,
    id: ++jsonRpcId,
    method: 'ethera_buildSignedUserOpsTx',
    params: [userOps, { chainId }],
  }

  const response = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (!response.ok) {
    const text = await response.text()
    throw new Error(`Bundler HTTP ${response.status}: ${text}`)
  }

  const payload = (await response.json()) as JsonRpcResponse<SignedTxResp>
  if (payload.error) {
    throw new BundlerError(
      payload.error.code,
      payload.error.message,
      payload.error.data
    )
  }
  if (!payload.result) {
    throw new Error('Bundler returned an empty result')
  }
  return payload.result
}
