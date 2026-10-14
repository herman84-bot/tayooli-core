import { Metadata } from 'next'
import Link from 'next/link'
import { UserX } from 'lucide-react'
import { Logo } from '@/components/brand/Logo'

export const metadata: Metadata = { title: 'Pendaftaran Ditutup | Tayooli ERP' }

export default function RegisterPage() {
  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center bg-background px-4 py-8">
      <div className="w-full max-w-md">
        {/* Brand */}
        <div className="flex items-center justify-center gap-2.5 mb-6">
          <Logo size={36} />
          <span className="text-lg font-semibold tracking-tight text-foreground">
            Tayooli
          </span>
        </div>

        {/* Card */}
        <div className="rounded-2xl border border-border/70 bg-card p-6 shadow-sm sm:p-8 text-center space-y-5">
          <div className="mx-auto w-12 h-12 rounded-full bg-amber-500/10 border border-amber-500/20 text-amber-600 dark:text-amber-400 flex items-center justify-center">
            <UserX className="w-6 h-6" />
          </div>

          <div className="space-y-2">
            <h1 className="text-xl font-bold tracking-tight text-foreground">
              Pendaftaran Akun Dinonaktifkan
            </h1>
            <p className="text-sm leading-relaxed text-muted-foreground">
              Pendaftaran akun mandiri saat ini ditutup. Akses akun pengguna ke sistem Tayooli ERP dibuat dan dikelola langsung oleh administrator perusahaan Anda.
            </p>
          </div>

          <div className="pt-2">
            <Link
              href="/login"
              className="inline-flex items-center justify-center w-full min-h-[44px] px-4 py-2.5 rounded-xl font-bold text-sm bg-primary hover:bg-primary/90 text-primary-foreground shadow-sm transition-all"
            >
              Kembali ke Halaman Masuk
            </Link>
          </div>
        </div>

        <p className="mt-8 text-center text-xs text-muted-foreground/60">
          &copy; 2026 Tayooli
        </p>
      </div>
    </div>
  )
}
