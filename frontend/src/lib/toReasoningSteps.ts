import type { ReasoningStep, SimulateResponse } from '../api/types'

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
export function toReasoningSteps(result: SimulateResponse): ReasoningStep[] {
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
