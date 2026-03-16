import { Handle, Position, NodeProps } from 'reactflow'

interface GethNodeData {
  label: string
  port: number
  connected: boolean
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function GethNode({ data }: NodeProps<GethNodeData>) {
  const borderColor = data.connected ? '#505070' : '#FF3D5A'
  const bgColor = data.connected ? '#1A1A25' : 'rgba(255,61,90,0.07)'

  return (
    <div
      style={{
        fontFamily: '"IBM Plex Mono", monospace',
        background: bgColor,
        border: `1.5px solid ${borderColor}`,
        minWidth: 100,
        padding: '8px 12px',
        transition: 'all 0.2s',
      }}
    >
      <Handle type="target" position={Position.Top} style={handleStyle} />
      <Handle type="source" position={Position.Right} style={handleStyle} />
      <Handle type="source" position={Position.Left} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, color: '#A0A0C0', letterSpacing: '0.05em' }}>{data.label}</div>
        <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>:{data.port}</div>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4, marginTop: 3 }}>
          <span style={{ width: 5, height: 5, borderRadius: '50%', background: data.connected ? '#505070' : '#FF3D5A', display: 'inline-block' }} />
          <span style={{ fontSize: 9, color: '#6060A0', letterSpacing: '0.15em', textTransform: 'uppercase' }}>
            {data.connected ? 'Running' : 'Down'}
          </span>
        </div>
      </div>
    </div>
  )
}
