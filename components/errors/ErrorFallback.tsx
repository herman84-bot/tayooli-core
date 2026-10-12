"use client"

import { useEffect } from "react"
import { AlertTriangle, RefreshCw, Home } from "lucide-react"

export interface ErrorFallbackProps {
  error: Error & { digest?: string }
  reset: () => void
  /** Where "Kembali ke Dasbor" leads. */
  homeHref?: string
}

/**
 * Friendly Indonesian fallback for a crashed route segment.
 * Never renders error.message: it can carry internal detail. Only the
 * Next.js digest is shown, as a reference code to match server logs.
 */
export function ErrorFallback({ error, reset, homeHref = "/dashboard" }: ErrorFallbackProps) {
  useEffect(() => {
    console.error("Route error boundary:", error)
  }, [error])

  return (
    <div role="alert" className="min-h-[60vh] flex items-center justify-center px-4 py-12">
      <div className="max-w-md w-full bg-white rounded-xl border border-red-200 p-8 text-center shadow-xs">
        <AlertTriangle className="w-12 h-12 text-red-500 mx-auto mb-4" aria-hidden="true" />
        <h2 className="text-lg font-bold text-slate-900">Terjadi kendala pada halaman ini</h2>
        <p className="text-sm text-slate-500 mt-2">
          Halaman tidak dapat ditampilkan. Data Anda aman. Silakan coba lagi, atau kembali ke Dasbor.
        </p>
        {error.digest && (
          <p className="text-xs text-slate-400 mt-3 font-mono">Kode referensi: {error.digest}</p>
        )}
        <div className="mt-6 flex flex-col sm:flex-row gap-2 justify-center">
          <button
            type="button"
            onClick={() => reset()}
            className="inline-flex items-center justify-center gap-2 px-4 py-2.5 min-h-[44px] rounded-lg font-semibold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] transition"
          >
            <RefreshCw className="w-4 h-4" aria-hidden="true" />
            Coba Lagi
          </button>
          <a
            href={homeHref}
            className="inline-flex items-center justify-center gap-2 px-4 py-2.5 min-h-[44px] rounded-lg font-semibold text-sm border border-slate-200 text-slate-700 hover:bg-slate-50 transition"
          >
            <Home className="w-4 h-4" aria-hidden="true" />
            Kembali ke Dasbor
          </a>
        </div>
      </div>
    </div>
  )
}
