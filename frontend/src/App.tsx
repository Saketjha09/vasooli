import { Route, Routes } from 'react-router-dom'
import { CaseDetailView } from './pages/CaseDetailView'
import { CaseSimulator } from './pages/CaseSimulator'
import { GuardrailSpotlight } from './pages/GuardrailSpotlight'
import { SummaryView } from './pages/SummaryView'

export default function App() {
  return (
    <div className="min-h-screen bg-slate-50">
      <Routes>
        <Route path="/" element={<SummaryView />} />
        <Route path="/cases/:id" element={<CaseDetailView />} />
        <Route path="/guardrail-spotlight" element={<GuardrailSpotlight />} />
        <Route path="/simulate" element={<CaseSimulator />} />
      </Routes>
    </div>
  )
}
