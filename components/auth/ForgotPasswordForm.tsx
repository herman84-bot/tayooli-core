'use client'

import { useLayoutEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { AlertCircle, CheckCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { extractApiErrorMessage } from '@/lib/api/errors'

const forgotPasswordSchema = z.object({
  email: z.string().trim().email('Format email tidak valid'),
})
type ForgotPasswordFormData = z.infer<typeof forgotPasswordSchema>

export function ForgotPasswordForm() {
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const errorBoxRef = useRef<HTMLDivElement>(null)
  const successBoxRef = useRef<HTMLDivElement>(null)

  const { register, handleSubmit, formState: { errors } } = useForm<ForgotPasswordFormData>({
    resolver: zodResolver(forgotPasswordSchema),
  })

  useLayoutEffect(() => {
    if (errorBoxRef.current) {
      errorBoxRef.current.style.display = error ? 'flex' : 'none'
      const text = errorBoxRef.current.querySelector('[data-testid="forgot-error-text"]')
      if (text) text.textContent = error ?? ''
    }
  }, [error])

  useLayoutEffect(() => {
    if (successBoxRef.current) {
      successBoxRef.current.style.display = success ? 'flex' : 'none'
      const text = successBoxRef.current.querySelector('[data-testid="forgot-success-text"]')
      if (text) text.textContent = success ?? ''
    }
  }, [success])

  const onSubmit = async (data: ForgotPasswordFormData) => {
    setError(null)
    setSuccess(null)
    setIsLoading(true)
    try {
      const res = await fetch('/api/v1/auth/forgot-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: data.email }),
      })
      if (!res.ok) {
        const msg = await extractApiErrorMessage(res, 'Gagal mengirim email reset password')
        throw new Error(msg)
      }
      setSuccess('Link reset password telah dikirim. Cek Inbox atau folder Spam/Junk email Anda.')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Gagal mengirim email reset password'
      setError(msg)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
        <div
          ref={errorBoxRef}
          data-testid="forgot-error"
          role="alert"
          style={{ display: "none" }}
          className="items-center gap-2 text-sm text-destructive bg-destructive/5 border border-destructive/20 p-3 rounded-lg"
        >
          <AlertCircle className="w-4 h-4 flex-shrink-0" />
          <span data-testid="forgot-error-text" />
        </div>

        <div
          ref={successBoxRef}
          data-testid="forgot-success"
          role="status"
          style={{ display: "none" }}
          className="items-center gap-2 text-sm text-green-600 bg-green-50 border border-green-200 p-3 rounded-lg"
        >
          <CheckCircle className="w-4 h-4 flex-shrink-0" />
          <span data-testid="forgot-success-text" />
        </div>

        <div className="space-y-1.5">
          <label htmlFor="email" className="text-sm font-medium text-foreground">
            Email
          </label>
          <Input
            id="email"
            type="email"
            placeholder="nama@perusahaan.com"
            {...register('email')}
          />
          {errors.email && (
            <p className="text-sm text-destructive">{errors.email.message}</p>
          )}
        </div>

        <Button type="submit" className="w-full font-semibold" disabled={isLoading}>
          {isLoading ? 'Mengirim...' : 'Kirim Link Reset'}
        </Button>
      </form>

      <p className="mt-4 text-center text-sm text-muted-foreground">
        <a href="/login" className="font-medium text-primary hover:text-primary/80 transition-colors">
          ← Kembali ke login
        </a>
      </p>
    </>
  )
}
