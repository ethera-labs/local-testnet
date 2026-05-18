import { Handle, Position, NodeProps } from 'reactflow'
import type { ServiceStatus } from '../../../api/health'
import StatusDot from './StatusDot'

interface ProposerNodeData {
  label: string
  port: number
  status?: ServiceStatus
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function ProposerNode({ data }: NodeProps<ProposerNodeData>) {
  return (
    <div
      style={{
        fontFamily: '"IBM Plex Mono", monospace',
        background: '#1A1A25',
        border: '1.5px solid #505070',
        minWidth: 110,
        padding: '8px 12px',
        position: 'relative',
      }}
    >
      <StatusDot status={data.status} />
      <Handle type="target" position={Position.Top} style={handleStyle} />
      <Handle type="source" position={Position.Bottom} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, color: '#A0A0C0', letterSpacing: '0.05em' }}>{data.label}</div>
        <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>:{data.port}</div>
        <div style={{ fontSize: 9, color: '#6060A0', letterSpacing: '0.2em', textTransform: 'uppercase', marginTop: 3 }}>
          proposer
        </div>
      </div>
    </div>
  )
}
