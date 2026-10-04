'use client'

import { useEffect, useState } from 'react'
import { CheckCircle2, CreditCard, KeyRound, ShieldCheck } from 'lucide-react'

type GatewayId = 'pakasir' | 'midtrans'

interface TenantConfig {
  tenantId: string
  provider: GatewayId
  slug?: string
  hasApiKey: boolean
  hasServerKey: boolean
  hasCredentials: boolean
  isProduction?: boolean
  baseUrl?: string
  settlementBankName?: string
  settlementBankAccount?: string
  settlementHolderName?: string
  gatewayFeePercent?: number
  isActive: boolean
}

const GATEWAYS: { id: GatewayId; name: string; desc: string }[] = [
  {
    id: 'pakasir',
    name: 'Pakasir',
    desc: 'Payment link — QRIS & Virtual Account. Per-tenant project slug + API key.',
  },
  {
    id: 'midtrans',
    name: 'Midtrans',
    desc: 'Snap — QRIS, Virtual Account & e-wallet. Per-tenant merchant Server Key.',
  },
]

const EMPTY_FORM = {
  tenantId: '',
  provider: 'pakasir' as GatewayId,
  slug: '',
  apiKey: '',
  serverKey: '',
  clientKey: '',
  isProduction: false,
  settlementBankName: '',
  settlementBankAccount: '',
  settlementHolderName: '',
}

export default function PaymentGatewaysPage() {
  const [configs, setConfigs] = useState<TenantConfig[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)
  const [form, setForm] = useState(EMPTY_FORM)

  async function loadConfigs() {
    try {
      const res = await fetch('/api/payments/configs')
      const data = await res.json()
      setConfigs(Array.isArray(data) ? data : [])
    } catch {
      setError('Gagal memuat konfigurasi tenant.')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void loadConfigs()
  }, [])

  function handleChange(e: React.ChangeEvent<HTMLInputElement>) {
    const { name, value, type, checked } = e.target
    setForm((prev) => ({
      ...prev,
      [name]: type === 'checkbox' ? checked : value,
    }))
  }

  function handleSelectGateway(id: GatewayId) {
    setForm((prev) => ({ ...prev, provider: id }))
  }

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    setSaving(true)
    setSaved(false)
    setError(null)
    try {
      const body: Record<string, unknown> = {
        tenantId: form.tenantId,
        provider: form.provider,
        settlementBankName: form.settlementBankName || undefined,
        settlementBankAccount: form.settlementBankAccount || undefined,
        settlementHolderName: form.settlementHolderName || undefined,
      }
      if (form.provider === 'pakasir') {
        body.slug = form.slug
        if (form.apiKey) body.apiKey = form.apiKey
      } else {
        if (form.serverKey) body.serverKey = form.serverKey
        if (form.clientKey) body.clientKey = form.clientKey
        body.isProduction = form.isProduction
      }

      const res = await fetch('/api/payments/configs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const data = await res.json()
      if (!res.ok) {
        setError(data.error ?? 'Gagal menyimpan konfigurasi.')
        return
      }
      setForm(EMPTY_FORM)
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
      await loadConfigs()
    } catch {
      setError('Gagal menyimpan konfigurasi.')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="px-8 py-8 space-y-6 max-w-4xl">
      <div>
        <h1 className="text-xl font-bold text-zinc-900">Payment Gateway per Tenant</h1>
        <p className="text-sm text-zinc-500 mt-1">
          Pilih gateway untuk tiap tenant — Pakasir atau Midtrans. Model B:
          setiap tenant punya akun gateway sendiri, dana masuk langsung ke
          rekening bank tenant.
        </p>
      </div>

      {/* Info banner */}
      <div className="flex items-start gap-3 rounded-2xl border border-indigo-100 bg-indigo-50/60 px-5 py-4">
        <ShieldCheck className="h-5 w-5 text-indigo-600 mt-0.5 shrink-0" />
        <div className="text-sm text-indigo-900/80 space-y-1">
          <p className="font-semibold text-indigo-900">Alur uang</p>
          <p>
            Customer bayar → gateway (akun milik tenant) → settlement ke{' '}
            <strong>rekening bank tenant</strong>. Platform hanya mengarahkan dan
            mencatat status pembayaran via webhook.
          </p>
        </div>
      </div>

      {/* Form */}
      <form
        onSubmit={handleSubmit}
        className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm p-6 space-y-4"
      >
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-bold text-zinc-900">Tambah / Perbarui Tenant</h2>
          {saved && (
            <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-700">
              <CheckCircle2 className="h-3.5 w-3.5" /> Tersimpan
            </span>
          )}
        </div>

        {error && (
          <div className="px-4 py-3 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700 font-medium">
            {error}
          </div>
        )}

        {/* Gateway selector */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          {GATEWAYS.map((g) => (
            <button
              key={g.id}
              type="button"
              onClick={() => handleSelectGateway(g.id)}
              className={`text-left rounded-xl border p-4 transition-all ${
                form.provider === g.id
                  ? 'border-indigo-400 bg-indigo-50/70 ring-2 ring-indigo-500/20'
                  : 'border-zinc-200 hover:border-zinc-300'
              }`}
            >
              <div className="flex items-center gap-2">
                <span className="h-7 w-7 rounded-lg bg-zinc-900 text-white flex items-center justify-center text-[10px] font-black">
                  {g.id === 'midtrans' ? 'MT' : 'PAK'}
                </span>
                <span className="text-sm font-bold text-zinc-900">{g.name}</span>
              </div>
              <p className="text-[11px] text-zinc-500 mt-1.5 leading-relaxed">{g.desc}</p>
            </button>
          ))}
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div>
            <label htmlFor="tenantId" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Tenant ID *
            </label>
            <input
              id="tenantId"
              name="tenantId"
              value={form.tenantId}
              onChange={handleChange}
              placeholder="cth: tenant-acme"
              required
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
          </div>

          {form.provider === 'pakasir' ? (
            <div>
              <label htmlFor="slug" className="block text-xs font-semibold text-zinc-600 mb-1.5">
                Pakasir Project Slug *
              </label>
              <input
                id="slug"
                name="slug"
                value={form.slug}
                onChange={handleChange}
                placeholder="cth: acme-project"
                className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
              />
            </div>
          ) : (
            <div>
              <label htmlFor="isProduction" className="flex items-center gap-2 pt-6 text-xs font-semibold text-zinc-600">
                <input
                  id="isProduction"
                  name="isProduction"
                  type="checkbox"
                  checked={form.isProduction}
                  onChange={handleChange}
                  className="h-4 w-4 rounded border-zinc-300 text-indigo-600 focus:ring-indigo-500"
                />
                Mode Produksi (sandbox jika nonaktif)
              </label>
            </div>
          )}
        </div>

        {form.provider === 'pakasir' ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="apiKey" className="block text-xs font-semibold text-zinc-600 mb-1.5">
                API Key <span className="text-zinc-400">(opsional — mode live)</span>
              </label>
              <input
                id="apiKey"
                name="apiKey"
                type="password"
                value={form.apiKey}
                onChange={handleChange}
                placeholder="Kosongkan untuk mode demo"
                className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
              />
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label htmlFor="serverKey" className="block text-xs font-semibold text-zinc-600 mb-1.5">
                Midtrans Server Key <span className="text-zinc-400">(opsional — mode live)</span>
              </label>
              <input
                id="serverKey"
                name="serverKey"
                type="password"
                value={form.serverKey}
                onChange={handleChange}
                placeholder="Kosongkan untuk mode demo"
                className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
              />
            </div>
            <div>
              <label htmlFor="clientKey" className="block text-xs font-semibold text-zinc-600 mb-1.5">
                Client Key <span className="text-zinc-400">(untuk snap.js)</span>
              </label>
              <input
                id="clientKey"
                name="clientKey"
                value={form.clientKey}
                onChange={handleChange}
                placeholder="Opsional"
                className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
              />
            </div>
          </div>
        )}

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label htmlFor="settlementBankName" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Bank Settlement
            </label>
            <input
              id="settlementBankName"
              name="settlementBankName"
              value={form.settlementBankName}
              onChange={handleChange}
              placeholder="cth: BCA"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
          </div>
          <div>
            <label htmlFor="settlementBankAccount" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              No. Rekening
            </label>
            <input
              id="settlementBankAccount"
              name="settlementBankAccount"
              value={form.settlementBankAccount}
              onChange={handleChange}
              placeholder="1234567890"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
          </div>
          <div>
            <label htmlFor="settlementHolderName" className="block text-xs font-semibold text-zinc-600 mb-1.5">
              Atas Nama
            </label>
            <input
              id="settlementHolderName"
              name="settlementHolderName"
              value={form.settlementHolderName}
              onChange={handleChange}
              placeholder="PT Acme Indonesia"
              className="w-full rounded-xl border border-zinc-200 px-3.5 py-2.5 text-sm text-zinc-900 placeholder:text-zinc-400 focus:outline-none focus:ring-2 focus:ring-indigo-500/30 focus:border-indigo-400 transition-colors"
            />
          </div>
        </div>

        <div className="flex justify-end">
          <button
            type="submit"
            disabled={saving}
            className="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? 'Menyimpan…' : 'Simpan Konfigurasi'}
          </button>
        </div>
      </form>

      {/* Config list */}
      <div className="bg-white rounded-2xl border border-zinc-200/80 shadow-sm overflow-hidden">
        <div className="px-6 py-4 border-b border-zinc-100">
          <h2 className="text-sm font-bold text-zinc-900">Daftar Tenant Terdaftar</h2>
        </div>
        {loading ? (
          <div className="p-8 space-y-3 animate-pulse">
            {Array.from({ length: 2 }).map((_, i) => (
              <div key={i} className="h-12 bg-zinc-100 rounded-xl" />
            ))}
          </div>
        ) : configs.length === 0 ? (
          <div className="px-6 py-10 text-center">
            <KeyRound className="h-8 w-8 text-zinc-300 mx-auto mb-2" />
            <p className="text-sm text-zinc-500">
              Belum ada tenant terkonfigurasi. Tambahkan lewat form di atas.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-zinc-100">
            {configs.map((cfg) => (
              <div key={cfg.tenantId} className="px-6 py-4 flex items-center justify-between gap-4">
                <div className="min-w-0">
                  <p className="text-sm font-semibold text-zinc-900">{cfg.tenantId}</p>
                  <p className="text-xs text-zinc-500 mt-0.5 font-mono">
                    {cfg.provider === 'midtrans' ? 'midtrans' : 'pakasir'}
                    {cfg.provider === 'pakasir' && cfg.slug
                      ? ` · project: ${cfg.slug}`
                      : cfg.provider === 'midtrans'
                        ? ` · ${cfg.isProduction ? 'production' : 'sandbox'}`
                        : ''}
                    {cfg.settlementBankName &&
                      ` · settlement: ${cfg.settlementBankName} ${cfg.settlementBankAccount ?? ''}`}
                  </p>
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  <span
                    className={`inline-flex items-center gap-1 px-3 py-1.5 rounded-full text-xs font-semibold border ${
                      cfg.hasCredentials
                        ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                        : 'bg-amber-50 text-amber-700 border-amber-200'
                    }`}
                  >
                    {cfg.hasCredentials ? 'LIVE' : 'DEMO'}
                  </span>
                  <span className="inline-flex items-center gap-1 px-3 py-1.5 rounded-full text-xs font-semibold bg-zinc-100 text-zinc-700 border border-zinc-200">
                    <CreditCard className="h-3 w-3" />
                    {cfg.provider === 'midtrans' ? 'Midtrans' : 'Pakasir'}
                  </span>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
