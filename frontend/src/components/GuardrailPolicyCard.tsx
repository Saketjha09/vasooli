import { useEffect, useState } from 'react'
import { getGuardrailPolicy } from '../api/client'
import type { GuardrailPolicy } from '../api/types'

function formatHour(hour: number): string {
  return `${String(hour).padStart(2, '0')}:00`
}

/**
 * Quiet, read-only context card: "here's what governs every automated
 * decision." Fetches independently of the rest of SummaryView's data (it
 * doesn't depend on any case/batch data existing — guardrail.Caps is fixed
 * server config, always available) so it never blocks on, or is blocked by,
 * the batch-run state. Degrades gracefully before any batch has run: the
 * endpoint has nothing to do with case data, so it succeeds identically
 * whether 0 or 40 cases exist — the only states this card actually has are
 * loading (briefly, on mount) and a genuine fetch failure (network/backend
 * down), never an "empty" state.
 */
export function GuardrailPolicyCard() {
  const [policy, setPolicy] = useState<GuardrailPolicy | 'loading' | 'error'>('loading')

  useEffect(() => {
    let ignore = false
    getGuardrailPolicy()
      .then((p) => {
        if (!ignore) setPolicy(p)
      })
      .catch(() => {
        if (!ignore) setPolicy('error')
      })
    return () => {
      ignore = true
    }
  }, [])

  return (
    <div className="rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
      <p className="text-sm text-slate-600">
        These fixed caps govern every automated decision below — no case can bypass them.
      </p>

      {policy === 'loading' && <p className="mt-2 text-xs text-slate-400">Loading policy…</p>}

      {policy === 'error' && (
        <p className="mt-2 text-xs text-slate-400">Policy details are temporarily unavailable.</p>
      )}

      {policy !== 'loading' && policy !== 'error' && (
        <dl className="mt-3 grid grid-cols-1 gap-3 text-sm sm:grid-cols-3">
          <div>
            <dt className="text-xs font-semibold text-slate-500 uppercase">Max Contact Attempts</dt>
            <dd className="mt-0.5 font-medium text-slate-800">{policy.maxContactAttempts}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-slate-500 uppercase">Max Discount</dt>
            <dd className="mt-0.5 font-medium text-slate-800">{policy.maxDiscountPct}%</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-slate-500 uppercase">Contact Hours</dt>
            <dd className="mt-0.5 font-medium text-slate-800">
              {formatHour(policy.contactWindowStart)}–{formatHour(policy.contactWindowEnd)}
            </dd>
          </div>
        </dl>
      )}
    </div>
  )
}
