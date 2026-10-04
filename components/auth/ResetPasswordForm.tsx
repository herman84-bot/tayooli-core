'use client'

import { useLayoutEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, AlertCircle, CheckCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useSearchParams } from 'next/navigation'
import { extractApiErrorMessage } from '@/lib/api/errors'

const resetPasswordSchema = z.object({
  password: z.string().min(8, 'Kata sandi minimal 8 karakter'),
  confirmPassword: z.string().min(8, 'Konfirmasi kata sandi minimal 8 karakter'),
}).refine((data) => data.password === data.confirmPassword, {
  message: 'Kata sandi tidak cocok',
  path: ['confirmPassword'],
})
type ResetPasswordFormData = z.infer<typeof resetPasswordSchema>

export function ResetPasswordForm() {
  const searchParams = useSearchParams()
  const token = searchParams.get('token')

  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const errorBoxRef = useRef<HTMLDivElement>(null)
  const successBoxRef = useRef<HTMLDivElement>(null)
  const [showPassword, setShowPassword] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<ResetPasswordFormData>({
    resolver: zodResolver(resetPasswordSchema),
  })

  useLayoutEffect(() => {
    if (errorBoxRef.current) {
      errorBoxRef.current.style.display = error ? 'flex' : 'none'
      const text = errorBoxRef.current.querySelector('[data-testid="reset-error-text"]')
      if (text) text.textContent = error ?? ''
    }
  }, [error])

  useLayoutEffect(() => {
    if (successBoxRef.current) {
      successBoxRef.current.style.display = success ? 'flex' : 'none'
      const text = successBoxRef.current.querySelector('[data-testid="reset-success-text"]')
      if (text) text.textContent = success ?? ''
    }
  }, [success])

  const onSubmit = async (data: ResetPasswordFormData) => {
    setError(null)
    setSuccess(null)

    if (!token) {
      setError('Token reset password tidak valid')
      return
    }

    setIsLoading(true)
    try {
      const res = await fetch('/api/v1/auth/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, password: data.password }),
      })
      if (!res.ok) {
        const msg = await extractApiErrorMessage(res, 'Gagal reset password')
        throw new Error(msg)
      }
      setSuccess('Password berhasil direset. Silakan login dengan password baru.')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Gagal reset password'
      setError(msg)
    } finally {
      setIsLoading(false)
    }
  }

  if (!token) {
    return (
      <div className="text-center text-sm text-destructive">
        Token reset password tidak valid atau sudah kedaluwarsa.
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
      <div
        ref={errorBoxRef}
        data-testid="reset-error"
        role="alert"
        style={{ display: "none" }}
        className="items-center gap-2 text-sm text-destructive bg-destructive/5 border border-destructive/20 p-3 rounded-lg"
      >
        <AlertCircle className="w-4 h-4 flex-shrink-0" />
        <span data-testid="reset-error-text" />
      </div>

      <div
        ref={successBoxRef}
        data-testid="reset-success"
        role="status"
        style={{ display: "none" }}
        className="items-center gap-2 text-sm text-green-600 bg-green-50 border border-green-200 p-3 rounded-lg"
      >
        <CheckCircle className="w-4 h-4 flex-shrink-0" />
        <span data-testid="reset-success-text" />
      </div>

      <div className="space-y-1.5">
        <label htmlFor="password" className="text-sm font-medium text-foreground">
          Kata Sandi Baru
        </label>
        <div className="relative">
          <Input
            id="password"
            type={showPassword ? 'text' : 'password'}
            placeholder="••••••••"
            className="pr-10"
            {...register('password')}
          />
          <button
            type="button"
            tabIndex={-1}
            onClick={() => setShowPassword((v) => !v)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
            aria-label={showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
          >
            {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
        {errors.password && (
          <p className="text-sm text-destructive">{errors.password.message}</p>
        )}
      </div>

      <div className="space-y-1.5">
        <label htmlFor="confirmPassword" className="text-sm font-medium text-foreground">
          Konfirmasi Kata Sandi
        </label>
        <div className="relative">
          <Input
            id="confirmPassword"
            type={showPassword ? 'text' : 'password'}
            placeholder="••••••••"
            className="pr-10"
            {...register('confirmPassword')}
          />
          <button
            type="button"
            tabIndex={-1}
            onClick={() => setShowPassword((v) => !v)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
            aria-label={showPassword ? 'Sembunyikan kata sandi' : 'Tampilkan kata sandi'}
          >
            {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
        {errors.confirmPassword && (
          <p className="text-sm text-destructive">{errors.confirmPassword.message}</p>
        )}
      </div>

      <Button type="submit" className="w-full font-semibold" disabled={isLoading}>
        {isLoading ? 'Meriset...' : 'Reset Password'}
      </Button>
    </form>
  )
}
