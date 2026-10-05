'use client'

import { useState, useEffect, useCallback } from 'react'
import Link from 'next/link'
import { usePathname } from 'next/navigation'
import {
  LayoutDashboard,
  Store,
  Box,
  Warehouse,
  Truck,
  ClipboardCheck,
  FileCheck,
  ShoppingBag,
  AlertTriangle,
  ScanLine,
  MessageCircle,
  Settings,
  LogOut,
  ChevronLeft,
  ChevronRight,
  X,
} from 'lucide-react'
import { useAuth } from '@/hooks/useAuth'
import { useIsMobile } from '@/hooks/use-mobile'
import { cn } from '@/lib/utils'
import { Logo } from '@/components/brand/Logo'

interface NavItem {
  href: string
  label: string
  icon: React.ElementType
  exact?: boolean
  tutorial?: string
}

interface NavGroup {
  label: string
  items: NavItem[]
}

const navGroups: NavGroup[] = [
  {
    label: 'Overview',
    items: [
      { href: '/dashboard', label: 'Dashboard', icon: LayoutDashboard, exact: true },
    ],
  },
  {
    label: 'Inventory',
    items: [
      { href: '/products', label: 'Products', icon: Box },
    ],
  },
  {
    label: 'Warehouse & POS',
    items: [
      { href: '/wms', label: 'Warehouse & Stock', icon: Warehouse, exact: true },
      { href: '/wms/delivery-orders', label: 'Surat Jalan (DO)', icon: FileCheck },
      { href: '/wms/marketplace', label: 'Marketplace Omnichannel', icon: ShoppingBag },
      { href: '/wms/transfers', label: 'Stock Transfers', icon: Truck },
      { href: '/wms/opname', label: 'Stock Opname', icon: ClipboardCheck },
      { href: '/wms/scrap', label: 'Barang Rusak / Scrap', icon: AlertTriangle },
      { href: '/wms/scanner', label: 'Barcode Scanner', icon: ScanLine },
      { href: '/pos', label: 'Point of Sale', icon: Store },
    ],
  },
  {
    label: 'Account',
    items: [
      { href: '/settings', label: 'Settings', icon: Settings },
      { href: '/help', label: 'Help & Support', icon: MessageCircle, tutorial: 'help-support' },
    ],
  },
]

/** Width classes for sidebar modes. */
export const SIDEBAR_EXPANDED_W = 'w-60'
export const SIDEBAR_COLLAPSED_W = 'w-14'

interface SidebarProps {
  /** Mobile drawer open state. Controlled by layout. */
  mobileOpen?: boolean
  /** Close mobile drawer. */
  onMobileClose?: () => void
}

export default function Sidebar({ mobileOpen = false, onMobileClose }: SidebarProps) {
  const pathname = usePathname()
  const { user, logout } = useAuth()
  const isMobile = useIsMobile()

  const [collapsed, setCollapsed] = useState(() => {
    if (typeof window === 'undefined') return false
    return localStorage.getItem('sidebarCollapsed') === 'true'
  })

  const toggleCollapsed = useCallback(() => {
    setCollapsed((prev) => {
      const next = !prev
      localStorage.setItem('sidebarCollapsed', String(next))
      return next
    })
  }, [])

  // Auto-collapse on mobile breakpoint
  useEffect(() => {
    if (isMobile) setCollapsed(false)
  }, [isMobile])

  // Close mobile drawer on navigation
  useEffect(() => {
    if (isMobile && mobileOpen) onMobileClose?.()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pathname])

  const userInitial = user?.email?.charAt(0).toUpperCase() ?? '?'

  /* ── Mobile drawer ────────────────────────────────────────────────── */
  if (isMobile) {
    return (
      <>
        {/* Backdrop */}
        {mobileOpen && (
          <div
            className="fixed inset-0 z-40 bg-black/30 transition-opacity"
            onClick={onMobileClose}
            aria-hidden="true"
          />
        )}

        {/* Drawer */}
        <aside
          className={cn(
            'fixed inset-y-0 left-0 z-50 w-60 bg-background border-r border-border flex flex-col',
            'transition-transform duration-200 ease-in-out',
            mobileOpen ? 'translate-x-0' : '-translate-x-full',
          )}
        >
          {/* Brand + close */}
          <div className="px-4 py-4 border-b border-border flex items-center gap-2.5">
            <Logo size={32} />
            <div className="min-w-0 flex-1">
              <span className="block font-semibold text-foreground text-sm tracking-tight leading-tight">
                Tayooli
              </span>
              <span className="block text-[11px] text-muted-foreground leading-tight">
                ERP · Standalone
              </span>
            </div>
            <button
              onClick={onMobileClose}
              className="shrink-0 p-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
              aria-label="Close menu"
            >
              <X className="h-4 w-4" />
            </button>
          </div>

          {/* Nav */}
          <nav className="flex-1 px-2.5 py-3 overflow-y-auto">
            {navGroups.map((group, gi) => (
              <div key={group.label} className={gi > 0 ? 'mt-4' : undefined}>
                <p className="px-2.5 mb-1 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground/60">
                  {group.label}
                </p>
                <div className="space-y-px">
                  {group.items.map(({ href, label, icon: Icon, exact }) => {
                    const active = exact
                      ? pathname === href
                      : pathname === href || pathname.startsWith(href + '/')
                    return (
                      <Link
                        key={href}
                        href={href}
                        aria-current={active ? 'page' : undefined}
                        className={cn(
                          'flex items-center gap-2 px-2.5 py-1.5 rounded-md text-[13px] transition-colors',
                          active
                            ? 'bg-primary/8 text-primary font-medium'
                            : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
                        )}
                      >
                        <Icon className={cn('h-[15px] w-[15px]', active ? 'text-primary' : 'text-muted-foreground/70')} />
                        {label}
                      </Link>
                    )
                  })}
                </div>
              </div>
            ))}
          </nav>

          {/* User */}
          <div className="border-t border-border px-3 py-2.5">
            {user && (
              <div className="flex items-center justify-between">
                <div className="min-w-0 flex-1">
                  <div className="text-xs font-medium text-foreground truncate">{user.email}</div>
                </div>
                <button
                  onClick={() => {
                    void (async () => {
                      await logout()
                      window.location.replace('/login')
                    })()
                  }}
                  className="ml-2 shrink-0 flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                  aria-label="Logout"
                >
                  <LogOut className="h-3.5 w-3.5" />
                </button>
              </div>
            )}
          </div>
        </aside>
      </>
    )
  }

  /* ── Desktop sidebar ──────────────────────────────────────────────── */
  return (
    <aside
      className={cn(
        'shrink-0 border-r border-border bg-background flex flex-col transition-all duration-200 ease-in-out',
        collapsed ? SIDEBAR_COLLAPSED_W : SIDEBAR_EXPANDED_W,
      )}
    >
      {/* Brand + toggle */}
      <div className={cn(
        'border-b border-border flex items-center',
        collapsed ? 'px-2 py-3 justify-center' : 'px-4 py-4 gap-2.5',
      )}>
        {collapsed ? (
          <Logo size={28} />
        ) : (
          <>
            <Logo size={32} />
            <div className="min-w-0 flex-1">
              <span className="block font-semibold text-foreground text-sm tracking-tight leading-tight">
                Tayooli
              </span>
              <span className="block text-[11px] text-muted-foreground leading-tight">
                ERP · Standalone
              </span>
            </div>
          </>
        )}
        <button
          onClick={toggleCollapsed}
          className={cn(
            'shrink-0 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors',
            collapsed ? 'p-1.5' : 'p-1',
          )}
          aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed ? (
            <ChevronRight className="h-4 w-4" />
          ) : (
            <ChevronLeft className="h-4 w-4" />
          )}
        </button>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-2 py-3 overflow-y-auto overflow-x-hidden" data-tutorial="sidebar">
        {navGroups.map((group, gi) => (
          <div key={group.label} className={gi > 0 ? 'mt-4' : undefined}>
            {!collapsed && (
              <p className="px-2.5 mb-1 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground/60">
                {group.label}
              </p>
            )}
            {collapsed && gi > 0 && (
              <div className="mx-2 my-2 border-t border-border/50" />
            )}
            <div className={collapsed ? 'space-y-1' : 'space-y-px'}>
              {group.items.map((item) => {
                const { href, label, icon: Icon, exact, tutorial } = item
                const active = exact
                  ? pathname === href
                  : pathname === href || pathname.startsWith(href + '/')
                return (
                  <Link
                    key={href}
                    href={href}
                    aria-current={active ? 'page' : undefined}
                    title={collapsed ? label : undefined}
                    data-tutorial={tutorial}
                    className={cn(
                      'group/item flex items-center rounded-md transition-colors',
                      collapsed
                        ? 'justify-center px-0 py-2 mx-auto w-10'
                        : 'gap-2 px-2.5 py-1.5',
                      active
                        ? 'bg-primary/8 text-primary font-medium'
                        : 'text-muted-foreground hover:bg-muted/60 hover:text-foreground',
                    )}
                  >
                    <Icon
                      className={cn(
                        'shrink-0',
                        collapsed ? 'h-[18px] w-[18px]' : 'h-[15px] w-[15px]',
                        active ? 'text-primary' : 'text-muted-foreground/70',
                      )}
                    />
                    {!collapsed && <span className="text-[13px] truncate">{label}</span>}
                    {/* Tooltip for collapsed mode */}
                    {collapsed && (
                      <span className="pointer-events-none absolute left-full ml-2 z-50 hidden group-hover/item:flex items-center whitespace-nowrap rounded-md bg-foreground px-2.5 py-1 text-xs font-medium text-background shadow-md">
                        {label}
                      </span>
                    )}
                  </Link>
                )
              })}
            </div>
          </div>
        ))}
      </nav>

      {/* User info & logout */}
      <div className={cn('border-t border-border', collapsed ? 'px-2 py-2.5' : 'px-3 py-2.5')}>
        {user && (
          collapsed ? (
            <div className="flex flex-col items-center gap-1.5">
              <div
                className="h-7 w-7 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-semibold"
                title={user.email}
              >
                {userInitial}
              </div>
              <button
                onClick={() => {
                  void (async () => {
                    await logout()
                    window.location.replace('/login')
                  })()
                }}
                className="p-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                aria-label="Logout"
                title="Logout"
              >
                <LogOut className="h-3.5 w-3.5" />
              </button>
            </div>
          ) : (
            <div className="flex items-center justify-between">
              <div className="min-w-0 flex-1">
                <div className="text-xs font-medium text-foreground truncate">{user.email}</div>
              </div>
              <button
                onClick={() => {
                  void (async () => {
                    await logout()
                    window.location.replace('/login')
                  })()
                }}
                className="ml-2 shrink-0 flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
                aria-label="Logout"
              >
                <LogOut className="h-3.5 w-3.5" />
              </button>
            </div>
          )
        )}
      </div>
    </aside>
  )
}
