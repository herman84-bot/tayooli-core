'use client'

import { useParams } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft } from 'lucide-react'
import { usePO } from '@/lib/queries/po'
import { StatusBadge } from '@/components/ui/status-badge'
import { getClientErrorMessage } from '@/lib/api/errors'

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs text-zinc-500 font-medium uppercase tracking-wide">{label}</dt>
      <dd className="text-sm text-zinc-800 font-medium">{value}</dd>
    </div>
  )
}

export default function PurchaseOrderDetailPage() {
  const params = useParams()
  const id = typeof params.id === 'string' ? params.id : ''
  const { data: po, isLoading, isError, error } = usePO(id)

  return (
    <div className="px-8 py-8 space-y-6 max-w-3xl">
      {/* Back nav */}
      <Link
        href="/dashboard/purchase-orders"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Purchase Order
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
          <p className="font-semibold text-amber-800 text-sm">Purchase order tidak dapat dimuat</p>
          <p className="text-xs text-amber-700">{getClientErrorMessage(error)}</p>
        </div>
      )}

      {/* Detail */}
      {po && (
        <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
          {/* Header */}
          <div className="px-6 py-5 border-b border-zinc-100 flex items-start justify-between gap-4">
            <div>
              <h1 className="text-lg font-bold text-zinc-900 font-mono">{po.po_number}</h1>
              <p className="text-xs text-zinc-500 mt-0.5">ID: {po.id}</p>
            </div>
            <StatusBadge status={po.status} />
          </div>

          {/* Fields */}
          <dl className="px-6 py-6 grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-5">
            <Field label="Nomor PO" value={<span className="font-mono">{po.po_number}</span>} />
            <Field label="Vendor ID" value={<span className="font-mono">{po.vendor_id}</span>} />
            <Field
              label="Jumlah"
              value={new Intl.NumberFormat('id-ID', {
                style: 'currency',
                currency: po.currency ?? 'IDR',
              }).format(parseFloat(po.amount))}
            />
            <Field label="Qty" value={po.qty} />
            <Field label="Mata Uang" value={po.currency} />
            <Field label="Status" value={<StatusBadge status={po.status} />} />
            <Field
              label="Dibuat"
              value={new Date(po.created_at).toLocaleString('id-ID')}
            />
            <Field
              label="Diperbarui"
              value={new Date(po.updated_at).toLocaleString('id-ID')}
            />
          </dl>
        </div>
      )}
    </div>
  )
}
