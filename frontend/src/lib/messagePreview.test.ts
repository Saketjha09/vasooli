import { describe, expect, it } from 'vitest'
import { messagePreviewFor } from './messagePreview'

describe('messagePreviewFor', () => {
  it('escalate always has an explicit non-blank message, not a blank state', () => {
    const result = messagePreviewFor('disputed', 'escalate', 5000)
    expect(result.channel).toBe('none')
    expect(result.body.length).toBeGreaterThan(0)
    expect(result.body).toMatch(/escalated to a human agent/)
  })

  it('silent_retry has an explicit non-blank message, not a blank state', () => {
    const result = messagePreviewFor('transient_gateway', 'silent_retry', 5000)
    expect(result.channel).toBe('none')
    expect(result.body).toMatch(/retried automatically/)
  })

  it('nudge/card_expired uses email and mentions the real amount', () => {
    const result = messagePreviewFor('card_expired', 'nudge', 1000)
    expect(result.channel).toBe('email')
    expect(result.body).toContain('₹1,000.00')
    expect(result.subject).toMatch(/card/i)
  })

  it('nudge/checkout_friction uses sms', () => {
    const result = messagePreviewFor('checkout_friction', 'nudge', 500)
    expect(result.channel).toBe('sms')
    expect(result.subject).toBeUndefined()
  })

  it('incentivized_nudge is always email regardless of root cause, and computes the discounted amount correctly', () => {
    const upgraded = messagePreviewFor('card_expired', 'incentivized_nudge', 1000)
    const direct = messagePreviewFor('willful_nonpayment', 'incentivized_nudge', 1000)
    expect(upgraded.channel).toBe('email')
    expect(direct.channel).toBe('email')
    // 8% off 1000 = 920
    expect(upgraded.body).toContain('₹920.00')
    expect(direct.body).toContain('₹920.00')
  })
})
