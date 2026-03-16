import { Handle, Position, NodeProps } from 'reactflow'

interface BoostNodeData {
  label: string
  port: number
  active: boolean
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function BoostNode({ data }: NodeProps<BoostNodeData>) {
  return (
    <div
      style={{
        fontFamily: '"IBM Plex Mono", monospace',
        background: '#1A1A25',
        border: '1.5px solid #505070',
        minWidth: 130,
        padding: '8px 12px',
      }}
    >
      <Handle id="from-rollup" type="target" position={Position.Top} style={handleStyle} />
      <Handle id="to-builder" type="source" position={Position.Bottom} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, color: '#A0A0C0', letterSpacing: '0.05em' }}>{data.label}</div>
        <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>:{data.port}</div>
      </div>
    </div>
  )
}
