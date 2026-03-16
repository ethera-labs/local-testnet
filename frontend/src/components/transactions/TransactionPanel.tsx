import { useState } from 'react'
import type { FlowMode } from '../visualization/TransactionFlowPanel'
import MintForm from './MintForm'
import BridgeForm from './BridgeForm'
import ScenariosForm from './ScenariosForm'
import StressForm from './StressForm'
import TransactionStatus from './TransactionStatus'

interface TransactionPanelProps {
  mode: 'mint' | 'bridge' | 'atomicity' | 'stress'
  onSelectFlow?: (mode: FlowMode) => void
}

export default function TransactionPanel({
  mode,
  onSelectFlow,
}: TransactionPanelProps) {
  const [activeInstanceId, setActiveInstanceId] = useState<string | null>(null)

  const handleClose = () => setActiveInstanceId(null)

  return (
    <div className="space-y-6">
      {mode === 'bridge' && activeInstanceId ? (
        <TransactionStatus instanceId={activeInstanceId} onClose={handleClose} />
      ) : mode === 'mint' ? (
        <MintForm onSelectFlow={onSelectFlow} />
      ) : mode === 'bridge' ? (
        <BridgeForm onSubmit={setActiveInstanceId} onSelectFlow={onSelectFlow} />
      ) : mode === 'atomicity' ? (
        <ScenariosForm />
      ) : (
        <StressForm />
      )}
    </div>
  )
}
