import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import BarcodeScannerPage from '@/app/(app)/wms/scanner/page'
import type { StockLPN, StockLPNDetail } from '@/lib/api'

// Mock useBarcodeScanner
const mockTriggerScan = jest.fn()
const mockPlayTone = jest.fn()
const mockTriggerHaptic = jest.fn()

let mockOnScanCallback: ((code: string) => void) | null = null

jest.mock('@/hooks/useBarcodeScanner', () => ({
  useBarcodeScanner: (options: any) => {
    mockOnScanCallback = options.onScan
    return {
      triggerScan: (code: string) => options.onScan(code),
      playTone: mockPlayTone,
      triggerHaptic: mockTriggerHaptic,
    }
  },
  playScannerTone: jest.fn(),
  triggerHaptic: jest.fn(),
}))

// Mock useWMS
jest.mock('@/hooks/useWMS', () => ({
  useConfirmPutaway: () => ({ mutateAsync: jest.fn() }),
  usePutawayPending: () => ({ data: [] }),
  useDeliveryOrders: () => ({ data: [] }),
  useResolveBarcode: () => ({
    data: null,
    isFetching: false,
    isError: false,
  }),
  useWarehouseLocations: () => ({
    data: [
      { id: 'loc-rack-1', code: 'RAK-A-01', name: 'Rak A-01', warehouse_id: 'wh-1' },
      { id: 'loc-rack-2', code: 'RAK-B-02', name: 'Rak B-02', warehouse_id: 'wh-1' },
    ],
  }),
}))

// Mock shipping manifests
jest.mock('@/hooks/useWMSManifests', () => ({
  useShippingManifests: () => ({ data: [], isLoading: false }),
  useShippingManifestDetail: () => ({ data: null, isLoading: false }),
  useScanLoadingDO: () => ({ mutate: jest.fn(), isPending: false }),
}))

// Mock LPN data
const mockStockLPNs: StockLPN[] = [
  {
    id: 'lpn-1',
    warehouse_id: 'wh-1',
    lpn_code: 'LPN-20261008-0001',
    location_id: 'loc-stage-1',
    location_code: 'STAGING-IN-01',
    pallet_type: 'WOODEN',
    status: 'STAGED',
    max_weight_kg: 1000,
    total_weight_kg: 350,
    created_at: '2026-10-08T08:00:00Z',
    updated_at: '2026-10-08T08:00:00Z',
  },
  {
    id: 'lpn-2',
    warehouse_id: 'wh-1',
    lpn_code: 'LPN-20261008-0002',
    location_id: 'loc-stage-1',
    location_code: 'STAGING-IN-01',
    pallet_type: 'PLASTIC',
    status: 'STAGED',
    max_weight_kg: 800,
    total_weight_kg: 200,
    created_at: '2026-10-08T08:30:00Z',
    updated_at: '2026-10-08T08:30:00Z',
  },
]

const mockLPNDetail: StockLPNDetail = {
  lpn: mockStockLPNs[0],
  items: [
    {
      id: 'item-1',
      lpn_id: 'lpn-1',
      product_id: 'prod-1',
      product_sku: 'SKU-ROJO-10K',
      product_name: 'Beras Premium Rojolele 10kg',
      batch_id: 'b-1',
      batch_number: 'BATCH-20261008-01',
      quantity: 20,
      created_at: '2026-10-08T08:00:00Z',
      updated_at: '2026-10-08T08:00:00Z',
    },
    {
      id: 'item-2',
      lpn_id: 'lpn-1',
      product_id: 'prod-2',
      product_sku: 'SKU-SANIA-2L',
      product_name: 'Minyak Goreng Sania 2L',
      batch_id: 'b-2',
      batch_number: 'BATCH-20261008-02',
      quantity: 15,
      created_at: '2026-10-08T08:00:00Z',
      updated_at: '2026-10-08T08:00:00Z',
    },
  ],
}

const mockMoveLPNMutate = jest.fn()

jest.mock('@/hooks/useWMSDocksAndLPNs', () => ({
  useStockLPNs: () => ({
    data: mockStockLPNs,
    isLoading: false,
  }),
  useStockLPNDetail: (id?: string | null) => ({
    data: id === 'lpn-1' ? mockLPNDetail : null,
    isLoading: false,
  }),
  useMoveLPN: () => ({
    mutate: mockMoveLPNMutate,
    isPending: false,
  }),
}))

describe('Task 7: Barcode Scanner Pallet LPN Putaway Mode', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders mode switch button for Palet (LPN Putaway)', () => {
    render(<BarcodeScannerPage />)

    const palletTabBtn = screen.getByRole('button', { name: /Palet \(LPN Putaway\)/i })
    expect(palletTabBtn).toBeInTheDocument()
  })

  it('switches to PALLET_LPN mode and renders step indicators', () => {
    render(<BarcodeScannerPage />)

    const palletTabBtn = screen.getByRole('button', { name: /Palet \(LPN Putaway\)/i })
    fireEvent.click(palletTabBtn)

    expect(screen.getByText(/Putaway Palet LPN:/i)).toBeInTheDocument()
    expect(screen.getByText(/Langkah 1: Scan Barcode Palet LPN/i)).toBeInTheDocument()
    expect(screen.getByText(/Langkah 2: Scan Barcode Rak Lokasi Tujuan/i)).toBeInTheDocument()
    expect(screen.getByText('Belum Terpilih')).toBeInTheDocument()
  })

  it('Step 1: displays error if scanned code is not a valid LPN barcode', () => {
    render(<BarcodeScannerPage />)

    const palletTabBtn = screen.getByRole('button', { name: /Palet \(LPN Putaway\)/i })
    fireEvent.click(palletTabBtn)

    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'UNKNOWN-BARCODE-999' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    expect(mockPlayTone).toHaveBeenCalledWith('error')
    expect(screen.getByText(/bukan palet LPN valid!/i)).toBeInTheDocument()
  })

  it('Step 1: scanning pallet LPN matches LPN, displays pallet badge and line items list', () => {
    render(<BarcodeScannerPage />)

    const palletTabBtn = screen.getByRole('button', { name: /Palet \(LPN Putaway\)/i })
    fireEvent.click(palletTabBtn)

    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'LPN-20261008-0001' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    expect(mockPlayTone).toHaveBeenCalledWith('success')
    expect(
      screen.getByText(/Palet \[LPN-20261008-0001\] terpilih! Silakan scan barcode Rak Tujuan./i)
    ).toBeInTheDocument()

    // Step 1 updated status
    expect(screen.getByText('Sudah Terpilih')).toBeInTheDocument()

    // Pallet Info Badge
    expect(screen.getByText('Informasi Palet Terpilih')).toBeInTheDocument()
    expect(screen.getByText('WOODEN')).toBeInTheDocument()
    expect(screen.getByText('STAGING-IN-01')).toBeInTheDocument()
    expect(screen.getByText('350 kg')).toBeInTheDocument()

    // Table of Pallet line items
    expect(screen.getByText('SKU-ROJO-10K')).toBeInTheDocument()
    expect(screen.getByText('Beras Premium Rojolele 10kg')).toBeInTheDocument()
    expect(screen.getByText('BATCH-20261008-01')).toBeInTheDocument()
    expect(screen.getByText('20')).toBeInTheDocument()

    expect(screen.getByText('SKU-SANIA-2L')).toBeInTheDocument()
    expect(screen.getByText('Minyak Goreng Sania 2L')).toBeInTheDocument()
    expect(screen.getByText('BATCH-20261008-02')).toBeInTheDocument()
    expect(screen.getByText('15')).toBeInTheDocument()
  })

  it('allows operator to click "Ganti Palet" to reset selected pallet', () => {
    render(<BarcodeScannerPage />)

    const palletTabBtn = screen.getByRole('button', { name: /Palet \(LPN Putaway\)/i })
    fireEvent.click(palletTabBtn)

    // Step 1: Scan pallet
    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'LPN-20261008-0001' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    expect(screen.getByText('Informasi Palet Terpilih')).toBeInTheDocument()

    // Click "Ganti Palet"
    const resetBtn = screen.getByRole('button', { name: /Ganti Palet/i })
    fireEvent.click(resetBtn)

    expect(screen.queryByText('Informasi Palet Terpilih')).not.toBeInTheDocument()
    expect(screen.getByText('Belum Terpilih')).toBeInTheDocument()
  })

  it('Step 2: scanning invalid rack location plays error tone and shows error message', () => {
    render(<BarcodeScannerPage />)

    const palletTabBtn = screen.getByRole('button', { name: /Palet \(LPN Putaway\)/i })
    fireEvent.click(palletTabBtn)

    // Step 1: Scan pallet
    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'LPN-20261008-0001' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    // Step 2: Scan invalid rack
    fireEvent.change(input, { target: { value: 'RAK-INVALID-99' } })
    fireEvent.click(scanBtn)

    expect(mockPlayTone).toHaveBeenCalledWith('error')
    expect(screen.getByText(/Lokasi rak \[RAK-INVALID-99\] tidak valid atau tidak terdaftar!/i)).toBeInTheDocument()
    expect(mockMoveLPNMutate).not.toHaveBeenCalled()
  })

  it('Step 2: scanning valid rack triggers moveLPN mutation and on success plays success tone, shows banner, and resets for next pallet', async () => {
    mockMoveLPNMutate.mockImplementation(({ id, data }, { onSuccess }) => {
      onSuccess({
        id,
        location_id: data.target_location_id,
      })
    })

    render(<BarcodeScannerPage />)

    const palletTabBtn = screen.getByRole('button', { name: /Palet \(LPN Putaway\)/i })
    fireEvent.click(palletTabBtn)

    // Step 1: Scan Pallet LPN
    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'LPN-20261008-0001' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    // Step 2: Scan Rak Tujuan (RAK-A-01)
    fireEvent.change(input, { target: { value: 'RAK-A-01' } })
    fireEvent.click(scanBtn)

    expect(mockMoveLPNMutate).toHaveBeenCalledWith(
      {
        id: 'lpn-1',
        data: { target_location_id: 'loc-rack-1' },
      },
      expect.any(Object)
    )

    expect(mockPlayTone).toHaveBeenCalledWith('success')
    expect(
      screen.getByText(
        /BERHASIL: Seluruh isi palet \[LPN-20261008-0001\] berhasil dipindahkan ke rak \[RAK-A-01\]!/i
      )
    ).toBeInTheDocument()

    // Steps reset for next pallet
    expect(screen.queryByText('Informasi Palet Terpilih')).not.toBeInTheDocument()
    expect(screen.getByText('Belum Terpilih')).toBeInTheDocument()
  })

  it('Step 2: triggers moveLPN mutation and on mutation error plays error tone and shows error banner', async () => {
    mockMoveLPNMutate.mockImplementation(({ id, data }, { onError }) => {
      onError(new Error('Kapasitas rak tidak mencukupi untuk menampung palet ini!'))
    })

    render(<BarcodeScannerPage />)

    const palletTabBtn = screen.getByRole('button', { name: /Palet \(LPN Putaway\)/i })
    fireEvent.click(palletTabBtn)

    // Step 1: Scan Pallet LPN
    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'LPN-20261008-0001' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    // Step 2: Scan Rak Tujuan (RAK-A-01)
    fireEvent.change(input, { target: { value: 'RAK-A-01' } })
    fireEvent.click(scanBtn)

    expect(mockMoveLPNMutate).toHaveBeenCalledWith(
      {
        id: 'lpn-1',
        data: { target_location_id: 'loc-rack-1' },
      },
      expect.any(Object)
    )

    expect(mockPlayTone).toHaveBeenCalledWith('error')
    expect(
      screen.getByText(/Kapasitas rak tidak mencukupi untuk menampung palet ini!/i)
    ).toBeInTheDocument()
  })
})
