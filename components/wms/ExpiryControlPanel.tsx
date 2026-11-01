'use client'

import { useState } from 'react'
import Link from 'next/link'
import { CalendarClock, AlertOctagon, AlertTriangle, Clock, ExternalLink } from 'lucide-react'
import { useExpiryReport } from '@/hooks/useExpiryReport'
import type { ExpiringBatchLine, ExpirySeverity } from '@/lib/api'

// Kontrol expired per batch (OCA wms §1.3 FEFO). Data dari GET /wms/expiry-report.
// Batch yang sudah lewat tanggal expired diblokir backend dari Surat Jalan, POS,
// transfer & marketplace; jalan keluarnya hanya Scrap / Opname / Karantina.

export const EXPIRY_WINDOWS = [30, 60, 90] as const

const SEVERITY_META: Record<ExpirySeverity, { label: string; badge: string }> = {
  EXPIRED: {
    label: 'Expired',
    badge: 'bg-rose-100 text-rose-700 dark:bg-rose-950/40 dark:text-rose-300',
  },
  CRITICAL: {
    label: 'Kritis',
    badge: 'bg-amber-100 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300',
  },
  WARNING: {
    label: 'Segera',
    badge: 'bg-yellow-50 text-yellow-800 dark:bg-yellow-950/30 dark:text-yellow-300',
  },
}

export function formatExpiryDate(value: string): string {
  // Backend sends a calendar date (e.g. "2026-10-15T00:00:00Z"); show it as-is, not shifted by timezone.
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(value)
  if (!m) return value
  const d = new Date(Date.UTC(Number(m[1]), Number(m[2]) - 1, Number(m[3])))
  return d.toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric', timeZone: 'UTC' })
}

export function daysLeftLabel(days: number): string {
  if (days < 0) return `Lewat ${Math.abs(days)} hari`
  if (days === 0) return 'Expired hari ini'
  return `${days} hari lagi`
}

function formatQty(q: string): string {
  const n = Number(q)
  return Number.isFinite(n) ? n.toLocaleString('id-ID') : q
}

function SummaryTile({
  icon: Icon,
  label,
  batches,
  qty,
  className,
}: {
  icon: React.ElementType
  label: string
  batches: number
  qty: string
  className: string
}) {
  return (
    <div className={`rounded-lg border p-3 ${className}`}>
      <div className="flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-wider">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      <div className="mt-1 text-lg font-bold tabular-nums">{batches} batch</div>
      <div className="text-[11px] opacity-80 tabular-nums">{formatQty(qty)} unit</div>
    </div>
  )
}

export function ExpiryControlPanel({ warehouseId }: { warehouseId?: string }) {
  const [days, setDays] = useState<number>(30)
  const { data, isLoading, error } = useExpiryReport(days, warehouseId)
  const lines: ExpiringBatchLine[] = data?.lines ?? []
  const s = data?.summary

  return (
    <section className="border border-border/60 rounded-lg bg-card" aria-label="Kontrol Expired per Batch">
      <div className="px-5 py-3 border-b border-border/60 flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-2">
          <CalendarClock className="h-3.5 w-3.5 text-muted-foreground" />
          <h2 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
            Kontrol Expired per Batch
          </h2>
        </div>
        <div className="flex items-center gap-1" role="group" aria-label="Jendela hari">
          {EXPIRY_WINDOWS.map((w) => (
            <button
              key={w}
              type="button"
              onClick={() => setDays(w)}
              aria-pressed={days === w}
              className={`px-2.5 py-1 rounded-md text-[11px] font-medium border transition-colors ${
                days === w
                  ? 'bg-primary text-primary-foreground border-primary'
                  : 'border-border/60 text-muted-foreground hover:bg-muted/50'
              }`}
            >
              {w} hari
            </button>
          ))}
        </div>
      </div>

      <div className="p-5 space-y-4">
        <p className="text-[11px] text-muted-foreground">
          Batch yang sudah lewat tanggal expired otomatis diblokir dari Surat Jalan, Kasir (POS), Transfer, dan
          Marketplace. Keluarkan lewat menu Scrap atau Karantina.
        </p>

        {error ? (
          <div role="alert" className="px-3 py-4 text-center text-xs text-rose-700 dark:text-rose-300 bg-rose-50 dark:bg-rose-950/20 rounded-lg">
            Gagal memuat data expired. {error instanceof Error ? error.message : ''}
          </div>
        ) : isLoading ? (
          <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">Memuat data expired…</div>
        ) : (
          <>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
              <SummaryTile
                icon={AlertOctagon}
                label="Sudah Expired"
                batches={s?.expired_batches ?? 0}
                qty={s?.expired_qty ?? '0'}
                className="border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900/40 dark:bg-rose-950/20 dark:text-rose-200"
              />
              <SummaryTile
                icon={AlertTriangle}
                label="Kritis (≤ 7 hari)"
                batches={s?.critical_batches ?? 0}
                qty={s?.critical_qty ?? '0'}
                className="border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/20 dark:text-amber-200"
              />
              <SummaryTile
                icon={Clock}
                label={`Segera (≤ ${data?.days ?? days} hari)`}
                batches={s?.warning_batches ?? 0}
                qty={s?.warning_qty ?? '0'}
                className="border-yellow-200 bg-yellow-50 text-yellow-800 dark:border-yellow-900/40 dark:bg-yellow-950/20 dark:text-yellow-200"
              />
            </div>

            {lines.length === 0 ? (
              <div className="px-3 py-6 text-center text-xs text-muted-foreground bg-muted/20 rounded-lg">
                Tidak ada batch yang expired atau akan expired dalam {data?.days ?? days} hari.
              </div>
            ) : (
              <div className="overflow-x-auto -mx-1">
                <table className="w-full min-w-[640px] text-[12px]">
                  <thead>
                    <tr className="text-left text-[10px] uppercase tracking-wider text-muted-foreground border-b border-border/60">
                      <th className="py-2 px-1 font-semibold">Status</th>
                      <th className="py-2 px-1 font-semibold">Produk</th>
                      <th className="py-2 px-1 font-semibold">Batch</th>
                      <th className="py-2 px-1 font-semibold">Tgl Expired</th>
                      <th className="py-2 px-1 font-semibold">Sisa Umur</th>
                      <th className="py-2 px-1 font-semibold">Gudang / Rak</th>
                      <th className="py-2 px-1 font-semibold text-right">Qty</th>
                    </tr>
                  </thead>
                  <tbody>
                    {lines.map((l) => {
                      const meta = SEVERITY_META[l.severity] ?? SEVERITY_META.WARNING
                      return (
                        <tr key={`${l.batch_id}-${l.location_id}`} className="border-b border-border/40 last:border-0">
                          <td className="py-2 px-1">
                            <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${meta.badge}`}>{meta.label}</span>
                          </td>
                          <td className="py-2 px-1">
                            <div className="font-medium text-foreground">{l.product_name || '-'}</div>
                            <div className="font-mono text-[10px] text-muted-foreground">{l.product_sku}</div>
                          </td>
                          <td className="py-2 px-1 font-mono">{l.batch_number}</td>
                          <td className="py-2 px-1 tabular-nums">{formatExpiryDate(l.expiry_date)}</td>
                          <td className={`py-2 px-1 tabular-nums ${l.days_left < 0 ? 'text-rose-600 font-semibold' : ''}`}>
                            {daysLeftLabel(l.days_left)}
                          </td>
                          <td className="py-2 px-1">
                            <div>{l.warehouse_name || '-'}</div>
                            <div className="font-mono text-[10px] text-muted-foreground">{l.location_code}</div>
                          </td>
                          <td className="py-2 px-1 text-right font-semibold tabular-nums">{formatQty(l.quantity)}</td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>
            )}

            {(s?.expired_batches ?? 0) > 0 && (
              <div className="flex justify-end">
                <Link href="/wms/scrap" className="text-[11px] font-medium text-primary hover:underline inline-flex items-center gap-1">
                  Proses batch expired di menu Scrap <ExternalLink className="h-3 w-3" />
                </Link>
              </div>
            )}
          </>
        )}
      </div>
    </section>
  )
}

export default ExpiryControlPanel
