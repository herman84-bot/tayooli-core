"use client"

import React, { useState } from "react"
import {
  ArrowDownToLine,
  CheckCircle2,
  AlertCircle,
  HelpCircle,
  Warehouse,
  Layers,
  Clock,
  ShieldAlert,
  Search,
  Filter,
} from "lucide-react"
import {
  usePutawayPending,
  useConfirmPutaway,
  useWarehouseLocations,
  useProductDefaultLocations,
} from "@/hooks/useWMS"
import type { PutawayPendingLine, WarehouseLocation } from "@/lib/api"

interface PutawayViewProps {
  warehouseId: string | null
  onRefresh?: () => void
}

export function PutawayView({ warehouseId, onRefresh }: PutawayViewProps) {
  const { data: pending = [], isLoading, refetch } = usePutawayPending(warehouseId)
  const { data: locations = [] } = useWarehouseLocations(warehouseId)
  const { mutate: confirmPutaway, isPending: isSubmitting } = useConfirmPutaway()

  const [selectedLine, setSelectedLine] = useState<PutawayPendingLine | null>(null)
  const [destLocationId, setDestLocationId] = useState<string>("")
  const [putawayQty, setPutawayQty] = useState<string>("")
  const [reason, setReason] = useState<string>("")
  const [errorMsg, setErrorMsg] = useState<string | null>(null)
  const [successToast, setSuccessToast] = useState<string | null>(null)
  const [search, setSearch] = useState<string>("")

  // Filter only internal racks for putaway destinations
  const internalRacks = locations.filter((l) => l.type === "INTERNAL")

  const filteredPending = pending.filter((line) => {
    if (!search.trim()) return true
    const q = search.toLowerCase()
    return (
      (line.product_name && line.product_name.toLowerCase().includes(q)) ||
      (line.product_sku && line.product_sku.toLowerCase().includes(q)) ||
      (line.batch_number && line.batch_number.toLowerCase().includes(q)) ||
      (line.source_receipt_number && line.source_receipt_number.toLowerCase().includes(q))
    )
  })

  const openPutawayModal = (line: PutawayPendingLine) => {
    setSelectedLine(line)
    setPutawayQty(String(line.quantity))
    // Default to suggested location if internal rack, else first rack
    const suggestedId = line.suggested_location_id || line.default_location_id
    if (suggestedId && internalRacks.some((r) => r.id === suggestedId)) {
      setDestLocationId(suggestedId)
    } else if (internalRacks.length > 0) {
      setDestLocationId(internalRacks[0].id)
    } else {
      setDestLocationId("")
    }
    setReason("")
    setErrorMsg(null)
  }

  const handleConfirm = () => {
    if (!selectedLine || !warehouseId) return
    const qtyNum = parseFloat(putawayQty)
    if (!qtyNum || qtyNum <= 0) {
      setErrorMsg("Jumlah putaway harus lebih besar dari 0")
      return
    }
    if (qtyNum > Number(selectedLine.quantity)) {
      setErrorMsg(`Jumlah putaway tidak boleh melebihi stok tersedia (${selectedLine.quantity})`)
      return
    }
    if (!destLocationId) {
      setErrorMsg("Pilih rak internal tujuan putaway")
      return
    }

    // Invariant 4: If overriding default rack, reason is required
    const isOverridingDefault =
      selectedLine.default_location_id &&
      destLocationId !== selectedLine.default_location_id
    if (isOverridingDefault && !reason.trim()) {
      setErrorMsg("Alasan wajib diisi jika lokasi tujuan berbeda dari rak default produk (ADR-014 Invariant 4)")
      return
    }

    confirmPutaway(
      {
        warehouse_id: warehouseId,
        product_id: selectedLine.product_id,
        batch_id: selectedLine.batch_id,
        quantity: qtyNum,
        dest_location_id: destLocationId,
        reason: isOverridingDefault ? reason.trim() : undefined,
      },
      {
        onSuccess: () => {
          setSelectedLine(null)
          setSuccessToast(`Berhasil putaway ${qtyNum} unit ke rak tujuan`)
          refetch()
          onRefresh?.()
          setTimeout(() => setSuccessToast(null), 4000)
        },
        onError: (err: unknown) => {
          const msg = err instanceof Error ? err.message : "Gagal memproses putaway"
          setErrorMsg(msg)
        },
      }
    )
  }

  if (!warehouseId) {
    return (
      <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 p-12 text-center">
        <Warehouse className="mx-auto h-12 w-12 text-slate-400" />
        <h3 className="mt-3 text-base font-medium text-slate-800">Pilih Gudang Terlebih Dahulu</h3>
        <p className="mt-1 text-sm text-slate-500">
          Pilih salah satu gudang pada filter di atas untuk melihat barang di Staging Inbound yang menunggu putaway.
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {successToast && (
        <div className="flex items-center gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
          <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600" />
          <span>{successToast}</span>
        </div>
      )}

      {/* Toolbar */}
      <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 bg-white p-3 rounded-lg border border-slate-200 shadow-sm">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-400" />
          <input
            type="text"
            placeholder="Cari produk, SKU, batch, atau nomor penerimaan..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-4 py-1.5 text-sm rounded-md border border-slate-200 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
          />
        </div>
        <div className="text-xs text-slate-500 font-medium">
          Total antrean putaway: <span className="font-semibold text-slate-900">{filteredPending.length} baris</span>
        </div>
      </div>

      {/* Table */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-slate-50/80 text-xs font-semibold text-slate-600 border-b border-slate-200 uppercase tracking-wider">
              <tr>
                <th className="px-4 py-3">Produk & SKU</th>
                <th className="px-4 py-3">Batch / Lot</th>
                <th className="px-4 py-3">Kedaluwarsa</th>
                <th className="px-4 py-3">Status Lot</th>
                <th className="px-4 py-3 text-right">Qty di Staging</th>
                <th className="px-4 py-3">Saran Rak Tujuan</th>
                <th className="px-4 py-3 text-center">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {isLoading ? (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-slate-400">
                    Memuat antrean putaway...
                  </td>
                </tr>
              ) : filteredPending.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-12 text-center text-slate-400">
                    <Layers className="mx-auto h-8 w-8 text-slate-300 mb-2" />
                    Tidak ada barang di Staging Inbound yang menunggu putaway untuk gudang ini.
                  </td>
                </tr>
              ) : (
                filteredPending.map((line) => {
                  const isDefaultRack = line.suggestion_source === "DEFAULT_RACK"
                  const isReceiptDest = line.suggestion_source === "RECEIPT_DEST"
                  return (
                    <tr key={`${line.product_id}-${line.batch_id}`} className="hover:bg-slate-50/60 transition-colors">
                      <td className="px-4 py-3">
                        <div className="font-medium text-slate-900">{line.product_name || "Produk Tanpa Nama"}</div>
                        <div className="text-xs text-slate-500 font-mono">{line.product_sku || "-"}</div>
                        {line.source_receipt_number && (
                          <div className="text-[11px] text-slate-400 mt-0.5">GR: {line.source_receipt_number}</div>
                        )}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs font-medium text-slate-800">
                        {line.batch_number}
                      </td>
                      <td className="px-4 py-3 text-xs text-slate-600">
                        {line.expiry_date ? (
                          <span className="inline-flex items-center gap-1">
                            <Clock className="h-3 w-3 text-slate-400" />
                            {new Date(line.expiry_date).toLocaleDateString("id-ID")}
                          </span>
                        ) : (
                          <span className="text-slate-400">-</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                            line.batch_status === "RELEASED"
                              ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                              : "bg-amber-50 text-amber-700 border border-amber-200"
                          }`}
                        >
                          {line.batch_status === "RELEASED" ? "Dirilis" : "Tertahan (ON_HOLD)"}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right font-semibold text-slate-900">
                        {Number(line.quantity).toLocaleString("id-ID")}
                      </td>
                      <td className="px-4 py-3">
                        {line.suggested_location_code ? (
                          <div className="inline-flex items-center gap-1.5">
                            <span className="font-mono text-xs font-semibold px-2 py-0.5 bg-slate-100 rounded text-slate-800 border border-slate-200">
                              {line.suggested_location_code}
                            </span>
                            <span
                              className={`text-[10px] uppercase font-bold px-1.5 py-0.2 rounded ${
                                isDefaultRack
                                  ? "bg-blue-50 text-blue-600 border border-blue-200"
                                  : isReceiptDest
                                  ? "bg-purple-50 text-purple-600 border border-purple-200"
                                  : "bg-slate-50 text-slate-600"
                              }`}
                            >
                              {isDefaultRack ? "Rak Default" : "Tujuan GR"}
                            </span>
                          </div>
                        ) : (
                          <span className="text-xs text-slate-400 italic">Belum ditentukan</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-center">
                        <button
                          onClick={() => openPutawayModal(line)}
                          className="inline-flex items-center gap-1 px-3 py-1 text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 rounded-md transition-colors shadow-sm"
                        >
                          <ArrowDownToLine className="h-3.5 w-3.5" />
                          Putaway ke Rak
                        </button>
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Confirmation Modal */}
      {selectedLine && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in">
          <div className="w-full max-w-lg rounded-xl bg-white p-6 shadow-xl border border-slate-100 space-y-5">
            <div>
              <h3 className="text-lg font-semibold text-slate-900">Konfirmasi Putaway ke Rak Internal</h3>
              <p className="text-xs text-slate-500 mt-1">
                Pindahkan barang dari area Staging Inbound ke rak penyimpanan tetap (sentry-wms §1.2 step 2).
              </p>
            </div>

            {errorMsg && (
              <div className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
                <AlertCircle className="h-4 w-4 shrink-0 text-rose-600 mt-0.5" />
                <span>{errorMsg}</span>
              </div>
            )}

            <div className="rounded-lg bg-slate-50 border border-slate-200 p-3 space-y-1.5 text-xs">
              <div className="flex justify-between">
                <span className="text-slate-500">Produk:</span>
                <span className="font-semibold text-slate-800">{selectedLine.product_name}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">SKU:</span>
                <span className="font-mono text-slate-700">{selectedLine.product_sku}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Batch / Lot:</span>
                <span className="font-mono font-medium text-slate-800">{selectedLine.batch_number}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-slate-500">Stok Tersedia di Staging:</span>
                <span className="font-semibold text-slate-900">{selectedLine.quantity} unit</span>
              </div>
            </div>

            <div className="space-y-4 text-sm">
              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">
                  Jumlah yang Disimpan ke Rak <span className="text-rose-500">*</span>
                </label>
                <input
                  type="number"
                  step="any"
                  value={putawayQty}
                  onChange={(e) => setPutawayQty(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 text-sm font-semibold"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-slate-700 mb-1">
                  Pilih Rak Internal Tujuan <span className="text-rose-500">*</span>
                </label>
                <select
                  value={destLocationId}
                  onChange={(e) => setDestLocationId(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-md focus:outline-none focus:ring-2 focus:ring-primary/20 text-sm"
                >
                  <option value="">-- Pilih Rak Internal --</option>
                  {internalRacks.map((rack) => (
                    <option key={rack.id} value={rack.id}>
                      {rack.code} {rack.name ? `(${rack.name})` : ""}
                      {selectedLine.default_location_id === rack.id ? " [Rak Default]" : ""}
                    </option>
                  ))}
                </select>
              </div>

              {/* Invariant 4 Warning & Reason Field */}
              {selectedLine.default_location_id &&
                destLocationId &&
                destLocationId !== selectedLine.default_location_id && (
                  <div className="rounded-lg border border-amber-300 bg-amber-50 p-3 space-y-2">
                    <div className="flex items-center gap-2 text-xs font-semibold text-amber-800">
                      <ShieldAlert className="h-4 w-4 text-amber-600 shrink-0" />
                      Penyimpangan dari Rak Default Produk Terdeteksi
                    </div>
                    <p className="text-[11px] text-amber-700">
                      Sistem mencatat rak default produk ini berbeda dari rak yang Anda pilih. Sesuai kebijakan ISO &
                      ADR-014 Invariant 4, Anda wajib menyertakan alasan penyimpangan rak.
                    </p>
                    <div>
                      <label className="block text-xs font-medium text-amber-900 mb-1">
                        Alasan Penyimpangan Rak <span className="text-rose-600">*</span>
                      </label>
                      <input
                        type="text"
                        placeholder="Contoh: Rak default penuh, sedang dalam perbaikan..."
                        value={reason}
                        onChange={(e) => setReason(e.target.value)}
                        className="w-full px-3 py-1.5 border border-amber-300 rounded bg-white text-xs text-slate-800 focus:outline-none focus:ring-2 focus:ring-amber-500"
                      />
                    </div>
                  </div>
                )}
            </div>

            <div className="flex justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                type="button"
                onClick={() => setSelectedLine(null)}
                className="px-4 py-2 text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-md transition-colors"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={handleConfirm}
                disabled={isSubmitting}
                className="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 rounded-md transition-colors shadow-sm"
              >
                {isSubmitting ? "Menyimpan..." : "Konfirmasi Putaway"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
