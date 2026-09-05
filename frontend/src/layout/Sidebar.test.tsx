import { cleanup, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import * as client from '../api/client'
import { DashboardDataProvider } from '../context/DashboardDataContext'
import { Sidebar } from './Sidebar'

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

function renderSidebar() {
  vi.spyOn(client, 'getBatchSummary').mockResolvedValue({ totalCases: 0, tierBreakdown: {}, moneyRecovered: 0 })
  vi.spyOn(client, 'listCases').mockResolvedValue([])
  return render(
    <MemoryRouter>
      <DashboardDataProvider>
        <Sidebar />
      </DashboardDataProvider>
    </MemoryRouter>,
  )
}

describe('Sidebar', () => {
  it('renders all 4 nav links and the Run Batch action', () => {
    renderSidebar()

    expect(screen.getByRole('link', { name: 'Overview' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Cases' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Guardrail Spotlight' })).toBeTruthy()
    expect(screen.getByRole('link', { name: 'Simulator' })).toBeTruthy()
    expect(screen.getByText('Run Batch')).toBeTruthy()
  })

  it('nav links point at the correct routes', () => {
    renderSidebar()

    expect(screen.getByRole('link', { name: 'Overview' }).getAttribute('href')).toBe('/')
    expect(screen.getByRole('link', { name: 'Cases' }).getAttribute('href')).toBe('/cases')
    expect(screen.getByRole('link', { name: 'Guardrail Spotlight' }).getAttribute('href')).toBe('/guardrail-spotlight')
    expect(screen.getByRole('link', { name: 'Simulator' }).getAttribute('href')).toBe('/simulate')
  })
})
