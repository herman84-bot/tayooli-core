/**
 * POS regression tests for QA finding "POS menampilkan data tenant lain".
 *
 * Root cause: app/(app)/pos/page.tsx fell back to a hardcoded DEFAULT_PRODUCTS
 * catalog (and invented stock=50 / price=25000 / fake scanned items) whenever
 * the tenant had no products, so a new tenant saw goods it never created.
 */

import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import POSPage from '@/app/(app)/pos/page'

let mockProducts: Array<Record<string, unknown>> = []

jest.mock('next/link', () => ({
  __esModule: true,
  default: ({ children, href }: { children: React.ReactNode; href: string }) => <a href={href}>{children}</a>,
}))

jest.mock('qrcode', () => ({ toDataURL: jest.fn().mockResolvedValue('') }))

// Forward triggerScan straight to onScan so the real lookup logic runs.
jest.mock('@/hooks/useBarcodeScanner', () => ({
  useBarcodeScanner: ({ onScan }: { onScan: (code: string) => void }) => ({ triggerScan: onScan }),
}))

jest.mock('@/hooks/useProducts', () => ({
  useProducts: () => ({ data: mockProducts }),
}))

jest.mock('@/hooks/usePOS', () => ({
  usePOSOrders: () => ({ data: [], refetch: jest.fn() }),
  usePOSCheckout: () => ({ mutateAsync: jest.fn(), isPending: false }),
}))

jest.mock('@/hooks/useWMSLedger', () => ({
  useWMSStock: () => ({ data: [] }),
}))

const DEMO_NAMES = [/Beras Premium Rojolele/, /Minyak Goreng Sania/, /Gula Pasir Gulaku/, /Deterjen Rinso/]

describe('POS tenant data isolation', () => {
  beforeEach(() => {
    mockProducts = []
  })

  it('empty tenant sees an empty catalog, never hardcoded demo products', () => {
    render(<POSPage />)
    for (const name of DEMO_NAMES) {
      expect(screen.queryByText(name)).not.toBeInTheDocument()
    }
    expect(screen.getByText('Belum ada produk')).toBeInTheDocument()
  })

  it('shows only the tenant own products, with stock 0 when no stock data', () => {
    mockProducts = [
      { id: '11111111-1111-1111-1111-111111111111', name: 'Produk Tenant Saya', sku: 'MY-SKU-1', price: 12000 },
    ]
    render(<POSPage />)
    expect(screen.getByText('Produk Tenant Saya')).toBeInTheDocument()
    expect(screen.getByText('Stok: 0')).toBeInTheDocument()
    expect(screen.queryByText('Belum ada produk')).not.toBeInTheDocument()
    for (const name of DEMO_NAMES) {
      expect(screen.queryByText(name)).not.toBeInTheDocument()
    }
  })

  it('unknown scanned code shows an error instead of inventing a product', () => {
    render(<POSPage />)
    fireEvent.change(screen.getByPlaceholderText(/Scan barcode/i), { target: { value: 'UNKNOWN-12345' } })
    fireEvent.click(screen.getByRole('button', { name: /Masuk Keranjang/i }))

    expect(screen.getByText(/tidak ditemukan di master produk/i)).toBeInTheDocument()
    expect(screen.queryByText(/Barang Barcode \[UNKNOWN-12345\]/)).not.toBeInTheDocument()
    expect(screen.getByText('Keranjang Masih Kosong')).toBeInTheDocument()
  })
})
