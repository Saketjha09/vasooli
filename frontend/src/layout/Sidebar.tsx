import { NavLink } from 'react-router-dom'
import { useDashboardData } from '../context/useDashboardData'

const NAV_ITEMS = [
  { to: '/', label: 'Overview', end: true },
  { to: '/cases', label: 'Cases', end: false },
  { to: '/guardrail-spotlight', label: 'Guardrail Spotlight', end: false },
  { to: '/simulate', label: 'Simulator', end: false },
]

/**
 * Persistent app-wide sidebar. Run Batch lives here, not on any one page —
 * it affects data across Overview and Cases both, via DashboardDataContext.
 */
export function Sidebar() {
  const { running, error, runBatch } = useDashboardData()

  return (
    <aside className="flex w-56 shrink-0 flex-col gap-6 border-r border-ring bg-white p-4">
      <div>
        <p className="text-lg font-bold tracking-tight text-ink">Vasooli</p>
        <p className="text-xs text-ink-muted">AI revenue recovery</p>
      </div>

      <button
        type="button"
        onClick={runBatch}
        disabled={running}
        className="rounded-control bg-ink px-4 py-2 text-sm font-semibold text-white hover:bg-ink-secondary disabled:opacity-50"
      >
        {running ? 'Running…' : 'Run Batch'}
      </button>

      {error && <p className="text-xs text-critical-text">{error}</p>}

      <nav className="flex flex-col gap-1">
        {NAV_ITEMS.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.end}
            className={({ isActive }) =>
              `rounded-control px-3 py-2 text-sm font-medium ${
                isActive ? 'bg-page text-ink' : 'text-ink-secondary hover:bg-page hover:text-ink'
              }`
            }
          >
            {item.label}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
