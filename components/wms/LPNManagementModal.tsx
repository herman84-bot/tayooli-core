"use client"

// FE-10: Modal for managing Pallet LPN (License Plate Number) containers.
// Enables containerizing received batch items, inspecting pallet contents,
// and printing ISO-28219 thermal barcode stickers.

import React, { useState, useMemo } from "react"
import {
  X,
  Plus,
  Box,
  Layers,
  Printer,
  Search,
  ArrowLeft,
  CheckCircle2,
  AlertCircle,
  Package,
  Weight,
  MapPin,
  ChevronRight,
  Sparkles,
} from "lucide-react"
import {
  useStockLPNs,
  useStockLPNDetail,
  useCreateLPN,
  useAddLPNItem,
} from "@/hooks/useWMSDocksAndLPNs"
import { useWarehouseLocations, usePutawayPending } from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import { api, type StockLPN, type StockLPNDetail, type PalletType, type LPNStatus } from "@/lib/api"
import { PrintLPNLabel } from "@/components/wms/PrintLPNLabel"

export interface LPNManagementModalProps {
  isOpen: boolean
  onClose: () => void
  warehouseId: string | null
}

const STATUS_CONFIG: Record<
  LPNStatus,
  { label: string; cls: string }
> = {
  STAGED: { label: "Staging (STAGED)", cls: "bg-amber-50 text-amber-700 border-amber-200" },
  STORED: { label: "Tersimpan (STORED)", cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  PICKED: { label: "Diambil (PICKED)", cls: "bg-blue-50 text-blue-700 border-blue-200" },
  SHIPPED: { label: "Terkirim (SHIPPED)", cls: "bg-purple-50 text-purple-700 border-purple-200" },
  DECOMMISSIONED: { label: "Nonaktif", cls: "bg-slate-100 text-slate-600 border-slate-200" },
}

const PALLET_TYPE_CONFIG: Record<
  PalletType,
  { label: string; desc: string }
> = {
  WOODEN: { label: "Kayu (WOODEN)", desc: "Standar ISPM-15" },
  PLASTIC: { label: "Plastik (PLASTIC)", desc: "HDPE Food Grade" },
  METAL: { label: "Logam (METAL)", desc: "Baja Heavy Duty" },
  CAGE: { label: "Keranjang (CAGE)", desc: "Stillage Jaring" },
}

function parseNum(val: unknown): number {
  if (typeof val === "number") return val
  if (typeof val === "string") {
    const parsed = parseFloat(val)
    return isNaN(parsed) ? 0 : parsed
  }
  return 0
}

export function LPNManagementModal({ isOpen, onClose, warehouseId }: LPNManagementModalProps) {
  // Navigation / views inside modal: list | detail | create | add_item
  const [activeView, setActiveView] = useState<"list" | "detail" | "create" | "add_item">("list")
  const [selectedLpnId, setSelectedLpnId] = useState<string | null>(null)
  const [statusFilter, setStatusFilter] = useState<string>("ALL")
  const [searchQuery, setSearchQuery] = useState<string>("")
  const [printDetail, setPrintDetail] = useState<StockLPNDetail | null>(null)
  const [notice, setNotice] = useState<{ type: "success" | "error"; text: string } | null>(null)

  // Create LPN Form State
  const [newLocationId, setNewLocationId] = useState<string>("")
  const [newPalletType, setNewPalletType] = useState<PalletType>("WOODEN")
  const [newMaxWeight, setNewMaxWeight] = useState<string>("1000")
  const [newLpnCode, setNewLpnCode] = useState<string>("")
  const [newNotes, setNewNotes] = useState<string>("")
  const [isCreating, setIsCreating] = useState<boolean>(false)

  // Add Item Form State
  const [itemProductId, setItemProductId] = useState<string>("")
  const [itemBatchId, setItemBatchId] = useState<string>("")
  const [itemQuantity, setItemQuantity] = useState<string>("")
  const [isAddingItem, setIsAddingItem] = useState<boolean>(false)

  // Queries
  const { data: lpns = [], isLoading: isLoadingLPNs, refetch: refetchLPNs } = useStockLPNs(
    warehouseId,
    statusFilter === "ALL" ? undefined : statusFilter
  )
  const { data: selectedLpnDetail, isLoading: isLoadingDetail, refetch: refetchDetail } =
    useStockLPNDetail(selectedLpnId)
  const { data: locations = [] } = useWarehouseLocations(warehouseId)
  const { data: products = [] } = useProducts()
  const { data: putawayPending = [] } = usePutawayPending(warehouseId)

  // Mutations
  const createLPNMut = useCreateLPN()
  const addItemMut = useAddLPNItem()

  // Filtered LPNs by search
  const filteredLPNs = useMemo(() => {
    return lpns.filter((l) => {
      if (!searchQuery.trim()) return true
      const q = searchQuery.toLowerCase()
      return (
        l.lpn_code.toLowerCase().includes(q) ||
        (l.location_code && l.location_code.toLowerCase().includes(q)) ||
        (l.location_name && l.location_name.toLowerCase().includes(q))
      )
    })
  }, [lpns, searchQuery])

  if (!isOpen) return null

  // Open "Buat Palet Baru" form
  const handleOpenCreate = () => {
    setNewLocationId(locations[0]?.id || "")
    setNewPalletType("WOODEN")
    setNewMaxWeight("1000")
    setNewLpnCode("")
    setNewNotes("")
    setNotice(null)
    setActiveView("create")
  }

  // Submit Create LPN
  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!warehouseId) {
      setNotice({ type: "error", text: "Gudang belum dipilih." })
      return
    }
    if (!newLocationId) {
      setNotice({ type: "error", text: "Pilih lokasi rak / staging awal untuk palet." })
      return
    }

    setIsCreating(true)
    setNotice(null)
    try {
      const res = await createLPNMut.mutateAsync({
        warehouse_id: warehouseId,
        location_id: newLocationId,
        pallet_type: newPalletType,
        max_weight_kg: parseNum(newMaxWeight) || 1000,
        lpn_code: newLpnCode.trim() || undefined,
        notes: newNotes.trim() || undefined,
      })
      const createdLpn = res?.data
      setNotice({
        type: "success",
        text: `Palet LPN ${createdLpn?.lpn_code || ""} berhasil dibuat!`,
      })
      refetchLPNs()
      if (createdLpn?.id) {
        setSelectedLpnId(createdLpn.id)
        setActiveView("detail")
      } else {
        setActiveView("list")
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Gagal membuat palet LPN."
      setNotice({ type: "error", text: msg })
    } finally {
      setIsCreating(false)
    }
  }

  // Open Add Item Form
  const handleOpenAddItem = (lpnId: string) => {
    setSelectedLpnId(lpnId)
    setItemProductId("")
    setItemBatchId("")
    setItemQuantity("")
    setNotice(null)
    setActiveView("add_item")
  }

  // Submit Add Item
  const handleAddItemSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedLpnId) return
    const qty = parseNum(itemQuantity)
    if (!itemProductId) {
      setNotice({ type: "error", text: "Pilih produk terlebih dahulu." })
      return
    }
    if (!itemBatchId) {
      setNotice({ type: "error", text: "ID Batch wajib diisi." })
      return
    }
    if (qty <= 0) {
      setNotice({ type: "error", text: "Kuantitas barang harus lebih dari 0." })
      return
    }

    setIsAddingItem(true)
    setNotice(null)
    try {
      await addItemMut.mutateAsync({
        lpnId: selectedLpnId,
        data: {
          product_id: itemProductId,
          batch_id: itemBatchId,
          quantity: qty,
        },
      })
      setNotice({
        type: "success",
        text: "Item batch berhasil ditambahkan ke dalam kontainer palet!",
      })
      refetchDetail()
      refetchLPNs()
      setActiveView("detail")
    } catch (err) {
      const msg = err instanceof Error ? err.message : "Gagal menambahkan item ke palet."
      setNotice({ type: "error", text: msg })
    } finally {
      setIsAddingItem(false)
    }
  }

  // Trigger Print Label
  const handlePrintLabel = async (lpn: StockLPN) => {
    try {
      const res = await api.wms.lpns.get(lpn.id)
      if (res.data) {
        setPrintDetail(res.data)
      } else {
        setPrintDetail({ lpn, items: [] })
      }
    } catch (err: unknown) {
      console.error("Gagal memuat detail LPN untuk cetak label:", err)
      setPrintDetail({ lpn, items: [] })
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-3 md:p-6 overflow-y-auto">
      <div className="bg-white rounded-2xl shadow-2xl max-w-4xl w-full flex flex-col max-h-[90vh] overflow-hidden border border-slate-200">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200 bg-slate-50/60">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-emerald-100 text-emerald-700 rounded-xl">
              <Box className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
                Manajemen Palet & Kontainer LPN
                <span className="text-[11px] font-mono px-2 py-0.5 bg-slate-200 text-slate-700 rounded-md font-normal">
                  License Plate Number
                </span>
              </h2>
              <p className="text-xs text-slate-500">
                Wadah palet penampung lot barang di staging & rak gudang (ISO-28219)
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 text-slate-400 hover:text-slate-600 rounded-lg hover:bg-slate-100 transition"
            aria-label="Tutup"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Notice Alert */}
        {notice && (
          <div
            role="status"
            className={`mx-6 mt-4 flex items-start gap-2.5 rounded-lg border px-4 py-3 text-xs ${
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
            <span className="flex-1 font-medium">{notice.text}</span>
            <button onClick={() => setNotice(null)} aria-label="Tutup pemberitahuan">
              <X className="h-4 w-4" />
            </button>
          </div>
        )}

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto p-6">
          {/* VIEW: LIST OF LPNs */}
          {activeView === "list" && (
            <div className="space-y-4">
              {/* Controls bar */}
              <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3">
                {/* Search */}
                <div className="relative flex-1 max-w-sm">
                  <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
                  <input
                    type="text"
                    placeholder="Cari kode LPN atau kode lokasi..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="w-full pl-9 pr-3 py-1.5 text-xs border border-slate-300 rounded-lg bg-slate-50 focus:bg-white focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>

                {/* Status Tabs & Action */}
                <div className="flex items-center gap-2">
                  <div className="inline-flex rounded-lg border border-slate-200 bg-slate-50 p-1 text-xs">
                    {(["ALL", "STAGED", "STORED", "PICKED"] as const).map((st) => (
                      <button
                        key={st}
                        onClick={() => setStatusFilter(st)}
                        className={`px-2.5 py-1 rounded font-medium transition-colors ${
                          statusFilter === st
                            ? "bg-white text-slate-900 shadow-sm"
                            : "text-slate-500 hover:text-slate-800"
                        }`}
                      >
                        {st === "ALL" ? "Semua" : st}
                      </button>
                    ))}
                  </div>

                  <button
                    onClick={handleOpenCreate}
                    className="inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-semibold bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg shadow-sm transition"
                  >
                    <Plus className="w-4 h-4" />
                    Buat Palet LPN Baru
                  </button>
                </div>
              </div>

              {/* LPN List Table */}
              <div className="rounded-xl border border-slate-200 overflow-hidden bg-white shadow-sm">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 border-b border-slate-200 text-slate-600 font-semibold uppercase tracking-wider text-[11px]">
                    <tr>
                      <th className="px-4 py-3">Kode LPN</th>
                      <th className="px-4 py-3">Tipe Palet</th>
                      <th className="px-4 py-3">Status</th>
                      <th className="px-4 py-3">Lokasi Saat Ini</th>
                      <th className="px-4 py-3 text-right">Berat / Kapasitas</th>
                      <th className="px-4 py-3 text-right">Aksi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {isLoadingLPNs ? (
                      <tr>
                        <td colSpan={6} className="px-4 py-8 text-center text-slate-400">
                          Memuat data palet LPN...
                        </td>
                      </tr>
                    ) : filteredLPNs.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="px-4 py-12 text-center text-slate-400">
                          <Layers className="mx-auto h-8 w-8 text-slate-300 mb-2" />
                          Belum ada wadah palet LPN di gudang ini.
                          <div className="mt-2">
                            <button
                              onClick={handleOpenCreate}
                              className="text-xs text-emerald-600 hover:text-emerald-700 font-semibold inline-flex items-center gap-1"
                            >
                              <Plus className="w-3.5 h-3.5" /> Buat Palet Pertama
                            </button>
                          </div>
                        </td>
                      </tr>
                    ) : (
                      filteredLPNs.map((lpn) => {
                        const statusBadge = STATUS_CONFIG[lpn.status] ?? STATUS_CONFIG.STAGED
                        const palletTypeInfo = PALLET_TYPE_CONFIG[lpn.pallet_type] ?? PALLET_TYPE_CONFIG.WOODEN
                        const totWeight = parseNum(lpn.total_weight_kg)
                        const maxWeight = parseNum(lpn.max_weight_kg)

                        return (
                          <tr key={lpn.id} className="hover:bg-slate-50/70 transition">
                            <td className="px-4 py-3">
                              <span className="font-mono font-bold text-slate-900 block">{lpn.lpn_code}</span>
                              {lpn.notes && (
                                <span className="text-[10px] text-slate-400 truncate max-w-[180px] block">
                                  {lpn.notes}
                                </span>
                              )}
                            </td>
                            <td className="px-4 py-3">
                              <span className="font-medium text-slate-700 block">{palletTypeInfo.label}</span>
                              <span className="text-[10px] text-slate-400">{palletTypeInfo.desc}</span>
                            </td>
                            <td className="px-4 py-3">
                              <span
                                className={`inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-semibold border ${statusBadge.cls}`}
                              >
                                {statusBadge.label}
                              </span>
                            </td>
                            <td className="px-4 py-3">
                              <div className="flex items-center gap-1 font-mono text-slate-800 font-semibold">
                                <MapPin className="w-3 h-3 text-slate-400" />
                                {lpn.location_code || lpn.location_name || "-"}
                              </div>
                            </td>
                            <td className="px-4 py-3 text-right">
                              <span className="font-mono font-bold text-slate-900">{totWeight}</span>
                              <span className="text-slate-400 font-mono text-[11px]"> / {maxWeight} kg</span>
                            </td>
                            <td className="px-4 py-3 text-right">
                              <div className="flex items-center justify-end gap-1.5">
                                <button
                                  onClick={() => {
                                    setSelectedLpnId(lpn.id)
                                    setActiveView("detail")
                                  }}
                                  className="px-2.5 py-1 text-[11px] font-semibold text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-md transition"
                                >
                                  Lihat Isi
                                </button>
                                <button
                                  onClick={() => handleOpenAddItem(lpn.id)}
                                  className="px-2.5 py-1 text-[11px] font-semibold text-emerald-700 bg-emerald-50 hover:bg-emerald-100 border border-emerald-200 rounded-md transition"
                                >
                                  Tambah Item
                                </button>
                                <button
                                  onClick={() => handlePrintLabel(lpn)}
                                  className="p-1 text-slate-500 hover:text-slate-800 hover:bg-slate-100 rounded-md transition"
                                  title="Cetak Label Palet"
                                  aria-label="Cetak Label Palet"
                                >
                                  <Printer className="w-4 h-4" />
                                </button>
                              </div>
                            </td>
                          </tr>
                        )
                      })
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* VIEW: LPN DETAIL ("Lihat Isi") */}
          {activeView === "detail" && (
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b pb-3">
                <button
                  onClick={() => {
                    setActiveView("list")
                    refetchLPNs()
                  }}
                  className="inline-flex items-center gap-1 text-xs text-slate-600 hover:text-slate-900 font-medium"
                >
                  <ArrowLeft className="w-4 h-4" /> Kembali ke Daftar Palet
                </button>
                {selectedLpnDetail && (
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleOpenAddItem(selectedLpnDetail.lpn.id)}
                      className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-semibold bg-emerald-600 text-white hover:bg-emerald-700 rounded-lg shadow-sm transition"
                    >
                      <Plus className="w-3.5 h-3.5" /> Tambah Item ke Palet
                    </button>
                    <button
                      onClick={() => handlePrintLabel(selectedLpnDetail.lpn)}
                      className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-semibold bg-slate-800 text-white hover:bg-slate-900 rounded-lg shadow-sm transition"
                    >
                      <Printer className="w-3.5 h-3.5" /> Cetak Label Palet
                    </button>
                  </div>
                )}
              </div>

              {isLoadingDetail ? (
                <div className="py-12 text-center text-slate-400 text-xs">Memuat rincian palet LPN...</div>
              ) : !selectedLpnDetail ? (
                <div className="py-12 text-center text-rose-500 text-xs">Detail palet tidak ditemukan.</div>
              ) : (
                <div className="space-y-4">
                  {/* LPN Overview Header */}
                  <div className="bg-slate-50 border border-slate-200 rounded-xl p-4 grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
                    <div>
                      <span className="text-slate-400 block text-[10px] uppercase font-bold">Kode Kontainer LPN</span>
                      <span className="font-mono font-black text-sm text-slate-900">
                        {selectedLpnDetail.lpn.lpn_code}
                      </span>
                    </div>
                    <div>
                      <span className="text-slate-400 block text-[10px] uppercase font-bold">Tipe & Status</span>
                      <span className="font-semibold text-slate-800 block">
                        {selectedLpnDetail.lpn.pallet_type}
                      </span>
                      <span className="text-[10px] font-bold text-emerald-700">
                        {selectedLpnDetail.lpn.status}
                      </span>
                    </div>
                    <div>
                      <span className="text-slate-400 block text-[10px] uppercase font-bold">Lokasi Rak</span>
                      <span className="font-mono font-bold text-slate-800">
                        {selectedLpnDetail.lpn.location_code || selectedLpnDetail.lpn.location_name || "-"}
                      </span>
                    </div>
                    <div>
                      <span className="text-slate-400 block text-[10px] uppercase font-bold">Total Muatan</span>
                      <span className="font-mono font-bold text-slate-900">
                        {selectedLpnDetail.lpn.total_weight_kg} / {selectedLpnDetail.lpn.max_weight_kg} kg
                      </span>
                    </div>
                  </div>

                  {/* Items in LPN Table */}
                  <div className="rounded-xl border border-slate-200 overflow-hidden bg-white">
                    <div className="px-4 py-2.5 bg-slate-50 border-b border-slate-200 font-semibold text-xs text-slate-700 flex justify-between">
                      <span>Daftar Item Batch Dalam Palet ({selectedLpnDetail.items.length} Batch)</span>
                      <span className="font-mono">
                        Total:{" "}
                        {selectedLpnDetail.items.reduce((s, it) => s + parseNum(it.quantity), 0)}{" "}
                        unit
                      </span>
                    </div>
                    <table className="w-full text-left text-xs">
                      <thead className="bg-slate-50/50 border-b border-slate-200 text-slate-500 font-bold uppercase text-[10px]">
                        <tr>
                          <th className="px-4 py-2.5">Produk</th>
                          <th className="px-4 py-2.5">Batch / Lot</th>
                          <th className="px-4 py-2.5">Tgl Kedaluwarsa</th>
                          <th className="px-4 py-2.5 text-right">Kuantitas</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100">
                        {selectedLpnDetail.items.length === 0 ? (
                          <tr>
                            <td colSpan={4} className="px-4 py-8 text-center text-slate-400">
                              Palet ini masih kosong. Klik &quot;Tambah Item ke Palet&quot; untuk mengalokasikan batch barang masuk.
                            </td>
                          </tr>
                        ) : (
                          selectedLpnDetail.items.map((item) => (
                            <tr key={item.id} className="hover:bg-slate-50/50">
                              <td className="px-4 py-3">
                                <span className="font-medium text-slate-900 block">
                                  {item.product_name || "Produk"}
                                </span>
                                <span className="font-mono text-slate-400 text-[10px]">
                                  {item.product_sku || "-"}
                                </span>
                              </td>
                              <td className="px-4 py-3 font-mono font-bold text-slate-800">
                                {item.batch_number || item.batch_id}
                              </td>
                              <td className="px-4 py-3 text-slate-600 font-mono">
                                {item.expiry_date ? item.expiry_date.slice(0, 10) : "-"}
                              </td>
                              <td className="px-4 py-3 text-right font-mono font-bold text-emerald-700">
                                {item.quantity}
                              </td>
                            </tr>
                          ))
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* VIEW: CREATE NEW PALLET LPN */}
          {activeView === "create" && (
            <form onSubmit={handleCreateSubmit} className="space-y-4 max-w-xl mx-auto">
              <div className="flex items-center justify-between border-b pb-3 mb-2">
                <button
                  type="button"
                  onClick={() => setActiveView("list")}
                  className="inline-flex items-center gap-1 text-xs text-slate-600 hover:text-slate-900 font-medium"
                >
                  <ArrowLeft className="w-4 h-4" /> Batal
                </button>
                <h3 className="text-sm font-bold text-slate-900">Buat Palet LPN Baru</h3>
              </div>

              <div>
                <label htmlFor="newLocationSelect" className="block text-xs font-semibold text-slate-700 mb-1">
                  Lokasi Staging / Rak Awal <span className="text-rose-500">*</span>
                </label>
                <select
                  id="newLocationSelect"
                  aria-label="Lokasi Staging / Rak Awal"
                  value={newLocationId}
                  onChange={(e) => setNewLocationId(e.target.value)}
                  className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg bg-slate-50 focus:bg-white"
                  required
                >
                  <option value="">-- Pilih Lokasi Staging atau Rak --</option>
                  {locations.map((loc) => (
                    <option key={loc.id} value={loc.id}>
                      {loc.code} - {loc.name} ({loc.type})
                    </option>
                  ))}
                </select>
                <p className="text-[11px] text-slate-400 mt-0.5">
                  Biasanya ditempatkan di Staging Inbound sebelum proses putaway ke rak.
                </p>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                <div>
                  <label htmlFor="newPalletTypeSelect" className="block text-xs font-semibold text-slate-700 mb-1">Tipe Palet</label>
                  <select
                    id="newPalletTypeSelect"
                    aria-label="Tipe Palet"
                    value={newPalletType}
                    onChange={(e) => setNewPalletType(e.target.value as PalletType)}
                    className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg bg-slate-50 focus:bg-white"
                  >
                    <option value="WOODEN">Kayu (WOODEN)</option>
                    <option value="PLASTIC">Plastik (PLASTIC)</option>
                    <option value="METAL">Logam / Besi (METAL)</option>
                    <option value="CAGE">Keranjang (CAGE)</option>
                  </select>
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Kapasitas Beban Maks (kg)
                  </label>
                  <input
                    type="number"
                    min="1"
                    step="0.01"
                    value={newMaxWeight}
                    onChange={(e) => setNewMaxWeight(e.target.value)}
                    className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg font-mono"
                    required
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Kode LPN Kustom (Opsional)
                </label>
                <input
                  type="text"
                  placeholder="Kosongkan untuk nomor otomatis (LPN-YYYYMMDD-XXXX)"
                  value={newLpnCode}
                  onChange={(e) => setNewLpnCode(e.target.value)}
                  className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg font-mono"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Catatan / Keterangan (Opsional)
                </label>
                <textarea
                  rows={2}
                  placeholder="Catatan kondisi palet atau instruksi khusus..."
                  value={newNotes}
                  onChange={(e) => setNewNotes(e.target.value)}
                  className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg"
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-3 border-t">
                <button
                  type="button"
                  onClick={() => setActiveView("list")}
                  className="px-4 py-2 text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={isCreating}
                  className="px-4 py-2 text-xs font-semibold text-white bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 rounded-lg shadow-sm"
                >
                  {isCreating ? "Menyimpan Palet..." : "Buat Palet LPN"}
                </button>
              </div>
            </form>
          )}

          {/* VIEW: ADD ITEM TO LPN */}
          {activeView === "add_item" && (
            <form onSubmit={handleAddItemSubmit} className="space-y-4 max-w-xl mx-auto">
              <div className="flex items-center justify-between border-b pb-3 mb-2">
                <button
                  type="button"
                  onClick={() => {
                    setActiveView(selectedLpnDetail ? "detail" : "list")
                  }}
                  className="inline-flex items-center gap-1 text-xs text-slate-600 hover:text-slate-900 font-medium"
                >
                  <ArrowLeft className="w-4 h-4" /> Batal
                </button>
                <h3 className="text-sm font-bold text-slate-900">
                  Tambah Item Batch ke Palet{" "}
                  {selectedLpnDetail?.lpn.lpn_code ? `(${selectedLpnDetail.lpn.lpn_code})` : ""}
                </h3>
              </div>

              {/* Quick Pick from Pending Staging Batches */}
              {putawayPending.length > 0 && (
                <div className="bg-emerald-50/70 border border-emerald-200 rounded-xl p-3 space-y-2">
                  <div className="flex items-center gap-1.5 text-xs font-semibold text-emerald-900">
                    <Sparkles className="w-3.5 h-3.5 text-emerald-600" />
                    Pilih Cepat Dari Barang Staging Masuk:
                  </div>
                  <div className="max-h-32 overflow-y-auto divide-y divide-emerald-100 bg-white rounded-lg border border-emerald-200">
                    {putawayPending.slice(0, 5).map((line) => (
                      <button
                        key={`${line.product_id}-${line.batch_id}`}
                        type="button"
                        onClick={() => {
                          setItemProductId(line.product_id)
                          setItemBatchId(line.batch_id)
                          setItemQuantity(String(line.quantity))
                        }}
                        className="w-full text-left px-3 py-1.5 text-xs hover:bg-emerald-50 flex items-center justify-between transition"
                      >
                        <div>
                          <span className="font-semibold text-slate-900">{line.product_name}</span>
                          <span className="ml-2 font-mono text-[10px] text-slate-500">
                            Batch: {line.batch_number}
                          </span>
                        </div>
                        <span className="font-mono font-bold text-emerald-700">
                          {line.quantity} unit
                        </span>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Product selection */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Produk <span className="text-rose-500">*</span>
                </label>
                <select
                  value={itemProductId}
                  onChange={(e) => setItemProductId(e.target.value)}
                  className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg bg-slate-50 focus:bg-white"
                  required
                >
                  <option value="">-- Pilih Produk Master --</option>
                  {products.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} ({p.sku})
                    </option>
                  ))}
                </select>
              </div>

              {/* Batch ID */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Batch ID (UUID) <span className="text-rose-500">*</span>
                </label>
                <input
                  type="text"
                  placeholder="Masukkan UUID Batch atau pilih dari daftar cepat di atas"
                  value={itemBatchId}
                  onChange={(e) => setItemBatchId(e.target.value)}
                  className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg font-mono"
                  required
                />
              </div>

              {/* Quantity */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Kuantitas Unit <span className="text-rose-500">*</span>
                </label>
                <input
                  type="number"
                  min="0.0001"
                  step="any"
                  placeholder="Contoh: 50"
                  value={itemQuantity}
                  onChange={(e) => setItemQuantity(e.target.value)}
                  className="w-full px-3 py-2 text-xs border border-slate-300 rounded-lg font-mono"
                  required
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-3 border-t">
                <button
                  type="button"
                  onClick={() => setActiveView(selectedLpnDetail ? "detail" : "list")}
                  className="px-4 py-2 text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={isAddingItem}
                  className="px-4 py-2 text-xs font-semibold text-white bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 rounded-lg shadow-sm"
                >
                  {isAddingItem ? "Menyimpan Item..." : "Masukkan ke Palet"}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>

      {/* Print LPN Label Modal Overlay */}
      {printDetail && (
        <PrintLPNLabel
          lpnDetail={printDetail}
          onClose={() => setPrintDetail(null)}
        />
      )}
    </div>
  )
}
