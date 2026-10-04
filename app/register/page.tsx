import { Metadata } from 'next'
import Link from 'next/link'
import { RegisterForm } from '@/components/auth/RegisterForm'
import { Logo } from '@/components/brand/Logo'

export const metadata: Metadata = { title: 'Daftar | Tayooli ERP' }

export default function RegisterPage() {
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
            <div className="h-2 w-2 rounded-full bg-muted" />
            <div className="h-2 w-2 rounded-full bg-muted" />
            <div className="h-2 w-2 rounded-full bg-muted" />
          </div>
          <span className="text-xs text-muted-foreground">1 dari 4</span>
        </div>

        {/* Card */}
        <div className="rounded-xl border border-border/70 bg-card p-6 shadow-sm sm:p-7">
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            Buat Akun
          </h1>
          <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
            Isi data di bawah untuk mulai.
          </p>

          <div className="mt-6">
            <RegisterForm />
          </div>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Sudah punya akun?{' '}
          <Link href="/login" className="font-medium text-primary hover:text-primary/80 transition-colors">
            Masuk
          </Link>
        </p>

        <p className="mt-8 text-center text-xs text-muted-foreground/60">
          &copy; 2026 Tayooli
        </p>
      </div>
    </div>
  )
}
