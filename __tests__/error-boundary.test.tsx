/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom'
import { ErrorFallback } from '@/components/errors/ErrorFallback'
import AppError from '@/app/(app)/error'

describe('ErrorFallback', () => {
  let spy: jest.SpyInstance
  beforeEach(() => {
    spy = jest.spyOn(console, 'error').mockImplementation(() => {})
  })
  afterEach(() => spy.mockRestore())

  it('shows a friendly message and never the raw error text', () => {
    const err = Object.assign(new Error('pq: relation "users" does not exist'), { digest: 'abc123' })
    render(<ErrorFallback error={err} reset={jest.fn()} />)

    expect(screen.getByRole('alert')).toBeInTheDocument()
    expect(screen.getByText('Terjadi kendala pada halaman ini')).toBeInTheDocument()
    expect(screen.getByText(/Kode referensi: abc123/)).toBeInTheDocument()
    expect(screen.queryByText(/relation "users"/)).not.toBeInTheDocument()
  })

  it('calls reset when "Coba Lagi" is clicked', () => {
    const reset = jest.fn()
    render(<AppError error={new Error('boom')} reset={reset} />)
    fireEvent.click(screen.getByRole('button', { name: /Coba Lagi/ }))
    expect(reset).toHaveBeenCalledTimes(1)
  })

  it('links back to the dashboard and hides the digest line when absent', () => {
    render(<ErrorFallback error={new Error('x')} reset={jest.fn()} />)
    expect(screen.getByRole('link', { name: /Kembali ke Dasbor/ })).toHaveAttribute('href', '/dashboard')
    expect(screen.queryByText(/Kode referensi/)).not.toBeInTheDocument()
  })
})
