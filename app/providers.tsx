'use client'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useEffect, useState, type ReactNode } from 'react'
import { useAuth } from '@/hooks/useAuth'

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

  return (
    <QueryClientProvider client={queryClient}>
      <AuthHydrator>{children}</AuthHydrator>
    </QueryClientProvider>
  )
}
