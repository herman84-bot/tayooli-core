'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Logo } from '@/components/brand/Logo'
import { DISABLE_BILLING } from '@/lib/config/demo'

export default function WorkspacePage() {
  const router = useRouter()
  const [companyName, setCompanyName] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!companyName.trim() || companyName.trim().length < 2) {
      setError('Nama perusahaan harus minimal 2 karakter')
      return
    }
    setError('')
    setIsLoading(true)
    try {
      const res = await fetch('/api/v1/workspaces', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ company_name: companyName.trim() }),
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || 'Gagal membuat workspace')
      }
      if (DISABLE_BILLING) {
        router.push('/dashboard')
      } else {
        router.push('/onboarding/trial')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Gagal membuat workspace')
      setIsLoading(false)
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
            <div className="h-2 w-2 rounded-full bg-primary" />
            <div className="h-2 w-2 rounded-full bg-muted" />
          </div>
          <span className="text-xs text-muted-foreground">3 dari 4</span>
        </div>

        {/* Card */}
        <div className="rounded-xl border border-border/70 bg-card p-6 shadow-sm sm:p-7">
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            Nama Toko / Perusahaan
          </h1>
          <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
            Ditampilkan pada struk kasir, surat jalan, dan dashboard operasional.
          </p>

          <form onSubmit={handleSubmit} className="mt-6 space-y-5">
            {error && (
              <div className="text-sm text-destructive bg-destructive/5 border border-destructive/20 p-3 rounded-lg">
                {error}
              </div>
            )}

            <div className="space-y-1.5">
              <label htmlFor="companyName" className="text-sm font-medium text-foreground">
                Nama Perusahaan
              </label>
              <Input
                id="companyName"
                type="text"
                placeholder="PT Maju Jaya"
                value={companyName}
                onChange={(e) => setCompanyName(e.target.value)}
                autoFocus
              />
            </div>

            <Button type="submit" className="w-full font-semibold" disabled={isLoading}>
              {isLoading ? 'Memproses...' : 'Lanjutkan'}
            </Button>
          </form>
        </div>

        <p className="mt-8 text-center text-xs text-muted-foreground/60">
          &copy; 2026 Tayooli
        </p>
      </div>
    </div>
  )
}
