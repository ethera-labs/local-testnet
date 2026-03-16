import { useState } from 'react'
import type { FlowMode } from '../visualization/TransactionFlowPanel'
import { useTransactionStore } from '../../stores/transactionStore'
import { ethers } from 'ethers'
import {
  CHAIN_A_ID,
  CHAIN_B_ID,
  buildMintTx,
  getSigner,
  getTokenAddress,
  parseAmount,
  waitForTransactionReceipt,
} from '../../api/rollup'
import BalanceDisplay from '../wallet/BalanceDisplay'

interface MintFormProps {
  onSelectFlow?: (mode: FlowMode) => void
}

export default function MintForm({ onSelectFlow: _onSelectFlow }: MintFormProps) {
  const [amountA, setAmountA] = useState('')
  const [amountB, setAmountB] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { addTransaction, updateTransaction, setFlowStep } =
    useTransactionStore()

  const submitMint = async (chain: 'A' | 'B', amount: string) => {
    const signer = getSigner(chain)
    const toAddress = await signer.getAddress()
    const provider = signer.provider as ethers.JsonRpcProvider | null
    if (!provider) throw new Error(`Missing provider for chain ${chain}`)

    const expectedChainId = chain === 'A' ? CHAIN_A_ID : CHAIN_B_ID
    const network = await provider.getNetwork()
    if (Number(network.chainId) !== expectedChainId) {
      throw new Error(
        `RPC chainId mismatch for chain ${chain}: got ${network.chainId}, expected ${expectedChainId}`
      )
    }

    const signedTx = await buildMintTx(
      getTokenAddress(chain),
      toAddress,
      parseAmount(amount),
      signer,
      expectedChainId
    )

    const response = await provider.broadcastTransaction(signedTx)
    addTransaction({
      instanceId: response.hash,
      type: 'mint',
      status: 'pending',
      chainId: expectedChainId,
      createdAt: new Date(),
    })

    try {
      const receipt = await waitForTransactionReceipt(provider, response.hash, {
        timeoutMs: 60000,
        pollIntervalMs: 600,
        maxNotFoundRetries: 10,
      })
      updateTransaction(response.hash, {
        status: receipt.status === 1 ? 'committed' : 'aborted',
        decision: receipt.status === 1,
        decidedAt: new Date(),
      })
    } catch (err) {
      updateTransaction(response.hash, {
        status: 'aborted',
        decision: false,
        decidedAt: new Date(),
      })
      throw err
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      if (!amountA && !amountB) throw new Error('Enter at least one amount')

      const mintStep =
        amountA && amountB ? 'minting_both' : amountA ? 'minting_a' : 'minting_b'
      setFlowStep(mintStep)

      if (amountA) await submitMint('A', amountA)
      if (amountB) await submitMint('B', amountB)

      setFlowStep('complete')
      setTimeout(() => setFlowStep('idle'), 800)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setFlowStep('idle')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      {/* Chain A */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="font-display text-[10px] tracking-[0.25em] uppercase text-text-secondary">
            Chain A <span className="text-text-dim">/ {CHAIN_A_ID}</span>
          </label>
          <BalanceDisplay chain="A" />
        </div>
        <div className="relative">
          <input
            type="text"
            value={amountA}
            onChange={(e) => setAmountA(e.target.value)}
            placeholder="0.000"
            className="w-full bg-bg px-3 py-2.5 border border-border text-text-primary text-sm font-mono
              placeholder:text-text-dim focus:outline-none focus:border-cyan transition-colors"
          />
          <span className="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] text-text-dim font-display tracking-widest uppercase">
            tokens
          </span>
        </div>
      </div>

      {/* Chain B */}
      <div>
        <div className="flex items-center justify-between mb-2">
          <label className="font-display text-[10px] tracking-[0.25em] uppercase text-text-secondary">
            Chain B <span className="text-text-dim">/ {CHAIN_B_ID}</span>
          </label>
          <BalanceDisplay chain="B" />
        </div>
        <div className="relative">
          <input
            type="text"
            value={amountB}
            onChange={(e) => setAmountB(e.target.value)}
            placeholder="0.000"
            className="w-full bg-bg px-3 py-2.5 border border-border text-text-primary text-sm font-mono
              placeholder:text-text-dim focus:outline-none focus:border-cyan transition-colors"
          />
          <span className="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] text-text-dim font-display tracking-widest uppercase">
            tokens
          </span>
        </div>
      </div>

      {error && (
        <div className="border border-error/40 bg-error/5 px-3 py-2.5 text-error text-xs font-mono">
          <span className="text-error/60 mr-2">!</span>{error}
        </div>
      )}

      <button
        type="submit"
        disabled={loading || (!amountA && !amountB)}
        className={`w-full py-3 font-display text-[11px] tracking-[0.3em] uppercase border transition-all ${
          loading || (!amountA && !amountB)
            ? 'border-border text-text-dim cursor-not-allowed'
            : 'border-cyan text-cyan hover:bg-cyan hover:text-bg glow-cyan'
        }`}
      >
        {loading ? (
          <span className="flex items-center justify-center gap-2">
            <span className="w-1 h-1 bg-current rounded-full indicator-active" />
            <span className="w-1 h-1 bg-current rounded-full indicator-active" style={{ animationDelay: '0.3s' }} />
            <span className="w-1 h-1 bg-current rounded-full indicator-active" style={{ animationDelay: '0.6s' }} />
            <span className="ml-2">Broadcasting</span>
          </span>
        ) : (
          'Submit Mint'
        )}
      </button>
    </form>
  )
}
