import { formatMoney } from './format'

export type MessageChannel = 'sms' | 'email' | 'none'

export interface MessagePreview {
  channel: MessageChannel
  subject?: string
  body: string
}

// Channel is fully determined by (rootCause, tierChosen) per
// backend/internal/strategy/strategy.go's fixed rules — verified against
// that file, not guessed:
//   - silent_retry / escalate: no contact channel (Channel: ChannelNone)
//   - nudge: email for card_expired/insufficient_funds, sms for
//     checkout_friction (willful_nonpayment never reaches plain "nudge")
//   - incentivized_nudge: ALWAYS email, regardless of root cause — both the
//     direct willful_nonpayment path and the NudgeIgnoredUpgrade path set
//     Channel: ChannelEmail explicitly.
function channelFor(rootCause: string, tierChosen: string): MessageChannel {
  if (tierChosen === 'nudge') {
    return rootCause === 'checkout_friction' ? 'sms' : 'email'
  }
  if (tierChosen === 'incentivized_nudge') {
    return 'email'
  }
  return 'none'
}

// strategy.go's incentivizedNudgeDiscountPct is a fixed constant (8.0), not
// a per-case computed value — safe to hardcode here rather than derive it,
// same reasoning as FAILURE_CODE_OPTIONS mirroring the backend's fixed map.
const INCENTIVIZED_DISCOUNT_PCT = 8

/**
 * The actual customer-facing message this case's tier/channel would
 * produce. escalate/silent_retry deliberately return an explicit
 * "no message" state (with a reason), never a blank one.
 */
export function messagePreviewFor(rootCause: string, tierChosen: string, transactionAmount: number): MessagePreview {
  const channel = channelFor(rootCause, tierChosen)
  const amount = formatMoney(transactionAmount)

  if (tierChosen === 'silent_retry') {
    return {
      channel: 'none',
      body: 'No customer message sent — this was a transient gateway/bank issue, retried automatically without contacting the customer.',
    }
  }

  if (tierChosen === 'escalate') {
    return {
      channel: 'none',
      body: 'No automated customer-facing message — this case has been escalated to a human agent for manual review and outreach.',
    }
  }

  if (tierChosen === 'incentivized_nudge') {
    const discounted = formatMoney(transactionAmount * (1 - INCENTIVIZED_DISCOUNT_PCT / 100))
    return {
      channel,
      subject: `One-time ${INCENTIVIZED_DISCOUNT_PCT}% off to complete your payment`,
      body: `Hi, we still show an outstanding payment of ${amount}. Complete it now and save ${INCENTIVIZED_DISCOUNT_PCT}% (${discounted}): [payment link]. This offer won't be repeated.`,
    }
  }

  // tierChosen === 'nudge'
  if (rootCause === 'card_expired') {
    return {
      channel,
      subject: 'Action needed: update your card details',
      body: `Hi, we noticed your payment of ${amount} didn't go through because your card appears to have expired. Please update your card details to complete this payment: [update-card link]`,
    }
  }
  if (rootCause === 'insufficient_funds') {
    return {
      channel,
      subject: 'Your recent payment didn’t go through',
      body: `Hi, your payment of ${amount} couldn't be completed due to insufficient funds. We'll retry automatically, or you can complete it now here: [payment link]`,
    }
  }
  // checkout_friction
  return {
    channel,
    body: `Hi! Your ${amount} order is almost done — you didn't finish checkout. Complete it here: [link]`,
  }
}
