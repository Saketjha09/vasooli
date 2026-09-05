/** ₹ with 2 decimals, e.g. "₹8,932.00" — the money-recovered formatting convention used across the dashboard. */
export function formatMoney(amount: number): string {
  return `₹${amount.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`
}

export function formatConfidence(confidence: number): string {
  return `${Math.round(confidence * 100)}%`
}

export function formatTimestamp(iso: string): string {
  return new Date(iso).toLocaleString('en-IN', {
    dateStyle: 'medium',
    timeStyle: 'short',
  })
}

const TIER_LABELS: Record<string, string> = {
  silent_retry: 'Silent Retry',
  nudge: 'Nudge',
  incentivized_nudge: 'Incentivized Nudge',
  escalate: 'Escalate',
}

export function tierLabel(tier: string): string {
  return TIER_LABELS[tier] ?? tier
}

const AGENT_LABELS: Record<string, string> = {
  diagnosis: 'Diagnosis',
  strategy: 'Strategy',
  guardrail: 'Policy Guardrail',
  execution: 'Execution',
  promise: 'Promise Tracker',
}

export function agentLabel(agentName: string): string {
  return AGENT_LABELS[agentName] ?? agentName
}

const STATUS_LABELS: Record<string, string> = {
  processing: 'Processing',
  recovered: 'Recovered',
  held: 'Held',
  open: 'Open',
  escalated: 'Escalated',
  error: 'Error',
}

export function statusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status
}

const ROOT_CAUSE_LABELS: Record<string, string> = {
  transient_gateway: 'Transient Gateway',
  card_expired: 'Card Expired',
  insufficient_funds: 'Insufficient Funds',
  checkout_friction: 'Checkout Friction',
  willful_nonpayment: 'Willful Nonpayment',
  disputed: 'Disputed',
}

export function rootCauseLabel(rootCause: string): string {
  return ROOT_CAUSE_LABELS[rootCause] ?? rootCause
}

export const ROOT_CAUSE_ORDER = [
  'transient_gateway',
  'card_expired',
  'insufficient_funds',
  'checkout_friction',
  'willful_nonpayment',
  'disputed',
]

export const ROOT_CAUSE_COLORS: Record<string, string> = {
  transient_gateway: '#0ea5e9', // sky
  card_expired: '#8b5cf6', // violet
  insufficient_funds: '#f59e0b', // amber
  checkout_friction: '#ec4899', // pink
  willful_nonpayment: '#f97316', // orange
  disputed: '#ef4444', // red
}

/**
 * True only for a case whose final tier is incentivized_nudge AND whose
 * root cause is NOT willful_nonpayment. Derived, not stored: Strategy's
 * rule table (backend/internal/strategy/strategy.go) only ever assigns
 * incentivized_nudge directly for willful_nonpayment (historyScore >= 40);
 * every other root cause starts at "nudge" and can only reach
 * incentivized_nudge via the pipeline's tier-3 "nudge ignored once" upgrade
 * cycle. So this combination is a reliable, order-independent signal that a
 * case went through 2 pipeline cycles, without needing a dedicated backend
 * field.
 */
export function isTierUpgraded(caseSummary: { tierChosen: string; rootCause: string }): boolean {
  return caseSummary.tierChosen === 'incentivized_nudge' && caseSummary.rootCause !== 'willful_nonpayment'
}

/**
 * Templated next-step suggestion per root cause, for the Insights panel.
 * card_expired/insufficient_funds/checkout_friction are the only root
 * causes that ever actually reach status="open" in this pipeline (see
 * isTierUpgraded's doc comment) — and only via the same condition that
 * triggers the tier-3 upgrade: a nudge went unanswered (no_response), then
 * the resulting incentivized_nudge ALSO went unanswered. So every open
 * case in those 3 groups has already had 2 automated attempts, not 0 —
 * the suggestion text reflects that automation is already exhausted here,
 * not that nothing's been tried yet.
 */
const INSIGHT_SUGGESTIONS: Record<string, string> = {
  card_expired: '2 automated attempts unanswered — consider manual outreach or a direct support call',
  insufficient_funds: '2 automated attempts unanswered — consider a payment plan offer via manual contact',
  checkout_friction: '2 automated attempts unanswered — consider manual follow-up or a live-chat prompt',
  transient_gateway: 'consider checking gateway health for repeated transient failures',
  willful_nonpayment: 'consider a firmer collections outreach track',
  disputed: 'consider manual compliance review',
}

export function insightSuggestion(rootCause: string): string {
  return INSIGHT_SUGGESTIONS[rootCause] ?? 'consider a manual review of this root cause'
}
