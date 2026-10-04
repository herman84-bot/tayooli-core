'use client'

import { useState } from 'react'
import Link from 'next/link'
import { useRouter } from 'next/navigation'
import { ArrowLeft } from 'lucide-react'
import { CreateVendorInputSchema, type CreateVendorInput } from '@/lib/schemas/vendor'
import { useCreateVendor } from '@/lib/queries/vendors'
import { getClientErrorMessage } from '@/lib/api/errors'

interface FieldErrors {
  name?: string
  email?: string
  phone?: string
  address?: string
  bank_account?: string
  bank_name?: string
  tax_id?: string
}

interface FormState {
  name: string
  email: string
  phone: string
  address: string
  bank_account: string
  bank_name: string
  tax_id: string
}

export default function CreateVendorPage() {
  const router = useRouter()
  const { mutate, isPending } = useCreateVendor()

  const [form, setForm] = useState<FormState>({
    name: '',
    email: '',
    phone: '',
    address: '',
    bank_account: '',
    bank_name: '',
    tax_id: '',
  })

  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [submitError, setSubmitError] = useState<string | null>(null)

  function handleChange(e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) {
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

    const raw: Partial<CreateVendorInput> = {
      name: form.name,
    }
    if (form.email) {
      raw.email = form.email
    }
    if (form.phone) {
      raw.phone = form.phone
    }
    if (form.address) {
      raw.address = form.address
    }
    if (form.bank_account) {
      raw.bank_account = form.bank_account
    }
    if (form.bank_name) {
      raw.bank_name = form.bank_name
    }
    if (form.tax_id) {
      raw.tax_id = form.tax_id
    }

    const result = CreateVendorInputSchema.safeParse(raw)
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
        router.push('/dashboard/vendors')
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
        href="/dashboard/vendors"
        className="inline-flex items-center gap-1.5 text-xs text-zinc-500 hover:text-indigo-600 transition-colors font-medium"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Kembali ke Daftar Vendor
      </Link>

      {/* Page title */}
      <div>
        <h1 className="text-xl font-bold text-zinc-900">Tambah Vendor</h1>
        <p className="text-sm text-zinc-500 mt-1">Isi detail vendor baru untuk menambahkannya ke sistem.</p>
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

          {/* name */}
          <div>
            <label htmlFor="name" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Nama Vendor <span className="text-rose-500">*</span>
            </label>
            <input
              id="name"
              name="name"
              type="text"
              autoComplete="off"
              value={form.name}
              onChange={handleChange}
              placeholder="PT Vendor Sejahtera"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.name && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.name}</p>
            )}
          </div>

          {/* email */}
          <div>
            <label htmlFor="email" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Email <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <input
              id="email"
              name="email"
              type="email"
              autoComplete="off"
              value={form.email}
              onChange={handleChange}
              placeholder="vendor@example.com"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.email && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.email}</p>
            )}
          </div>

          {/* phone */}
          <div>
            <label htmlFor="phone" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Telepon <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <input
              id="phone"
              name="phone"
              type="text"
              autoComplete="off"
              value={form.phone}
              onChange={handleChange}
              placeholder="+62812345678"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.phone && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.phone}</p>
            )}
          </div>

          {/* address */}
          <div>
            <label htmlFor="address" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Alamat <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <textarea
              id="address"
              name="address"
              rows={2}
              value={form.address}
              onChange={handleChange}
              placeholder="Alamat lengkap vendor"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors resize-none"
            />
            {fieldErrors.address && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.address}</p>
            )}
          </div>

          {/* bank_account */}
          <div>
            <label htmlFor="bank_account" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              No. Rekening Bank <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <input
              id="bank_account"
              name="bank_account"
              type="text"
              autoComplete="off"
              value={form.bank_account}
              onChange={handleChange}
              placeholder="1234567890"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.bank_account && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.bank_account}</p>
            )}
          </div>

          {/* bank_name */}
          <div>
            <label htmlFor="bank_name" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Nama Bank <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <input
              id="bank_name"
              name="bank_name"
              type="text"
              autoComplete="off"
              value={form.bank_name}
              onChange={handleChange}
              placeholder="Bank Mandiri"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.bank_name && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.bank_name}</p>
            )}
          </div>

          {/* tax_id */}
          <div>
            <label htmlFor="tax_id" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              NPWP <span className="text-zinc-400 font-normal">(opsional)</span>
            </label>
            <input
              id="tax_id"
              name="tax_id"
              type="text"
              autoComplete="off"
              value={form.tax_id}
              onChange={handleChange}
              placeholder="00.000.000.0-000.000"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
            {fieldErrors.tax_id && (
              <p className="text-xs text-rose-600 mt-1">{fieldErrors.tax_id}</p>
            )}
          </div>

          {/* Submit */}
          <div className="flex items-center justify-end pt-2">
            <button
              type="submit"
              disabled={isPending}
              className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isPending ? 'Memproses...' : 'Tambah Vendor'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
