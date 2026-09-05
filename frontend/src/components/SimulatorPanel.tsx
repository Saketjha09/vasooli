import { useState } from 'react'
import { simulateCase } from '../api/client'
import type { GuardrailPolicy, SimulateResponse } from '../api/types'
import { FAILURE_CODE_OPTIONS, failureCodeLabel, formatConfidence, tierLabel } from '../lib/format'
import { toReasoningSteps } from '../lib/toReasoningSteps'
import { ReasoningChain } from './ReasoningChain'

/**
 * One simulator form + result. Extracted from CaseSimulator so the page can
 * render either one panel (default) or two side by side (Compare mode) —
 * each panel owns its own independent inputs/result/running/error state.
 */
export function SimulatorPanel({ policy, label }: { policy: GuardrailPolicy | null; label?: string }) {
  const [failureCode, setFailureCode] = useState(FAILURE_CODE_OPTIONS[0].code)
  const [historyScore, setHistoryScore] = useState(60)
  const [disputedFlag, setDisputedFlag] = useState(false)
  const [doNotContact, setDoNotContact] = useState(false)
  const [simulatedHour, setSimulatedHour] = useState(12)
  const [priorContactAttempts, setPriorContactAttempts] = useState(0)

  const [result, setResult] = useState<SimulateResponse | null>(null)
  const [running, setRunning] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const idPrefix = label ? `${label}-` : ''

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
    <div className="space-y-4">
      {label && <p className="text-xs font-bold tracking-wide text-ink-muted uppercase">{label}</p>}

      <div className="space-y-4 rounded-card bg-white p-5 shadow-card">
        <div>
          <label className="mb-1 block text-sm font-medium text-ink-secondary" htmlFor={`${idPrefix}failureCode`}>
            Failure Code
          </label>
          <select
            id={`${idPrefix}failureCode`}
            value={failureCode}
            onChange={(e) => setFailureCode(e.target.value)}
            disabled={disputedFlag}
            className="w-full rounded-control border border-ring px-3 py-1.5 text-sm disabled:cursor-not-allowed disabled:bg-page disabled:text-ink-muted"
          >
            {FAILURE_CODE_OPTIONS.map((opt) => (
              <option key={opt.code} value={opt.code}>
                {failureCodeLabel(opt.code)}
              </option>
            ))}
          </select>
          {disputedFlag && (
            <p className="mt-1 text-xs text-ink-muted">Ignored — disputed cases always escalate regardless of failure code.</p>
          )}
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-ink-secondary" htmlFor={`${idPrefix}historyScore`}>
            History Score (0–100)
          </label>
          <input
            id={`${idPrefix}historyScore`}
            type="number"
            min={0}
            max={100}
            value={historyScore}
            onChange={(e) => setHistoryScore(Number(e.target.value))}
            className="w-full rounded-control border border-ring px-3 py-1.5 text-sm"
          />
          <p className="mt-1 text-xs text-ink-muted">
            Only changes the outcome for willful_nonpayment cases — 40 or above gets an 8% incentive offer instead of
            escalation.
          </p>
        </div>

        <div className="flex flex-wrap gap-6">
          <div>
            <label className="flex items-center gap-2 text-sm text-ink-secondary">
              <input type="checkbox" checked={disputedFlag} onChange={(e) => setDisputedFlag(e.target.checked)} />
              Disputed
            </label>
            <p className="mt-1 text-xs text-ink-muted">Overrides every other field — always escalates, blocked from contact.</p>
          </div>
          <div>
            <label className="flex items-center gap-2 text-sm text-ink-secondary">
              <input type="checkbox" checked={doNotContact} onChange={(e) => setDoNotContact(e.target.checked)} />
              Do Not Contact
            </label>
            <p className="mt-1 text-xs text-ink-muted">Hard block on contact, independent of Disputed — also escalates.</p>
          </div>
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-ink-secondary" htmlFor={`${idPrefix}simulatedHour`}>
            Hour of Day (0–23)
          </label>
          <input
            id={`${idPrefix}simulatedHour`}
            type="number"
            min={0}
            max={23}
            value={simulatedHour}
            onChange={(e) => setSimulatedHour(Number(e.target.value))}
            className="w-full rounded-control border border-ring px-3 py-1.5 text-sm"
          />
          {policy && (
            <p className="mt-1 text-xs text-ink-muted">
              Current allowed contact window: {String(policy.contactWindowStart).padStart(2, '0')}:00–
              {String(policy.contactWindowEnd).padStart(2, '0')}:00
            </p>
          )}
        </div>

        <div>
          <label className="mb-1 block text-sm font-medium text-ink-secondary" htmlFor={`${idPrefix}priorContactAttempts`}>
            Prior Contact Attempts
          </label>
          <input
            id={`${idPrefix}priorContactAttempts`}
            type="number"
            min={0}
            value={priorContactAttempts}
            onChange={(e) => setPriorContactAttempts(Number(e.target.value))}
            className="w-full rounded-control border border-ring px-3 py-1.5 text-sm"
          />
          {policy && <p className="mt-1 text-xs text-ink-muted">Cap: {policy.maxContactAttempts} attempts</p>}
        </div>

        <button
          type="button"
          onClick={handleRun}
          disabled={running}
          className="rounded-control bg-ink px-4 py-2 text-sm font-semibold text-white hover:bg-ink-secondary disabled:opacity-50"
        >
          {running ? 'Running…' : 'Run Simulation'}
        </button>
      </div>

      {error && <p className="rounded-card bg-critical-bg p-3 text-sm text-critical-text">{error}</p>}

      {result && (
        <div className="space-y-4">
          <div className="rounded-card border-2 border-dashed border-ink-muted bg-page p-4">
            <p className="text-xs font-bold tracking-wide text-ink-muted uppercase">Simulated — not a real transaction</p>
            <p className="mt-1 text-sm text-ink-secondary">
              Final tier: <span className="font-semibold text-ink">{tierLabel(result.guardrailFinalTier)}</span> —{' '}
              <span className="font-semibold text-ink">
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
