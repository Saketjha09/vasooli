import { Link } from 'react-router-dom'
import { GuardrailPolicyCard } from '../components/GuardrailPolicyCard'
import { RecoveryBreakdown } from '../components/RecoveryBreakdown'
import { RootCauseChart } from '../components/RootCauseChart'
import { StatTile } from '../components/StatTile'
import { TierBreakdownChart } from '../components/TierBreakdownChart'
import { useDashboardData } from '../context/useDashboardData'
import { formatMoney } from '../lib/format'

export function OverviewPage() {
  const { cases, summary, loading, error } = useDashboardData()

  const disputedCaseExists = cases.some((c) => c.rootCause === 'disputed')
  const escalatedCount = cases.filter((c) => c.status === 'escalated').length
  const rootCauseCounts = cases.reduce<Record<string, number>>((acc, c) => {
    acc[c.rootCause] = (acc[c.rootCause] ?? 0) + 1
    return acc
  }, {})

  return (
    <div className="mx-auto max-w-6xl space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-ink">Overview</h1>
        <p className="text-sm text-ink-secondary">AI revenue recovery: diagnosis, strategy, and guardrails, fully explainable.</p>
      </div>

      {error && <p className="rounded-card bg-critical-bg p-3 text-sm text-critical-text">{error}</p>}

      {loading ? (
        <p className="text-sm text-ink-secondary">Loading…</p>
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

          <div className="rounded-card bg-white p-5 shadow-card">
            <h2 className="mb-3 text-xs font-bold tracking-wide text-ink-muted uppercase">Recovery Breakdown</h2>
            <RecoveryBreakdown cases={cases} />
          </div>

          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <div className="rounded-card bg-white p-5 shadow-card">
              <h2 className="mb-3 text-xs font-bold tracking-wide text-ink-muted uppercase">Tier Breakdown</h2>
              <TierBreakdownChart tierBreakdown={summary?.tierBreakdown ?? {}} />
            </div>
            <div className="rounded-card bg-white p-5 shadow-card">
              <h2 className="mb-3 text-xs font-bold tracking-wide text-ink-muted uppercase">Root Cause Distribution</h2>
              <RootCauseChart counts={rootCauseCounts} />
            </div>
          </div>

          <div className="space-y-2">
            <GuardrailPolicyCard />
            <Link to="/simulate" className="inline-block text-sm font-medium text-ink-secondary hover:text-ink hover:underline">
              Try the simulator — test a hypothetical case against these rules →
            </Link>
          </div>

          {disputedCaseExists && (
            <Link
              to="/guardrail-spotlight"
              className="block rounded-card bg-warn-bg p-4 text-sm font-medium text-warn-text hover:brightness-95"
            >
              1 case required a policy lockout — see why the system refused to act →
            </Link>
          )}
        </>
      )}
    </div>
  )
}
