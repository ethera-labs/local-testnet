import { Handle, Position, NodeProps } from 'reactflow'

interface PublisherNodeData {
  label: string
  port: number
  active: boolean
  coordinating?: boolean
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function PublisherNode({ data }: NodeProps<PublisherNodeData>) {
  const borderColor = data.coordinating ? '#9B6DFF' : data.active ? '#7B4DDD' : '#505070'
  const bgColor = data.coordinating ? 'rgba(155,109,255,0.1)' : data.active ? 'rgba(123,77,221,0.06)' : '#1A1A25'

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
      <Handle type="target" position={Position.Right} style={handleStyle} />
      <Handle type="source" position={Position.Top} style={handleStyle} />
      <Handle type="source" position={Position.Bottom} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, fontWeight: 500, color: borderColor, letterSpacing: '0.05em' }}>
          {data.label}
        </div>
        <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>:{data.port}</div>
        {data.coordinating && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4, marginTop: 4 }}>
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#9B6DFF', display: 'inline-block' }} className="indicator-active" />
            <span style={{ fontSize: 9, color: '#9B6DFF', letterSpacing: '0.2em', textTransform: 'uppercase' }}>Coordinating</span>
          </div>
        )}
      </div>
    </div>
  )
}
