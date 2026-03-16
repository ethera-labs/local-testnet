export type FlowMode = 'normal' | 'xt'

interface FlowStep {
  title: string
  detail: string
  subDetails?: string[]
}

interface FlowContent {
  title: string
  subtitle: string
  tags: string[]
  steps: FlowStep[]
  timing?: { title: string; items: string[] }
  note?: string
}

const flowContent: Record<FlowMode, FlowContent> = {
  normal: {
    title: 'Normal Transaction Flow',
    subtitle: 'Flashblocks + rollup-boost path',
    tags: ['engine_FCU', 'engine_getPayload', 'mempool', 'flashblocks'],
    steps: [
      {
        title: 'Submit to builder RPC',
        detail: 'User sends tx to op-rbuilder. The tx enters the mempool (reth transaction pool) ordered by gas price/priority.',
      },
      {
        title: 'Start block build (FCU)',
        detail: 'op-node calls engine_forkchoiceUpdated on rollup-boost, which forwards to both op-geth (proposer) and op-rbuilder (builder).',
        subDetails: ['op-rbuilder starts building flashblocks immediately', 'Fallback block (deposits only) is built first'],
      },
      {
        title: 'Flashblock building loop',
        detail: 'Every 250ms (configurable), op-rbuilder builds a new flashblock chunk. Each flashblock pulls txs from the mempool using best_transactions_with_attributes().',
        subDetails: ['Txs ordered by priority (tip/gas price)', 'Executed until gas/DA limits reached', 'Published via WebSocket to subscribers'],
      },
      {
        title: 'Assemble final payload',
        detail: 'op-node calls engine_getPayload; rollup-boost fetches the builder payload (all flashblocks merged) and the proposer fallback payload from op-geth.',
      },
      {
        title: 'Validate builder payload',
        detail: 'rollup-boost validates the builder payload by calling op-geth engine_newPayload. This ensures the payload is valid before returning it.',
      },
      {
        title: 'Fallback on invalid',
        detail: 'If validation fails, rollup-boost returns the op-geth payload instead. The builder payload is discarded.',
      },
      {
        title: 'Finalize chain head',
        detail: 'op-node submits engine_newPayload + engine_FCU (no attrs) to advance the canonical head.',
      },
    ],
    timing: {
      title: 'Flashblock Timeline (2s block, 250ms interval)',
      items: [
        '0ms: Fallback block (deposits only)',
        '250ms: Flashblock 1 → pull mempool txs',
        '500ms: Flashblock 2 → pull mempool txs',
        '750ms: Flashblock 3 → pull mempool txs',
        '...continues every 250ms...',
        '2000ms: Block finalized (state root calculated)',
      ],
    },
    note: 'If flashblocks are disabled, normal txs go directly to op-geth mempool instead.',
  },
  xt: {
    title: 'Cross-Chain XT Flow',
    subtitle: 'Flashblocks pull model + sidecar coordination + 2PC',
    tags: ['POST /transactions', 'hold mechanism', 'CIRC', '2PC', 'putInbox'],
    steps: [
      {
        title: 'Submit XT bundle',
        detail: 'User submits cross-chain transaction to the sidecar via POST /xt. The sidecar queues it for coordination.',
        subDetails: ['XT contains txs for multiple chains: {chainA: tx1, chainB: tx2}'],
      },
      {
        title: 'Flashblock boundary polling',
        detail: 'At each flashblock boundary (every 250ms), op-rbuilder calls POST /transactions on the sidecar with current chain state.',
        subDetails: [
          'Request includes: chain_id, block_number, flashblock_index, state_root, timestamp, gas_limit',
          'This is the "pull" model: builder pulls txs from sidecar',
        ],
      },
      {
        title: 'Hold mechanism (waiting for chains)',
        detail: 'Sidecar returns {hold: true, poll_after_ms: 50} if not ready. Builder retries up to max_retries (default 5) before timing out.',
        subDetails: [
          'Hold happens while waiting for ALL chains to reach same flashblock',
          'State is "frozen" once all chains arrive',
          'Default timeout: 200ms × 5 retries = 1 second max wait',
        ],
      },
      {
        title: 'Simulate + mailbox trace',
        detail: 'Once all chains frozen, sidecar simulates each tx using debug_traceCall with prestateTracer to analyze mailbox operations.',
        subDetails: [
          'mailbox.read() → creates CrossRollupDependency (needs data from other chain)',
          'mailbox.write() → creates CrossRollupMessage (sends data to other chain)',
          'putInbox txs are staged for fulfilled dependencies',
        ],
      },
      {
        title: 'CIRC message exchange',
        detail: 'Sidecars exchange Cross-chain Input/output Router messages. Chain A sends mailbox.write() data to Chain B to fulfill mailbox.read() dependencies.',
        subDetails: [
          'Messages sent via POST /mailbox to peer sidecars',
          'Re-simulation with state overrides after receiving data',
          'Loop until all dependencies satisfied',
        ],
      },
      {
        title: 'Vote + 2PC decision',
        detail: 'Each sidecar votes YES (simulation succeeded) or NO (failed). Decision requires all chains to vote YES for COMMIT.',
        subDetails: ['Any NO vote → immediate ABORT', 'All YES votes → COMMIT'],
      },
      {
        title: 'Deliver required payload',
        detail: 'Sidecar responds to builder with transactions. On COMMIT: {hold: false, transactions: [{raw, required: true}]}. On ABORT: empty list.',
        subDetails: [
          'required: true → tx MUST succeed or flashblock build fails',
          'Includes putInbox txs first, then main XT tx',
          'Builder executes in order: putInbox → mainTx',
        ],
      },
      {
        title: 'Payload validation',
        detail: 'rollup-boost validates the builder payload via op-geth engine_newPayload. Required sidecar txs must have succeeded.',
      },
      {
        title: 'Strict mode fallback',
        detail: 'CRITICAL: Strict mode must be enabled for XTs. Without it, rollup-boost can fallback to op-geth payload which does NOT contain the XT.',
        subDetails: ['Strict mode: builder payload required, no fallback', 'Without strict: XT atomicity can be broken across chains'],
      },
    ],
    timing: {
      title: 'Hold Mechanism Timeline',
      items: [
        'Builder A polls → Sidecar A: {hold: true} (waiting for B)',
        'Builder B polls → Sidecar B records state',
        'Builder A retries → Sidecar A: {hold: true} (simulating)',
        'Sidecars exchange CIRC messages',
        'Builder A retries → Sidecar A: {hold: true} (voting)',
        '2PC completes → Decision: COMMIT',
        'Builder A retries → Sidecar A: {hold: false, txs: [...]}',
      ],
    },
    note: 'Flashblocks pull model: builders poll at each boundary, sidecar holds until coordination complete, then delivers required txs.',
  },
}

interface TransactionFlowPanelProps {
  mode: FlowMode | null
  onSelect: (mode: FlowMode) => void
  onClear: () => void
}

export default function TransactionFlowPanel({ mode, onSelect, onClear }: TransactionFlowPanelProps) {
  const activeFlow = mode ? flowContent[mode] : null
  const accentColor = mode === 'xt' ? 'amber' : 'cyan'

  return (
    <div className="p-5">
      {/* Selector */}
      <div className="flex items-center justify-between mb-4">
        <span className="font-display text-[10px] tracking-[0.3em] uppercase text-text-secondary">
          Protocol Flow
        </span>
        <div className="flex items-center gap-2">
          <button
            onClick={() => onSelect('normal')}
            className={`px-2.5 py-1 font-display text-[9px] tracking-widest uppercase border transition-all ${
              mode === 'normal'
                ? 'border-cyan text-cyan bg-cyan/5'
                : 'border-border text-text-dim hover:border-border-bright hover:text-text-secondary'
            }`}
          >
            Normal TX
          </button>
          <button
            onClick={() => onSelect('xt')}
            className={`px-2.5 py-1 font-display text-[9px] tracking-widest uppercase border transition-all ${
              mode === 'xt'
                ? 'border-amber text-amber bg-amber/5'
                : 'border-border text-text-dim hover:border-border-bright hover:text-text-secondary'
            }`}
          >
            Submit XT
          </button>
          <button
            onClick={onClear}
            disabled={!mode}
            className="text-[9px] font-display tracking-widest uppercase transition-colors disabled:text-border disabled:cursor-not-allowed text-text-dim hover:text-text-secondary"
          >
            Clear
          </button>
        </div>
      </div>

      {activeFlow ? (
        <div className="space-y-4">
          {/* Title */}
          <div>
            <h4 className={`font-display text-sm tracking-wider ${accentColor === 'amber' ? 'text-amber' : 'text-cyan'}`}>
              {activeFlow.title}
            </h4>
            <p className="text-[10px] text-text-dim font-mono mt-0.5">{activeFlow.subtitle}</p>
          </div>

          {/* Tags */}
          <div className="flex flex-wrap gap-1.5">
            {activeFlow.tags.map((tag) => (
              <span
                key={tag}
                className="px-2 py-0.5 text-[9px] font-mono bg-bg border border-border text-text-dim"
              >
                {tag}
              </span>
            ))}
          </div>

          {/* Steps */}
          <div className="space-y-3 max-h-[320px] overflow-y-auto pr-1">
            {activeFlow.steps.map((step, index) => (
              <div key={step.title} className="flex gap-3">
                <div
                  className={`flex h-5 w-5 shrink-0 items-center justify-center border text-[9px] font-mono mt-0.5 ${
                    accentColor === 'amber'
                      ? 'border-amber/40 text-amber/70'
                      : 'border-cyan/40 text-cyan/70'
                  }`}
                >
                  {index + 1}
                </div>
                <div className="min-w-0">
                  <p className="text-[11px] font-display tracking-wide text-text-primary">{step.title}</p>
                  <p className="text-[10px] text-text-secondary font-mono mt-0.5 leading-relaxed">{step.detail}</p>
                  {step.subDetails && step.subDetails.length > 0 && (
                    <ul className="mt-1.5 space-y-0.5">
                      {step.subDetails.map((sub, i) => (
                        <li key={i} className="text-[10px] text-text-dim font-mono pl-3 border-l border-border">
                          {sub}
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>
            ))}
          </div>

          {/* Timing */}
          {activeFlow.timing && (
            <div className="border border-border bg-bg p-3">
              <p className="font-display text-[9px] tracking-widest uppercase text-text-secondary mb-2">
                {activeFlow.timing.title}
              </p>
              <div className="space-y-1">
                {activeFlow.timing.items.map((item, i) => (
                  <p key={i} className="text-[10px] text-text-dim font-mono">{item}</p>
                ))}
              </div>
            </div>
          )}

          {/* Note */}
          {activeFlow.note && (
            <div className="border-l-2 border-border-bright pl-3 py-1">
              <p className="text-[10px] text-text-dim font-mono leading-relaxed">{activeFlow.note}</p>
            </div>
          )}
        </div>
      ) : (
        <div className="border border-dashed border-border px-4 py-6 text-center">
          <p className="text-[10px] text-text-dim font-mono">
            Select a flow above or click edges in the diagram
          </p>
        </div>
      )}
    </div>
  )
}
