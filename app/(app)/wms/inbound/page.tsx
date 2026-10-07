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
  Factory,
  Truck,
  Building2,
  Layers,
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
import { ExportModal, ExportButton } from "@/components/ui/ExportModal"
import { api, Product, StockReceipt, StockReceiptInput, StockReceiptStatus, StockReceiptType } from "@/lib/api"

// ---------------------------------------------------------------------------
// Helpers & Badges
// ---------------------------------------------------------------------------

const STATUS_LABEL: Record<StockReceiptStatus, { text: string; cls: string }> = {
  DRAFT: { text: "Draf", cls: "bg-amber-50 text-amber-700 border-amber-200" },
  POSTED: { text: "Sudah Masuk Stok", cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  CANCELLED: { text: "Dibatalkan", cls: "bg-slate-100 text-slate-600 border-slate-200" },
}

const TYPE_CONFIG: Record<StockReceiptType, { text: string; icon: React.ComponentType<{ className?: string }>; cls: string; desc: string }> = {
  PRODUCTION: {
    text: "Hasil Produksi",
    icon: Factory,
    cls: "bg-emerald-50 text-emerald-700 border-emerald-200",
    desc: "Barang jadi dari dapur, bengkel, atau lini produksi internal",
  },
  TRANSFER: {
    text: "Transfer Cabang",
    icon: Truck,
    cls: "bg-blue-50 text-blue-700 border-blue-200",
    desc: "Penerimaan kiriman mutasi stok dari gudang utama/pusat ke gudang cabang atau toko",
  },
  VENDOR: {
    text: "Pemasok Luar",
    icon: Building2,
    cls: "bg-purple-50 text-purple-700 border-purple-200",
    desc: "Pembelian bahan baku atau barang dagang dari vendor/supplier luar",
  },
}

const REJECT_REASONS = ["Rusak / pecah", "Kemasan sobek", "Kedaluwarsa", "Salah kirim", "Kurang lengkap", "Cacat produksi"]

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

function TypeBadge({ type }: { type?: StockReceiptType }) {
  const t = TYPE_CONFIG[type ?? "PRODUCTION"] ?? TYPE_CONFIG.PRODUCTION
  const Icon = t.icon
  return (
    <span className={`inline-flex items-center gap-1 rounded-full border px-2.5 py-0.5 text-xs font-medium ${t.cls}`}>
      <Icon className="h-3 w-3" />
      {t.text}
    </span>
  )
}

// ---------------------------------------------------------------------------
// Form Line & State
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
  receipt_type: StockReceiptType
  warehouse_id: string
  dest_location_id: string
  from_name: string
  from_warehouse_id: string
  source_ref: string
  notes: string
  lines: Line[]
}

const emptyForm = (warehouseId = ""): FormState => ({
  id: null,
  receipt_type: "PRODUCTION",
  warehouse_id: warehouseId,
  dest_location_id: "",
  from_name: "Hasil Produksi",
  from_warehouse_id: "",
  source_ref: "",
  notes: "",
  lines: [],
})

// ---------------------------------------------------------------------------
// Main Page View
// ---------------------------------------------------------------------------

type View = { mode: "list" } | { mode: "form" } | { mode: "detail"; id: string }

export default function InboundPage() {
  const [view, setView] = useState<View>({ mode: "list" })
  const [form, setForm] = useState<FormState>(emptyForm())
  const [notice, setNotice] = useState<{ type: "success" | "error"; text: string } | null>(null)

  const { data: warehouses = [] } = useWarehouses()

  const openNew = (initialType: StockReceiptType = "PRODUCTION") => {
    const defaultWh = warehouses[0]?.id ?? ""
    setForm({
      ...emptyForm(defaultWh),
      receipt_type: initialType,
      from_name: initialType === "PRODUCTION" ? "Hasil Produksi" : initialType === "TRANSFER" ? "" : "",
    })
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
        <ReceiptList onNew={() => openNew("PRODUCTION")} onOpen={(id) => { setNotice(null); setView({ mode: "detail", id }) }} />
      )}

      {view.mode === "form" && (
        <ReceiptForm
          form={form}
          setForm={setForm}
          onCancel={() => setView(form.id ? { mode: "detail", id: form.id } : { mode: "list" })}
          onSaved={(id) => {
            setNotice({ type: "success", text: "Draf tersimpan. Stok belum bertambah sebelum Anda menekan tombol Konfirmasi." })
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
// Inbound Receipts List
// ---------------------------------------------------------------------------

function ReceiptList({ onNew, onOpen }: { onNew: () => void; onOpen: (id: string) => void }) {
  const { data: warehouses = [] } = useWarehouses()
  const [warehouseId, setWarehouseId] = useState("")
  const [selectedType, setSelectedType] = useState<StockReceiptType | "ALL">("ALL")

  const { data: receipts = [], isLoading, isError, error } = useStockReceipts(
    warehouseId || null,
    selectedType === "ALL" ? null : selectedType
  )

  const whName = (id: string) => warehouses.find((w) => w.id === id)?.name ?? "-"
  const [exporting, setExporting] = useState(false)

  return (
    <>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold text-foreground">
            <PackagePlus className="h-6 w-6 text-primary" /> Barang Masuk
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Penerimaan barang dari <strong>Hasil Produksi</strong> (dapur/pabrik), <strong>Transfer Antar-Gudang</strong> (ke cabang/toko), atau <strong>Pemasok Luar</strong>. Stok cabang siap dijual di kasir POS setelah dikonfirmasi.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <ExportButton onClick={() => setExporting(true)} disabled={receipts.length === 0} />
          <button
            onClick={onNew}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 min-h-[44px] text-sm font-medium text-primary-foreground hover:bg-primary/90"
          >
            <Plus className="h-4 w-4" /> Terima Barang
          </button>
        </div>
      </div>

      <ExportModal<StockReceipt>
        open={exporting}
        onClose={() => setExporting(false)}
        title="Barang Masuk"
        filename="barang-masuk"
        allRows={receipts}
        visibleRows={receipts}
        visibleSummary={[
          `Gudang: ${warehouseId ? whName(warehouseId) : "Semua"}`,
          `Sumber: ${selectedType === "ALL" ? "Semua" : TYPE_CONFIG[selectedType].text}`,
        ]}
        columns={[
          { header: "No. Penerimaan", value: (r) => r.receipt_number, width: 20 },
          { header: "Tanggal", value: (r) => fmtDate(r.created_at), width: 20 },
          { header: "Sumber", value: (r) => TYPE_CONFIG[r.receipt_type]?.text ?? r.receipt_type, width: 18 },
          { header: "Dari", value: (r) => r.supplier_name || r.from_warehouse_name || r.from_name || "", width: 24 },
          { header: "Gudang Tujuan", value: (r) => whName(r.warehouse_id), width: 22 },
          { header: "Jumlah Item", value: (r) => r.item_count, width: 12, align: "right" },
          { header: "Qty Diterima", value: (r) => num(r.total_accepted_qty), width: 14, align: "right" },
          { header: "Qty Ditolak", value: (r) => num(r.total_rejected_qty), width: 14, align: "right" },
          { header: "Status", value: (r) => STATUS_LABEL[r.status]?.text ?? r.status, width: 18 },
          { header: "Ref", value: (r) => r.supplier_ref || r.source_ref || "", width: 18 },
          { header: "Catatan", value: (r) => r.notes ?? "", width: 30 },
        ]}
        filters={[
          { type: "dateRange", id: "date", label: "Tanggal penerimaan", getDate: (r) => r.created_at },
          {
            type: "select",
            id: "status",
            label: "Status",
            options: (Object.keys(STATUS_LABEL) as StockReceiptStatus[]).map((s) => ({ value: s, label: STATUS_LABEL[s].text })),
            match: (r, v) => r.status === v,
          },
          {
            type: "select",
            id: "type",
            label: "Sumber",
            options: (Object.keys(TYPE_CONFIG) as StockReceiptType[]).map((t) => ({ value: t, label: TYPE_CONFIG[t].text })),
            match: (r, v) => r.receipt_type === v,
          },
        ]}
      />

      {/* Filter Tabs & Warehouse Selector */}
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border pb-3">
        <div className="flex flex-wrap gap-1.5">
          <button
            type="button"
            onClick={() => setSelectedType("ALL")}
            className={`rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
              selectedType === "ALL"
                ? "bg-primary text-primary-foreground"
                : "border border-border bg-card text-muted-foreground hover:bg-muted"
            }`}
          >
            Semua Sumber
          </button>
          <button
            type="button"
            onClick={() => setSelectedType("PRODUCTION")}
            className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
              selectedType === "PRODUCTION"
                ? "bg-emerald-600 text-white"
                : "border border-border bg-card text-muted-foreground hover:bg-muted"
            }`}
          >
            <Factory className="h-3.5 w-3.5" /> Hasil Produksi
          </button>
          <button
            type="button"
            onClick={() => setSelectedType("TRANSFER")}
            className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
              selectedType === "TRANSFER"
                ? "bg-blue-600 text-white"
                : "border border-border bg-card text-muted-foreground hover:bg-muted"
            }`}
          >
            <Truck className="h-3.5 w-3.5" /> Transfer Cabang
          </button>
          <button
            type="button"
            onClick={() => setSelectedType("VENDOR")}
            className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
              selectedType === "VENDOR"
                ? "bg-purple-600 text-white"
                : "border border-border bg-card text-muted-foreground hover:bg-muted"
            }`}
          >
            <Building2 className="h-3.5 w-3.5" /> Pemasok Luar
          </button>
        </div>

        <div className="flex items-center gap-2">
          <label htmlFor="wh-filter" className="text-xs text-muted-foreground">Gudang Penerima:</label>
          <select
            id="wh-filter"
            value={warehouseId}
            onChange={(e) => setWarehouseId(e.target.value)}
            className="rounded-md border border-border bg-background px-3 py-1.5 text-xs"
          >
            <option value="">Semua gudang</option>
            {warehouses.map((w) => <option key={w.id} value={w.id}>{w.name}</option>)}
          </select>
        </div>
      </div>

      {/* Receipts Table */}
      <div className="overflow-x-auto rounded-xl border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b border-border bg-muted/40 text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-4 py-3">No. Penerimaan</th>
              <th className="px-4 py-3">Tipe Sumber</th>
              <th className="px-4 py-3">Asal / Pengirim</th>
              <th className="px-4 py-3">No. Referensi</th>
              <th className="px-4 py-3">Gudang Penerima</th>
              <th className="px-4 py-3 text-right">Diterima</th>
              <th className="px-4 py-3 text-right">Ditolak</th>
              <th className="px-4 py-3">Status</th>
              <th className="px-4 py-3">Tanggal</th>
            </tr>
          </thead>
          <tbody>
            {isLoading && (
              <tr><td colSpan={9} className="px-4 py-8 text-center text-muted-foreground">Memuat data penerimaan...</td></tr>
            )}
            {isError && (
              <tr><td colSpan={9} className="px-4 py-8 text-center text-rose-600">Gagal memuat data: {errMsg(error)}</td></tr>
            )}
            {!isLoading && !isError && receipts.length === 0 && (
              <tr>
                <td colSpan={9} className="px-4 py-10 text-center text-muted-foreground">
                  Belum ada catatan barang masuk. Tekan <strong>Terima Barang</strong> untuk mencatat hasil produksi, transfer cabang, atau barang pemasok.
                </td>
              </tr>
            )}
            {receipts.map((r) => {
              const displayOrigin =
                r.receipt_type === "TRANSFER"
                  ? (r.from_warehouse_name || (r.from_warehouse_id ? whName(r.from_warehouse_id) : r.from_name || "Gudang Pengirim"))
                  : (r.from_name || r.supplier_name || "-")
              return (
                <tr key={r.id} onClick={() => onOpen(r.id)} className="cursor-pointer border-b border-border last:border-0 hover:bg-muted/30">
                  <td className="px-4 py-3 font-mono text-xs font-semibold">{r.receipt_number}</td>
                  <td className="px-4 py-3"><TypeBadge type={r.receipt_type} /></td>
                  <td className="px-4 py-3 font-medium text-foreground">{displayOrigin}</td>
                  <td className="px-4 py-3 font-mono text-xs text-muted-foreground">{r.source_ref || r.supplier_ref || "-"}</td>
                  <td className="px-4 py-3">{whName(r.warehouse_id)}</td>
                  <td className="px-4 py-3 text-right font-medium text-emerald-700">{fmtQty(r.total_accepted_qty)}</td>
                  <td className="px-4 py-3 text-right">{num(r.total_rejected_qty) > 0 ? <span className="text-rose-600">{fmtQty(r.total_rejected_qty)}</span> : "-"}</td>
                  <td className="px-4 py-3"><StatusBadge status={r.status} /></td>
                  <td className="px-4 py-3 text-xs text-muted-foreground">{fmtDate(r.created_at)}</td>
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>
    </>
  )
}

// ---------------------------------------------------------------------------
// Inbound Receipt Form (Create / Edit Draft)
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

  // Other warehouses available as sender (excluding destination warehouse)
  const senderWarehouses = useMemo(
    () => warehouses.filter((w) => w.id !== form.warehouse_id),
    [warehouses, form.warehouse_id]
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
        lines: [
          ...f.lines,
          {
            product_id: p.id,
            product_name: p.name,
            product_sku: p.sku,
            accepted: String(qty),
            rejected: "0",
            reject_reason: "",
          },
        ],
      }
    })
  }

  const handleScan = async (code: string) => {
    setError(null)
    const local = products.find((p) => p.sku.toLowerCase() === code.toLowerCase())
    if (local) {
      addProduct(local)
      setScanInfo(`+1 ${local.name}`)
      return
    }
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

  const setReceiptType = (t: StockReceiptType) => {
    setForm((f) => ({
      ...f,
      receipt_type: t,
      from_name: t === "PRODUCTION" ? (f.from_name || "Dapur / Pabrik Utama") : "",
      from_warehouse_id: t === "TRANSFER" ? (senderWarehouses[0]?.id ?? "") : "",
    }))
  }

  const validate = (): string | null => {
    if (!form.warehouse_id) return "Pilih gudang penerima tujuan."
    if (!form.dest_location_id) return "Pilih rak/lokasi tempat barang disimpan."

    if (form.receipt_type === "TRANSFER") {
      if (!form.from_warehouse_id && !form.from_name.trim()) {
        return "Pilih gudang pengirim atau isi asal transfer."
      }
      if (form.from_warehouse_id && form.from_warehouse_id === form.warehouse_id) {
        return "Gudang pengirim (asal) tidak boleh sama dengan gudang penerima."
      }
    } else if (form.receipt_type === "VENDOR") {
      if (!form.from_name.trim()) return "Isi nama pemasok."
    }

    if (form.lines.length === 0) return "Tambahkan minimal satu barang."
    for (const [i, l] of form.lines.entries()) {
      const a = num(l.accepted), r = num(l.rejected)
      if (a < 0 || r < 0) return `Baris ${i + 1}: jumlah tidak boleh bernilai negatif.`
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
      receipt_type: form.receipt_type,
      warehouse_id: form.warehouse_id,
      dest_location_id: form.dest_location_id,
      from_name: form.from_name.trim() || undefined,
      from_warehouse_id: form.receipt_type === "TRANSFER" && form.from_warehouse_id ? form.from_warehouse_id : undefined,
      source_ref: form.source_ref.trim() || undefined,
      supplier_name: form.from_name.trim() || undefined,
      supplier_ref: form.source_ref.trim() || undefined,
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
  const inputClass = "w-full rounded-md border border-border bg-background px-3 py-2 text-sm"

  return (
    <>
      <div className="flex items-center gap-3">
        <button onClick={onCancel} className="rounded-md p-1.5 hover:bg-muted" aria-label="Kembali"><ArrowLeft className="h-5 w-5" /></button>
        <div>
          <h1 className="text-xl font-semibold">{form.id ? "Ubah Draf Barang Masuk" : "Penerimaan Barang Baru"}</h1>
          <p className="text-xs text-muted-foreground">Catat barang dari hasil produksi internal, transfer cabang, atau kiriman pemasok.</p>
        </div>
      </div>

      {/* Step 1: Source & Routing */}
      <section className="space-y-5 rounded-xl border border-border bg-card p-5">
        <div>
          <h2 className="text-sm font-semibold text-foreground">1. Pilih Sumber Barang Datang</h2>
          <p className="mt-0.5 text-xs text-muted-foreground">Tentukan apakah barang ini hasil produksi dapur/pabrik, kiriman transfer antar-gudang, atau pembelian dari pemasok luar.</p>
        </div>

        {/* 3 Source Selection Cards */}
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          {(["PRODUCTION", "TRANSFER", "VENDOR"] as StockReceiptType[]).map((t) => {
            const conf = TYPE_CONFIG[t]
            const Icon = conf.icon
            const isSelected = form.receipt_type === t
            return (
              <button
                key={t}
                type="button"
                onClick={() => setReceiptType(t)}
                className={`flex flex-col items-start rounded-xl border p-3.5 text-left transition-all ${
                  isSelected
                    ? "border-primary bg-primary/5 ring-1 ring-primary shadow-sm"
                    : "border-border bg-card hover:bg-muted/50"
                }`}
              >
                <div className="flex items-center gap-2">
                  <span className={`inline-flex rounded-lg p-2 ${isSelected ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"}`}>
                    <Icon className="h-4 w-4" />
                  </span>
                  <span className="font-semibold text-sm text-foreground">{conf.text}</span>
                </div>
                <p className="mt-2 text-xs text-muted-foreground leading-relaxed">{conf.desc}</p>
              </button>
            )
          })}
        </div>

        {/* Dynamic Fields Based on Source */}
        <div className="grid gap-4 md:grid-cols-2 pt-2 border-t border-border">
          {form.receipt_type === "PRODUCTION" && (
            <>
              <div>
                <label htmlFor="source-name" className="mb-1 block text-sm font-medium">Nama pemasok / asal produksi</label>
                <input
                  id="source-name"
                  className={inputClass}
                  value={form.from_name}
                  placeholder="Contoh: Dapur Pusat, Lini Produksi A"
                  onChange={(e) => setForm((f) => ({ ...f, from_name: e.target.value }))}
                />
              </div>
              <div>
                <label htmlFor="source-ref" className="mb-1 block text-sm font-medium">No. Batch / SPK Produksi <span className="text-muted-foreground">(opsional)</span></label>
                <input
                  id="source-ref"
                  className={inputClass}
                  value={form.source_ref}
                  placeholder="Contoh: BATCH-202610-001"
                  onChange={(e) => setForm((f) => ({ ...f, source_ref: e.target.value }))}
                />
              </div>
            </>
          )}

          {form.receipt_type === "TRANSFER" && (
            <>
              <div>
                <label htmlFor="from-warehouse" className="mb-1 block text-sm font-medium">Gudang Pengirim (Asal Transfer)</label>
                <select
                  id="from-warehouse"
                  className={inputClass}
                  value={form.from_warehouse_id}
                  onChange={(e) => {
                    const sel = warehouses.find((w) => w.id === e.target.value)
                    setForm((f) => ({ ...f, from_warehouse_id: e.target.value, from_name: sel?.name ?? "" }))
                  }}
                >
                  <option value="">Pilih gudang asal</option>
                  {senderWarehouses.map((w) => <option key={w.id} value={w.id}>{w.name}</option>)}
                </select>
              </div>
              <div>
                <label htmlFor="transfer-ref" className="mb-1 block text-sm font-medium">No. Surat Jalan / No. Transfer</label>
                <input
                  id="transfer-ref"
                  className={inputClass}
                  value={form.source_ref}
                  placeholder="Contoh: TR-202610-001 / DO-012"
                  onChange={(e) => setForm((f) => ({ ...f, source_ref: e.target.value }))}
                />
              </div>
            </>
          )}

          {form.receipt_type === "VENDOR" && (
            <>
              <div>
                <label htmlFor="vendor-name" className="mb-1 block text-sm font-medium">Nama Pemasok / Vendor</label>
                <input
                  id="vendor-name"
                  className={inputClass}
                  value={form.from_name}
                  placeholder="Contoh: CV Sumber Rejeki"
                  onChange={(e) => setForm((f) => ({ ...f, from_name: e.target.value }))}
                />
              </div>
              <div>
                <label htmlFor="vendor-ref" className="mb-1 block text-sm font-medium">No. Surat Jalan Pemasok <span className="text-muted-foreground">(opsional)</span></label>
                <input
                  id="vendor-ref"
                  className={inputClass}
                  value={form.source_ref}
                  placeholder="Contoh: SJ-2026-888"
                  onChange={(e) => setForm((f) => ({ ...f, source_ref: e.target.value }))}
                />
              </div>
            </>
          )}

          {/* Receiving Destination Warehouse & Internal Rack Location */}
          <div>
            <label htmlFor="dest-warehouse" className="mb-1 block text-sm font-medium">Gudang Penerima (Tujuan)</label>
            <select
              id="dest-warehouse"
              className={inputClass}
              value={form.warehouse_id}
              onChange={(e) => setForm((f) => ({ ...f, warehouse_id: e.target.value, dest_location_id: "" }))}
            >
              <option value="">Pilih gudang penerima</option>
              {warehouses.map((w) => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>

          <div>
            <label htmlFor="dest-location" className="mb-1 block text-sm font-medium">Simpan di Rak / Lokasi</label>
            <select
              id="dest-location"
              className={inputClass}
              value={form.dest_location_id}
              disabled={!form.warehouse_id}
              onChange={(e) => setForm((f) => ({ ...f, dest_location_id: e.target.value }))}
            >
              <option value="">{form.warehouse_id ? "Pilih lokasi rak" : "Pilih gudang penerima dulu"}</option>
              {internalLocations.map((l) => <option key={l.id} value={l.id}>{l.code} — {l.name}</option>)}
            </select>
            {form.warehouse_id && internalLocations.length === 0 && (
              <p className="mt-1 text-xs text-amber-700">Gudang ini belum memiliki rak simpan internal. Buat di menu Warehouse &amp; Stock.</p>
            )}
          </div>
        </div>
      </section>

      {/* Step 2: Line Items with Barcode Scanner & Search */}
      <section className="space-y-4 rounded-xl border border-border bg-card p-5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="text-sm font-semibold text-foreground">2. Barang yang Diterima</h2>
            <p className="text-xs text-muted-foreground">Scan barcode fisik barang atau cari nama produk/SKU di bawah.</p>
          </div>
          <span className="inline-flex items-center gap-1.5 rounded-md bg-muted px-2.5 py-1 text-xs font-medium text-foreground">
            <ScanLine className="h-3.5 w-3.5 text-primary" /> Scanner Barcode Siap
          </span>
        </div>

        <div className="relative">
          <input
            className={inputClass}
            placeholder="Ketik nama produk atau scan barcode SKU..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-label="Cari produk"
          />
          {searchResults.length > 0 && (
            <ul className="absolute z-10 mt-1 w-full overflow-hidden rounded-md border border-border bg-popover shadow-md">
              {searchResults.map((p: Product) => (
                <li key={p.id}>
                  <button
                    type="button"
                    className="flex w-full justify-between px-3 py-2.5 text-left text-sm hover:bg-muted"
                    onClick={() => { addProduct(p); setSearch("") }}
                  >
                    <span className="font-medium">{p.name}</span>
                    <span className="font-mono text-xs text-muted-foreground">{p.sku}</span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
        {scanInfo && <p className="text-xs font-medium text-emerald-700">Berhasil ditambahkan: {scanInfo}</p>}

        {form.lines.length === 0 ? (
          <p className="rounded-md border border-dashed border-border px-4 py-8 text-center text-sm text-muted-foreground">
            Belum ada barang dalam daftar penerimaan. Scan barcode atau cari produk di atas.
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-xs text-muted-foreground border-b border-border">
                <tr>
                  <th className="py-2.5 pr-2">Nama Produk / SKU</th>
                  <th className="w-32 py-2.5 pr-2">Jumlah Baik</th>
                  <th className="w-32 py-2.5 pr-2">Jumlah Rusak</th>
                  <th className="py-2.5 pr-2">Alasan Rusak</th>
                  <th className="w-10" />
                </tr>
              </thead>
              <tbody>
                {form.lines.map((l, i) => (
                  <tr key={l.product_id} className="border-t border-border align-top">
                    <td className="py-2.5 pr-2">
                      <div className="font-medium">{l.product_name}</div>
                      <div className="font-mono text-xs text-muted-foreground">{l.product_sku}</div>
                    </td>
                    <td className="py-2.5 pr-2">
                      <input
                        type="number"
                        min={0}
                        className={inputClass}
                        value={l.accepted}
                        aria-label={`Jumlah baik ${l.product_name}`}
                        onChange={(e) => updateLine(i, { accepted: e.target.value })}
                      />
                    </td>
                    <td className="py-2.5 pr-2">
                      <input
                        type="number"
                        min={0}
                        className={inputClass}
                        value={l.rejected}
                        aria-label={`Jumlah rusak ${l.product_name}`}
                        onChange={(e) => updateLine(i, { rejected: e.target.value })}
                      />
                    </td>
                    <td className="py-2.5 pr-2">
                      {num(l.rejected) > 0 ? (
                        <select
                          className={inputClass}
                          value={l.reject_reason}
                          aria-label={`Alasan rusak ${l.product_name}`}
                          onChange={(e) => updateLine(i, { reject_reason: e.target.value })}
                        >
                          <option value="">Pilih alasan rusak</option>
                          {REJECT_REASONS.map((r) => <option key={r} value={r}>{r}</option>)}
                        </select>
                      ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
                    </td>
                    <td className="py-2.5">
                      <button
                        onClick={() => removeLine(i)}
                        className="rounded p-1.5 text-muted-foreground hover:bg-muted hover:text-rose-600"
                        aria-label={`Hapus ${l.product_name}`}
                      >
                        <Trash2 className="h-4 w-4" />
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <div className="flex items-center gap-2 rounded-lg bg-muted/40 p-3 text-xs text-muted-foreground">
          <Layers className="h-4 w-4 shrink-0 text-muted-foreground" />
          <span>Barang baik langsung masuk ke rak simpan. Barang rusak otomatis dipisahkan ke lokasi Scrap (@SCRAP) dan tidak dijual di kasir POS.</span>
        </div>
      </section>

      {/* Step 3: Notes */}
      <section className="rounded-xl border border-border bg-card p-5">
        <label htmlFor="notes" className="mb-1 block text-sm font-medium">Catatan Penerimaan <span className="text-muted-foreground">(opsional)</span></label>
        <textarea
          id="notes"
          rows={2}
          className={inputClass}
          value={form.notes}
          placeholder="Contoh: Kondisi paket rapi, batch produksi pagi..."
          onChange={(e) => setForm((f) => ({ ...f, notes: e.target.value }))}
        />
      </section>

      {error && (
        <div role="alert" className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" /> {error}
        </div>
      )}

      <div className="flex justify-end gap-2.5">
        <button onClick={onCancel} className="rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted">Batal</button>
        <button
          onClick={save}
          disabled={saving}
          className="rounded-lg bg-primary px-5 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-60"
        >
          {saving ? "Menyimpan Draf..." : "Simpan Draf Penerimaan"}
        </button>
      </div>
    </>
  )
}

// ---------------------------------------------------------------------------
// Inbound Receipt Detail & Confirmation
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

  if (isLoading) return <p className="text-sm text-muted-foreground">Memuat detail penerimaan...</p>
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
  const fromWh = warehouses.find((w) => w.id === r.from_warehouse_id)

  const originTitle =
    r.receipt_type === "TRANSFER"
      ? (r.from_warehouse_name || fromWh?.name || r.from_name || "Gudang Pengirim")
      : (r.from_name || r.supplier_name || "-")

  const doPost = async () => {
    try {
      await postMut.mutateAsync(r.id)
      setConfirming(false)
      onNotice({ type: "success", text: `Penerimaan ${r.receipt_number} berhasil dikonfirmasi! Stok sudah bertambah di ${wh?.name ?? "gudang"}.` })
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
      onNotice({ type: "success", text: r.status === "POSTED" ? "Penerimaan dibatalkan dan stok dikembalikan ke posisi semula." : "Draf penerimaan dibatalkan." })
    } catch (e) {
      setCancelOpen(false)
      onNotice({ type: "error", text: errMsg(e) })
    }
  }

  const toForm = (): FormState => ({
    id: r.id,
    receipt_type: r.receipt_type ?? "PRODUCTION",
    warehouse_id: r.warehouse_id,
    dest_location_id: r.dest_location_id,
    from_name: r.from_name || r.supplier_name || "",
    from_warehouse_id: r.from_warehouse_id ?? "",
    source_ref: r.source_ref || r.supplier_ref || "",
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
            <div className="flex items-center gap-2.5">
              <h1 className="font-mono text-lg font-bold">{r.receipt_number}</h1>
              <TypeBadge type={r.receipt_type} />
              <StatusBadge status={r.status} />
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              Asal: <strong className="text-foreground">{originTitle}</strong>
              {(r.source_ref || r.supplier_ref) && ` · Ref: ${r.source_ref || r.supplier_ref}`}
              {" → Tujuan: "}
              <strong className="text-foreground">{wh?.name ?? "-"}</strong>
              {loc ? ` (Rak: ${loc.code})` : ""}
            </p>
          </div>
        </div>

        <div className="flex gap-2">
          {r.status === "DRAFT" && (
            <>
              <button onClick={() => onEdit(toForm())} className="rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted font-medium">Ubah</button>
              <button
                onClick={() => setConfirming(true)}
                className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
              >
                Konfirmasi
              </button>
            </>
          )}
          {r.status !== "CANCELLED" && (
            <button
              onClick={() => setCancelOpen(true)}
              className="inline-flex items-center gap-1.5 rounded-lg border border-rose-200 px-4 py-2 text-sm text-rose-700 hover:bg-rose-50"
            >
              <Ban className="h-4 w-4" /> Batalkan
            </button>
          )}
        </div>
      </div>

      {r.status === "DRAFT" && (
        <p className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-2.5 text-sm text-amber-800">
          Status masih Draf. Stok belum masuk ke buku besar (ledger). Silakan periksa barang, lalu tekan <strong>Konfirmasi Masuk Stok</strong>.
        </p>
      )}
      {r.status === "POSTED" && (
        <p className="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-2.5 text-sm text-emerald-800">
          Barang sudah resmi masuk ke gudang <strong>{wh?.name ?? ""}</strong> dan tercatat di buku besar mutasi. Stok siap dipindahkan atau dijual di kasir POS.
        </p>
      )}
      {r.status === "CANCELLED" && r.cancel_reason && (
        <p className="rounded-lg border border-border bg-muted/40 px-4 py-2.5 text-sm text-muted-foreground">
          Dibatalkan pada {fmtDate(r.cancelled_at)}. Alasan: {r.cancel_reason}
        </p>
      )}

      {/* Items Breakdown */}
      <div className="overflow-x-auto rounded-xl border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="border-b border-border bg-muted/40 text-left text-xs uppercase text-muted-foreground">
            <tr>
              <th className="px-4 py-3">Produk</th>
              <th className="px-4 py-3 text-right">Diterima Baik</th>
              <th className="px-4 py-3 text-right">Ditolak Rusak</th>
              <th className="px-4 py-3">Alasan Rusak</th>
            </tr>
          </thead>
          <tbody>
            {items.map((it) => (
              <tr key={it.id} className="border-b border-border last:border-0">
                <td className="px-4 py-3">
                  <div className="font-medium text-foreground">{it.product_name}</div>
                  <div className="font-mono text-xs text-muted-foreground">{it.product_sku}</div>
                </td>
                <td className="px-4 py-3 text-right font-medium text-emerald-700">{fmtQty(it.accepted_qty)}</td>
                <td className="px-4 py-3 text-right">{num(it.rejected_qty) > 0 ? <span className="font-medium text-rose-600">{fmtQty(it.rejected_qty)}</span> : "-"}</td>
                <td className="px-4 py-3 text-muted-foreground">{it.reject_reason || "-"}</td>
              </tr>
            ))}
          </tbody>
          <tfoot className="border-t border-border bg-muted/20 text-sm font-medium">
            <tr>
              <td className="px-4 py-3 font-semibold">Total Unit</td>
              <td className="px-4 py-3 text-right font-semibold text-emerald-700">{fmtQty(r.total_accepted_qty)}</td>
              <td className="px-4 py-3 text-right font-semibold text-rose-600">{num(r.total_rejected_qty) > 0 ? fmtQty(r.total_rejected_qty) : "-"}</td>
              <td />
            </tr>
          </tfoot>
        </table>
      </div>

      <dl className="grid gap-x-6 gap-y-1.5 text-xs text-muted-foreground sm:grid-cols-2 rounded-xl border border-border bg-card p-4">
        <div>Tipe Penerimaan: <span className="font-medium text-foreground">{TYPE_CONFIG[r.receipt_type ?? "PRODUCTION"]?.text}</span></div>
        <div>Asal / Pengirim: <span className="font-medium text-foreground">{originTitle}</span></div>
        <div>Waktu Dibuat: {fmtDate(r.created_at)}</div>
        {r.posted_at && <div>Waktu Dikonfirmasi: {fmtDate(r.posted_at)}</div>}
        {r.notes && <div className="sm:col-span-2">Catatan: {r.notes}</div>}
      </dl>

      {/* Confirmation Dialog */}
      {confirming && (
        <Dialog title="Konfirmasi penerimaan?" onClose={() => setConfirming(false)}>
          <p className="text-sm text-muted-foreground">
            {fmtQty(r.total_accepted_qty)} barang baik akan ditambahkan ke stok {wh?.name ?? "gudang"}
            {num(r.total_rejected_qty) > 0 ? `, dan ${fmtQty(r.total_rejected_qty)} barang rusak dicatat ke Scrap` : ""}.
            Setelah dikonfirmasi, isi penerimaan tidak bisa diubah lagi.
          </p>
          <div className="mt-5 flex justify-end gap-2">
            <button onClick={() => setConfirming(false)} className="rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted">Batal</button>
            <button
              onClick={doPost}
              disabled={postMut.isPending}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-60"
            >
              {postMut.isPending ? "Memproses..." : "Ya, konfirmasi"}
            </button>
          </div>
        </Dialog>
      )}

      {/* Cancel Dialog */}
      {cancelOpen && (
        <Dialog title="Batalkan Penerimaan?" onClose={() => setCancelOpen(false)}>
          <p className="text-sm text-muted-foreground">
            {r.status === "POSTED"
              ? "Stok yang telah masuk akan dibalikkan kembali (reverse movement). Pembatalan otomatis ditolak jika barang telah terjual di kasir POS atau dipindahkan ke tempat lain."
              : "Draf penerimaan ini akan ditandai dibatalkan dan tidak dapat digunakan lagi."}
          </p>
          <label htmlFor="cancel-reason" className="mb-1 mt-4 block text-sm font-medium">Alasan Pembatalan</label>
          <input
            id="cancel-reason"
            className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Contoh: salah input batch, kiriman ditarik kembali..."
          />
          <div className="mt-5 flex justify-end gap-2">
            <button onClick={() => setCancelOpen(false)} className="rounded-lg border border-border px-4 py-2 text-sm hover:bg-muted">Kembali</button>
            <button
              onClick={doCancel}
              disabled={!reason.trim() || cancelMut.isPending}
              className="rounded-lg bg-rose-600 px-4 py-2 text-sm font-medium text-white hover:bg-rose-700 disabled:opacity-60"
            >
              {cancelMut.isPending ? "Memproses..." : "Batalkan Penerimaan"}
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
      <div className="w-full max-w-md rounded-xl bg-background p-6 shadow-xl border border-border">
        <div className="mb-3 flex items-start justify-between">
          <h2 className="text-base font-semibold text-foreground">{title}</h2>
          <button onClick={onClose} aria-label="Tutup" className="rounded p-1 hover:bg-muted"><X className="h-4 w-4" /></button>
        </div>
        {children}
      </div>
    </div>
  )
}
