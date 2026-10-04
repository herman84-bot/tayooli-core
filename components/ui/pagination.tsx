'use client'

import { cn } from '@/lib/utils'

export interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
  /** Optional per-page selector (10/20/50). When provided, renders next to the pager. */
  perPage?: number
  onPerPageChange?: (perPage: number) => void
  perPageOptions?: number[]
}

function getPageNumbers(page: number, totalPages: number): (number | 'ellipsis')[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1)
  }

  const pages: (number | 'ellipsis')[] = [1]

  if (page > 3) {
    pages.push('ellipsis')
  }

  const start = Math.max(2, page - 1)
  const end = Math.min(totalPages - 1, page + 1)

  for (let i = start; i <= end; i++) {
    pages.push(i)
  }

  if (page < totalPages - 2) {
    pages.push('ellipsis')
  }

  pages.push(totalPages)

  return pages
}

export function Pagination({
  page,
  totalPages,
  onPageChange,
  perPage,
  onPerPageChange,
  perPageOptions = [10, 20, 50],
}: PaginationProps) {
  const pages = totalPages > 1 ? getPageNumbers(page, totalPages) : []

  // Dengan selector ukuran halaman, tampilkan selector walau hanya 1 halaman.
  if (totalPages <= 1 && !onPerPageChange) return null

  const pageSizeSelect = onPerPageChange && perPage ? (
    <label className="flex items-center gap-1.5 text-xs text-zinc-500">
      <span>Per halaman</span>
      <select
        value={perPage}
        onChange={(e) => onPerPageChange(Number(e.target.value))}
        aria-label="Ukuran halaman"
        className="px-2 py-1 rounded-lg text-sm bg-zinc-100 text-zinc-700 hover:bg-zinc-200 border border-zinc-200 focus:outline-none focus:ring-1 focus:ring-indigo-500 transition-colors"
      >
        {perPageOptions.map((opt) => (
          <option key={opt} value={opt}>
            {opt}
          </option>
        ))}
      </select>
    </label>
  ) : null

  return (
    <div className="flex items-center justify-between gap-4">
      {pageSizeSelect}
      {totalPages > 1 && (
        <nav className="flex items-center justify-center gap-1" aria-label="Pagination">
      <button
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        className="px-3 py-1 rounded-lg text-sm bg-zinc-100 text-zinc-700 hover:bg-zinc-200 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
      >
        Sebelumnya
      </button>

      {pages.map((p, i) =>
        p === 'ellipsis' ? (
          <span key={`ellipsis-${i}`} className="px-2 py-1 text-sm text-zinc-400">
            ...
          </span>
        ) : (
          <button
            key={p}
            onClick={() => onPageChange(p)}
            className={cn(
              'px-3 py-1 rounded-lg text-sm transition-colors',
              p === page
                ? 'bg-indigo-600 text-white'
                : 'bg-zinc-100 text-zinc-700 hover:bg-zinc-200',
            )}
            aria-current={p === page ? 'page' : undefined}
          >
            {p}
          </button>
        ),
      )}

      <button
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        className="px-3 py-1 rounded-lg text-sm bg-zinc-100 text-zinc-700 hover:bg-zinc-200 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
      >
        Selanjutnya
      </button>
        </nav>
      )}
    </div>
  )
}
