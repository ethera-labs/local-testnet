import { useState } from 'react'
import { ethers } from 'ethers'
import {
  CHAIN_A_ID,
  CHAIN_B_ID,
  buildMintTx,
  buildApproveTx,
  buildBridgeERC20ToTx,
  buildBridgeEthToTx,
  buildBridgeReceiveEthTx,
  buildBridgeReceiveTokensTx,
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

type ScenarioId =
  | 'cross-chain-mint'
  | 'bridge-native-fail'
  | 'bridge-oog'
  | 'native-receive-no-send'
  | 'native-overdraft'
  | 'eth-bridge-oog'

type CategoryId = 'happy' | 'bridge' | 'eth' | 'native'

interface ScenarioDef {
  id: ScenarioId
  label: string
  description: string
  chainADesc: string
  chainBDesc: string
  expected: 'commit' | 'abort'
  category: CategoryId
}

interface CategoryDef {
  id: CategoryId | 'all'
  label: string
}

const CATEGORIES: CategoryDef[] = [
  { id: 'all', label: 'All' },
  { id: 'happy', label: 'Happy path' },
  { id: 'bridge', label: 'Bridge fails' },
  { id: 'eth', label: 'ETH bridge fails' },
  { id: 'native', label: 'Native fails' },
]

const SCENARIOS: ScenarioDef[] = [
  {
    id: 'cross-chain-mint',
    label: 'Cross-Chain Mint XT',
    description: 'Mint 1 TKN on both chains as a single atomic XT',
    chainADesc: 'token.mint(1 TKN → self)',
    chainBDesc: 'token.mint(1 TKN → self)',
    expected: 'commit',
    category: 'happy',
  },
  {
    id: 'bridge-native-fail',
    label: 'Bridge + Native Overdraft',
    description: 'Bridge send on A with no paired receive on B; B half is an impossible ETH transfer.',
    chainADesc: 'approve + bridgeERC20To(0.1 TKN → B) → no ACK → REVERT',
    chainBDesc: 'transfer(balance + 1 ETH → self) → FAIL',
    expected: 'abort',
    category: 'bridge',
  },
  {
    id: 'bridge-oog',
    label: 'Bridge Out-of-Gas',
    description: 'Bridge send on A paired with an under-gassed receive on B; A reverts too (no ACK).',
    chainADesc: 'approve + bridgeERC20To(0.1 TKN → B) → no ACK → REVERT',
    chainBDesc: 'receiveTokens(gas: 300k) → OOG',
    expected: 'abort',
    category: 'bridge',
  },
  {
    id: 'eth-bridge-oog',
    label: 'ETH Bridge Out-of-Gas',
    description: 'bridgeEthTo on A paired with an under-gassed receiveETH on B; A reverts too (no ACK).',
    chainADesc: 'bridgeEthTo(0.01 ETH → B) → no ACK → REVERT',
    chainBDesc: 'receiveETH(gas: 300k) → OOG',
    expected: 'abort',
    category: 'eth',
  },
  {
    id: 'native-receive-no-send',
    label: 'Native + Receive Without Send',
    description: 'ETH self-transfer on A paired with a bridge receive on B that has no matching send',
    chainADesc: 'transfer(0.1 ETH → self)',
    chainBDesc: 'receiveTokens() → no matching send → FAIL',
    expected: 'abort',
    category: 'native',
  },
  {
    id: 'native-overdraft',
    label: 'Native Success + Overdraft',
    description: 'Valid ETH self-transfer on A paired with an ETH overdraft on B',
    chainADesc: 'transfer(balance / 2 → self)',
    chainBDesc: 'transfer(balance + 1 ETH → self) → FAIL',
    expected: 'abort',
    category: 'native',
  },
]

interface ScenarioState {
  running: boolean
  outcome?: 'committed' | 'aborted'
  error?: string
}

export default function ScenariosForm() {
  const [states, setStates] = useState<Record<ScenarioId, ScenarioState>>(
    () => Object.fromEntries(SCENARIOS.map(s => [s.id, { running: false }])) as Record<ScenarioId, ScenarioState>
  )
  const [category, setCategory] = useState<CategoryId | 'all'>('all')
  const [running, setRunning] = useState(false)

  const { addTransaction, updateTransaction } = useTransactionStore()

  const runScenario = async (id: ScenarioId) => {
    setStates(prev => ({ ...prev, [id]: { running: true } }))

    try {
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
      const sessionId = generateSessionId()

      const nonceA = await providerA.getTransactionCount(senderA, 'pending')
      const nonceB = await providerB.getTransactionCount(senderB, 'pending')

      const transactions: Record<number, string[]> = {}
      let txABytes: string
      let txBBytes: string

      switch (id) {
        case 'cross-chain-mint': {
          const amount = parseAmount('1')
          txABytes = await buildMintTx(tokenA, senderA, amount, signerA, CHAIN_A_ID)
          txBBytes = await buildMintTx(tokenB, senderB, amount, signerB, CHAIN_B_ID)
          transactions[CHAIN_A_ID] = [txABytes]
          transactions[CHAIN_B_ID] = [txBBytes]
          break
        }
        case 'bridge-native-fail': {
          const balanceB = await providerB.getBalance(senderB)
          const amount = parseAmount('0.1')
          const txAApprove = await buildApproveTx(tokenA, bridgeA, amount, signerA, CHAIN_A_ID, nonceA)
          txABytes = await buildBridgeERC20ToTx(bridgeA, CHAIN_B_ID, tokenA, amount, senderB, sessionId, signerA, CHAIN_A_ID, nonceA + 1)
          txBBytes = await buildNativeTransferTx(senderB, balanceB + ethers.parseEther('1'), signerB, CHAIN_B_ID, nonceB)
          transactions[CHAIN_A_ID] = [txAApprove, txABytes]
          transactions[CHAIN_B_ID] = [txBBytes]
          break
        }
        case 'bridge-oog': {
          const amount = parseAmount('0.1')
          const txAApprove = await buildApproveTx(tokenA, bridgeA, amount, signerA, CHAIN_A_ID, nonceA)
          txABytes = await buildBridgeERC20ToTx(bridgeA, CHAIN_B_ID, tokenA, amount, senderB, sessionId, signerA, CHAIN_A_ID, nonceA + 1)
          txBBytes = await buildBridgeReceiveTokensTx(bridgeB, sessionId, CHAIN_A_ID, CHAIN_B_ID, bridgeA, senderB, signerB, CHAIN_B_ID, nonceB, 300000n)
          transactions[CHAIN_A_ID] = [txAApprove, txABytes]
          transactions[CHAIN_B_ID] = [txBBytes]
          break
        }
        case 'eth-bridge-oog': {
          const value = ethers.parseEther('0.01')
          txABytes = await buildBridgeEthToTx(bridgeA, CHAIN_B_ID, senderB, sessionId, value, signerA, CHAIN_A_ID, nonceA)
          txBBytes = await buildBridgeReceiveEthTx(bridgeB, sessionId, CHAIN_A_ID, CHAIN_B_ID, bridgeA, senderB, signerB, CHAIN_B_ID, nonceB, 300000n)
          transactions[CHAIN_A_ID] = [txABytes]
          transactions[CHAIN_B_ID] = [txBBytes]
          break
        }
        case 'native-receive-no-send': {
          txABytes = await buildNativeTransferTx(senderA, ethers.parseEther('0.1'), signerA, CHAIN_A_ID, nonceA)
          txBBytes = await buildBridgeReceiveTokensTx(bridgeB, sessionId, CHAIN_A_ID, CHAIN_B_ID, bridgeA, senderB, signerB, CHAIN_B_ID, nonceB)
          transactions[CHAIN_A_ID] = [txABytes]
          transactions[CHAIN_B_ID] = [txBBytes]
          break
        }
        case 'native-overdraft': {
          const balanceA = await providerA.getBalance(senderA)
          const balanceB = await providerB.getBalance(senderB)
          txABytes = await buildNativeTransferTx(senderA, balanceA / 2n, signerA, CHAIN_A_ID, nonceA)
          txBBytes = await buildNativeTransferTx(senderB, balanceB + ethers.parseEther('1'), signerB, CHAIN_B_ID, nonceB)
          transactions[CHAIN_A_ID] = [txABytes]
          transactions[CHAIN_B_ID] = [txBBytes]
          break
        }
        default:
          throw new Error(`Unknown scenario: ${id}`)
      }

      const txHashA = ethers.Transaction.from(txABytes).hash!
      const txHashB = ethers.Transaction.from(txBBytes).hash!

      const response = await submitXT(transactions)
      const instanceId = response.instance_id

      addTransaction({ instanceId: txHashA, type: 'scenario', status: 'pending', chainId: CHAIN_A_ID, createdAt: new Date() })
      addTransaction({ instanceId: txHashB, type: 'scenario', status: 'pending', chainId: CHAIN_B_ID, createdAt: new Date() })

      try {
        const decision = await waitForDecision(instanceId, 60000)
        if (decision) {
          try {
            const [receiptA, receiptB] = await Promise.all([
              waitForTransactionReceipt(providerA, txHashA, { timeoutMs: 30000 }),
              waitForTransactionReceipt(providerB, txHashB, { timeoutMs: 30000 }),
            ])
            updateTransaction(txHashA, { status: receiptA.status === 1 ? 'committed' : 'aborted', decision: receiptA.status === 1, decidedAt: new Date() })
            updateTransaction(txHashB, { status: receiptB.status === 1 ? 'committed' : 'aborted', decision: receiptB.status === 1, decidedAt: new Date() })
            setStates(prev => ({ ...prev, [id]: { running: false, outcome: 'committed' } }))
          } catch {
            updateTransaction(txHashA, { status: 'aborted', decision: false, decidedAt: new Date() })
            updateTransaction(txHashB, { status: 'aborted', decision: false, decidedAt: new Date() })
            setStates(prev => ({ ...prev, [id]: { running: false, outcome: 'aborted' } }))
          }
        } else {
          updateTransaction(txHashA, { status: 'aborted', decision: false, decidedAt: new Date() })
          updateTransaction(txHashB, { status: 'aborted', decision: false, decidedAt: new Date() })
          setStates(prev => ({ ...prev, [id]: { running: false, outcome: 'aborted' } }))
        }
      } catch (err) {
        updateTransaction(txHashA, { status: 'aborted', decision: false, decidedAt: new Date() })
        updateTransaction(txHashB, { status: 'aborted', decision: false, decidedAt: new Date() })
        setStates(prev => ({ ...prev, [id]: { running: false, error: err instanceof Error ? err.message : 'timeout' } }))
      }
    } catch (err) {
      setStates(prev => ({
        ...prev,
        [id]: { running: false, error: err instanceof Error ? err.message : 'Unknown error' },
      }))
    }
  }

  const visibleScenarios = SCENARIOS.filter(s => category === 'all' || s.category === category)

  const runAllVisible = async () => {
    if (running) return
    setRunning(true)
    try {
      for (const s of visibleScenarios) {
        await runScenario(s.id)
      }
    } finally {
      setRunning(false)
    }
  }

  const visibleCounts = visibleScenarios.reduce(
    (acc, s) => {
      const st = states[s.id]
      if (st.error) acc.err += 1
      else if (st.outcome === 'committed') acc.commit += 1
      else if (st.outcome === 'aborted') acc.abort += 1
      else acc.pending += 1
      return acc
    },
    { commit: 0, abort: 0, err: 0, pending: 0 }
  )

  return (
    <div className="space-y-3">
      <div className="bg-bg-elevated border border-border px-3 py-2.5 space-y-2">
        <div className="flex flex-wrap items-center gap-1.5">
          {CATEGORIES.map(c => {
            const count = c.id === 'all'
              ? SCENARIOS.length
              : SCENARIOS.filter(s => s.category === c.id).length
            const active = category === c.id
            return (
              <button
                key={c.id}
                type="button"
                onClick={() => setCategory(c.id)}
                className={`px-2 py-1 font-display text-[9px] tracking-[0.2em] uppercase border transition-colors ${
                  active
                    ? 'border-amber text-amber bg-amber/5'
                    : 'border-border text-text-secondary hover:border-border-bright hover:text-text-primary'
                }`}
              >
                {c.label} <span className="text-text-dim ml-0.5">{count}</span>
              </button>
            )
          })}
        </div>
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-3 text-[10px] font-mono">
            {visibleCounts.commit > 0 && <span className="text-cyan">{visibleCounts.commit} commit</span>}
            {visibleCounts.abort > 0 && <span className="text-amber">{visibleCounts.abort} abort</span>}
            {visibleCounts.err > 0 && <span className="text-error">{visibleCounts.err} err</span>}
            {visibleCounts.pending > 0 && <span className="text-text-dim">{visibleCounts.pending} pending</span>}
          </div>
          <button
            type="button"
            onClick={runAllVisible}
            disabled={running}
            className={`px-3 py-1.5 font-display text-[9px] tracking-[0.2em] uppercase border transition-all ${
              running
                ? 'border-border text-text-dim cursor-not-allowed'
                : 'border-amber text-amber hover:bg-amber hover:text-bg'
            }`}
          >
            {running ? 'Running…' : `Run ${visibleScenarios.length}`}
          </button>
        </div>
      </div>

      {visibleScenarios.map(scenario => {
        const state = states[scenario.id]
        const isCommit = scenario.expected === 'commit'
        const accentColor = isCommit ? 'cyan' : 'amber'

        return (
          <div
            key={scenario.id}
            className={`border bg-bg p-4 transition-colors ${
              isCommit ? 'border-cyan/20' : 'border-amber/20'
            }`}
          >
            {/* Header row */}
            <div className="flex items-start justify-between gap-3 mb-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2 mb-0.5">
                  <span
                    className={`px-1.5 py-0.5 text-[9px] font-display tracking-widest uppercase border ${
                      isCommit
                        ? 'border-cyan/40 text-cyan/80 bg-cyan/5'
                        : 'border-amber/40 text-amber/80 bg-amber/5'
                    }`}
                  >
                    {scenario.expected}
                  </span>
                  <span className="font-display text-[11px] tracking-wide text-text-primary">
                    {scenario.label}
                  </span>
                </div>
                <p className="text-[10px] text-text-dim font-mono">{scenario.description}</p>
              </div>

              <button
                onClick={() => runScenario(scenario.id)}
                disabled={state.running}
                className={`flex-none px-3 py-1.5 font-display text-[9px] tracking-[0.2em] uppercase border transition-all ${
                  state.running
                    ? 'border-border text-text-dim cursor-not-allowed'
                    : accentColor === 'cyan'
                    ? 'border-cyan text-cyan hover:bg-cyan hover:text-bg'
                    : 'border-amber text-amber hover:bg-amber hover:text-bg'
                }`}
              >
                {state.running ? (
                  <span className="flex items-center gap-1.5">
                    <span className="w-1 h-1 bg-current rounded-full indicator-active" />
                    <span className="w-1 h-1 bg-current rounded-full indicator-active" style={{ animationDelay: '0.3s' }} />
                    <span className="w-1 h-1 bg-current rounded-full indicator-active" style={{ animationDelay: '0.6s' }} />
                  </span>
                ) : 'Run'}
              </button>
            </div>

            {/* Chain descriptions */}
            <div className="grid grid-cols-2 gap-2">
              <div className="bg-bg-elevated border border-border px-2.5 py-2">
                <p className="text-[9px] font-display tracking-widest uppercase text-text-dim mb-1">Chain A</p>
                <p className="text-[10px] font-mono text-text-secondary">{scenario.chainADesc}</p>
              </div>
              <div className="bg-bg-elevated border border-border px-2.5 py-2">
                <p className="text-[9px] font-display tracking-widest uppercase text-text-dim mb-1">Chain B</p>
                <p className="text-[10px] font-mono text-text-secondary">{scenario.chainBDesc}</p>
              </div>
            </div>

            {/* Result */}
            {(state.outcome || state.error) && (
              <div className={`mt-2.5 px-2.5 py-2 border text-[10px] font-mono flex items-center gap-2 ${
                state.error
                  ? 'border-error/30 bg-error/5 text-error'
                  : state.outcome === 'committed'
                  ? 'border-cyan/30 bg-cyan/5 text-cyan'
                  : 'border-amber/30 bg-amber/5 text-amber'
              }`}>
                <span className={`w-1.5 h-1.5 rounded-full flex-none ${
                  state.error ? 'bg-error' : state.outcome === 'committed' ? 'bg-cyan' : 'bg-amber'
                }`} />
                {state.error
                  ? state.error
                  : state.outcome === 'committed'
                  ? 'Committed - both chains executed'
                  : 'Aborted - neither chain executed'}
                {!state.error && (() => {
                  const matched = state.outcome === (scenario.expected === 'commit' ? 'committed' : 'aborted')
                  return matched
                    ? <span className="ml-auto text-text-dim text-[9px] uppercase tracking-widest">as expected</span>
                    : <span className="ml-auto text-error text-[9px] uppercase tracking-widest">unexpected</span>
                })()}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
