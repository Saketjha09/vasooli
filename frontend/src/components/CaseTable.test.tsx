import { act, cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it } from 'vitest'
import type { CaseSummary } from '../api/types'
import { CaseTable } from './CaseTable'

// vitest doesn't auto-run RTL's cleanup between tests the way Jest's
// testEnvironment integration does — without this, each test's render()
// piles onto the previous test's still-mounted DOM in jsdom's shared
// document.body, which is why "Tier" briefly appeared to match multiple
// elements (leftover <p>Tier</p> from an earlier test, not an app bug).
afterEach(cleanup)

const TIERS = ['silent_retry', 'nudge', 'incentivized_nudge', 'escalate']
const STATUSES = ['processing', 'recovered', 'held', 'open', 'escalated', 'error']
const ROOT_CAUSES = [
  'transient_gateway',
  'card_expired',
  'insufficient_funds',
  'checkout_friction',
  'willful_nonpayment',
  'disputed',
]

function makeCase(i: number, tierChosen: string, status: string, rootCause: string): CaseSummary {
  return {
    caseId: `case-${i}`,
    transactionId: `10000000-0000-4000-8000-${String(i).padStart(12, '0')}`,
    rootCause,
    confidence: 0.9,
    tierChosen,
    status,
    amount: status === 'recovered' ? 100 : 0,
    transactionAmount: 100,
  }
}

// A small dataset guaranteed to include every declared tier and every
// declared status at least once, so "select every option" is a meaningful
// check (if any option's clicks got dropped, the count would drop below the
// full set).
const mockCases: CaseSummary[] = Array.from({ length: 12 }, (_, i) =>
  makeCase(i, TIERS[i % TIERS.length], STATUSES[i % STATUSES.length], ROOT_CAUSES[i % ROOT_CAUSES.length]),
)

function getPillButtons(groupLabel: string): HTMLButtonElement[] {
  // "Tier"/"Status" also appear as table column headers, so scope to the <p>
  // filter-group label specifically, not any match.
  const heading = screen.getByText(groupLabel, { selector: 'p' })
  const group = heading.parentElement
  if (!group) throw new Error(`could not find filter group container for "${groupLabel}"`)
  return Array.from(group.querySelectorAll('button'))
}

function showingCountText(): string {
  return screen.getByText(/Showing \d+ of \d+ cases/).textContent ?? ''
}

describe('CaseTable filter pills', () => {
  it('shows all cases with deliberate, individually-flushed clicks on every Tier and Status option', () => {
    render(
      <MemoryRouter>
        <CaseTable cases={mockCases} />
      </MemoryRouter>,
    )

    // Deliberate clicks: each wrapped in its own act() (what fireEvent does
    // internally too), so React flushes a re-render between every single
    // click — this is the "worked fine in my manual testing" case.
    for (const btn of getPillButtons('Tier')) {
      act(() => btn.click())
    }
    for (const btn of getPillButtons('Status')) {
      act(() => btn.click())
    }

    expect(showingCountText()).toBe(`Showing ${mockCases.length} of ${mockCases.length} cases`)
  })

  it('REPRODUCES the original bug scenario: rapid/batched clicks on every Tier and Status option still show the full count', () => {
    render(
      <MemoryRouter>
        <CaseTable cases={mockCases} />
      </MemoryRouter>,
    )

    const tierButtons = getPillButtons('Tier')
    const statusButtons = getPillButtons('Status')

    // The actual repro: fire every click in this category inside a SINGLE
    // act() call, with no flush between them. Before the fix (toggleSet
    // computing `next` from a closed-over `set` variable instead of React's
    // functional updater), each of these calls would compute `next` from
    // the SAME stale base Set, since no re-render happens between them —
    // silently dropping all but the last click's effect. React's automatic
    // batching genuinely does this when multiple state updates occur within
    // one synchronous block, which is exactly what "fast enough clicks"
    // amounts to.
    act(() => {
      for (const btn of tierButtons) btn.click()
    })
    act(() => {
      for (const btn of statusButtons) btn.click()
    })

    expect(showingCountText()).toBe(`Showing ${mockCases.length} of ${mockCases.length} cases`)
  })

  it('treats a strict subset of selected options as a real filter (sanity check the fix did not disable filtering)', () => {
    render(
      <MemoryRouter>
        <CaseTable cases={mockCases} />
      </MemoryRouter>,
    )

    const tierButtons = getPillButtons('Tier')
    // Click only the first tier pill (silent_retry) — should filter down to
    // however many mock cases have that tier (12 cases cycling through 4
    // tiers = 3 with silent_retry).
    act(() => tierButtons[0].click())

    const expectedCount = mockCases.filter((c) => c.tierChosen === TIERS[0]).length
    expect(showingCountText()).toBe(`Showing ${expectedCount} of ${mockCases.length} cases`)
  })

  it('Clear filters resets to the full unfiltered set', () => {
    render(
      <MemoryRouter>
        <CaseTable cases={mockCases} />
      </MemoryRouter>,
    )

    const tierButtons = getPillButtons('Tier')
    act(() => tierButtons[0].click())
    expect(showingCountText()).not.toBe(`Showing ${mockCases.length} of ${mockCases.length} cases`)

    const clearButton = screen.getByText('Clear filters')
    act(() => clearButton.click())

    expect(showingCountText()).toBe(`Showing ${mockCases.length} of ${mockCases.length} cases`)
    expect(screen.queryByText('Clear filters')).toBeNull()
  })
})
