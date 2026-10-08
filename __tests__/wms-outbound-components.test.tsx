import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom'
import { PrintPickingList } from '@/components/wms/PrintPickingList'
import { PrintThermalAWB } from '@/components/wms/PrintThermalAWB'
import { WaveReleaseModal } from '@/components/wms/WaveReleaseModal'
import { WavePickingSubView } from '@/components/wms/WavePickingSubView'
import { PackingStationSubView } from '@/components/wms/PackingStationSubView'
import { ActivityTimelineDrawer } from '@/components/wms/ActivityTimelineDrawer'
import DeliveryOrdersPanel from '@/components/wms/DeliveryOrdersPanel'
import { PickingTaskDetail, DeliveryOrder, DeliveryOrderItem } from '@/lib/api'

// Mock useWarehouses and useWarehouseLocations
jest.mock('@/hooks/useWMS', () => ({
  useWarehouses: () => ({
    data: [{ id: 'wh-1', code: 'WH-MAIN', name: 'Gudang Utama', is_active: true }],
    isLoading: false,
  }),
  useWarehouseLocations: () => ({
    data: [{ id: 'loc-1', code: 'RCK-A-01', name: 'Rak A1', type: 'INTERNAL' }],
    isLoading: false,
  }),
  useDeliveryOrders: () => ({
    data: [],
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

// Mock useProducts
jest.mock('@/hooks/useProducts', () => ({
  useProducts: () => ({
    data: [{ id: 'prod-1', name: 'Beras Ramos 5kg', sku: 'BRS-RAMOS-5K' }],
    isLoading: false,
  }),
}))

// Mock api
jest.mock('@/lib/api', () => ({
  ...jest.requireActual('@/lib/api'),
  api: {
    wms: {
      pickWaves: {
        list: jest.fn().mockResolvedValue({
          data: [
            {
              id: 'w-1',
              wave_number: 'WAVE-20261008-001',
              order_type: 'MARKETPLACE',
              expedition_name: 'JNE',
              route_zone: 'JABODETABEK',
              status: 'OPEN',
              total_orders: 2,
              total_lines: 4,
              created_at: '2026-10-08T08:00:00Z',
              updated_at: '2026-10-08T08:00:00Z',
            },
          ],
        }),
        create: jest.fn().mockResolvedValue({
          data: {
            id: 'w-1',
            wave_number: 'WAVE-20261008-001',
            order_type: 'MARKETPLACE',
            status: 'OPEN',
            total_orders: 2,
            total_lines: 4,
          },
        }),
        get: jest.fn().mockResolvedValue({
          data: {
            wave: { id: 'w-1', wave_number: 'WAVE-20261008-001', status: 'OPEN', total_orders: 2, total_lines: 4 },
            picking_tasks: [],
          },
        }),
        release: jest.fn().mockResolvedValue({ data: { id: 'w-1', status: 'RELEASED' } }),
      },
      deliveryOrders: {
        list: jest.fn().mockResolvedValue({
          data: [
            {
              id: 'do-123',
              do_number: 'DO-20261008-0001',
              warehouse_id: 'wh-1',
              customer_name: 'PT Maju Bersama',
              recipient_name: 'Bpk. Budi Santoso',
              order_type: 'DIRECT_DO',
              status: 'CONFIRMED',
              created_at: '2026-10-08T08:00:00Z',
            },
          ],
        }),
        get: jest.fn().mockResolvedValue({
          delivery_order: {
            id: 'do-123',
            do_number: 'DO-20261008-0001',
            status: 'CONFIRMED',
          },
          items: [
            {
              id: 'doi-1',
              delivery_order_id: 'do-123',
              product_id: 'prod-1',
              product_name: 'Beras Ramos 5kg',
              product_sku: 'BRS-RAMOS-5K',
              quantity: '2',
              packed_qty: '0',
            },
          ],
        }),
        scanPackItem: jest.fn().mockResolvedValue({
          data: { item_id: 'doi-1', packed_qty: '1', product_name: 'Beras 5kg', product_sku: 'BRS-01', item_completed: false },
        }),
        completePack: jest.fn().mockResolvedValue({ data: { id: 'do-123', status: 'PACKED' } }),
      },
      trace: {
        auditTrail: jest.fn().mockResolvedValue({
          data: [
            {
              id: 'log-1',
              entity_type: 'delivery_order',
              entity_id: 'do-123',
              action: 'ORDER_CONFIRMED',
              user_name: 'Bpk. Supervisor',
              created_at: '2026-10-08T09:00:00Z',
            },
          ],
        }),
      },
    },
    customers: {
      list: jest.fn().mockResolvedValue({
        data: [{ id: 'cust-1', name: 'PT Maju Bersama', phone: '08123456789' }],
      }),
      create: jest.fn().mockResolvedValue({
        id: 'cust-new',
        name: 'Toko Berkah Baru',
        phone: '08999999999',
        address: 'Jl. Merdeka No. 1',
      }),
    },
  },
}))

describe('WMS Outbound Components (FE-06, FE-07, FE-08, CR-02b)', () => {
  const dummyDO: DeliveryOrder = {
    id: 'do-123',
    tenant_id: 'tenant-1',
    do_number: 'DO-20261008-0001',
    sales_order_id: null,
    warehouse_id: 'wh-1',
    customer_id: 'cust-1',
    customer_name: 'PT Maju Bersama',
    recipient_name: 'Bpk. Budi Santoso',
    order_type: 'DIRECT_DO',
    expedition_name: 'JNE',
    tracking_number: 'JNE88990011',
    status: 'CONFIRMED',
    package_weight_kg: '2.5',
    package_length_cm: '30',
    package_width_cm: '20',
    package_height_cm: '15',
    packaging_type: 'KARTON',
    created_at: '2026-10-08T08:00:00Z',
    updated_at: '2026-10-08T08:00:00Z',
  }

  const dummyItems: DeliveryOrderItem[] = [
    {
      id: 'doi-1',
      tenant_id: 'tenant-1',
      delivery_order_id: 'do-123',
      product_id: 'prod-1',
      product_name: 'Beras Ramos 5kg',
      product_sku: 'BRS-RAMOS-5K',
      quantity: '5',
      packed_qty: '0',
      location_id: 'loc-1',
      location_code: 'RCK-A-01',
      batch_number: 'LOT-2026-001',
      expiry_date: '2027-01-01T00:00:00Z',
      is_free_item: false,
      created_at: '2026-10-08T08:00:00Z',
    },
    {
      id: 'doi-2',
      tenant_id: 'tenant-1',
      delivery_order_id: 'do-123',
      product_id: 'prod-2',
      product_name: 'Minyak Goreng 1L (Bonus)',
      product_sku: 'MNY-BONUS-1L',
      quantity: '1',
      packed_qty: '0',
      location_id: 'loc-2',
      location_code: 'RCK-B-05',
      batch_number: 'LOT-2026-FREE',
      expiry_date: '2027-06-01T00:00:00Z',
      is_free_item: true,
      created_at: '2026-10-08T08:00:00Z',
    },
  ]

  const dummyTaskDetail: PickingTaskDetail = {
    task: {
      id: 'pt-1',
      delivery_order_id: 'do-123',
      task_number: 'PICK-DO-20261008-0001',
      status: 'IN_PROGRESS',
      picker_name: 'Ahmad Dani',
    },
    items: [
      {
        id: 'pti-1',
        task_id: 'pt-1',
        product_id: 'prod-1',
        product_name: 'Beras Ramos 5kg',
        product_sku: 'BRS-RAMOS-5K',
        batch_id: 'b-1',
        source_location_id: 'loc-1',
        location_code: 'RCK-A-01',
        shelf_order: 1,
        batch_number: 'LOT-2026-001',
        expiry_date: '2027-01-01T00:00:00Z',
        requested_qty: '5',
        picked_qty: '0',
        damaged_qty: '0',
        status: 'PENDING',
        is_free_item: false,
      },
      {
        id: 'pti-2',
        task_id: 'pt-1',
        product_id: 'prod-2',
        product_name: 'Minyak Goreng 1L (Bonus)',
        product_sku: 'MNY-BONUS-1L',
        batch_id: 'b-2',
        source_location_id: 'loc-2',
        location_code: 'RCK-B-05',
        shelf_order: 2,
        batch_number: 'LOT-2026-FREE',
        expiry_date: '2027-06-01T00:00:00Z',
        requested_qty: '1',
        picked_qty: '0',
        damaged_qty: '0',
        status: 'PENDING',
        is_free_item: true,
      },
    ],
    delivery_order: dummyDO,
  }

  describe('FE-06: PrintPickingList', () => {
    it('renders picking task header, shelf order, items, and BONUS badge for free item', () => {
      const onClose = jest.fn()
      render(<PrintPickingList detail={dummyTaskDetail} onClose={onClose} />)

      expect(screen.getByText(/DAFTAR AMBIL BARANG \(PICKING LIST\)/i)).toBeInTheDocument()
      expect(screen.getByText(/PICK-DO-20261008-0001/i)).toBeInTheDocument()
      expect(screen.getByText('DO-20261008-0001')).toBeInTheDocument()
      expect(screen.getByText(/RCK-A-01/i)).toBeInTheDocument()
      expect(screen.getByText(/RCK-B-05/i)).toBeInTheDocument()
      expect(screen.getByText(/Beras Ramos 5kg/i)).toBeInTheDocument()
      expect(screen.getByText(/Minyak Goreng 1L \(Bonus\)/i)).toBeInTheDocument()

      // Free item BONUS badge check (PDF-01)
      const bonusBadges = screen.getAllByText('BONUS')
      expect(bonusBadges.length).toBeGreaterThanOrEqual(1)
    })

    it('triggers window.print when print button is clicked', () => {
      const printSpy = jest.spyOn(window, 'print').mockImplementation(() => {})
      const onClose = jest.fn()
      render(<PrintPickingList detail={dummyTaskDetail} onClose={onClose} />)

      const printBtn = screen.getByRole('button', { name: /Cetak Dokumen/i })
      fireEvent.click(printBtn)
      expect(printSpy).toHaveBeenCalled()
      printSpy.mockRestore()
    })
  })

  describe('FE-08: PrintThermalAWB', () => {
    it('renders 100x150 mm thermal shipping sticker with barcode, tracking number, and package dimensions', () => {
      const onClose = jest.fn()
      render(<PrintThermalAWB order={dummyDO} items={dummyItems} onClose={onClose} />)

      expect(screen.getByText(/Label Thermal AWB \(100x150 mm\)/i)).toBeInTheDocument()
      expect(screen.getByText('JNE')).toBeInTheDocument()
      expect(screen.getByText('JNE88990011')).toBeInTheDocument()
      expect(screen.getByText(/PT Maju Bersama/i)).toBeInTheDocument()
      expect(screen.getByText(/2.5 kg/i)).toBeInTheDocument()
      expect(screen.getByText(/30x20x15 cm/i)).toBeInTheDocument()

      // Free item BONUS preview check
      expect(screen.getByText('BONUS')).toBeInTheDocument()
    })

    it('triggers window.print when Cetak Thermal button is clicked', () => {
      const printSpy = jest.spyOn(window, 'print').mockImplementation(() => {})
      const onClose = jest.fn()
      render(<PrintThermalAWB order={dummyDO} items={dummyItems} onClose={onClose} />)

      const printBtn = screen.getByRole('button', { name: /Cetak Thermal/i })
      fireEvent.click(printBtn)
      expect(printSpy).toHaveBeenCalled()
      printSpy.mockRestore()
    })
  })

  describe('WaveReleaseModal (PDF-05)', () => {
    const dummyWarehouses = [
      {
        id: 'wh-1',
        tenant_id: 'tenant-1',
        code: 'WH-MAIN',
        name: 'Gudang Utama',
        is_active: true,
        created_at: '2026-10-08T08:00:00Z',
        updated_at: '2026-10-08T08:00:00Z',
      },
    ]

    it('renders wave tabs and allows toggling to Create Wave form', async () => {
      const onClose = jest.fn()
      await React.act(async () => {
        render(
          <WaveReleaseModal
            isOpen={true}
            onClose={onClose}
            warehouses={dummyWarehouses}
            defaultWarehouseId="wh-1"
          />
        )
      })

      expect(screen.getByText(/Pelepasan Gelombang Picking/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Daftar Wave Aktif/i })).toBeInTheDocument()

      const createTabBtn = screen.getByRole('button', { name: /\+ Buat Wave Baru/i })
      await React.act(async () => {
        fireEvent.click(createTabBtn)
      })

      expect(screen.getByText(/Tipe Pesanan \/ Saluran \(Order Type\)/i)).toBeInTheDocument()
      expect(screen.getByText(/Surat Jalan Langsung \(DIRECT_DO\)/i)).toBeInTheDocument()
    })
  })

  describe('FE-07: WavePickingSubView & PackingStationSubView Real UI', () => {
    it('renders WavePickingSubView with metrics, wave list, and release buttons', async () => {
      await React.act(async () => {
        render(<WavePickingSubView warehouseId="wh-1" />)
      })

      expect(screen.getByText(/Total Gelombang/i)).toBeInTheDocument()
      expect(screen.getByText(/WAVE-20261008-001/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /\+ Buat Wave Baru/i })).toBeInTheDocument()
    })

    it('renders PackingStationSubView with order selection, scanner prompt, and audio beeper toggle', async () => {
      await React.act(async () => {
        render(<PackingStationSubView warehouseId="wh-1" />)
      })

      expect(screen.getByText(/Pesanan Siap Kemas/i)).toBeInTheDocument()
      expect(screen.getByText('DO-20261008-0001')).toBeInTheDocument()

      // Select order
      const orderBtn = screen.getByText('DO-20261008-0001')
      await React.act(async () => {
        fireEvent.click(orderBtn)
      })

      expect(screen.getByText(/Pindai Barcode \/ SKU Barang/i)).toBeInTheDocument()
      expect(screen.getByText(/Kemajuan Pemeriksaan Kemasan/i)).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /Audio Aktif/i })).toBeInTheDocument()
    })
  })

  describe('CR-02b: Customer Selection & Quick Add Modal', () => {
    it('opens Quick Customer modal and registers new customer on submit', async () => {
      await React.act(async () => {
        render(<DeliveryOrdersPanel />)
      })

      // Click "Buat Surat Jalan Baru"
      const createDOBtn = screen.getByRole('button', { name: /Buat Surat Jalan Baru/i })
      await React.act(async () => {
        fireEvent.click(createDOBtn)
      })

      expect(screen.getByText(/Pilih Pelanggan \(Customer\)/i)).toBeInTheDocument()

      // Click "+ Cepat"
      const quickAddBtn = screen.getByRole('button', { name: /\+ Cepat/i })
      await React.act(async () => {
        fireEvent.click(quickAddBtn)
      })

      expect(screen.getByText(/Tambah Pelanggan Cepat/i)).toBeInTheDocument()

      // Fill form
      const nameInput = screen.getByPlaceholderText(/e\.g\. Toko Berkah Abadi/i)
      await React.act(async () => {
        fireEvent.change(nameInput, { target: { value: 'Toko Berkah Baru' } })
      })

      // Submit quick customer form
      const saveBtn = screen.getByRole('button', { name: /Simpan & Pilih/i })
      await React.act(async () => {
        fireEvent.click(saveBtn)
      })

      // Verify customer API called
      const { api } = require('@/lib/api')
      expect(api.customers.create).toHaveBeenCalledWith(
        expect.objectContaining({ name: 'Toko Berkah Baru' })
      )
    })
  })

  describe('CR-05b: ActivityTimelineDrawer & Actor Audit', () => {
    it('renders ActivityTimelineDrawer with actors, metadata, and close action', async () => {
      const onClose = jest.fn()
      render(
        <ActivityTimelineDrawer
          isOpen={true}
          onClose={onClose}
          title="DO-20261008-0001"
          subtitle="Surat Jalan Keluar"
          entityType="delivery_order"
          entityId="do-123"
          actors={{
            created_by_name: 'Staf Inbound/Outbound',
            created_at: '2026-10-08T08:00:00Z',
            confirmed_by_name: 'Bpk. Supervisor',
            confirmed_at: '2026-10-08T08:30:00Z',
            packed_by_name: 'Petugas Meja Kemas',
            dispatched_by_name: 'Sopir Ekspedisi',
          }}
          metadata={{
            status: 'CONFIRMED',
            reference_type: 'DIRECT_DO',
            product_name: 'Beras Ramos 5kg',
            quantity: 10,
          }}
        />
      )

      expect(screen.getByText(/Riwayat Aktivitas & Jejak Audit \(CR-05b\)/i)).toBeInTheDocument()
      expect(screen.getByText('DO-20261008-0001')).toBeInTheDocument()
      expect(screen.getAllByText('Staf Inbound/Outbound').length).toBeGreaterThanOrEqual(1)
      expect(screen.getAllByText('Bpk. Supervisor').length).toBeGreaterThanOrEqual(1)
      expect(screen.getAllByText('Petugas Meja Kemas').length).toBeGreaterThanOrEqual(1)
      expect(screen.getAllByText('Sopir Ekspedisi').length).toBeGreaterThanOrEqual(1)

      const closeBtn = screen.getByRole('button', { name: /Tutup panel/i })
      fireEvent.click(closeBtn)
      expect(onClose).toHaveBeenCalled()
    })
  })
})
