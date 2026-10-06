'use client'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useEffect, useState, type ReactNode } from 'react'
import { useAuth, useAuthStore } from '@/hooks/useAuth'

/** Identity key: cache must never outlive the user/tenant that fetched it. */
function identityKey(user: { id?: string; tenantId?: string; tenant_id?: string } | null): string {
  if (!user) return ''
  return `${user.id ?? ''}:${user.tenantId ?? user.tenant_id ?? ''}`
}

interface ProvidersProps {
  children: ReactNode
}

function AuthHydrator({ children }: { children: ReactNode }) {
  const { hydrate } = useAuth()
  // Hanya gating pada hidrasi AWAL, bukan pada setiap toggle isLoading.
  // Store isLoading ikut berubah saat login()/logout() berjalan — jika dipakai
  // sebagai gate, SELURUH tree (termasuk LoginForm) unmount/remount setiap
  // percobaan login, sehingga error state lokal di LoginForm hilang.
  const [hydrated, setHydrated] = useState(false)

  useEffect(() => {
    let active = true
    hydrate().finally(() => {
      if (active) setHydrated(true)
    })
    return () => {
      active = false
    }
  }, [hydrate])

  if (!hydrated) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  return <>{children}</>
}

export default function Providers({ children }: ProvidersProps) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            refetchOnWindowFocus: false,
            retry: 2,
            retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10_000),
          },
          mutations: {
            retry: 1,
          },
        },
      }),
  )

  // Drop every cached query whenever the signed-in identity changes
  // (logout, login as another user/tenant). Without this, POS/products/stock
  // fetched for tenant A stay visible to tenant B until staleTime expires.
  useEffect(() => {
    let prev = identityKey(useAuthStore.getState().user as never)
    return useAuthStore.subscribe((state) => {
      const next = identityKey(state.user as never)
      if (next !== prev) {
        prev = next
        queryClient.cancelQueries()
        queryClient.clear()
      }
    })
  }, [queryClient])

  return (
    <QueryClientProvider client={queryClient}>
      <AuthHydrator>{children}</AuthHydrator>
    </QueryClientProvider>
  )
}
