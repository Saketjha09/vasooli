import { agentLabel } from '../lib/format'

const AGENT_COLORS: Record<string, string> = {
  diagnosis: 'bg-sky-100 text-sky-800',
  strategy: 'bg-violet-100 text-violet-800',
  guardrail: 'bg-red-100 text-red-800',
  execution: 'bg-emerald-100 text-emerald-800',
  promise: 'bg-amber-100 text-amber-800',
}

/** Color-coded pill identifying which agent produced a reasoning step. */
export function AgentBadge({ agentName }: { agentName: string }) {
  const colorClass = AGENT_COLORS[agentName] ?? 'bg-slate-100 text-slate-800'
  return (
    <span className={`inline-block rounded-full px-2.5 py-0.5 text-xs font-semibold ${colorClass}`}>
      {agentLabel(agentName)}
    </span>
  )
}
