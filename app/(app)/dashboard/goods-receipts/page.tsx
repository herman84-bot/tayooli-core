'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useGRs } from '@/lib/queries/gr'
import { StatusBadge } from '@/components/ui/status-badge'
import { Pagination } from '@/components/ui/pagination'
import { getClientErrorMessage } from '@/lib/api/errors'

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

export default function GoodsReceiptsPage() {
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(20)
  const { data: list, isLoading, isError, error } = useGRs({ page, perPage })
  const grs = list?.data ?? []
  const total = list?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / perPage))
  const startIndex = total > 0 ? (page - 1) * perPage + 1 : 0
  const endIndex = Math.min(page * perPage, total)

  return (
    <div className="px-8 py-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-zinc-900">Goods Receipts</h1>
          <p className="text-sm text-zinc-500 mt-1">Semua goods receipt dari API.</p>
        </div>
        <Link
          href="/dashboard/goods-receipts/new"
          className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors"
        >
          + Buat GR
        </Link>
      </div>

      <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-100 bg-zinc-50/70">
                {['GR ID', 'PO ID', 'Vendor', 'Jumlah Diterima', 'Qty', 'Status', 'Diterima Pada'].map(
                  (h) => (
                    <th
                      key={h}
                      className="px-4 py-3 text-left text-xs font-semibold text-zinc-500 uppercase tracking-wider"
                    >
                      {h}
                    </th>
                  ),
                )}
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-100">
              {isLoading && Array.from({ length: 5 }).map((_, i) => <SkeletonRow key={i} />)}

              {isError && (
                <tr>
                  <td colSpan={7} className="px-4 py-10 text-center">
                    <div className="space-y-2">
                      <p className="text-sm font-semibold text-zinc-700">
                        Tidak dapat memuat data goods receipt
                      </p>
                      <p className="text-xs text-zinc-400">{getClientErrorMessage(error)}</p>
                    </div>
                  </td>
                </tr>
              )}

              {!isLoading && !isError && grs?.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-10 text-center text-sm text-zinc-400">
                    Tidak ada goods receipt ditemukan.
                  </td>
                </tr>
              )}

              {grs?.map((gr) => (
                <tr key={gr.id} className="hover:bg-zinc-50/60 transition-colors">
                  <td className="px-4 py-3 font-mono text-zinc-700 font-medium">
                    <Link
                      href={`/dashboard/goods-receipts/${gr.id}`}
                      className="hover:text-indigo-600 transition-colors"
                    >
                      {gr.id.slice(0, 8)}…
                    </Link>
                  </td>
                  <td className="px-4 py-3 font-mono text-zinc-500 text-xs">
                    {gr.po_id.slice(0, 8)}…
                  </td>
                  <td className="px-4 py-3 text-zinc-600">{gr.vendor_id.slice(0, 8)}…</td>
                  <td className="px-4 py-3 text-zinc-800 font-medium">
                    {new Intl.NumberFormat('id-ID', {
                      style: 'currency',
                      currency: gr.currency ?? 'IDR',
                      maximumFractionDigits: 0,
                    }).format(parseFloat(gr.received_amount))}
                  </td>
                  <td className="px-4 py-3 text-zinc-600">{gr.received_qty}</td>
                  <td className="px-4 py-3">
                    <StatusBadge status={gr.status} />
                  </td>
                  <td className="px-4 py-3 text-zinc-500 text-xs">
                    {new Date(gr.received_at).toLocaleDateString('id-ID')}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

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
            onPerPageChange={(pp) => {
              setPerPage(pp)
              setPage(1)
            }}
          />
        </div>
      )}
    </div>
  )
}
