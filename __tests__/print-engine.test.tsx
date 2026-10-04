import React from 'react'
import { render, screen, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { PrintDeliveryOrder } from '@/components/wms/PrintDeliveryOrder'
import { PrintSalesInvoice } from '@/components/sales/PrintSalesInvoice'
import type { DeliveryOrder, DeliveryOrderItem } from '@/lib/api'
import type { SalesInvoice } from '@/hooks/useSalesInvoices'

// Mock qrcode module to avoid canvas dependency in jsdom
jest.mock('qrcode', () => ({
  toDataURL: jest.fn().mockResolvedValue('data:image/png;base64,MOCK_QR'),
}))

describe('Requirement #14: Official Document Generation & Print Engine', () => {
  describe('PrintDeliveryOrder Component (Surat Jalan A4)', () => {
    const mockDO: DeliveryOrder = {
      id: 'do-1111-2222-3333-4444',
      tenant_id: 'tenant-alpha-001',
      sales_order_id: 'so-5555-6666-7777-8888',
      warehouse_id: 'wh-cakung-01',
      do_number: 'DO-2026-0909-001',
      status: 'SHIPPED',
      expedition_name: 'JNE Trucking (JTR)',
      tracking_number: 'JTR-88291039',
      driver_name: 'Budi Santoso',
      vehicle_plate: 'B 9182 TAY',
      recipient_name: 'PT Mitra Sukses Mandiri',
      created_at: '2026-09-09T08:30:00Z',
      updated_at: '2026-09-09T09:00:00Z',
    }

    const mockItems: DeliveryOrderItem[] = [
      {
        id: 'doi-001',
        tenant_id: 'tenant-alpha-001',
        delivery_order_id: 'do-1111-2222-3333-4444',
        product_id: 'prod-001',
        quantity: 25,
        location_id: 'loc-001',
        product_name: 'Kertas HVS A4 80gsm PaperOne',
        product_sku: 'PAP-A4-80G',
        location_code: 'RACK-B2-04',
        created_at: '2026-09-09T08:30:00Z',
      },
    ]

    it('renders company KOP and document header correctly', async () => {
      render(
        <PrintDeliveryOrder
          deliveryOrder={mockDO}
          items={mockItems}
          warehouseName="Gudang Distribusi Cakung"
          onClose={jest.fn()}
        />
      )

      await waitFor(() => {
        expect(screen.getByText('SURAT JALAN PENGIRIMAN (DELIVERY ORDER)')).toBeInTheDocument()
      })

      expect(screen.getAllByText('PT TAYOOLI DISTRIBUSI UTAMA').length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('NOMOR: DO-2026-0909-001')).toBeInTheDocument()
      expect(screen.getByText(': JNE Trucking (JTR)')).toBeInTheDocument()
      expect(screen.getByText(': B 9182 TAY')).toBeInTheDocument()
      expect(screen.getByText(': Budi Santoso')).toBeInTheDocument()
      expect(screen.getByText(': PT Mitra Sukses Mandiri')).toBeInTheDocument()
    })

    it('renders item list with SKU, description, rack location, and quantity', async () => {
      render(
        <PrintDeliveryOrder
          deliveryOrder={mockDO}
          items={mockItems}
          onClose={jest.fn()}
        />
      )

      await waitFor(() => {
        expect(screen.getByText('PAP-A4-80G')).toBeInTheDocument()
      })

      expect(screen.getByText('Kertas HVS A4 80gsm PaperOne')).toBeInTheDocument()
      expect(screen.getByText('RACK-B2-04')).toBeInTheDocument()
      expect(screen.getAllByText('25').length).toBeGreaterThanOrEqual(1)
    })

    it('renders the 3 required official signature blocks', async () => {
      render(
        <PrintDeliveryOrder
          deliveryOrder={mockDO}
          items={mockItems}
          onClose={jest.fn()}
        />
      )

      await waitFor(() => {
        expect(screen.getByText('Yang Menyerahkan')).toBeInTheDocument()
      })

      // 1. Yang Menyerahkan (Petugas Gudang)
      expect(screen.getByText('Yang Menyerahkan')).toBeInTheDocument()
      expect(screen.getByText('Petugas Gudang / Fulfillment')).toBeInTheDocument()

      // 2. Yang Membawa (Sopir / Kurir)
      expect(screen.getByText('Yang Membawa')).toBeInTheDocument()
      expect(screen.getByText('Pengemudi / Kurir Ekspedisi')).toBeInTheDocument()

      // 3. Yang Menerima (Customer)
      expect(screen.getByText('Yang Menerima')).toBeInTheDocument()
      expect(screen.getByText('Penerima / Staf Logistik Pemesan')).toBeInTheDocument()
    })

    it('applies .no-print to interactive controls and modal overlays and verifies @media print styles', async () => {
      const { container } = render(
        <PrintDeliveryOrder
          deliveryOrder={mockDO}
          items={mockItems}
          onClose={jest.fn()}
        />
      )

      await waitFor(() => {
        expect(container.querySelector('.no-print')).toBeInTheDocument()
      })

      // Check document sheet max-width and min-height (A4 dimensions: 210mm x 297mm)
      const docSheet = container.querySelector('[style*="297mm"]') as HTMLElement
      expect(docSheet).toBeInTheDocument()
      expect(docSheet.style.minHeight).toBe('297mm')
      expect(docSheet.style.maxWidth).toBe('210mm')

      // Check for style tag in document containing @media print rules
      const styles = Array.from(document.querySelectorAll('style'))
      const printStyle = styles.find((s) => s.textContent?.includes('@media print'))
      expect(printStyle).toBeDefined()
      expect(printStyle?.textContent).toContain('@media print')
      expect(printStyle?.textContent).toContain('.no-print')
      expect(printStyle?.textContent).toMatch(/display:\s*none\s*!important/)
      expect(printStyle?.textContent).toMatch(/size:\s*A4 portrait/)
      expect(printStyle?.textContent).toMatch(/margin:\s*8mm/)
    })
  })

  describe('PrintSalesInvoice Component (Faktur Penjualan A4)', () => {
    const mockInvoice: SalesInvoice = {
      id: 'inv-9999-8888-7777-6666',
      invoiceNumber: 'INV-2026-09-0012',
      orderId: 'so-5555-6666-7777-8888',
      amount: 1250000,
      status: 'PAID',
      dueDate: '2026-09-20T00:00:00Z',
      createdAt: '2026-09-09T10:00:00Z',
    }

    it('renders company KOP and sales invoice titles', async () => {
      render(<PrintSalesInvoice invoice={mockInvoice} onClose={jest.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('FAKTUR PENJUALAN')).toBeInTheDocument()
      })

      expect(screen.getAllByText('PT TAYOOLI DISTRIBUSI UTAMA').length).toBeGreaterThanOrEqual(1)
      expect(screen.getByText('SALES INVOICE')).toBeInTheDocument()
      expect(screen.getByText(': INV-2026-09-0012')).toBeInTheDocument()
    })

    it('renders accurate Indonesian Terbilang for invoice amount', async () => {
      render(<PrintSalesInvoice invoice={mockInvoice} onClose={jest.fn()} />)

      await waitFor(() => {
        expect(
          screen.getByText(/“Satu Juta Dua Ratus Lima Puluh Ribu Rupiah”/)
        ).toBeInTheDocument()
      })
    })

    it('renders tax breakdown (PPN 11% & DPP), bank details, and payment stamp', async () => {
      render(<PrintSalesInvoice invoice={mockInvoice} onClose={jest.fn()} />)

      await waitFor(() => {
        expect(screen.getByText('Dasar Pengenaan Pajak (DPP)')).toBeInTheDocument()
      })

      expect(screen.getByText('PPN (11%)')).toBeInTheDocument()
      expect(screen.getByText('TOTAL TAGIHAN')).toBeInTheDocument()

      // Bank coordinates
      expect(screen.getByText('Bank Central Asia (BCA)')).toBeInTheDocument()
      expect(screen.getByText('883-091-2300')).toBeInTheDocument()
      expect(screen.getByText('Bank Mandiri')).toBeInTheDocument()
      expect(screen.getByText('120-00-9988771-2')).toBeInTheDocument()

      // Stamp watermark for PAID
      expect(screen.getByText('LUNAS / PAID')).toBeInTheDocument()
    })

    it('applies print isolation CSS rules with A4 dimensions and hidden .no-print elements', async () => {
      const { container } = render(
        <PrintSalesInvoice invoice={mockInvoice} onClose={jest.fn()} />
      )

      await waitFor(() => {
        expect(container.querySelector('.no-print')).toBeInTheDocument()
      })

      // Check document sheet max-width and min-height (A4 dimensions: 210mm x 297mm)
      const docSheet = container.querySelector('[style*="297mm"]') as HTMLElement
      expect(docSheet).toBeInTheDocument()
      expect(docSheet.style.minHeight).toBe('297mm')
      expect(docSheet.style.maxWidth).toBe('210mm')

      const styles = Array.from(document.querySelectorAll('style'))
      const printStyle = styles.find((s) => s.textContent?.includes('@media print'))
      expect(printStyle).toBeDefined()
      expect(printStyle?.textContent).toContain('@media print')
      expect(printStyle?.textContent).toContain('.no-print')
      expect(printStyle?.textContent).toMatch(/display:\s*none\s*!important/)
      expect(printStyle?.textContent).toMatch(/size:\s*A4 portrait/)
      expect(printStyle?.textContent).toMatch(/margin:\s*8mm/)
    })
  })
})
