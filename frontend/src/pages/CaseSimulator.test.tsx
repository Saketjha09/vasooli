import { act, cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import { CaseSimulator } from './CaseSimulator'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('CaseSimulator', () => {
  it('surfaces the guardrail rule AND full reason in the always-visible chain text, not hidden behind "Why?"', async () => {
    vi.spyOn(client, 'getGuardrailPolicy').mockResolvedValue({
      maxContactAttempts: 3,
      maxDiscountPct: 10,
      contactWindowStart: 9,
      contactWindowEnd: 20,
    })
    vi.spyOn(client, 'simulateCase').mockResolvedValue({
      rootCause: 'card_expired',
      confidence: 0.95,
      diagnosisEvidence: ['failure_code=CARD_EXPIRED'],
      tierChosen: 'nudge',
      strategyAlternatives: ['silent_retry: not applicable', 'incentivized_nudge: not yet', 'escalate: not warranted'],
      guardrailAllowed: false,
      guardrailHeld: true,
      guardrailFinalTier: 'nudge',
      guardrailRuleFired: 'contact_hours',
      guardrailReason: 'simulatedHour=3, window=[9,20); outside contact hours, held for retry',
      guardrailEvidence: ['customer.disputed_flag=false', 'simulatedHour=3, window=[9,20)'],
    })

    render(
      <MemoryRouter>
        <CaseSimulator />
      </MemoryRouter>,
    )

    const runButton = screen.getByText('Run Simulation')
    await act(async () => {
      runButton.click()
    })

    // The exact scenario the reviewer asked to confirm: someone testing an
    // out-of-window hour must see the rule + reason immediately.
    const headline = await screen.findByText(/Held — contact_hours: simulatedHour=3/)
    expect(headline).toBeTruthy()

    // Prove it's NOT inside the collapsed <details> "Why?" section — a
    // judge shouldn't have to expand anything to see this.
    expect(headline.closest('details')).toBeNull()

    // The simulation banner is present and unambiguous.
    expect(screen.getByText('Simulated — not a real transaction')).toBeTruthy()
  })

  it('disables the Failure Code dropdown when Disputed is checked', () => {
    vi.spyOn(client, 'getGuardrailPolicy').mockResolvedValue({
      maxContactAttempts: 3,
      maxDiscountPct: 10,
      contactWindowStart: 9,
      contactWindowEnd: 20,
    })

    render(
      <MemoryRouter>
        <CaseSimulator />
      </MemoryRouter>,
    )

    const disputedCheckbox = screen.getByLabelText('Disputed') as HTMLInputElement
    const failureCodeSelect = screen.getByLabelText('Failure Code') as HTMLSelectElement

    expect(failureCodeSelect.disabled).toBe(false)
    act(() => disputedCheckbox.click())
    expect(failureCodeSelect.disabled).toBe(true)
  })
})
