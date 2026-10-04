'use client'

import { useDashboardSummary } from '@/lib/queries/dashboard'
import { useInvoices } from '@/lib/queries/invoices'
import Link from 'next/link'
import { TooltipWalkthrough } from '@/components/tutorial/TooltipWalkthrough'
import {
  FileText,
  CheckCircle,
  Clock,
  XCircle,
  CreditCard,
  Building2,
  ShoppingCart,
  Package,
  TrendingUp,
  AlertTriangle,
  Store,
  Warehouse,
  Users,
  ExternalLink,
  RefreshCw,
} from 'lucide-react'
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from 'recharts'
import type { DashboardSummary, MonthlyTrend, TopVendor } from '@/lib/schemas/dashboard'
import type { InvoiceStatus } from '@/lib/schemas/invoice'
import { formatCurrency } from '@/lib/currency'

// ── Helpers ──────────────────────────────────────────────────────────────────

function formatCompact(value: string | number): string {
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (Number.isNaN(num)) return '0'
  if (num >= 1_000_000_000) return `${(num / 1_000_000_000).toFixed(1)}M`
  if (num >= 1_000_000) return `${(num / 1_000_000).toFixed(1)}Jt`
  if (num >= 1_000) return `${(num / 1_000).toFixed(1)}K`
  return num.toString()
}

function countByStatus(invoices: { status: InvoiceStatus }[], status: InvoiceStatus) {
  return invoices.filter((inv) => inv.status === status).length
}

// ── Stat Card ────────────────────────────────────────────────────────────────

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
    <div className="border border-border/60 rounded-lg p-4 flex items-center gap-3">
      <div className={`h-9 w-9 rounded-lg flex items-center justify-center shrink-0 ${bg}`}>
        <Icon className={`h-4 w-4 ${iconClass}`} />
      </div>
      <div className="min-w-0">
        <div className="text-xl font-bold tracking-tight text-foreground">{value}</div>
        <div className="text-[11px] text-muted-foreground leading-tight">{label}</div>
        {subtext && <div className="text-[10px] text-muted-foreground/70 mt-0.5 truncate">{subtext}</div>}
      </div>
    </div>
  )
}

// ── Status Badge ─────────────────────────────────────────────────────────────

function StatusBadge({ status, count }: { status: string; count: number }) {
  const config: Record<string, { bg: string; text: string; label: string }> = {
    pending: { bg: 'bg-amber-50', text: 'text-amber-700', label: 'Menunggu' },
    approved: { bg: 'bg-emerald-50', text: 'text-emerald-700', label: 'Disetujui' },
    rejected: { bg: 'bg-rose-50', text: 'text-rose-700', label: 'Ditolak' },
    pending_review: { bg: 'bg-blue-50', text: 'text-blue-700', label: 'Review AI' },
  }
  const c = config[status] ?? { bg: 'bg-zinc-50', text: 'text-zinc-700', label: status }
  return (
    <div className={`${c.bg} ${c.text} rounded-lg px-3 py-2.5 text-center`}>
      <div className="text-lg font-bold">{count}</div>
      <div className="text-[11px] mt-0.5">{c.label}</div>
    </div>
  )
}

// ── Pie chart colors ─────────────────────────────────────────────────────────

const PIE_COLORS = ['#f59e0b', '#10b981', '#ef4444', '#3b82f6']

// ── Loading Skeleton ─────────────────────────────────────────────────────────

function DashboardSkeleton() {
  return (
    <div className="px-6 py-6 space-y-6 animate-pulse">
      <div>
        <div className="h-5 w-36 bg-zinc-200 rounded-md" />
        <div className="h-3 w-52 bg-zinc-100 rounded-md mt-1.5" />
      </div>
      <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-6 gap-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="border border-border/60 rounded-lg p-4 h-20" />
        ))}
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="border border-border/60 rounded-lg h-64" />
        <div className="border border-border/60 rounded-lg h-64" />
      </div>
      <div className="border border-border/60 rounded-lg h-52" />
    </div>
  )
}

// ── Error State ──────────────────────────────────────────────────────────────

function DashboardError({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="px-6 py-6">
      <div className="border border-border/60 rounded-lg p-8 text-center max-w-sm mx-auto">
        <AlertTriangle className="h-5 w-5 text-amber-500 mx-auto mb-3" />
        <h2 className="text-sm font-semibold text-foreground mb-1">Gagal memuat data dashboard</h2>
        <p className="text-xs text-muted-foreground mb-4">
          Terjadi kesalahan saat mengambil data dari server.
        </p>
        <button
          onClick={onRetry}
          className="px-4 py-1.5 bg-primary text-primary-foreground text-xs font-medium rounded-md hover:bg-primary/90 transition-colors"
        >
          Coba Lagi
        </button>
      </div>
    </div>
  )
}

// ── Custom Tooltip for Bar Chart ─────────────────────────────────────────────

interface TooltipPayloadItem {
  value: number
  dataKey: string
  payload: MonthlyTrend
}

function BarChartTooltip({ active, payload, label }: { active?: boolean; payload?: TooltipPayloadItem[]; label?: string }) {
  if (!active || !payload?.length) return null
  return (
    <div className="bg-white border border-zinc-200 rounded-xl shadow-lg p-3 text-sm">
      <div className="font-semibold text-zinc-800 mb-1">{label}</div>
      {payload.map((item) => (
        <div key={item.dataKey} className="text-zinc-600">
          {item.dataKey === 'invoice_count' ? (
            <>
              Invoice: <span className="font-medium text-zinc-800">{item.value}</span>
            </>
          ) : (
            <>
              Total: <span className="font-medium text-zinc-800">{formatCompact(item.value)}</span>
            </>
          )}
        </div>
      ))}
    </div>
  )
}

// ── Main Dashboard Page ──────────────────────────────────────────────────────

export default function DashboardPage() {
  const { data: summary, isLoading, error, refetch } = useDashboardSummary()
  const { data: invoiceList, isLoading: invoicesLoading } = useInvoices()
  const invoices = invoiceList?.data ?? []

  // Fallback stats from invoice list if summary is loading
  const total = summary?.invoices.total ?? invoiceList?.total ?? 0
  const pending = summary?.invoices.pending ?? (invoices.length > 0 ? countByStatus(invoices, 'pending') : 0)
  const approved = summary?.invoices.approved ?? (invoices.length > 0 ? countByStatus(invoices, 'approved') : 0)
  const rejected = summary?.invoices.rejected ?? (invoices.length > 0 ? countByStatus(invoices, 'rejected') : 0)

  if (error && !summary && !isLoading) {
    return <DashboardError onRetry={() => void refetch()} />
  }

  return (
    <div className="px-6 py-6 space-y-6">
      <TooltipWalkthrough />

      {/* Page header */}
      <div className="flex items-start justify-between gap-3" data-tutorial="dashboard-header">
        <div>
          <h1 className="text-lg font-bold tracking-tight text-foreground">Dashboard</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            Konsolidasi metrik kasir (POS), pergudangan (WMS), dan pembelian vendor.
          </p>
        </div>
        <button
          type="button"
          onClick={() => void refetch()}
          disabled={isLoading}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 border border-border/60 rounded-lg text-xs font-medium text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors disabled:opacity-50 shrink-0"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${isLoading ? 'animate-spin' : ''}`} />
          Segarkan
        </button>
      </div>

      {/* ── Stats Grid ────────────────────────────────────────────────────── */}
      {isLoading && !summary ? (
        <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-7 gap-3">
          {Array.from({ length: 7 }).map((_, i) => (
            <div key={i} className="border border-border/60 rounded-lg p-4 h-20 animate-pulse" />
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-7 gap-3">
          <StatCard
            label="Omzet Penjualan (POS)"
            value={summary ? formatCurrency(summary.pos.total_revenue, { compact: true }) : 0}
            icon={TrendingUp}
            iconClass="text-emerald-600"
            bg="bg-emerald-50"
            subtext={summary ? `Piutang (AR): ${formatCurrency(summary.sales_invoices.accounts_receivable, { compact: true })}` : undefined}
          />
          <StatCard
            label="Total Invoice Vendor"
            value={total}
            icon={FileText}
            iconClass="text-zinc-600"
            bg="bg-zinc-100"
            subtext={summary ? `Total: ${formatCurrency(summary.invoices.total_amount)}` : undefined}
          />
          <StatCard
            label="Disetujui"
            value={approved}
            icon={CheckCircle}
            iconClass="text-zinc-600"
            bg="bg-zinc-100"
            subtext={summary ? formatCurrency(summary.invoices.approved_amount) : undefined}
          />
          <StatCard
            label="Menunggu Persetujuan"
            value={pending}
            icon={Clock}
            iconClass="text-zinc-600"
            bg="bg-zinc-100"
          />
          <StatCard
            label="Vendor Aktif"
            value={summary?.vendors.active ?? 0}
            icon={Building2}
            iconClass="text-zinc-600"
            bg="bg-zinc-100"
          />
          <StatCard
            label="Purchase Orders"
            value={summary?.purchase_orders.total ?? 0}
            icon={ShoppingCart}
            iconClass="text-zinc-600"
            bg="bg-zinc-100"
          />
          <StatCard
            label="Goods Receipts"
            value={summary?.goods_receipts.total ?? 0}
            icon={Package}
            iconClass="text-zinc-600"
            bg="bg-zinc-100"
          />
        </div>
      )}

      {/* ── POS Kasir & WMS Pergudangan ───────────────────────────────────── */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Aktivitas Kasir POS */}
        <div className="border border-border/60 rounded-lg">
          <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Store className="h-3.5 w-3.5 text-muted-foreground" />
              <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Aktivitas Kasir Hari Ini</h2>
            </div>
            <Link href="/pos" className="text-[11px] font-medium text-primary hover:underline flex items-center gap-1">
              Buka POS <ExternalLink className="h-3 w-3" />
            </Link>
          </div>

          {summary ? (
            <div className="p-5 space-y-4">
              <div className="grid grid-cols-3 gap-2.5">
                <div className="bg-muted/40 rounded-lg p-2.5 text-center">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Omzet Hari Ini</div>
                  <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                    {formatCurrency(summary.pos.today_revenue, { compact: true })}
                  </div>
                </div>
                <div className="bg-muted/40 rounded-lg p-2.5 text-center">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Struk</div>
                  <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                    {summary.pos.today_orders_count}
                  </div>
                </div>
                <div className="bg-muted/40 rounded-lg p-2.5 text-center">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Pelanggan</div>
                  <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                    {summary.customers.active}
                  </div>
                </div>
              </div>

              <div>
                <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70 mb-1.5">
                  Transaksi Terakhir
                </div>
                {summary.pos.recent_orders.length > 0 ? (
                  <ul className="divide-y divide-border/40 border border-border/50 rounded-lg overflow-hidden">
                    {summary.pos.recent_orders.map((order) => (
                      <li key={order.order_number} className="px-3 py-2 flex items-center justify-between gap-2 text-[12px]">
                        <div className="min-w-0">
                          <div className="font-mono text-foreground font-medium truncate">{order.order_number}</div>
                          <div className="text-[10px] text-muted-foreground truncate">{order.customer_name}</div>
                        </div>
                        <div className="text-right shrink-0">
                          <div className="font-mono font-semibold text-foreground tabular-nums">
                            {formatCurrency(order.total_amount)}
                          </div>
                          <div className="text-[10px] text-muted-foreground">{order.payment_method}</div>
                        </div>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">
                    Belum ada transaksi POS tercatat.
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="px-5 py-8 text-center text-xs text-muted-foreground">
              {isLoading ? 'Memuat...' : 'Belum ada data kasir.'}
            </div>
          )}
        </div>

        {/* Pergudangan & Peringatan Stok */}
        <div className="border border-border/60 rounded-lg">
          <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Warehouse className="h-3.5 w-3.5 text-muted-foreground" />
              <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Pergudangan &amp; Stok</h2>
            </div>
            <Link href="/wms" className="text-[11px] font-medium text-primary hover:underline flex items-center gap-1">
              Kelola Gudang <ExternalLink className="h-3 w-3" />
            </Link>
          </div>

          {summary ? (
            <div className="p-5 space-y-4">
              <div className="grid grid-cols-3 gap-2.5">
                <div className="bg-muted/40 rounded-lg p-2.5 text-center">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Gudang Aktif</div>
                  <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                    {summary.wms.total_warehouses}
                  </div>
                </div>
                <div className="bg-muted/40 rounded-lg p-2.5 text-center">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-wide">SKU Aktif</div>
                  <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                    {summary.wms.total_skus}
                  </div>
                </div>
                <div className="bg-muted/40 rounded-lg p-2.5 text-center">
                  <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Mutasi Hari Ini</div>
                  <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                    {summary.wms.today_movements}
                  </div>
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <span className="text-[10px] font-semibold uppercase tracking-wider text-amber-600 flex items-center gap-1">
                    <AlertTriangle className="h-3 w-3" /> Stok Menipis (&le; 5 unit)
                  </span>
                  <Link href="/products" className="text-[10px] font-medium text-muted-foreground hover:text-primary">
                    Master Produk &rarr;
                  </Link>
                </div>
                {summary.wms.low_stock_items.length > 0 ? (
                  <ul className="space-y-1.5">
                    {summary.wms.low_stock_items.map((item) => (
                      <li
                        key={item.sku}
                        className="flex items-center justify-between gap-2 px-3 py-2 rounded-lg bg-amber-50 dark:bg-amber-950/20 border border-amber-200 dark:border-amber-900/40 text-[12px]"
                      >
                        <div className="min-w-0">
                          <div className="font-medium text-foreground truncate">{item.name}</div>
                          <div className="font-mono text-[10px] text-muted-foreground">{item.sku}</div>
                        </div>
                        <span className="shrink-0 px-2 py-0.5 rounded-full text-[11px] font-bold bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300 tabular-nums">
                          Sisa {item.current_stock}
                        </span>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">
                    Seluruh SKU berada di atas ambang batas aman.
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="px-5 py-8 text-center text-xs text-muted-foreground">
              {isLoading ? 'Memuat...' : 'Belum ada data gudang.'}
            </div>
          )}
        </div>
      </div>

      {/* ── Invoice Status Breakdown + Payment Pipeline ───────────────────── */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Invoice Status Breakdown (Pie Chart) */}
        <div className="border border-border/60 rounded-lg">
          <div className="px-5 py-3 border-b border-border/60">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Status Invoice</h2>
          </div>
          {summary && summary.invoices.total > 0 ? (
            <div className="p-5 flex items-center gap-5">
              <div className="w-40 h-40 shrink-0">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={[
                        { name: 'Menunggu', value: summary.invoices.pending },
                        { name: 'Disetujui', value: summary.invoices.approved },
                        { name: 'Ditolak', value: summary.invoices.rejected },
                        { name: 'Review AI', value: summary.invoices.pending_review },
                      ]}
                      cx="50%"
                      cy="50%"
                    innerRadius={36}
                    outerRadius={60}
                      paddingAngle={3}
                      dataKey="value"
                    >
                      {PIE_COLORS.map((color) => (
                        <Cell key={color} fill={color} />
                      ))}
                    </Pie>
                    <Tooltip
                      formatter={(value, name) => [`${value}`, String(name)]}
                    />
                  </PieChart>
                </ResponsiveContainer>
              </div>
              <div className="grid grid-cols-2 gap-2.5 flex-1">
                <StatusBadge status="pending" count={summary.invoices.pending} />
                <StatusBadge status="approved" count={summary.invoices.approved} />
                <StatusBadge status="rejected" count={summary.invoices.rejected} />
                <StatusBadge status="pending_review" count={summary.invoices.pending_review} />
              </div>
            </div>
          ) : (
            <div className="px-5 py-8 text-center text-xs text-muted-foreground">
              {isLoading ? 'Memuat...' : 'Belum ada data invoice.'}
            </div>
          )}
        </div>

        {/* Payment Pipeline */}
        <div className="border border-border/60 rounded-lg">
          <div className="px-5 py-3 border-b border-border/60">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Pipeline Pembayaran</h2>
          </div>
          {summary ? (
            <div className="p-5 space-y-3.5">
              <div className="flex items-center justify-between">
                <span className="text-xs text-muted-foreground">Total Payment Orders</span>
                <span className="text-base font-bold text-foreground">{summary.payments.total}</span>
              </div>
              <div className="w-full bg-zinc-100 rounded-full h-2 overflow-hidden">
                <div
                  className="bg-emerald-500 h-full rounded-full transition-all"
                  style={{
                    width: summary.payments.total > 0
                      ? `${(summary.payments.paid / summary.payments.total) * 100}%`
                      : '0%',
                  }}
                />
              </div>
              <div className="flex justify-between text-xs text-muted-foreground">
                <span>{summary.payments.paid} dibayar</span>
                <span>{summary.payments.total - summary.payments.paid} pending</span>
              </div>
              <div className="grid grid-cols-2 gap-3 pt-1">
                <div className="rounded-lg p-3">
                  <div className="text-[10px] uppercase tracking-wide text-emerald-600 mb-0.5">Sudah Dibayar</div>
                  <div className="text-sm font-bold text-foreground">{formatCurrency(summary.payments.paid_amount)}</div>
                </div>
                <div className="rounded-lg p-3">
                  <div className="text-[10px] uppercase tracking-wide text-amber-600 mb-0.5">Menunggu Pembayaran</div>
                  <div className="text-sm font-bold text-foreground">{formatCurrency(summary.payments.pending_amount)}</div>
                </div>
              </div>
            </div>
          ) : (
            <div className="px-5 py-8 text-center text-xs text-muted-foreground">
              {isLoading ? 'Memuat...' : 'Belum ada data pembayaran.'}
            </div>
          )}
        </div>
      </div>

      {/* ── Monthly Invoice Trend (Bar Chart) ─────────────────────────────── */}
      {summary && summary.monthly_trend.length > 0 && (
        <div className="border border-border/60 rounded-lg">
          <div className="px-5 py-3 border-b border-border/60">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Trend Invoice 6 Bulan Terakhir</h2>
          </div>
          <div className="p-5">
            <ResponsiveContainer width="100%" height={280}>
              <BarChart data={summary.monthly_trend} barGap={4}>
                <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" vertical={false} />
                <XAxis
                  dataKey="month"
                  tick={{ fontSize: 11, fill: '#a3a3a3' }}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: '#a3a3a3' }}
                  axisLine={false}
                  tickLine={false}
                  tickFormatter={formatCompact}
                />
                <Tooltip content={<BarChartTooltip />} />
                <Bar
                  dataKey="invoice_count"
                  name="Jumlah Invoice"
                  fill="#1e6b58"
                  radius={[3, 3, 0, 0]}
                  maxBarSize={40}
                />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* ── Top 5 Vendors Table ───────────────────────────────────────────── */}
      {summary && summary.top_vendors.length > 0 && (
        <div className="border border-border/60 rounded-lg">
          <div className="px-5 py-3 border-b border-border/60">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Top 5 Vendor Berdasarkan Nilai Invoice</h2>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border/60">
                  <th className="text-left px-5 py-2.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">#</th>
                  <th className="text-left px-5 py-2.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">Vendor</th>
                  <th className="text-right px-5 py-2.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">Invoice</th>
                  <th className="text-right px-5 py-2.5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70">Total Nilai</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/40">
                {summary.top_vendors.map((vendor, idx) => (
                  <tr key={vendor.vendor_id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-5 py-2.5 text-muted-foreground font-mono text-[11px]">{idx + 1}</td>
                    <td className="px-5 py-2.5">
                      <div className="font-medium text-foreground text-[13px]">{vendor.vendor_name}</div>
                      <div className="text-[10px] text-muted-foreground/60 font-mono">{vendor.vendor_id}</div>
                    </td>
                    <td className="px-5 py-2.5 text-right text-muted-foreground tabular-nums">{vendor.invoice_count}</td>
                    <td className="px-5 py-2.5 text-right font-medium text-foreground tabular-nums">{formatCurrency(vendor.total_amount)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ── Quick links ───────────────────────────────────────────────────── */}
      <div className="flex items-center gap-3" data-tutorial="create-invoice">
        <Link href="/dashboard/payment-orders" className="inline-flex items-center gap-2 rounded-lg border border-border/60 px-3.5 py-2 text-xs font-medium text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors">
          <CreditCard className="h-3.5 w-3.5" />
          Payment Orders
        </Link>
        <Link href="/dashboard/vendors" className="inline-flex items-center gap-2 rounded-lg border border-border/60 px-3.5 py-2 text-xs font-medium text-muted-foreground hover:bg-muted/50 hover:text-foreground transition-colors">
          <Building2 className="h-3.5 w-3.5" />
          Vendors
        </Link>
      </div>

      {/* ── Recent invoices teaser (fallback from list API) ───────────────── */}
      <div className="border border-border/60 rounded-lg">
        <div className="px-5 py-3 border-b border-border/60">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Invoice Terbaru</h2>
        </div>
        {invoicesLoading ? (
          <div className="p-5 space-y-2.5">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-7 bg-zinc-100 rounded-md animate-pulse" />
            ))}
          </div>
        ) : invoices.length === 0 ? (
          <div className="px-5 py-8 text-center text-xs text-muted-foreground">
            Belum ada invoice tersedia.
          </div>
        ) : (
          <ul className="divide-y divide-border/40">
            {invoices.slice(0, 5).map((inv) => (
              <li key={inv.id} className="px-5 py-2.5 flex items-center justify-between text-[13px]">
                <span className="font-mono text-muted-foreground">{inv.invoice_number}</span>
                <span className="font-medium text-foreground tabular-nums">{formatCurrency(inv.amount)}</span>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
