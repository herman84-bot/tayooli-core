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

describe('Sidebar Navigation - Invoices vs Sales Invoices terminology', () => {
  it('renders Invoice Vendor link pointing to /dashboard/invoices', () => {
    render(<Sidebar />)
    const purchaseInvoicesLink = screen.getByRole('link', { name: /Invoice Vendor/i })
    expect(purchaseInvoicesLink).toBeInTheDocument()
    expect(purchaseInvoicesLink).toHaveAttribute('href', '/dashboard/invoices')
  })

  it('renders Faktur Penjualan link pointing to /sales-invoices', () => {
    render(<Sidebar />)
    const salesInvoicesLink = screen.getByRole('link', { name: /Faktur Penjualan/i })
    expect(salesInvoicesLink).toBeInTheDocument()
    expect(salesInvoicesLink).toHaveAttribute('href', '/sales-invoices')
  })

  it('does not have ambiguous bare "Invoices" label in sidebar', () => {
    render(<Sidebar />)
    // All invoice links should be qualified as either "Invoice Vendor" or "Faktur Penjualan"
    const exactInvoice = screen.queryByRole('link', { name: /^Invoices$/i })
    expect(exactInvoice).toBeNull()
  })
})
