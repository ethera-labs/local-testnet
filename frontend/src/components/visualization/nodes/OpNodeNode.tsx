import { Handle, Position, NodeProps } from 'reactflow'

interface OpNodeNodeData {
  label: string
  port?: number
  active: boolean
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function OpNodeNode({ data }: NodeProps<OpNodeNodeData>) {
  const borderColor = data.active ? '#00D4A8' : '#505070'
  const bgColor = data.active ? 'rgba(0,212,168,0.07)' : '#1A1A25'

  return (
    <div
      style={{
        fontFamily: '"IBM Plex Mono", monospace',
        background: bgColor,
        border: `1.5px solid ${borderColor}`,
        minWidth: 110,
        padding: '8px 12px',
        transition: 'all 0.2s',
      }}
    >
      <Handle type="target" position={Position.Left} style={handleStyle} />
      <Handle id="to-boost" type="source" position={Position.Bottom} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, color: data.active ? '#00D4A8' : '#A0A0C0', letterSpacing: '0.05em' }}>
          {data.label}
        </div>
        {data.port && (
          <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>:{data.port}</div>
        )}
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4, marginTop: 3 }}>
          <span
            style={{ width: 5, height: 5, borderRadius: '50%', background: data.active ? '#00D4A8' : '#505070', display: 'inline-block' }}
            className={data.active ? 'indicator-active' : ''}
          />
          <span style={{ fontSize: 9, color: '#6060A0', letterSpacing: '0.15em', textTransform: 'uppercase' }}>
            {data.active ? 'Active' : 'Idle'}
          </span>
        </div>
      </div>
    </div>
  )
}
