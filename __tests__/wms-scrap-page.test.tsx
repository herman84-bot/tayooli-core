/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import StockScrapPage from '@/app/(app)/wms/scrap/page'

const mockScraps = [
  {
    id: 'sc-1',
    scrap_number: 'SCR-202611-001',
    warehouse_id: 'wh-1',
    product_id: 'p-1',
    product_name: 'Kopi Robusta',
    source_location_id: 'loc-12345678',
    quantity: 5,
    scrap_reason: 'Kemasan rusak',
    created_at: '2026-11-01T00:00:00Z',
  },
]

jest.mock('@/hooks/useWMS', () => ({
  useStockScraps: () => ({ data: mockScraps, isLoading: false, refetch: jest.fn() }),
  useCreateStockScrap: () => ({ mutateAsync: jest.fn(), isPending: false }),
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

describe('StockScrapPage UI and Typography Quality', () => {
  it('renders table headers in sentence case without uppercase tracking-wider', () => {
    render(<StockScrapPage />)

    const scrapHeader = screen.getByRole('columnheader', { name: /Nomor Barang Rusak/i })
    expect(scrapHeader.closest('thead')?.className).not.toMatch(/uppercase/)
    expect(scrapHeader.closest('thead')?.className).not.toMatch(/tracking-wider/)
  })
})
