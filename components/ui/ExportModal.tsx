'use client'

import React, { useEffect, useId, useMemo, useRef, useState } from 'react'
import { X, FileSpreadsheet, FileText, Printer, Download, Filter, Columns3, RotateCcw } from 'lucide-react'
import { runExport, type ExportColumn, type ExportFormat } from '@/lib/export'
import { cn } from '@/lib/utils'

export type ExportFilter<T> =
  | {
      type: 'select'
      id: string
      label: string
      options: { value: string; label: string }[]
      /** Return true if the row matches the chosen value. "ALL" is handled by the modal. */
      match: (row: T, value: string) => boolean
    }
  | {
      type: 'dateRange'
      id: string
      label: string
      getDate: (row: T) => string | undefined | null
    }

export interface ExportModalProps<T> {
  open: boolean
  onClose: () => void
  title: string
  filename: string
  columns: ExportColumn<T>[]
  /** All available rows (unfiltered by the page). */
  allRows: T[]
  /** Rows as currently shown on the page (after page search/filters). Optional. */
  visibleRows?: T[]
  /** Short description of the page's active filters, e.g. `Pencarian: "kopi"`. */
  visibleSummary?: string[]
  filters?: ExportFilter<T>[]
}

const FORMATS: { id: ExportFormat; label: string; ext: string; desc: string; Icon: React.ElementType }[] = [
  { id: 'xlsx', label: 'Excel', ext: '.xlsx', desc: 'Angka tetap numerik, siap untuk rumus & pivot.', Icon: FileSpreadsheet },
  { id: 'csv', label: 'CSV', ext: '.csv', desc: 'Teks universal untuk impor ke aplikasi lain.', Icon: FileText },
  { id: 'pdf', label: 'PDF', ext: '.pdf', desc: 'Laporan A4 siap cetak / simpan sebagai PDF.', Icon: Printer },
]

type Scope = 'visible' | 'custom'

export const EXPORT_TIMEOUT_MS = 20_000

function inRange(iso: string | null | undefined, from: string, to: string): boolean {
  if (!from && !to) return true
  if (!iso) return false
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return false
  if (from && t < new Date(`${from}T00:00:00`).getTime()) return false
  if (to && t > new Date(`${to}T23:59:59.999`).getTime()) return false
  return true
}

/** State lives in the inner component, which mounts only while open → fresh state on every open. */
export function ExportModal<T>(props: ExportModalProps<T>) {
  return props.open ? <ExportDialog {...props} /> : null
}

function ExportDialog<T>({
  open,
  onClose,
  title,
  filename,
  columns,
  allRows,
  visibleRows,
  visibleSummary = [],
  filters = [],
}: ExportModalProps<T>) {
  const uid = useId()
  const dialogRef = useRef<HTMLDivElement>(null)
  const hasVisible = visibleRows !== undefined
  const [scope, setScope] = useState<Scope>(hasVisible ? 'visible' : 'custom')
  const [format, setFormat] = useState<ExportFormat>('xlsx')
  const [selects, setSelects] = useState<Record<string, string>>({})
  const [ranges, setRanges] = useState<Record<string, { from: string; to: string }>>({})
  const [enabledCols, setEnabledCols] = useState<boolean[]>(() => columns.map(() => true))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)


  // Escape closes; focus on open and lock body scroll.
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !busy) onClose()
    }
    document.addEventListener('keydown', onKey)
    const prevOverflow = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    queueMicrotask(() => dialogRef.current?.focus())
    return () => {
      document.removeEventListener('keydown', onKey)
      document.body.style.overflow = prevOverflow
    }
  }, [open, onClose, busy])

  const customRows = useMemo(() => {
    return allRows.filter((row) =>
      filters.every((f) => {
        if (f.type === 'select') {
          const v = selects[f.id] ?? 'ALL'
          return v === 'ALL' || f.match(row, v)
        }
        const r = ranges[f.id] ?? { from: '', to: '' }
        return inRange(f.getDate(row), r.from, r.to)
      })
    )
  }, [allRows, filters, selects, ranges])

  const rows = scope === 'visible' && visibleRows ? visibleRows : customRows
  const activeColumns = columns.filter((_, i) => enabledCols[i])

  const summary = useMemo(() => {
    if (scope === 'visible') return visibleSummary.length ? visibleSummary : ['Sesuai tampilan halaman']
    const parts: string[] = []
    for (const f of filters) {
      if (f.type === 'select') {
        const v = selects[f.id] ?? 'ALL'
        if (v !== 'ALL') parts.push(`${f.label}: ${f.options.find((o) => o.value === v)?.label ?? v}`)
      } else {
        const r = ranges[f.id]
        if (r?.from || r?.to) parts.push(`${f.label}: ${r.from || '…'} s/d ${r.to || '…'}`)
      }
    }
    return parts.length ? parts : ['Semua data']
  }, [scope, visibleSummary, filters, selects, ranges])

  const rangeInvalid = filters.some((f) => {
    if (f.type !== 'dateRange') return false
    const r = ranges[f.id]
    return !!(r?.from && r?.to && r.from > r.to)
  })

  const canExport = !busy && rows.length > 0 && activeColumns.length > 0 && !rangeInvalid

  const resetFilters = () => {
    setSelects({})
    setRanges({})
  }

  // Not async before runExport: PDF must open its window inside the click gesture.
  const handleExport = () => {
    if (!canExport) return
    setError(null)
    setBusy(true)
    // Guard: a stalled lazy chunk (slow network / stale deploy) must not leave
    // the dialog stuck on "Memproses…" forever.
    let timer: ReturnType<typeof setTimeout> | undefined
    const timeout = new Promise<never>((_, reject) => {
      timer = setTimeout(
        () => reject(new Error('Ekspor terlalu lama. Muat ulang halaman lalu coba lagi.')),
        EXPORT_TIMEOUT_MS
      )
    })
    Promise.race([runExport(format, { title, filename, columns: activeColumns, rows, filterSummary: summary }), timeout])
      .finally(() => clearTimeout(timer))
      .then(() => onClose())
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Gagal mengekspor data.'))
      .finally(() => setBusy(false))
  }

  if (!open) return null

  const titleId = `${uid}-title`
  const fieldCls =
    'w-full min-h-[40px] rounded-lg border border-slate-300 bg-white px-3 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-slate-100'

  return (
    <div className="fixed inset-0 z-[60] flex items-end sm:items-center justify-center">
      <div className="absolute inset-0 bg-slate-900/50" onClick={() => !busy && onClose()} aria-hidden="true" />
      <div
        ref={dialogRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className="relative w-full sm:max-w-xl max-h-[92vh] flex flex-col bg-white rounded-t-2xl sm:rounded-2xl shadow-2xl outline-none"
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-4 px-5 sm:px-6 py-4 border-b border-slate-200">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-blue-50 text-[#2563EB] border border-blue-100">
              <Download className="w-5 h-5" aria-hidden="true" />
            </div>
            <div>
              <h2 id={titleId} className="text-base sm:text-lg font-bold text-slate-900">
                Ekspor {title}
              </h2>
              <p className="text-xs text-slate-500">Saring data, pilih format, lalu unduh.</p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            disabled={busy}
            aria-label="Tutup dialog ekspor"
            className="p-2 min-h-[40px] min-w-[40px] rounded-lg text-slate-500 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:opacity-50"
          >
            <X className="w-5 h-5" aria-hidden="true" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-y-auto px-5 sm:px-6 py-5 space-y-6">
          {/* 1. Filter */}
          <section aria-labelledby={`${uid}-filter`}>
            <div className="flex items-center justify-between mb-3">
              <h3 id={`${uid}-filter`} className="flex items-center gap-2 text-sm font-semibold text-slate-900">
                <Filter className="w-4 h-4 text-slate-500" aria-hidden="true" />
                1. Filter data
              </h3>
              {scope === 'custom' && filters.length > 0 && (
                <button
                  type="button"
                  onClick={resetFilters}
                  className="inline-flex items-center gap-1 text-xs font-medium text-blue-700 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 rounded"
                >
                  <RotateCcw className="w-3 h-3" aria-hidden="true" /> Reset
                </button>
              )}
            </div>

            <div role="radiogroup" aria-label="Cakupan data" className="grid gap-2 sm:grid-cols-2">
              {hasVisible && (
                <ScopeOption
                  checked={scope === 'visible'}
                  onSelect={() => setScope('visible')}
                  title="Sesuai tampilan"
                  hint={`${visibleRows!.length.toLocaleString('id-ID')} data dari halaman`}
                />
              )}
              <ScopeOption
                checked={scope === 'custom'}
                onSelect={() => setScope('custom')}
                title={filters.length ? 'Filter khusus' : 'Semua data'}
                hint={`${customRows.length.toLocaleString('id-ID')} dari ${allRows.length.toLocaleString('id-ID')} data`}
              />
            </div>

            {scope === 'custom' && filters.length > 0 && (
              <div className="mt-3 grid gap-3 sm:grid-cols-2 rounded-xl border border-slate-200 bg-slate-50 p-3">
                {filters.map((f) =>
                  f.type === 'select' ? (
                    <div key={f.id}>
                      <label htmlFor={`${uid}-${f.id}`} className="block text-xs font-medium text-slate-700 mb-1">
                        {f.label}
                      </label>
                      <select
                        id={`${uid}-${f.id}`}
                        className={fieldCls}
                        value={selects[f.id] ?? 'ALL'}
                        onChange={(e) => setSelects((s) => ({ ...s, [f.id]: e.target.value }))}
                      >
                        <option value="ALL">Semua</option>
                        {f.options.map((o) => (
                          <option key={o.value} value={o.value}>
                            {o.label}
                          </option>
                        ))}
                      </select>
                    </div>
                  ) : (
                    <fieldset key={f.id} className="sm:col-span-2">
                      <legend className="block text-xs font-medium text-slate-700 mb-1">{f.label}</legend>
                      <div className="grid grid-cols-2 gap-2">
                        <input
                          type="date"
                          aria-label={`${f.label} dari`}
                          className={fieldCls}
                          value={ranges[f.id]?.from ?? ''}
                          onChange={(e) =>
                            setRanges((r) => ({ ...r, [f.id]: { from: e.target.value, to: r[f.id]?.to ?? '' } }))
                          }
                        />
                        <input
                          type="date"
                          aria-label={`${f.label} sampai`}
                          className={fieldCls}
                          value={ranges[f.id]?.to ?? ''}
                          onChange={(e) =>
                            setRanges((r) => ({ ...r, [f.id]: { from: r[f.id]?.from ?? '', to: e.target.value } }))
                          }
                        />
                      </div>
                    </fieldset>
                  )
                )}
                {rangeInvalid && (
                  <p role="alert" className="sm:col-span-2 text-xs text-red-700">
                    Tanggal awal tidak boleh setelah tanggal akhir.
                  </p>
                )}
              </div>
            )}
          </section>

          {/* 2. Format */}
          <section aria-labelledby={`${uid}-format`}>
            <h3 id={`${uid}-format`} className="text-sm font-semibold text-slate-900 mb-3">
              2. Format berkas
            </h3>
            <div role="radiogroup" aria-labelledby={`${uid}-format`} className="grid grid-cols-3 gap-2">
              {FORMATS.map(({ id, label, ext, Icon }) => {
                const active = format === id
                return (
                  <button
                    key={id}
                    type="button"
                    role="radio"
                    aria-checked={active}
                    onClick={() => setFormat(id)}
                    className={cn(
                      'flex flex-col items-center gap-1 rounded-xl border-2 p-3 min-h-[84px] transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
                      active ? 'border-[#2563EB] bg-blue-50 text-[#1D4ED8]' : 'border-slate-200 text-slate-700 hover:border-slate-300 hover:bg-slate-50'
                    )}
                  >
                    <Icon className="w-6 h-6" aria-hidden="true" />
                    <span className="text-sm font-semibold">{label}</span>
                    <span className="text-[11px] text-slate-500">{ext}</span>
                  </button>
                )
              })}
            </div>
            <p className="mt-2 text-xs text-slate-600">{FORMATS.find((f) => f.id === format)?.desc}</p>
          </section>

          {/* 3. Columns */}
          <details className="group rounded-xl border border-slate-200">
            <summary className="flex cursor-pointer items-center justify-between gap-2 px-3 py-2.5 text-sm font-semibold text-slate-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 rounded-xl">
              <span className="flex items-center gap-2">
                <Columns3 className="w-4 h-4 text-slate-500" aria-hidden="true" />
                3. Kolom ({activeColumns.length}/{columns.length})
              </span>
              <span className="text-xs font-normal text-slate-500 group-open:hidden">Atur</span>
            </summary>
            <div className="grid grid-cols-2 gap-1 border-t border-slate-200 p-3">
              {columns.map((c, i) => (
                <label key={c.header} className="flex items-center gap-2 rounded px-1 py-1.5 text-sm text-slate-700 hover:bg-slate-50 cursor-pointer">
                  <input
                    type="checkbox"
                    className="h-4 w-4 accent-[#2563EB]"
                    checked={enabledCols[i] ?? true}
                    onChange={(e) => setEnabledCols((arr) => arr.map((v, j) => (j === i ? e.target.checked : v)))}
                  />
                  {c.header}
                </label>
              ))}
            </div>
          </details>

          {error && (
            <p role="alert" className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700">
              {error}
            </p>
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center gap-3 border-t border-slate-200 bg-slate-50 px-5 sm:px-6 py-4 rounded-b-none sm:rounded-b-2xl">
          <p className="text-xs text-slate-600" aria-live="polite">
            <span className="font-semibold text-slate-900">{rows.length.toLocaleString('id-ID')}</span> baris ·{' '}
            {activeColumns.length} kolom
          </p>
          <div className="ml-auto flex gap-2">
            <button
              type="button"
              onClick={onClose}
              disabled={busy}
              className="px-4 min-h-[44px] rounded-lg border border-slate-300 bg-white text-sm font-medium text-slate-700 hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:opacity-50"
            >
              Batal
            </button>
            <button
              type="button"
              onClick={handleExport}
              disabled={!canExport}
              className="inline-flex items-center gap-2 px-4 min-h-[44px] rounded-lg bg-[#2563EB] text-sm font-semibold text-white hover:bg-[#1D4ED8] focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {busy ? (
                <span className="h-4 w-4 rounded-full border-2 border-white border-t-transparent animate-spin" aria-hidden="true" />
              ) : (
                <Download className="w-4 h-4" aria-hidden="true" />
              )}
              {busy ? 'Memproses…' : format === 'pdf' ? 'Cetak / PDF' : 'Unduh'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

function ScopeOption({ checked, onSelect, title, hint }: { checked: boolean; onSelect: () => void; title: string; hint: string }) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={checked}
      onClick={onSelect}
      className={cn(
        'flex items-start gap-3 rounded-xl border-2 p-3 text-left min-h-[56px] transition-colors focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500',
        checked ? 'border-[#2563EB] bg-blue-50' : 'border-slate-200 hover:bg-slate-50'
      )}
    >
      <span
        className={cn('mt-0.5 h-4 w-4 shrink-0 rounded-full border-2', checked ? 'border-[#2563EB] bg-[#2563EB] ring-2 ring-inset ring-white' : 'border-slate-400')}
        aria-hidden="true"
      />
      <span>
        <span className="block text-sm font-semibold text-slate-900">{title}</span>
        <span className="block text-xs text-slate-600">{hint}</span>
      </span>
    </button>
  )
}

export function ExportButton({ onClick, disabled }: { onClick: () => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      className="inline-flex items-center justify-center gap-2 px-4 py-2.5 min-h-[44px] rounded-lg border border-slate-200 bg-white text-sm font-semibold text-slate-700 hover:bg-slate-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500 disabled:opacity-50 transition-colors"
    >
      <Download className="w-4 h-4" aria-hidden="true" />
      <span>Ekspor</span>
    </button>
  )
}
