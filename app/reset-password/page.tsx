import { Metadata } from 'next'
import Link from 'next/link'
import { ResetPasswordForm } from '@/components/auth/ResetPasswordForm'
import { Logo } from '@/components/brand/Logo'

export const metadata: Metadata = { title: 'Reset Password | Tayooli ERP' }

export default async function ResetPasswordPage({
  searchParams,
}: {
  searchParams: Promise<{ invite?: string }>
}) {
  const isInvite = (await searchParams).invite === '1'
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
            {isInvite ? 'Aktifkan Akun' : 'Reset Kata Sandi'}
          </h1>
          <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
            {isInvite ? 'Buat kata sandi untuk akun undangan Anda.' : 'Masukkan kata sandi baru Anda.'}
          </p>

          <div className="mt-6">
            <ResetPasswordForm />
          </div>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          <Link href="/login" className="font-medium text-primary hover:text-primary/80 transition-colors">
            Kembali ke login
          </Link>
        </p>

        <p className="mt-8 text-center text-xs text-muted-foreground/60">
          &copy; 2026 Tayooli
        </p>
      </div>
    </div>
  )
}
