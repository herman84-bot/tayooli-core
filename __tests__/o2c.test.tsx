import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import CustomersPage from '@/app/(app)/customers/page';
import SalesOrdersPage from '@/app/(app)/sales-orders/page';
import SalesInvoicesPage from '@/app/(app)/sales-invoices/page';

// Mock the hooks
jest.mock('@/hooks/useCustomers', () => ({
  useCustomers: () => ({
    customers: [
      {
        id: 'a1b2c3d4-0000-0000-0000-000000000001',
        name: 'Acme Corp',
        email: 'contact@acme.com',
        phone: '0812-3456-7890',
        address: 'Jakarta',
      },
    ],
    loading: false,
    error: null,
    createCustomer: jest.fn(),
    refresh: jest.fn(),
  }),
}));

jest.mock('@/hooks/useSalesOrders', () => ({
  useSalesOrders: () => ({
    orders: [
      {
        id: 'e1f2a3b4-0000-0000-0000-000000000002',
        customer_id: 'a1b2c3d4-0000-0000-0000-000000000001',
        customer_name: 'Acme Corp',
        order_number: 'SO-1001',
        total_amount: 1250,
        status: 'PENDING',
        created_at: '2026-07-10T00:00:00Z',
      },
    ],
    loading: false,
    error: null,
    createSalesOrder: jest.fn(),
    refresh: jest.fn(),
  }),
}));

jest.mock('@/hooks/useSalesInvoices', () => ({
  useSalesInvoices: () => ({
    invoices: [
      {
        id: 'c4d5e6f7-0000-0000-0000-000000000003',
        invoiceNumber: 'INV-2001',
        orderId: 'e1f2a3b4-0000-0000-0000-000000000002',
        amount: 1250,
        status: 'PAID',
        dueDate: '2026-07-20',
        createdAt: '2026-07-10T00:00:00Z',
      },
    ],
    loading: false,
    error: null,
    createSalesInvoice: jest.fn(),
    refresh: jest.fn(),
  }),
}));

describe('O2C / Sales Module Pages', () => {
  it('renders Customers Page correctly', () => {
    render(<CustomersPage />);
    expect(screen.getByText('Customers')).toBeInTheDocument();
    expect(screen.getByText('Acme Corp')).toBeInTheDocument();
    expect(screen.getByText('contact@acme.com')).toBeInTheDocument();
    expect(screen.getByText('0812-3456-7890')).toBeInTheDocument();
  });

  it('renders Sales Orders Page correctly', () => {
    render(<SalesOrdersPage />);
    expect(screen.getByText('Sales Orders')).toBeInTheDocument();
    expect(screen.getByText('SO-1001')).toBeInTheDocument();
  });

  it('renders Sales Invoices Page correctly', () => {
    render(<SalesInvoicesPage />);
    expect(screen.getByText('Sales Invoices')).toBeInTheDocument();
    expect(screen.getByText('INV-2001')).toBeInTheDocument();
    expect(screen.getByText('Paid')).toBeInTheDocument();
  });
});
