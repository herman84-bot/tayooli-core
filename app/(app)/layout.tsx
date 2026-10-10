'use client'

import { useState, useEffect } from 'react'
import { usePathname } from 'next/navigation'
import { useAuth } from '@/hooks/useAuth'
import { useIsMobile } from '@/hooks/use-mobile'
import Sidebar from '@/components/layout/Sidebar'
import { Menu } from 'lucide-react'
import { canAccessRoute, homeRouteFor } from '@/lib/rbac'

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading, user } = useAuth()
  const isMobile = useIsMobile()
  const pathname = usePathname() ?? ''
  const [mobileOpen, setMobileOpen] = useState(false)

  const role = user?.role
  const allowed = canAccessRoute(role, pathname)

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      window.location.replace('/login')
    }
  }, [isAuthenticated, isLoading])

  // RBAC route guard: arahkan ke halaman awal role jika URL tidak diizinkan.
  useEffect(() => {
    if (isLoading || !isAuthenticated || allowed) return
    const home = homeRouteFor(role)
    if (home !== pathname) window.location.replace(home)
  }, [isLoading, isAuthenticated, allowed, role, pathname])

  return (
    <div className="min-h-screen bg-background flex">
      {/* ── Sidebar ── */}
      <Sidebar mobileOpen={mobileOpen} onMobileClose={() => setMobileOpen(false)} />

      {/* ── Main content ── */}
      <main className="flex-1 min-w-0 overflow-y-auto">
        {/* Mobile header with hamburger */}
        {isMobile && (
          <div className="sticky top-0 z-30 flex items-center h-12 px-4 border-b border-border bg-background/95 backdrop-blur">
            <button
              onClick={() => setMobileOpen(true)}
              className="p-1.5 -ml-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
              aria-label="Open menu"
            >
              <Menu className="h-5 w-5" />
            </button>
            <span className="ml-3 text-sm font-semibold text-foreground">Tayooli</span>
          </div>
        )}

        {isLoading ? (
          <div className="flex items-center justify-center h-full min-h-96">
            <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
        ) : !isAuthenticated ? null : !allowed ? (
          <div role="alert" className="flex items-center justify-center min-h-96 p-6 text-sm text-muted-foreground">
            Akses terbatas untuk role Anda. Mengalihkan...
          </div>
        ) : (
          children
        )}
      </main>
    </div>
  )
}
