import { useEffect, useState } from 'react'
import { useTransactionStore } from '../../stores/transactionStore'
import { getXTStatus, waitForDecision } from '../../api/sidecar'

interface TransactionStatusProps {
  instanceId: string
  onClose: () => void
}

const steps = [
  { id: 'submit',   label: 'Submit XT' },
  { id: 'simulate', label: 'Simulate on both chains' },
  { id: 'circ',     label: 'Exchange CIRC messages' },
  { id: 'vote',     label: 'Collect votes' },
  { id: 'decide',   label: 'Make decision' },
  { id: 'deliver',  label: 'Deliver to builders' },
]

export default function TransactionStatus({ instanceId, onClose }: TransactionStatusProps) {
  const [status, setStatus] = useState<string>('pending')
  const [decision, setDecision] = useState<boolean | null>(null)
  const [error, setError] = useState<string | null>(null)
  const { updateTransaction } = useTransactionStore()
  const [startedAt] = useState(() => Date.now())
  const [copied, setCopied] = useState(false)
  const [elapsedMs, setElapsedMs] = useState<number | null>(null)

  const handleCopy = () => {
    navigator.clipboard.writeText(instanceId)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  useEffect(() => {
    let mounted = true

    const checkStatus = async () => {
      try {
        const result = await waitForDecision(instanceId, 30000)
        if (mounted) {
          setElapsedMs(Date.now() - startedAt)
          setDecision(result)
          setStatus(result ? 'committed' : 'aborted')
          updateTransaction(instanceId, {
            status: result ? 'committed' : 'aborted',
            decision: result,
            decidedAt: new Date(),
          })
        }
      } catch (err) {
        if (mounted) setError(err instanceof Error ? err.message : 'Unknown error')
      }
    }

    checkStatus()

    const interval = setInterval(async () => {
      try {
        const response = await getXTStatus(instanceId)
        if (mounted) setStatus(response.status)
      } catch {
        // ignore polling errors
      }
    }, 500)

    return () => {
      mounted = false
      clearInterval(interval)
    }
  }, [instanceId, updateTransaction])

  const isStepComplete = (index: number) =>
    decision !== null ||
    (status === 'simulating' && index < 1) ||
    (status === 'waiting_circ' && index < 2) ||
    (status === 'simulated' && index < 3) ||
    (status === 'voted' && index < 4)

  return (
    <div className="cb border border-border bg-bg-elevated">
      {/* Header */}
      <div className="border-b border-border px-4 py-3 flex items-start justify-between">
        <div>
          <div className="font-display text-[10px] tracking-[0.3em] uppercase text-text-secondary mb-1">
            XT Status
          </div>
          <div className="flex items-center gap-1.5">
            <div className="font-mono text-[11px] text-text-secondary">
              {instanceId.slice(0, 20)}…{instanceId.slice(-4)}
            </div>
            <button
              onClick={handleCopy}
              className="text-text-dim hover:text-amber transition-colors"
              title="Copy full XT ID"
            >
              {copied ? (
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="square">
                  <path d="M20 6L9 17l-5-5" />
                </svg>
              ) : (
                <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="square">
                  <rect x="9" y="9" width="13" height="13" />
                  <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
                </svg>
              )}
            </button>
          </div>
        </div>
        <button
          onClick={onClose}
          className="text-text-dim hover:text-amber transition-colors p-1"
        >
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="square">
            <path d="M18 6 6 18M6 6l12 12" />
          </svg>
        </button>
      </div>

      {/* Decision banner */}
      <div className="px-4 py-3 border-b border-border">
        {decision === null ? (
          <div className="flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-yellow-400 indicator-active" />
            <span className="font-display text-[11px] tracking-[0.2em] uppercase text-yellow-400">
              {status.replace(/_/g, ' ')}
            </span>
          </div>
        ) : decision ? (
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-cyan glow-cyan" />
              <span className="font-display text-[11px] tracking-[0.2em] uppercase text-cyan">
                Committed
              </span>
            </div>
            {elapsedMs !== null && (
              <span className="font-mono text-[10px] text-text-dim">
                {elapsedMs < 1000 ? `${elapsedMs}ms` : `${(elapsedMs / 1000).toFixed(1)}s`}
              </span>
            )}
          </div>
        ) : (
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-error" />
              <span className="font-display text-[11px] tracking-[0.2em] uppercase text-error">
                Aborted
              </span>
            </div>
            {elapsedMs !== null && (
              <span className="font-mono text-[10px] text-text-dim">
                {elapsedMs < 1000 ? `${elapsedMs}ms` : `${(elapsedMs / 1000).toFixed(1)}s`}
              </span>
            )}
          </div>
        )}
      </div>

      {/* Steps */}
      <div className="px-4 py-3 space-y-2.5">
        {steps.map((step, index) => {
          const complete = isStepComplete(index)
          const active = !complete && status === step.id
          return (
            <div key={step.id} className="flex items-center gap-3">
              <span
                className={`w-1.5 h-1.5 rounded-full flex-none ${
                  complete ? 'bg-cyan' : active ? 'bg-yellow-400 indicator-active' : 'bg-border-bright'
                }`}
              />
              <span className={`text-[11px] font-mono ${complete ? 'text-text-primary' : 'text-text-dim'}`}>
                {step.label}
              </span>
              {complete && (
                <span className="ml-auto text-[9px] font-display tracking-widest uppercase text-cyan/60">done</span>
              )}
            </div>
          )
        })}
      </div>

      {error && (
        <div className="mx-4 mb-4 border border-error/40 bg-error/5 px-3 py-2 text-error text-[11px] font-mono">
          <span className="text-error/60 mr-2">!</span>{error}
        </div>
      )}
    </div>
  )
}
