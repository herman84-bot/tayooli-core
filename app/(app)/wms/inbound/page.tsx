"use client"

import React, { useMemo, useState } from "react"
import {
  PackagePlus,
  Plus,
  ScanLine,
  Trash2,
  X,
  CheckCircle2,
  AlertCircle,
  ArrowLeft,
  Ban,
} from "lucide-react"
import {
  useWarehouses,
  useWarehouseLocations,
  useStockReceipts,
  useStockReceipt,
  useCreateStockReceipt,
  useUpdateStockReceipt,
  usePostStockReceipt,
  useCancelStockReceipt,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import { useBarcodeScanner } from "@/hooks/useBarcodeScanner"
import { api, Product, StockReceipt, StockReceiptInput, StockReceiptStatus } from "@/lib/api"

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const STATUS_LABEL: Record<StockReceiptStatus, { text: string; cls: string }> = {
  DRAFT: { text: "Draf", cls: "bg-amber-50 text-amber-700 border-amber-200" },
  POSTED: { text: "Sudah Masuk Stok", cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  CANCELLED: { text: "Dibatalkan", cls: "bg-slate-100 text-slate-600 border-slate-200" },
}

const REJECT_REASONS = ["Rusak / pecah", "Kemasan sobek", "Kedaluwarsa", "Salah kirim", "Kurang lengkap"]

function num(v: string | number | undefined | null): number {
  if (v === undefined || v === null || v === "") return 0
  const n = typeof v === "number" ? v : parseFloat(v)
  return Number.isFinite(n) ? n : 0
}

function fmtQty(v: string | number | undefined | null): string {
  return num(v).toLocaleString("id-ID", { maximumFractionDigits: 4 })
}

function fmtDate(iso?: string): string {
  if (!iso) return "-"
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return "-"
  return d.toLocaleString("id-ID", { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" })
}

function errMsg(e: unknown): string {
  return e instanceof Error && e.message ? e.message : "Terjadi kesalahan. Coba lagi."
}

function StatusBadge({ status }: { status: StockReceiptStatus }) {
  const s = STATUS_LABEL[status] ?? STATUS_LABEL.DRAFT
  return <span className={`inline-flex rounded-full border px-2.5 py-0.5 text-xs font-medium ${s.cls}`}>{s.text}</span>
}

// ---------------------------------------------------------------------------
// Form line state
// ---------------------------------------------------------------------------

interface Line {
  product_id: string
  product_name: string
  product_sku: string
  accepted: string
  rejected: string
  reject_reason: string
}

interface FormState {
  id: string | null
  warehouse_id: string
  dest_location_id: string
  supplier_name: string
  supplier_ref: string
  notes: string
  lines: Line[]
}

const emptyForm = (warehouseId = ""): FormState => ({
  id: null,
  warehouse_id: warehouseId,
  dest_location_id: "",
  supplier_name: "",
  supplier_ref: "",
  notes: "",
  lines: [],
})

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

type View = { mode: "list" } | { mode: "form" } | { mode: "detail"; id: string }

export default function InboundPage() {
  const [view, setView] = useState<View>({ mode: "list" })
  const [form, setForm] = useState<FormState>(emptyForm())
  const [notice, setNotice] = useState<{ type: "success" | "error"; text: string } | null>(null)

  const { data: warehouses = [] } = useWarehouses()

  const openNew = () => {
    setForm(emptyForm(warehouses[0]?.id ?? ""))
    setNotice(null)
    setView({ mode: "form" })
  }

  return (
    <div className="mx-auto max-w-5xl space-y-6 p-4 md:p-6">
      {notice && (
        <div
          role="status"
          className={`flex items-start gap-2 rounded-lg border px-4 py-3 text-sm ${
            notice.type === "success"
              ? "border-emerald-200 bg-emerald-50 text-emerald-800"
              : "border-rose-200 bg-rose-50 text-rose-800"
          }`}
        >
          {notice.type === "success" ? <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0" /> : <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />}
          <span className="flex-1">{notice.text}</span>
          <button onClick={() => setNotice(null)} aria-label="Tutup pesan"><X className="h-4 w-4" /></button>
        </div>
      )}

      {view.mode === "list" && (
        <ReceiptList onNew={openNew} onOpen={(id) => { setNotice(null); setView({ mode: "detail", id }) }} />
      )}

      {view.mode === "form" && (
        <ReceiptForm
          form={form}
          setForm={setForm}
          onCancel={() => setView(form.id ? { mode: "detail", id: form.id } : { mode: "list" })}
          onSaved={(id) => {
            setNotice({ type: "success", text: "Draf tersimpan. Stok belum berubah sampai Anda menekan Konfirmasi." })
            setView({ mode: "detail", id })
          }}
        />
      )}

      {view.mode === "detail" && (
        <ReceiptDetail
          id={view.id}
          onBack={() => setView({ mode: "list" })}
          onEdit={(f) => { setForm(f); setNotice(null); setView({ mode: "form" }) }}
          onNotice={setNotice}
        />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

function ReceiptList({ onNew, onOpen }: { onNew: () => void; onOpen: (id: string) => void }) {
  const { data: warehouses = [] } = useWarehouses()
  const [warehouseId, setWarehouseId] = useState("")
  const { data: receipts = [], isLoading, isError, error } = useStockReceipts(warehouseId || null)
  const whName = (id: string) => warehouses.find((w) => w.id === id)?.name ?? "-"

  return (
    <>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold text-foreground">
            <PackagePlus className="h-6 w-6 text-primary" /> Barang Masuk
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Catat barang yang datang dari pemasok. Stok baru bertambah setelah penerimaan dikonfirmasi.
          </p>
        </div>
        <button
          onClick={onNew}
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" /> Terima Barang
        </button>
      </div>

      <div className="flex items-center gap-2">
        <label htmlFor="wh-filter" className="text-sm text-muted-foreground">Gudang</label>
        <select
          id="wh-filter"
          value={warehouseId}
          onChange={(e) => setWarehouseId(e.target.value)}
          className="rounded-md border border-border bg-background px-3 py-1.5 text-sm"
        >
          <option value="">Semua gudang</option>
          {warehouses.map((w) => <option key={w.id} value={w.id}>{w.name}</option>)}
        </select>
      </div>

      <div className="overflow-x-auto rounded-xl border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b border-border bg-muted/40 text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-4 py-3">No. Penerimaan</th>
              <th className="px-4 py-3">Pemasok</th>
              <th className="px-4 py-3">Gudang</th>
              <th className="px-4 py-3 text-right">Diterima</th>
              <th className="px-4 py-3 text-right">Ditolak</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Tanggal</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-muted-foreground">Memuat data...</td></tr>
            )}
            {isError && (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-rose-600">Gagal memuat data: {errMsg(error)}</td></tr>
            )}
            {!isLoading && !isError && receipts.length === 0 && (
              <tr>
                <td colSpan={7} className="px-4 py-10 text-center text-muted-foreground">
                  Belum ada barang masuk. Tekan <strong>Terima Barang</strong> saat kiriman pemasok datang.
                </td>
              </tr>
            )}
            {receipts.map((r) => (
              <tr key={r.id} onClick={() => onOpen(r.id)} className="cursor-pointer border-b border-border last:border-0 hover:bg-muted/30">
                <td className="px-4 py-3 font-mono text-xs">{r.receipt_number}</td>
                <td className="px-4 py-3">{r.supplier_name}</td>
                <td className="px-4 py-3">{whName(r.warehouse_id)}</td>
                <td className="px-4 py-3 text-right">{fmtQty(r.total_accepted_qty)}</td>
                <td className="px-4 py-3 text-right">{num(r.total_rejected_qty) > 0 ? fmtQty(r.total_rejected_qty) : "-"}</td>
                <td className="px-4 py-3"><StatusBadge status={r.status} /></td>
                <td className="px-4 py-3 text-muted-foreground">{fmtDate(r.created_at)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  )
}

// ---------------------------------------------------------------------------
// Form (create / edit draft)
// ---------------------------------------------------------------------------

function ReceiptForm({
  form,
  setForm,
  onCancel,
  onSaved,
}: {
  form: FormState
  setForm: React.Dispatch<React.SetStateAction<FormState>>
  onCancel: () => void
  onSaved: (id: string) => void
}) {
  const { data: warehouses = [] } = useWarehouses()
  const { data: products = [] } = useProducts()
  const { data: locations = [] } = useWarehouseLocations(form.warehouse_id || null)
  const createMut = useCreateStockReceipt()
  const updateMut = useUpdateStockReceipt()

  const [search, setSearch] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [scanInfo, setScanInfo] = useState<string | null>(null)

  const internalLocations = useMemo(
    () => locations.filter((l) => l.type === "INTERNAL" && l.warehouse_id === form.warehouse_id),
    [locations, form.warehouse_id]
  )

  const searchResults = useMemo(() => {
    const q = search.trim().toLowerCase()
    if (!q) return []
    return products
      .filter((p) => p.name.toLowerCase().includes(q) || p.sku.toLowerCase().includes(q))
      .slice(0, 8)
  }, [search, products])

  const addProduct = (p: { id: string; name: string; sku: string }, qty = 1) => {
    setForm((f) => {
      const idx = f.lines.findIndex((l) => l.product_id === p.id)
      if (idx >= 0) {
        const lines = [...f.lines]
        lines[idx] = { ...lines[idx], accepted: String(num(lines[idx].accepted) + qty) }
        return { ...f, lines }
      }
      return {
        ...f,
        lines: [...f.lines, { product_id: p.id, product_name: p.name, product_sku: p.sku, accepted: String(qty), rejected: "0", reject_reason: "" }],
      }
    })
  }

  const handleScan = async (code: string) => {
    setError(null)
    // 1) Local match on SKU first (fast, works offline of the resolver).
    const local = products.find((p) => p.sku.toLowerCase() === code.toLowerCase())
    if (local) {
      addProduct(local)
      setScanInfo(`+1 ${local.name}`)
      return
    }
    // 2) Ask the backend resolver (barcode / marketplace mapping).
    try {
      const r = await api.wms.barcodes.resolve(code)
      const qty = num(r.multiplier) || 1
      addProduct({ id: r.product_id, name: r.name, sku: r.sku }, qty)
      setScanInfo(`+${qty} ${r.name}`)
    } catch {
      setScanInfo(null)
      setError(`Barcode "${code}" tidak dikenali. Cari produknya secara manual.`)
    }
  }

  useBarcodeScanner({ onScan: handleScan })

  const updateLine = (i: number, patch: Partial<Line>) =>
    setForm((f) => ({ ...f, lines: f.lines.map((l, idx) => (idx === i ? { ...l, ...patch } : l)) }))
  const removeLine = (i: number) => setForm((f) => ({ ...f, lines: f.lines.filter((_, idx) => idx !== i) }))

  const validate = (): string | null => {
    if (!form.warehouse_id) return "Pilih gudang tujuan."
    if (!form.dest_location_id) return "Pilih rak/lokasi tempat barang disimpan."
    if (!form.supplier_name.trim()) return "Isi nama pemasok."
    if (form.lines.length === 0) return "Tambahkan minimal satu barang."
    for (const [i, l] of form.lines.entries()) {
      const a = num(l.accepted), r = num(l.rejected)
      if (a < 0 || r < 0) return `Baris ${i + 1}: jumlah tidak boleh negatif.`
      if (a + r <= 0) return `Baris ${i + 1}: isi jumlah barang yang diterima.`
      if (r > 0 && !l.reject_reason.trim()) return `Baris ${i + 1}: isi alasan barang ditolak.`
    }
    return null
  }

  const save = async () => {
    const v = validate()
    if (v) { setError(v); return }
    setError(null)
    const payload: StockReceiptInput = {
      warehouse_id: form.warehouse_id,
      dest_location_id: form.dest_location_id,
      supplier_name: form.supplier_name.trim(),
      supplier_ref: form.supplier_ref.trim() || undefined,
      notes: form.notes.trim() || undefined,
      items: form.lines.map((l) => ({
        product_id: l.product_id,
        accepted_qty: num(l.accepted),
        rejected_qty: num(l.rejected),
        reject_reason: num(l.rejected) > 0 ? l.reject_reason.trim() : undefined,
      })),
    }
    try {
      const res = form.id
        ? await updateMut.mutateAsync({ id: form.id, data: payload })
        : await createMut.mutateAsync(payload)
      onSaved(res.receipt.id)
    } catch (e) {
      setError(errMsg(e))
    }
  }

  const saving = createMut.isPending || updateMut.isPending
  const input = "w-full rounded-md border border-border bg-background px-3 py-2 text-sm"

  return (
    <>
      <div className="flex items-center gap-3">
        <button onClick={onCancel} className="rounded-md p-1.5 hover:bg-muted" aria-label="Kembali"><ArrowLeft className="h-5 w-5" /></button>
        <h1 className="text-xl font-semibold">{form.id ? "Ubah Draf Penerimaan" : "Terima Barang"}</h1>
      </div>

      {/* Step 1 */}
      <section className="space-y-4 rounded-xl border border-border bg-card p-5">
        <h2 className="text-sm font-semibold text-foreground">1. Dari mana dan disimpan di mana</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <div>
            <label htmlFor="supplier" className="mb-1 block text-sm font-medium">Nama pemasok</label>
            <input id="supplier" className={input} value={form.supplier_name} placeholder="Contoh: CV Sumber Rejeki"
              onChange={(e) => setForm((f) => ({ ...f, supplier_name: e.target.value }))} />
          </div>
          <div>
            <label htmlFor="supplier-ref" className="mb-1 block text-sm font-medium">No. surat jalan pemasok <span className="text-muted-foreground">(opsional)</span></label>
            <input id="supplier-ref" className={input} value={form.supplier_ref}
              onChange={(e) => setForm((f) => ({ ...f, supplier_ref: e.target.value }))} />
          </div>
          <div>
            <label htmlFor="warehouse" className="mb-1 block text-sm font-medium">Gudang tujuan</label>
            <select id="warehouse" className={input} value={form.warehouse_id}
              onChange={(e) => setForm((f) => ({ ...f, warehouse_id: e.target.value, dest_location_id: "" }))}>
              <option value="">Pilih gudang</option>
              {warehouses.map((w) => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label htmlFor="location" className="mb-1 block text-sm font-medium">Simpan di rak / lokasi</label>
            <select id="location" className={input} value={form.dest_location_id} disabled={!form.warehouse_id}
              onChange={(e) => setForm((f) => ({ ...f, dest_location_id: e.target.value }))}>
              <option value="">{form.warehouse_id ? "Pilih lokasi" : "Pilih gudang dulu"}</option>
              {internalLocations.map((l) => <option key={l.id} value={l.id}>{l.code} — {l.name}</option>)}
            </select>
            {form.warehouse_id && internalLocations.length === 0 && (
              <p className="mt-1 text-xs text-amber-700">Gudang ini belum punya lokasi simpan. Tambahkan dulu di menu Warehouse &amp; Stock.</p>
            )}
          </div>
        </div>
      </section>

      {/* Step 2 */}
      <section className="space-y-4 rounded-xl border border-border bg-card p-5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="text-sm font-semibold text-foreground">2. Barang yang datang</h2>
          <span className="inline-flex items-center gap-1.5 text-xs text-muted-foreground">
            <ScanLine className="h-4 w-4" /> Scanner barcode aktif. Scan barang yang sama untuk menambah jumlahnya.
          </span>
        </div>

        <div className="relative">
          <input className={input} placeholder="Cari nama produk atau SKU..." value={search}
            onChange={(e) => setSearch(e.target.value)} aria-label="Cari produk" />
          {searchResults.length > 0 && (
            <ul className="absolute z-10 mt-1 w-full overflow-hidden rounded-md border border-border bg-popover shadow-md">
              {searchResults.map((p: Product) => (
                <li key={p.id}>
                  <button type="button" className="flex w-full justify-between px-3 py-2 text-left text-sm hover:bg-muted"
                    onClick={() => { addProduct(p); setSearch("") }}>
                    <span>{p.name}</span><span className="font-mono text-xs text-muted-foreground">{p.sku}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
        {scanInfo && <p className="text-xs text-emerald-700">Ditambahkan: {scanInfo}</p>}

        {form.lines.length === 0 ? (
          <p className="rounded-md border border-dashed border-border px-4 py-6 text-center text-sm text-muted-foreground">
            Belum ada barang. Scan barcode atau cari produk di atas.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-xs text-muted-foreground">
                <tr>
                  <th className="py-2 pr-2">Produk</th>
                  <th className="w-28 py-2 pr-2">Jumlah baik</th>
                  <th className="w-28 py-2 pr-2">Jumlah rusak</th>
                  <th className="py-2 pr-2">Alasan rusak</th>
                  <th className="w-10" />
                </tr>
              </thead>
              <tbody>
                {form.lines.map((l, i) => (
                  <tr key={l.product_id} className="border-t border-border align-top">
                    <td className="py-2 pr-2">
                      <div className="font-medium">{l.product_name}</div>
                      <div className="font-mono text-xs text-muted-foreground">{l.product_sku}</div>
                    </td>
                    <td className="py-2 pr-2">
                      <input type="number" min={0} className={input} value={l.accepted} aria-label={`Jumlah baik ${l.product_name}`}
                        onChange={(e) => updateLine(i, { accepted: e.target.value })} />
                    </td>
                    <td className="py-2 pr-2">
                      <input type="number" min={0} className={input} value={l.rejected} aria-label={`Jumlah rusak ${l.product_name}`}
                        onChange={(e) => updateLine(i, { rejected: e.target.value })} />
                    </td>
                    <td className="py-2 pr-2">
                      {num(l.rejected) > 0 ? (
                        <select className={input} value={l.reject_reason} aria-label={`Alasan rusak ${l.product_name}`}
                          onChange={(e) => updateLine(i, { reject_reason: e.target.value })}>
                          <option value="">Pilih alasan</option>
                          {REJECT_REASONS.map((r) => <option key={r} value={r}>{r}</option>)}
                        </select>
                      ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
                    </td>
                    <td className="py-2">
                      <button onClick={() => removeLine(i)} className="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-rose-600" aria-label={`Hapus ${l.product_name}`}>
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <p className="text-xs text-muted-foreground">Barang rusak tidak masuk stok jual. Barang itu dicatat di menu Barang Rusak / Scrap.</p>
      </section>

      <section className="rounded-xl border border-border bg-card p-5">
        <label htmlFor="notes" className="mb-1 block text-sm font-medium">Catatan <span className="text-muted-foreground">(opsional)</span></label>
        <textarea id="notes" rows={2} className={input} value={form.notes}
          onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))} />
      </section>

      {error && (
        <div role="alert" className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" /> {error}
        </div>
      )}

      <div className="flex justify-end gap-2">
        <button onClick={onCancel} className="rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted">Batal</button>
        <button onClick={save} disabled={saving}
          className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-60">
          {saving ? "Menyimpan..." : "Simpan Draf"}
        </button>
      </div>
    </>
  )
}

// ---------------------------------------------------------------------------
// Detail
// ---------------------------------------------------------------------------

function ReceiptDetail({
  id,
  onBack,
  onEdit,
  onNotice,
}: {
  id: string
  onBack: () => void
  onEdit: (f: FormState) => void
  onNotice: (n: { type: "success" | "error"; text: string } | null) => void
}) {
  const { data, isLoading, isError, error } = useStockReceipt(id)
  const { data: warehouses = [] } = useWarehouses()
  const { data: locations = [] } = useWarehouseLocations(data?.receipt.warehouse_id ?? null)
  const postMut = usePostStockReceipt()
  const cancelMut = useCancelStockReceipt()
  const [confirming, setConfirming] = useState(false)
  const [cancelOpen, setCancelOpen] = useState(false)
  const [reason, setReason] = useState("")

  if (isLoading) return <p className="text-sm text-muted-foreground">Memuat...</p>
  if (isError || !data) {
    return (
      <div className="space-y-3">
        <button onClick={onBack} className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"><ArrowLeft className="h-4 w-4" /> Kembali</button>
        <p className="text-sm text-rose-600">Gagal memuat penerimaan: {errMsg(error)}</p>
      </div>
    )
  }

  const r: StockReceipt = data.receipt
  const items = data.items ?? []
  const wh = warehouses.find((w) => w.id === r.warehouse_id)
  const loc = locations.find((l) => l.id === r.dest_location_id)

  const doPost = async () => {
    try {
      await postMut.mutateAsync(r.id)
      setConfirming(false)
      onNotice({ type: "success", text: `Penerimaan ${r.receipt_number} dikonfirmasi. Stok sudah bertambah.` })
    } catch (e) {
      setConfirming(false)
      onNotice({ type: "error", text: errMsg(e) })
    }
  }

  const doCancel = async () => {
    if (!reason.trim()) return
    try {
      await cancelMut.mutateAsync({ id: r.id, reason: reason.trim() })
      setCancelOpen(false)
      setReason("")
      onNotice({ type: "success", text: r.status === "POSTED" ? "Penerimaan dibatalkan dan stok dikembalikan." : "Draf dibatalkan." })
    } catch (e) {
      setCancelOpen(false)
      onNotice({ type: "error", text: errMsg(e) })
    }
  }

  const toForm = (): FormState => ({
    id: r.id,
    warehouse_id: r.warehouse_id,
    dest_location_id: r.dest_location_id,
    supplier_name: r.supplier_name,
    supplier_ref: r.supplier_ref ?? "",
    notes: r.notes ?? "",
    lines: items.map((it) => ({
      product_id: it.product_id,
      product_name: it.product_name,
      product_sku: it.product_sku,
      accepted: String(num(it.accepted_qty)),
      rejected: String(num(it.rejected_qty)),
      reject_reason: it.reject_reason ?? "",
    })),
  })

  return (
    <>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <button onClick={onBack} className="mt-0.5 rounded-md p-1.5 hover:bg-muted" aria-label="Kembali"><ArrowLeft className="h-5 w-5" /></button>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="font-mono text-lg font-semibold">{r.receipt_number}</h1>
              <StatusBadge status={r.status} />
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              {r.supplier_name}{r.supplier_ref ? ` · SJ ${r.supplier_ref}` : ""} → {wh?.name ?? "-"}{loc ? ` / ${loc.code}` : ""}
            </p>
          </div>
        </div>

        <div className="flex gap-2">
          {r.status === "DRAFT" && (
            <>
              <button onClick={() => onEdit(toForm())} className="rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted">Ubah</button>
              <button onClick={() => setConfirming(true)}
                className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90">
                Konfirmasi
              </button>
            </>
          )}
          {r.status !== "CANCELLED" && (
            <button onClick={() => setCancelOpen(true)} className="inline-flex items-center gap-1.5 rounded-lg border border-rose-200 px-4 py-2 text-sm text-rose-700 hover:bg-rose-50">
              <Ban className="h-4 w-4" /> Batalkan
            </button>
          )}
        </div>
      </div>

      {r.status === "DRAFT" && (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm text-amber-800">
          Masih draf. Stok belum berubah. Cek jumlahnya, lalu tekan Konfirmasi.
        </p>
      )}
      {r.status === "CANCELLED" && r.cancel_reason && (
        <p className="rounded-lg border border-border bg-muted/40 px-4 py-2.5 text-sm text-muted-foreground">
          Dibatalkan {fmtDate(r.cancelled_at)}. Alasan: {r.cancel_reason}
        </p>
      )}

      <div className="overflow-x-auto rounded-xl border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b border-border bg-muted/40 text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-4 py-3">Produk</th>
              <th className="px-4 py-3 text-right">Baik</th>
              <th className="px-4 py-3 text-right">Rusak</th>
              <th className="px-4 py-3">Alasan rusak</th>
            </tr>
          </thead>
          <tbody>
            {items.map((it) => (
              <tr key={it.id} className="border-b border-border last:border-0">
                <td className="px-4 py-3">
                  <div className="font-medium">{it.product_name}</div>
                  <div className="font-mono text-xs text-muted-foreground">{it.product_sku}</div>
                </td>
                <td className="px-4 py-3 text-right">{fmtQty(it.accepted_qty)}</td>
                <td className="px-4 py-3 text-right">{num(it.rejected_qty) > 0 ? fmtQty(it.rejected_qty) : "-"}</td>
                <td className="px-4 py-3 text-muted-foreground">{it.reject_reason || "-"}</td>
              </tr>
            ))}
          </tbody>
          <tfoot className="border-t border-border bg-muted/20 text-sm font-medium">
            <tr>
              <td className="px-4 py-3">Total</td>
              <td className="px-4 py-3 text-right">{fmtQty(r.total_accepted_qty)}</td>
              <td className="px-4 py-3 text-right">{num(r.total_rejected_qty) > 0 ? fmtQty(r.total_rejected_qty) : "-"}</td>
              <td />
            </tr>
          </tfoot>
        </table>
      </div>

      <dl className="grid gap-x-6 gap-y-1 text-xs text-muted-foreground sm:grid-cols-2">
        <div>Dibuat: {fmtDate(r.created_at)}</div>
        {r.posted_at && <div>Dikonfirmasi: {fmtDate(r.posted_at)}</div>}
        {r.notes && <div className="sm:col-span-2">Catatan: {r.notes}</div>}
      </dl>

      {confirming && (
        <Dialog title="Konfirmasi penerimaan?" onClose={() => setConfirming(false)}>
          <p className="text-sm text-muted-foreground">
            {fmtQty(r.total_accepted_qty)} barang baik akan ditambahkan ke stok {wh?.name ?? "gudang"}
            {num(r.total_rejected_qty) > 0 ? `, dan ${fmtQty(r.total_rejected_qty)} barang rusak dicatat ke Scrap` : ""}.
            Setelah dikonfirmasi, isi penerimaan tidak bisa diubah lagi.
          </p>
          <div className="mt-5 flex justify-end gap-2">
            <button onClick={() => setConfirming(false)} className="rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted">Batal</button>
            <button onClick={doPost} disabled={postMut.isPending}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-60">
              {postMut.isPending ? "Memproses..." : "Ya, konfirmasi"}
            </button>
          </div>
        </Dialog>
      )}

      {cancelOpen && (
        <Dialog title="Batalkan penerimaan?" onClose={() => setCancelOpen(false)}>
          <p className="text-sm text-muted-foreground">
            {r.status === "POSTED"
              ? "Stok yang sudah masuk akan dikurangi kembali. Pembatalan ditolak kalau barangnya sudah terpakai atau terjual."
              : "Draf ini tidak akan bisa dipakai lagi."}
          </p>
          <label htmlFor="cancel-reason" className="mb-1 mt-4 block text-sm font-medium">Alasan pembatalan</label>
          <input id="cancel-reason" className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
            value={reason} onChange={(e) => setReason(e.target.value)} placeholder="Contoh: salah input jumlah" />
          <div className="mt-5 flex justify-end gap-2">
            <button onClick={() => setCancelOpen(false)} className="rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted">Kembali</button>
            <button onClick={doCancel} disabled={!reason.trim() || cancelMut.isPending}
              className="rounded-lg bg-rose-600 px-4 py-2 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-60">
              {cancelMut.isPending ? "Memproses..." : "Batalkan penerimaan"}
            </button>
          </div>
        </Dialog>
      )}
    </>
  )
}

function Dialog({ title, onClose, children }: { title: string; onClose: () => void; children: React.ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" role="dialog" aria-modal="true" aria-label={title}>
      <div className="w-full max-w-md rounded-xl bg-background p-6 shadow-xl">
        <div className="mb-3 flex items-start justify-between">
          <h2 className="text-base font-semibold">{title}</h2>
          <button onClick={onClose} aria-label="Tutup"><X className="h-4 w-4" /></button>
        </div>
        {children}
      </div>
    </div>
  )
}
