'use client'

import { useState } from 'react'
import { cn } from '@/lib/utils'
import { X, Send, CheckCircle } from 'lucide-react'

interface ReportFormProps {
  onClose: () => void
}

const CATEGORIES = [
  'Bug',
  'Akun',
  'Pembayaran',
  'Fitur',
  'Performa',
  'Lainnya',
]

export default function ReportForm({ onClose }: ReportFormProps) {
  const [category, setCategory] = useState('')
  const [description, setDescription] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isSent, setIsSent] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async () => {
    if (!category || !description.trim()) {
      setError('Pilih kategori dan isi deskripsi')
      return
    }

    setIsSubmitting(true)
    setError('')

    try {
      const res = await fetch('/api/v1/support/report', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({
          category,
          description: description.trim(),
        }),
      })

      if (!res.ok) {
        throw new Error('Failed to send')
      }

      setIsSent(true)
    } catch {
      setError('Gagal mengirim laporan. Silakan coba lagi.')
    } finally {
      setIsSubmitting(false)
    }
  }

  if (isSent) {
    return (
      <div className="flex flex-col items-center justify-center py-8 px-4">
        <CheckCircle className="h-10 w-10 text-primary mb-3" />
        <p className="text-sm font-medium text-foreground text-center">
          Laporan sudah diteruskan!
        </p>
        <p className="text-xs text-muted-foreground text-center mt-1">
          Tim kami akan segera merespon.
        </p>
        <button
          onClick={onClose}
          className="mt-4 px-4 py-1.5 text-xs font-medium text-primary hover:bg-primary/10 rounded-md transition-colors"
        >
          Tutup
        </button>
      </div>
    )
  }

  return (
    <div className="p-4 space-y-3">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">
          Laporkan Masalah
        </h3>
        <button
          onClick={onClose}
          className="p-1 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
          aria-label="Close"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Category */}
      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">
          Kategori
        </label>
        <div className="flex flex-wrap gap-1.5">
          {CATEGORIES.map((cat) => (
            <button
              key={cat}
              onClick={() => setCategory(cat)}
              className={cn(
                'px-2.5 py-1 text-xs font-medium rounded-md transition-colors',
                category === cat
                  ? 'bg-primary text-primary-foreground'
                  : 'bg-zinc-100 text-muted-foreground hover:bg-zinc-200'
              )}
            >
              {cat}
            </button>
          ))}
        </div>
      </div>

      {/* Description */}
      <div>
        <label className="block text-xs font-medium text-muted-foreground mb-1.5">
          Deskripsi
        </label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Jelaskan masalah yang kamu alami..."
          rows={3}
          className="w-full px-3 py-2 text-sm bg-zinc-100 rounded-lg border-0 outline-none placeholder:text-muted-foreground resize-none"
        />
      </div>

      {/* Error */}
      {error && (
        <p className="text-xs text-red-500">{error}</p>
      )}

      {/* Submit */}
      <button
        onClick={handleSubmit}
        disabled={isSubmitting || !category || !description.trim()}
        className={cn(
          'w-full flex items-center justify-center gap-2 py-2 rounded-lg text-sm font-medium transition-colors',
          'bg-primary text-primary-foreground hover:bg-primary/90',
          'disabled:opacity-50 disabled:cursor-not-allowed'
        )}
      >
        {isSubmitting ? (
          <span className="h-4 w-4 border-2 border-primary-foreground border-t-transparent rounded-full animate-spin" />
        ) : (
          <Send className="h-4 w-4" />
        )}
        {isSubmitting ? 'Mengirim...' : 'Kirim Laporan'}
      </button>
    </div>
  )
}
