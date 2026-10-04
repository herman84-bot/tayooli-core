import { Metadata } from 'next'
import Link from 'next/link'
import { ForgotPasswordForm } from '@/components/auth/ForgotPasswordForm'
import { Logo } from '@/components/brand/Logo'

export const metadata: Metadata = { title: 'Lupa Password | Tayooli ERP' }

export default function ForgotPasswordPage() {
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

        {/* Card */}
        <div className="rounded-xl border border-border/70 bg-card p-6 shadow-sm sm:p-7">
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            Lupa Kata Sandi
          </h1>
          <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
            Masukkan email Anda dan kami akan mengirimkan link untuk reset password.
          </p>

          <div className="mt-6">
            <ForgotPasswordForm />
          </div>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Ingat kata sandi Anda?{' '}
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
