import { Link } from 'react-router-dom'
import type { CaseSummary } from '../api/types'
import { formatConfidence, formatMoney, tierLabel } from '../lib/format'
import { StatusBadge } from './StatusBadge'

/** The case list table on the Summary View — each row links to its detail page. */
export function CaseTable({ cases }: { cases: CaseSummary[] }) {
  if (cases.length === 0) {
    return <p className="text-sm text-slate-500">No cases yet — run the batch to load the demo dataset.</p>
  }

  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50">
          <tr>
            <th className="px-4 py-2 text-left font-semibold text-slate-600">Root Cause</th>
            <th className="px-4 py-2 text-left font-semibold text-slate-600">Tier</th>
            <th className="px-4 py-2 text-left font-semibold text-slate-600">Confidence</th>
            <th className="px-4 py-2 text-left font-semibold text-slate-600">Status</th>
            <th className="px-4 py-2 text-right font-semibold text-slate-600">Amount</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white">
          {cases.map((c) => (
            <tr key={c.caseId} className="hover:bg-slate-50">
              <td className="px-4 py-2">
                <Link to={`/cases/${c.caseId}`} className="text-slate-900 hover:underline">
                  {c.rootCause}
                </Link>
              </td>
              <td className="px-4 py-2 text-slate-700">{tierLabel(c.tierChosen)}</td>
              <td className="px-4 py-2 text-slate-700">{formatConfidence(c.confidence)}</td>
              <td className="px-4 py-2">
                <StatusBadge status={c.status} />
              </td>
              <td className="px-4 py-2 text-right text-slate-700">{formatMoney(c.amount)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
