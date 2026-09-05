// Mirrors backend/internal/api/dto.go exactly — same field names (as
// camelCase JSON already produces), same semantics. Keep this file in sync
// if dto.go changes; it is the frontend's only source of truth for API
// shapes, not a guess.

/** Response to POST /api/batch/run. */
export interface BatchRunResponse {
  totalCases: number
  /** Keyed by final tier: "silent_retry" | "nudge" | "incentivized_nudge" | "escalate". */
  tierBreakdown: Record<string, number>
  moneyRecovered: number
}

/** Response to GET /api/batch/summary — same shape as BatchRunResponse, always freshly queried. */
export interface BatchSummary {
  totalCases: number
  tierBreakdown: Record<string, number>
  moneyRecovered: number
}

/**
 * Response to GET /api/guardrail/policy — the fixed caps guardrail.Check
 * enforces, sourced from server config, NOT derived from cases. Always
 * available and non-empty regardless of whether any batch has run yet.
 */
export interface GuardrailPolicy {
  maxContactAttempts: number
  maxDiscountPct: number
  contactWindowStart: number // hour, 0-23, inclusive
  contactWindowEnd: number // hour, 0-23, exclusive
}

/** One row of GET /api/cases. */
export interface CaseSummary {
  caseId: string
  transactionId: string
  rootCause: string
  confidence: number
  tierChosen: string
  /** "processing" | "recovered" | "held" | "open" | "escalated" | "error" */
  status: string
  /** Recovered amount if status is "recovered" — 0 otherwise (see transactionAmount for the original amount regardless of outcome). */
  amount: number
  /** The original transaction amount, regardless of outcome — for "amount at risk" style aggregates that `amount` alone can't express. */
  transactionAmount: number
}

/**
 * One entry from a case's audit_log, in the exact order the agent ran.
 * This is a FLAT, chronological list — not one-entry-per-agent. A case that
 * went through the MRD tier-3 "nudge ignored once -> incentivized_nudge"
 * upgrade has a SECOND full pass appended after the first: another
 * strategy/guardrail/execution/promise step later in this same array (two
 * entries with agentName="strategy" at different indices). Render as a
 * simple ordered timeline — never assume a fixed length or one entry per
 * agent name.
 */
export interface ReasoningStep {
  agentName: 'diagnosis' | 'strategy' | 'guardrail' | 'execution' | 'promise'
  decision: string
  /** Only present for diagnosis (a real 0-1 confidence score); absent (not 0) for every other agent. */
  confidence?: number
  /**
   * Meaning varies by agent:
   * - strategy: a genuine one-line "why not" per tier NOT chosen.
   * - diagnosis/guardrail/execution/promise: the evidence actually used,
   *   including checks that passed — not a rejected-candidates list.
   */
  alternatives: string[]
  /** Only present on an execution step whose decision is "recovered". */
  amount?: number
  timestamp: string
}

/** Response to GET /api/cases/:id and GET /api/guardrail/spotlight (same shape, the latter pre-filtered server-side). */
export interface CaseDetail {
  caseId: string
  transactionId: string
  rootCause: string
  confidence: number
  tierChosen: string
  status: string
  amount: number
  reasoningChain: ReasoningStep[]
}

/** One row of GET /api/cases/:id/audit — the raw, un-opinionated audit log. */
export interface AuditEntry {
  id: string
  caseId: string
  agentName: string
  decision: string
  confidence?: number
  alternatives: string[]
  amount?: number
  timestamp: string
}

export interface ApiErrorBody {
  error: string
}

/**
 * Request body for POST /api/simulate — a hypothetical case's inputs, not a
 * real transaction. failureCode is required unless disputedFlag is true
 * (diagnosis.Classify checks disputed status first, unconditionally, before
 * ever looking at the failure code).
 */
export interface SimulateRequest {
  failureCode: string
  historyScore: number
  disputedFlag: boolean
  doNotContact: boolean
  simulatedHour: number
  priorContactAttempts: number
}

/**
 * Response to POST /api/simulate: the genuine output of
 * diagnosis.Classify -> strategy.SelectTier -> guardrail.Check run against
 * the hypothetical inputs — nothing persisted. tierChosen is Strategy's own
 * pick before guardrail; guardrailFinalTier may differ if guardrail blocked
 * or held it.
 */
export interface SimulateResponse {
  rootCause: string
  confidence: number
  diagnosisEvidence: string[]
  tierChosen: string
  strategyAlternatives: string[]
  guardrailAllowed: boolean
  guardrailHeld: boolean
  guardrailFinalTier: string
  guardrailRuleFired: string
  guardrailReason: string
  guardrailEvidence: string[]
}
