import { Handle, Position, NodeProps } from 'reactflow'

interface SidecarNodeData {
  label: string
  port: number
  active: boolean
  processing?: boolean
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function SidecarNode({ data }: NodeProps<SidecarNodeData>) {
  const borderColor = data.processing ? '#FF6B00' : data.active ? '#00D4A8' : '#505070'
  const bgColor = data.processing ? 'rgba(255,107,0,0.08)' : data.active ? 'rgba(0,212,168,0.07)' : '#1A1A25'

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
      <Handle type="source" position={Position.Right} style={handleStyle} />
      <Handle type="source" position={Position.Bottom} style={handleStyle} />
      <Handle type="target" position={Position.Top} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, fontWeight: 500, color: borderColor, letterSpacing: '0.05em' }}>
          {data.label}
        </div>
        <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>:{data.port}</div>
        {data.processing && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4, marginTop: 4 }}>
            <span style={{ width: 6, height: 6, borderRadius: '50%', background: '#FF6B00', display: 'inline-block' }} className="indicator-active" />
            <span style={{ fontSize: 9, color: '#FF6B00', letterSpacing: '0.2em', textTransform: 'uppercase' }}>Processing</span>
          </div>
        )}
        {!data.processing && data.active && (
          <div style={{ display: 'flex', justifyContent: 'center', marginTop: 3 }}>
            <span style={{ width: 5, height: 5, borderRadius: '50%', background: '#00D4A8', display: 'inline-block' }} className="indicator-active" />
          </div>
        )}
      </div>
    </div>
  )
}
