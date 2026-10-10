import React from 'react'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import '@testing-library/jest-dom'
import { renderHook } from '@testing-library/react'
import BarcodeScannerPage from '@/app/(app)/wms/scanner/page'
import { useManualRefresh, MIN_REFRESH_FEEDBACK_MS, REFRESH_DONE_VISIBLE_MS } from '@/hooks/useManualRefresh'
import { RefreshButton } from '@/components/ui/RefreshButton'

const mockConfirm = jest.fn()
const mockResolve = jest.fn()
const mockPackScan = jest.fn()
let mockResolvedData: unknown = null
let mockResolveError = false

jest.mock('@/hooks/useBarcodeScanner', () => ({
  useBarcodeScanner: (options: { onScan: (c: string) => void }) => ({
    triggerScan: (code: string) => options.onScan(code),
    playTone: jest.fn(),
    triggerHaptic: jest.fn(),
  }),
  playScannerTone: jest.fn(),
  triggerHaptic: jest.fn(),
}))

jest.mock('@/hooks/useWMS', () => ({
  useResolveBarcode: () => ({ data: mockResolvedData, isFetching: false, isError: mockResolveError }),
  useWarehouseLocations: () => ({
    data: [{ id: 'loc-rack-1', code: 'RAK-A-01', name: 'Rak A-01', warehouse_id: 'wh-1' }],
  }),
  useConfirmPutaway: () => ({ mutateAsync: mockConfirm }),
  usePutawayPending: (wh: string | null) => ({
    data: wh
      ? [
          {
            product_id: 'prod-1',
            batch_id: 'batch-1',
            batch_number: 'B-001',
            quantity: '12',
            staging_location_id: 'stg-1',
            batch_status: 'ACTIVE',
            suggested_location_id: 'loc-rack-1',
            suggested_location_code: 'RAK-A-01',
          },
        ]
      : [],
  }),
  useDeliveryOrders: () => ({
    data: [{ id: 'do-1', do_number: 'DO-001', status: 'CONFIRMED', customer_name: 'Toko A' }],
  }),
}))

jest.mock('@/hooks/useWMSManifests', () => ({
  useShippingManifests: () => ({ data: [] }),
  useShippingManifestDetail: () => ({ data: null }),
  useScanLoadingDO: () => ({ mutate: jest.fn() }),
}))
jest.mock('@/hooks/useWMSDocksAndLPNs', () => ({
  useStockLPNs: () => ({ data: [] }),
  useStockLPNDetail: () => ({ data: null, isLoading: false }),
  useMoveLPN: () => ({ mutate: jest.fn() }),
}))
jest.mock('@/lib/api', () => ({
  api: {
    wms: {
      barcodes: { resolve: (...a: unknown[]) => mockResolve(...a) },
      deliveryOrders: { scanPackItem: (...a: unknown[]) => mockPackScan(...a) },
    },
  },
}))

function scan(code: string) {
  const input = screen.getByPlaceholderText(/Ketik barcode/i)
  fireEvent.change(input, { target: { value: code } })
  fireEvent.submit(input.closest('form')!)
}

beforeEach(() => {
  jest.clearAllMocks()
  mockResolvedData = null
  mockResolveError = false
})

describe('Scanner PUTAWAY mode uses real backend', () => {
  test('rejects an unregistered rack without calling API', () => {
    render(<BarcodeScannerPage />)
    scan('RAK-FAKE-99')
    expect(screen.getByText(/tidak terdaftar di master lokasi/i)).toBeInTheDocument()
    expect(mockConfirm).not.toHaveBeenCalled()
  })

  test('rack then product posts putaway confirm with staging line', async () => {
    mockResolve.mockResolvedValue({ product_id: 'prod-1', name: 'Beras', sku: 'S1' })
    mockConfirm.mockResolvedValue({})
    render(<BarcodeScannerPage />)
    scan('RAK-A-01')
    scan('8991234')
    await waitFor(() =>
      expect(mockConfirm).toHaveBeenCalledWith({
        warehouse_id: 'wh-1',
        product_id: 'prod-1',
        batch_id: 'batch-1',
        quantity: 12,
        dest_location_id: 'loc-rack-1',
      })
    )
    expect(await screen.findByText(/BERHASIL: 12 Beras/)).toBeInTheDocument()
  })

  test('product with no staging stock shows error and does not post', async () => {
    mockResolve.mockResolvedValue({ product_id: 'prod-OTHER', name: 'Gula', sku: 'S2' })
    render(<BarcodeScannerPage />)
    scan('RAK-A-01')
    scan('GULA')
    expect(await screen.findByText(/tidak memiliki stok di area staging/i)).toBeInTheDocument()
    expect(mockConfirm).not.toHaveBeenCalled()
  })

  test('server failure is surfaced, not reported as success', async () => {
    mockResolve.mockResolvedValue({ product_id: 'prod-1', name: 'Beras', sku: 'S1' })
    mockConfirm.mockRejectedValue(new Error('insufficient staging stock'))
    render(<BarcodeScannerPage />)
    scan('RAK-A-01')
    scan('BERAS')
    expect(await screen.findByText('insufficient staging stock')).toBeInTheDocument()
    expect(screen.queryByText(/BERHASIL/)).not.toBeInTheDocument()
  })
})

describe('Scanner OUTBOUND mode validates against Surat Jalan on server', () => {
  test('requires DO selection', () => {
    render(<BarcodeScannerPage />)
    fireEvent.click(screen.getByText('Outbound Picking'))
    scan('ANY')
    expect(screen.getByText(/Pilih Surat Jalan terlebih dahulu/)).toBeInTheDocument()
    expect(mockPackScan).not.toHaveBeenCalled()
  })

  test('match and mismatch come from pack/scan API', async () => {
    render(<BarcodeScannerPage />)
    fireEvent.click(screen.getByText('Outbound Picking'))
    fireEvent.change(screen.getByLabelText(/Pilih Surat Jalan/i), { target: { value: 'do-1' } })

    mockPackScan.mockResolvedValueOnce({
      data: { product_name: 'Beras', product_sku: 'S1', packed_qty: 1, requested_qty: 5, packed_items: 0, total_items: 2, order_completed: false },
    })
    scan('S1')
    expect(await screen.findByText('COCOK')).toBeInTheDocument()
    expect(mockPackScan).toHaveBeenCalledWith('do-1', 'S1', 1)

    mockPackScan.mockRejectedValueOnce(new Error('barcode does not match delivery order'))
    scan('ROJO-FAKE')
    expect(await screen.findByText('TIDAK COCOK')).toBeInTheDocument()
  })
})

describe('Scanner no longer fabricates products', () => {
  test('unknown barcode shows not-registered, no demo product', () => {
    mockResolveError = true
    render(<BarcodeScannerPage />)
    fireEvent.click(screen.getByText('Cek SKU'))
    scan('SKU-ROJO-10K')
    expect(screen.getByText(/tidak terdaftar di master produk/i)).toBeInTheDocument()
    expect(screen.queryByText('Beras Premium Rojolele 10kg')).not.toBeInTheDocument()
  })
})

describe('useManualRefresh', () => {
  beforeEach(() => jest.useFakeTimers())
  afterEach(() => jest.useRealTimers())

  test('spinner stays visible at least MIN_REFRESH_FEEDBACK_MS even when refetch is instant', async () => {
    const fn = jest.fn().mockResolvedValue({ isError: false })
    const { result } = renderHook(() => useManualRefresh([fn]))
    let p: Promise<void> = Promise.resolve()
    act(() => {
      p = result.current.refresh()
      void result.current.refresh() // double click ignored
    })
    expect(fn).toHaveBeenCalledTimes(1)
    expect(result.current.status).toBe('refreshing')
    await act(async () => {
      jest.advanceTimersByTime(MIN_REFRESH_FEEDBACK_MS - 50)
    })
    expect(result.current.status).toBe('refreshing')
    await act(async () => {
      jest.advanceTimersByTime(60)
      await p
    })
    expect(result.current.status).toBe('done')
    expect(result.current.lastRefreshedAt).not.toBeNull()
    await act(async () => {
      jest.advanceTimersByTime(REFRESH_DONE_VISIBLE_MS)
    })
    expect(result.current.status).toBe('idle')
  })

  test('reports error when refetch result isError', async () => {
    const fn = jest.fn().mockResolvedValue({ isError: true, error: new Error('Network down') })
    const { result } = renderHook(() => useManualRefresh([fn]))
    let p: Promise<void> = Promise.resolve()
    act(() => {
      p = result.current.refresh()
    })
    await act(async () => {
      jest.advanceTimersByTime(MIN_REFRESH_FEEDBACK_MS)
      await p
    })
    expect(result.current.status).toBe('error')
    expect(result.current.refreshError).toBe('Network down')
  })

  test('RefreshButton renders spinning icon, then success label', () => {
    const { rerender } = render(<RefreshButton status="refreshing" onClick={() => {}} showLabel />)
    const btn = screen.getByRole('button')
    expect(btn).toBeDisabled()
    expect(btn.querySelector('svg')).toHaveClass('animate-spin')
    expect(btn).toHaveTextContent('Menyegarkan...')
    rerender(<RefreshButton status="done" onClick={() => {}} showLabel />)
    expect(screen.getByRole('button')).toHaveTextContent('Diperbarui')
    expect(screen.getByRole('button')).not.toBeDisabled()
  })
})