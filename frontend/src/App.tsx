import { Route, Routes } from 'react-router-dom'
import { DashboardDataProvider } from './context/DashboardDataContext'
import { Layout } from './layout/Layout'
import { CaseDetailView } from './pages/CaseDetailView'
import { CasesPage } from './pages/CasesPage'
import { CaseSimulator } from './pages/CaseSimulator'
import { GuardrailSpotlight } from './pages/GuardrailSpotlight'
import { OverviewPage } from './pages/OverviewPage'

export default function App() {
  return (
    <div className="min-h-screen bg-page">
      <DashboardDataProvider>
        <Routes>
          <Route element={<Layout />}>
            <Route path="/" element={<OverviewPage />} />
            <Route path="/cases" element={<CasesPage />} />
            <Route path="/cases/:id" element={<CaseDetailView />} />
            <Route path="/guardrail-spotlight" element={<GuardrailSpotlight />} />
            <Route path="/simulate" element={<CaseSimulator />} />
          </Route>
        </Routes>
      </DashboardDataProvider>
    </div>
  )
}
