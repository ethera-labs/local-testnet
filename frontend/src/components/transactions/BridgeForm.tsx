import { useState } from 'react'
import { ethers } from 'ethers'
import type { FlowMode } from '../visualization/TransactionFlowPanel'
import { useTransactionStore } from '../../stores/transactionStore'
import { submitXT, waitForDecision } from '../../api/sidecar'
import {
  CHAIN_A_ID,
  CHAIN_B_ID,
  buildApproveTx,
  buildBridgeERC20ToTx,
  buildBridgeReceiveTokensTx,
  generateSessionId,
  getBridgeAddress,
  getSigner,
  getTokenAddress,
  parseAmount,
  getProvider,
  waitForTransactionReceipt,
} from '../../api/rollup'
import BalanceDisplay from '../wallet/BalanceDisplay'

interface BridgeFormProps {
  onSubmit: (instanceId: string) => void
  onSelectFlow?: (mode: FlowMode) => void
}

type Direction = 'a_to_b' | 'b_to_a'

export default function BridgeForm({ onSubmit, onSelectFlow }: BridgeFormProps) {
  const [amount, setAmount] = useState('')
  const [direction, setDirection] = useState<Direction>('a_to_b')
  const [repeatCount, setRepeatCount] = useState('1')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { addTransaction, updateTransaction, setFlowStep } = useTransactionStore()

  const sourceChain = direction === 'a_to_b' ? CHAIN_A_ID : CHAIN_B_ID
  const destChain = direction === 'a_to_b' ? CHAIN_B_ID : CHAIN_A_ID

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)
    setLoading(true)

    try {
      if (!amount) throw new Error('Enter an amount')

      const repeat = Math.max(1, Number.parseInt(repeatCount, 10) || 1)
      const parsedAmount = parseAmount(amount)

      onSelectFlow?.('xt')
      setFlowStep('submitting')

      const signerA = getSigner('A')
      const signerB = getSigner('B')
      const senderA = await signerA.getAddress()
      const senderB = await signerB.getAddress()
      const bridgeA = getBridgeAddress('A')
      const bridgeB = getBridgeAddress('B')
      const tokenA = getTokenAddress('A')
      const tokenB = getTokenAddress('B')
      const providerA = getProvider('A')
      const providerB = getProvider('B')

      const baseNonceA = await providerA.getTransactionCount(senderA, 'pending')
      const baseNonceB = await providerB.getTransactionCount(senderB, 'pending')

      const trackXT = (
        instanceId: string,
        txHashA: string,
        txHashB: string,
        animateFlow: boolean
      ) => {
        if (animateFlow) {
          const runFlow = async () => {
            setFlowStep('forward_to_peer')
            await new Promise(r => setTimeout(r, 500))
            setFlowStep('lock_builder_a')
            await new Promise(r => setTimeout(r, 300))
            setFlowStep('lock_builder_b')
            await new Promise(r => setTimeout(r, 300))
            setFlowStep('simulating_a')
            await new Promise(r => setTimeout(r, 400))
            setFlowStep('simulating_b')
            await new Promise(r => setTimeout(r, 400))
            setFlowStep('circ_exchange')
            await new Promise(r => setTimeout(r, 500))
            setFlowStep('voting')
            await new Promise(r => setTimeout(r, 600))
          }
          runFlow()
        }

        waitForDecision(instanceId, 60000).then(async (decision) => {
          if (animateFlow) {
            setFlowStep('decided')
            await new Promise(r => setTimeout(r, 300))
          }

          if (decision) {
            if (animateFlow) {
              setFlowStep('delivering')
              await new Promise(r => setTimeout(r, 400))
              setFlowStep('confirming')
            }

            try {
              const [receiptA, receiptB] = await Promise.all([
                waitForTransactionReceipt(providerA, txHashA, { timeoutMs: 30000 }),
                waitForTransactionReceipt(providerB, txHashB, { timeoutMs: 30000 }),
              ])
              updateTransaction(txHashA, {
                status: receiptA.status === 1 ? 'committed' : 'aborted',
                decision: receiptA.status === 1,
                decidedAt: new Date(),
              })
              updateTransaction(txHashB, {
                status: receiptB.status === 1 ? 'committed' : 'aborted',
                decision: receiptB.status === 1,
                decidedAt: new Date(),
              })
              if (animateFlow) {
                setFlowStep('complete')
                setTimeout(() => setFlowStep('idle'), 1000)
              }
            } catch (err) {
              console.error('Error waiting for receipts:', err)
              updateTransaction(txHashA, { status: 'aborted', decision: false, decidedAt: new Date() })
              updateTransaction(txHashB, { status: 'aborted', decision: false, decidedAt: new Date() })
              if (animateFlow) setFlowStep('idle')
            }
          } else {
            updateTransaction(txHashA, { status: 'aborted', decision: false, decidedAt: new Date() })
            updateTransaction(txHashB, { status: 'aborted', decision: false, decidedAt: new Date() })
            if (animateFlow) setFlowStep('idle')
          }
        }).catch(err => {
          console.error('Error waiting for decision:', err)
          updateTransaction(txHashA, { status: 'aborted', decision: false, decidedAt: new Date() })
          updateTransaction(txHashB, { status: 'aborted', decision: false, decidedAt: new Date() })
          if (animateFlow) setFlowStep('idle')
        })
      }

      for (let i = 0; i < repeat; i += 1) {
        const sessionId = generateSessionId()
        const nonceA = direction === 'a_to_b' ? baseNonceA + i * 2 : baseNonceA + i
        const nonceB = direction === 'a_to_b' ? baseNonceB + i : baseNonceB + i * 2
        const transactions: Record<number, string[]> = {}
        let txABytes: string
        let txBBytes: string

        if (direction === 'a_to_b') {
          const txAApprove = await buildApproveTx(tokenA, bridgeA, parsedAmount, signerA, CHAIN_A_ID, nonceA)
          const txABridge = await buildBridgeERC20ToTx(bridgeA, CHAIN_B_ID, tokenA, parsedAmount, senderB, sessionId, signerA, CHAIN_A_ID, nonceA + 1)
          const txBReceive = await buildBridgeReceiveTokensTx(bridgeB, sessionId, CHAIN_A_ID, CHAIN_B_ID, bridgeA, senderB, signerB, CHAIN_B_ID, nonceB)
          transactions[CHAIN_A_ID] = [txAApprove, txABridge]
          transactions[CHAIN_B_ID] = [txBReceive]
          txABytes = txABridge
          txBBytes = txBReceive
        } else {
          const txBApprove = await buildApproveTx(tokenB, bridgeB, parsedAmount, signerB, CHAIN_B_ID, nonceB)
          const txBBridge = await buildBridgeERC20ToTx(bridgeB, CHAIN_A_ID, tokenB, parsedAmount, senderA, sessionId, signerB, CHAIN_B_ID, nonceB + 1)
          const txAReceive = await buildBridgeReceiveTokensTx(bridgeA, sessionId, CHAIN_B_ID, CHAIN_A_ID, bridgeB, senderA, signerA, CHAIN_A_ID, nonceA)
          transactions[CHAIN_B_ID] = [txBApprove, txBBridge]
          transactions[CHAIN_A_ID] = [txAReceive]
          txABytes = txAReceive
          txBBytes = txBBridge
        }

        const txHashA = ethers.Transaction.from(txABytes).hash!
        const txHashB = ethers.Transaction.from(txBBytes).hash!

        const response = await submitXT(transactions)
        const instanceId = response.instance_id

        addTransaction({ instanceId: txHashA, type: 'bridge', status: 'pending', chainId: CHAIN_A_ID, createdAt: new Date() })
        addTransaction({ instanceId: txHashB, type: 'bridge', status: 'pending', chainId: CHAIN_B_ID, createdAt: new Date() })

        trackXT(instanceId, txHashA, txHashB, repeat === 1)
        if (repeat === 1) onSubmit(instanceId)
      }

      if (repeat > 1) setFlowStep('idle')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setFlowStep('idle')
    } finally {
      setLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-5">
      <div className="bg-bg border border-border px-3 py-3 space-y-2">
        <div className="flex items-center justify-between">
          <span className="font-display text-[10px] tracking-[0.25em] uppercase text-text-secondary">
            Chain A Balance
          </span>
          <BalanceDisplay chain="A" remoteChain="B" />
        </div>
        <div className="h-px bg-border" />
        <div className="flex items-center justify-between">
          <span className="font-display text-[10px] tracking-[0.25em] uppercase text-text-secondary">
            Chain B Balance
          </span>
          <BalanceDisplay chain="B" remoteChain="A" />
        </div>
      </div>

      <div>
        <label className="font-display text-[10px] tracking-[0.25em] uppercase text-text-secondary block mb-2">
          Direction
        </label>
        <div className="grid grid-cols-2 gap-2">
          <button
            type="button"
            onClick={() => setDirection('a_to_b')}
            className={`py-2.5 font-display text-[11px] tracking-[0.2em] uppercase border transition-all ${
              direction === 'a_to_b'
                ? 'border-amber text-amber bg-amber/5'
                : 'border-border text-text-secondary hover:border-border-bright hover:text-text-primary'
            }`}
          >
            A → B
          </button>
          <button
            type="button"
            onClick={() => setDirection('b_to_a')}
            className={`py-2.5 font-display text-[11px] tracking-[0.2em] uppercase border transition-all ${
              direction === 'b_to_a'
                ? 'border-amber text-amber bg-amber/5'
                : 'border-border text-text-secondary hover:border-border-bright hover:text-text-primary'
            }`}
          >
            B → A
          </button>
        </div>
      </div>

      <div>
        <label className="font-display text-[10px] tracking-[0.25em] uppercase text-text-secondary block mb-2">
          Amount to Bridge
        </label>
        <div className="relative">
          <input
            type="text"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder="0.000"
            className="w-full bg-bg px-3 py-2.5 border border-border text-text-primary text-sm font-mono
              placeholder:text-text-dim focus:outline-none focus:border-amber transition-colors"
          />
          <span className="absolute right-3 top-1/2 -translate-y-1/2 text-[10px] text-text-dim font-display tracking-widest uppercase">
            tokens
          </span>
        </div>
      </div>

      <div>
        <label className="font-display text-[10px] tracking-[0.25em] uppercase text-text-secondary block mb-2">
          Repeat Count
        </label>
        <input
          type="number"
          min={1}
          value={repeatCount}
          onChange={(e) => setRepeatCount(e.target.value)}
          className="w-full bg-bg px-3 py-2.5 border border-border text-text-primary text-sm font-mono
            focus:outline-none focus:border-amber transition-colors"
        />
        <p className="mt-1.5 text-[10px] text-text-dim font-mono">
          One XT per request. XTs queued and processed in publisher order.
        </p>
      </div>

      <div className="bg-bg border border-border px-3 py-3">
        <div className="flex items-center justify-between text-[11px]">
          <span className="text-text-dim font-display tracking-widest uppercase">From</span>
          <span className="font-mono text-text-secondary">Chain {sourceChain}</span>
        </div>
        <div className="my-1.5 h-px bg-border" />
        <div className="flex items-center justify-between text-[11px]">
          <span className="text-text-dim font-display tracking-widest uppercase">To</span>
          <span className="font-mono text-text-secondary">Chain {destChain}</span>
        </div>
      </div>

      {error && (
        <div className="border border-error/40 bg-error/5 px-3 py-2.5 text-error text-xs font-mono">
          <span className="text-error/60 mr-2">!</span>{error}
        </div>
      )}

      <button
        type="submit"
        disabled={loading || !amount}
        className={`w-full py-3 font-display text-[11px] tracking-[0.3em] uppercase border transition-all ${
          loading || !amount
            ? 'border-border text-text-dim cursor-not-allowed'
            : 'border-amber text-amber hover:bg-amber hover:text-bg glow-amber'
        }`}
      >
        {loading ? (
          <span className="flex items-center justify-center gap-2">
            <span className="w-1 h-1 bg-current rounded-full indicator-active" />
            <span className="w-1 h-1 bg-current rounded-full indicator-active" style={{ animationDelay: '0.3s' }} />
            <span className="w-1 h-1 bg-current rounded-full indicator-active" style={{ animationDelay: '0.6s' }} />
            <span className="ml-2">Coordinating</span>
          </span>
        ) : (
          'Submit Cross-Chain XT'
        )}
      </button>
    </form>
  )
}
