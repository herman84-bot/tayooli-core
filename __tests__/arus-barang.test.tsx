import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import ArusBarangPage from '@/app/(app)/wms/arus-barang/page'

let currentMode = 'masuk'
const mockRouterReplace = jest.fn((url: string) => {
  const match = url.match(/mode=(masuk|keluar)/)
  if (match) {
    currentMode = match[1]
  }
})

const mockSearchParams = {
  get: (key: string) => (key === 'mode' ? currentMode : null),
}

jest.mock('next/navigation', () => ({
  useRouter: () => ({
    replace: mockRouterReplace,
    push: jest.fn(),
  }),
  useSearchParams: () => mockSearchParams,
}))

const WH = 'wh-1'
const receiptDraft = {
  id: 'r-1',
  tenant_id: 't',
  receipt_number: 'GR-20261007-001',
  warehouse_id: WH,
  dest_location_id: 'loc-stg',
  supplier_name: 'PT Mitra Sejati',
  status: 'POSTED',
  created_by: 'u',
  created_at: '2026-10-07T03:00:00Z',
  updated_at: '2026-10-07T03:00:00Z',
  item_count: 1,
  total_accepted_qty: '50',
  total_rejected_qty: '0',
  on_hold_batch_count: 1,
}

const mockPutawayPending = [
  {
    warehouse_id: WH,
    product_id: 'p-1',
    product_name: 'Kopi Robusta 500g',
    product_sku: 'KOP-ROB-500',
    batch_id: 'b-1',
    batch_number: 'BATCH-20261007-001',
    batch_status: 'ON_HOLD',
    staging_location_id: 'loc-stg',
    staging_location_code: 'STG-IN',
    quantity: 50,
    default_location_id: 'loc-rack-1',
    suggested_location_id: 'loc-rack-1',
    suggested_location_code: 'A-01-01',
    suggestion_source: 'DEFAULT_RACK',
  },
]

jest.mock('@/hooks/useWMS', () => ({
  useWarehouses: () => ({
    data: [{ id: 'wh-1', name: 'Gudang Cakung' }],
    isLoading: false,
  }),
  useWarehouseLocations: () => ({
    data: [
      { id: 'loc-stg', warehouse_id: 'wh-1', code: 'STG-IN', name: 'Staging Inbound', type: 'STAGING_INBOUND' },
      { id: 'loc-rack-1', warehouse_id: 'wh-1', code: 'A-01-01', name: 'Rak Kopi', type: 'INTERNAL' },
      { id: 'loc-rack-2', warehouse_id: 'wh-1', code: 'B-02-01', name: 'Rak Alternatif', type: 'INTERNAL' },
    ],
  }),
  useStockReceipts: () => ({
    data: [receiptDraft],
    isLoading: false,
    isError: false,
    refetch: jest.fn(),
  }),
  useStockReceipt: () => ({
    data: {
      receipt: receiptDraft,
      items: [
        {
          id: 'i-1',
          product_id: 'p-1',
          product_name: 'Kopi Robusta 500g',
          product_sku: 'KOP-ROB-500',
          batch_number: 'BATCH-20261007-001',
          accepted_qty: '50',
          rejected_qty: '0',
        },
      ],
    },
    isLoading: false,
  }),
  useCreateStockReceipt: () => ({ mutateAsync: jest.fn() }),
  useUpdateStockReceipt: () => ({ mutateAsync: jest.fn() }),
  usePostStockReceipt: () => ({ mutateAsync: jest.fn() }),
  useCancelStockReceipt: () => ({ mutateAsync: jest.fn() }),
  useReleaseStockReceipt: () => ({ mutateAsync: jest.fn() }),
  useDeliveryOrders: () => ({ data: [], isLoading: false, refetch: jest.fn() }),
  useCreateDeliveryOrder: () => ({ mutateAsync: jest.fn() }),
  useDispatchDeliveryOrder: () => ({ mutateAsync: jest.fn() }),
  usePutawayPending: () => ({
    data: mockPutawayPending,
    isLoading: false,
    refetch: jest.fn(),
  }),
  useConfirmPutaway: () => ({ mutate: jest.fn(), isPending: false }),
  useWMSSettings: () => ({
    data: { require_release_approval: true },
    refetch: jest.fn(),
  }),
  useUpdateWMSSettings: () => ({ mutate: jest.fn(), isPending: false }),
  useProductDefaultLocations: () => ({
    data: [
      {
        product_id: 'p-1',
        product_name: 'Kopi Robusta 500g',
        product_sku: 'KOP-ROB-500',
        warehouse_id: 'wh-1',
        warehouse_name: 'Gudang Cakung',
        location_id: 'loc-rack-1',
        location_code: 'A-01-01',
      },
    ],
    refetch: jest.fn(),
  }),
  useSetDefaultLocation: () => ({ mutate: jest.fn(), isPending: false }),
  useDeleteDefaultLocation: () => ({ mutate: jest.fn() }),
  useBatchTrace: () => ({ data: null, isLoading: false }),
  useAuditTrail: () => ({ data: [], isLoading: false }),
  useQCInspections: () => ({ data: [], isLoading: false }),
  useReceiptQC: () => ({ data: null, isLoading: false }),
  useSubmitQC: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useQuarantineStock: () => ({ data: [], isLoading: false }),
  useReleaseQuarantine: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useScrapQuarantine: () => ({ mutateAsync: jest.fn(), isPending: false }),
}))

jest.mock('@/hooks/useProducts', () => ({
  useProducts: () => ({
    data: [{ id: 'p-1', name: 'Kopi Robusta 500g', sku: 'KOP-ROB-500' }],
  }),
}))

describe('Unified Arus Barang Page (KO-2a & Sprint 1)', () => {
  beforeEach(() => {
    currentMode = 'masuk'
    if (typeof localStorage !== 'undefined') {
      localStorage.clear()
    }
    jest.clearAllMocks()
  })

  it('renders page header and MASUK / KELUAR toggle buttons', () => {
    render(<ArusBarangPage />)
    expect(screen.getByText(/Arus Barang \(Masuk & Keluar\)/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /BARANG MASUK/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /BARANG KELUAR/i })).toBeInTheDocument()
  })

  it('renders MASUK sub-tabs: Penerimaan, Putaway, Lacak Batch, QC', () => {
    render(<ArusBarangPage />)
    expect(screen.getByRole('button', { name: /Penerimaan \(GR\)/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Putaway ke Rak/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Lacak Batch \(Traceability\)/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /QC & Karantina/i })).toBeInTheDocument()
  })

  it('switches to KELUAR mode when toggle clicked', () => {
    const { rerender } = render(<ArusBarangPage />)
    const keluarBtn = screen.getByRole('button', { name: /BARANG KELUAR/i })
    fireEvent.click(keluarBtn)

    expect(mockRouterReplace).toHaveBeenCalledWith('/wms/arus-barang?mode=keluar')
    rerender(<ArusBarangPage />) // router.replace updates the search params
    expect(screen.getByRole('button', { name: /Surat Jalan \(DO\)/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Picking Wave/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Packing Station/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Manifest & Muat/i })).toBeInTheDocument()
  })

  // Regression (prod bug): ?mode=keluar must win over a stale localStorage "masuk".
  it('URL ?mode=keluar overrides stale localStorage mode', () => {
    localStorage.setItem('wms_arus_barang_mode', 'masuk')
    currentMode = 'keluar'
    render(<ArusBarangPage />)
    expect(screen.getByRole('button', { name: /Surat Jalan \(DO\)/i })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Penerimaan \(GR\)/i })).not.toBeInTheDocument()
    expect(mockRouterReplace).not.toHaveBeenCalled()
    expect(localStorage.getItem('wms_arus_barang_mode')).toBe('keluar')
  })

  it('URL without mode falls back to localStorage via router.replace', () => {
    localStorage.setItem('wms_arus_barang_mode', 'keluar')
    currentMode = ''
    render(<ArusBarangPage />)
    expect(mockRouterReplace).toHaveBeenCalledWith('/wms/arus-barang?mode=keluar')
  })

  // Regression (prod bug): "Buat Surat Jalan" was a silent no-op.
  it('Buat Surat Jalan button opens the create DO form', () => {
    currentMode = 'keluar'
    render(<ArusBarangPage />)
    fireEvent.click(screen.getByRole('button', { name: /Buat Surat Jalan/i }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('switches to Putaway tab and displays pending lines in staging', () => {
    render(<ArusBarangPage />)
    const putawayTab = screen.getByRole('button', { name: /Putaway ke Rak/i })
    fireEvent.click(putawayTab)

    expect(screen.getByText('Kopi Robusta 500g')).toBeInTheDocument()
    expect(screen.getByText('BATCH-20261007-001')).toBeInTheDocument()
    expect(screen.getByText('A-01-01')).toBeInTheDocument()
    expect(screen.getByText('Rak Default')).toBeInTheDocument()
  })
})
