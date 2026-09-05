import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getGuardrailSpotlight } from '../api/client'
import type { CaseDetail, ReasoningStep } from '../api/types'
import { ReasoningChain } from '../components/ReasoningChain'
import { formatConfidence, tierLabel } from '../lib/format'

/**
 * The dedicated "graceful failure" callout (MRD section 3 / PRD FR8): the
 * disputed-case lockout. Leads with the guardrail step's rule and reason
 * before anything else, so a judge sees why the system refused to act in
 * the first glance — the full reasoning chain follows below for context.
 */
export function GuardrailSpotlight() {
  const [caseDetail, setCaseDetail] = useState<CaseDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    getGuardrailSpotlight()
      .then(setCaseDetail)
      .catch((err) => setError(err instanceof Error ? err.message : 'Failed to load the guardrail spotlight.'))
      .finally(() => setLoading(false))
  }, [])

  const guardrailStep: ReasoningStep | undefined = caseDetail?.reasoningChain.find(
    (step) => step.agentName === 'guardrail',
  )

  return (
    <div className="mx-auto max-w-3xl space-y-6 p-6">
      <Link to="/" className="text-sm text-slate-500 hover:underline">
        ← Back to summary
      </Link>

      {loading && <p className="text-sm text-slate-500">Loading…</p>}
      {error && (
        <p className="rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">{error}</p>
      )}

      {caseDetail && (
        <>
          <div className="rounded-lg border-2 border-red-300 bg-red-50 p-6 shadow-sm">
            <p className="text-xs font-bold tracking-wide text-red-700 uppercase">
              Policy Guardrail — Action Blocked
            </p>
            <h1 className="mt-1 text-2xl font-bold text-red-900">
              This case was locked before any recovery action could fire.
            </h1>
            <p className="mt-2 text-sm text-red-800">
              Root cause: <strong>disputed</strong> ({formatConfidence(caseDetail.confidence)} confidence). A live
              dispute/chargeback flag means contacting this customer automatically would be compliance-unsafe —
              the Policy Guardrail Agent detected this before Execution ever ran, and routed the case straight
              to <strong>{tierLabel(caseDetail.tierChosen)}</strong> instead.
            </p>
            {guardrailStep && (
              <div className="mt-4 rounded-md bg-white p-3 text-sm text-slate-800">
                <p className="font-semibold text-slate-900">{guardrailStep.decision}</p>
                <ul className="mt-1 list-inside list-disc text-slate-600">
                  {guardrailStep.alternatives.map((alt, i) => (
                    <li key={i}>{alt}</li>
                  ))}
                </ul>
              </div>
            )}
          </div>

          <div>
            <h2 className="mb-3 text-lg font-semibold text-slate-900">Full Reasoning Chain</h2>
            <ReasoningChain steps={caseDetail.reasoningChain} />
          </div>
        </>
      )}
    </div>
  )
}
