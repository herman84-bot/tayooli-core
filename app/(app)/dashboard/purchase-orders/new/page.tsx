'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ArrowLeft } from 'lucide-react'
import { CreatePOInputSchema, type CreatePOInput } from '@/lib/schemas/po'
import { useCreatePO } from '@/lib/queries/po'
import { getClientErrorMessage } from '@/lib/api/errors'

interface FieldErrors {
  vendor_id?: string
  po_number?: string
  amount?: string
  qty?: string
  currency?: string
}

interface FormState {
  vendor_id: string
  po_number: string
  amount: string
  qty: string
  currency: string
}

export default function NewPurchaseOrderPage() {
  const router = useRouter()
  const { mutate, isPending } = useCreatePO()

  const [form, setForm] = useState<FormState>({
    vendor_id: '',
    po_number: '',
    amount: '',
    qty: '',
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

    const raw: Partial<CreatePOInput> = {
      vendor_id: form.vendor_id,
      po_number: form.po_number,
      amount: form.amount,
      qty: Number(form.qty),
      currency: form.currency || 'IDR',
    }

    const result = CreatePOInputSchema.safeParse(raw)
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
        router.push('/dashboard/purchase-orders')
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
        href="/dashboard/purchase-orders"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Purchase Order
      </Link>

      {/* Page title */}
      <div>
        <h1 className="text-xl font-bold text-zinc-900">Buat Purchase Order Baru</h1>
        <p className="text-sm text-zinc-500 mt-1">Isi detail PO untuk dikirim ke sistem.</p>
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

          {/* po_number */}
          <div>
            <label htmlFor="po_number" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Nomor PO
            </label>
            <input
              id="po_number"
              name="po_number"
              type="text"
              autoComplete="off"
              value={form.po_number}
              onChange={handleChange}
              placeholder="Contoh: PO-2024-001"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.po_number && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.po_number}</p>
            )}
          </div>

          {/* amount */}
          <div>
            <label htmlFor="amount" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Jumlah
            </label>
            <input
              id="amount"
              name="amount"
              type="text"
              inputMode="decimal"
              autoComplete="off"
              value={form.amount}
              onChange={handleChange}
              placeholder="Contoh: 12345000.00"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.amount && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.amount}</p>
            )}
          </div>

          {/* qty */}
          <div>
            <label htmlFor="qty" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Qty
            </label>
            <input
              id="qty"
              name="qty"
              type="number"
              inputMode="numeric"
              min={1}
              autoComplete="off"
              value={form.qty}
              onChange={handleChange}
              placeholder="Contoh: 10"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.qty && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.qty}</p>
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
              {isPending ? 'Memproses…' : 'Buat Purchase Order'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
