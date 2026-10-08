import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import BarcodeScannerPage from '@/app/(app)/wms/scanner/page'
import type { ShippingManifest, ShippingManifestDetail } from '@/lib/api'

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
  useResolveBarcode: () => ({
    data: null,
    isFetching: false,
    isError: false,
  }),
  useWarehouseLocations: () => ({
    data: [{ id: 'loc-1', code: 'RAK-A-01', name: 'Rak A1' }],
  }),
}))

// Mock manifests data
const mockActiveManifests: ShippingManifest[] = [
  {
    id: 'm-1',
    warehouse_id: 'wh-1',
    manifest_number: 'MAN-20261008-001',
    expedition_name: 'JNE Trucking',
    driver_name: 'Budi Santoso',
    vehicle_plate: 'B 1234 CD',
    total_packages: 2,
    total_weight_kg: 25.5,
    status: 'STAGED',
    created_at: '2026-10-08T08:00:00Z',
    updated_at: '2026-10-08T08:00:00Z',
  },
  {
    id: 'm-2',
    warehouse_id: 'wh-1',
    manifest_number: 'MAN-20261008-002',
    expedition_name: 'SiCepat Cargo',
    driver_name: 'Agus Pratama',
    vehicle_plate: 'D 5678 EF',
    total_packages: 3,
    total_weight_kg: 40.0,
    status: 'LOADED',
    created_at: '2026-10-08T09:00:00Z',
    updated_at: '2026-10-08T09:00:00Z',
  },
]

const mockManifestDetail: ShippingManifestDetail = {
  manifest: mockActiveManifests[0],
  items: [
    {
      delivery_order_id: 'do-1',
      do_number: 'DO-20261008-001',
      customer_name: 'PT Toko Makmur',
      scanned: false,
    },
    {
      delivery_order_id: 'do-2',
      do_number: 'DO-20261008-002',
      customer_name: 'UD Sumber Rejeki',
      scanned: true,
      scanned_at: '2026-10-08T10:00:00Z',
      scanned_by_name: 'Operator 1',
    },
  ],
}

const mockScanLoadingMutate = jest.fn()

jest.mock('@/hooks/useWMSManifests', () => ({
  useShippingManifests: () => ({
    data: mockActiveManifests,
    isLoading: false,
  }),
  useShippingManifestDetail: (id?: string) => ({
    data: id === 'm-1' ? mockManifestDetail : null,
    isLoading: false,
  }),
  useScanLoadingDO: () => ({
    mutate: mockScanLoadingMutate,
    isPending: false,
  }),
}))

describe('Task 6: Barcode Scanner Truck Loading Mode', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  it('renders mode switch button for Truck (Loading Truk)', () => {
    render(<BarcodeScannerPage />)

    const truckTabBtn = screen.getByRole('button', { name: /Truck \(Loading Truk\)/i })
    expect(truckTabBtn).toBeInTheDocument()
  })

  it('switches to LOADING_TRUCK mode and renders manifest dropdown', () => {
    render(<BarcodeScannerPage />)

    const truckTabBtn = screen.getByRole('button', { name: /Truck \(Loading Truk\)/i })
    fireEvent.click(truckTabBtn)

    expect(screen.getByText(/Pemuatan Armada:/i)).toBeInTheDocument()
    expect(
      screen.getByLabelText(/Pilih Manifest Pengiriman \(Armada \/ Truk\)/i)
    ).toBeInTheDocument()

    // Options exist in select
    expect(screen.getByText(/MAN-20261008-001/i)).toBeInTheDocument()
    expect(screen.getByText(/MAN-20261008-002/i)).toBeInTheDocument()
  })

  it('displays warning if barcode scanned without selecting manifest', () => {
    render(<BarcodeScannerPage />)

    const truckTabBtn = screen.getByRole('button', { name: /Truck \(Loading Truk\)/i })
    fireEvent.click(truckTabBtn)

    // Manual input without selecting manifest
    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'DO-20261008-001' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    expect(mockPlayTone).toHaveBeenCalledWith('error')
    expect(screen.getByText('Pilih manifest tujuan terlebih dahulu')).toBeInTheDocument()
    expect(mockScanLoadingMutate).not.toHaveBeenCalled()
  })

  it('displays manifest metadata, progress bar, and DO checklist when manifest selected', () => {
    render(<BarcodeScannerPage />)

    const truckTabBtn = screen.getByRole('button', { name: /Truck \(Loading Truk\)/i })
    fireEvent.click(truckTabBtn)

    const select = screen.getByLabelText(/Pilih Manifest Pengiriman \(Armada \/ Truk\)/i)
    fireEvent.change(select, { target: { value: 'm-1' } })

    // Manifest metadata
    expect(screen.getByText('Budi Santoso')).toBeInTheDocument()
    expect(screen.getByText('B 1234 CD')).toBeInTheDocument()

    // Progress counter: 1 of 2 (UD Sumber Rejeki is scanned)
    expect(screen.getByText(/1 dari 2 Koli Termuat \(50%\)/i)).toBeInTheDocument()

    // DO items & badges
    expect(screen.getByText('DO-20261008-001')).toBeInTheDocument()
    expect(screen.getByText('DO-20261008-002')).toBeInTheDocument()
    expect(screen.getByText('Belum Dimuat')).toBeInTheDocument()
    expect(screen.getByText('Dimuat')).toBeInTheDocument()
  })

  it('calls scanLoadingMutation and plays success tone on successful scan', async () => {
    mockScanLoadingMutate.mockImplementation(({ manifestId, barcode }, { onSuccess }) => {
      onSuccess({ success: true })
    })

    render(<BarcodeScannerPage />)

    const truckTabBtn = screen.getByRole('button', { name: /Truck \(Loading Truk\)/i })
    fireEvent.click(truckTabBtn)

    const select = screen.getByLabelText(/Pilih Manifest Pengiriman \(Armada \/ Truk\)/i)
    fireEvent.change(select, { target: { value: 'm-1' } })

    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'DO-20261008-001' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    expect(mockScanLoadingMutate).toHaveBeenCalledWith(
      { manifestId: 'm-1', barcode: 'DO-20261008-001' },
      expect.any(Object)
    )

    expect(mockPlayTone).toHaveBeenCalledWith('success')
    expect(
      screen.getByText(/Surat Jalan \[DO-20261008-001\] berhasil dimuat ke armada!/i)
    ).toBeInTheDocument()
  })

  it('calls scanLoadingMutation and plays failure tone with misload banner on error', async () => {
    mockScanLoadingMutate.mockImplementation(({ manifestId, barcode }, { onError }) => {
      onError(new Error('MISLOAD: Barcode tidak terdaftar dalam manifest ini!'))
    })

    render(<BarcodeScannerPage />)

    const truckTabBtn = screen.getByRole('button', { name: /Truck \(Loading Truk\)/i })
    fireEvent.click(truckTabBtn)

    const select = screen.getByLabelText(/Pilih Manifest Pengiriman \(Armada \/ Truk\)/i)
    fireEvent.change(select, { target: { value: 'm-1' } })

    const input = screen.getByPlaceholderText(/Ketik barcode \/ SKU lalu Enter/i)
    fireEvent.change(input, { target: { value: 'UNKNOWN-DO-999' } })
    const scanBtn = screen.getByRole('button', { name: /^Scan$/i })
    fireEvent.click(scanBtn)

    expect(mockScanLoadingMutate).toHaveBeenCalledWith(
      { manifestId: 'm-1', barcode: 'UNKNOWN-DO-999' },
      expect.any(Object)
    )

    expect(mockPlayTone).toHaveBeenCalledWith('error')
    expect(screen.getByText('PERINGATAN MISLOAD')).toBeInTheDocument()
    expect(
      screen.getByText(/MISLOAD: Barcode tidak terdaftar dalam manifest ini!/i)
    ).toBeInTheDocument()
  })
})
