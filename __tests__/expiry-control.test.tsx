import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom'
import { ExpiryControlPanel, formatExpiryDate, daysLeftLabel } from '@/components/wms/ExpiryControlPanel'
import type { ExpiryReport } from '@/lib/api'
import { api } from '@/lib/api'

const report: ExpiryReport = {
  days: 30,
  as_of: '2026-11-01',
  summary: {
    expired_batches: 1,
    expired_qty: '12',
    critical_batches: 1,
    critical_qty: '3',
    warning_batches: 1,
    warning_qty: '5',
  },
  lines: [
    {
      batch_id: 'b1', batch_number: 'LOT-OLD', expiry_date: '2026-10-25T00:00:00Z', days_left: -7,
      severity: 'EXPIRED', batch_status: 'RELEASED', product_id: 'p1', product_name: 'Susu UHT 1L',
      product_sku: 'SUSU-1', warehouse_id: 'w1', warehouse_name: 'Gudang Utama', location_id: 'l1',
      location_code: 'RAK-A1', location_type: 'INTERNAL', quantity: '12',
    },
    {
      batch_id: 'b2', batch_number: 'LOT-SOON', expiry_date: '2026-11-04T00:00:00Z', days_left: 3,
      severity: 'CRITICAL', batch_status: 'RELEASED', product_id: 'p1', product_name: 'Susu UHT 1L',
      product_sku: 'SUSU-1', warehouse_id: 'w1', warehouse_name: 'Gudang Utama', location_id: 'l1',
      location_code: 'RAK-A1', location_type: 'INTERNAL', quantity: '3',
    },
    {
      batch_id: 'b3', batch_number: 'LOT-NEXT', expiry_date: '2026-11-20T00:00:00Z', days_left: 19,
      severity: 'WARNING', batch_status: 'RELEASED', product_id: 'p2', product_name: 'Roti Tawar',
      product_sku: 'ROTI-1', warehouse_id: 'w1', warehouse_name: 'Gudang Utama', location_id: 'l2',
      location_code: 'RAK-B2', location_type: 'INTERNAL', quantity: '5',
    },
  ],
}

const calls: Array<{ days: number; warehouseId?: string }> = []
let mockResult: { data: ExpiryReport | null; isLoading: boolean; error: unknown } = {
  data: report,
  isLoading: false,
  error: null,
}

jest.mock('@/hooks/useExpiryReport', () => ({
  useExpiryReport: (days: number, warehouseId?: string) => {
    calls.push({ days, warehouseId })
    return mockResult
  },
}))

beforeEach(() => {
  calls.length = 0
  mockResult = { data: report, isLoading: false, error: null }
})

describe('ExpiryControlPanel', () => {
  it('shows per-batch summary, rows and severity labels', () => {
    render(<ExpiryControlPanel />)
    expect(screen.getByText('Kontrol Expired per Batch')).toBeInTheDocument()
    expect(screen.getByText('Sudah Expired')).toBeInTheDocument()
    expect(screen.getByText('12 unit')).toBeInTheDocument()
    expect(screen.getByText('LOT-OLD')).toBeInTheDocument()
    expect(screen.getByText('LOT-SOON')).toBeInTheDocument()
    expect(screen.getByText('LOT-NEXT')).toBeInTheDocument()
    expect(screen.getByText('Expired')).toBeInTheDocument()
    expect(screen.getByText('Kritis')).toBeInTheDocument()
    expect(screen.getByText('Segera')).toBeInTheDocument()
    expect(screen.getByText('Lewat 7 hari')).toBeInTheDocument()
    expect(screen.getByText('3 hari lagi')).toBeInTheDocument()
    expect(screen.getAllByText('RAK-A1')).toHaveLength(2)
    expect(screen.getByRole('link', { name: /menu Scrap/i })).toHaveAttribute('href', '/wms/scrap')
  })

  it('requests a new window when 60/90 hari is selected', () => {
    render(<ExpiryControlPanel warehouseId="wh-9" />)
    expect(calls[0]).toEqual({ days: 30, warehouseId: 'wh-9' })
    fireEvent.click(screen.getByRole('button', { name: '90 hari' }))
    expect(calls[calls.length - 1]).toEqual({ days: 90, warehouseId: 'wh-9' })
    expect(screen.getByRole('button', { name: '90 hari' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('shows an empty state and hides the scrap link when nothing expires', () => {
    mockResult = {
      data: { ...report, summary: { ...report.summary, expired_batches: 0, critical_batches: 0, warning_batches: 0 }, lines: [] },
      isLoading: false,
      error: null,
    }
    render(<ExpiryControlPanel />)
    expect(screen.getByText(/Tidak ada batch yang expired/)).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /menu Scrap/i })).not.toBeInTheDocument()
  })

  it('shows the backend error message', () => {
    mockResult = { data: null, isLoading: false, error: new Error('server down') }
    render(<ExpiryControlPanel />)
    expect(screen.getByRole('alert')).toHaveTextContent('server down')
  })
})

describe('expiry helpers', () => {
  it('formats a calendar date without timezone shift', () => {
    expect(formatExpiryDate('2026-10-15T00:00:00Z')).toMatch(/15/)
    expect(formatExpiryDate('2026-10-15T00:00:00Z')).toMatch(/2026/)
  })
  it('labels days left', () => {
    expect(daysLeftLabel(-2)).toBe('Lewat 2 hari')
    expect(daysLeftLabel(0)).toBe('Expired hari ini')
    expect(daysLeftLabel(5)).toBe('5 hari lagi')
  })
})

describe('api.wms.expiry.report', () => {
  it('calls GET /wms/expiry-report with days and warehouse_id', async () => {
    const fetchMock = jest.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: { get: () => 'application/json' },
      json: async () => ({ data: report }),
    })
    const original = global.fetch
    global.fetch = fetchMock as unknown as typeof fetch
    try {
      const res = await api.wms.expiry.report(60, 'wh-1')
      expect(res.data.lines).toHaveLength(3)
      const url = String(fetchMock.mock.calls[0][0])
      expect(url).toContain('/wms/expiry-report?')
      expect(url).toContain('days=60')
      expect(url).toContain('warehouse_id=wh-1')
    } finally {
      global.fetch = original
    }
  })
})
