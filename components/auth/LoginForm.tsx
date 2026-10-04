"use client"

import { useLayoutEffect, useRef, useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"
import { Eye, EyeOff, AlertCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useAuth } from "@/hooks/useAuth"

const loginSchema = z.object({
  email: z.string().trim().email("Format email tidak valid"),
  password: z.string().min(8, "Kata sandi minimal 8 karakter"),
})
type LoginFormData = z.infer<typeof loginSchema>

export function LoginForm() {
  const { login } = useAuth()
  const [error, setError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const errorBoxRef = useRef<HTMLDivElement>(null)

  const [showPassword, setShowPassword] = useState(false)

  const { register, handleSubmit, formState: { errors } } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  })

  // Sinkronkan error ke DOM langsung SETELAH commit React (useLayoutEffect),
  // sehingga tidak ada render berikutnya yang menimpa textContent yang di-set
  // ref. Tanpa style/class display di JSX: React tidak pernah menimpa display
  // yang di-set ref di sini.
  useLayoutEffect(() => {
    if (errorBoxRef.current) {
      errorBoxRef.current.style.display = error ? "flex" : "none"
      const text = errorBoxRef.current.querySelector('[data-testid="login-error-text"]')
      if (text) text.textContent = error ?? ""
    }
  }, [error])

  const onSubmit = async (data: LoginFormData) => {
    setError(null)
    setIsLoading(true)
    try {
      await login(data.email, data.password)
      // Login sukses — hard navigation (sesi berbasis cookie, aman).
      // JANGAN pindahkan ke tempat lain: navigasi harus berjalan tanpa
      // bergantung pada commit state async (env dev ini tidak andal).
      window.location.replace("/dashboard")
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Login gagal"
      setError(msg)
      setIsLoading(false)
    }
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-5">
      {/* Error message — box selalu dirender; visibilitas dikontrol via style ref
          (state update async tidak andal di environment dev ini). */}
      <div
        ref={errorBoxRef}
        data-testid="login-error"
        role="alert"
        style={{ display: "none" }}
        className="items-center gap-2 text-sm text-destructive bg-destructive/5 border border-destructive/20 p-3 rounded-lg"
      >
        <AlertCircle className="w-4 h-4 flex-shrink-0" />
        {/* Teks di-set via ref di useLayoutEffect (bukan children React) —
            React tidak pernah menyentuh text node span ini setelah commit. */}
        <span data-testid="login-error-text" />
      </div>

      {/* Email field */}
      <div className="space-y-1.5">
        <label htmlFor="email" className="text-sm font-medium text-foreground">
          Email
        </label>
        <Input
          id="email"
          type="email"
          placeholder="nama@perusahaan.com"
          {...register("email")}
        />
        {errors.email && (
          <p className="text-sm text-destructive">{errors.email.message}</p>
        )}
      </div>

      {/* Password field */}
      <div className="space-y-1.5">
        <label htmlFor="password" className="text-sm font-medium text-foreground">
          Kata sandi
        </label>
        <div className="relative">
          <Input
            id="password"
            type={showPassword ? "text" : "password"}
            placeholder="••••••••"
            className="pr-10"
            {...register("password")}
          />
          <button
            type="button"
            tabIndex={-1}
            onClick={() => setShowPassword((v) => !v)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
            aria-label={showPassword ? "Sembunyikan kata sandi" : "Tampilkan kata sandi"}
          >
            {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
        {errors.password && (
          <p className="text-sm text-destructive">{errors.password.message}</p>
        )}
      </div>

      {/* Forgot password link */}
      <div className="flex justify-end">
        <a
          href="/forgot-password"
          className="text-sm text-muted-foreground hover:text-foreground transition-colors"
        >
          Lupa password?
        </a>
      </div>

      {/* Submit button */}
      <Button
        type="submit"
        className="w-full font-semibold"
        disabled={isLoading}
      >
        {isLoading ? "Memproses…" : "Masuk"}
      </Button>
    </form>
  )
}
