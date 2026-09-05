interface StatTileProps {
  label: string
  value: string
  accent?: 'default' | 'success' | 'warning'
}

const ACCENT_CLASSES: Record<NonNullable<StatTileProps['accent']>, string> = {
  default: 'text-ink',
  success: 'text-good-text',
  warning: 'text-warn-text',
}

/** A single summary metric tile — total cases, money recovered, etc. */
export function StatTile({ label, value, accent = 'default' }: StatTileProps) {
  return (
    <div className="rounded-card bg-white p-5 shadow-card">
      <p className="text-xs font-semibold tracking-wide text-ink-muted uppercase">{label}</p>
      <p className={`mt-1.5 text-3xl font-bold tracking-tight tabular-nums ${ACCENT_CLASSES[accent]}`}>{value}</p>
    </div>
  )
}
