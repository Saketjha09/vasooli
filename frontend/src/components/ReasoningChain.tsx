import type { ReasoningStep } from '../api/types'
import { agentLabel, formatConfidence, formatMoney, formatTimestamp } from '../lib/format'
import { AgentBadge } from './AgentBadge'

const AGENT_DOT_COLORS: Record<string, string> = {
  diagnosis: 'bg-sky-500',
  strategy: 'bg-violet-500',
  guardrail: 'bg-red-500',
  execution: 'bg-emerald-500',
  promise: 'bg-amber-500',
}

/**
 * The step-by-step reasoning chain — the project's core differentiator.
 * Renders `reasoningChain` as a flat, ordered vertical timeline. A case
 * that went through the tier-3 "nudge ignored" upgrade simply has a second
 * strategy/guardrail/execution/promise group appended later in the same
 * list — no special grouping is applied, since that's the honest shape of
 * what happened (see api/types.ts's ReasoningStep doc comment).
 */
export function ReasoningChain({ steps }: { steps: ReasoningStep[] }) {
  if (steps.length === 0) {
    return <p className="text-sm text-slate-500">No reasoning steps recorded for this case yet.</p>
  }

  return (
    <ol className="relative space-y-6 border-l-2 border-slate-200 pl-6">
      {steps.map((step, i) => (
        <li key={i} className="relative">
          <span
            className={`absolute -left-[31px] top-1 h-3.5 w-3.5 rounded-full ring-4 ring-white ${
              AGENT_DOT_COLORS[step.agentName] ?? 'bg-slate-400'
            }`}
            aria-hidden
          />
          <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <AgentBadge agentName={step.agentName} />
                <span className="text-base font-semibold text-slate-900">{step.decision}</span>
              </div>
              <div className="flex items-center gap-2 text-xs text-slate-500">
                {step.confidence !== undefined && (
                  <span className="font-medium text-slate-700">
                    {formatConfidence(step.confidence)} confidence
                  </span>
                )}
                {step.amount !== undefined && (
                  <span className="font-medium text-emerald-700">{formatMoney(step.amount)}</span>
                )}
                <span>{formatTimestamp(step.timestamp)}</span>
              </div>
            </div>
            {step.alternatives.length > 0 && (
              <details className="mt-2 text-sm text-slate-600">
                <summary className="cursor-pointer font-medium text-slate-500 select-none">
                  Why? ({agentLabel(step.agentName)} reasoning)
                </summary>
                <ul className="mt-2 list-inside list-disc space-y-1">
                  {step.alternatives.map((alt, j) => (
                    <li key={j}>{alt}</li>
                  ))}
                </ul>
              </details>
            )}
          </div>
        </li>
      ))}
    </ol>
  )
}
