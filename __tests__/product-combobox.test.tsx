/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { ProductCombobox } from '@/components/products/ProductCombobox'
import type { Product } from '@/lib/api'

const sampleProducts: Product[] = [
  { id: 'p-1', tenant_id: 't-1', name: 'Kopi Susu 250ml', sku: 'KOP-250', price: 15000, created_at: '', updated_at: '' },
  { id: 'p-2', tenant_id: 't-1', name: 'Teh Melati', sku: 'TEH-MEL', price: 8000, created_at: '', updated_at: '' },
  { id: 'p-3', tenant_id: 't-1', name: 'Kopi Hitam Tubruk', sku: 'KOP-TUB', price: 10000, created_at: '', updated_at: '' },
]

jest.mock('@/lib/api', () => {
  const actual = jest.requireActual('@/lib/api')
  return {
    ...actual,
    api: {
      ...actual.api,
      products: {
        list: jest.fn(async (params?: { q?: string; per_page?: number }) => {
          const q = (params?.q ?? '').toLowerCase()
          const filtered = sampleProducts.filter(
            (p) => !q || p.name.toLowerCase().includes(q) || p.sku.toLowerCase().includes(q),
          )
          return { data: filtered, total: filtered.length, page: 1, per_page: 20 }
        }),
      },
    },
  }
})

describe('ProductCombobox', () => {
  it('renders placeholder and current selection label', () => {
    const onSelect = jest.fn()
    render(<ProductCombobox value={sampleProducts[0]} onSelect={onSelect} placeholder="Pilih produk..." />)
    expect(screen.getByRole('button')).toHaveTextContent('Kopi Susu 250ml')
  })

  it('opens panel and searches with debounce', async () => {
    const onSelect = jest.fn()
    render(<ProductCombobox value={null} onSelect={onSelect} placeholder="Pilih..." />)

    fireEvent.click(screen.getByRole('button'))
    const input = screen.getByPlaceholderText('Ketik nama atau SKU produk...')
    expect(input).toBeInTheDocument()

    fireEvent.change(input, { target: { value: 'teh' } })

    await waitFor(() => {
      expect(screen.getByText('Teh Melati')).toBeInTheDocument()
    })
    expect(screen.queryByText('Kopi Hitam Tubruk')).not.toBeInTheDocument()

    fireEvent.click(screen.getByText('Teh Melati'))
    expect(onSelect).toHaveBeenCalledWith(expect.objectContaining({ id: 'p-2' }))
  })

  it('allows clearing selection when nullable', () => {
    const onSelect = jest.fn()
    render(<ProductCombobox value={sampleProducts[0]} onSelect={onSelect} allowClear />)
    const clearBtn = screen.getByLabelText('Hapus pilihan produk')
    fireEvent.click(clearBtn)
    expect(onSelect).toHaveBeenCalledWith(null)
  })
})
