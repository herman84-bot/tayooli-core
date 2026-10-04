import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import { extractApiErrorMessage } from '@/lib/api/errors'
import { useAuthStore } from '@/hooks/useAuth'
import { LoginForm } from '@/components/auth/LoginForm'
import { RegisterForm } from '@/components/auth/RegisterForm'
import { ForgotPasswordForm } from '@/components/auth/ForgotPasswordForm'
import { ResetPasswordForm } from '@/components/auth/ResetPasswordForm'

// Mock next/navigation
const mockSearchParamsGet = jest.fn()
jest.mock('next/navigation', () => ({
  useSearchParams: () => ({
    get: mockSearchParamsGet,
  }),
}))

describe('RED TEAM SECURITY AUDIT: Authentication Suite', () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    mockSearchParamsGet.mockReturnValue('valid-audit-token')
  })

  afterEach(() => {
    global.fetch = originalFetch
    jest.restoreAllMocks()
  })

  // =========================================================================
  // 1. Information Disclosure & User Enumeration
  // =========================================================================
  describe('1. Information Disclosure & Enumeration', () => {
    it('prevents user enumeration: 401 with standard "invalid credentials" always resolves to generic "Email atau kata sandi salah"', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({
          error: { code: 'Unauthorized', message: 'invalid credentials' },
        }),
      } as unknown as Response)

      await expect(
        useAuthStore.getState().login('attacker_target@company.com', 'wrongPassword123')
      ).rejects.toThrow('Email atau kata sandi salah')
    })

    it('prevents user enumeration: 401 with "unauthorized" or default fallback maps to generic "Email atau kata sandi salah"', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => ({
          error: 'Unauthorized',
        }),
      } as unknown as Response)

      await expect(
        useAuthStore.getState().login('attacker_target@company.com', 'wrongPassword123')
      ).rejects.toThrow('Email atau kata sandi salah')
    })

    it('prevents user enumeration: 401 with empty or non-JSON body maps to generic "Email atau kata sandi salah"', async () => {
      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 401,
        json: async () => {
          throw new Error('Not JSON')
        },
      } as unknown as Response)

      await expect(
        useAuthStore.getState().login('nonexistent@company.com', 'wrongPassword123')
      ).rejects.toThrow('Email atau kata sandi salah')
    })

    it('does not leak backend HTML stack traces or server crash logs in login()', async () => {
      const internalStackTraceHtml = `
        <html><body>
        <h1>500 Internal Server Error</h1>
        <pre>panic: runtime error: invalid memory address or nil pointer dereference
        [signal SIGSEGV: code=0x1 addr=0x0 pc=0x8321a4]
        goroutine 42 [running]:
        github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/auth.(*Usecase).LoginByEmail(...)
        /opt/tayooli/backend/go-core/internal/usecase/auth/auth_usecase.go:88 +0x140
        </pre>
        </body></html>
      `

      global.fetch = jest.fn().mockResolvedValue({
        ok: false,
        status: 500,
        json: async () => {
          throw new SyntaxError('Unexpected token < in JSON at position 0')
        },
        text: async () => internalStackTraceHtml,
      } as unknown as Response)

      let caughtError: Error | null = null
      try {
        await useAuthStore.getState().login('user@company.com', 'password123')
      } catch (err: any) {
        caughtError = err
      }

      expect(caughtError).not.toBeNull()
      // Verifies that internal stack trace HTML was not leaked into the thrown error
      expect(caughtError?.message).toBe('Login gagal')
      expect(caughtError?.message).not.toContain('goroutine')
      expect(caughtError?.message).not.toContain('SIGSEGV')
      expect(caughtError?.message).not.toContain('/opt/tayooli')
    })
  })

  // =========================================================================
  // 2. Cross-Site Scripting (XSS) via Error Messages
  // =========================================================================
  describe('2. Cross-Site Scripting (XSS) via Error Messages', () => {
    const xssPayloads = [
      "<script>alert('XSS_EXEC')</script>",
      "<img src=x onerror=\"alert('XSS_IMG')\" />",
      '"><svg/onload=alert("XSS_SVG")>',
      '<iframe src="javascript:alert(1)"></iframe>',
      '<body onload=alert(1)>',
    ]

    xssPayloads.forEach((payload, idx) => {
      it(`[LoginForm] safely neutralizes XSS payload #${idx + 1} using textContent (no HTML execution)`, async () => {
        global.fetch = jest.fn().mockResolvedValue({
          ok: false,
          status: 400,
          json: async () => ({
            error: { code: 'Bad Request', message: payload },
          }),
        } as unknown as Response)

        render(<LoginForm />)
        const emailInput = screen.getByLabelText(/^Email$/i)
        const passwordInput = screen.getByLabelText(/^Kata sandi$/i)
        const submitBtn = screen.getByRole('button', { name: /Masuk/i })

        fireEvent.change(emailInput, { target: { value: 'test@example.com' } })
        fireEvent.change(passwordInput, { target: { value: 'password123' } })
        fireEvent.click(submitBtn)

        await waitFor(() => {
          const errorTextSpan = screen.getByTestId('login-error-text')
          // Rendered strictly as textContent, exact string match
          expect(errorTextSpan.textContent).toBe(payload)
          // Ensure no executable elements were injected into the DOM
          expect(document.querySelector('script')).toBeNull()
          expect(document.querySelector('iframe')).toBeNull()
          expect(document.querySelector('img[src="x"]')).toBeNull()
        })
      })

      it(`[RegisterForm] safely neutralizes XSS payload #${idx + 1} using textContent`, async () => {
        global.fetch = jest.fn().mockResolvedValue({
          ok: false,
          status: 400,
          json: async () => ({
            error: { code: 'Bad Request', message: payload },
          }),
        } as unknown as Response)

        render(<RegisterForm />)
        const fullNameInput = screen.getByLabelText(/^Nama Lengkap$/i)
        const emailInput = screen.getByLabelText(/^Email$/i)
        const passwordInput = screen.getByLabelText(/^Kata Sandi$/i)
        const confirmPasswordInput = screen.getByLabelText(/^Konfirmasi Kata Sandi$/i)
        const submitBtn = screen.getByRole('button', { name: /Buat Akun/i })

        fireEvent.change(fullNameInput, { target: { value: 'Test User' } })
        fireEvent.change(emailInput, { target: { value: 'test@example.com' } })
        fireEvent.change(passwordInput, { target: { value: 'password123' } })
        fireEvent.change(confirmPasswordInput, { target: { value: 'password123' } })
        fireEvent.click(submitBtn)

        await waitFor(() => {
          const errorTextSpan = screen.getByTestId('register-error-text')
          expect(errorTextSpan.textContent).toBe(payload)
          expect(document.querySelector('script')).toBeNull()
          expect(document.querySelector('iframe')).toBeNull()
          expect(document.querySelector('img[src="x"]')).toBeNull()
        })
      })

      it(`[ForgotPasswordForm] safely neutralizes XSS payload #${idx + 1} using textContent`, async () => {
        global.fetch = jest.fn().mockResolvedValue({
          ok: false,
          status: 400,
          json: async () => ({
            error: { code: 'Bad Request', message: payload },
          }),
        } as unknown as Response)

        render(<ForgotPasswordForm />)
        const emailInput = screen.getByLabelText(/^Email$/i)
        const submitBtn = screen.getByRole('button', { name: /Kirim Link Reset/i })

        fireEvent.change(emailInput, { target: { value: 'test@example.com' } })
        fireEvent.click(submitBtn)

        await waitFor(() => {
          const errorTextSpan = screen.getByTestId('forgot-error-text')
          expect(errorTextSpan.textContent).toBe(payload)
          expect(document.querySelector('script')).toBeNull()
          expect(document.querySelector('iframe')).toBeNull()
          expect(document.querySelector('img[src="x"]')).toBeNull()
        })
      })

      it(`[ResetPasswordForm] safely neutralizes XSS payload #${idx + 1} using textContent`, async () => {
        global.fetch = jest.fn().mockResolvedValue({
          ok: false,
          status: 400,
          json: async () => ({
            error: { code: 'Bad Request', message: payload },
          }),
        } as unknown as Response)

        render(<ResetPasswordForm />)
        const passwordInput = screen.getByLabelText(/^Kata Sandi Baru$/i)
        const confirmPasswordInput = screen.getByLabelText(/^Konfirmasi Kata Sandi$/i)
        const submitBtn = screen.getByRole('button', { name: /Reset Password/i })

        fireEvent.change(passwordInput, { target: { value: 'password123' } })
        fireEvent.change(confirmPasswordInput, { target: { value: 'password123' } })
        fireEvent.click(submitBtn)

        await waitFor(() => {
          const errorTextSpan = screen.getByTestId('reset-error-text')
          expect(errorTextSpan.textContent).toBe(payload)
          expect(document.querySelector('script')).toBeNull()
          expect(document.querySelector('iframe')).toBeNull()
          expect(document.querySelector('img[src="x"]')).toBeNull()
        })
      })
    })
  })

  // =========================================================================
  // 3. Credential & Token Leaks
  // =========================================================================
  describe('3. Credential & Token Leaks', () => {
    it('ensures passwords and secrets are never logged to console.* methods during login', async () => {
      const spyLog = jest.spyOn(console, 'log').mockImplementation(() => {})
      const spyWarn = jest.spyOn(console, 'warn').mockImplementation(() => {})
      const spyError = jest.spyOn(console, 'error').mockImplementation(() => {})
      const spyInfo = jest.spyOn(console, 'info').mockImplementation(() => {})

      const sensitivePassword = 'SuperSecret_Adversarial_P@ssw0rd!_123'
      const sensitiveToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.sensitive_payload'

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          user: {
            id: '550e8400-e29b-41d4-a716-446655440001',
            email: 'admin@company.com',
            role: 'admin',
            tenantId: 'tenant-123',
          },
          token: sensitiveToken,
        }),
      } as unknown as Response)

      await useAuthStore.getState().login('admin@company.com', sensitivePassword)

      // Verify no console call received the password or token
      const allCalls = [
        ...spyLog.mock.calls,
        ...spyWarn.mock.calls,
        ...spyError.mock.calls,
        ...spyInfo.mock.calls,
      ].flat()

      for (const callArg of allCalls) {
        const stringified = typeof callArg === 'string' ? callArg : JSON.stringify(callArg)
        expect(stringified).not.toContain(sensitivePassword)
        expect(stringified).not.toContain(sensitiveToken)
      }
    })

    it('enforces credentials: "include" across all useAuth network requests (login, logout, hydrate)', async () => {
      const mockFetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          user: { id: '1', email: 'test@example.com', role: 'admin', tenantId: 't1' },
        }),
      } as unknown as Response)

      global.fetch = mockFetch

      // 1. login()
      await useAuthStore.getState().login('test@example.com', 'password123')
      expect(mockFetch).toHaveBeenLastCalledWith(
        '/api/v1/auth/login',
        expect.objectContaining({
          credentials: 'include',
        })
      )

      // 2. hydrate()
      await useAuthStore.getState().hydrate()
      expect(mockFetch).toHaveBeenLastCalledWith(
        '/api/v1/auth/me',
        expect.objectContaining({
          credentials: 'include',
        })
      )

      // 3. logout()
      await useAuthStore.getState().logout()
      expect(mockFetch).toHaveBeenLastCalledWith(
        '/api/v1/auth/logout',
        expect.objectContaining({
          credentials: 'include',
        })
      )
    })

    it('does not store passwords or raw session tokens in localStorage (partialize whitelist)', async () => {
      // Clear localStorage
      localStorage.clear()

      global.fetch = jest.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          user: {
            id: 'user-001',
            email: 'test@company.com',
            role: 'accountant',
            tenant_id: 'tenant-999',
          },
          token: 'sensitive-jwt-token-should-not-be-in-localstorage',
        }),
      } as unknown as Response)

      await useAuthStore.getState().login('test@company.com', 'verySecretPassword123')

      const storedAuthRaw = localStorage.getItem('tayooli-auth')
      expect(storedAuthRaw).not.toBeNull()
      const parsed = JSON.parse(storedAuthRaw!)

      // Assert password is NOT stored
      expect(JSON.stringify(parsed)).not.toContain('verySecretPassword123')
      // Assert raw backend JWT token is NOT persisted in tayooli-auth zustand storage
      expect(JSON.stringify(parsed)).not.toContain('sensitive-jwt-token-should-not-be-in-localstorage')
      // Only state.user and state.isAuthenticated are persisted
      expect(Object.keys(parsed.state)).toEqual(['user', 'isAuthenticated'])
    })
  })

  // =========================================================================
  // 4. Prototype Pollution & Type Injection
  // =========================================================================
  describe('4. Prototype Pollution & Type Injection', () => {
    it('safely handles malicious __proto__ object injection without polluting Object.prototype', async () => {
      // Simulate malicious JSON response payload containing __proto__
      const maliciousJson = JSON.parse('{"__proto__": {"polluted_prop": "CRITICAL_POLLUTION"}}')

      const mockRes = {
        json: async () => maliciousJson,
      } as unknown as Response

      const result = await extractApiErrorMessage(mockRes, 'Default Fallback')

      // Assert fallback was returned
      expect(result).toBe('Default Fallback')
      // Assert Object.prototype was NOT polluted
      expect((Object.prototype as any).polluted_prop).toBeUndefined()
      expect(({} as any).polluted_prop).toBeUndefined()
    })

    it('safely handles deeply nested __proto__ injection inside error object', async () => {
      const maliciousJson = JSON.parse(
        '{"error": {"__proto__": {"isAdmin": true, "message": "polluted_msg"}}}'
      )

      const mockRes = {
        json: async () => maliciousJson,
      } as unknown as Response

      const result = await extractApiErrorMessage(mockRes, 'Fallback')

      // Prototype remains completely clean
      expect((Object.prototype as any).isAdmin).toBeUndefined()
      expect(({} as any).isAdmin).toBeUndefined()
    })

    it('safely handles constructor and prototype poison attacks', async () => {
      const maliciousPayload = {
        constructor: {
          prototype: {
            exploit: 'active',
          },
        },
        error: {
          constructor: {
            prototype: {
              subExploit: 'active',
            },
          },
        },
      }

      const mockRes = {
        json: async () => maliciousPayload,
      } as unknown as Response

      const result = await extractApiErrorMessage(mockRes, 'Safe Fallback')

      expect(result).toBe('Safe Fallback')
      expect((Object.prototype as any).exploit).toBeUndefined()
      expect((Object.prototype as any).subExploit).toBeUndefined()
    })

    it('safely handles corrupted/unexpected types (null, numbers, boolean, arrays, circular structures)', async () => {
      const corruptedResponses = [
        null,
        12345,
        true,
        false,
        ['unexpected', 'array'],
        { error: null },
        { error: 123 },
        { error: [] },
        { error: { message: null } },
        { error: { message: {} } },
        { error: { message: [] } },
        { message: 999 },
      ]

      for (const item of corruptedResponses) {
        const mockRes = {
          json: async () => item,
        } as unknown as Response

        // Must never throw and always gracefully return fallback
        const result = await extractApiErrorMessage(mockRes, 'Safe Fallback')
        expect(typeof result).toBe('string')
        expect(result).toBe('Safe Fallback')
      }
    })
  })
})
