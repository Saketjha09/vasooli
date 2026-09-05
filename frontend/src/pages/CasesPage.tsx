import { CaseTable } from '../components/CaseTable'
import { InsightsPanel } from '../components/InsightsPanel'
import { useDashboardData } from '../context/useDashboardData'

export function CasesPage() {
  const { cases, loading, error } = useDashboardData()

  return (
    <div className="mx-auto max-w-6xl space-y-6 p-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight text-ink">Cases</h1>
        <p className="text-sm text-ink-secondary">Every case from the last batch run, sortable and filterable.</p>
      </div>

      {error && <p className="rounded-card bg-critical-bg p-3 text-sm text-critical-text">{error}</p>}

      {loading ? (
        <p className="text-sm text-ink-secondary">Loading…</p>
      ) : (
        <>
          <div>
            <h2 className="mb-3 text-xs font-bold tracking-wide text-ink-muted uppercase">Insights</h2>
            <InsightsPanel cases={cases} />
          </div>

          <div>
            <h2 className="mb-3 text-xs font-bold tracking-wide text-ink-muted uppercase">Cases</h2>
            <CaseTable cases={cases} />
          </div>
        </>
      )}
    </div>
  )
}
