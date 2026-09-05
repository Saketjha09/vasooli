import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getCase, getCaseAudit } from '../api/client'
import type { AuditEntry, CaseDetail } from '../api/types'
import { ReasoningChain } from '../components/ReasoningChain'
import { StatusBadge } from '../components/StatusBadge'
import { agentLabel, formatConfidence, formatMoney, formatTimestamp, tierLabel } from '../lib/format'

export function CaseDetailView() {
  const { id } = useParams<{ id: string }>()
  const [caseDetail, setCaseDetail] = useState<CaseDetail | null>(null)
  const [auditLog, setAuditLog] = useState<AuditEntry[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    setLoading(true)
    setError(null)
    Promise.all([getCase(id), getCaseAudit(id)])
      .then(([detail, audit]) => {
        setCaseDetail(detail)
        setAuditLog(audit)
      })
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load case.'))
      .finally(() => setLoading(false))
  }, [id])

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <Link to="/" className="text-sm text-slate-500 hover:underline">
        ← Back to summary
      </Link>

      {loading && <p className="text-sm text-slate-500">Loading…</p>}
      {error && (
        <p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</p>
      )}

      {caseDetail && (
        <>
          <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h1 className="text-xl font-bold text-slate-900">{caseDetail.rootCause}</h1>
              <StatusBadge status={caseDetail.status} />
            </div>
            <dl className="mt-3 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
              <div>
                <dt className="text-slate-500">Tier Chosen</dt>
                <dd className="font-medium text-slate-900">{tierLabel(caseDetail.tierChosen)}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Confidence</dt>
                <dd className="font-medium text-slate-900">{formatConfidence(caseDetail.confidence)}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Amount</dt>
                <dd className="font-medium text-slate-900">{formatMoney(caseDetail.amount)}</dd>
              </div>
              <div>
                <dt className="text-slate-500">Case ID</dt>
                <dd className="truncate font-mono text-xs text-slate-500">{caseDetail.caseId}</dd>
              </div>
            </dl>
          </div>

          <div>
            <h2 className="mb-3 text-lg font-semibold text-slate-900">Reasoning Chain</h2>
            <ReasoningChain steps={caseDetail.reasoningChain} />
          </div>

          <details className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
            <summary className="cursor-pointer text-sm font-semibold text-slate-700 select-none">
              View raw audit log ({auditLog.length} entries)
            </summary>
            <div className="mt-3 overflow-x-auto">
              <table className="min-w-full divide-y divide-slate-200 text-xs">
                <thead>
                  <tr className="text-left text-slate-500">
                    <th className="py-1 pr-4">Timestamp</th>
                    <th className="py-1 pr-4">Agent</th>
                    <th className="py-1 pr-4">Decision</th>
                    <th className="py-1 pr-4">Confidence</th>
                    <th className="py-1 pr-4">Amount</th>
                    <th className="py-1">Alternatives</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {auditLog.map((entry) => (
                    <tr key={entry.id}>
                      <td className="py-1 pr-4 whitespace-nowrap text-slate-500">
                        {formatTimestamp(entry.timestamp)}
                      </td>
                      <td className="py-1 pr-4">{agentLabel(entry.agentName)}</td>
                      <td className="py-1 pr-4">{entry.decision}</td>
                      <td className="py-1 pr-4">
                        {entry.confidence !== undefined ? formatConfidence(entry.confidence) : '—'}
                      </td>
                      <td className="py-1 pr-4">
                        {entry.amount !== undefined ? formatMoney(entry.amount) : '—'}
                      </td>
                      <td className="py-1">{entry.alternatives.join('; ')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </details>
        </>
      )}
    </div>
  )
}
