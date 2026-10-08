'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useDashboardSummary } from '@/lib/queries/dashboard'
import { TooltipWalkthrough } from '@/components/tutorial/TooltipWalkthrough'
import {
  TrendingUp,
  Receipt,
  Package,
  Warehouse,
  AlertTriangle,
  Store,
  ExternalLink,
  RefreshCw,
  ArrowLeftRight,
  ArrowDownLeft,
  Truck,
  Users,
} from 'lucide-react'
import { EMPTY_DASHBOARD_SUMMARY } from '@/lib/schemas/dashboard'
import { formatCurrency } from '@/lib/currency'

// Dashboard tayooli-core: 4 area ringkasan.
// 1) Ribbon keuangan & mutasi fisik (penjualan, kas masuk, valuasi stok, barang keluar)
// 2) POS kasir & penjualan  3) Gudang & monitoring stok (WMS)
// 4) Aktivitas Barang Keluar & Pemenuhan (Outbound Wave, DO, Top Produk & Top Pelanggan)

function formatTime(createdAt: string): string {
  if (!createdAt) return ''
  const d = new Date(createdAt)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

function SectionHeader({
  icon: Icon,
  title,
  href,
  linkLabel,
}: {
  icon: React.ElementType
  title: string
  href?: string
  linkLabel?: string
}) {
  return (
    <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between">
      <div className="flex items-center gap-2">
        <Icon className="h-3.5 w-3.5 text-muted-foreground" />
        <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">{title}</h2>
      </div>
      {href && linkLabel && (
        <Link href={href} className="text-[11px] font-medium text-primary hover:underline flex items-center gap-1">
          {linkLabel} <ExternalLink className="h-3 w-3" />
        </Link>
      )}
    </div>
  )
}

interface StatCardProps {
  label: string
  value: number | string
  icon: React.ElementType
  iconClass: string
  bg: string
  subtext?: string
}

function StatCard({ label, value, icon: Icon, iconClass, bg, subtext }: StatCardProps) {
  return (
    <div className="border border-border/60 rounded-lg p-4 flex items-center gap-3 bg-card">
      <div className={`h-9 w-9 rounded-lg flex items-center justify-center shrink-0 ${bg}`}>
        <Icon className={`h-4 w-4 ${iconClass}`} />
      </div>
      <div className="min-w-0">
        <div className="text-xl font-bold tracking-tight text-foreground font-mono">{value}</div>
        <div className="text-[11px] text-muted-foreground leading-tight">{label}</div>
        {subtext && <div className="text-[10px] text-muted-foreground/70 mt-0.5 truncate">{subtext}</div>}
      </div>
    </div>
  )
}

function MiniStat({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="bg-muted/40 rounded-lg p-2.5 text-center border border-border/30">
      <div className="text-[10px] text-muted-foreground uppercase tracking-wide">{label}</div>
      <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">{value}</div>
    </div>
  )
}

export default function DashboardPage() {
  const [period, setPeriod] = useState<'7d' | '30d' | '90d'>('30d')
  const { data: summary, isLoading, error, refetch } = useDashboardSummary(period)
  const s = summary ?? EMPTY_DASHBOARD_SUMMARY
  const loading = isLoading && !summary
  const status = (error as { response?: { status?: number } } | null)?.response?.status
  const needsLogin = status === 401

  return (
    <div className="px-6 py-6 space-y-6">
      <TooltipWalkthrough />

      <div className="flex items-start justify-between gap-3" data-tutorial="dashboard-header">
        <div>
          <h1 className="text-lg font-bold tracking-tight text-foreground">Dashboard</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            Ringkasan kasir POS, gudang dan stok, penjualan, serta aktivitas barang keluar.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Link
            href="/pos"
            data-tutorial="pos-button"
            className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs font-semibold transition-colors"
          >
            <Store className="h-3.5 w-3.5" />
            Buka Kasir POS
          </Link>
          <button
            type="button"
            onClick={() => void refetch()}
            disabled={isLoading}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 border border-border/60 rounded-lg text-xs font-medium text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isLoading ? 'animate-spin' : ''}`} />
            Segarkan
          </button>
        </div>
      </div>

      {error && !isLoading && (
        <div
          role="alert"
          className="flex items-center justify-between gap-3 p-3 rounded-lg border border-amber-500/30 bg-amber-50 dark:bg-amber-950/20 text-amber-800 dark:text-amber-200 text-xs"
        >
          <div className="flex items-center gap-2 min-w-0">
            <AlertTriangle className="h-4 w-4 text-amber-600 shrink-0" />
            <span className="truncate">
              {needsLogin
                ? 'Sesi berakhir atau belum masuk. Silakan masuk untuk melihat data asli.'
                : 'Gagal memuat data dari server. Angka di bawah belum terisi.'}
            </span>
          </div>
          {needsLogin ? (
            <Link
              href="/login"
              className="px-2.5 py-1 bg-amber-600 hover:bg-amber-700 text-white rounded-md font-medium transition-colors shrink-0"
            >
              Masuk
            </Link>
          ) : (
          <button
            type="button"
            onClick={() => void refetch()}
            className="px-2.5 py-1 bg-amber-600 hover:bg-amber-700 text-white rounded-md font-medium transition-colors shrink-0"
          >
            Coba Lagi
          </button>
          )}
        </div>
      )}

      <div className={`grid grid-cols-2 lg:grid-cols-4 gap-3 ${loading ? 'opacity-50' : ''}`}>
        <StatCard
          label="Total Penjualan"
          value={formatCurrency(s.financial_overview.total_revenue, { compact: true })}
          icon={TrendingUp}
          iconClass="text-emerald-600"
          bg="bg-emerald-50 dark:bg-emerald-950/30"
          subtext={`Piutang: ${formatCurrency(s.financial_overview.accounts_receivable, { compact: true })}`}
        />
        <StatCard
          label="Kas Masuk"
          value={formatCurrency(s.financial_overview.cash_inflow, { compact: true })}
          icon={ArrowDownLeft}
          iconClass="text-emerald-600"
          bg="bg-emerald-50 dark:bg-emerald-950/30"
          subtext="Penerimaan POS & faktur lunas"
        />
        <StatCard
          label="Valuasi Stok"
          value={formatCurrency(s.wms.total_stock_value, { compact: true })}
          icon={Package}
          iconClass="text-blue-600"
          bg="bg-blue-50 dark:bg-blue-950/30"
          subtext={`${s.wms.total_physical_units} unit di ${s.wms.total_warehouses} gudang`}
        />
        <StatCard
          label="Barang Keluar Hari Ini"
          value={`${s.outbound.qty_today} unit`}
          icon={Truck}
          iconClass="text-indigo-600"
          bg="bg-indigo-50 dark:bg-indigo-950/30"
          subtext={`Bulan ini: ${s.outbound.qty_month} unit`}
        />
      </div>

      <div className={`grid grid-cols-1 lg:grid-cols-2 gap-4 ${loading ? 'opacity-50' : ''}`}>
        <section className="border border-border/60 rounded-lg bg-card">
          <SectionHeader icon={Store} title="POS Kasir & Penjualan" href="/pos" linkLabel="Buka POS Kasir →" />
          <div className="p-5 space-y-4">
            <div className="grid grid-cols-3 gap-2.5">
              <MiniStat label="Omzet Hari Ini" value={formatCurrency(s.pos.today_revenue, { compact: true })} />
              <MiniStat label="Transaksi Hari Ini" value={s.pos.today_orders_count} />
              <MiniStat label="Rata-rata Keranjang" value={formatCurrency(s.pos.average_basket_size, { compact: true })} />
            </div>
            <div className="grid grid-cols-3 gap-2.5">
              <MiniStat label="Sales Order" value={s.sales_orders.total} />
              <MiniStat label="Order Pending" value={s.sales_orders.pending} />
              <MiniStat label="Pelanggan" value={s.customers.active} />
            </div>
            <div>
              <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70 mb-1.5">
                Transaksi Terakhir
              </div>
              {s.pos.recent_orders.length > 0 ? (
                <ul className="divide-y divide-border/40 border border-border/50 rounded-lg overflow-hidden">
                  {s.pos.recent_orders.map((order) => {
                    const time = formatTime(order.created_at)
                    return (
                      <li key={order.order_number} className="px-3 py-2 flex items-center justify-between gap-2 text-[12px]">
                        <div className="min-w-0">
                          <div className="flex items-center gap-1.5">
                            <span className="font-mono text-foreground font-medium truncate">{order.order_number}</span>
                            {time && <span className="text-[10px] text-muted-foreground tabular-nums shrink-0">{time}</span>}
                          </div>
                          <div className="text-[10px] text-muted-foreground truncate">{order.customer_name}</div>
                        </div>
                        <div className="text-right shrink-0 flex items-center gap-2">
                          <span className="px-1.5 py-0.5 rounded text-[10px] font-medium uppercase bg-muted text-muted-foreground border border-border/50">
                            {order.payment_method}
                          </span>
                          <span className="font-mono font-semibold text-foreground tabular-nums">{formatCurrency(order.total_amount)}</span>
                        </div>
                      </li>
                    )
                  })}
                </ul>
              ) : (
                <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">
                  Belum ada transaksi POS tercatat.
                </div>
              )}
            </div>
          </div>
        </section>

        <section className="border border-border/60 rounded-lg bg-card">
          <SectionHeader icon={Warehouse} title="Gudang & Monitoring Stok" href="/wms" linkLabel="Kelola Gudang" />
          <div className="p-5 space-y-4">
            <div className="grid grid-cols-3 gap-2.5">
              <MiniStat label="Gudang / Lokasi" value={`${s.wms.total_warehouses} / ${s.wms.total_locations}`} />
              <MiniStat label="SKU Aktif" value={s.wms.total_skus} />
              <MiniStat
                label="Mutasi Hari Ini"
                value={
                  <span className="inline-flex items-center gap-1">
                    <ArrowLeftRight className="h-3 w-3 text-muted-foreground" />
                    {s.wms.today_movements}
                  </span>
                }
              />
            </div>
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-[10px] font-semibold uppercase tracking-wider text-amber-600 dark:text-amber-400 flex items-center gap-1">
                  <AlertTriangle className="h-3 w-3" /> Stok Menipis (≤ 5 unit)
                </span>
                <Link href="/products" className="text-[10px] font-medium text-muted-foreground hover:text-primary">
                  Master Produk →
                </Link>
              </div>
              {s.wms.low_stock_items.length > 0 ? (
                <ul className="space-y-1.5">
                  {s.wms.low_stock_items.map((item) => (
                    <li
                      key={item.sku}
                      className="flex items-center justify-between gap-2 px-3 py-2 rounded-lg bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-900/40 text-[12px]"
                    >
                      <div className="min-w-0">
                        <div className="font-medium text-foreground truncate">{item.name}</div>
                        <div className="font-mono text-[10px] text-muted-foreground">
                          {item.sku} • Min: {item.min_threshold}
                        </div>
                      </div>
                      <span className="px-2 py-0.5 rounded-full text-[11px] font-bold bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300 tabular-nums shrink-0">
                        Sisa {item.current_stock}
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">
                  Tidak ada SKU di bawah batas stok aman.
                </div>
              )}
            </div>
          </div>
        </section>
      </div>

      <section className={`border border-border/60 rounded-lg bg-card ${loading ? 'opacity-50' : ''}`}>
        <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between flex-wrap gap-2">
          <div className="flex items-center gap-2">
            <Truck className="h-4 w-4 text-indigo-600" />
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
              Aktivitas Barang Keluar & Pemenuhan (WMS Outbound)
            </h2>
          </div>
          <div className="flex items-center gap-2">
            <div className="inline-flex rounded-lg border border-border/60 p-0.5 bg-muted/40 text-[11px]">
              {(['7d', '30d', '90d'] as const).map((p) => (
                <button
                  key={p}
                  type="button"
                  onClick={() => setPeriod(p)}
                  className={`px-2.5 py-0.5 rounded-md font-medium transition ${
                    period === p
                      ? 'bg-background text-foreground shadow-xs'
                      : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {p === '7d' ? '7 Hari' : p === '30d' ? '30 Hari' : '90 Hari'}
                </button>
              ))}
            </div>
            <Link
              href="/wms/arus-barang"
              className="text-[11px] font-medium text-primary hover:underline flex items-center gap-1"
            >
              Buka Arus Barang →
            </Link>
          </div>
        </div>

        <div className="p-5 space-y-4">
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2.5">
            <MiniStat label="Keluar Hari Ini" value={`${s.outbound.qty_today} unit`} />
            <MiniStat label="Keluar Bulan Ini" value={`${s.outbound.qty_month} unit`} />
            <MiniStat label="Total Surat Jalan" value={s.sales_orders.total} />
            <MiniStat label="DO Dikonfirmasi" value={s.sales_orders.confirmed} />
            <MiniStat label="DO Pending" value={s.sales_orders.pending} />
            <MiniStat label="Pelanggan Aktif" value={s.customers.active} />
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 pt-2">
            {/* Top 10 Produk Keluar */}
            <div className="space-y-1.5">
              <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70 flex items-center justify-between">
                <span>Top 10 Produk Keluar ({period === '7d' ? '7 Hari' : period === '30d' ? '30 Hari' : '90 Hari'})</span>
                <span>Jumlah Fisik</span>
              </div>
              {s.outbound.top_products.length > 0 ? (
                <ul className="divide-y divide-border/40 border border-border/50 rounded-lg overflow-hidden max-h-60 overflow-y-auto">
                  {s.outbound.top_products.map((p, idx) => (
                    <li key={p.product_id || idx} className="px-3 py-2 flex items-center justify-between gap-2 text-[12px] hover:bg-muted/30">
                      <div className="min-w-0">
                        <div className="font-medium text-foreground truncate">{p.product_name || 'Produk Tanpa Nama'}</div>
                        <div className="text-[10px] text-muted-foreground font-mono">{p.product_sku || '-'}</div>
                      </div>
                      <span className="font-mono font-bold text-indigo-600 tabular-nums shrink-0">
                        {p.quantity} unit
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">
                  Belum ada transaksi barang keluar pada periode ini.
                </div>
              )}
            </div>

            {/* Top 10 Pelanggan Teraktif */}
            <div className="space-y-1.5">
              <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70 flex items-center justify-between">
                <span>Top 10 Pelanggan Teraktif ({period === '7d' ? '7 Hari' : period === '30d' ? '30 Hari' : '90 Hari'})</span>
                <span>Nilai Transaksi</span>
              </div>
              {s.outbound.top_customers.length > 0 ? (
                <ul className="divide-y divide-border/40 border border-border/50 rounded-lg overflow-hidden max-h-60 overflow-y-auto">
                  {s.outbound.top_customers.map((c, idx) => (
                    <li key={c.customer_id || idx} className="px-3 py-2 flex items-center justify-between gap-2 text-[12px] hover:bg-muted/30">
                      <div className="min-w-0">
                        <div className="font-medium text-foreground truncate">{c.customer_name || 'Pelanggan'}</div>
                        <div className="text-[10px] text-muted-foreground">{c.order_count} pesanan</div>
                      </div>
                      <span className="font-mono font-bold text-emerald-600 tabular-nums shrink-0">
                        {formatCurrency(c.total_revenue)}
                      </span>
                    </li>
                  ))}
                </ul>
              ) : (
                <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">
                  Belum ada transaksi pelanggan pada periode ini.
                </div>
              )}
            </div>
          </div>
        </div>
      </section>
    </div>
  )
}
