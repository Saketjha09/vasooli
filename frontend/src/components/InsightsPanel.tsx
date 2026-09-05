import type { CaseSummary } from '../api/types'
import { formatMoney, insightSuggestion, rootCauseLabel } from '../lib/format'

/**
 * Aggregates OPEN cases (not held/processing — those are just timing
 * delays, not stuck) by root cause into templated next-step suggestions,
 * sorted by group size descending (biggest opportunity first).
 */
export function InsightsPanel({ cases }: { cases: CaseSummary[] }) {
  const openCases = cases.filter((c) => c.status === 'open')

  if (openCases.length === 0) {
    return (
      <div className="rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800">
        No open cases — the batch is fully processed.
      </div>
    )
  }

  const groups = new Map<string, CaseSummary[]>()
  for (const c of openCases) {
    const group = groups.get(c.rootCause) ?? []
    group.push(c)
    groups.set(c.rootCause, group)
  }

  const sortedGroups = Array.from(groups.entries())
    .map(([rootCause, group]) => ({
      rootCause,
      count: group.length,
      amountAtRisk: group.reduce((sum, c) => sum + c.transactionAmount, 0),
    }))
    .sort((a, b) => b.count - a.count)

  return (
    <ul className="space-y-2">
      {sortedGroups.map((g) => (
        <li key={g.rootCause} className="rounded-lg border border-slate-200 bg-white p-3 text-sm shadow-sm">
          <span className="font-semibold text-slate-900">
            {g.count} case{g.count === 1 ? '' : 's'} stuck on {rootCauseLabel(g.rootCause)}
          </span>{' '}
          <span className="text-slate-500">({formatMoney(g.amountAtRisk)} at risk)</span>
          <span className="text-slate-700"> — {insightSuggestion(g.rootCause)}.</span>
        </li>
      ))}
    </ul>
  )
}
