'use client'

import { useState } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, CheckCircle, Clock, XCircle, CreditCard, Ban } from 'lucide-react'
import { usePaymentOrder, useApprovePaymentOrder, usePayPaymentOrder, useRejectPaymentOrder } from '@/lib/queries/payment-orders'
import { type PaymentOrderStatus, paymentMethodLabels } from '@/lib/schemas/payment-order'
import { cn } from '@/lib/utils'
import { getClientErrorMessage } from '@/lib/api/errors'

// ── Status config ─────────────────────────────────────────────────────────────
const statusConfig: Record<
  PaymentOrderStatus,
  { label: string; icon: React.ElementType; classes: string }
> = {
  draft: {
    label: 'Draft',
    icon: Clock,
    classes: 'bg-zinc-50 text-zinc-700 border-zinc-200',
  },
  pending_approval: {
    label: 'Menunggu Persetujuan',
    icon: Clock,
    classes: 'bg-amber-50 text-amber-700 border-amber-200',
  },
  approved: {
    label: 'Disetujui',
    icon: CheckCircle,
    classes: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  },
  paid: {
    label: 'Dibayar',
    icon: CreditCard,
    classes: 'bg-indigo-50 text-indigo-700 border-indigo-200',
  },
  rejected: {
    label: 'Ditolak',
    icon: XCircle,
    classes: 'bg-rose-50 text-rose-600 border-rose-200',
  },
  cancelled: {
    label: 'Dibatalkan',
    icon: Ban,
    classes: 'bg-zinc-50 text-zinc-500 border-zinc-200',
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

export default function PaymentOrderDetailPage() {
  const params = useParams()
  const id = typeof params.id === 'string' ? params.id : ''
  const { data: po, isLoading, isError, error } = usePaymentOrder(id)
  const { mutate: approve, isPending: isApprovePending, isSuccess: isApproveSuccess, isError: isApproveError } = useApprovePaymentOrder()
  const { mutate: pay, isPending: isPayPending, isSuccess: isPaySuccess, isError: isPayError } = usePayPaymentOrder()
  const { mutate: reject, isPending: isRejectPending, isSuccess: isRejectSuccess, isError: isRejectError } = useRejectPaymentOrder()
  const [feedback, setFeedback] = useState<'idle' | 'approved' | 'paid' | 'rejected' | 'error'>('idle')

  function handleApprove() {
    approve(id, {
      onSuccess: () => setFeedback('approved'),
      onError: () => setFeedback('error'),
    })
  }

  function handlePay() {
    pay(id, {
      onSuccess: () => setFeedback('paid'),
      onError: () => setFeedback('error'),
    })
  }

  function handleReject() {
    reject(id, {
      onSuccess: () => setFeedback('rejected'),
      onError: () => setFeedback('error'),
    })
  }

  const canApprove = po?.status === 'draft'
  const canPay = po?.status === 'approved'
  const canReject = po?.status === 'draft'

  return (
    <div className="px-8 py-8 space-y-6 max-w-3xl">
      {/* Back nav */}
      <Link
        href="/dashboard/payment-orders"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Payment Orders
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
            Payment Order tidak dapat dimuat
          </p>
          <p className="text-xs text-amber-700">
            {getClientErrorMessage(error)}
          </p>
        </div>
      )}

      {/* Payment order detail */}
      {po && (
        <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
          {/* Header */}
          <div className="px-6 py-5 border-b border-zinc-100 flex items-start justify-between gap-4">
            <div>
              <h1 className="text-lg font-bold text-zinc-900 font-mono">
                Payment Order
              </h1>
              <p className="text-xs text-zinc-500 mt-0.5">ID: {po.id}</p>
            </div>
            {(() => {
              const cfg = statusConfig[po.status]
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
                currency: po.currency ?? 'IDR',
              }).format(parseFloat(po.amount))}
            />
            <Field label="Mata Uang" value={po.currency ?? 'IDR'} />
            <Field label="Metode Pembayaran" value={paymentMethodLabels[po.payment_method] ?? po.payment_method} />
            <Field label="Invoice ID" value={<span className="font-mono">{po.invoice_id}</span>} />
            {po.reference_number && (
              <Field label="Nomor Referensi" value={po.reference_number} />
            )}
            {po.notes && (
              <Field label="Catatan" value={po.notes} />
            )}
            <Field label="Tenant ID" value={<span className="font-mono">{po.tenant_id}</span>} />
            {po.created_by && (
              <Field label="Dibuat Oleh" value={<span className="font-mono">{po.created_by}</span>} />
            )}
            {po.approved_by && (
              <Field label="Disetujui Oleh" value={<span className="font-mono">{po.approved_by}</span>} />
            )}
            {po.paid_at && (
              <Field label="Dibayar Pada" value={new Date(po.paid_at).toLocaleString('id-ID')} />
            )}
            <Field label="Dibuat" value={new Date(po.created_at).toLocaleString('id-ID')} />
            <Field label="Diperbarui" value={new Date(po.updated_at).toLocaleString('id-ID')} />
          </dl>

          {/* Action footer */}
          {(canApprove || canPay || canReject) && (
            <div className="px-6 py-4 border-t border-zinc-100 bg-zinc-50/50 flex items-center justify-between gap-4">
              <p className="text-xs text-zinc-500">
                {canApprove && 'Payment order ini menunggu persetujuan.'}
                {canPay && 'Payment order telah disetujui. Siap untuk dibayar.'}
              </p>
              <div className="flex items-center gap-2">
                {canApprove && (
                  <button
                    onClick={handleApprove}
                    disabled={isApprovePending || isApproveSuccess}
                    className="px-5 py-2 bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
                  >
                    {isApprovePending ? 'Memproses...' : isApproveSuccess ? 'Disetujui' : 'Setujui'}
                  </button>
                )}
                {canPay && (
                  <button
                    onClick={handlePay}
                    disabled={isPayPending || isPaySuccess}
                    className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
                  >
                    {isPayPending ? 'Memproses...' : isPaySuccess ? 'Dibayar' : 'Bayar'}
                  </button>
                )}
                {canReject && (
                  <button
                    onClick={handleReject}
                    disabled={isRejectPending || isRejectSuccess}
                    className="px-5 py-2 bg-rose-600 hover:bg-rose-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
                  >
                    {isRejectPending ? 'Memproses...' : isRejectSuccess ? 'Ditolak' : 'Tolak'}
                  </button>
                )}
              </div>
            </div>
          )}

          {/* Feedback messages */}
          {feedback === 'approved' && (
            <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-emerald-50 border border-emerald-200 rounded-xl text-xs text-emerald-700 font-medium">
              Payment order berhasil disetujui. Data diperbarui.
            </div>
          )}
          {feedback === 'paid' && (
            <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-indigo-50 border border-indigo-200 rounded-xl text-xs text-indigo-700 font-medium">
              Payment order berhasil dibayar. Invoice telah ditandai sebagai lunas.
            </div>
          )}
          {feedback === 'rejected' && (
            <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700 font-medium">
              Payment order berhasil ditolak. Data diperbarui.
            </div>
          )}
          {(feedback === 'error' || isApproveError || isPayError || isRejectError) && (
            <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-amber-50 border border-amber-200 rounded-xl text-xs text-amber-700">
              Operasi gagal. Silakan coba lagi atau hubungi administrator.
            </div>
          )}
        </div>
      )}
    </div>
  )
}
