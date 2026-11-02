/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import WMSDashboardPage from '@/app/(app)/wms/page'

jest.mock('@/hooks/useWMS', () => ({
  useWarehouses: () => ({
    data: [{ id: 'wh-1', code: 'WH1', name: 'Gudang Utama' }],
    isLoading: false,
    refetch: jest.fn(),
  }),
  useWarehouseLocations: () => ({ data: [], isLoading: false, refetch: jest.fn() }),
  useStockTransfers: () => ({ data: [], isLoading: false, refetch: jest.fn() }),
  useCreateWarehouse: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useCreateLocation: () => ({ mutateAsync: jest.fn(), isPending: false }),
}))

jest.mock('@/hooks/useWMSLedger', () => ({
  useWMSMovements: () => ({ data: [], isLoading: false, refetch: jest.fn() }),
  useWMSStock: () => ({ data: [], isLoading: false, refetch: jest.fn() }),
}))

jest.mock('@/hooks/useManualRefresh', () => ({
  useManualRefresh: () => ({
    refresh: jest.fn(),
    refreshStatus: 'idle',
    refreshError: null,
    dataUpdatedAt: Date.now(),
  }),
}))

describe('WMSDashboardPage UI and Typography Quality', () => {
  it('renders quick metric cards and selector in human sentence case without screaming uppercase tracking', () => {
    render(<WMSDashboardPage />)

    const whSelectorLabel = screen.getByText('Pilih Gudang Aktif')
    expect(whSelectorLabel.className).not.toMatch(/uppercase/)
    expect(whSelectorLabel.className).not.toMatch(/tracking-wider/)

    const totalRakLabel = screen.getByText('Total Rak')
    expect(totalRakLabel.className).not.toMatch(/uppercase/)
    expect(totalRakLabel.className).not.toMatch(/tracking-wider/)

    const transferLabel = screen.getByText('Transfer Dalam Pengiriman')
    expect(transferLabel.className).not.toMatch(/uppercase/)
    expect(transferLabel.className).not.toMatch(/tracking-wider/)
  })
})
