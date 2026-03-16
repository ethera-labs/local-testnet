import { useState } from 'react'
import { ethers } from 'ethers'
import {
  CHAIN_A_ID,
  CHAIN_B_ID,
  buildBridgeSendTx,
  buildBridgeReceiveTx,
  buildNativeTransferTx,
  generateSessionId,
  getBridgeAddress,
  getSigner,
  getTokenAddress,
  parseAmount,
  getProvider,
  waitForTransactionReceipt,
} from '../../api/rollup'
import { submitXT, waitForDecision } from '../../api/sidecar'
import { useTransactionStore } from '../../stores/transactionStore'

type StressTestId = 'burst-bridge' | 'bidirectional' | 'mixed-normal-xt' | 'half-wrong-nonce'

interface TestProgress {
  running: boolean
  submitted: number
  total: number
  committed: number
  aborted: number
  errors: number
}

const emptyProgress = (): TestProgress => ({
  running: false, submitted: 0, total: 0, committed: 0, aborted: 0, errors: 0,
})

interface StressTest {
  id: StressTestId
  label: string
  description: string
  detail: string
}

const STRESS_TESTS: StressTest[] = [
  {
    id: 'burst-bridge',
    label: 'Burst Bridge A→B',
    description: 'N sequential bridge XTs from the same account with incremented nonces',
    detail: 'Each XT: bridge.send(A) + bridge.receive(B). Nonces offset per iteration.',
  },
  {
    id: 'bidirectional',
    label: 'Bidirectional Burst',
    description: 'N rounds of A→B immediately followed by B→A, interleaving nonces on both chains',
    detail: 'Pair i: A→B uses nonce 2i, B→A uses nonce 2i+1 on both chains. Net token balance unchanged.',
  },
  {
    id: 'mixed-normal-xt',
    label: 'Mixed Normal + XT',
    description: 'N rounds: native ETH self-transfer on A (awaits receipt), then bridge XT A→B',
    detail: 'Native is mined before bridge so builder pool has it. Nonces: native then bridge per round.',
  },
  {
    id: 'half-wrong-nonce',
    label: 'Half Wrong Nonce',
    description: 'N/2 valid bridge XTs interleaved with N/2 stale-nonce XTs — only valid ones should commit',
    detail: 'Wrong nonces are set below current nonce and will be rejected. Valid XTs proceed normally.',
  },
]

export default function StressForm() {
  const [count, setCount] = useState('5')
  const [delayMs, setDelayMs] = useState('100')
  const [amount, setAmount] = useState('0.1')
  const [progress, setProgress] = useState<Record<StressTestId, TestProgress>>(
    () => Object.fromEntries(STRESS_TESTS.map(t => [t.id, emptyProgress()])) as Record<StressTestId, TestProgress>
  )
  const [errors, setErrors] = useState<Record<StressTestId, string | null>>(
    () => Object.fromEntries(STRESS_TESTS.map(t => [t.id, null])) as Record<StressTestId, string | null>
  )

  const { addTransaction, updateTransaction } = useTransactionStore()

  const trackXT = (
    instanceId: string,
    txHashA: string,
    txHashB: string,
    testId: StressTestId
  ) => {
    waitForDecision(instanceId, 60000).then(async decision => {
      if (decision) {
        try {
          const providerA = getProvider('A')
          const providerB = getProvider('B')
          const [receiptA, receiptB] = await Promise.all([
            waitForTransactionReceipt(providerA, txHashA, { timeoutMs: 60000, maxNotFoundRetries: 30 }),
            waitForTransactionReceipt(providerB, txHashB, { timeoutMs: 60000, maxNotFoundRetries: 30 }),
          ])
          updateTransaction(txHashA, { status: receiptA.status === 1 ? 'committed' : 'aborted', decision: receiptA.status === 1, decidedAt: new Date() })
          updateTransaction(txHashB, { status: receiptB.status === 1 ? 'committed' : 'aborted', decision: receiptB.status === 1, decidedAt: new Date() })
          setProgress(prev => ({
            ...prev,
            [testId]: { ...prev[testId], committed: prev[testId].committed + 1 },
          }))
        } catch {
          updateTransaction(txHashA, { status: 'aborted', decision: false, decidedAt: new Date() })
          updateTransaction(txHashB, { status: 'aborted', decision: false, decidedAt: new Date() })
          setProgress(prev => ({
            ...prev,
            [testId]: { ...prev[testId], aborted: prev[testId].aborted + 1 },
          }))
        }
      } else {
        updateTransaction(txHashA, { status: 'aborted', decision: false, decidedAt: new Date() })
        updateTransaction(txHashB, { status: 'aborted', decision: false, decidedAt: new Date() })
        setProgress(prev => ({
          ...prev,
          [testId]: { ...prev[testId], aborted: prev[testId].aborted + 1 },
        }))
      }
    }).catch(() => {
      setProgress(prev => ({
        ...prev,
        [testId]: { ...prev[testId], errors: prev[testId].errors + 1 },
      }))
    })
  }

  const submitBridgeXT = async (
    testId: StressTestId,
    signerA: ethers.Wallet,
    signerB: ethers.Wallet,
    senderA: string,
    senderB: string,
    bridgeA: string,
    bridgeB: string,
    tokenA: string,
    bridgeAmount: bigint,
    sessionId: string,
    nonceA: number,
    nonceB: number
  ) => {
    const txABytes = await buildBridgeSendTx(
      bridgeA, CHAIN_B_ID, tokenA, senderA, senderB,
      bridgeAmount, sessionId, bridgeB, signerA, CHAIN_A_ID, nonceA
    )
    const txBBytes = await buildBridgeReceiveTx(
      bridgeB, CHAIN_A_ID, senderA, senderB, sessionId, bridgeA,
      signerB, CHAIN_B_ID, nonceB
    )
    const response = await submitXT({ [CHAIN_A_ID]: [txABytes], [CHAIN_B_ID]: [txBBytes] })
    const txHashA = ethers.Transaction.from(txABytes).hash!
    const txHashB = ethers.Transaction.from(txBBytes).hash!
    addTransaction({ instanceId: txHashA, type: 'scenario', status: 'pending', chainId: CHAIN_A_ID, createdAt: new Date() })
    addTransaction({ instanceId: txHashB, type: 'scenario', status: 'pending', chainId: CHAIN_B_ID, createdAt: new Date() })
    setProgress(prev => ({ ...prev, [testId]: { ...prev[testId], submitted: prev[testId].submitted + 1 } }))
    trackXT(response.instance_id, txHashA, txHashB, testId)
  }

  const submitBToAXT = async (
    testId: StressTestId,
    signerA: ethers.Wallet,
    signerB: ethers.Wallet,
    senderA: string,
    senderB: string,
    bridgeA: string,
    bridgeB: string,
    tokenB: string,
    bridgeAmount: bigint,
    sessionId: string,
    nonceA: number,
    nonceB: number
  ) => {
    const txBBytes = await buildBridgeSendTx(
      bridgeB, CHAIN_A_ID, tokenB, senderB, senderA,
      bridgeAmount, sessionId, bridgeA, signerB, CHAIN_B_ID, nonceB
    )
    const txABytes = await buildBridgeReceiveTx(
      bridgeA, CHAIN_B_ID, senderB, senderA, sessionId, bridgeB,
      signerA, CHAIN_A_ID, nonceA
    )
    const response = await submitXT({ [CHAIN_A_ID]: [txABytes], [CHAIN_B_ID]: [txBBytes] })
    const txHashA = ethers.Transaction.from(txABytes).hash!
    const txHashB = ethers.Transaction.from(txBBytes).hash!
    addTransaction({ instanceId: txHashA, type: 'scenario', status: 'pending', chainId: CHAIN_A_ID, createdAt: new Date() })
    addTransaction({ instanceId: txHashB, type: 'scenario', status: 'pending', chainId: CHAIN_B_ID, createdAt: new Date() })
    setProgress(prev => ({ ...prev, [testId]: { ...prev[testId], submitted: prev[testId].submitted + 1 } }))
    trackXT(response.instance_id, txHashA, txHashB, testId)
  }

  const runBurstBridge = async (N: number, delay: number, bridgeAmount: bigint) => {
    const id: StressTestId = 'burst-bridge'
    const signerA = getSigner('A')
    const signerB = getSigner('B')
    const senderA = await signerA.getAddress()
    const senderB = await signerB.getAddress()
    const bridgeA = getBridgeAddress('A')
    const bridgeB = getBridgeAddress('B')
    const tokenA = getTokenAddress('A')
    const providerA = getProvider('A')
    const providerB = getProvider('B')
    const baseNonceA = await providerA.getTransactionCount(senderA, 'pending')
    const baseNonceB = await providerB.getTransactionCount(senderB, 'pending')

    for (let i = 0; i < N; i++) {
      await submitBridgeXT(
        id, signerA, signerB, senderA, senderB, bridgeA, bridgeB, tokenA,
        bridgeAmount, generateSessionId(), baseNonceA + i, baseNonceB + i
      )
      if (i < N - 1) await new Promise(r => setTimeout(r, delay))
    }
  }

  const runBidirectional = async (N: number, delay: number, bridgeAmount: bigint) => {
    const id: StressTestId = 'bidirectional'
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

    for (let i = 0; i < N; i++) {
      // A→B
      await submitBridgeXT(
        id, signerA, signerB, senderA, senderB, bridgeA, bridgeB, tokenA,
        bridgeAmount, generateSessionId(), baseNonceA + 2 * i, baseNonceB + 2 * i
      )
      if (delay > 0) await new Promise(r => setTimeout(r, delay))
      // B→A
      await submitBToAXT(
        id, signerA, signerB, senderA, senderB, bridgeA, bridgeB, tokenB,
        bridgeAmount, generateSessionId(), baseNonceA + 2 * i + 1, baseNonceB + 2 * i + 1
      )
      if (i < N - 1 && delay > 0) await new Promise(r => setTimeout(r, delay))
    }
  }

  const runMixedNormalXT = async (N: number, delay: number, bridgeAmount: bigint) => {
    const id: StressTestId = 'mixed-normal-xt'
    const signerA = getSigner('A')
    const signerB = getSigner('B')
    const senderA = await signerA.getAddress()
    const senderB = await signerB.getAddress()
    const bridgeA = getBridgeAddress('A')
    const bridgeB = getBridgeAddress('B')
    const tokenA = getTokenAddress('A')
    const providerA = getProvider('A')
    const providerB = getProvider('B')
    let nonceA = await providerA.getTransactionCount(senderA, 'pending')
    let nonceB = await providerB.getTransactionCount(senderB, 'pending')

    for (let i = 0; i < N; i++) {
      // Native self-transfer on A — must be mined before bridge so builder pool has it
      const nativeTxBytes = await buildNativeTransferTx(
        senderA, ethers.parseEther('0.1'), signerA, CHAIN_A_ID, nonceA
      )
      const nativeResponse = await providerA.broadcastTransaction(nativeTxBytes)
      addTransaction({ instanceId: nativeResponse.hash, type: 'native', status: 'pending', chainId: CHAIN_A_ID, createdAt: new Date() })

      let receipt
      try {
        receipt = await waitForTransactionReceipt(providerA, nativeResponse.hash, { timeoutMs: 30000 })
      } catch {
        updateTransaction(nativeResponse.hash, { status: 'aborted', decision: false, decidedAt: new Date() })
        throw new Error(`Native tx ${nativeResponse.hash} did not confirm in time`)
      }
      updateTransaction(nativeResponse.hash, {
        status: receipt.status === 1 ? 'committed' : 'aborted',
        decision: receipt.status === 1,
        decidedAt: new Date(),
      })
      nonceA += 1

      if (delay > 0) await new Promise(r => setTimeout(r, delay))

      // Bridge XT — uses next nonce on A (native just mined), sequential on B
      await submitBridgeXT(
        id, signerA, signerB, senderA, senderB, bridgeA, bridgeB, tokenA,
        bridgeAmount, generateSessionId(), nonceA, nonceB
      )
      nonceA += 1
      nonceB += 1

      if (i < N - 1 && delay > 0) await new Promise(r => setTimeout(r, delay))
    }
  }

  const runHalfWrongNonce = async (N: number, delay: number, bridgeAmount: bigint) => {
    const id: StressTestId = 'half-wrong-nonce'
    const half = Math.max(1, Math.floor(N / 2))
    const signerA = getSigner('A')
    const signerB = getSigner('B')
    const senderA = await signerA.getAddress()
    const senderB = await signerB.getAddress()
    const bridgeA = getBridgeAddress('A')
    const bridgeB = getBridgeAddress('B')
    const tokenA = getTokenAddress('A')
    const providerA = getProvider('A')
    const providerB = getProvider('B')
    const baseNonceA = await providerA.getTransactionCount(senderA, 'pending')
    const baseNonceB = await providerB.getTransactionCount(senderB, 'pending')

    for (let i = 0; i < half; i++) {
      // Valid XT
      await submitBridgeXT(
        id, signerA, signerB, senderA, senderB, bridgeA, bridgeB, tokenA,
        bridgeAmount, generateSessionId(), baseNonceA + i, baseNonceB + i
      )
      if (delay > 0) await new Promise(r => setTimeout(r, delay))

      // Wrong-nonce XT (nonce below base — definitely stale)
      const wrongNonceA = Math.max(0, baseNonceA - 1 - i)
      const wrongNonceB = Math.max(0, baseNonceB - 1 - i)
      await submitBridgeXT(
        id, signerA, signerB, senderA, senderB, bridgeA, bridgeB, tokenA,
        bridgeAmount, generateSessionId(), wrongNonceA, wrongNonceB
      )
      if (i < half - 1 && delay > 0) await new Promise(r => setTimeout(r, delay))
    }
  }

  const runTest = async (id: StressTestId) => {
    const N = Math.max(1, Math.min(50, parseInt(count, 10) || 5))
    const delay = Math.max(0, parseInt(delayMs, 10) || 100)
    const bridgeAmount = parseAmount(amount || '0.1')

    const total = id === 'half-wrong-nonce' ? Math.floor(N / 2) * 2 : id === 'bidirectional' ? N * 2 : N
    setProgress(prev => ({ ...prev, [id]: { running: true, submitted: 0, total, committed: 0, aborted: 0, errors: 0 } }))
    setErrors(prev => ({ ...prev, [id]: null }))

    try {
      if (id === 'burst-bridge') await runBurstBridge(N, delay, bridgeAmount)
      else if (id === 'bidirectional') await runBidirectional(N, delay, bridgeAmount)
      else if (id === 'mixed-normal-xt') await runMixedNormalXT(N, delay, bridgeAmount)
      else if (id === 'half-wrong-nonce') await runHalfWrongNonce(N, delay, bridgeAmount)
    } catch (err) {
      setErrors(prev => ({ ...prev, [id]: err instanceof Error ? err.message : 'Unknown error' }))
    } finally {
      setProgress(prev => ({ ...prev, [id]: { ...prev[id], running: false } }))
    }
  }

  return (
    <div className="space-y-5">
      {/* Config */}
      <div className="grid grid-cols-3 gap-3">
        <div>
          <label className="font-display text-[9px] tracking-[0.25em] uppercase text-text-dim block mb-1.5">
            Count
          </label>
          <input
            type="number"
            min={1}
            max={50}
            value={count}
            onChange={e => setCount(e.target.value)}
            className="w-full bg-bg px-2.5 py-2 border border-border text-text-primary text-sm font-mono focus:outline-none focus:border-error/60 transition-colors"
          />
        </div>
        <div>
          <label className="font-display text-[9px] tracking-[0.25em] uppercase text-text-dim block mb-1.5">
            Delay (ms)
          </label>
          <input
            type="number"
            min={0}
            value={delayMs}
            onChange={e => setDelayMs(e.target.value)}
            className="w-full bg-bg px-2.5 py-2 border border-border text-text-primary text-sm font-mono focus:outline-none focus:border-error/60 transition-colors"
          />
        </div>
        <div>
          <label className="font-display text-[9px] tracking-[0.25em] uppercase text-text-dim block mb-1.5">
            Amount
          </label>
          <div className="relative">
            <input
              type="text"
              value={amount}
              onChange={e => setAmount(e.target.value)}
              className="w-full bg-bg px-2.5 py-2 border border-border text-text-primary text-sm font-mono focus:outline-none focus:border-error/60 transition-colors pr-8"
            />
            <span className="absolute right-2 top-1/2 -translate-y-1/2 text-[9px] text-text-dim font-display uppercase">
              TKN
            </span>
          </div>
        </div>
      </div>

      {/* Test cards */}
      <div className="space-y-2.5">
        {STRESS_TESTS.map(test => {
          const p = progress[test.id]
          const err = errors[test.id]
          const pct = p.total > 0 ? (p.submitted / p.total) * 100 : 0
          const decidedTotal = p.committed + p.aborted + p.errors
          const anyRunning = Object.values(progress).some(q => q.running)

          return (
            <div
              key={test.id}
              className="border border-border bg-bg p-3.5"
            >
              <div className="flex items-start justify-between gap-3 mb-2.5">
                <div className="min-w-0">
                  <p className="font-display text-[11px] tracking-wide text-text-primary mb-0.5">
                    {test.label}
                  </p>
                  <p className="text-[10px] text-text-dim font-mono leading-relaxed">{test.description}</p>
                </div>
                <button
                  onClick={() => runTest(test.id)}
                  disabled={anyRunning}
                  className={`flex-none px-3 py-1.5 font-display text-[9px] tracking-[0.2em] uppercase border transition-all ${
                    anyRunning
                      ? 'border-border text-text-dim cursor-not-allowed'
                      : 'border-error text-error hover:bg-error hover:text-bg'
                  }`}
                >
                  {p.running ? (
                    <span className="flex items-center gap-1.5">
                      <span className="w-1 h-1 bg-current rounded-full indicator-active" />
                      <span className="w-1 h-1 bg-current rounded-full indicator-active" style={{ animationDelay: '0.3s' }} />
                      <span className="w-1 h-1 bg-current rounded-full indicator-active" style={{ animationDelay: '0.6s' }} />
                    </span>
                  ) : 'Run'}
                </button>
              </div>

              {/* Progress */}
              {(p.submitted > 0 || p.running) && (
                <div className="space-y-2">
                  {/* Progress bar */}
                  <div className="h-0.5 bg-border w-full">
                    <div
                      className="h-0.5 bg-error/60 transition-all duration-300"
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  {/* Counters */}
                  <div className="flex items-center gap-3 text-[9px] font-mono">
                    <span className="text-text-dim">
                      <span className="text-text-secondary">{p.submitted}</span>/{p.total} submitted
                    </span>
                    {p.committed > 0 && (
                      <span className="text-cyan">{p.committed} committed</span>
                    )}
                    {p.aborted > 0 && (
                      <span className="text-amber">{p.aborted} aborted</span>
                    )}
                    {p.errors > 0 && (
                      <span className="text-error">{p.errors} errors</span>
                    )}
                    {!p.running && decidedTotal === p.submitted && p.submitted > 0 && (
                      <span className="ml-auto text-text-dim">done</span>
                    )}
                  </div>
                </div>
              )}

              {/* Detail / error */}
              {err ? (
                <p className="mt-2 text-[9px] font-mono text-error border-l border-error/40 pl-2">{err}</p>
              ) : p.submitted === 0 && !p.running && (
                <p className="text-[9px] text-text-dim font-mono border-l border-border pl-2 mt-1">{test.detail}</p>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
