import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import Sidebar from '@/components/layout/Sidebar'

jest.mock('next/navigation', () => ({
  usePathname: () => '/dashboard',
}))

jest.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({
    user: { email: 'admin@tayooli.com' },
    logout: jest.fn(),
  }),
}))

jest.mock('@/hooks/use-mobile', () => ({
  useIsMobile: () => false,
}))

describe('Sidebar Navigation - 12 Clean Core Modules Hierarchy', () => {
  it('renders all 12 core clean module links', () => {
    render(<Sidebar />)

    // 1. Dashboard (/dashboard)
    expect(screen.getByRole('link', { name: /^Dasbor$/i })).toHaveAttribute('href', '/dashboard')

    // 2. Products (/products)
    expect(screen.getByRole('link', { name: /^Produk$/i })).toHaveAttribute('href', '/products')

    // 3. Barang Masuk & Keluar (/wms/arus-barang) per KO-2a
    expect(screen.getByRole('link', { name: /^Barang Masuk & Keluar$/i })).toHaveAttribute('href', '/wms/arus-barang')

    // 4. Warehouse & Stock (/wms)
    expect(screen.getByRole('link', { name: /^Gudang & Stok$/i })).toHaveAttribute('href', '/wms')

    // 5. Marketplace Omnichannel (/wms/marketplace)
    expect(screen.getByRole('link', { name: /Marketplace/i })).toHaveAttribute('href', '/wms/marketplace')

    // 6. Stock Transfers (/wms/transfers)
    expect(screen.getByRole('link', { name: /^Transfer Stok$/i })).toHaveAttribute('href', '/wms/transfers')

    // 7. Stock Opname (/wms/opname)
    expect(screen.getByRole('link', { name: /^Stok Opname$/i })).toHaveAttribute('href', '/wms/opname')

    // 8. Barang Rusak Scrap (/wms/scrap)
    expect(screen.getByRole('link', { name: /Barang Rusak/i })).toHaveAttribute('href', '/wms/scrap')

    // 9. Barcode Scanner (/wms/scanner)
    expect(screen.getByRole('link', { name: /^Pemindai Barcode$/i })).toHaveAttribute('href', '/wms/scanner')

    // 10. Point of Sale (/pos)
    expect(screen.getByRole('link', { name: /^Kasir \(POS\)$/i })).toHaveAttribute('href', '/pos')

    // 11. Settings (/settings)
    expect(screen.getByRole('link', { name: /^Pengaturan$/i })).toHaveAttribute('href', '/settings')

    // 12. Help & Support (/help)
    expect(screen.getByRole('link', { name: /^Bantuan$/i })).toHaveAttribute('href', '/help')
  })

  it('strictly excludes legacy enterprise modules from sidebar', () => {
    render(<Sidebar />)

    // Enterprise accounting / journal / CoA must not be in sidebar
    expect(screen.queryByRole('link', { name: /Bagan Akun/i })).toBeNull()
    expect(screen.queryByRole('link', { name: /Jurnal Umum/i })).toBeNull()
    expect(screen.queryByRole('link', { name: /Chart of Accounts/i })).toBeNull()
    expect(screen.queryByRole('link', { name: /^Invoices$/i })).toBeNull()
  })
})
