import { Handle, Position, NodeProps } from 'reactflow'

interface RollupNodeData {
  label: string
  chainId: number
  connected: boolean
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function RollupNode({ data }: NodeProps<RollupNodeData>) {
  const borderColor = data.connected ? '#00D4A8' : '#FF3D5A'
  const bgColor = data.connected ? 'rgba(0,212,168,0.07)' : 'rgba(255,61,90,0.07)'

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
      <Handle type="target" position={Position.Left} style={handleStyle} />
      <Handle type="source" position={Position.Right} style={handleStyle} />
      <Handle id="to-builder" type="source" position={Position.Bottom} style={handleStyle} />

      <div style={{ textAlign: 'center' }}>
        <div style={{ fontSize: 11, color: data.connected ? '#00D4A8' : '#FF3D5A', letterSpacing: '0.05em' }}>
          {data.label}
        </div>
        <div style={{ fontSize: 10, color: '#8080A0', marginTop: 2 }}>Chain {data.chainId}</div>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 4, marginTop: 3 }}>
          <span
            style={{ width: 5, height: 5, borderRadius: '50%', background: borderColor, display: 'inline-block' }}
            className={data.connected ? 'indicator-active' : ''}
          />
          <span style={{ fontSize: 9, color: '#6060A0', letterSpacing: '0.15em', textTransform: 'uppercase' }}>
            {data.connected ? 'Connected' : 'Down'}
          </span>
        </div>
      </div>
    </div>
  )
}
