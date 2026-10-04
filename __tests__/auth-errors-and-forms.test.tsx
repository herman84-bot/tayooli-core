import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { extractApiErrorMessage } from '@/lib/api/errors'
import { useAuthStore } from '@/hooks/useAuth'
import { LoginForm } from '@/components/auth/LoginForm'
import { RegisterForm } from '@/components/auth/RegisterForm'
import { ForgotPasswordForm } from '@/components/auth/ForgotPasswordForm'
import { ResetPasswordForm } from '@/components/auth/ResetPasswordForm'

// Mock next/navigation for ResetPasswordForm
const mockSearchParamsGet = jest.fn()
jest.mock('next/navigation', () => ({
  useSearchParams: () => ({
    get: mockSearchParamsGet,
  }),
}))

describe('extractApiErrorMessage', () => {
  it('extracts error.message from backend Go envelope', async () => {
    const mockRes = {
      json: async () => ({
        error: { code: 'Unauthorized', message: 'invalid credentials' },
      }),
    } as unknown as Response

    const result = await extractApiErrorMessage(mockRes, 'Default fallback')
    expect(result).toBe('invalid credentials')
  })

  it('extracts "email sudah terdaftar" from nested Go error envelope {"error": {"code": "Bad Request", "message": "email sudah terdaftar"}}', async () => {
    const mockRes = {
      json: async () => ({
        error: { code: 'Bad Request', message: 'email sudah terdaftar' },
      }),
    } as unknown as Response

    const result = await extractApiErrorMessage(mockRes, 'Default fallback')
    expect(result).toBe('email sudah terdaftar')
    expect(result).not.toBe('[object Object]')
  })

  it('extracts plain string error', async () => {
    const mockRes = {
      json: async () => ({ error: 'backend unreachable' }),
    } as unknown as Response

    const result = await extractApiErrorMessage(mockRes, 'Default fallback')
    expect(result).toBe('backend unreachable')
  })

  it('extracts top-level message string', async () => {
    const mockRes = {
      json: async () => ({ message: 'user not found' }),
    } as unknown as Response

    const result = await extractApiErrorMessage(mockRes, 'Default fallback')
    expect(result).toBe('user not found')
  })

  it('falls back to default when response body is not JSON (e.g. HTML proxy error)', async () => {
    const mockRes = {
      json: async () => {
        throw new Error('Unexpected token < in JSON at position 0')
      },
    } as unknown as Response

    const result = await extractApiErrorMessage(mockRes, 'Login gagal')
    expect(result).toBe('Login gagal')
  })

  it('falls back to default when body has no error or message properties', async () => {
    const mockRes = {
      json: async () => ({}),
    } as unknown as Response

    const result = await extractApiErrorMessage(mockRes, 'Login gagal')
    expect(result).toBe('Login gagal')
  })
})

describe('useAuthStore login error handling', () => {
  const originalFetch = global.fetch

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('converts 401 with standard "invalid credentials" into friendly Indonesian message', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({
        error: { code: 'Unauthorized', message: 'invalid credentials' },
      }),
    } as unknown as Response)

    await expect(
      useAuthStore.getState().login('user@test.com', 'wrongpassword')
    ).rejects.toThrow('Email atau kata sandi salah')
  })

  it('converts 401 with non-JSON or fallback into "Email atau kata sandi salah"', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => {
        throw new Error('Invalid JSON')
      },
    } as unknown as Response)

    await expect(
      useAuthStore.getState().login('user@test.com', 'wrongpassword')
    ).rejects.toThrow('Email atau kata sandi salah')
  })

  it('preserves specific Indonesian error message on 401 if provided by backend', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({
        error: { code: 'Unauthorized', message: 'Akun Anda dinonaktifkan' },
      }),
    } as unknown as Response)

    await expect(
      useAuthStore.getState().login('user@test.com', 'password123')
    ).rejects.toThrow('Akun Anda dinonaktifkan')
  })

  it('throws extracted error on 500 without converting to [object Object]', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 500,
      json: async () => ({
        error: { code: 'Internal Server Error', message: 'Koneksi database terputus' },
      }),
    } as unknown as Response)

    let errorThrown: any
    try {
      await useAuthStore.getState().login('user@test.com', 'password123')
    } catch (e: any) {
      errorThrown = e
    }
    expect(errorThrown).toBeDefined()
    expect(errorThrown.message).toBe('Koneksi database terputus')
    expect(errorThrown.message).not.toBe('[object Object]')
  })

  it('handles 502 Bad Gateway with HTML error body gracefully with fallback message', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 502,
      json: async () => {
        throw new Error('Unexpected token < in JSON at position 0')
      },
    } as unknown as Response)

    let errorThrown: any
    try {
      await useAuthStore.getState().login('user@test.com', 'password123')
    } catch (e: any) {
      errorThrown = e
    }
    expect(errorThrown).toBeDefined()
    expect(errorThrown.message).toBe('Login gagal')
    expect(errorThrown.message).not.toContain('[object Object]')
  })
})

describe('Auth forms initial SSR/hydration hidden containers', () => {
  beforeEach(() => {
    mockSearchParamsGet.mockReturnValue('test-reset-token')
  })

  it('LoginForm renders error box with style display: none initially', () => {
    render(<LoginForm />)
    const errorBox = screen.getByTestId('login-error')
    expect(errorBox).toHaveStyle({ display: 'none' })
  })

  it('RegisterForm renders error box with style display: none initially', () => {
    render(<RegisterForm />)
    const errorBox = screen.getByTestId('register-error')
    expect(errorBox).toHaveStyle({ display: 'none' })
  })

  it('ForgotPasswordForm renders error and success boxes with style display: none initially', () => {
    render(<ForgotPasswordForm />)
    const errorBox = screen.getByTestId('forgot-error')
    const successBox = screen.getByTestId('forgot-success')
    expect(errorBox).toHaveStyle({ display: 'none' })
    expect(successBox).toHaveStyle({ display: 'none' })
  })

  it('ResetPasswordForm renders error and success boxes with style display: none initially', () => {
    render(<ResetPasswordForm />)
    const errorBox = screen.getByTestId('reset-error')
    const successBox = screen.getByTestId('reset-success')
    expect(errorBox).toHaveStyle({ display: 'none' })
    expect(successBox).toHaveStyle({ display: 'none' })
  })
})

describe('LoginForm UI error rendering and behavior', () => {
  const originalFetch = global.fetch

  afterEach(() => {
    global.fetch = originalFetch
  })

  it('displays clean Indonesian "Email atau kata sandi salah" on 401 invalid credentials in UI and never "[object Object]"', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 401,
      json: async () => ({
        error: { code: 'Unauthorized', message: 'invalid credentials' },
      }),
    } as unknown as Response)

    render(<LoginForm />)
    const emailInput = screen.getByLabelText(/^Email$/i)
    const passwordInput = screen.getByLabelText(/^Kata sandi$/i)
    const submitBtn = screen.getByRole('button', { name: /Masuk/i })

    fireEvent.change(emailInput, { target: { value: 'user@example.com' } })
    fireEvent.change(passwordInput, { target: { value: 'password123' } })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      const errorBox = screen.getByTestId('login-error')
      expect(errorBox).toHaveStyle({ display: 'flex' })
      const errorText = screen.getByTestId('login-error-text')
      expect(errorText.textContent).toBe('Email atau kata sandi salah')
      expect(errorText.textContent).not.toContain('[object Object]')
    })
  })

  it('displays clean fallback error on 502 HTML response in UI', async () => {
    global.fetch = jest.fn().mockResolvedValue({
      ok: false,
      status: 502,
      json: async () => {
        throw new Error('Unexpected token < in JSON at position 0')
      },
    } as unknown as Response)

    render(<LoginForm />)
    const emailInput = screen.getByLabelText(/^Email$/i)
    const passwordInput = screen.getByLabelText(/^Kata sandi$/i)
    const submitBtn = screen.getByRole('button', { name: /Masuk/i })

    fireEvent.change(emailInput, { target: { value: 'user@example.com' } })
    fireEvent.change(passwordInput, { target: { value: 'password123' } })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      const errorBox = screen.getByTestId('login-error')
      expect(errorBox).toHaveStyle({ display: 'flex' })
      const errorText = screen.getByTestId('login-error-text')
      expect(errorText.textContent).toBe('Login gagal')
      expect(errorText.textContent).not.toContain('[object Object]')
    })
  })
})
