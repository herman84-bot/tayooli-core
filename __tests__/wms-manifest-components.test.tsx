import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { SignatureCanvas } from '@/components/wms/SignatureCanvas'
import { PrintShippingManifest } from '@/components/wms/PrintShippingManifest'
import { ShippingManifestsPanel } from '@/components/wms/ShippingManifestsPanel'
import DeliveryOrdersPanel from '@/components/wms/DeliveryOrdersPanel'
import type { ShippingManifestDetail, ShippingManifest, DeliveryOrder } from '@/lib/api'

// Setup canvas mock for JSDOM
beforeAll(() => {
  HTMLCanvasElement.prototype.getContext = jest.fn().mockReturnValue({
    clearRect: jest.fn(),
    beginPath: jest.fn(),
    moveTo: jest.fn(),
    lineTo: jest.fn(),
    stroke: jest.fn(),
    lineWidth: 1,
    lineCap: 'round',
    lineJoin: 'round',
    strokeStyle: '#000',
  })
  HTMLCanvasElement.prototype.toDataURL = jest.fn().mockReturnValue('data:image/png;base64,mockedSignature')
  HTMLCanvasElement.prototype.getBoundingClientRect = jest.fn().mockReturnValue({
    left: 0,
    top: 0,
    width: 500,
    height: 180,
    right: 500,
    bottom: 180,
  })
  window.print = jest.fn()
})

// Mock sample data
const mockManifests: ShippingManifest[] = [
  {
    id: 'm-1',
    warehouse_id: 'wh-1',
    manifest_number: 'MAN-20261008-001',
    expedition_name: 'JNE Trucking (JTR)',
    driver_name: 'Budi Santoso',
    vehicle_plate: 'B 1234 CD',
    driver_phone: '08123456789',
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
    driver_phone: '08987654321',
    total_packages: 3,
    total_weight_kg: 40.0,
    status: 'LOADED',
    created_at: '2026-10-08T09:00:00Z',
    updated_at: '2026-10-08T09:00:00Z',
  },
  {
    id: 'm-3',
    warehouse_id: 'wh-1',
    manifest_number: 'MAN-20261008-003',
    expedition_name: 'J&T Express',
    driver_name: 'Heri Susanto',
    vehicle_plate: 'B 9999 GH',
    driver_phone: '0811223344',
    total_packages: 1,
    total_weight_kg: 10.0,
    status: 'DISPATCHED',
    driver_signature_svg: '<svg><path d="M 10 10 L 50 50" /></svg>',
    created_at: '2026-10-08T10:00:00Z',
    updated_at: '2026-10-08T10:00:00Z',
  },
]

const mockManifestDetail: ShippingManifestDetail = {
  manifest: mockManifests[0],
  items: [
    {
      delivery_order_id: 'do-1',
      do_number: 'DO/20261008/1001',
      customer_name: 'PT Maju Bersama',
      destination_city: 'Surabaya',
      package_weight_kg: 12.5,
      packaging_type: 'Koli Kardus',
      scanned: true,
      scanned_at: '2026-10-08T08:15:00Z',
    },
    {
      delivery_order_id: 'do-2',
      do_number: 'DO/20261008/1002',
      customer_name: 'CV Harapan Baru',
      destination_city: 'Semarang',
      package_weight_kg: 13.0,
      packaging_type: 'Koli Karung',
      scanned: false,
    },
  ],
}

const mockAvailableDOs: DeliveryOrder[] = [
  {
    id: 'do-packed-1',
    tenant_id: 't-1',
    warehouse_id: 'wh-1',
    do_number: 'DO/20261008/2001',
    status: 'PACKED',
    customer_name: 'Toko Sumber Rejeki',
    package_weight_kg: 15.0,
    packaging_type: 'Koli',
    sales_order_id: null,
    created_at: '2026-10-08T07:00:00Z',
    updated_at: '2026-10-08T07:00:00Z',
  },
  {
    id: 'do-packed-2',
    tenant_id: 't-1',
    warehouse_id: 'wh-1',
    do_number: 'DO/20261008/2002',
    status: 'PACKED',
    customer_name: 'PT Jaya Abadi',
    package_weight_kg: 20.0,
    packaging_type: 'Koli',
    sales_order_id: null,
    created_at: '2026-10-08T07:10:00Z',
    updated_at: '2026-10-08T07:10:00Z',
  },
]

// Mock hooks
jest.mock('@/hooks/useWMSManifests', () => {
  const manifests = [
    {
      id: 'm-1',
      warehouse_id: 'wh-1',
      manifest_number: 'MAN-20261008-001',
      expedition_name: 'JNE Trucking (JTR)',
      driver_name: 'Budi Santoso',
      vehicle_plate: 'B 1234 CD',
      driver_phone: '08123456789',
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
      driver_phone: '08987654321',
      total_packages: 3,
      total_weight_kg: 40.0,
      status: 'LOADED',
      created_at: '2026-10-08T09:00:00Z',
      updated_at: '2026-10-08T09:00:00Z',
    },
    {
      id: 'm-3',
      warehouse_id: 'wh-1',
      manifest_number: 'MAN-20261008-003',
      expedition_name: 'J&T Express',
      driver_name: 'Heri Susanto',
      vehicle_plate: 'B 9999 GH',
      driver_phone: '0811223344',
      total_packages: 1,
      total_weight_kg: 10.0,
      status: 'DISPATCHED',
      driver_signature_svg: '<svg><path d="M 10 10 L 50 50" /></svg>',
      created_at: '2026-10-08T10:00:00Z',
      updated_at: '2026-10-08T10:00:00Z',
    },
  ]

  const detail = {
    manifest: manifests[0],
    items: [
      {
        delivery_order_id: 'do-1',
        do_number: 'DO/20261008/1001',
        customer_name: 'PT Maju Bersama',
        destination_city: 'Surabaya',
        package_weight_kg: 12.5,
        packaging_type: 'Koli Kardus',
        scanned: true,
        scanned_at: '2026-10-08T08:15:00Z',
      },
      {
        delivery_order_id: 'do-2',
        do_number: 'DO/20261008/1002',
        customer_name: 'CV Harapan Baru',
        destination_city: 'Semarang',
        package_weight_kg: 13.0,
        packaging_type: 'Koli Karung',
        scanned: false,
      },
    ],
  }

  return {
    useShippingManifests: (params?: any) => {
      let filtered = [...manifests]
      if (params?.status && params.status !== 'ALL') {
        filtered = filtered.filter((m) => m.status === params.status)
      }
      return {
        data: filtered,
        isLoading: false,
        refetch: jest.fn(),
        isFetching: false,
      }
    },
    useShippingManifestDetail: (id?: string) => ({
      data: id ? detail : null,
      isLoading: false,
      refetch: jest.fn(),
    }),
    useCreateShippingManifest: () => ({
      mutateAsync: (global as any).__mockMutateCreate,
      isPending: false,
    }),
    useScanLoadingDO: () => ({
      mutateAsync: (global as any).__mockMutateScan,
      isPending: false,
    }),
    useDispatchShippingManifest: () => ({
      mutateAsync: (global as any).__mockMutateDispatch,
      isPending: false,
    }),
    useWMSOutboundKPI: () => ({
      data: {
        dock_to_stock_avg_minutes: 45,
        receiving_accuracy_pct: 99.2,
        po_compliance_pct: 98.5,
        backlog_inbound_count: 3,
      },
      isLoading: false,
    }),
  }
})

jest.mock('@/hooks/useWMS', () => ({
  useWarehouses: () => ({
    data: [{ id: 'wh-1', code: 'WH-MAIN', name: 'Gudang Utama Cakung', is_active: true }],
    isLoading: false,
  }),
  useWarehouseLocations: () => ({
    data: [{ id: 'loc-1', code: 'RCK-A-01', name: 'Rak A1', type: 'INTERNAL' }],
    isLoading: false,
  }),
  useDeliveryOrders: () => ({
    data: [
      {
        id: 'do-packed-1',
        tenant_id: 't-1',
        warehouse_id: 'wh-1',
        do_number: 'DO/20261008/2001',
        status: 'PACKED',
        customer_name: 'Toko Sumber Rejeki',
        package_weight_kg: 15.0,
        packaging_type: 'Koli',
        sales_order_id: null,
        created_at: '2026-10-08T07:00:00Z',
        updated_at: '2026-10-08T07:00:00Z',
      },
      {
        id: 'do-packed-2',
        tenant_id: 't-1',
        warehouse_id: 'wh-1',
        do_number: 'DO/20261008/2002',
        status: 'PACKED',
        customer_name: 'PT Jaya Abadi',
        package_weight_kg: 20.0,
        packaging_type: 'Koli',
        sales_order_id: null,
        created_at: '2026-10-08T07:10:00Z',
        updated_at: '2026-10-08T07:10:00Z',
      },
    ],
    isLoading: false,
    refetch: jest.fn(),
    isFetching: false,
  }),
  useCreateDeliveryOrder: () => ({
    mutateAsync: jest.fn().mockResolvedValue({}),
    isPending: false,
  }),
  useDispatchDeliveryOrder: () => ({
    mutateAsync: jest.fn().mockResolvedValue({}),
    isPending: false,
  }),
  useAuditTrail: () => ({
    data: [],
    isLoading: false,
  }),
}))

jest.mock('@/hooks/useProducts', () => ({
  useProducts: () => ({
    data: [{ id: 'prod-1', name: 'Beras Ramos 5kg', sku: 'BRS-RAMOS-5K' }],
    isLoading: false,
  }),
}))

jest.mock('@/hooks/useWMSLedger', () => ({
  useWMSStock: () => ({
    data: [{ product_id: 'prod-1', location_id: 'loc-1', quantity: 100, available_qty: 100 }],
    isLoading: false,
  }),
  useWMSMovements: () => ({ data: [], isLoading: false }),
}))

jest.mock('@/lib/api', () => ({
  ...jest.requireActual('@/lib/api'),
  api: {
    wms: {
      manifests: {
        list: jest.fn().mockResolvedValue({ data: [] }),
        get: jest.fn().mockImplementation(() =>
          Promise.resolve({
            data: {
              manifest: {
                id: 'm-1',
                warehouse_id: 'wh-1',
                manifest_number: 'MAN-20261008-001',
                expedition_name: 'JNE Trucking (JTR)',
                driver_name: 'Budi Santoso',
                vehicle_plate: 'B 1234 CD',
                driver_phone: '08123456789',
                total_packages: 2,
                total_weight_kg: 25.5,
                status: 'STAGED',
                created_at: '2026-10-08T08:00:00Z',
                updated_at: '2026-10-08T08:00:00Z',
              },
              items: [],
            },
          })
        ),
        create: jest.fn().mockResolvedValue({ data: {} }),
        scanLoading: jest.fn().mockResolvedValue({ data: {} }),
        dispatch: jest.fn().mockResolvedValue({ data: {} }),
      },
      deliveryOrders: {
        list: jest.fn().mockResolvedValue({ data: [] }),
        get: jest.fn().mockResolvedValue({ data: { order: {}, items: [] } }),
      },
      pickWaves: {
        list: jest.fn().mockResolvedValue({ data: [] }),
      },
    },
    customers: {
      list: jest.fn().mockResolvedValue({ data: [] }),
      create: jest.fn().mockResolvedValue({ id: 'c-new', name: 'Toko Baru' }),
    },
  },
}))

const mockMutateCreate = jest.fn()
const mockMutateScan = jest.fn()
const mockMutateDispatch = jest.fn()

;(global as any).__mockMutateCreate = mockMutateCreate
;(global as any).__mockMutateScan = mockMutateScan
;(global as any).__mockMutateDispatch = mockMutateDispatch

describe('Task 5: WMS Shipping Manifests & Digital Signature Components', () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockMutateCreate.mockResolvedValue({
      data: {
        id: 'm-1',
        warehouse_id: 'wh-1',
        manifest_number: 'MAN-20261008-001',
        expedition_name: 'JNE Trucking (JTR)',
        driver_name: 'Supir Uji Coba',
        vehicle_plate: 'B 7777 ABC',
        total_packages: 1,
        total_weight_kg: 15.0,
        status: 'STAGED',
        created_at: '2026-10-08T08:00:00Z',
        updated_at: '2026-10-08T08:00:00Z',
      },
    })
    mockMutateScan.mockResolvedValue({ data: {} })
    mockMutateDispatch.mockResolvedValue({ data: {} })
  })

  describe('1. SignatureCanvas Component', () => {
    it('renders signature canvas and placeholder prompt', () => {
      render(<SignatureCanvas onSave={jest.fn()} />)

      expect(screen.getByLabelText(/Kanvas Tanda Tangan Digital/i)).toBeInTheDocument()
      expect(screen.getByText(/Tanda tangani di sini dengan mouse atau jari/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Hapus \/ Ulangi/i })).toBeDisabled()
      expect(screen.getByRole('button', { name: /Terapkan Tanda Tangan/i })).toBeDisabled()
    })

    it('validates empty canvas if save is attempted without stroke', () => {
      const onSaveMock = jest.fn()
      render(<SignatureCanvas onSave={onSaveMock} />)

      // Directly trigger save (even if button is disabled in UI, test programmatic safety)
      const saveBtn = screen.getByRole('button', { name: /Terapkan Tanda Tangan/i })
      fireEvent.click(saveBtn)

      expect(onSaveMock).not.toHaveBeenCalled()
    })

    it('captures strokes via mouse events and calls onSave with SVG string', async () => {
      const onSaveMock = jest.fn()
      render(<SignatureCanvas onSave={onSaveMock} />)

      const canvas = screen.getByLabelText(/Kanvas Tanda Tangan Digital/i)

      // Simulate mouse draw
      fireEvent.mouseDown(canvas, { clientX: 20, clientY: 30 })
      fireEvent.mouseMove(canvas, { clientX: 50, clientY: 60 })
      fireEvent.mouseMove(canvas, { clientX: 80, clientY: 90 })
      fireEvent.mouseUp(canvas)

      // Save button should now be enabled
      const saveBtn = screen.getByRole('button', { name: /Terapkan Tanda Tangan/i })
      expect(saveBtn).not.toBeDisabled()

      fireEvent.click(saveBtn)

      expect(onSaveMock).toHaveBeenCalledTimes(1)
      const arg = onSaveMock.mock.calls[0][0]
      expect(arg).toContain('<svg')
      expect(arg).toContain('path')
    })

    it('clears drawing when clear button is clicked', () => {
      const onSaveMock = jest.fn()
      render(<SignatureCanvas onSave={onSaveMock} />)

      const canvas = screen.getByLabelText(/Kanvas Tanda Tangan Digital/i)
      fireEvent.mouseDown(canvas, { clientX: 10, clientY: 10 })
      fireEvent.mouseMove(canvas, { clientX: 30, clientY: 30 })
      fireEvent.mouseUp(canvas)

      const clearBtn = screen.getByRole('button', { name: /Hapus \/ Ulangi/i })
      expect(clearBtn).not.toBeDisabled()

      fireEvent.click(clearBtn)
      expect(clearBtn).toBeDisabled()
    })
  })

  describe('2. PrintShippingManifest Component', () => {
    it('renders manifest document header, expedition metadata, items, and totals', () => {
      render(
        <PrintShippingManifest
          manifestDetail={mockManifestDetail}
          onClose={jest.fn()}
        />
      )

      expect(screen.getAllByText(/MANIFEST SERAH TERIMA PENGIRIMAN/i).length).toBeGreaterThan(0)
      expect(screen.getByText('MAN-20261008-001')).toBeInTheDocument()
      expect(screen.getByText(/: JNE Trucking \(JTR\)/i)).toBeInTheDocument()
      expect(screen.getByText(/: Budi Santoso/i)).toBeInTheDocument()
      expect(screen.getAllByText(/B 1234 CD/i).length).toBeGreaterThan(0)

      // Delivery Order rows
      expect(screen.getByText('DO/20261008/1001')).toBeInTheDocument()
      expect(screen.getByText('PT Maju Bersama')).toBeInTheDocument()
      expect(screen.getByText('Surabaya')).toBeInTheDocument()
      expect(screen.getByText('DO/20261008/1002')).toBeInTheDocument()
      expect(screen.getByText('Semarang')).toBeInTheDocument()

      // Summary
      expect(screen.getByText(/TOTAL SUMMARY:/i)).toBeInTheDocument()
      expect(screen.getByText(/25.50 kg/i)).toBeInTheDocument()
      expect(screen.getAllByText('2').length).toBeGreaterThan(0) // 2 koli

      // Signature section
      expect(screen.getByText(/Diserahkan Oleh \(Staf Gudang\)/i)).toBeInTheDocument()
      expect(screen.getByText(/Diterima Oleh \(Sopir Ekspedisi\)/i)).toBeInTheDocument()
    })

    it('triggers window.print and onClose actions', () => {
      const onCloseMock = jest.fn()
      render(
        <PrintShippingManifest
          manifestDetail={mockManifestDetail}
          onClose={onCloseMock}
        />
      )

      const printBtn = screen.getByRole('button', { name: /Cetak \/ Cetak PDF/i })
      fireEvent.click(printBtn)
      expect(window.print).toHaveBeenCalledTimes(1)

      const closeBtn = screen.getByRole('button', { name: /Tutup/i })
      fireEvent.click(closeBtn)
      expect(onCloseMock).toHaveBeenCalledTimes(1)
    })
  })

  describe('3. ShippingManifestsPanel Component', () => {
    it('renders metrics summary and manifests list with status badges', () => {
      render(<ShippingManifestsPanel warehouseId="wh-1" />)

      // Metrics
      expect(screen.getByText('Total Manifest')).toBeInTheDocument()
      expect(screen.getByText('Staged (Siap Muat)')).toBeInTheDocument()
      expect(screen.getByText('Loaded (Terpindai Penuh)')).toBeInTheDocument()
      expect(screen.getByText('Dispatched (Berangkat)')).toBeInTheDocument()

      // Table rows
      expect(screen.getByText('MAN-20261008-001')).toBeInTheDocument()
      expect(screen.getByText('MAN-20261008-002')).toBeInTheDocument()
      expect(screen.getByText('MAN-20261008-003')).toBeInTheDocument()
      expect(screen.getAllByText(/STAGED \(Siap Muat\)/i).length).toBeGreaterThan(0)
      expect(screen.getByText(/LOADED \(Terpindai\)/i)).toBeInTheDocument()
      expect(screen.getAllByText(/DISPATCHED \(Berangkat\)/i).length).toBeGreaterThan(0)
    })

    it('opens Create Manifest modal, calculates selected DOs weight, and submits', async () => {
      render(<ShippingManifestsPanel warehouseId="wh-1" />)

      // Click "Buat Manifest Baru"
      const createBtn = screen.getByRole('button', { name: /Buat Manifest Baru/i })
      fireEvent.click(createBtn)

      expect(screen.getByText(/Buat Manifest Ekspedisi Baru/i)).toBeInTheDocument()
      expect(screen.getByText(/Pilih Surat Jalan Siap Muat/i)).toBeInTheDocument()

      // Fill driver & vehicle plate
      const driverInput = screen.getByPlaceholderText(/Contoh: Budi Santoso/i)
      const plateInput = screen.getByPlaceholderText(/Contoh: B 9482 TYN/i)
      fireEvent.change(driverInput, { target: { value: 'Supir Uji Coba' } })
      fireEvent.change(plateInput, { target: { value: 'B 7777 ABC' } })

      // Select first DO checkbox
      const doRows = screen.getAllByRole('checkbox')
      expect(doRows.length).toBeGreaterThan(1) // Header checkbox + row checkboxes
      fireEvent.click(doRows[1])

      // Verify live summary
      expect(screen.getByText(/Terpilih:/i)).toHaveTextContent(/1 Surat Jalan/i)

      // Submit form
      const submitBtn = screen.getByRole('button', { name: /^Buat Manifest$/i })
      fireEvent.click(submitBtn)

      await waitFor(() => {
        expect(mockMutateCreate).toHaveBeenCalledTimes(1)
        expect(mockMutateCreate).toHaveBeenCalledWith(
          expect.objectContaining({
            driver_name: 'Supir Uji Coba',
            vehicle_plate: 'B 7777 ABC',
            delivery_order_ids: ['do-packed-1'],
          })
        )
      })
    })

    it('opens Loading Scan drawer and submits scanned barcode', async () => {
      render(<ShippingManifestsPanel warehouseId="wh-1" />)

      // Click first "Scan / Detail" button
      const scanBtns = screen.getAllByRole('button', { name: /Scan \/ Detail/i })
      fireEvent.click(scanBtns[0])

      expect(screen.getByText(/Loading Scan & Verifikasi Koli Truk/i)).toBeInTheDocument()
      expect(screen.getByText(/Progress Pemuatan ke Truk:/i)).toBeInTheDocument()

      // Verify item rows
      expect(screen.getByText('DO/20261008/1001')).toBeInTheDocument()
      expect(screen.getByText('Termuat')).toBeInTheDocument()
      expect(screen.getByText('DO/20261008/1002')).toBeInTheDocument()

      // Scan barcode input
      const scanInput = screen.getByPlaceholderText(/Scan barcode Surat Jalan/i)
      fireEvent.change(scanInput, { target: { value: 'DO/20261008/1002' } })

      const verifyBtn = screen.getByRole('button', { name: /Verifikasi Muat/i })
      fireEvent.click(verifyBtn)

      await waitFor(() => {
        expect(mockMutateScan).toHaveBeenCalledTimes(1)
        expect(mockMutateScan).toHaveBeenCalledWith({
          id: 'm-1',
          barcode: 'DO/20261008/1002',
        })
      })
    })

    it('opens Dispatch modal with digital signature and submits dispatch mutation', async () => {
      render(<ShippingManifestsPanel warehouseId="wh-1" />)

      // Click Dispatch on first manifest
      const dispatchBtns = screen.getAllByRole('button', { name: /^Dispatch$/i })
      fireEvent.click(dispatchBtns[0])

      expect(screen.getByText(/Dispatch & Serah Terima Pengemudi/i)).toBeInTheDocument()
      expect(screen.getByText(/Tanda Tangan Digital Pengemudi \/ Kurir/i)).toBeInTheDocument()

      // Draw signature on canvas
      const canvas = screen.getByLabelText(/Kanvas Tanda Tangan Digital/i)
      fireEvent.mouseDown(canvas, { clientX: 10, clientY: 10 })
      fireEvent.mouseMove(canvas, { clientX: 40, clientY: 40 })
      fireEvent.mouseUp(canvas)

      // Click "Terapkan Tanda Tangan"
      const applySigBtn = screen.getByRole('button', { name: /Terapkan Tanda Tangan/i })
      fireEvent.click(applySigBtn)

      expect(screen.getByText(/Tanda tangan tersimpan dan siap diverifikasi/i)).toBeInTheDocument()

      // Click "Konfirmasi & Dispatch"
      const confirmDispatchBtn = screen.getByRole('button', { name: /Konfirmasi & Dispatch/i })
      fireEvent.click(confirmDispatchBtn)

      await waitFor(() => {
        expect(mockMutateDispatch).toHaveBeenCalledTimes(1)
        expect(mockMutateDispatch).toHaveBeenCalledWith(
          expect.objectContaining({
            id: 'm-1',
            driver_signature_svg: expect.stringContaining('<svg'),
          })
        )
      })
    })
  })

  describe('4. DeliveryOrdersPanel Tab Toggle Integration', () => {
    it('toggles between Surat Jalan (DO) and Manifest Ekspedisi submodules', async () => {
      render(<DeliveryOrdersPanel />)

      // Initially shows Surat Jalan (DO) tab active
      expect(screen.getByRole('button', { name: /Surat Jalan \(DO\)/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Manifest Ekspedisi/i })).toBeInTheDocument()
      expect(screen.getByText('Total Surat Jalan')).toBeInTheDocument()

      // Switch to Manifest Ekspedisi tab
      const manifestTabBtn = screen.getByRole('button', { name: /Manifest Ekspedisi/i })
      fireEvent.click(manifestTabBtn)

      // ShippingManifestsPanel should now be rendered
      expect(screen.getByText(/Total Manifest/i)).toBeInTheDocument()
      expect(screen.getByText('MAN-20261008-001')).toBeInTheDocument()

      // Switch back to DO tab
      const doTabBtn = screen.getByRole('button', { name: /Surat Jalan \(DO\)/i })
      fireEvent.click(doTabBtn)

      expect(screen.getByText(/Total Surat Jalan/i)).toBeInTheDocument()
    })
  })
})
