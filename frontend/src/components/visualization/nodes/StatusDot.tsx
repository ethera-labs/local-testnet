import type { ServiceStatus } from '../../../api/health'

interface StatusDotProps {
  status?: ServiceStatus
}

// Palette mirrors the header chips in App.tsx so diagram and header read
// the same status at a glance. "missing" stays muted because the container
// is intentionally absent, not broken.
const STATUS_PALETTE: Record<ServiceStatus, { color: string; glow: boolean }> = {
  up: { color: '#00D4A8', glow: true },
  starting: { color: '#FBBF24', glow: true },
  down: { color: '#EF4444', glow: false },
  missing: { color: '#3A3A4A', glow: false },
}

export default function StatusDot({ status }: StatusDotProps) {
  const resolved = status ?? 'missing'
  const palette = STATUS_PALETTE[resolved]
  return (
    <span
      style={{
        position: 'absolute',
        top: 4,
        right: 4,
        width: 6,
        height: 6,
        borderRadius: '50%',
        background: palette.color,
      }}
      className={palette.glow ? 'indicator-active' : undefined}
      aria-label={`status: ${resolved}`}
      title={`status: ${resolved}`}
    />
  )
}
