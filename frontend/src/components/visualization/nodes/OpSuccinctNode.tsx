import { Handle, Position, NodeProps } from 'reactflow'
import type { ServiceStatus } from '../../../api/health'
import StatusDot from './StatusDot'

interface OpSuccinctNodeData {
  label: string
  port?: number
  variant: 'prover' | 'postgres'
  status?: ServiceStatus
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function OpSuccinctNode({ data }: NodeProps<OpSuccinctNodeData>) {
  const accent = data.variant === 'postgres' ? '#A78BFA' : '#F0ABFC'
  return (
    <div
      style={{
        fontFamily: '"IBM Plex Mono", monospace',
        background: 'rgba(167,139,250,0.06)',
        border: `1.5px solid ${accent}`,
        minWidth: 120,
        padding: '8px 12px',
        position: 'relative',
      }}
    >
      <StatusDot status={data.status} />
      <Handle type="target" position={Position.Left} style={handleStyle} />
      <Handle type="source" position={Position.Right} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, color: accent, letterSpacing: '0.05em' }}>{data.label}</div>
        {data.port && (
          <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>:{data.port}</div>
        )}
        <div style={{ fontSize: 9, color: accent, letterSpacing: '0.2em', textTransform: 'uppercase', marginTop: 3 }}>
          {data.variant === 'postgres' ? 'state-store' : 'op-succinct'}
        </div>
      </div>
    </div>
  )
}
