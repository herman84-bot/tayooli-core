/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import TransfersPage from '@/app/(app)/wms/transfers/page'

const mockTransfers = [
  {
    id: 'tr-1',
    transfer_number: 'TR-2026-001',
    from_warehouse_id: 'wh-1',
    to_warehouse_id: 'wh-2',
    status: 'IN_TRANSIT',
    created_at: '2026-11-01T00:00:00Z',
    items: [],
  },
]

jest.mock('@/hooks/useWMS', () => ({
  useStockTransfers: () => ({ data: mockTransfers, isLoading: false, refetch: jest.fn() }),
  useWarehouses: () => ({
    data: [
      { id: 'wh-1', code: 'W1', name: 'Gudang Barat' },
      { id: 'wh-2', code: 'W2', name: 'Gudang Timur' },
    ],
    isLoading: false,
    refetch: jest.fn(),
  }),
  useWarehouseLocations: () => ({ data: [], isLoading: false, refetch: jest.fn() }),
  useCreateTransfer: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useSubmitTransfer: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useApproveTransfer: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useRejectTransfer: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useDispatchTransfer: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useReceiveTransfer: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useCancelTransfer: () => ({ mutateAsync: jest.fn(), isPending: false }),
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

describe('TransfersPage UI and Typography Quality', () => {
  it('renders step tracker labels and travel status in human sentence case without screaming uppercase tracking', () => {
    render(<TransfersPage />)

    const travelStatus = screen.getByText('Status Perjalanan')
    expect(travelStatus.className).not.toMatch(/uppercase/)
    expect(travelStatus.className).not.toMatch(/tracking-wider/)

    const dikirimStep = screen.getByText('Dikirim')
    expect(dikirimStep.className).not.toMatch(/uppercase/)
    expect(dikirimStep.className).not.toMatch(/tracking-wider/)
  })
})
