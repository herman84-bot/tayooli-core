import React from 'react'
import { render, screen } from '@testing-library/react'
import { PrintDeliveryOrder } from '@/components/wms/PrintDeliveryOrder'
import { PrintSalesInvoice } from '@/components/sales/PrintSalesInvoice'
import { PrintBAK } from '@/components/wms/PrintBAK'
import { PrintShippingManifest } from '@/components/wms/PrintShippingManifest'
import type { DeliveryOrder, DeliveryOrderItem, QCInspection, QCInspectionItem, ShippingManifestDetail } from '@/lib/api'
import type { SalesInvoice } from '@/hooks/useSalesInvoices'

const mockDO: DeliveryOrder = {
  id: 'do-12345678-uuid',
  do_number: 'DO-2026-0001',
  order_type: 'SALES_ORDER',
  status: 'DRAFT',
  recipient_name: 'PT Mitra Sejahtera',
  recipient_address: 'Jl. Pemuda No. 10, Surabaya',
  recipient_phone: '081234567890',
  recipient_city: 'Surabaya',
  created_at: '2026-10-10T08:00:00Z',
  updated_at: '2026-10-10T08:00:00Z',
  total_items: 2,
  total_qty: 10,
}

const mockDOItems: DeliveryOrderItem[] = [
  {
    id: 'doi-1',
    delivery_order_id: 'do-12345678-uuid',
    product_id: 'prod-1',
    product_name: 'Barang Logistik A',
    sku: 'LOG-A',
    quantity: 10,
    unit: 'PCS',
    created_at: '2026-10-10T08:00:00Z',
  },
]

const mockInvoice: SalesInvoice = {
  id: 'inv-123',
  invoiceNumber: 'INV/2026/001',
  orderId: 'so-123',
  status: 'ISSUED',
  subtotal: 1000000,
  taxRate: 11,
  taxAmount: 110000,
  discountAmount: 0,
  totalAmount: 1110000,
  paidAmount: 0,
  outstandingAmount: 1110000,
  dueDate: '2026-11-10',
  createdAt: '2026-10-10T08:00:00Z',
  updatedAt: '2026-10-10T08:00:00Z',
}

const mockQCInspection: QCInspection = {
  id: 'qc-1',
  receipt_id: 'rec-1',
  receipt_number: 'GR-2026-001',
  warehouse_id: 'WH-01',
  inspector_id: 'user-1',
  inspector_name: 'Ahmad QC',
  status: 'COMPLETED',
  total_inspected: 100,
  total_passed: 95,
  total_rejected: 5,
  total_quarantine: 0,
  bak_number: 'BAK-2026-001',
  driver_name: 'Pak Budi',
  driver_signed: true,
  supplier_name: 'PT Vendor Jaya',
  created_at: '2026-10-10T08:00:00Z',
  updated_at: '2026-10-10T08:00:00Z',
}

const mockQCItems: QCInspectionItem[] = [
  {
    id: 'qci-1',
    inspection_id: 'qc-1',
    product_id: 'prod-1',
    product_name: 'Barang Rusak',
    sku: 'BRG-01',
    inspected_qty: 100,
    passed_qty: 95,
    damaged_qty: 5,
    damage_reason: 'Kardus penyok',
    action: 'REJECT',
    created_at: '2026-10-10T08:00:00Z',
  },
]

const mockManifestDetail: ShippingManifestDetail = {
  manifest: {
    id: 'man-1',
    manifest_number: 'MAN-2026-0001',
    warehouse_id: 'wh-1',
    warehouse_name: 'Gudang Cakung',
    expedition_name: 'JNE Express',
    vehicle_plate: 'B 1234 CD',
    driver_name: 'Joko',
    status: 'IN_TRANSIT',
    total_packages: 10,
    created_at: '2026-10-10T08:00:00Z',
    updated_at: '2026-10-10T08:00:00Z',
  },
  items: [],
}

describe('Dynamic Company Profile on Printable Documents', () => {
  test('PrintDeliveryOrder displays custom company name, address, and contact', () => {
    render(
      <PrintDeliveryOrder
        deliveryOrder={mockDO}
        items={mockDOItems}
        companyProfile={{
          name: 'PT Mega Surya Logistik',
          division: 'Divisi Pergudangan Modern',
          address: 'Jl. Rungkut Industri III No. 8, Surabaya',
          phone: '(031) 843-9999',
          email: 'logistik@megasurya.com',
          website: 'www.megasurya.com',
        }}
        onClose={jest.fn()}
      />
    )

    expect(screen.getByText('PT Mega Surya Logistik')).toBeInTheDocument()
    expect(screen.getByText('Divisi Pergudangan Modern')).toBeInTheDocument()
    expect(screen.getByText('Jl. Rungkut Industri III No. 8, Surabaya')).toBeInTheDocument()
    expect(
      screen.getByText((content) => content.includes('(031) 843-9999') && content.includes('logistik@megasurya.com'))
    ).toBeInTheDocument()
    // Initial avatar 'M'
    expect(screen.getByText('M')).toBeInTheDocument()
  })

  test('PrintDeliveryOrder falls back safely to default company branding when profile is empty', () => {
    render(<PrintDeliveryOrder deliveryOrder={mockDO} items={mockDOItems} onClose={jest.fn()} />)

    expect(screen.getByText('PT TAYOOLI DISTRIBUSI UTAMA')).toBeInTheDocument()
    expect(screen.getByText('T')).toBeInTheDocument()
  })

  test('PrintSalesInvoice displays custom company name and NPWP', () => {
    render(
      <PrintSalesInvoice
        invoice={mockInvoice}
        companyProfile={{
          name: 'PT Sumber Berkah Sejahtera',
          division: 'Divisi Niaga & Distribusi',
          address: 'Gedung Bursa Efek Tower 2, Jakarta',
          tax_id: '01.999.888.7-001.000',
          phone: '(021) 515-8888',
          email: 'finance@sumberberkah.com',
        }}
        onClose={jest.fn()}
      />
    )

    // Kop header & manager signature block
    const companyHeadings = screen.getAllByText('PT Sumber Berkah Sejahtera')
    expect(companyHeadings.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('Divisi Niaga & Distribusi')).toBeInTheDocument()
    expect(screen.getByText((content) => content.includes('01.999.888.7-001.000'))).toBeInTheDocument()
    expect(screen.getByText('S')).toBeInTheDocument()
  })

  test('PrintBAK displays custom company name and division', () => {
    render(
      <PrintBAK
        inspection={mockQCInspection}
        items={mockQCItems}
        companyProfile={{
          name: 'PT Karya Logistik Mandiri',
          division: 'Divisi Quality Control & Retur',
        }}
        onClose={jest.fn()}
      />
    )

    expect(screen.getByText('PT Karya Logistik Mandiri')).toBeInTheDocument()
    expect(screen.getByText('Divisi Quality Control & Retur')).toBeInTheDocument()
  })

  test('PrintShippingManifest displays custom company name and division', () => {
    render(
      <PrintShippingManifest
        manifestDetail={mockManifestDetail}
        companyProfile={{
          name: 'PT Express Distribusi Cepat',
          division: 'Divisi Moda Transportasi & Dispatch',
        }}
        onClose={jest.fn()}
      />
    )

    expect(screen.getByText('PT Express Distribusi Cepat')).toBeInTheDocument()
    expect(screen.getByText('Divisi Moda Transportasi & Dispatch')).toBeInTheDocument()
  })
})
