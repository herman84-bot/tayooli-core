import { Metadata } from "next"
import Link from "next/link"
import { LoginForm } from "@/components/auth/LoginForm"
import { RedirectIfAuthenticated } from "@/components/auth/RedirectIfAuthenticated"
import { Logo } from "@/components/brand/Logo"

export const metadata: Metadata = { title: "Masuk | Tayooli ERP" }

export default function LoginPage() {
  return (
    <div className="relative flex min-h-screen flex-col items-center justify-center bg-background px-4 py-8">
      {/* Jika sudah login, arahkan ke dashboard (dan setelah login sukses). */}
      <RedirectIfAuthenticated />

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
            Masuk ke akun Anda
          </h1>
          <p className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
            Kelola invoice, persetujuan, dan pembayaran bisnis Anda dalam satu tempat.
          </p>

          <div className="mt-6">
            <LoginForm />
          </div>
        </div>

        <p className="mt-6 text-center text-sm text-muted-foreground">
          Belum punya akun?{' '}
          <Link href="/register" className="font-medium text-primary hover:text-primary/80 transition-colors">
            Daftar
          </Link>
        </p>

        {process.env.NODE_ENV === "development" && (
          <div className="mt-5 text-center">
            <p className="text-[11px] uppercase tracking-wider text-muted-foreground/70">Demo</p>
            <p className="mt-1 font-mono text-xs text-muted-foreground">admin@test.com / password123</p>
          </div>
        )}

        <p className="mt-8 text-center text-xs text-muted-foreground/60">
          © 2026 Tayooli
        </p>
      </div>
    </div>
  )
}
