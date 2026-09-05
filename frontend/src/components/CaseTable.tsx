import { Fragment, useMemo, useState, type Dispatch, type SetStateAction } from 'react'
import { Link } from 'react-router-dom'
import { getCase } from '../api/client'
import type { CaseDetail, CaseSummary } from '../api/types'
import {
  agentLabel,
  formatConfidence,
  formatMoney,
  isTierUpgraded,
  rootCauseLabel,
  statusLabel,
  tierLabel,
} from '../lib/format'
import { StatusBadge } from './StatusBadge'

type SortColumn = 'rootCause' | 'tierChosen' | 'status' | 'confidence' | 'transactionAmount' | 'amount'
type SortDirection = 'asc' | 'desc'

const TIER_OPTIONS = ['silent_retry', 'nudge', 'incentivized_nudge', 'escalate']
const STATUS_OPTIONS = ['processing', 'recovered', 'held', 'open', 'escalated', 'error']
const ROOT_CAUSE_OPTIONS = [
  'transient_gateway',
  'card_expired',
  'insufficient_funds',
  'checkout_friction',
  'willful_nonpayment',
  'disputed',
]

function FilterPills({
  options,
  selected,
  onToggle,
  labelFor,
}: {
  options: string[]
  selected: Set<string>
  onToggle: (value: string) => void
  labelFor: (value: string) => string
}) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {options.map((opt) => {
        const active = selected.has(opt)
        return (
          <button
            key={opt}
            type="button"
            onClick={() => onToggle(opt)}
            className={`rounded-full border px-2.5 py-1 text-xs font-medium transition-colors ${
              active
                ? 'border-ink bg-ink text-white'
                : 'border-ring bg-white text-ink-secondary hover:bg-page'
            }`}
          >
            {labelFor(opt)}
          </button>
        )
      })}
    </div>
  )
}

function SortHeader({
  column,
  label,
  sortColumn,
  sortDirection,
  onSort,
  align = 'left',
}: {
  column: SortColumn
  label: string
  sortColumn: SortColumn
  sortDirection: SortDirection
  onSort: (column: SortColumn) => void
  align?: 'left' | 'right'
}) {
  const active = sortColumn === column
  return (
    <th className={`px-3 py-2 font-semibold text-ink-secondary ${align === 'right' ? 'text-right' : 'text-left'}`}>
      <button type="button" onClick={() => onSort(column)} className="inline-flex items-center gap-1 hover:text-ink">
        {label}
        {active && <span aria-hidden>{sortDirection === 'asc' ? '▲' : '▼'}</span>}
      </button>
    </th>
  )
}

/** Pure display — fetching/caching is owned by CaseTable (see previewCache) so re-expanding a row doesn't refetch. */
function ExpandedPreview({ detail }: { detail: CaseDetail | 'loading' | 'error' }) {
  if (detail === 'loading') {
    return <p className="p-3 text-xs text-ink-muted">Loading preview…</p>
  }
  if (detail === 'error') {
    return <p className="p-3 text-xs text-critical-text">Failed to load preview.</p>
  }

  const byAgent = (name: string) => detail.reasoningChain.find((s) => s.agentName === name)
  const diagnosis = byAgent('diagnosis')
  const strategy = byAgent('strategy')
  const guardrail = byAgent('guardrail')
  const execution = byAgent('execution')

  return (
    <div className="space-y-1 border-t border-ring bg-page p-3 text-xs text-ink-secondary">
      {diagnosis && (
        <p>
          <span className="font-semibold text-ink">{agentLabel('diagnosis')}:</span>{' '}
          {diagnosis.alternatives[0] ?? diagnosis.decision}
        </p>
      )}
      {strategy && (
        <p>
          <span className="font-semibold text-ink">{agentLabel('strategy')}:</span> chose {tierLabel(strategy.decision)}
          {strategy.alternatives[0] && <> — {strategy.alternatives[0]}</>}
        </p>
      )}
      {guardrail && (
        <p>
          <span className="font-semibold text-ink">{agentLabel('guardrail')}:</span> {guardrail.decision}
        </p>
      )}
      {execution && (
        <p>
          <span className="font-semibold text-ink">{agentLabel('execution')}:</span> {execution.decision}
        </p>
      )}
      <Link to={`/cases/${detail.caseId}`} className="inline-block pt-1 font-medium text-ink hover:underline">
        View full case →
      </Link>
    </div>
  )
}

/** The case list table on the Summary View: sortable, filterable, searchable, with expandable inline previews. */
export function CaseTable({ cases }: { cases: CaseSummary[] }) {
  const [search, setSearch] = useState('')
  const [sortColumn, setSortColumn] = useState<SortColumn>('rootCause')
  const [sortDirection, setSortDirection] = useState<SortDirection>('asc')
  const [tierFilter, setTierFilter] = useState<Set<string>>(new Set())
  const [statusFilter, setStatusFilter] = useState<Set<string>>(new Set())
  const [rootCauseFilter, setRootCauseFilter] = useState<Set<string>>(new Set())
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [previewCache, setPreviewCache] = useState<Record<string, CaseDetail | 'loading' | 'error'>>({})

  // Functional updater form, deliberately: computing `next` from a `set`
  // value captured by closure (the previous version) breaks under React's
  // automatic batching when multiple pill clicks land close enough together
  // to be batched before a re-render occurs between them — each call would
  // compute `next` from the SAME stale base, silently dropping all but the
  // last click's effect. Reading from `prev` inside the updater always sees
  // the latest pending state, regardless of click timing.
  const toggleSet = (setter: Dispatch<SetStateAction<Set<string>>>, value: string) => {
    setter((prev) => {
      const next = new Set(prev)
      if (next.has(value)) next.delete(value)
      else next.add(value)
      return next
    })
  }

  const handleSort = (column: SortColumn) => {
    if (sortColumn === column) {
      setSortDirection((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortColumn(column)
      setSortDirection('asc')
    }
  }

  const toggleExpanded = (caseId: string) => {
    const next = new Set(expanded)
    if (next.has(caseId)) {
      next.delete(caseId)
    } else {
      next.add(caseId)
      if (!(caseId in previewCache)) {
        setPreviewCache((prev) => ({ ...prev, [caseId]: 'loading' }))
        getCase(caseId)
          .then((detail) => setPreviewCache((prev) => ({ ...prev, [caseId]: detail })))
          .catch(() => setPreviewCache((prev) => ({ ...prev, [caseId]: 'error' })))
      }
    }
    setExpanded(next)
  }

  // "Every option in a category selected" is treated the SAME as "none
  // selected" (no filter on that dimension) explicitly, rather than relying
  // solely on the emergent fact that a case's value must be in a full-size
  // Set — that's a correct but implicit property, and stays correct even
  // if the option list and Set ever diverge for some other reason.
  const tierFilterActive = tierFilter.size > 0 && tierFilter.size < TIER_OPTIONS.length
  const statusFilterActive = statusFilter.size > 0 && statusFilter.size < STATUS_OPTIONS.length
  const rootCauseFilterActive = rootCauseFilter.size > 0 && rootCauseFilter.size < ROOT_CAUSE_OPTIONS.length

  const visibleCases = useMemo(() => {
    const q = search.trim().toLowerCase()
    let result = cases.filter((c) => {
      if (tierFilterActive && !tierFilter.has(c.tierChosen)) return false
      if (statusFilterActive && !statusFilter.has(c.status)) return false
      if (rootCauseFilterActive && !rootCauseFilter.has(c.rootCause)) return false
      if (q) {
        const haystack = `${c.rootCause} ${c.tierChosen} ${c.status} ${c.caseId}`.toLowerCase()
        if (!haystack.includes(q)) return false
      }
      return true
    })

    result = [...result].sort((a, b) => {
      let cmp = 0
      if (sortColumn === 'confidence' || sortColumn === 'transactionAmount' || sortColumn === 'amount') {
        cmp = a[sortColumn] - b[sortColumn]
      } else {
        cmp = String(a[sortColumn]).localeCompare(String(b[sortColumn]))
      }
      if (cmp === 0) cmp = a.transactionId.localeCompare(b.transactionId) // stable tie-break
      return sortDirection === 'asc' ? cmp : -cmp
    })

    return result
  }, [
    cases,
    search,
    tierFilter,
    statusFilter,
    rootCauseFilter,
    tierFilterActive,
    statusFilterActive,
    rootCauseFilterActive,
    sortColumn,
    sortDirection,
  ])

  const hasActiveFilters = search.trim() !== '' || tierFilter.size > 0 || statusFilter.size > 0 || rootCauseFilter.size > 0
  const clearFilters = () => {
    setSearch('')
    setTierFilter(new Set())
    setStatusFilter(new Set())
    setRootCauseFilter(new Set())
  }

  if (cases.length === 0) {
    return <p className="text-sm text-ink-secondary">No cases yet — run the batch to load the demo dataset.</p>
  }

  return (
    <div className="space-y-3">
      <div className="space-y-3 rounded-card bg-white p-4 shadow-card">
        <input
          type="text"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Search root cause, tier, status, or case ID…"
          className="w-full rounded-control border border-ring px-3 py-1.5 text-sm focus:border-ink-muted focus:outline-none"
        />
        <div className="flex flex-wrap gap-4">
          <div>
            <p className="mb-1 text-xs font-semibold tracking-wide text-ink-muted uppercase">Tier</p>
            <FilterPills options={TIER_OPTIONS} selected={tierFilter} onToggle={(v) => toggleSet(setTierFilter, v)} labelFor={tierLabel} />
          </div>
          <div>
            <p className="mb-1 text-xs font-semibold tracking-wide text-ink-muted uppercase">Status</p>
            <FilterPills options={STATUS_OPTIONS} selected={statusFilter} onToggle={(v) => toggleSet(setStatusFilter, v)} labelFor={statusLabel} />
          </div>
          <div>
            <p className="mb-1 text-xs font-semibold tracking-wide text-ink-muted uppercase">Root Cause</p>
            <FilterPills
              options={ROOT_CAUSE_OPTIONS}
              selected={rootCauseFilter}
              onToggle={(v) => toggleSet(setRootCauseFilter, v)}
              labelFor={rootCauseLabel}
            />
          </div>
        </div>
        <div className="flex items-center justify-between">
          <p className="text-xs text-ink-muted">
            Showing {visibleCases.length} of {cases.length} cases
          </p>
          {hasActiveFilters && (
            <button type="button" onClick={clearFilters} className="text-xs font-medium text-ink-secondary underline hover:text-ink">
              Clear filters
            </button>
          )}
        </div>
      </div>

      <div className="overflow-x-auto rounded-card bg-white shadow-card">
        <table className="min-w-full divide-y divide-ring text-sm">
          <thead className="bg-page">
            <tr>
              <th className="w-8 px-3 py-2" />
              <SortHeader column="rootCause" label="Root Cause" sortColumn={sortColumn} sortDirection={sortDirection} onSort={handleSort} />
              <SortHeader column="tierChosen" label="Tier" sortColumn={sortColumn} sortDirection={sortDirection} onSort={handleSort} />
              <SortHeader column="confidence" label="Confidence" sortColumn={sortColumn} sortDirection={sortDirection} onSort={handleSort} />
              <SortHeader column="status" label="Status" sortColumn={sortColumn} sortDirection={sortDirection} onSort={handleSort} />
              <SortHeader
                column="transactionAmount"
                label="Amount"
                sortColumn={sortColumn}
                sortDirection={sortDirection}
                onSort={handleSort}
                align="right"
              />
              <SortHeader
                column="amount"
                label="Recovered"
                sortColumn={sortColumn}
                sortDirection={sortDirection}
                onSort={handleSort}
                align="right"
              />
            </tr>
          </thead>
          <tbody className="divide-y divide-ring bg-white">
            {visibleCases.map((c) => {
              const isExpanded = expanded.has(c.caseId)
              const upgraded = isTierUpgraded(c)
              return (
                <Fragment key={c.caseId}>
                  <tr className="hover:bg-page">
                    <td className="px-3 py-1.5">
                      <button
                        type="button"
                        onClick={() => toggleExpanded(c.caseId)}
                        aria-label={isExpanded ? 'Collapse preview' : 'Expand preview'}
                        className="text-ink-muted hover:text-ink"
                      >
                        {isExpanded ? '▾' : '▸'}
                      </button>
                    </td>
                    <td className="px-3 py-1.5">
                      <Link to={`/cases/${c.caseId}`} className="text-ink hover:underline">
                        {rootCauseLabel(c.rootCause)}
                      </Link>
                    </td>
                    <td className="px-3 py-1.5 text-ink-secondary">
                      <span className="inline-flex items-center gap-1.5">
                        {tierLabel(c.tierChosen)}
                        {upgraded && (
                          <span
                            className="rounded-full bg-info-bg px-1.5 py-0.5 text-[10px] font-semibold text-info-text"
                            title="Upgraded from nudge after it went unanswered (MRD tier 3)"
                          >
                            ⇧ Upgraded
                          </span>
                        )}
                      </span>
                    </td>
                    <td className="px-3 py-1.5 text-ink-secondary">{formatConfidence(c.confidence)}</td>
                    <td className="px-3 py-1.5">
                      <StatusBadge status={c.status} />
                    </td>
                    <td className="px-3 py-1.5 text-right text-ink-secondary">{formatMoney(c.transactionAmount)}</td>
                    <td className="px-3 py-1.5 text-right text-ink-secondary">
                      {c.status === 'recovered' ? formatMoney(c.amount) : <span className="text-ink-muted">—</span>}
                    </td>
                  </tr>
                  {isExpanded && (
                    <tr>
                      <td colSpan={7} className="p-0">
                        <ExpandedPreview detail={previewCache[c.caseId] ?? 'loading'} />
                      </td>
                    </tr>
                  )}
                </Fragment>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
