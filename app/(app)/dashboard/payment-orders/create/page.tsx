'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ArrowLeft } from 'lucide-react'
import { CreatePaymentOrderInputSchema, type CreatePaymentOrderInput, paymentMethodLabels } from '@/lib/schemas/payment-order'
import { useCreatePaymentOrder } from '@/lib/queries/payment-orders'
import { getClientErrorMessage } from '@/lib/api/errors'

interface FieldErrors {
  invoice_id?: string
  amount?: string
  currency?: string
  payment_method?: string
  reference_number?: string
  notes?: string
}

interface FormState {
  invoice_id: string
  amount: string
  currency: string
  payment_method: string
  reference_number: string
  notes: string
}

export default function NewPaymentOrderPage() {
  const router = useRouter()
  const { mutate, isPending } = useCreatePaymentOrder()

  const [form, setForm] = useState<FormState>({
    invoice_id: '',
    amount: '',
    currency: 'IDR',
    payment_method: 'bank_transfer',
    reference_number: '',
    notes: '',
  })

  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [submitError, setSubmitError] = useState<string | null>(null)

  function handleChange(e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) {
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

    const raw: Partial<CreatePaymentOrderInput> = {
      invoice_id: form.invoice_id,
      amount: form.amount,
      currency: form.currency || 'IDR',
      payment_method: form.payment_method as CreatePaymentOrderInput['payment_method'],
    }
    if (form.reference_number) {
      raw.reference_number = form.reference_number
    }
    if (form.notes) {
      raw.notes = form.notes
    }

    const result = CreatePaymentOrderInputSchema.safeParse(raw)
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
        router.push('/dashboard/payment-orders')
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
        href="/dashboard/payment-orders"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Payment Orders
      </Link>

      {/* Page title */}
      <div>
        <h1 className="text-xl font-bold text-zinc-900">Buat Payment Order</h1>
        <p className="text-sm text-zinc-500 mt-1">Isi detail payment order untuk mengajukan pembayaran.</p>
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

          {/* invoice_id */}
          <div>
            <label
              htmlFor="invoice_id"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
              Invoice ID
            </label>
            <input
              id="invoice_id"
              name="invoice_id"
              type="text"
              autoComplete="off"
              value={form.invoice_id}
              onChange={handleChange}
              placeholder="Masukkan Invoice ID (UUID)"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors font-mono"
            />
            {fieldErrors.invoice_id && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.invoice_id}</p>
            )}
          </div>

          {/* amount */}
          <div>
            <label
              htmlFor="amount"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
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
              placeholder="Contoh: 5000000.00"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.amount && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.amount}</p>
            )}
          </div>

          {/* currency */}
          <div>
            <label
              htmlFor="currency"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
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

          {/* payment_method */}
          <div>
            <label
              htmlFor="payment_method"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
              Metode Pembayaran
            </label>
            <select
              id="payment_method"
              name="payment_method"
              value={form.payment_method}
              onChange={handleChange}
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            >
              {Object.entries(paymentMethodLabels).map(([value, label]) => (
                <option key={value} value={value}>{label}</option>
              ))}
            </select>
            {fieldErrors.payment_method && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.payment_method}</p>
            )}
          </div>

          {/* reference_number */}
          <div>
            <label
              htmlFor="reference_number"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
              Nomor Referensi{' '}
              <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <input
              id="reference_number"
              name="reference_number"
              type="text"
              autoComplete="off"
              value={form.reference_number}
              onChange={handleChange}
              placeholder="Contoh: REF-2024-001"
              maxLength={100}
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.reference_number && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.reference_number}</p>
            )}
          </div>

          {/* notes */}
          <div>
            <label
              htmlFor="notes"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
              Catatan{' '}
              <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <textarea
              id="notes"
              name="notes"
              rows={3}
              value={form.notes}
              onChange={handleChange}
              placeholder="Catatan tambahan mengenai payment order ini"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors resize-none"
            />
            {fieldErrors.notes && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.notes}</p>
            )}
          </div>

          {/* Submit */}
          <div className="flex items-center justify-end pt-2">
            <button
              type="submit"
              disabled={isPending}
              className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isPending ? 'Memproses...' : 'Buat Payment Order'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
