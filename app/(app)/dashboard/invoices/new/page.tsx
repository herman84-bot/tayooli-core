'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ArrowLeft } from 'lucide-react'
import { CreateInvoiceInputSchema, type CreateInvoiceInput } from '@/lib/schemas/invoice'
import { useCreateInvoice } from '@/lib/queries/invoices'
import { getClientErrorMessage } from '@/lib/api/errors'

interface FieldErrors {
  vendor_id?: string
  invoice_number?: string
  amount?: string
  currency?: string
  due_date?: string
}

interface FormState {
  vendor_id: string
  invoice_number: string
  amount: string
  currency: string
  due_date: string
}

export default function NewInvoicePage() {
  const router = useRouter()
  const { mutate, isPending } = useCreateInvoice()

  const [form, setForm] = useState<FormState>({
    vendor_id: '',
    invoice_number: '',
    amount: '',
    currency: 'IDR',
    due_date: '',
  })

  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [submitError, setSubmitError] = useState<string | null>(null)

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const { name, value } = e.target
    setForm((prev) => ({ ...prev, [name]: value }))
    // Clear the field-level error as the user corrects the input
    if (fieldErrors[name as keyof FieldErrors]) {
      setFieldErrors((prev) => ({ ...prev, [name]: undefined }))
    }
  }

  function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSubmitError(null)
    setFieldErrors({})

    // Build the payload — only include due_date when the user filled it in
    const raw: Partial<CreateInvoiceInput> = {
      vendor_id: form.vendor_id,
      invoice_number: form.invoice_number,
      amount: form.amount,
      currency: form.currency || 'IDR',
    }
    if (form.due_date) {
      raw.due_date = form.due_date
    }

    const result = CreateInvoiceInputSchema.safeParse(raw)
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
        router.push('/dashboard/invoices')
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
        href="/dashboard/invoices"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Purchase Invoice
      </Link>

      {/* Page title */}
      <div>
        <h1 className="text-xl font-bold text-zinc-900">Buat Purchase Invoice Baru</h1>
        <p className="text-sm text-zinc-500 mt-1">Catat tagihan vendor dan isi detail invoice pembelian untuk dikirim ke sistem.</p>
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
            <label
              htmlFor="vendor_id"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
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

          {/* invoice_number */}
          <div>
            <label
              htmlFor="invoice_number"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
              Nomor Purchase Invoice (Vendor Bill #)
            </label>
            <input
              id="invoice_number"
              name="invoice_number"
              type="text"
              autoComplete="off"
              value={form.invoice_number}
              onChange={handleChange}
              placeholder="Contoh: INV-2024-001"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.invoice_number && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.invoice_number}</p>
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
              placeholder="Contoh: 12345000.00"
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

          {/* due_date (optional) */}
          <div>
            <label
              htmlFor="due_date"
              className="block text-xs font-semibold text-zinc-600 mb-1.5"
            >
              Tanggal Jatuh Tempo{' '}
              <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <input
              id="due_date"
              name="due_date"
              type="date"
              value={form.due_date}
              onChange={handleChange}
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.due_date && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.due_date}</p>
            )}
          </div>

          {/* Submit */}
          <div className="flex items-center justify-end pt-2">
            <button
              type="submit"
              disabled={isPending}
              className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isPending ? 'Memproses…' : 'Simpan Purchase Invoice'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
