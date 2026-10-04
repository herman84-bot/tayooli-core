'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'
import { useSearchParams } from 'next/navigation'
import { Mail, CheckCircle, AlertCircle, Loader2 } from 'lucide-react'
import { Logo } from '@/components/brand/Logo'

type VerifyState = 'loading' | 'success' | 'error' | 'pending'

export default function VerifyEmailPage() {
  const searchParams = useSearchParams()
  const token = searchParams.get('token')
  const email = searchParams.get('email')

  const [state, setState] = useState<VerifyState>(token ? 'loading' : 'pending')
  const [resendCooldown, setResendCooldown] = useState(0)
  const [resendMessage, setResendMessage] = useState('')

  useEffect(() => {
    if (!token) return

    const verify = async () => {
      try {
        const res = await fetch(`/api/v1/auth/verify-email?token=${token}`)
        if (res.ok) {
          setState('success')
          setTimeout(() => {
            window.location.replace('/onboarding/workspace')
          }, 2000)
        } else {
          setState('error')
        }
      } catch {
        setState('error')
      }
    }
    verify()
  }, [token])

  useEffect(() => {
    if (resendCooldown <= 0) return
    const timer = setTimeout(() => setResendCooldown((c) => c - 1), 1000)
    return () => clearTimeout(timer)
  }, [resendCooldown])

  const handleResend = async () => {
    if (!email || resendCooldown > 0) return
    setResendMessage('')
    try {
      const res = await fetch('/api/v1/auth/resend-verification', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email }),
      })
      if (res.ok) {
        setResendCooldown(60)
        setResendMessage('Link verifikasi dikirim ulang.')
      } else {
        setResendMessage('Gagal mengirim ulang. Coba lagi nanti.')
      }
    } catch {
      setResendMessage('Gagal mengirim ulang. Coba lagi nanti.')
    }
  }

  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center bg-background px-4 py-8">
      <div className="w-full max-w-sm">
        {/* Brand */}
        <div className="flex items-center justify-center gap-2.5 mb-6">
          <Logo size={36} />
          <span className="text-lg font-semibold tracking-tight text-foreground">
            Tayooli
          </span>
        </div>

        {/* Step indicator */}
        <div className="flex items-center justify-center gap-2 mb-6">
          <div className="flex items-center gap-1.5">
            <div className="h-2 w-2 rounded-full bg-primary" />
            <div className="h-2 w-2 rounded-full bg-primary" />
            <div className="h-2 w-2 rounded-full bg-muted" />
            <div className="h-2 w-2 rounded-full bg-muted" />
          </div>
          <span className="text-xs text-muted-foreground">2 dari 4</span>
        </div>

        {/* Card */}
        <div className="rounded-xl border border-border/70 bg-card p-6 shadow-sm sm:p-7">
          {state === 'loading' && (
            <div className="flex flex-col items-center py-8">
              <Loader2 className="h-8 w-8 text-primary animate-spin mb-4" />
              <p className="text-sm text-muted-foreground">Memverifikasi...</p>
            </div>
          )}

          {state === 'success' && (
            <div className="flex flex-col items-center py-8">
              <CheckCircle className="h-12 w-12 text-primary mb-4" />
              <h1 className="text-xl font-semibold tracking-tight text-foreground">
                Email Terverifikasi
              </h1>
              <p className="mt-2 text-sm text-muted-foreground text-center">
                Mengalihkan ke setup workspace...
              </p>
            </div>
          )}

          {state === 'error' && (
            <div className="flex flex-col items-center py-8">
              <AlertCircle className="h-12 w-12 text-destructive mb-4" />
              <h1 className="text-xl font-semibold tracking-tight text-foreground">
                Token Tidak Valid
              </h1>
              <p className="mt-2 text-sm text-muted-foreground text-center">
                Link verifikasi tidak valid atau sudah kedaluwarsa.
              </p>
              <button
                onClick={handleResend}
                disabled={resendCooldown > 0}
                className="mt-4 text-sm font-medium text-primary hover:text-primary/80 disabled:text-muted-foreground disabled:cursor-not-allowed"
              >
                {resendCooldown > 0
                  ? `Kirim ulang dalam ${resendCooldown}s`
                  : 'Kirim Ulang Link'}
              </button>
            </div>
          )}

          {state === 'pending' && (
            <div className="flex flex-col items-center py-8">
              <Mail className="h-12 w-12 text-primary mb-4" />
              <h1 className="text-xl font-semibold tracking-tight text-foreground">
                Cek Email Anda
              </h1>
              <p className="mt-2 text-sm text-muted-foreground text-center">
                Kami kirim link verifikasi ke{' '}
                <span className="font-medium text-foreground">{email}</span>
              </p>
              <button
                onClick={handleResend}
                disabled={resendCooldown > 0}
                className="mt-4 text-sm font-medium text-primary hover:text-primary/80 disabled:text-muted-foreground disabled:cursor-not-allowed"
              >
                {resendCooldown > 0
                  ? `Kirim ulang dalam ${resendCooldown}s`
                  : 'Kirim Ulang'}
              </button>
              {resendMessage && (
                <p className="mt-2 text-xs text-muted-foreground">{resendMessage}</p>
              )}
            </div>
          )}
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          <Link href="/login" className="font-medium text-primary hover:text-primary/80 transition-colors">
            Kembali ke Masuk
          </Link>
        </p>

        <p className="mt-8 text-center text-xs text-muted-foreground/60">
          &copy; 2026 Tayooli
        </p>
      </div>
    </div>
  )
}
