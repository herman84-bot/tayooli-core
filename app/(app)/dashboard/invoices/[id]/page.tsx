'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, CheckCircle, Clock, XCircle, AlertCircle, GitCompareArrows, ExternalLink } from 'lucide-react'
import { useInvoice, useApproveInvoice, useRejectInvoice } from '@/lib/queries/invoices'
import { usePO } from '@/lib/queries/po'
import { type InvoiceStatus, type Invoice } from '@/lib/schemas/invoice'
import type { PO } from '@/lib/schemas/po'
import { cn } from '@/lib/utils'
import { getClientErrorMessage } from '@/lib/api/errors'
import { StatusBadge } from '@/components/ui/status-badge'

// ── Status config ─────────────────────────────────────────────────────────────
const statusConfig: Record<
  InvoiceStatus,
  { label: string; icon: React.ElementType; classes: string }
> = {
  pending: {
    label: 'Menunggu Persetujuan',
    icon: Clock,
    classes: 'bg-amber-50 text-amber-700 border-amber-200',
  },
  ai_processed: {
    label: 'OCR Selesai',
    icon: CheckCircle,
    classes: 'bg-indigo-50 text-indigo-700 border-indigo-200',
  },
  ai_failed: {
    label: 'OCR Gagal',
    icon: AlertCircle,
    classes: 'bg-red-50 text-red-700 border-red-200',
  },
  approved: {
    label: 'Disetujui',
    icon: CheckCircle,
    classes: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  },
  rejected: {
    label: 'Ditolak',
    icon: XCircle,
    classes: 'bg-rose-50 text-rose-600 border-rose-200',
  },
  pending_review: {
    label: 'Perlu Review',
    icon: AlertCircle,
    classes: 'bg-orange-50 text-orange-700 border-orange-200',
  },
}

// ── Field row ────────────────────────────────────────────────────────────────
function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-zinc-500 font-medium uppercase tracking-wide">{label}</dt>
      <dd className="text-sm text-zinc-800 font-medium">{value}</dd>
    </div>
  )
}

// ── Match explanations ─────────────────────────────────────────────────────
const matchExplanations: Record<string, { text: string; color: string }> = {
  matched: { text: 'Cocok — Invoice sesuai dengan PO dan GR', color: 'text-emerald-700 bg-emerald-50 border-emerald-200' },
  amount_mismatch: { text: 'Jumlah tidak sesuai — Invoice dan PO memiliki selisih di luar toleransi', color: 'text-rose-700 bg-rose-50 border-rose-200' },
  qty_mismatch: { text: 'Qty tidak sesuai — Jumlah diterima (GR) berbeda dari PO', color: 'text-rose-700 bg-rose-50 border-rose-200' },
  no_po: { text: 'PO tidak ditemukan — Invoice tidak terkait dengan Purchase Order', color: 'text-amber-700 bg-amber-50 border-amber-200' },
  no_gr: { text: 'GR tidak ditemukan — Belum ada penerimaan barang untuk PO ini', color: 'text-amber-700 bg-amber-50 border-amber-200' },
}

const matchStatusLabels: Record<string, string> = {
  matched: 'Cocok',
  amount_mismatch: 'Jumlah Tidak Sesuai',
  qty_mismatch: 'Qty Tidak Sesuai',
  no_po: 'Tanpa PO',
  no_gr: 'Tanpa GR',
}

const matchStatusColorMap: Record<string, string> = {
  matched: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  amount_mismatch: 'bg-rose-50 text-rose-600 border-rose-200',
  qty_mismatch: 'bg-rose-50 text-rose-600 border-rose-200',
  no_po: 'bg-amber-50 text-amber-700 border-amber-200',
  no_gr: 'bg-amber-50 text-amber-700 border-amber-200',
}

// ── MatchResultCard ──────────────────────────────────────────────────────────
function MatchResultCard({ invoice, linkedPO }: { invoice: Invoice; linkedPO?: PO }) {
  if (invoice.match_result == null) return null

  const explanation = matchExplanations[invoice.match_result]

  return (
    <div className="col-span-full">
      <div className="bg-zinc-50 rounded-xl border border-zinc-200 p-4 space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <GitCompareArrows className="h-4 w-4 text-zinc-500" />
            <h3 className="text-sm font-semibold text-zinc-800">Hasil 3-Way Match</h3>
          </div>
          <StatusBadge
            status={invoice.match_result}
            colorMap={matchStatusColorMap}
            label={matchStatusLabels[invoice.match_result] ?? invoice.match_result}
          />
        </div>

        {explanation && (
          <p className={cn('text-xs font-medium px-3 py-2 rounded-lg border', explanation.color)}>
            {explanation.text}
          </p>
        )}

        {invoice.po_id && linkedPO && (
          <div className="grid grid-cols-3 gap-4 pt-2 border-t border-zinc-200">
            <div className="space-y-0.5">
              <p className="text-xs text-zinc-500 font-medium uppercase tracking-wide">PO Number</p>
              <Link
                href={`/dashboard/purchase-orders/${linkedPO.id}`}
                className="text-indigo-600 hover:text-indigo-700 font-mono text-sm inline-flex items-center gap-1"
              >
                {linkedPO.po_number}
                <ExternalLink className="h-3 w-3" />
              </Link>
            </div>
            <div className="space-y-0.5">
              <p className="text-xs text-zinc-500 font-medium uppercase tracking-wide">PO Amount</p>
              <p className="text-sm text-zinc-800 font-medium">
                {new Intl.NumberFormat('id-ID', {
                  style: 'currency',
                  currency: 'IDR',
                  maximumFractionDigits: 0,
                }).format(parseFloat(linkedPO.amount))}
              </p>
            </div>
            <div className="space-y-0.5">
              <p className="text-xs text-zinc-500 font-medium uppercase tracking-wide">PO Qty</p>
              <p className="text-sm text-zinc-800 font-medium">{linkedPO.qty}</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

export default function InvoiceDetailPage() {
  const params = useParams()
  const id = typeof params.id === 'string' ? params.id : ''
  const { data: invoice, isLoading, isError, error } = useInvoice(id)
  const { data: linkedPO } = usePO(invoice?.po_id ?? '')
  const { mutate: approve, isPending: isApprovePending, isSuccess: isApproveSuccess, isError: isApproveError } = useApproveInvoice()
  const { mutate: reject, isPending: isRejectPending, isSuccess: isRejectSuccess, isError: isRejectError } = useRejectInvoice()
  const [feedback, setFeedback] = useState<'idle' | 'approved' | 'rejected' | 'error'>('idle')
  const [showAiFailedConfirm, setShowAiFailedConfirm] = useState(false)

  function handleApprove() {
    if (invoice?.status === 'ai_failed') {
      setShowAiFailedConfirm(true)
      return
    }
    executeApprove()
  }

  function executeApprove() {
    setShowAiFailedConfirm(false)
    approve(id, {
      onSuccess: () => setFeedback('approved'),
      onError: () => setFeedback('error'),
    })
  }

  function handleReject() {
    reject(id, {
      onSuccess: () => setFeedback('rejected'),
      onError: () => setFeedback('error'),
    })
  }

  const canApprove = invoice?.status === 'pending'
  const canReject = invoice?.status === 'pending' || invoice?.status === 'pending_review'

  return (
    <div className="px-8 py-8 space-y-6 max-w-3xl">
      {/* Back nav */}
      <Link
        href="/dashboard/invoices"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Purchase Invoice
      </Link>

      {/* Loading */}
      {isLoading && (
        <div className="bg-white rounded-2xl border border-zinc-200/80 p-8 space-y-4 animate-pulse">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-5 bg-zinc-100 rounded-lg w-2/3" />
          ))}
        </div>
      )}

      {/* Error */}
      {isError && (
        <div className="bg-amber-50 border border-amber-200 rounded-2xl p-6 space-y-2">
          <p className="font-semibold text-amber-800 text-sm">
            Purchase invoice tidak dapat dimuat
          </p>
          <p className="text-xs text-amber-700">
            {getClientErrorMessage(error)}
          </p>
        </div>
      )}

      {/* Invoice detail */}
      {invoice && (
        <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
          {/* Header */}
          <div className="px-6 py-5 border-b border-zinc-100 flex items-start justify-between gap-4">
            <div>
              <h1 className="text-lg font-bold text-zinc-900 font-mono">
                {invoice.invoice_number}
              </h1>
              <p className="text-xs text-zinc-500 mt-0.5">ID: {invoice.id}</p>
            </div>
            {(() => {
              const cfg = statusConfig[invoice.status]
              const Icon = cfg.icon
              return (
                <span
                  className={cn(
                    'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-semibold border',
                    cfg.classes,
                  )}
                >
                  <Icon className="h-3.5 w-3.5" />
                  {cfg.label}
                </span>
              )
            })()}
          </div>

          {/* Fields */}
          <dl className="px-6 py-6 grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-5">
            <Field
              label="Jumlah"
              value={new Intl.NumberFormat('id-ID', {
                style: 'currency',
                currency: invoice.currency ?? 'IDR',
              }).format(parseFloat(invoice.amount))}
            />
            <Field label="Mata Uang" value={invoice.currency ?? 'IDR'} />
            <Field label="Vendor ID" value={<span className="font-mono">{invoice.vendor_id}</span>} />
            <Field label="Tenant ID" value={<span className="font-mono">{invoice.tenant_id}</span>} />
            {invoice.due_date && (
              <Field label="Jatuh Tempo" value={invoice.due_date.slice(0, 10)} />
            )}
            {invoice.ai_confidence_score != null && (
              <Field
                label="AI Confidence"
                value={`${(invoice.ai_confidence_score * 100).toFixed(1)}%`}
              />
            )}
            <MatchResultCard invoice={invoice} linkedPO={linkedPO} />
            <Field label="Dibuat" value={new Date(invoice.created_at).toLocaleString('id-ID')} />
            <Field label="Diperbarui" value={new Date(invoice.updated_at).toLocaleString('id-ID')} />
          </dl>

          {/* Action footer */}
          {(canApprove || canReject) && (
            <div className="px-6 py-4 border-t border-zinc-100 bg-zinc-50/50 flex items-center justify-between gap-4">
              <p className="text-xs text-zinc-500">
                {invoice.status === 'pending_review'
                  ? 'Purchase invoice memerlukan review manual. Setujui atau tolak.'
                  : 'Purchase invoice ini menunggu persetujuan.'}
              </p>
              <div className="flex items-center gap-2">
                {canApprove && (
                  <button
                    onClick={handleApprove}
                    disabled={isApprovePending || isApproveSuccess}
                    className="px-5 py-2 bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
                  >
                    {isApprovePending ? 'Memproses…' : isApproveSuccess ? 'Disetujui' : 'Setujui Purchase Invoice'}
                  </button>
                )}
                {canReject && (
                  <button
                    onClick={handleReject}
                    disabled={isRejectPending || isRejectSuccess}
                    className="px-5 py-2 bg-rose-600 hover:bg-rose-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
                  >
                    {isRejectPending ? 'Memproses…' : isRejectSuccess ? 'Ditolak' : 'Tolak Purchase Invoice'}
                  </button>
                )}
              </div>
            </div>
          )}

          {/* Feedback messages */}
          {feedback === 'approved' && (
            <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-emerald-50 border border-emerald-200 rounded-xl text-xs text-emerald-700 font-medium">
              Purchase invoice berhasil disetujui. Data diperbarui.
            </div>
          )}
          {feedback === 'rejected' && (
            <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700 font-medium">
              Purchase invoice berhasil ditolak. Data diperbarui.
            </div>
          )}
          {(feedback === 'error' || isApproveError || isRejectError) && (
            <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-amber-50 border border-amber-200 rounded-xl text-xs text-amber-700">
              Operasi gagal. Silakan coba lagi atau hubungi administrator.
            </div>
          )}
        </div>
      )}

      {/* In-app confirmation dialog for ai_failed invoice approval */}
      {showAiFailedConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4">
            <h3 className="text-base font-bold text-slate-900 flex items-center gap-2">
              <span className="text-amber-500">⚠️</span> Peringatan Verifikasi Dokumen
            </h3>
            <p className="text-sm text-slate-600 leading-relaxed">
              Purchase invoice ini tidak terverifikasi dokumennya karena proses OCR gagal. Data keuangan dalam invoice ini belum divalidasi oleh sistem. Anda yakin ingin tetap menyetujui invoice ini?
            </p>
            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                type="button"
                onClick={() => setShowAiFailedConfirm(false)}
                className="px-4 py-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50 text-sm font-medium transition"
              >
                Batal
              </button>
              <button
                type="button"
                disabled={isApprovePending}
                onClick={executeApprove}
                className="px-4 py-2 rounded-lg bg-emerald-600 text-white text-sm font-semibold hover:bg-emerald-700 shadow-sm transition disabled:opacity-50"
              >
                {isApprovePending ? 'Memproses...' : 'Tetap Setujui'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
