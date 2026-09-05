import { statusLabel } from '../lib/format'

// The 3-color status system, reused everywhere a status appears (badges,
// stat tiles, recovery breakdown, chart bars) — not picked per-component.
// open/held share "warn" (both are still-pending, not yet resolved);
// processing/error aren't part of the 3-state system the demo data ever
// actually reaches, so they stay neutral/critical respectively rather than
// inventing a 4th color.
const STATUS_COLORS: Record<string, string> = {
  processing: 'bg-page text-ink-muted',
  recovered: 'bg-good-bg text-good-text',
  held: 'bg-warn-bg text-warn-text',
  open: 'bg-warn-bg text-warn-text',
  escalated: 'bg-critical-bg text-critical-text',
  error: 'bg-critical-bg text-critical-text',
}

export function StatusBadge({ status }: { status: string }) {
  const colorClass = STATUS_COLORS[status] ?? 'bg-page text-ink-muted'
  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-semibold ${colorClass}`}>
      {statusLabel(status)}
    </span>
  )
}
