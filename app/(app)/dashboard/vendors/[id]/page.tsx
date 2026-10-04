'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { ArrowLeft, Star } from 'lucide-react'
import { useVendor, useDeleteVendor, useRateVendor } from '@/lib/queries/vendors'
import { type VendorStatus, statusLabels, RateVendorInputSchema } from '@/lib/schemas/vendor'
import { cn } from '@/lib/utils'
import { getClientErrorMessage } from '@/lib/api/errors'

// ── Status config ─────────────────────────────────────────────────────────────
const statusConfig: Record<VendorStatus, { label: string; classes: string }> = {
  active: { label: 'Aktif', classes: 'bg-emerald-50 text-emerald-700 border-emerald-200' },
  inactive: { label: 'Nonaktif', classes: 'bg-zinc-50 text-zinc-500 border-zinc-200' },
  suspended: { label: 'Ditangguhkan', classes: 'bg-amber-50 text-amber-700 border-amber-200' },
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

// ── Rating display ───────────────────────────────────────────────────────────
function RatingDisplay({ avg, count }: { avg: string; count: number }) {
  const rating = parseFloat(avg)
  return (
    <div className="flex items-center gap-2">
      <span className="text-amber-500 text-lg">
        {'★'.repeat(Math.round(rating))}
        {'☆'.repeat(5 - Math.round(rating))}
      </span>
      <span className="text-sm text-zinc-600 font-medium">{avg}</span>
      <span className="text-xs text-zinc-400">({count} rating)</span>
    </div>
  )
}

export default function VendorDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = typeof params.id === 'string' ? params.id : ''
  const { data: vendor, isLoading, isError, error } = useVendor(id)
  const { mutate: deleteVendor, isPending: isDeletePending } = useDeleteVendor()
  const { mutate: rateVendor, isPending: isRatePending } = useRateVendor()

  const [feedback, setFeedback] = useState<'idle' | 'deleted' | 'rated' | 'error'>('idle')
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [ratingValue, setRatingValue] = useState(5)
  const [ratingComment, setRatingComment] = useState('')
  const [ratingError, setRatingError] = useState<string | null>(null)

  function handleDelete() {
    setShowDeleteConfirm(true)
  }

  function executeDelete() {
    setShowDeleteConfirm(false)
    deleteVendor(id, {
      onSuccess: () => {
        setFeedback('deleted')
        setTimeout(() => router.push('/dashboard/vendors'), 1500)
      },
      onError: () => setFeedback('error'),
    })
  }

  function handleRate(e: React.FormEvent) {
    e.preventDefault()
    setRatingError(null)

    const result = RateVendorInputSchema.safeParse({ rating: ratingValue, comment: ratingComment || undefined })
    if (!result.success) {
      setRatingError(result.error.issues[0]?.message ?? 'Rating tidak valid')
      return
    }

    rateVendor(
      { id, data: result.data },
      {
        onSuccess: () => {
          setFeedback('rated')
          setRatingComment('')
        },
        onError: (err) => {
          setRatingError(getClientErrorMessage(err))
        },
      },
    )
  }

  return (
    <div className="px-8 py-8 space-y-6 max-w-3xl">
      {/* Back nav */}
      <Link
        href="/dashboard/vendors"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Vendor
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
            Vendor tidak dapat dimuat
          </p>
          <p className="text-xs text-amber-700">
            {getClientErrorMessage(error)}
          </p>
        </div>
      )}

      {/* Vendor detail */}
      {vendor && (
        <>
          <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
            {/* Header */}
            <div className="px-6 py-5 border-b border-zinc-100 flex items-start justify-between gap-4">
              <div>
                <h1 className="text-lg font-bold text-zinc-900">{vendor.name}</h1>
                <p className="text-xs text-zinc-500 mt-0.5">ID: {vendor.id}</p>
              </div>
              <span
                className={cn(
                  'inline-flex items-center px-3 py-1.5 rounded-full text-xs font-semibold border',
                  statusConfig[vendor.status].classes,
                )}
              >
                {statusConfig[vendor.status].label}
              </span>
            </div>

            {/* Fields */}
            <dl className="px-6 py-6 grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-5">
              <Field label="Rating" value={<RatingDisplay avg={vendor.avg_rating} count={vendor.rating_count} />} />
              <Field label="Tenant ID" value={<span className="font-mono">{vendor.tenant_id}</span>} />
              {vendor.email && <Field label="Email" value={vendor.email} />}
              {vendor.phone && <Field label="Telepon" value={vendor.phone} />}
              {vendor.address && <Field label="Alamat" value={vendor.address} />}
              {vendor.bank_account && <Field label="No. Rekening" value={vendor.bank_account} />}
              {vendor.bank_name && <Field label="Nama Bank" value={vendor.bank_name} />}
              {vendor.tax_id && <Field label="NPWP" value={vendor.tax_id} />}
              <Field label="Dibuat" value={new Date(vendor.created_at).toLocaleString('id-ID')} />
              <Field label="Diperbarui" value={new Date(vendor.updated_at).toLocaleString('id-ID')} />
            </dl>

            {/* Action footer */}
            {vendor.status === 'active' && (
              <div className="px-6 py-4 border-t border-zinc-100 bg-zinc-50/50 flex items-center justify-between gap-4">
                <Link
                  href={`/dashboard/vendors/${id}/edit`}
                  className="px-5 py-2 bg-zinc-600 hover:bg-zinc-700 text-white text-sm font-semibold rounded-xl transition-colors shadow-sm"
                >
                  Edit
                </Link>
                <button
                  onClick={handleDelete}
                  disabled={isDeletePending}
                  className="px-5 py-2 bg-rose-600 hover:bg-rose-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
                >
                  {isDeletePending ? 'Memproses...' : 'Nonaktifkan'}
                </button>
              </div>
            )}

            {/* Feedback messages */}
            {feedback === 'deleted' && (
              <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-emerald-50 border border-emerald-200 rounded-xl text-xs text-emerald-700 font-medium">
                Vendor berhasil dinonaktifkan. Mengarahkan ke daftar vendor...
              </div>
            )}
            {feedback === 'error' && (
              <div className="mx-6 mb-4 mt-2 px-4 py-3 bg-amber-50 border border-amber-200 rounded-xl text-xs text-amber-700">
                Operasi gagal. Silakan coba lagi atau hubungi administrator.
              </div>
            )}
          </div>

          {/* Rating form */}
          <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
            <div className="px-6 py-4 border-b border-zinc-100">
              <h2 className="text-sm font-semibold text-zinc-900">Beri Rating Vendor</h2>
            </div>
            <form onSubmit={handleRate} className="px-6 py-5 space-y-4">
              {/* Star selector */}
              <div>
                <label className="block text-xs font-semibold text-zinc-600 mb-2">Rating</label>
                <div className="flex items-center gap-1">
                  {[1, 2, 3, 4, 5].map((star) => (
                    <button
                      key={star}
                      type="button"
                      onClick={() => setRatingValue(star)}
                      className="p-0.5 transition-colors"
                    >
                      <Star
                        className={cn(
                          'h-6 w-6',
                          star <= ratingValue
                            ? 'fill-amber-400 text-amber-400'
                            : 'fill-none text-zinc-300',
                        )}
                      />
                    </button>
                  ))}
                </div>
              </div>

              {/* Comment */}
              <div>
                <label htmlFor="rating-comment" className="block text-xs font-semibold text-zinc-600 mb-1.5">
                  Komentar <span className="text-zinc-400 font-normal">(opsional)</span>
                </label>
                <textarea
                  id="rating-comment"
                  rows={2}
                  value={ratingComment}
                  onChange={(e) => setRatingComment(e.target.value)}
                  placeholder="Bagaimana pengalaman Anda dengan vendor ini?"
                  className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors resize-none"
                />
              </div>

              {ratingError && (
                <p className="text-xs text-rose-600">{ratingError}</p>
              )}

              {feedback === 'rated' && (
                <div className="px-4 py-3 bg-emerald-50 border border-emerald-200 rounded-xl text-xs text-emerald-700 font-medium">
                  Rating berhasil dikirim. Terima kasih!
                </div>
              )}

              <button
                type="submit"
                disabled={isRatePending}
                className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {isRatePending ? 'Mengirim...' : 'Kirim Rating'}
              </button>
            </form>
          </div>
        </>
      )}

      {/* In-app confirmation dialog for deleting vendor */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="w-full max-w-sm bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4">
            <h3 className="text-base font-bold text-slate-900">Konfirmasi Nonaktifkan Vendor</h3>
            <p className="text-sm text-slate-600">
              Apakah Anda yakin ingin menonaktifkan vendor ini? Data riwayat tetap tersimpan.
            </p>
            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                type="button"
                onClick={() => setShowDeleteConfirm(false)}
                className="px-4 py-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50 text-sm font-medium transition"
              >
                Batal
              </button>
              <button
                type="button"
                disabled={isDeletePending}
                onClick={executeDelete}
                className="px-4 py-2 rounded-lg bg-rose-600 text-white text-sm font-semibold hover:bg-rose-700 shadow-sm transition disabled:opacity-50"
              >
                {isDeletePending ? 'Memproses...' : 'Nonaktifkan'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
