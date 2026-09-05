import type { CaseSummary } from '../api/types'
import { formatMoney } from '../lib/format'

interface Bucket {
  key: string
  label: string
  color: string
  count: number
  amount: number
}

const BUCKET_ORDER: { key: string; label: string; color: string; statuses: string[] }[] = [
  { key: 'recovered', label: 'Recovered', color: '#10b981', statuses: ['recovered'] },
  { key: 'escalated', label: 'Escalated', color: '#ef4444', statuses: ['escalated'] },
  { key: 'open', label: 'Open', color: '#f59e0b', statuses: ['open', 'held', 'processing'] },
  { key: 'error', label: 'Error', color: '#6b7280', statuses: ['error'] },
]

/**
 * Recovered/Escalated/Open are a PARTITION of the same case set, not
 * sequential funnel stages (a case doesn't pass through all 3) — so this
 * renders as a single 100%-stacked bar sized by dollar amount, not a
 * narrowing funnel shape, to avoid implying a drop-off that isn't real.
 * "Open" groups open/held/processing (none of those are resolved yet);
 * "Error" only shows if it actually occurs.
 */
export function RecoveryBreakdown({ cases }: { cases: CaseSummary[] }) {
  const buckets: Bucket[] = BUCKET_ORDER.map((b) => {
    const matched = cases.filter((c) => b.statuses.includes(c.status))
    return {
      key: b.key,
      label: b.label,
      color: b.color,
      count: matched.length,
      amount: matched.reduce((sum, c) => sum + c.transactionAmount, 0),
    }
  }).filter((b) => b.count > 0)

  const totalAtRisk = cases.reduce((sum, c) => sum + c.transactionAmount, 0)

  if (cases.length === 0) {
    return <p className="text-sm text-slate-500">No cases yet.</p>
  }

  return (
    <div>
      <p className="mb-2 text-sm text-slate-600">
        Total At Risk: <span className="font-semibold text-slate-900">{formatMoney(totalAtRisk)}</span>
      </p>
      <div className="flex h-6 w-full overflow-hidden rounded-md border border-slate-200">
        {buckets.map((b) => (
          <div
            key={b.key}
            style={{ width: `${totalAtRisk > 0 ? (b.amount / totalAtRisk) * 100 : 100 / buckets.length}%`, backgroundColor: b.color }}
            title={`${b.label}: ${formatMoney(b.amount)} (${b.count} case${b.count === 1 ? '' : 's'})`}
          />
        ))}
      </div>
      <div className="mt-3 flex flex-wrap gap-x-6 gap-y-2 text-sm">
        {buckets.map((b) => (
          <div key={b.key} className="flex items-center gap-1.5">
            <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: b.color }} aria-hidden />
            <span className="text-slate-700">
              {b.label}: <span className="font-medium text-slate-900">{formatMoney(b.amount)}</span>{' '}
              <span className="text-slate-500">
                ({b.count} case{b.count === 1 ? '' : 's'})
              </span>
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}
