'use client'

import { useParams, useRouter, useSearchParams } from 'next/navigation'
import { Suspense, useEffect, useState } from 'react'
import { ArrowLeft, CheckCircle2, Loader2, ShieldCheck } from 'lucide-react'

/**
 * Local demo checkout page.
 *
 * When no real gateway credentials are configured (demo mode), providers
 * point their `paymentLink` at this page so the full flow can be exercised
 * in the preview: pick a payment method → "pay" → the simulate endpoint
 * completes the transaction → the invoice becomes Paid.
 */
function DemoCheckout() {
  const params = useParams<{ orderId: string }>()
  const searchParams = useSearchParams()
  const router = useRouter()

  const orderId = params?.orderId ?? ''
  const provider = searchParams.get('provider') === 'midtrans' ? 'midtrans' : 'pakasir'
  const amount = Number(searchParams.get('amount') ?? 0)
  const method = searchParams.get('method') === 'va' ? 'Virtual Account' : 'QRIS'

  const [paying, setPaying] = useState(false)
  const [done, setDone] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    // Never render the raw order id without a value.
    if (!orderId) {
      router.replace('/sales-invoices')
    }
  }, [orderId, router])

  async function handlePay() {
    setPaying(true)
    setError(null)
    try {
      const res = await fetch('/api/payments/simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ orderId, paymentMethod: method === 'QRIS' ? 'qris' : 'va' }),
      })
      const data = await res.json()
      if (!res.ok) {
        setError(data.error ?? 'Simulasi pembayaran gagal.')
        return
      }
      setDone(true)
    } catch {
      setError('Simulasi pembayaran gagal — coba lagi.')
    } finally {
      setPaying(false)
    }
  }

  const gatewayName = provider === 'midtrans' ? 'Midtrans' : 'Pakasir'

  return (
    <div className="min-h-screen bg-zinc-50 flex items-center justify-center px-4 py-10">
      <div className="w-full max-w-md">
        <a
          href="/sales-invoices"
          className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium mb-4"
        >
          <ArrowLeft className="h-3.5 w-3.5" /> Kembali ke Sales Invoices
        </a>

        <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
          {/* Brand header */}
          <div className="px-6 py-5 bg-zinc-900 text-white flex items-center justify-between">
            <div className="flex items-center gap-2.5">
              <div className="h-8 w-8 rounded-lg bg-indigo-500 flex items-center justify-center text-xs font-black">
                {provider === 'midtrans' ? 'MT' : 'PAK'}
              </div>
              <div>
                <p className="text-sm font-bold">{gatewayName}</p>
                <p className="text-[10px] text-zinc-400 font-mono uppercase tracking-wider">
                  Demo Checkout — Sandbox
                </p>
              </div>
            </div>
            <ShieldCheck className="h-5 w-5 text-emerald-400" />
          </div>

          {done ? (
            <div className="px-6 py-10 text-center">
              <CheckCircle2 className="h-12 w-12 text-emerald-500 mx-auto mb-3" />
              <h2 className="text-lg font-bold text-zinc-900">Pembayaran Berhasil</h2>
              <p className="text-sm text-zinc-500 mt-1">
                Transaksi <span className="font-mono text-zinc-700">{orderId}</span>{' '}
                telah ditandai lunas.
              </p>
              <button
                onClick={() => router.push('/sales-invoices')}
                className="mt-6 px-6 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors"
              >
                Lihat Invoice
              </button>
            </div>
          ) : (
            <div className="px-6 py-6 space-y-5">
              <div className="flex items-center justify-between text-sm">
                <span className="text-zinc-500">Total Pembayaran</span>
                <span className="font-bold text-zinc-900 text-lg">
                  {new Intl.NumberFormat('id-ID', {
                    style: 'currency',
                    currency: 'IDR',
                    maximumFractionDigits: 0,
                  }).format(amount)}
                </span>
              </div>

              <div className="rounded-xl border border-zinc-200 bg-zinc-50/60 px-4 py-3 text-xs text-zinc-600 space-y-1">
                <p className="flex justify-between">
                  <span>Metode</span>
                  <span className="font-semibold text-zinc-800">{method}</span>
                </p>
                <p className="flex justify-between">
                  <span>Order ID</span>
                  <span className="font-mono text-zinc-700">{orderId}</span>
                </p>
              </div>

              {error && (
                <div className="px-4 py-3 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700 font-medium">
                  {error}
                </div>
              )}

              <button
                onClick={handlePay}
                disabled={paying}
                className="w-full py-3 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-60 text-white text-sm font-bold rounded-xl transition-colors flex items-center justify-center gap-2"
              >
                {paying ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" /> Memproses…
                  </>
                ) : (
                  <>Bayar Sekarang ({method})</>
                )}
              </button>

              <p className="text-center text-[11px] text-zinc-400">
                Mode demo — menekan tombol akan menyelesaikan pembayaran secara
                simulasi.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default function DemoCheckoutPage() {
  return (
    <Suspense fallback={<div className="min-h-screen bg-zinc-50" />}>
      <DemoCheckout />
    </Suspense>
  )
}
