'use client'

import { useState, useCallback } from 'react'
import Link from 'next/link'
import { useVendors } from '@/lib/queries/vendors'
import { type VendorStatus, statusLabels } from '@/lib/schemas/vendor'
import { cn } from '@/lib/utils'
import { getClientErrorMessage } from '@/lib/api/errors'
import { Pagination } from '@/components/ui/pagination'

// ── Status badge ─────────────────────────────────────────────────────────────
const statusStyles: Record<VendorStatus, string> = {
  active: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  inactive: 'bg-zinc-50 text-zinc-500 border-zinc-200',
  suspended: 'bg-amber-50 text-amber-700 border-amber-200',
}

function StatusBadge({ status }: { status: VendorStatus }) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold border',
        statusStyles[status],
      )}
    >
      {statusLabels[status]}
    </span>
  )
}

// ── Rating stars ─────────────────────────────────────────────────────────────
function RatingStars({ avg, count }: { avg: string; count: number }) {
  const rating = parseFloat(avg)
  return (
    <span className="inline-flex items-center gap-1.5 text-xs text-zinc-600">
      <span className="text-amber-500">
        {'★'.repeat(Math.round(rating))}
        {'☆'.repeat(5 - Math.round(rating))}
      </span>
      <span className="text-zinc-400">({count})</span>
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

// ── Main page ────────────────────────────────────────────────────────────────
export default function VendorsPage() {
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const [searchInput, setSearchInput] = useState('')
  const [perPage, setPerPage] = useState(20)
  const { data: vendorList, isLoading, isError, error } = useVendors({ page, perPage, search: search || undefined })
  const vendors = vendorList?.data ?? []
  const total = vendorList?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / perPage))
  const startIndex = total > 0 ? (page - 1) * perPage + 1 : 0
  const endIndex = Math.min(page * perPage, total)

  const handleSearch = useCallback((e: React.FormEvent) => {
    e.preventDefault()
    setSearch(searchInput)
    setPage(1)
  }, [searchInput])

  const handleClearSearch = useCallback(() => {
    setSearchInput('')
    setSearch('')
    setPage(1)
  }, [])

  return (
    <div className="px-8 py-8 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-zinc-900">Vendors</h1>
          <p className="text-sm text-zinc-500 mt-1">Daftar vendor dari API.</p>
        </div>
        <Link href="/dashboard/vendors/create" className="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors">
          + Tambah Vendor
        </Link>
      </div>

      {/* Search bar */}
      <form onSubmit={handleSearch} className="flex items-center gap-2">
        <input
          type="text"
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          placeholder="Cari vendor berdasarkan nama..."
          className="w-full max-w-sm rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
        />
        <button
          type="submit"
          className="px-4 py-2.5 bg-zinc-100 hover:bg-zinc-200 text-zinc-700 text-sm font-medium rounded-xl transition-colors"
        >
          Cari
        </button>
        {search && (
          <button
            type="button"
            onClick={handleClearSearch}
            className="px-4 py-2.5 bg-zinc-100 hover:bg-zinc-200 text-zinc-700 text-sm font-medium rounded-xl transition-colors"
          >
            Hapus
          </button>
        )}
      </form>

      <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm">
            <thead>
              <tr className="border-b border-zinc-100 bg-zinc-50/70">
                {['Nama', 'Email', 'Telepon', 'Alamat', 'Status', 'Rating', 'Dibuat'].map((h) => (
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
                        Tidak dapat memuat data vendor
                      </p>
                      <p className="text-xs text-zinc-400">
                        {getClientErrorMessage(error)}
                      </p>
                    </div>
                  </td>
                </tr>
              )}

              {!isLoading && !isError && vendors.length === 0 && (
                <tr>
                  <td colSpan={7} className="px-4 py-10 text-center text-sm text-zinc-400">
                    {search ? `Tidak ada vendor ditemukan untuk "${search}".` : 'Tidak ada vendor ditemukan.'}
                  </td>
                </tr>
              )}

              {vendors.map((v) => (
                <tr key={v.id} className="hover:bg-zinc-50/60 transition-colors">
                  <td className="px-4 py-3 font-medium text-zinc-800">
                    <Link
                      href={`/dashboard/vendors/${v.id}`}
                      className="hover:text-indigo-600 transition-colors"
                    >
                      {v.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-zinc-600 text-xs">
                    {v.email ?? '--'}
                  </td>
                  <td className="px-4 py-3 text-zinc-600 text-xs">
                    {v.phone ?? '--'}
                  </td>
                  <td className="px-4 py-3 text-zinc-600 text-xs max-w-[200px] truncate" title={v.address || undefined}>
                    {v.address ?? '--'}
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge status={v.status} />
                  </td>
                  <td className="px-4 py-3">
                    <RatingStars avg={v.avg_rating} count={v.rating_count} />
                  </td>
                  <td className="px-4 py-3 text-zinc-500 text-xs">
                    {v.created_at.slice(0, 10)}
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
