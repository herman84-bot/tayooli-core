"use client"

import { useEffect } from "react"
import { useAuth } from "@/hooks/useAuth"

/**
 * Guard sebaliknya untuk halaman publik (login):
 * jika user sudah terautentikasi, arahkan ke /dashboard.
 * Hard navigation dipakai karena router.push() App Router tidak memicu
 * navigasi di environment dev ini (lihat LoginForm).
 */
export function RedirectIfAuthenticated() {
  const { isAuthenticated, isLoading } = useAuth()

  useEffect(() => {
    if (isAuthenticated && !isLoading) {
      window.location.href = "/dashboard"
    }
  }, [isAuthenticated, isLoading])

  return null
}
