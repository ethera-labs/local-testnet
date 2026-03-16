import { Handle, Position, NodeProps } from 'reactflow'

interface UserNodeData {
  label: string
}

const handleStyle = { background: '#505070', border: '1px solid #707090', width: 8, height: 8 }

export default function UserNode({ data }: NodeProps<UserNodeData>) {
  return (
    <div
      style={{
        fontFamily: '"IBM Plex Mono", monospace',
        background: '#1A1A25',
        border: '1.5px solid #505070',
        minWidth: 100,
        padding: '8px 12px',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 6,
      }}
    >
      <div style={{ padding: 6, border: '1px solid #404060', background: '#242435' }}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="#8080A0" strokeWidth="1.5" strokeLinecap="square">
          <path d="M19 21v-2a4 4 0 0 0-4-4H9a4 4 0 0 0-4 4v2" />
          <circle cx="12" cy="7" r="4" />
        </svg>
      </div>
      <div style={{ fontSize: 11, color: '#A0A0C0', letterSpacing: '0.05em' }}>{data.label}</div>

      <Handle type="source" position={Position.Bottom} style={handleStyle} />
      <Handle type="source" position={Position.Right} style={handleStyle} />
    </div>
  )
}
