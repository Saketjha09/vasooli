import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getGuardrailPolicy } from '../api/client'
import type { GuardrailPolicy } from '../api/types'
import { SimulatorPanel } from '../components/SimulatorPanel'

export function CaseSimulator() {
  const [policy, setPolicy] = useState<GuardrailPolicy | null>(null)
  const [compareMode, setCompareMode] = useState(false)

  useEffect(() => {
    getGuardrailPolicy()
      .then(setPolicy)
      .catch(() => {
        /* purely a hint next to the Hour field; fine to omit silently on failure */
      })
  }, [])

  return (
    <div className="mx-auto max-w-6xl space-y-6 p-6">
      <Link to="/" className="text-sm text-ink-muted hover:underline">
        ← Back to summary
      </Link>

      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-ink">Case Simulator</h1>
          <p className="text-sm text-ink-secondary">
            Test the real diagnosis, strategy, and guardrail logic against a hypothetical case — no data is saved.
          </p>
        </div>
        <label className="flex items-center gap-2 text-sm font-medium text-ink-secondary">
          <input type="checkbox" checked={compareMode} onChange={(e) => setCompareMode(e.target.checked)} />
          Compare two scenarios
        </label>
      </div>

      {compareMode ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <SimulatorPanel policy={policy} label="Scenario A" />
          <SimulatorPanel policy={policy} label="Scenario B" />
        </div>
      ) : (
        <div className="max-w-3xl">
          <SimulatorPanel policy={policy} />
        </div>
      )}
    </div>
  )
}
