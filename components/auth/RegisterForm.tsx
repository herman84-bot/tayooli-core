'use client'

import { useLayoutEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { Eye, EyeOff, AlertCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { extractApiErrorMessage } from '@/lib/api/errors'

const registerSchema = z.object({
  fullName: z.string().trim().min(2, 'Nama harus minimal 2 karakter'),
  email: z.string().trim().email('Format email tidak valid'),
  password: z.string().min(8, 'Kata sandi minimal 8 karakter'),
  confirmPassword: z.string().min(8, 'Konfirmasi kata sandi minimal 8 karakter'),
}).refine((data) => data.password === data.confirmPassword, {
  message: 'Kata sandi tidak cocok',
  path: ['confirmPassword'],
})
type RegisterFormData = z.infer<typeof registerSchema>

export function RegisterForm() {
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const errorBoxRef = useRef<HTMLDivElement>(null)
  const [showPassword, setShowPassword] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
  })

  useLayoutEffect(() => {
    if (errorBoxRef.current) {
      errorBoxRef.current.style.display = error ? 'flex' : 'none'
      const text = errorBoxRef.current.querySelector('[data-testid="register-error-text"]')
      if (text) text.textContent = error ?? ''
    }
  }, [error])

  const onSubmit = async (data: RegisterFormData) => {
    setError(null)
    setIsLoading(true)
    try {
      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          full_name: data.fullName,
          email: data.email,
          password: data.password,
        }),
      })
      if (!res.ok) {
        // Backend Go mengembalikan { error: { code, message } } — ekstrak
        // message-nya saja agar tidak muncul "[object Object]" di UI.
        const msg = await extractApiErrorMessage(res, 'Gagal membuat akun')
        throw new Error(msg)
      }
      window.location.replace('/onboarding/workspace')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Gagal membuat akun'
      setError(msg)
      setIsLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
      <div
        ref={errorBoxRef}
        data-testid="register-error"
        role="alert"
        style={{ display: "none" }}
        className="items-center gap-2 text-sm text-destructive bg-destructive/5 border border-destructive/20 p-3 rounded-lg"
      >
        <AlertCircle className="w-4 h-4 flex-shrink-0" />
        <span data-testid="register-error-text" />
      </div>

      <div className="space-y-1.5">
        <label htmlFor="fullName" className="text-sm font-medium text-foreground">
          Nama Lengkap
        </label>
        <Input id="fullName" type="text" placeholder="Budi Santoso" {...register('fullName')} />
        {errors.fullName && (
          <p className="text-sm text-destructive">{errors.fullName.message}</p>
        )}
      </div>

      <div className="space-y-1.5">
        <label htmlFor="email" className="text-sm font-medium text-foreground">
          Email
        </label>
        <Input id="email" type="email" placeholder="budi@perusahaan.com" {...register('email')} />
        {errors.email && (
          <p className="text-sm text-destructive">{errors.email.message}</p>
        )}
      </div>

      <div className="space-y-1.5">
        <label htmlFor="password" className="text-sm font-medium text-foreground">
          Kata Sandi
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
        {isLoading ? 'Memproses...' : 'Buat Akun'}
      </Button>
    </form>
  )
}
