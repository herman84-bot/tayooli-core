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
    expect(screen.getByRole('link', { name: /^Dashboard$/i })).toHaveAttribute('href', '/dashboard')

    // 2. Products (/products)
    expect(screen.getByRole('link', { name: /^Products$/i })).toHaveAttribute('href', '/products')

    // 3. Warehouse & Stock (/wms)
    expect(screen.getByRole('link', { name: /^Warehouse & Stock$/i })).toHaveAttribute('href', '/wms')

    // 4. Surat Jalan DO (/wms/delivery-orders)
    expect(screen.getByRole('link', { name: /Surat Jalan/i })).toHaveAttribute('href', '/wms/delivery-orders')

    // 5. Marketplace Omnichannel (/wms/marketplace)
    expect(screen.getByRole('link', { name: /Marketplace/i })).toHaveAttribute('href', '/wms/marketplace')

    // 6. Stock Transfers (/wms/transfers)
    expect(screen.getByRole('link', { name: /Stock Transfers/i })).toHaveAttribute('href', '/wms/transfers')

    // 7. Stock Opname (/wms/opname)
    expect(screen.getByRole('link', { name: /Stock Opname/i })).toHaveAttribute('href', '/wms/opname')

    // 8. Barang Rusak Scrap (/wms/scrap)
    expect(screen.getByRole('link', { name: /Barang Rusak/i })).toHaveAttribute('href', '/wms/scrap')

    // 9. Barcode Scanner (/wms/scanner)
    expect(screen.getByRole('link', { name: /Barcode Scanner/i })).toHaveAttribute('href', '/wms/scanner')

    // 10. Point of Sale (/pos)
    expect(screen.getByRole('link', { name: /Point of Sale/i })).toHaveAttribute('href', '/pos')

    // 11. Settings (/settings)
    expect(screen.getByRole('link', { name: /^Settings$/i })).toHaveAttribute('href', '/settings')

    // 12. Help & Support (/help)
    expect(screen.getByRole('link', { name: /Help & Support/i })).toHaveAttribute('href', '/help')
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
