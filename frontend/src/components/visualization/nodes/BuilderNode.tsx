import { Handle, Position, NodeProps } from 'reactflow'

interface BuilderNodeData {
  label: string
  port: number
  polling?: boolean
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function BuilderNode({ data }: NodeProps<BuilderNodeData>) {
  const borderColor = data.polling ? '#FFD600' : '#505070'
  const bgColor = data.polling ? 'rgba(255,214,0,0.07)' : '#1A1A25'

  return (
    <div
      style={{
        fontFamily: '"IBM Plex Mono", monospace',
        background: bgColor,
        border: `1.5px solid ${borderColor}`,
        minWidth: 120,
        padding: '8px 12px',
        transition: 'all 0.2s',
      }}
    >
      <Handle id="from-rollup" type="target" position={Position.Top} style={handleStyle} />
      <Handle type="source" position={Position.Top} style={handleStyle} />
      <Handle type="source" position={Position.Bottom} style={handleStyle} />
      <Handle type="target" position={Position.Bottom} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, fontWeight: 500, color: data.polling ? '#FFD600' : '#A0A0C0', letterSpacing: '0.05em' }}>
          {data.label}
        </div>
        <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>:{data.port}</div>
        {data.polling && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4, marginTop: 4 }}>
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#FFD600', display: 'inline-block' }} className="indicator-active" />
            <span style={{ fontSize: 9, color: '#FFD600', letterSpacing: '0.2em', textTransform: 'uppercase' }}>Polling</span>
          </div>
        )}
      </div>
    </div>
  )
}
