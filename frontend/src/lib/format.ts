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
