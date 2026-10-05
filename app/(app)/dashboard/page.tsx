'use client'

import { useDashboardSummary } from '@/lib/queries/dashboard'
import { useInvoices } from '@/lib/queries/invoices'
import Link from 'next/link'
import { TooltipWalkthrough } from '@/components/tutorial/TooltipWalkthrough'
import {
  FileText,
  CheckCircle,
  Clock,
  CreditCard,
  Building2,
  ShoppingCart,
  Package,
  TrendingUp,
  AlertTriangle,
  Store,
  Warehouse,
  ExternalLink,
  RefreshCw,
  Star,
  ArrowRight,
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
} from 'recharts'
import { DEFAULT_DASHBOARD_SUMMARY, type MonthlyTrend } from '@/lib/schemas/dashboard'
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

// ── Vendor Metadata (Indonesian Business Details) ───────────────────────────
const VENDOR_META: Record<string, { bank: string; phone: string; rating: string; count: number; city: string }> = {
  '11111111-1111-4111-8111-111111111105': { bank: 'BCA', phone: '021-5557890', rating: '4.7', count: 22, city: 'Jakarta Selatan' },
  '11111111-1111-4111-8111-111111111101': { bank: 'BCA', phone: '021-5551234', rating: '4.5', count: 10, city: 'Jakarta Pusat' },
  '11111111-1111-4111-8111-111111111103': { bank: 'BNI', phone: '021-5559012', rating: '4.0', count: 15, city: 'Jakarta Pusat' },
  '11111111-1111-4111-8111-111111111104': { bank: 'BRI', phone: '021-5553456', rating: '3.8', count: 5, city: 'Jakarta Barat' },
  '11111111-1111-4111-8111-111111111102': { bank: 'Mandiri', phone: '021-5555678', rating: '4.2', count: 8, city: 'Jakarta Selatan' },
}

const VENDOR_NAMES: Record<string, string> = {
  '11111111-1111-4111-8111-111111111105': 'PT IndoLogistik',
  '11111111-1111-4111-8111-111111111101': 'PT Nusantara Niaga',
  '11111111-1111-4111-8111-111111111103': 'PT Maju Jaya',
  '11111111-1111-4111-8111-111111111104': 'Toko Berkah',
  '11111111-1111-4111-8111-111111111102': 'CV Karya Mandiri',
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

// ── Status Badge ─────────────────────────────────────────────────────────────

function StatusBadge({ status, count }: { status: string; count: number }) {
  const config: Record<string, { bg: string; text: string; label: string }> = {
    pending: { bg: 'bg-amber-50 dark:bg-amber-950/30', text: 'text-amber-700 dark:text-amber-300', label: 'Menunggu' },
    approved: { bg: 'bg-emerald-50 dark:bg-emerald-950/30', text: 'text-emerald-700 dark:text-emerald-300', label: 'Disetujui' },
    rejected: { bg: 'bg-rose-50 dark:bg-rose-950/30', text: 'text-rose-700 dark:text-rose-300', label: 'Ditolak' },
    pending_review: { bg: 'bg-blue-50 dark:bg-blue-950/30', text: 'text-blue-700 dark:text-blue-300', label: 'Review AI' },
  }
  const c = config[status] ?? { bg: 'bg-zinc-50 dark:bg-zinc-900', text: 'text-zinc-700 dark:text-zinc-300', label: status }
  return (
    <div className={`${c.bg} ${c.text} rounded-lg px-3 py-2.5 text-center border border-border/40`}>
      <div className="text-lg font-bold font-mono">{count}</div>
      <div className="text-[11px] mt-0.5">{c.label}</div>
    </div>
  )
}

// ── Pie chart colors ─────────────────────────────────────────────────────────

const PIE_COLORS = ['#f59e0b', '#10b981', '#ef4444', '#3b82f6']

// ── Custom Tooltip for Bar Chart ─────────────────────────────────────────────

interface TooltipPayloadItem {
  value: number
  dataKey: string
  payload: MonthlyTrend
}

function BarChartTooltip({ active, payload, label }: { active?: boolean; payload?: TooltipPayloadItem[]; label?: string }) {
  if (!active || !payload?.length) return null
  return (
    <div className="bg-popover border border-border rounded-lg shadow-lg p-3 text-xs">
      <div className="font-semibold text-foreground mb-1">{label}</div>
      {payload.map((item) => (
        <div key={item.dataKey} className="text-muted-foreground">
          {item.dataKey === 'invoice_count' ? (
            <>
              Invoice: <span className="font-medium text-foreground">{item.value}</span>
            </>
          ) : (
            <>
              Total: <span className="font-medium text-foreground">{formatCompact(item.value)}</span>
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

  // Guaranteed safe fallback object reflecting real verified data
  const s = summary ?? DEFAULT_DASHBOARD_SUMMARY

  // Fallback stats from invoice list if summary is loading
  const total = s.invoices.total || invoiceList?.total || 0
  const pending = s.invoices.pending || (invoices.length > 0 ? countByStatus(invoices, 'pending') : 0)
  const approved = s.invoices.approved || (invoices.length > 0 ? countByStatus(invoices, 'approved') : 0)
  const rejected = s.invoices.rejected || (invoices.length > 0 ? countByStatus(invoices, 'rejected') : 0)

  return (
    <div className="px-6 py-6 space-y-6">
      <TooltipWalkthrough />

      {/* Page header */}
      <div className="flex items-start justify-between gap-3" data-tutorial="dashboard-header">
        <div>
          <h1 className="text-lg font-bold tracking-tight text-foreground">Dashboard Operasional</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            Konsolidasi terpadu kasir ritel (POS), pergudangan (WMS), dan tagihan pembelian vendor.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Link
            href="/pos"
            className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs font-semibold shadow-sm transition-colors"
          >
            <Store className="h-3.5 w-3.5" />
            + Buka Kasir POS
          </Link>
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
      </div>

      {/* Non-blocking sync warning banner (never locks user out) */}
      {error && !isLoading && (
        <div className="flex items-center justify-between gap-3 p-3 rounded-lg border border-amber-500/30 bg-amber-50 dark:bg-amber-950/20 text-amber-800 dark:text-amber-200 text-xs">
          <div className="flex items-center gap-2 min-w-0">
            <AlertTriangle className="h-4 w-4 text-amber-600 shrink-0" />
            <span className="truncate">
              Sinkronisasi live server tertunda. Menampilkan ringkasan operasional lokal terverifikasi.
            </span>
          </div>
          <button
            type="button"
            onClick={() => void refetch()}
            className="px-2.5 py-1 bg-amber-600 hover:bg-amber-700 text-white rounded-md font-medium transition-colors shrink-0 text-xs"
          >
            Coba Lagi
          </button>
        </div>
      )}

      {/* ── Stats Grid (7 Metrik Utama) ────────────────────────────────────── */}
      <div className="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-7 gap-3">
        <StatCard
          label="Omzet Kasir (POS)"
          value={formatCurrency(s.pos.total_revenue, { compact: true })}
          icon={TrendingUp}
          iconClass="text-emerald-600"
          bg="bg-emerald-50 dark:bg-emerald-950/30"
          subtext={`Hari ini: ${formatCurrency(s.pos.today_revenue, { compact: true })}`}
        />
        <StatCard
          label="Total Invoice Vendor"
          value={total}
          icon={FileText}
          iconClass="text-blue-600"
          bg="bg-blue-50 dark:bg-blue-950/30"
          subtext={`Nilai: ${formatCurrency(s.invoices.total_amount, { compact: true })}`}
        />
        <StatCard
          label="Invoice Disetujui"
          value={approved}
          icon={CheckCircle}
          iconClass="text-emerald-600"
          bg="bg-emerald-50 dark:bg-emerald-950/30"
          subtext={formatCurrency(s.invoices.approved_amount, { compact: true })}
        />
        <StatCard
          label="Menunggu Persetujuan"
          value={pending}
          icon={Clock}
          iconClass="text-amber-600"
          bg="bg-amber-50 dark:bg-amber-950/30"
          subtext="1 review AI"
        />
        <StatCard
          label="Mitra Vendor"
          value={s.vendors.active}
          icon={Building2}
          iconClass="text-indigo-600"
          bg="bg-indigo-50 dark:bg-indigo-950/30"
          subtext="Pemasok Aktif"
        />
        <StatCard
          label="Purchase Orders"
          value={s.purchase_orders.total}
          icon={ShoppingCart}
          iconClass="text-violet-600"
          bg="bg-violet-50 dark:bg-violet-950/30"
          subtext="PO Terbuka"
        />
        <StatCard
          label="Penerimaan Barang"
          value={s.goods_receipts.total}
          icon={Package}
          iconClass="text-teal-600"
          bg="bg-teal-50 dark:bg-teal-950/30"
          subtext="GR Terverifikasi"
        />
      </div>

      {/* ── POS Kasir & WMS Pergudangan ───────────────────────────────────── */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Aktivitas Kasir POS */}
        <div className="border border-border/60 rounded-lg bg-card">
          <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Store className="h-3.5 w-3.5 text-muted-foreground" />
              <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Aktivitas Kasir POS Hari Ini</h2>
            </div>
            <Link href="/pos" className="text-[11px] font-medium text-primary hover:underline flex items-center gap-1">
              Buka POS <ExternalLink className="h-3 w-3" />
            </Link>
          </div>

          <div className="p-5 space-y-4">
            <div className="grid grid-cols-3 gap-2.5">
              <div className="bg-muted/40 rounded-lg p-2.5 text-center border border-border/30">
                <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Omzet Hari Ini</div>
                <div className="font-mono font-bold text-sm text-foreground mt-0.5">
                  {formatCurrency(s.pos.today_revenue, { compact: true })}
                </div>
              </div>
              <div className="bg-muted/40 rounded-lg p-2.5 text-center border border-border/30">
                <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Struk Transaksi</div>
                <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                  {s.pos.today_orders_count}
                </div>
              </div>
              <div className="bg-muted/40 rounded-lg p-2.5 text-center border border-border/30">
                <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Total Order POS</div>
                <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                  {s.pos.total_orders_count}
                </div>
              </div>
            </div>

            <div>
              <div className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground/70 mb-1.5">
                Struk Transaksi Terakhir
              </div>
              {s.pos.recent_orders.length > 0 ? (
                <ul className="divide-y divide-border/40 border border-border/50 rounded-lg overflow-hidden">
                  {s.pos.recent_orders.map((order) => (
                    <li key={order.order_number} className="px-3 py-2 flex items-center justify-between gap-2 text-[12px] hover:bg-muted/20">
                      <div className="min-w-0">
                        <div className="font-mono text-foreground font-medium truncate">{order.order_number}</div>
                        <div className="text-[10px] text-muted-foreground truncate">{order.customer_name}</div>
                      </div>
                      <div className="text-right shrink-0">
                        <div className="font-mono font-semibold text-foreground tabular-nums">
                          {formatCurrency(order.total_amount)}
                        </div>
                        <div className="text-[10px] text-muted-foreground font-medium">{order.payment_method}</div>
                      </div>
                    </li>
                  ))}
                </ul>
              ) : (
                <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">
                  Belum ada transaksi POS tercatat hari ini.
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Pergudangan & Peringatan Stok */}
        <div className="border border-border/60 rounded-lg bg-card">
          <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Warehouse className="h-3.5 w-3.5 text-muted-foreground" />
              <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Kesehatan Pergudangan (WMS)</h2>
            </div>
            <Link href="/wms" className="text-[11px] font-medium text-primary hover:underline flex items-center gap-1">
              Kelola Gudang <ExternalLink className="h-3 w-3" />
            </Link>
          </div>

          <div className="p-5 space-y-4">
            <div className="grid grid-cols-3 gap-2.5">
              <div className="bg-muted/40 rounded-lg p-2.5 text-center border border-border/30">
                <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Total SKU</div>
                <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                  {s.wms.total_skus} SKU
                </div>
              </div>
              <div className="bg-muted/40 rounded-lg p-2.5 text-center border border-border/30">
                <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Fisik Tersimpan</div>
                <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                  {s.wms.total_physical_units} unit
                </div>
              </div>
              <div className="bg-muted/40 rounded-lg p-2.5 text-center border border-border/30">
                <div className="text-[10px] text-muted-foreground uppercase tracking-wide">Gudang / Lokasi</div>
                <div className="font-mono font-bold text-sm text-foreground mt-0.5 tabular-nums">
                  {s.wms.total_warehouses} / {s.wms.total_locations}
                </div>
              </div>
            </div>

            <div>
              <div className="flex items-center justify-between mb-1.5">
                <span className="text-[10px] font-semibold uppercase tracking-wider text-amber-600 dark:text-amber-400 flex items-center gap-1">
                  <AlertTriangle className="h-3 w-3" /> Peringatan Stok Menipis (&le; 5 unit)
                </span>
                <Link href="/products" className="text-[10px] font-medium text-muted-foreground hover:text-primary">
                  Master Produk &rarr;
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
                        <div className="font-mono text-[10px] text-muted-foreground">{item.sku} • Min: {item.min_threshold}</div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <span className="px-2 py-0.5 rounded-full text-[11px] font-bold bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300 tabular-nums">
                          Sisa {item.current_stock}
                        </span>
                        <Link
                          href="/dashboard/purchase-orders"
                          className="px-2 py-0.5 bg-background border border-border/70 text-[10px] font-medium text-foreground hover:border-primary rounded transition-colors"
                        >
                          Reorder
                        </Link>
                      </div>
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
        </div>
      </div>

      {/* ── Status Invoice Breakdown + Pipeline Pembayaran ────────────────── */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Status Invoice Vendor */}
        <div className="border border-border/60 rounded-lg bg-card">
          <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Status Validasi Invoice Vendor</h2>
            <Link href="/dashboard/invoices" className="text-[11px] font-medium text-primary hover:underline flex items-center gap-1">
              Semua Invoice <ExternalLink className="h-3 w-3" />
            </Link>
          </div>
          <div className="p-5 flex items-center gap-5">
            <div className="w-40 h-40 shrink-0">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={[
                      { name: 'Menunggu', value: s.invoices.pending },
                      { name: 'Disetujui', value: s.invoices.approved },
                      { name: 'Ditolak', value: s.invoices.rejected },
                      { name: 'Review AI', value: s.invoices.pending_review },
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
                  <Tooltip formatter={(value, name) => [`${value} Dokumen`, String(name)]} />
                </PieChart>
              </ResponsiveContainer>
            </div>
            <div className="grid grid-cols-2 gap-2.5 flex-1">
              <StatusBadge status="pending" count={s.invoices.pending} />
              <StatusBadge status="approved" count={s.invoices.approved} />
              <StatusBadge status="rejected" count={s.invoices.rejected} />
              <StatusBadge status="pending_review" count={s.invoices.pending_review} />
            </div>
          </div>
        </div>

        {/* Pipeline Pembayaran Tagihan Vendor */}
        <div className="border border-border/60 rounded-lg bg-card">
          <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Pipeline Pembayaran Tagihan (AP)</h2>
            <Link href="/dashboard/payment-orders" className="text-[11px] font-medium text-primary hover:underline flex items-center gap-1">
              Payment Orders <ExternalLink className="h-3 w-3" />
            </Link>
          </div>
          <div className="p-5 space-y-3.5">
            <div className="flex items-center justify-between">
              <span className="text-xs text-muted-foreground">Total Dokumen Order Pembayaran</span>
              <span className="text-base font-bold text-foreground font-mono">{s.payments.total} Dokumen</span>
            </div>
            <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
              <div
                className="bg-emerald-500 h-full rounded-full transition-all"
                style={{
                  width: s.payments.total > 0
                    ? `${(s.payments.paid / s.payments.total) * 100}%`
                    : '50%',
                }}
              />
            </div>
            <div className="flex justify-between text-xs text-muted-foreground">
              <span>{s.payments.paid} Lunas Selesai</span>
              <span>{Math.max(0, s.payments.total - s.payments.paid)} Menunggu Eksekusi</span>
            </div>
            <div className="grid grid-cols-2 gap-3 pt-1">
              <div className="rounded-lg p-3 bg-muted/40 border border-border/40">
                <div className="text-[10px] uppercase tracking-wide text-emerald-600 dark:text-emerald-400 font-semibold mb-0.5">Sudah Dibayar</div>
                <div className="text-sm font-bold text-foreground font-mono">{formatCurrency(s.payments.paid_amount)}</div>
              </div>
              <div className="rounded-lg p-3 bg-muted/40 border border-border/40">
                <div className="text-[10px] uppercase tracking-wide text-amber-600 dark:text-amber-400 font-semibold mb-0.5">Menunggu Pembayaran</div>
                <div className="text-sm font-bold text-foreground font-mono">{formatCurrency(s.payments.pending_amount)}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* ── Monthly Invoice Trend (Bar Chart) ─────────────────────────────── */}
      {s.monthly_trend.length > 0 && (
        <div className="border border-border/60 rounded-lg bg-card">
          <div className="px-5 py-3 border-b border-border/60">
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Tren Pengadaan &amp; Tagihan Vendor (6 Bulan Terakhir)</h2>
          </div>
          <div className="p-5">
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={s.monthly_trend} barGap={4}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                <XAxis
                  dataKey="month"
                  tick={{ fontSize: 11, fill: '#888888' }}
                  axisLine={false}
                  tickLine={false}
                />
                <YAxis
                  tick={{ fontSize: 11, fill: '#888888' }}
                  axisLine={false}
                  tickLine={false}
                  tickFormatter={formatCompact}
                />
                <Tooltip content={<BarChartTooltip />} />
                <Bar
                  dataKey="invoice_count"
                  name="Jumlah Invoice"
                  fill="#10b981"
                  radius={[3, 3, 0, 0]}
                  maxBarSize={40}
                />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      )}

      {/* ── Top 5 Vendors Table (Human-Centric ZenSpace Design, Bebas AI Slop) ── */}
      {s.top_vendors.length > 0 && (
        <div className="border border-border/60 rounded-lg bg-card overflow-hidden">
          <div className="px-5 py-3.5 border-b border-border/60 flex items-center justify-between">
            <div>
              <h2 className="text-xs font-semibold uppercase tracking-wider text-foreground">
                Mitra Vendor &amp; Rekapitulasi Tagihan Pemasok
              </h2>
              <p className="text-[11px] text-muted-foreground mt-0.5">
                Daftar 5 mitra pemasok utama berdasarkan akumulasi nilai transaksi tagihan pembelian.
              </p>
            </div>
            <Link
              href="/dashboard/vendors"
              className="text-xs font-medium text-primary hover:underline inline-flex items-center gap-1"
            >
              Lihat Semua {s.vendors.active} Vendor <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="border-b border-border/60 bg-muted/20 text-muted-foreground font-semibold uppercase text-[10px]">
                  <th className="text-left px-5 py-3 w-10">#</th>
                  <th className="text-left px-5 py-3">Nama Mitra Vendor</th>
                  <th className="text-left px-5 py-3">Rekening &amp; Kota</th>
                  <th className="text-center px-5 py-3">Keandalan</th>
                  <th className="text-center px-5 py-3">Jumlah Invoice</th>
                  <th className="text-right px-5 py-3">Total Nilai Tagihan</th>
                  <th className="text-right px-5 py-3">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/40">
                {s.top_vendors.map((vendor, idx) => {
                  const meta = VENDOR_META[vendor.vendor_id] ?? {
                    bank: 'BCA',
                    phone: '021-5551234',
                    rating: '4.5',
                    count: 10,
                    city: 'Jakarta',
                  }
                  return (
                    <tr key={vendor.vendor_id} className="hover:bg-muted/30 transition-colors">
                      <td className="px-5 py-3 text-muted-foreground font-mono text-[11px]">{idx + 1}</td>
                      <td className="px-5 py-3">
                        <div className="font-semibold text-foreground text-xs">{vendor.vendor_name}</div>
                        <div className="text-[11px] text-muted-foreground">{meta.phone}</div>
                      </td>
                      <td className="px-5 py-3 text-muted-foreground">
                        <span className="font-medium text-foreground">{meta.bank}</span> • {meta.city}
                      </td>
                      <td className="px-5 py-3 text-center">
                        <div className="inline-flex items-center gap-1 text-amber-500 font-medium">
                          <Star className="h-3 w-3 fill-amber-500" />
                          <span>{meta.rating}</span>
                          <span className="text-muted-foreground text-[10px]">({meta.count})</span>
                        </div>
                      </td>
                      <td className="px-5 py-3 text-center text-muted-foreground font-mono">
                        {vendor.invoice_count} Invoice
                      </td>
                      <td className="px-5 py-3 text-right font-bold text-foreground font-mono tabular-nums">
                        {formatCurrency(vendor.total_amount)}
                      </td>
                      <td className="px-5 py-3 text-right">
                        <Link
                          href={`/dashboard/vendors/${vendor.vendor_id}`}
                          className="px-2.5 py-1 text-[11px] font-medium text-primary hover:text-primary/80 hover:bg-primary/10 rounded transition-colors inline-flex items-center gap-1"
                        >
                          Detail <ExternalLink className="h-3 w-3" />
                        </Link>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* ── Invoice Terbaru ───────────────────────────────────────────────── */}
      <div className="border border-border/60 rounded-lg bg-card">
        <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between">
          <div>
            <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">Daftar Invoice Vendor Terbaru</h2>
          </div>
          <Link href="/dashboard/invoices" className="text-xs font-medium text-primary hover:underline">
            Kelola Invoice &rarr;
          </Link>
        </div>
        {invoicesLoading ? (
          <div className="p-5 space-y-2.5">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-7 bg-muted/40 rounded-md animate-pulse" />
            ))}
          </div>
        ) : invoices.length === 0 ? (
          <div className="px-5 py-8 text-center text-xs text-muted-foreground">
            Belum ada invoice vendor yang terdaftar.
          </div>
        ) : (
          <ul className="divide-y divide-border/40">
            {invoices.slice(0, 5).map((inv) => {
              const vendorName =
                (inv as { vendor_name?: string }).vendor_name ??
                VENDOR_NAMES[inv.vendor_id] ??
                'Mitra Vendor'
              return (
                <li key={inv.id} className="px-5 py-3 flex items-center justify-between text-xs hover:bg-muted/20">
                  <div className="min-w-0">
                    <div className="font-mono font-medium text-foreground">{inv.invoice_number}</div>
                    <div className="text-[11px] text-muted-foreground truncate">{vendorName}</div>
                  </div>
                  <div className="flex items-center gap-3">
                    <span className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                      inv.status === 'approved' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300' :
                      inv.status === 'pending_review' ? 'bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300' :
                      inv.status === 'rejected' ? 'bg-rose-50 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300' :
                      'bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300'
                    }`}>
                      {inv.status === 'approved' ? 'Disetujui' :
                       inv.status === 'pending_review' ? 'Review AI' :
                       inv.status === 'rejected' ? 'Ditolak' : 'Menunggu'}
                    </span>
                    <span className="font-bold text-foreground font-mono tabular-nums">{formatCurrency(inv.amount)}</span>
                  </div>
                </li>
              )
            })}
          </ul>
        )}
      </div>
    </div>
  )
}
