import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom'
import { PrintPickingList } from '@/components/wms/PrintPickingList'
import { PrintThermalAWB } from '@/components/wms/PrintThermalAWB'
import { WaveReleaseModal } from '@/components/wms/WaveReleaseModal'
import { PickingTaskDetail, DeliveryOrder, DeliveryOrderItem } from '@/lib/api'

// Mock api
jest.mock('@/lib/api', () => ({
  ...jest.requireActual('@/lib/api'),
  api: {
    wms: {
      pickWaves: {
        list: jest.fn().mockResolvedValue({ data: [] }),
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
        scanPackItem: jest.fn().mockResolvedValue({
          data: { item_id: 'doi-1', packed_qty: '1', product_name: 'Beras 5kg', product_sku: 'BRS-01', item_completed: false },
        }),
        completePack: jest.fn().mockResolvedValue({ data: { id: 'do-1', status: 'PACKED' } }),
      },
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
})
