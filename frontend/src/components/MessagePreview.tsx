import { messagePreviewFor } from '../lib/messagePreview'

const CHANNEL_LABELS: Record<string, string> = {
  sms: 'SMS',
  email: 'Email',
  none: 'No Contact',
}

/**
 * The actual customer-facing message this case's tier/channel produced —
 * concrete evidence a judge can read directly, complementing the abstract
 * reasoning chain above it. escalate/silent_retry show an explicit
 * "no message" state, never a blank one.
 */
export function MessagePreview({
  rootCause,
  tierChosen,
  transactionAmount,
}: {
  rootCause: string
  tierChosen: string
  transactionAmount: number
}) {
  const preview = messagePreviewFor(rootCause, tierChosen, transactionAmount)

  if (preview.channel === 'none') {
    return (
      <div className="rounded-card bg-page p-4 text-sm text-ink-secondary">
        <p className="text-xs font-semibold tracking-wide text-ink-muted uppercase">Customer Message</p>
        <p className="mt-1">{preview.body}</p>
      </div>
    )
  }

  return (
    <div className="rounded-card bg-white p-4 shadow-card">
      <div className="mb-2 flex items-center gap-2">
        <p className="text-xs font-semibold tracking-wide text-ink-muted uppercase">Customer Message</p>
        <span className="rounded-full bg-info-bg px-2 py-0.5 text-[10px] font-semibold text-info-text">
          {CHANNEL_LABELS[preview.channel]}
        </span>
      </div>
      {preview.subject && <p className="text-sm font-semibold text-ink">{preview.subject}</p>}
      <p className="mt-1 text-sm text-ink-secondary">{preview.body}</p>
    </div>
  )
}
