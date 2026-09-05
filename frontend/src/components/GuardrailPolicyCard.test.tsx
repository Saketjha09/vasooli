import { cleanup, render, screen, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import { GuardrailPolicyCard } from './GuardrailPolicyCard'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('GuardrailPolicyCard', () => {
  it('renders real policy values, independent of any case/batch data existing', async () => {
    // The whole point of this endpoint: it's sourced from server config, not
    // case data, so it must render correctly even with zero cases anywhere
    // else on the page. This component doesn't take `cases` as a prop at
    // all — there is nothing for it to depend on.
    vi.spyOn(client, 'getGuardrailPolicy').mockResolvedValue({
      maxContactAttempts: 3,
      maxDiscountPct: 10,
      contactWindowStart: 9,
      contactWindowEnd: 20,
    })

    render(<GuardrailPolicyCard />)

    expect(screen.getByText(/Loading policy/)).toBeTruthy()

    await waitFor(() => expect(screen.getByText('3')).toBeTruthy())
    expect(screen.getByText('10%')).toBeTruthy()
    expect(screen.getByText('09:00–20:00')).toBeTruthy()
    expect(screen.queryByText(/temporarily unavailable/)).toBeNull()
  })

  it('shows a quiet unavailable message on fetch failure, not a broken/empty layout', async () => {
    vi.spyOn(client, 'getGuardrailPolicy').mockRejectedValue(new Error('network error'))

    render(<GuardrailPolicyCard />)

    await waitFor(() => expect(screen.getByText(/temporarily unavailable/)).toBeTruthy())
    // The framing sentence should still be present even on error — this is
    // a quiet degrade, not a collapsed/missing card.
    expect(screen.getByText(/govern every automated decision/)).toBeTruthy()
  })
})
