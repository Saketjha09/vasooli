import { useContext } from 'react'
import { DashboardDataContext, type DashboardDataContextValue } from './dashboardDataContextBase'

export function useDashboardData(): DashboardDataContextValue {
  const ctx = useContext(DashboardDataContext)
  if (!ctx) {
    throw new Error('useDashboardData must be used within a DashboardDataProvider')
  }
  return ctx
}
