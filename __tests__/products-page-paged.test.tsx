/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ProductsPage from '@/app/(app)/products/page'
import type { Product } from '@/lib/api'

const sample: Product[] = Array.from({ length: 45 }, (_, i) => ({
  id: `p-${i + 1}`,
  tenant_id: 't-1',
  name: `Produk Alpha ${i + 1}`,
  sku: `SKU-${i + 1}`,
  price: 10000 + i * 500,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
}))

const mockList = jest.fn(async (params?: { page?: number; per_page?: number; q?: string }) => {
  const page = params?.page ?? 1
  const perPage = params?.per_page ?? 20
  const q = (params?.q ?? '').toLowerCase()
  const filtered = sample.filter(
    (p) => !q || p.name.toLowerCase().includes(q) || p.sku.toLowerCase().includes(q),
  )
  const offset = (page - 1) * perPage
  return {
    data: filtered.slice(offset, offset + perPage),
    total: filtered.length,
    page,
    per_page: perPage,
  }
})

jest.mock('@/lib/api', () => {
  const actual = jest.requireActual('@/lib/api')
  return {
    ...actual,
    api: {
      ...actual.api,
      products: {
        list: (p?: any) => mockList(p),
        inventory: jest.fn(async () => []),
      },
    },
  }
})

function renderWithClient() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <ProductsPage />
    </QueryClientProvider>,
  )
}

describe('ProductsPage server pagination & search', () => {
  beforeEach(() => {
    mockList.mockClear()
  })

  it('renders page 1 items and pagination controls', async () => {
    renderWithClient()
    await waitFor(() => {
      expect(screen.getByText('Produk Alpha 1')).toBeInTheDocument()
    })
    // 20 items per page by default
    expect(screen.getByText('Produk Alpha 20')).toBeInTheDocument()
    expect(screen.queryByText('Produk Alpha 21')).not.toBeInTheDocument()

    // Footer shows range & total
    expect(screen.getByText(/Menampilkan 1–20 dari 45 produk/)).toBeInTheDocument()
  })

  it('navigates to page 2 via pagination button', async () => {
    renderWithClient()
    await waitFor(() => {
      expect(screen.getByText('Produk Alpha 1')).toBeInTheDocument()
    })

    const page2Btn = screen.getByRole('button', { name: '2' })
    fireEvent.click(page2Btn)

    await waitFor(() => {
      expect(mockList).toHaveBeenCalledWith(expect.objectContaining({ page: 2 }))
    })

    await waitFor(() => {
      expect(screen.getByText('Produk Alpha 21')).toBeInTheDocument()
    })
  })

  it('debounces search input before firing server query', async () => {
    renderWithClient()
    await waitFor(() => {
      expect(screen.getByText('Produk Alpha 1')).toBeInTheDocument()
    })

    const input = screen.getByPlaceholderText(/Cari berdasarkan nama produk atau SKU/)
    fireEvent.change(input, { target: { value: 'SKU-42' } })

    // Waiting for 300ms debounce
    await waitFor(
      () => {
        expect(mockList).toHaveBeenCalledWith(expect.objectContaining({ q: 'SKU-42', page: 1 }))
      },
      { timeout: 1000 },
    )
  })
})
