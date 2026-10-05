"use client"

import { create } from "zustand"
import { persist, createJSONStorage } from "zustand/middleware"
import { extractApiErrorMessage } from "@/lib/api/errors"

interface User {
  id: string
  email: string
  role: string
  tenantId: string
}

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  isLoading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  hydrate: () => Promise<void>
}

/**
 * Canonical demo users — mirrors `lib/auth/demo-auth.ts` (DEMO_USERS).
 * Used to restore a valid demo session when the HttpOnly cookie cannot be
 * stored/sent (third-party cookie blocking or proxy stripping in the Freebuff
 * preview iframe). Demo credentials are public test accounts, so this is not
 * a security boundary — the real Go backend path never uses this fallback.
 */
const DEMO_USERS_BY_EMAIL: Record<string, User> = {
  "admin@test.com": {
    id: "550e8400-e29b-41d4-a716-446655440001",
    email: "admin@test.com",
    role: "admin",
    tenantId: "550e8400-e29b-41d4-a716-446655440000",
  },
  "accountant@test.com": {
    id: "550e8400-e29b-41d4-a716-446655440101",
    email: "accountant@test.com",
    role: "accountant",
    tenantId: "550e8400-e29b-41d4-a716-446655440000",
  },
  "approver@test.com": {
    id: "550e8400-e29b-41d4-a716-446655440102",
    email: "approver@test.com",
    role: "approver",
    tenantId: "550e8400-e29b-41d4-a716-446655440000",
  },
}

/**
 * Demo-mode fallback for `/api/v1/auth/me` failures (401/unreachable).
 * Only active when NO real backend is configured via NEXT_PUBLIC_API_URL.
 * With a backend configured, /me is the source of truth and a 401 keeps
 * logging the user out (no fallback) — a stale/fake local cookie must never
 * masquerade as a live session.
 */
function restoreDemoSession(): boolean {
  if (process.env.NEXT_PUBLIC_API_URL) return false
  const { user } = useAuthStore.getState()
  if (!user?.email) return false
  if (process.env.NODE_ENV === "production") return false

  const canonical = DEMO_USERS_BY_EMAIL[user.email.toLowerCase()]
  if (!canonical) return false

  useAuthStore.setState({ user: canonical, isAuthenticated: true, isLoading: false })
  return true
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      isLoading: true,

      login: async (email: string, password: string) => {
        set({ isLoading: true })
        const res = await fetch("/api/v1/auth/login", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ email, password }),
        })
        if (!res.ok) {
          set({ isLoading: false })
          // Backend mengembalikan { "error": { "code", "message" } } —
          // gunakan extractApiErrorMessage agar tidak pernah mengevaluasi ke [object Object].
          const extracted = await extractApiErrorMessage(res, "Login gagal")
          if (res.status === 401) {
            const isDefaultFallback =
              !extracted ||
              extracted === "Login gagal" ||
              extracted.toLowerCase() === "invalid credentials" ||
              extracted.toLowerCase() === "unauthorized"
            throw new Error(isDefaultFallback ? "Email atau kata sandi salah" : extracted)
          }
          throw new Error(extracted)
        }
        const data = await res.json()
        const user = data.user
          ? { ...data.user, tenantId: data.user.tenant_id ?? data.user.tenantId ?? "" }
          : null
        set({ user, isAuthenticated: true, isLoading: false })
      },

      logout: async () => {
        await fetch("/api/v1/auth/logout", { method: "POST", credentials: "include" })
        set({ user: null, isAuthenticated: false })
      },

      hydrate: async () => {
        set({ isLoading: true })
        try {
          const res = await fetch("/api/v1/auth/me", { credentials: "include" })
          if (res.ok) {
            const data = await res.json()
            set({ user: data.user, isAuthenticated: true, isLoading: false })
            return
          }
          // Cookie hilang/tidak terkirim (preview iframe) → demo fallback.
          if (restoreDemoSession()) return
          set({ user: null, isAuthenticated: false, isLoading: false })
        } catch {
          if (restoreDemoSession()) return
          set({ user: null, isAuthenticated: false, isLoading: false })
        }
      },
    }),
    {
      name: "tayooli-auth",
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({ user: state.user, isAuthenticated: state.isAuthenticated }),
    }
  )
)

export function useAuth() {
  const { user, isAuthenticated, isLoading, login, logout, hydrate } = useAuthStore()
  return { user, isAuthenticated, isLoading, login, logout, hydrate }
}
