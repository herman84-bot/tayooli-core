'use client'

import { useState } from 'react'
import Link from 'next/link'
import { usePaymentOrders, useApprovePaymentOrder, useRejectPaymentOrder, usePayPaymentOrder } from '@/lib/queries/payment-orders'
import { type PaymentOrderStatus, paymentMethodLabels } from '@/lib/schemas/payment-order'
import { cn } from '@/lib/utils'
import { getClientErrorMessage } from '@/lib/api/errors'
import { Pagination } from '@/components/ui/pagination'

// ── Status badge ─────────────────────────────────────────────────────────────
const statusStyles: Record<PaymentOrderStatus, string> = {
  draft: 'bg-zinc-50 text-zinc-700 border-zinc-200',
  pending_approval: 'bg-amber-50 text-amber-700 border-amber-200',
  approved: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  paid: 'bg-indigo-50 text-indigo-700 border-indigo-200',
  rejected: 'bg-rose-50 text-rose-600 border-rose-200',
  cancelled: 'bg-zinc-50 text-zinc-500 border-zinc-200',
}

const statusLabel: Record<PaymentOrderStatus, string> = {
  draft: 'Draft',
  pending_approval: 'Menunggu Persetujuan',
  approved: 'Disetujui',
  paid: 'Dibayar',
  rejected: 'Ditolak',
  cancelled: 'Dibatalkan',
}

function StatusBadge({ status }: { status: PaymentOrderStatus }) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold border',
        statusStyles[status],
      )}
    >
      {statusLabel[status]}
    </span>
  )
}

// ── Skeleton row ─────────────────────────────────────────────────────────────
function SkeletonRow() {
  return (
    <tr>
      {Array.from({ length: 7 }).map((_, i) => (
        <td key={i} className="px-4 py-3">
          <div className="h-4 bg-zinc-100 rounded animate-pulse" />
        </td>
      ))}
    </tr>
  )
}

// ── Action buttons ───────────────────────────────────────────────────────────
function ApproveButton({ id, status }: { id: string; status: PaymentOrderStatus }) {
  const { mutate, isPending } = useApprovePaymentOrder()
  if (status !== 'draft') return null

  return (
    <button
      onClick={() => mutate(id)}
      disabled={isPending}
      className="text-xs font-semibold px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
    >
      {isPending ? 'Memproses...' : 'Setujui'}
    </button>
  )
}

function PayButton({ id, status }: { id: string; status: PaymentOrderStatus }) {
  const { mutate, isPending } = usePayPaymentOrder()
  if (status !== 'approved') return null

  return (
    <button
      onClick={() => mutate(id)}
      disabled={isPending}
      className="text-xs font-semibold px-3 py-1.5 rounded-lg bg-indigo-600 hover:bg-indigo-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
    >
      {isPending ? 'Memproses...' : 'Bayar'}
    </button>
  )
}

function RejectButton({ id, status }: { id: string; status: PaymentOrderStatus }) {
  const { mutate, isPending } = useRejectPaymentOrder()
  if (status !== 'draft') return null

  return (
    <button
      onClick={() => mutate(id)}
      disabled={isPending}
      className="text-xs font-semibold px-3 py-1.5 rounded-lg bg-rose-600 hover:bg-rose-700 text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
    >
      {isPending ? 'Memproses...' : 'Tolak'}
    </button>
  )
}

// ── Main page ────────────────────────────────────────────────────────────────
export default function PaymentOrdersPage() {
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(20)
  const { data: orderList, isLoading, isError, error } = usePaymentOrders({ page, perPage })
  const orders = orderList?.data ?? []
  const total = orderList?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / perPage))
  const startIndex = total > 0 ? (page - 1) * perPage + 1 : 0
  const endIndex = Math.min(page * perPage, total)

  return (
    <div className="px-8 py-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-zinc-900">Payment Orders</h1>
          <p className="text-sm text-zinc-500 mt-1">Daftar perintah pembayaran dari API.</p>
        </div>
        <Link href="/dashboard/payment-orders/create" className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors">
          + Buat Payment Order
        </Link>
      </div>

      <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-100 bg-zinc-50/70">
                {['Invoice ID', 'Jumlah', 'Metode', 'Status', 'Dibayar', 'Dibuat', 'Aksi'].map((h) => (
                  <th
                    key={h}
                    className="px-4 py-3 text-left text-xs font-semibold text-zinc-500 uppercase tracking-wider"
                  >
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {isLoading &&
                Array.from({ length: 5 }).map((_, i) => <SkeletonRow key={i} />)}

              {isError && (
                <tr>
                  <td colSpan={7} className="px-4 py-10 text-center">
                    <div className="space-y-2">
                      <p className="text-sm font-semibold text-zinc-700">
                        Tidak dapat memuat data payment orders
                      </p>
                      <p className="text-xs text-zinc-400">
                        {getClientErrorMessage(error)}
                      </p>
                    </div>
                  </td>
                </tr>
              )}

              {!isLoading && !isError && orders.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-10 text-center text-sm text-zinc-400">
                    Tidak ada payment order ditemukan.
                  </td>
                </tr>
              )}

              {orders.map((po) => (
                <tr key={po.id} className="hover:bg-zinc-50/60 transition-colors">
                  <td className="px-4 py-3 font-mono text-zinc-700 font-medium">
                    <Link
                      href={`/dashboard/payment-orders/${po.id}`}
                      className="hover:text-indigo-600 transition-colors"
                    >
                      {po.invoice_id.slice(0, 8)}...
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-zinc-800 font-medium">
                    {new Intl.NumberFormat('id-ID', {
                      style: 'currency',
                      currency: po.currency ?? 'IDR',
                      maximumFractionDigits: 0,
                    }).format(parseFloat(po.amount))}
                  </td>
                  <td className="px-4 py-3 text-zinc-600 text-xs">
                    {paymentMethodLabels[po.payment_method] ?? po.payment_method}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={po.status} />
                  </td>
                  <td className="px-4 py-3 text-zinc-500 text-xs">
                    {po.paid_at ? po.paid_at.slice(0, 10) : '--'}
                  </td>
                  <td className="px-4 py-3 text-zinc-500 text-xs">
                    {po.created_at.slice(0, 10)}
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <ApproveButton id={po.id} status={po.status} />
                      <PayButton id={po.id} status={po.status} />
                      <RejectButton id={po.id} status={po.status} />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Pagination */}
      {!isLoading && !isError && total > 0 && (
        <div className="flex items-center justify-between">
          <p className="text-xs text-zinc-500">
            Menampilkan {startIndex}-{endIndex} dari {total}
          </p>
          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={setPage}
            perPage={perPage}
            onPerPageChange={(v) => {
              setPerPage(v)
              setPage(1)
            }}
          />
        </div>
      )}
    </div>
  )
}
