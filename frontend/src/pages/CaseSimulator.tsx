import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getGuardrailPolicy, simulateCase } from '../api/client'
import type { GuardrailPolicy, ReasoningStep, SimulateResponse } from '../api/types'
import { ReasoningChain } from '../components/ReasoningChain'
import { FAILURE_CODE_OPTIONS, failureCodeLabel, formatConfidence, tierLabel } from '../lib/format'

/**
 * Maps the flat SimulateResponseDTO into synthetic ReasoningSteps so the
 * simulator can reuse ReasoningChain as-is for visual consistency with Case
 * Detail View. Only 3 steps exist — diagnosis, strategy, guardrail — since
 * the simulator deliberately stops before execution/promise.
 *
 * The guardrail step's `decision` text deliberately deviates from real
 * cases' terser "held rule=X" convention: it embeds BOTH the rule fired and
 * the full human-readable reason directly in the always-visible headline
 * text, not just inside the collapsed "Why?" evidence list. A judge testing
 * an out-of-window hour should see *why* immediately, not have to expand
 * anything to find out.
 */
function toReasoningSteps(result: SimulateResponse): ReasoningStep[] {
  const now = new Date().toISOString()

  const guardrailState = result.guardrailAllowed ? 'Allowed' : result.guardrailHeld ? 'Held' : 'Blocked'
  const guardrailDecision = result.guardrailRuleFired
    ? `${guardrailState} — ${result.guardrailRuleFired}: ${result.guardrailReason}`
    : `${guardrailState}: ${result.guardrailReason}`

  return [
    {
      agentName: 'diagnosis',
      decision: result.rootCause,
      confidence: result.confidence,
      alternatives: result.diagnosisEvidence,
      timestamp: now,
    },
    {
      agentName: 'strategy',
      decision: result.tierChosen,
      alternatives: result.strategyAlternatives,
      timestamp: now,
    },
    {
      agentName: 'guardrail',
      decision: guardrailDecision,
      alternatives: result.guardrailEvidence,
      timestamp: now,
    },
  ]
}

export function CaseSimulator() {
  const [failureCode, setFailureCode] = useState(FAILURE_CODE_OPTIONS[0].code)
  const [historyScore, setHistoryScore] = useState(60)
  const [disputedFlag, setDisputedFlag] = useState(false)
  const [doNotContact, setDoNotContact] = useState(false)
  const [simulatedHour, setSimulatedHour] = useState(12)
  const [priorContactAttempts, setPriorContactAttempts] = useState(0)

  const [policy, setPolicy] = useState<GuardrailPolicy | null>(null)
  const [result, setResult] = useState<SimulateResponse | null>(null)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getGuardrailPolicy()
      .then(setPolicy)
      .catch(() => {
        /* purely a hint next to the Hour field; fine to omit silently on failure */
      })
  }, [])

  const handleRun = async () => {
    setRunning(true)
    setError(null)
    setResult(null)
    try {
      const res = await simulateCase({
        failureCode,
        historyScore,
        disputedFlag,
        doNotContact,
        simulatedHour,
        priorContactAttempts,
      })
      setResult(res)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Simulation failed.')
    } finally {
      setRunning(false)
    }
  }

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <Link to="/" className="text-sm text-slate-500 hover:underline">
        ← Back to summary
      </Link>

      <div>
        <h1 className="text-2xl font-bold text-slate-900">Case Simulator</h1>
        <p className="text-sm text-slate-500">
          Test the real diagnosis, strategy, and guardrail logic against a hypothetical case — no data is saved.
        </p>
      </div>

      <div className="space-y-4 rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="failureCode">
            Failure Code
          </label>
          <select
            id="failureCode"
            value={failureCode}
            onChange={(e) => setFailureCode(e.target.value)}
            disabled={disputedFlag}
            className="w-full rounded-md border border-slate-300 px-3 py-1.5 text-sm disabled:cursor-not-allowed disabled:bg-slate-100 disabled:text-slate-400"
          >
            {FAILURE_CODE_OPTIONS.map((opt) => (
              <option key={opt.code} value={opt.code}>
                {failureCodeLabel(opt.code)}
              </option>
            ))}
          </select>
          {disputedFlag && (
            <p className="mt-1 text-xs text-slate-500">Ignored — disputed cases always escalate regardless of failure code.</p>
          )}
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="historyScore">
            History Score (0–100)
          </label>
          <input
            id="historyScore"
            type="number"
            min={0}
            max={100}
            value={historyScore}
            onChange={(e) => setHistoryScore(Number(e.target.value))}
            className="w-full rounded-md border border-slate-300 px-3 py-1.5 text-sm"
          />
        </div>

        <div className="flex flex-wrap gap-6">
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input type="checkbox" checked={disputedFlag} onChange={(e) => setDisputedFlag(e.target.checked)} />
            Disputed
          </label>
          <label className="flex items-center gap-2 text-sm text-slate-700">
            <input type="checkbox" checked={doNotContact} onChange={(e) => setDoNotContact(e.target.checked)} />
            Do Not Contact
          </label>
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="simulatedHour">
            Hour of Day (0–23)
          </label>
          <input
            id="simulatedHour"
            type="number"
            min={0}
            max={23}
            value={simulatedHour}
            onChange={(e) => setSimulatedHour(Number(e.target.value))}
            className="w-full rounded-md border border-slate-300 px-3 py-1.5 text-sm"
          />
          {policy && (
            <p className="mt-1 text-xs text-slate-500">
              Current allowed contact window: {String(policy.contactWindowStart).padStart(2, '0')}:00–
              {String(policy.contactWindowEnd).padStart(2, '0')}:00
            </p>
          )}
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-slate-700" htmlFor="priorContactAttempts">
            Prior Contact Attempts
          </label>
          <input
            id="priorContactAttempts"
            type="number"
            min={0}
            value={priorContactAttempts}
            onChange={(e) => setPriorContactAttempts(Number(e.target.value))}
            className="w-full rounded-md border border-slate-300 px-3 py-1.5 text-sm"
          />
          {policy && <p className="mt-1 text-xs text-slate-500">Cap: {policy.maxContactAttempts} attempts</p>}
        </div>

        <button
          type="button"
          onClick={handleRun}
          disabled={running}
          className="rounded-md bg-slate-900 px-4 py-2 text-sm font-semibold text-white hover:bg-slate-700 disabled:opacity-50"
        >
          {running ? 'Running…' : 'Run Simulation'}
        </button>
      </div>

      {error && <p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</p>}

      {result && (
        <div className="space-y-4">
          <div className="rounded-lg border-2 border-dashed border-slate-300 bg-slate-50 p-4">
            <p className="text-xs font-bold tracking-wide text-slate-500 uppercase">Simulated — not a real transaction</p>
            <p className="mt-1 text-sm text-slate-700">
              Final tier: <span className="font-semibold text-slate-900">{tierLabel(result.guardrailFinalTier)}</span> —{' '}
              <span className="font-semibold text-slate-900">
                {result.guardrailAllowed ? 'Allowed' : result.guardrailHeld ? 'Held' : 'Blocked'}
              </span>
              {result.guardrailRuleFired && (
                <>
                  {' '}
                  (rule: <span className="font-mono text-xs">{result.guardrailRuleFired}</span>)
                </>
              )}
              {' — '}
              {formatConfidence(result.confidence)} diagnosis confidence
            </p>
          </div>

          <ReasoningChain steps={toReasoningSteps(result)} />
        </div>
      )}
    </div>
  )
}
