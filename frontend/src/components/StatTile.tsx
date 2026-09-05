interface StatTileProps {
  label: string
  value: string
  accent?: 'default' | 'success' | 'warning'
}

const ACCENT_CLASSES: Record<NonNullable<StatTileProps['accent']>, string> = {
  default: 'text-slate-900',
  success: 'text-emerald-600',
  warning: 'text-amber-600',
}

/** A single summary metric tile — total cases, money recovered, etc. */
export function StatTile({ label, value, accent = 'default' }: StatTileProps) {
  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
      <p className="text-sm font-medium text-slate-500">{label}</p>
      <p className={`mt-1 text-2xl font-semibold ${ACCENT_CLASSES[accent]}`}>{value}</p>
    </div>
  )
}
