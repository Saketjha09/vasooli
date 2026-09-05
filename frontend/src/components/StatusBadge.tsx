import { statusLabel } from '../lib/format'

const STATUS_COLORS: Record<string, string> = {
  processing: 'bg-slate-100 text-slate-700',
  recovered: 'bg-emerald-100 text-emerald-800',
  held: 'bg-amber-100 text-amber-800',
  open: 'bg-blue-100 text-blue-800',
  escalated: 'bg-red-100 text-red-800',
  error: 'bg-red-100 text-red-800',
}

export function StatusBadge({ status }: { status: string }) {
  const colorClass = STATUS_COLORS[status] ?? 'bg-slate-100 text-slate-700'
  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-semibold ${colorClass}`}>
      {statusLabel(status)}
    </span>
  )
}
