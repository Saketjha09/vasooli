import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getCase, getCaseAudit } from '../api/client'
import type { AuditEntry, CaseDetail } from '../api/types'
import { MessagePreview } from '../components/MessagePreview'
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
      <Link to="/" className="text-sm text-ink-muted hover:underline">
        ← Back to summary
      </Link>

      {loading && <p className="text-sm text-ink-secondary">Loading…</p>}
      {error && (
        <p className="rounded-card bg-critical-bg p-3 text-sm text-critical-text">{error}</p>
      )}

      {caseDetail && (
        <>
          <div className="rounded-card bg-white p-5 shadow-card">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h1 className="text-xl font-bold tracking-tight text-ink">{caseDetail.rootCause}</h1>
              <StatusBadge status={caseDetail.status} />
            </div>
            <dl className="mt-3 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
              <div>
                <dt className="text-ink-muted">Tier Chosen</dt>
                <dd className="font-medium text-ink">{tierLabel(caseDetail.tierChosen)}</dd>
              </div>
              <div>
                <dt className="text-ink-muted">Confidence</dt>
                <dd className="font-medium text-ink">{formatConfidence(caseDetail.confidence)}</dd>
              </div>
              <div>
                <dt className="text-ink-muted">Amount</dt>
                <dd className="font-medium text-ink">{formatMoney(caseDetail.amount)}</dd>
              </div>
              <div>
                <dt className="text-ink-muted">Case ID</dt>
                <dd className="truncate font-mono text-xs text-ink-muted">{caseDetail.caseId}</dd>
              </div>
            </dl>
          </div>

          <div>
            <h2 className="mb-3 text-xs font-bold tracking-wide text-ink-muted uppercase">Reasoning Chain</h2>
            <ReasoningChain steps={caseDetail.reasoningChain} />
          </div>

          <MessagePreview
            rootCause={caseDetail.rootCause}
            tierChosen={caseDetail.tierChosen}
            transactionAmount={caseDetail.transactionAmount}
          />

          <details className="rounded-card bg-white p-5 shadow-card">
            <summary className="cursor-pointer text-sm font-semibold text-ink-secondary select-none">
              View raw audit log ({auditLog.length} entries)
            </summary>
            <div className="mt-3 overflow-x-auto">
              <table className="min-w-full divide-y divide-ring text-xs">
                <thead>
                  <tr className="text-left text-ink-muted">
                    <th className="py-1 pr-4">Timestamp</th>
                    <th className="py-1 pr-4">Agent</th>
                    <th className="py-1 pr-4">Decision</th>
                    <th className="py-1 pr-4">Confidence</th>
                    <th className="py-1 pr-4">Amount</th>
                    <th className="py-1">Alternatives</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-ring">
                  {auditLog.map((entry) => (
                    <tr key={entry.id}>
                      <td className="py-1 pr-4 whitespace-nowrap text-ink-muted">
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
