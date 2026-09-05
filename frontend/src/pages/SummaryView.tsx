import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getBatchSummary, listCases, runBatch } from '../api/client'
import type { BatchSummary, CaseSummary } from '../api/types'
import { CaseTable } from '../components/CaseTable'
import { GuardrailPolicyCard } from '../components/GuardrailPolicyCard'
import { InsightsPanel } from '../components/InsightsPanel'
import { RecoveryBreakdown } from '../components/RecoveryBreakdown'
import { RootCauseChart } from '../components/RootCauseChart'
import { StatTile } from '../components/StatTile'
import { TierBreakdownChart } from '../components/TierBreakdownChart'
import { formatMoney } from '../lib/format'

export function SummaryView() {
  const [summary, setSummary] = useState<BatchSummary | null>(null)
  const [cases, setCases] = useState<CaseSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [summaryData, casesData] = await Promise.all([getBatchSummary(), listCases()])
      setSummary(summaryData)
      setCases(casesData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load dashboard data.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadData()
  }, [loadData])

  const handleRunBatch = async () => {
    setRunning(true)
    setError(null)
    try {
      await runBatch()
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to run the batch.')
    } finally {
      setRunning(false)
    }
  }

  const disputedCaseExists = cases.some((c) => c.rootCause === 'disputed')
  const escalatedCount = cases.filter((c) => c.status === 'escalated').length
  const rootCauseCounts = cases.reduce<Record<string, number>>((acc, c) => {
    acc[c.rootCause] = (acc[c.rootCause] ?? 0) + 1
    return acc
  }, {})

  return (
    <div className="mx-auto max-w-6xl space-y-6 p-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900">Vasooli — Batch Summary</h1>
          <p className="text-sm text-slate-500">AI revenue recovery: diagnosis, strategy, and guardrails, fully explainable.</p>
        </div>
        <button
          type="button"
          onClick={handleRunBatch}
          disabled={running}
          className="rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-700 disabled:opacity-50"
        >
          {running ? 'Running…' : 'Run Batch'}
        </button>
      </div>

      {error && (
        <p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</p>
      )}

      {loading ? (
        <p className="text-sm text-slate-500">Loading…</p>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            <StatTile label="Total Cases" value={String(summary?.totalCases ?? 0)} />
            <StatTile
              label="Money Recovered"
              value={formatMoney(summary?.moneyRecovered ?? 0)}
              accent="success"
            />
            <StatTile label="Escalated" value={String(escalatedCount)} accent={escalatedCount > 0 ? 'warning' : 'default'} />
          </div>

          <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
            <h2 className="mb-2 text-lg font-semibold text-slate-900">Recovery Breakdown</h2>
            <RecoveryBreakdown cases={cases} />
          </div>

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
              <h2 className="mb-2 text-lg font-semibold text-slate-900">Tier Breakdown</h2>
              <TierBreakdownChart tierBreakdown={summary?.tierBreakdown ?? {}} />
            </div>
            <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
              <h2 className="mb-2 text-lg font-semibold text-slate-900">Root Cause Distribution</h2>
              <RootCauseChart counts={rootCauseCounts} />
            </div>
          </div>

          <div className="space-y-2">
            <GuardrailPolicyCard />
            <Link to="/simulate" className="inline-block text-sm font-medium text-slate-600 hover:text-slate-900 hover:underline">
              Try the simulator — test a hypothetical case against these rules →
            </Link>
          </div>

          {disputedCaseExists && (
            <Link
              to="/guardrail-spotlight"
              className="block rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm font-medium text-amber-900 hover:bg-amber-100"
            >
              1 case required a policy lockout — see why the system refused to act →
            </Link>
          )}

          <div>
            <h2 className="mb-2 text-lg font-semibold text-slate-900">Insights</h2>
            <InsightsPanel cases={cases} />
          </div>

          <div>
            <h2 className="mb-2 text-lg font-semibold text-slate-900">Cases</h2>
            <CaseTable cases={cases} />
          </div>
        </>
      )}
    </div>
  )
}
