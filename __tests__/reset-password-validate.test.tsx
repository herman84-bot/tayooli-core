import React from 'react'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import '@testing-library/jest-dom'
import { ResetPasswordForm } from '@/components/auth/ResetPasswordForm'

const mockGet = jest.fn()
jest.mock('next/navigation', () => ({
  useSearchParams: () => ({ get: mockGet }),
}))

function respond(status: number, body: unknown) {
  return { ok: status >= 200 && status < 300, status, json: async () => body } as unknown as Response
}

describe('ResetPasswordForm token check on page open', () => {
  const originalFetch = global.fetch
  afterEach(() => {
    global.fetch = originalFetch
    mockGet.mockReset()
  })

  it('checks the token with GET /api/v1/auth/reset-password/validate when the page opens', async () => {
    mockGet.mockReturnValue('abc 123')
    global.fetch = jest.fn().mockResolvedValue(respond(200, { valid: true }))
    render(<ResetPasswordForm />)
    await waitFor(() =>
      expect(global.fetch).toHaveBeenCalledWith(
        '/api/v1/auth/reset-password/validate?token=abc%20123',
        expect.objectContaining({ cache: 'no-store' }),
      ),
    )
    expect(screen.getByLabelText('Kata Sandi Baru')).toBeInTheDocument()
  })

  it('shows the expired message and a link to request a new one, hiding the form', async () => {
    mockGet.mockReturnValue('old')
    global.fetch = jest.fn().mockResolvedValue(
      respond(400, { valid: false, reason: 'expired', error: 'Tautan reset password sudah kedaluwarsa (berlaku 1 jam). Minta tautan baru.' }),
    )
    render(<ResetPasswordForm />)
    expect(await screen.findByText(/sudah kedaluwarsa/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Kata Sandi Baru')).not.toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Minta tautan baru/i })).toHaveAttribute('href', '/forgot-password')
  })

  it('shows the already-used message for an invalid token', async () => {
    mockGet.mockReturnValue('used')
    global.fetch = jest.fn().mockResolvedValue(
      respond(400, { valid: false, reason: 'invalid', error: 'Tautan reset password tidak valid atau sudah pernah dipakai. Minta tautan baru.' }),
    )
    render(<ResetPasswordForm />)
    expect(await screen.findByText(/sudah pernah dipakai/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Kata Sandi Baru')).not.toBeInTheDocument()
  })

  it('renders a hostile server message as plain text (no HTML injection)', async () => {
    mockGet.mockReturnValue('x')
    const payload = '<img src="x" onerror="alert(1)"><script>alert(1)</script>'
    global.fetch = jest.fn().mockResolvedValue(respond(400, { valid: false, reason: 'invalid', error: payload }))
    render(<ResetPasswordForm />)
    expect(await screen.findByText(payload)).toBeInTheDocument()
    expect(document.querySelector('script')).toBeNull()
    expect(document.querySelector('img[src="x"]')).toBeNull()
  })

  it('keeps the form usable when the check itself fails (network/server error)', async () => {
    mockGet.mockReturnValue('tok')
    global.fetch = jest.fn().mockRejectedValue(new Error('offline'))
    render(<ResetPasswordForm />)
    await waitFor(() => expect(global.fetch).toHaveBeenCalled())
    expect(screen.getByLabelText('Kata Sandi Baru')).toBeInTheDocument()
  })

  it('invite link: activation wording, success points to login, dead link tells user to ask the admin', async () => {
    mockGet.mockImplementation((k: string) => (k === 'token' ? 'inv' : k === 'invite' ? '1' : null))
    global.fetch = jest
      .fn()
      .mockResolvedValueOnce(respond(200, { valid: true }))
      .mockResolvedValueOnce(respond(200, { message: 'ok' }))
    render(<ResetPasswordForm />)
    const submit = await screen.findByRole('button', { name: /Aktifkan Akun/i })
    fireEvent.change(screen.getByLabelText('Kata Sandi Baru'), { target: { value: 'rahasia123' } })
    fireEvent.change(screen.getByLabelText('Konfirmasi Kata Sandi'), { target: { value: 'rahasia123' } })
    fireEvent.click(submit)
    await waitFor(() =>
      expect(screen.getByTestId('reset-success-text').textContent).toMatch(/Akun Anda aktif/),
    )
    expect(screen.getByRole('link', { name: /Masuk sekarang/i })).toHaveAttribute('href', '/login')
  })

  it('expired invite link asks the employee to contact the admin, not to use forgot-password', async () => {
    mockGet.mockImplementation((k: string) => (k === 'token' ? 'old' : k === 'invite' ? '1' : null))
    global.fetch = jest.fn().mockResolvedValue(respond(400, { valid: false, reason: 'expired', error: 'x' }))
    render(<ResetPasswordForm />)
    expect(await screen.findByText(/Undangan ini sudah kedaluwarsa/)).toBeInTheDocument()
    expect(screen.getByText(/Kirim Ulang Undangan/)).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: /Minta tautan baru/i })).not.toBeInTheDocument()
  })

  it('does not call the API and shows a request-new-link message when the token is missing', () => {
    mockGet.mockReturnValue(null)
    global.fetch = jest.fn()
    render(<ResetPasswordForm />)
    expect(global.fetch).not.toHaveBeenCalled()
    expect(screen.getByText(/tidak lengkap/)).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Minta tautan baru/i })).toHaveAttribute('href', '/forgot-password')
  })
})
