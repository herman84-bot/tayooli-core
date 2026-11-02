/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import StockOpnamePage from '@/app/(app)/wms/opname/page'

const mockOpnames = [
  {
    id: 'op-1',
    opname_number: 'OPN-202611-001',
    warehouse_id: 'wh-1',
    status: 'IN_PROGRESS',
    conducted_by: 'Staff 1',
    created_at: '2026-11-01T00:00:00Z',
    items: [],
  },
]

jest.mock('@/hooks/useWMS', () => ({
  useStockOpnames: () => ({ data: mockOpnames, isLoading: false, refetch: jest.fn() }),
  useStockOpname: () => ({ data: null, isLoading: false, refetch: jest.fn() }),
  useCreateStockOpname: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useAddOpnameItem: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useCompleteStockOpname: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useWarehouses: () => ({
    data: [{ id: 'wh-1', code: 'W1', name: 'Gudang Pusat' }],
    isLoading: false,
    refetch: jest.fn(),
  }),
  useWarehouseLocations: () => ({ data: [], isLoading: false, refetch: jest.fn() }),
}))

jest.mock('@/hooks/useAuth', () => ({
  useAuthStore: () => ({
    user: { role: 'admin' },
  }),
}))

jest.mock('@/hooks/useManualRefresh', () => ({
  useManualRefresh: () => ({
    refresh: jest.fn(),
    refreshStatus: 'idle',
    refreshError: null,
    dataUpdatedAt: Date.now(),
  }),
}))

jest.mock('@/hooks/useBarcodeScanner', () => ({
  useBarcodeScanner: () => ({ triggerScan: jest.fn() }),
}))

describe('StockOpnamePage UI and Typography Quality', () => {
  it('renders table headers in sentence case without uppercase tracking-wider', () => {
    render(<StockOpnamePage />)

    const auditHeader = screen.getByRole('columnheader', { name: /Nomor Audit/i })
    expect(auditHeader.closest('thead')?.className).not.toMatch(/uppercase/)
    expect(auditHeader.closest('thead')?.className).not.toMatch(/tracking-wider/)
  })
})
