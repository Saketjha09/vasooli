import { useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { getBatchSummary, listCases, runBatch as runBatchRequest } from '../api/client'
import type { BatchSummary, CaseSummary } from '../api/types'
import { DashboardDataContext } from './dashboardDataContextBase'

/**
 * Shared cases/summary state for Overview and Cases — the two pages that
 * both need the same data and both have the "click Run Batch while already
 * viewing this page" staleness problem a persistent sidebar introduces.
 * Guardrail Spotlight / Case Detail / Simulator each fetch their own data
 * independently on mount and don't need this: navigating to them (a route
 * change, hence a remount) already gets them fresh data, so there's nothing
 * for a shared store to fix there.
 *
 * The context object itself lives in ./dashboardDataContextBase and the hook
 * that reads it in ./useDashboardData — a file exporting a component must
 * export components only for Fast Refresh to work, so the raw context and
 * the hook are deliberately split out into their own files.
 */
export function DashboardDataProvider({ children }: { children: ReactNode }) {
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

  const runBatch = useCallback(async () => {
    setRunning(true)
    setError(null)
    try {
      await runBatchRequest()
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to run the batch.')
    } finally {
      setRunning(false)
    }
  }, [loadData])

  const value = useMemo(
    () => ({ cases, summary, loading, running, error, runBatch }),
    [cases, summary, loading, running, error, runBatch],
  )

  return <DashboardDataContext.Provider value={value}>{children}</DashboardDataContext.Provider>
}
