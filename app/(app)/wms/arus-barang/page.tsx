"use client"

import React, { useEffect, useMemo, useState } from "react"
import { useSearchParams, useRouter } from "next/navigation"
import {
  ArrowDownToLine,
  ArrowUpFromLine,
  Plus,
  Trash2,
  X,
  CheckCircle2,
  AlertCircle,
  ArrowLeft,
  Factory,
  Truck,
  Building2,
  Layers,
  Settings,
  Search,
  Warehouse,
  Boxes,
  BadgeAlert,
  ClipboardList,
} from "lucide-react"
import {
  useWarehouses,
  useWarehouseLocations,
  useStockReceipts,
  useStockReceipt,
  useCreateStockReceipt,
  usePostStockReceipt,
  useCancelStockReceipt,
  useReleaseStockReceipt,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import { ExportModal, ExportButton } from "@/components/ui/ExportModal"
import { validateReceiptLines, isPastDate } from "@/lib/wms/validation"
import {
  StockReceipt,
  StockReceiptInput,
  StockReceiptStatus,
  StockReceiptType,
} from "@/lib/api"
import { PutawayView } from "@/components/wms/PutawayView"
import { TraceBatchView } from "@/components/wms/TraceBatchView"
import { WMSSettingsModal } from "@/components/wms/WMSSettingsModal"
import { QCQuarantineView } from "@/components/wms/QCQuarantineView"
import DeliveryOrdersPanel from "@/components/wms/DeliveryOrdersPanel"
import { WavePickingSubView } from "@/components/wms/WavePickingSubView"
import { PackingStationSubView } from "@/components/wms/PackingStationSubView"

// ---------------------------------------------------------------------------
// Helpers & Badges
// ---------------------------------------------------------------------------

const STATUS_LABEL: Record<StockReceiptStatus, { text: string; cls: string }> = {
  DRAFT: { text: "Draf", cls: "bg-amber-50 text-amber-700 border-amber-200" },
  POSTED: { text: "Sudah Masuk Stok", cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  CANCELLED: { text: "Dibatalkan", cls: "bg-slate-100 text-slate-600 border-slate-200" },
}

const TYPE_CONFIG: Record<
  StockReceiptType,
  { text: string; icon: React.ComponentType<{ className?: string }>; cls: string; desc: string }
> = {
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
    desc: "Penerimaan mutasi stok dari gudang utama/pusat",
  },
  VENDOR: {
    text: "Pemasok Luar",
    icon: Building2,
    cls: "bg-purple-50 text-purple-700 border-purple-200",
    desc: "Pembelian bahan baku atau barang dagang dari vendor luar",
  },
}

const REJECT_REASONS = [
  "Rusak / pecah",
  "Kemasan sobek",
  "Kedaluwarsa",
  "Salah kirim",
  "Kurang lengkap",
  "Cacat produksi",
]

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
  return d.toLocaleString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
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
// Main Arus Barang Page Component
// ---------------------------------------------------------------------------

type Mode = "masuk" | "keluar"
type MasukTab = "penerimaan" | "putaway" | "trace" | "qc"
type KeluarTab = "surat_jalan" | "picking" | "packing" | "manifest"

export default function ArusBarangPage() {
  const router = useRouter()
  const searchParams = useSearchParams()

  // URL ?mode= is the single source of truth (reactive). localStorage is only
  // a fallback when the URL carries no mode, and is applied via router.replace
  // so URL and UI can never diverge.
  const urlMode = searchParams.get("mode")?.toLowerCase()
  const mode: Mode = urlMode === "keluar" ? "keluar" : "masuk"
  const [masukTab, setMasukTab] = useState<MasukTab>("penerimaan")
  const [keluarTab, setKeluarTab] = useState<KeluarTab>("surat_jalan")
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>("")
  const [showSettingsModal, setShowSettingsModal] = useState(false)

  const { data: warehouses = [] } = useWarehouses()

  useEffect(() => {
    if (urlMode === "masuk" || urlMode === "keluar") {
      localStorage.setItem("wms_arus_barang_mode", urlMode)
      return
    }
    const saved = localStorage.getItem("wms_arus_barang_mode")
    router.replace(`/wms/arus-barang?mode=${saved === "keluar" ? "keluar" : "masuk"}`)
  }, [urlMode, router])

  // Select default warehouse if available
  useEffect(() => {
    if (!selectedWarehouseId && warehouses.length > 0) {
      setSelectedWarehouseId(warehouses[0].id)
    }
  }, [warehouses, selectedWarehouseId])

  const switchMode = (newMode: Mode) => {
    localStorage.setItem("wms_arus_barang_mode", newMode)
    router.replace(`/wms/arus-barang?mode=${newMode}`)
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6 p-4 md:p-6">
      {/* Top Header & Mode Toggle */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-200 pb-5">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 tracking-tight flex items-center gap-2">
            Arus Barang (Masuk & Keluar)
          </h1>
          <p className="mt-1 text-xs text-slate-500 max-w-2xl leading-relaxed">
            Pusat operasional aliran fisik barang. Terima barang ke Staging Inbound, kelola QC, lakukan Putaway ke rak,
            dan terbitkan Surat Jalan (DO) dengan rotasi FEFO per batch.
          </p>
        </div>

        <div className="flex items-center gap-3">
          {/* MASUK | KELUAR Toggle */}
          <div className="inline-flex rounded-lg bg-slate-100 p-1 border border-slate-200 shadow-inner">
            <button
              onClick={() => switchMode("masuk")}
              className={`inline-flex items-center gap-2 px-4 py-2 rounded-md text-xs font-semibold transition-all ${
                mode === "masuk"
                  ? "bg-emerald-600 text-white shadow-sm"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              <ArrowDownToLine className="h-4 w-4" />
              BARANG MASUK
            </button>
            <button
              onClick={() => switchMode("keluar")}
              className={`inline-flex items-center gap-2 px-4 py-2 rounded-md text-xs font-semibold transition-all ${
                mode === "keluar"
                  ? "bg-blue-600 text-white shadow-sm"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              <ArrowUpFromLine className="h-4 w-4" />
              BARANG KELUAR
            </button>
          </div>

          <button
            onClick={() => setShowSettingsModal(true)}
            className="p-2 text-slate-500 hover:text-slate-800 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 transition-colors shadow-sm"
            title="Pengaturan WMS & Kebijakan Rilis"
          >
            <Settings className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Global Warehouse Filter Toolbar */}
      <div className="flex flex-wrap items-center justify-between gap-3 bg-white p-3 rounded-xl border border-slate-200 shadow-sm">
        <div className="flex items-center gap-2">
          <Warehouse className="h-4 w-4 text-slate-400" />
          <span className="text-xs font-medium text-slate-600">Pilih Gudang Operasional:</span>
          <select
            value={selectedWarehouseId}
            onChange={(e) => setSelectedWarehouseId(e.target.value)}
            className="text-xs font-medium px-3 py-1.5 border border-slate-300 rounded-lg bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-primary/20"
          >
            {warehouses.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name}
              </option>
            ))}
          </select>
        </div>

        {/* Sub-tabs per mode */}
        {mode === "masuk" ? (
          <div className="flex items-center gap-1 overflow-x-auto text-xs">
            <button
              onClick={() => setMasukTab("penerimaan")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                masukTab === "penerimaan"
                  ? "bg-emerald-50 text-emerald-800 font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Penerimaan (GR)
            </button>
            <button
              onClick={() => setMasukTab("putaway")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                masukTab === "putaway"
                  ? "bg-emerald-50 text-emerald-800 font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Putaway ke Rak
            </button>
            <button
              onClick={() => setMasukTab("trace")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                masukTab === "trace"
                  ? "bg-emerald-50 text-emerald-800 font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Lacak Batch (Traceability)
            </button>
            <button
              onClick={() => setMasukTab("qc")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                masukTab === "qc"
                  ? "bg-emerald-50 text-emerald-800 font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              QC & Karantina
            </button>
          </div>
        ) : (
          <div className="flex items-center gap-1 overflow-x-auto text-xs">
            <button
              onClick={() => setKeluarTab("surat_jalan")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                keluarTab === "surat_jalan"
                  ? "bg-blue-50 text-blue-800 font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Surat Jalan (DO)
            </button>
            <button
              onClick={() => setKeluarTab("picking")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                keluarTab === "picking"
                  ? "bg-blue-50 text-blue-800 font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Picking Wave
            </button>
            <button
              onClick={() => setKeluarTab("packing")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                keluarTab === "packing"
                  ? "bg-blue-50 text-blue-800 font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Packing Station
            </button>
            <button
              onClick={() => setKeluarTab("manifest")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                keluarTab === "manifest"
                  ? "bg-blue-50 text-blue-800 font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Manifest & Muat
            </button>
          </div>
        )}
      </div>

      {/* Main Content Area */}
      {mode === "masuk" ? (
        <div>
          {masukTab === "penerimaan" && (
            <InboundReceivingSubView
              warehouseId={selectedWarehouseId}
              onGoToPutaway={() => setMasukTab("putaway")}
            />
          )}
          {masukTab === "putaway" && (
            <PutawayView warehouseId={selectedWarehouseId} />
          )}
          {masukTab === "trace" && <TraceBatchView />}
          {masukTab === "qc" && <QCQuarantineView warehouseId={selectedWarehouseId} />}
        </div>
      ) : (
        <div>
          {keluarTab === "surat_jalan" && (
            <DeliveryOrdersPanel embedded />
          )}
          {keluarTab === "picking" && (
            <WavePickingSubView warehouseId={selectedWarehouseId} />
          )}
          {keluarTab === "packing" && (
            <PackingStationSubView warehouseId={selectedWarehouseId} />
          )}
          {keluarTab === "manifest" && (
            <div className="rounded-xl border border-slate-200 bg-white p-12 text-center text-slate-400">
              <Truck className="mx-auto h-12 w-12 text-slate-300 mb-2" />
              <h3 className="text-base font-semibold text-slate-800">Manifest Ekspedisi & Serah Terima (Sprint 4)</h3>
              <p className="text-xs text-slate-500 mt-1 max-w-md mx-auto">
                Pencatatan nomor kendaraan, TTD digital sopir ekspedisi, dan surat muatan akan diaktifkan pada Sprint 4.
              </p>
            </div>
          )}
        </div>
      )}

      {/* Settings Modal */}
      <WMSSettingsModal
        isOpen={showSettingsModal}
        onClose={() => setShowSettingsModal(false)}
      />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Inbound Receiving Sub-View (Barang Masuk)
// ---------------------------------------------------------------------------

interface Line {
  product_id: string
  product_name: string
  product_sku: string
  batch_number: string
  expiry_date: string
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

function InboundReceivingSubView({
  warehouseId,
  onGoToPutaway,
}: {
  warehouseId: string
  onGoToPutaway: () => void
}) {
  const [view, setView] = useState<{ mode: "list" } | { mode: "form" } | { mode: "detail"; id: string }>({
    mode: "list",
  })
  const [form, setForm] = useState<FormState>(emptyForm(warehouseId))
  const [notice, setNotice] = useState<{ type: "success" | "error"; text: string } | null>(null)
  const [selectedType, setSelectedType] = useState<StockReceiptType | "ALL">("ALL")
  const [exporting, setExporting] = useState(false)

  const { data: warehouses = [] } = useWarehouses()
  const { data: receipts = [], isLoading, isError, error, refetch } = useStockReceipts(
    warehouseId || null,
    selectedType === "ALL" ? null : selectedType
  )

  const whName = (id: string) => warehouses.find((w) => w.id === id)?.name ?? "-"

  const openNew = (initialType: StockReceiptType = "PRODUCTION") => {
    setForm({
      ...emptyForm(warehouseId),
      receipt_type: initialType,
      from_name: initialType === "PRODUCTION" ? "Hasil Produksi" : "",
    })
    setNotice(null)
    setView({ mode: "form" })
  }

  return (
    <div className="space-y-6">
      {notice && (
        <div
          role="status"
          className={`flex items-start gap-2 rounded-lg border px-4 py-3 text-xs ${
            notice.type === "success"
              ? "border-emerald-200 bg-emerald-50 text-emerald-800"
              : "border-rose-200 bg-rose-50 text-rose-800"
          }`}
        >
          {notice.type === "success" ? (
            <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-600" />
          ) : (
            <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
          )}
          <span className="flex-1">{notice.text}</span>
          <button onClick={() => setNotice(null)} aria-label="Tutup pesan">
            <X className="h-4 w-4" />
          </button>
        </div>
      )}

      {view.mode === "list" && (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="text-xs text-slate-500 font-medium">Filter Sumber:</span>
              <div className="inline-flex rounded-lg border border-slate-200 bg-white p-1 text-xs">
                {(["ALL", "PRODUCTION", "TRANSFER", "VENDOR"] as const).map((t) => (
                  <button
                    key={t}
                    onClick={() => setSelectedType(t)}
                    className={`px-2.5 py-1 rounded font-medium transition-colors ${
                      selectedType === t
                        ? "bg-slate-900 text-white"
                        : "text-slate-600 hover:text-slate-900"
                    }`}
                  >
                    {t === "ALL" ? "Semua" : TYPE_CONFIG[t].text}
                  </button>
                ))}
              </div>
            </div>

            <div className="flex items-center gap-2">
              <ExportButton onClick={() => setExporting(true)} disabled={receipts.length === 0} />
              <button
                onClick={() => openNew("PRODUCTION")}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold bg-emerald-600 text-white hover:bg-emerald-700 rounded-lg shadow-sm transition-colors"
              >
                <Plus className="h-4 w-4" />
                Terima Barang
              </button>
            </div>
          </div>

          <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white shadow-sm">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-slate-200 bg-slate-50/80 text-xs uppercase text-slate-600 font-semibold tracking-wider">
                <tr>
                  <th className="px-4 py-3">No. Penerimaan</th>
                  <th className="px-4 py-3">Tipe Sumber</th>
                  <th className="px-4 py-3">Pengirim / Asal</th>
                  <th className="px-4 py-3">Referensi</th>
                  <th className="px-4 py-3 text-right">Diterima</th>
                  <th className="px-4 py-3 text-right">Ditolak</th>
                  <th className="px-4 py-3">Status Penerimaan</th>
                  <th className="px-4 py-3">Status Lot</th>
                  <th className="px-4 py-3">Tanggal</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 text-xs">
                {isLoading ? (
                  <tr>
                    <td colSpan={9} className="px-4 py-8 text-center text-slate-400">
                      Memuat data penerimaan...
                    </td>
                  </tr>
                ) : isError ? (
                  <tr>
                    <td colSpan={9} className="px-4 py-8 text-center text-rose-600">
                      Gagal memuat data: {errMsg(error)}
                    </td>
                  </tr>
                ) : receipts.length === 0 ? (
                  <tr>
                    <td colSpan={9} className="px-4 py-12 text-center text-slate-400">
                      <Layers className="mx-auto h-8 w-8 text-slate-300 mb-2" />
                      Belum ada catatan penerimaan barang di gudang ini.
                    </td>
                  </tr>
                ) : (
                  receipts.map((r) => {
                    const displayOrigin =
                      r.receipt_type === "TRANSFER"
                        ? r.from_warehouse_name || (r.from_warehouse_id ? whName(r.from_warehouse_id) : r.from_name)
                        : r.from_name || r.supplier_name || "-"
                    return (
                      <tr
                        key={r.id}
                        onClick={() => setView({ mode: "detail", id: r.id })}
                        className="cursor-pointer hover:bg-slate-50/70 transition-colors"
                      >
                        <td className="px-4 py-3 font-mono font-semibold text-slate-900">{r.receipt_number}</td>
                        <td className="px-4 py-3">
                          <TypeBadge type={r.receipt_type} />
                        </td>
                        <td className="px-4 py-3 font-medium text-slate-800">{displayOrigin}</td>
                        <td className="px-4 py-3 font-mono text-slate-500">{r.source_ref || r.supplier_ref || "-"}</td>
                        <td className="px-4 py-3 text-right font-bold text-emerald-700">
                          {fmtQty(r.total_accepted_qty)}
                        </td>
                        <td className="px-4 py-3 text-right">
                          {num(r.total_rejected_qty) > 0 ? (
                            <span className="text-rose-600 font-semibold">{fmtQty(r.total_rejected_qty)}</span>
                          ) : (
                            "-"
                          )}
                        </td>
                        <td className="px-4 py-3">
                          <StatusBadge status={r.status} />
                        </td>
                        <td className="px-4 py-3">
                          {r.on_hold_batch_count && r.on_hold_batch_count > 0 ? (
                            <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-bold bg-amber-50 text-amber-700 border border-amber-200">
                              <BadgeAlert className="h-3 w-3 text-amber-500" />
                              ON_HOLD ({r.on_hold_batch_count})
                            </span>
                          ) : r.status === "POSTED" ? (
                            <span className="inline-flex px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
                              RELEASED
                            </span>
                          ) : (
                            <span className="text-slate-400">-</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-slate-500">{fmtDate(r.created_at)}</td>
                      </tr>
                    )
                  })
                )}
              </tbody>
            </table>
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
            ]}
            filters={[]}
          />
        </div>
      )}

      {view.mode === "form" && (
        <InboundReceiptForm
          form={form}
          setForm={setForm}
          onCancel={() => setView({ mode: "list" })}
          onSaved={(id) => {
            setNotice({
              type: "success",
              text: "Draf penerimaan tersimpan. Stok akan masuk ke Staging Inbound setelah Anda menekan tombol Konfirmasi.",
            })
            setView({ mode: "detail", id })
            refetch()
          }}
        />
      )}

      {view.mode === "detail" && (
        <InboundReceiptDetail
          id={view.id}
          onBack={() => {
            setView({ mode: "list" })
            refetch()
          }}
          onGoToPutaway={onGoToPutaway}
          onNotice={setNotice}
        />
      )}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Inbound Receipt Form (with Batch & Expiry Date Inputs per KO-1 Invariant 1)
// ---------------------------------------------------------------------------

function InboundReceiptForm({
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

  const [search, setSearch] = useState("")
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const internalLocations = useMemo(
    () => locations.filter((l) => l.type === "INTERNAL" && l.warehouse_id === form.warehouse_id),
    [locations, form.warehouse_id]
  )

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
            batch_number: "", // optional, auto-generated if omitted
            expiry_date: "",
            accepted: String(qty),
            rejected: "0",
            reject_reason: "",
          },
        ],
      }
    })
  }

  const updateLine = (i: number, patch: Partial<Line>) =>
    setForm((f) => ({ ...f, lines: f.lines.map((l, idx) => (idx === i ? { ...l, ...patch } : l)) }))
  const removeLine = (i: number) =>
    setForm((f) => ({ ...f, lines: f.lines.filter((_, idx) => idx !== i) }))

  const handleSave = async () => {
    setError(null)
    if (!form.warehouse_id) {
      setError("Pilih gudang tujuan penerimaan.")
      return
    }
    if (!form.dest_location_id) {
      setError("Pilih rak/lokasi rencana penyimpanan.")
      return
    }
    const lineError = validateReceiptLines(form.lines)
    if (lineError) {
      setError(lineError)
      return
    }

    setSaving(true)
    try {
      const payload: StockReceiptInput = {
        receipt_type: form.receipt_type,
        warehouse_id: form.warehouse_id,
        dest_location_id: form.dest_location_id,
        from_name: form.from_name,
        from_warehouse_id: form.from_warehouse_id || undefined,
        source_ref: form.source_ref || undefined,
        notes: form.notes || undefined,
        items: form.lines.map((l) => ({
          product_id: l.product_id,
          batch_number: l.batch_number.trim() || undefined,
          expiry_date: l.expiry_date || undefined,
          accepted_qty: num(l.accepted),
          rejected_qty: num(l.rejected),
          reject_reason: l.reject_reason || undefined,
        })),
      }
      const res = await createMut.mutateAsync(payload)
      onSaved(res.receipt.id)
    } catch (e) {
      setError(errMsg(e))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6 rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
      <div className="flex items-center justify-between border-b border-slate-100 pb-4">
        <div>
          <h2 className="text-lg font-semibold text-slate-900">Form Catat Penerimaan Barang</h2>
          <p className="text-xs text-slate-500 mt-0.5">
            Setiap baris barang otomatis diberikan Batch / Lot untuk memastikan keterlacakan FEFO (KO-1 Invariant 1).
          </p>
        </div>
        <button onClick={onCancel} className="text-xs text-slate-500 hover:text-slate-800">
          Batal
        </button>
      </div>

      {error && (
        <div className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
          <AlertCircle className="h-4 w-4 shrink-0 text-rose-600 mt-0.5" />
          <span>{error}</span>
        </div>
      )}

      {/* Header Form */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs">
        <div>
          <label className="block font-medium text-slate-700 mb-1">Tipe Sumber Penerimaan</label>
          <select
            value={form.receipt_type}
            onChange={(e) => setForm({ ...form, receipt_type: e.target.value as StockReceiptType })}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg bg-white"
          >
            <option value="PRODUCTION">Hasil Produksi (Dapur/Pabrik)</option>
            <option value="TRANSFER">Transfer Cabang / Antar-Gudang</option>
            <option value="VENDOR">Pemasok / Vendor Luar</option>
          </select>
        </div>

        <div>
          <label className="block font-medium text-slate-700 mb-1">Nama Asal / Pengirim</label>
          <input
            type="text"
            value={form.from_name}
            onChange={(e) => setForm({ ...form, from_name: e.target.value })}
            placeholder="Contoh: Dapur Pusat, PT Sumber Makmur..."
            className="w-full px-3 py-2 border border-slate-300 rounded-lg"
          />
        </div>

        <div>
          <label className="block font-medium text-slate-700 mb-1">No. Surat Jalan / Referensi</label>
          <input
            type="text"
            value={form.source_ref}
            onChange={(e) => setForm({ ...form, source_ref: e.target.value })}
            placeholder="Contoh: SJ-2026-001"
            className="w-full px-3 py-2 border border-slate-300 rounded-lg font-mono"
          />
        </div>

        <div>
          <label className="block font-medium text-slate-700 mb-1">Gudang Penerima</label>
          <select
            value={form.warehouse_id}
            onChange={(e) => setForm({ ...form, warehouse_id: e.target.value, dest_location_id: "" })}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg bg-white"
          >
            <option value="">-- Pilih Gudang --</option>
            {warehouses.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block font-medium text-slate-700 mb-1">Rencana Rak Penyimpanan</label>
          <select
            value={form.dest_location_id}
            onChange={(e) => setForm({ ...form, dest_location_id: e.target.value })}
            disabled={!form.warehouse_id}
            className="w-full px-3 py-2 border border-slate-300 rounded-lg bg-white disabled:bg-slate-100"
          >
            <option value="">-- Pilih Rak Internal --</option>
            {internalLocations.map((l) => (
              <option key={l.id} value={l.id}>
                {l.code} {l.name ? `(${l.name})` : ""}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label className="block font-medium text-slate-700 mb-1">Catatan Tambahan</label>
          <input
            type="text"
            value={form.notes}
            onChange={(e) => setForm({ ...form, notes: e.target.value })}
            placeholder="Catatan kondisi kiriman..."
            className="w-full px-3 py-2 border border-slate-300 rounded-lg"
          />
        </div>
      </div>

      {/* Product Search & Line Items */}
      <div className="space-y-3 pt-3 border-t border-slate-100">
        <label className="block text-xs font-semibold text-slate-800 uppercase tracking-wider">
          Tambah Produk & Alokasi Batch
        </label>
        <div className="relative">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
          <input
            type="text"
            placeholder="Ketik nama produk atau scan barcode/SKU..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-4 py-2 border border-slate-300 rounded-lg text-xs"
          />
          {searchResults.length > 0 && (
            <div className="absolute left-0 right-0 top-full z-10 mt-1 max-h-48 overflow-y-auto rounded-lg border border-slate-200 bg-white shadow-lg">
              {searchResults.map((p) => (
                <button
                  key={p.id}
                  type="button"
                  onClick={() => {
                    addProduct(p)
                    setSearch("")
                  }}
                  className="w-full text-left px-3 py-2 text-xs hover:bg-slate-50 flex justify-between border-b border-slate-100 last:border-0"
                >
                  <span className="font-medium text-slate-800">{p.name}</span>
                  <span className="font-mono text-slate-400">{p.sku}</span>
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Lines Table */}
        <div className="rounded-xl border border-slate-200 overflow-hidden">
          <table className="w-full text-left text-xs">
            <thead className="bg-slate-50 text-slate-600 font-semibold border-b border-slate-200">
              <tr>
                <th className="px-3 py-2.5">Produk</th>
                <th className="px-3 py-2.5">Nomor Batch / Lot (Opsional)</th>
                <th className="px-3 py-2.5">Tgl Kedaluwarsa</th>
                <th className="px-3 py-2.5 text-right w-24">Qty Diterima</th>
                <th className="px-3 py-2.5 text-right w-24">Qty Ditolak</th>
                <th className="px-3 py-2.5">Alasan Tolak</th>
                <th className="px-3 py-2.5 text-center w-12">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {form.lines.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-3 py-6 text-center text-slate-400">
                    Belum ada produk yang ditambahkan. Gunakan pencarian di atas untuk menambahkan.
                  </td>
                </tr>
              ) : (
                form.lines.map((line, idx) => (
                  <tr key={line.product_id} className="hover:bg-slate-50/50">
                    <td className="px-3 py-2">
                      <div className="font-medium text-slate-800">{line.product_name}</div>
                      <div className="font-mono text-slate-400 text-[10px]">{line.product_sku}</div>
                    </td>
                    <td className="px-3 py-2">
                      <input
                        type="text"
                        placeholder="Otomatis jika kosong"
                        value={line.batch_number}
                        onChange={(e) => updateLine(idx, { batch_number: e.target.value })}
                        className="w-full px-2 py-1 border border-slate-300 rounded font-mono text-xs"
                      />
                    </td>
                    <td className="px-3 py-2">
                      <input
                        type="date"
                        value={line.expiry_date}
                        onChange={(e) => updateLine(idx, { expiry_date: e.target.value })}
                        aria-invalid={isPastDate(line.expiry_date)}
                        className={`w-full px-2 py-1 border rounded text-xs ${
                          isPastDate(line.expiry_date) ? "border-rose-400 bg-rose-50" : "border-slate-300"
                        }`}
                      />
                      {isPastDate(line.expiry_date) && (
                        <div className="mt-0.5 text-[10px] text-rose-600">Sudah kedaluwarsa — catat sebagai ditolak</div>
                      )}
                    </td>
                    <td className="px-3 py-2 text-right">
                      <input
                        type="number"
                        min="0"
                        step="1"
                        inputMode="numeric"
                        aria-label={`Qty diterima ${line.product_name}`}
                        value={line.accepted}
                        onChange={(e) => updateLine(idx, { accepted: e.target.value })}
                        className="w-full px-2 py-1 border border-slate-300 rounded text-right font-semibold text-xs"
                      />
                    </td>
                    <td className="px-3 py-2 text-right">
                      <input
                        type="number"
                        min="0"
                        step="1"
                        inputMode="numeric"
                        aria-label={`Qty ditolak ${line.product_name}`}
                        value={line.rejected}
                        onChange={(e) => updateLine(idx, { rejected: e.target.value })}
                        className="w-full px-2 py-1 border border-slate-300 rounded text-right text-xs"
                      />
                    </td>
                    <td className="px-3 py-2">
                      {num(line.rejected) > 0 ? (
                        <select
                          aria-label={`Alasan tolak ${line.product_name}`}
                          aria-required="true"
                          value={line.reject_reason}
                          onChange={(e) => updateLine(idx, { reject_reason: e.target.value })}
                          className="w-full px-2 py-1 border border-rose-300 rounded bg-rose-50/40 text-xs"
                        >
                          <option value="">Pilih alasan...</option>
                          {REJECT_REASONS.map((r) => (
                            <option key={r} value={r}>
                              {r}
                            </option>
                          ))}
                        </select>
                      ) : (
                        <span className="text-slate-300">-</span>
                      )}
                    </td>
                    <td className="px-3 py-2 text-center">
                      <button
                        type="button"
                        onClick={() => removeLine(idx)}
                        className="text-rose-500 hover:text-rose-700 p-1"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex justify-end gap-2 pt-4 border-t border-slate-100">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg"
        >
          Batal
        </button>
        <button
          type="button"
          onClick={handleSave}
          disabled={saving || !!validateReceiptLines(form.lines)}
          title={validateReceiptLines(form.lines) ?? undefined}
          className="px-4 py-2 text-xs font-semibold text-white bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg shadow-sm"
        >
          {saving ? "Menyimpan Draf..." : "Simpan Draf Penerimaan"}
        </button>
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Inbound Receipt Detail & Release Approval / Cancel
// ---------------------------------------------------------------------------

function InboundReceiptDetail({
  id,
  onBack,
  onGoToPutaway,
  onNotice,
}: {
  id: string
  onBack: () => void
  onGoToPutaway: () => void
  onNotice: (n: { type: "success" | "error"; text: string } | null) => void
}) {
  const { data, isLoading, isError, error, refetch } = useStockReceipt(id)
  const postMut = usePostStockReceipt()
  const cancelMut = useCancelStockReceipt()
  const releaseMut = useReleaseStockReceipt()

  const [cancelOpen, setCancelOpen] = useState(false)
  const [cancelReason, setCancelReason] = useState("")
  const [isProcessing, setIsProcessing] = useState(false)

  if (isLoading) return <p className="text-xs text-slate-400">Memuat detail penerimaan...</p>
  if (isError || !data) {
    return (
      <div className="space-y-3">
        <button onClick={onBack} className="inline-flex items-center gap-1 text-xs text-slate-600">
          <ArrowLeft className="h-3.5 w-3.5" /> Kembali
        </button>
        <p className="text-xs text-rose-600">Gagal memuat: {errMsg(error)}</p>
      </div>
    )
  }

  const r = data.receipt
  const items = data.items ?? []

  const handlePost = async () => {
    setIsProcessing(true)
    try {
      await postMut.mutateAsync(r.id)
      onNotice({
        type: "success",
        text: `Penerimaan ${r.receipt_number} berhasil dikonfirmasi ke Staging Inbound! Silakan lanjutkan proses Putaway ke rak.`,
      })
      refetch()
    } catch (e) {
      onNotice({ type: "error", text: errMsg(e) })
    } finally {
      setIsProcessing(false)
    }
  }

  const handleRelease = async () => {
    setIsProcessing(true)
    try {
      await releaseMut.mutateAsync(r.id)
      onNotice({
        type: "success",
        text: `Seluruh lot tertahan (ON_HOLD) pada penerimaan ${r.receipt_number} berhasil disetujui rilis (RELEASED) dan siap dijual!`,
      })
      refetch()
    } catch (e) {
      onNotice({ type: "error", text: errMsg(e) })
    } finally {
      setIsProcessing(false)
    }
  }

  const handleCancel = async () => {
    if (!cancelReason.trim()) return
    setIsProcessing(true)
    try {
      await cancelMut.mutateAsync({ id: r.id, reason: cancelReason.trim() })
      setCancelOpen(false)
      onNotice({
        type: "success",
        text:
          r.status === "POSTED"
            ? "Penerimaan dibatalkan dan kuantitas stok dikembalikan ke posisi semula."
            : "Draf penerimaan dibatalkan.",
      })
      refetch()
    } catch (e) {
      onNotice({ type: "error", text: errMsg(e) })
    } finally {
      setIsProcessing(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 pb-4">
        <div className="flex items-center gap-3">
          <button onClick={onBack} className="p-1 rounded hover:bg-slate-100 text-slate-600">
            <ArrowLeft className="h-4 w-4" />
          </button>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-bold font-mono text-slate-900">{r.receipt_number}</h2>
              <StatusBadge status={r.status} />
              <TypeBadge type={r.receipt_type} />
            </div>
            <p className="text-xs text-slate-500 mt-0.5">Dibuat pada {fmtDate(r.created_at)}</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {r.status === "DRAFT" && (
            <button
              onClick={handlePost}
              disabled={isProcessing}
              className="px-4 py-2 text-xs font-semibold bg-emerald-600 text-white hover:bg-emerald-700 disabled:opacity-50 rounded-lg shadow-sm"
            >
              {isProcessing ? "Memproses..." : "Konfirmasi Masuk Staging"}
            </button>
          )}

          {r.status === "POSTED" && r.on_hold_batch_count && r.on_hold_batch_count > 0 ? (
            <button
              onClick={handleRelease}
              disabled={isProcessing}
              className="px-4 py-2 text-xs font-semibold bg-amber-600 text-white hover:bg-amber-700 disabled:opacity-50 rounded-lg shadow-sm"
            >
              {isProcessing ? "Merilis..." : "Setujui Rilis (Release Approval PDF-06)"}
            </button>
          ) : null}

          {r.status === "POSTED" && (
            <button
              onClick={onGoToPutaway}
              className="px-3.5 py-2 text-xs font-semibold bg-primary text-primary-foreground hover:bg-primary/90 rounded-lg shadow-sm inline-flex items-center gap-1"
            >
              <ArrowDownToLine className="h-3.5 w-3.5" />
              Lanjut Putaway
            </button>
          )}

          {r.status !== "CANCELLED" && (
            <button
              onClick={() => setCancelOpen(true)}
              className="px-3 py-2 text-xs font-medium text-rose-700 hover:bg-rose-50 border border-rose-200 rounded-lg"
            >
              Batalkan Penerimaan
            </button>
          )}
        </div>
      </div>

      {/* On-Hold Notice */}
      {r.status === "POSTED" && r.on_hold_batch_count && r.on_hold_batch_count > 0 ? (
        <div className="flex items-start gap-2.5 rounded-xl border border-amber-300 bg-amber-50 p-4 text-xs text-amber-900">
          <BadgeAlert className="h-5 w-5 text-amber-600 shrink-0 mt-0.5" />
          <div className="space-y-1">
            <div className="font-semibold">Lot Berstatus Tertahan (ON_HOLD)</div>
            <p className="text-amber-800 leading-relaxed">
              Penerimaan ini menghasilkan {r.on_hold_batch_count} lot berstatus <strong>ON_HOLD</strong> karena
              kebijakan release approval aktif. Barang sudah berada di Staging Inbound tetapi tidak dapat dijual hingga
              Supervisor menekan tombol <strong>Setujui Rilis</strong> di atas.
            </p>
          </div>
        </div>
      ) : null}

      {/* Items Table */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
        <div className="px-4 py-3 bg-slate-50 border-b border-slate-200 font-semibold text-xs text-slate-700 uppercase tracking-wider">
          Rincian Barang & Lot
        </div>
        <table className="w-full text-left text-xs">
          <thead className="bg-slate-50 text-slate-600 border-b border-slate-200">
            <tr>
              <th className="px-4 py-2.5">Produk</th>
              <th className="px-4 py-2.5">Batch / Lot</th>
              <th className="px-4 py-2.5">Kedaluwarsa</th>
              <th className="px-4 py-2.5 text-right">Diterima</th>
              <th className="px-4 py-2.5 text-right">Ditolak</th>
              <th className="px-4 py-2.5">Alasan Tolak</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {items.map((it) => (
              <tr key={it.id}>
                <td className="px-4 py-2.5">
                  <div className="font-medium text-slate-900">{it.product_name}</div>
                  <div className="font-mono text-slate-400 text-[10px]">{it.product_sku}</div>
                </td>
                <td className="px-4 py-2.5 font-mono font-medium text-slate-800">
                  {it.batch_number || "Otomatis di Ledger"}
                </td>
                <td className="px-4 py-2.5 text-slate-600">{it.expiry_date ? fmtDate(it.expiry_date) : "-"}</td>
                <td className="px-4 py-2.5 text-right font-bold text-emerald-700">{fmtQty(it.accepted_qty)}</td>
                <td className="px-4 py-2.5 text-right">
                  {num(it.rejected_qty) > 0 ? (
                    <span className="text-rose-600 font-semibold">{fmtQty(it.rejected_qty)}</span>
                  ) : (
                    "-"
                  )}
                </td>
                <td className="px-4 py-2.5 text-slate-500">{it.reject_reason || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Cancel Modal */}
      {cancelOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="w-full max-w-md rounded-xl bg-white p-6 shadow-xl border border-slate-200 space-y-4">
            <h3 className="text-base font-semibold text-slate-900">Konfirmasi Pembatalan Penerimaan</h3>
            <p className="text-xs text-slate-500">
              {r.status === "POSTED"
                ? "Membatalkan penerimaan yang sudah diposting akan membalikkan mutasi buku besar jika stok belum terpakai."
                : "Draf penerimaan akan dibatalkan."}
            </p>
            <div>
              <label className="block text-xs font-medium text-slate-700 mb-1">
                Alasan Pembatalan <span className="text-rose-500">*</span>
              </label>
              <input
                type="text"
                placeholder="Contoh: Salah kirim, supplier membatalkan PO..."
                value={cancelReason}
                onChange={(e) => setCancelReason(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg text-xs"
              />
            </div>
            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setCancelOpen(false)}
                className="px-3 py-1.5 text-xs font-medium text-slate-600 bg-slate-100 rounded-md"
              >
                Tutup
              </button>
              <button
                onClick={handleCancel}
                disabled={!cancelReason.trim() || isProcessing}
                className="px-3 py-1.5 text-xs font-medium bg-rose-600 text-white rounded-md disabled:opacity-50"
              >
                {isProcessing ? "Membatalkan..." : "Ya, Batalkan"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

