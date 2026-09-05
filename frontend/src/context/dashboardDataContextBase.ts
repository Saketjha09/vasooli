import { createContext } from 'react'
import type { BatchSummary, CaseSummary } from '../api/types'

export interface DashboardDataContextValue {
  cases: CaseSummary[]
  summary: BatchSummary | null
  loading: boolean
  running: boolean
  error: string | null
  runBatch: () => Promise<void>
}

export const DashboardDataContext = createContext<DashboardDataContextValue | null>(null)
