'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ArrowLeft } from 'lucide-react'
import { CreateGRInputSchema, type CreateGRInput } from '@/lib/schemas/gr'
import { useCreateGR } from '@/lib/queries/gr'
import { getClientErrorMessage } from '@/lib/api/errors'

interface FieldErrors {
  po_id?: string
  vendor_id?: string
  received_qty?: string
  received_amount?: string
  currency?: string
}

interface FormState {
  po_id: string
  vendor_id: string
  received_qty: string
  received_amount: string
  currency: string
}

export default function NewGoodsReceiptPage() {
  const router = useRouter()
  const { mutate, isPending } = useCreateGR()

  const [form, setForm] = useState<FormState>({
    po_id: '',
    vendor_id: '',
    received_qty: '',
    received_amount: '',
    currency: 'IDR',
  })

  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [submitError, setSubmitError] = useState<string | null>(null)

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const { name, value } = e.target
    setForm((prev) => ({ ...prev, [name]: value }))
    if (fieldErrors[name as keyof FieldErrors]) {
      setFieldErrors((prev) => ({ ...prev, [name]: undefined }))
    }
  }

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSubmitError(null)
    setFieldErrors({})

    const raw: Partial<CreateGRInput> = {
      po_id: form.po_id,
      vendor_id: form.vendor_id,
      received_qty: Number(form.received_qty),
      received_amount: form.received_amount,
      currency: form.currency || 'IDR',
    }

    const result = CreateGRInputSchema.safeParse(raw)
    if (!result.success) {
      const errors: FieldErrors = {}
      for (const issue of result.error.issues) {
        const field = issue.path[0] as keyof FieldErrors
        if (field && !errors[field]) {
          errors[field] = issue.message
        }
      }
      setFieldErrors(errors)
      return
    }

    mutate(result.data, {
      onSuccess: () => {
        router.push('/dashboard/goods-receipts')
      },
      onError: (err) => {
        setSubmitError(getClientErrorMessage(err))
      },
    })
  }

  return (
    <div className="px-8 py-8 space-y-6 max-w-xl">
      {/* Back nav */}
      <Link
        href="/dashboard/goods-receipts"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Goods Receipt
      </Link>

      {/* Page title */}
      <div>
        <h1 className="text-xl font-bold text-zinc-900">Buat Goods Receipt Baru</h1>
        <p className="text-sm text-zinc-500 mt-1">Isi detail penerimaan barang untuk dikirim ke sistem.</p>
      </div>

      {/* Top-level error banner */}
      {submitError !== null && (
        <div className="px-4 py-3 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700 font-medium">
          {submitError}
        </div>
      )}

      {/* Form card */}
      <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm">
        <form onSubmit={handleSubmit} className="px-6 py-6 space-y-5" noValidate>

          {/* po_id */}
          <div>
            <label htmlFor="po_id" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              PO ID
            </label>
            <input
              id="po_id"
              name="po_id"
              type="text"
              autoComplete="off"
              value={form.po_id}
              onChange={handleChange}
              placeholder="UUID Purchase Order yang terkait"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.po_id && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.po_id}</p>
            )}
          </div>

          {/* vendor_id */}
          <div>
            <label htmlFor="vendor_id" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Vendor ID
            </label>
            <input
              id="vendor_id"
              name="vendor_id"
              type="text"
              autoComplete="off"
              value={form.vendor_id}
              onChange={handleChange}
              placeholder="Masukkan Vendor ID"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.vendor_id && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.vendor_id}</p>
            )}
          </div>

          {/* received_qty */}
          <div>
            <label htmlFor="received_qty" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Qty Diterima
            </label>
            <input
              id="received_qty"
              name="received_qty"
              type="number"
              inputMode="numeric"
              min={1}
              autoComplete="off"
              value={form.received_qty}
              onChange={handleChange}
              placeholder="Contoh: 10"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.received_qty && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.received_qty}</p>
            )}
          </div>

          {/* received_amount */}
          <div>
            <label htmlFor="received_amount" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Jumlah Diterima
            </label>
            <input
              id="received_amount"
              name="received_amount"
              type="text"
              inputMode="decimal"
              autoComplete="off"
              value={form.received_amount}
              onChange={handleChange}
              placeholder="Contoh: 12345000.00"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.received_amount && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.received_amount}</p>
            )}
          </div>

          {/* currency */}
          <div>
            <label htmlFor="currency" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Mata Uang
            </label>
            <input
              id="currency"
              name="currency"
              type="text"
              autoComplete="off"
              value={form.currency}
              onChange={handleChange}
              placeholder="IDR"
              maxLength={3}
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.currency && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.currency}</p>
            )}
          </div>

          {/* Submit */}
          <div className="flex items-center justify-end pt-2">
            <button
              type="submit"
              disabled={isPending}
              className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isPending ? 'Memproses…' : 'Buat Goods Receipt'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
