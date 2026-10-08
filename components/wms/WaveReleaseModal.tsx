"use client"

import React, { useState, useEffect } from "react"
import {
  Layers,
  X,
  Plus,
  Play,
  CheckCircle2,
  Clock,
  Truck,
  Building,
  RefreshCw,
  Boxes,
  FileText,
  AlertCircle,
  Users
} from "lucide-react"
import { api, Warehouse, PickWave, PickWaveDetail } from "@/lib/api"

interface WaveReleaseModalProps {
  isOpen: boolean
  onClose: () => void
  warehouses: Warehouse[]
  defaultWarehouseId?: string
  onWaveUpdated?: () => void
}

export function WaveReleaseModal({
  isOpen,
  onClose,
  warehouses,
  defaultWarehouseId,
  onWaveUpdated,
}: WaveReleaseModalProps) {
  const [activeTab, setActiveTab] = useState<"list" | "create">("list")
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>(
    defaultWarehouseId || warehouses[0]?.id || ""
  )
  const [orderType, setOrderType] = useState<string>("DIRECT_DO")
  const [expeditionName, setExpeditionName] = useState<string>("")
  const [routeZone, setRouteZone] = useState<string>("")
  const [notes, setNotes] = useState<string>("")

  const [waves, setWaves] = useState<PickWave[]>([])
  const [selectedWaveDetail, setSelectedWaveDetail] = useState<PickWaveDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [successMsg, setSuccessMsg] = useState<string | null>(null)

  useEffect(() => {
    if (isOpen) {
      loadWaves()
    }
  }, [isOpen, selectedWarehouseId])

  const loadWaves = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await api.wms.pickWaves.list({
        warehouse_id: selectedWarehouseId || undefined,
      })
      setWaves(res.data || [])
    } catch (err: any) {
      setError(err?.message || "Gagal memuat daftar wave")
    } finally {
      setLoading(false)
    }
  }

  const handleCreateWave = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedWarehouseId) {
      setError("Pilih gudang pemenuhan")
      return
    }

    try {
      setSubmitting(true)
      setError(null)
      setSuccessMsg(null)

      const res = await api.wms.pickWaves.create({
        warehouse_id: selectedWarehouseId,
        order_type: orderType,
        expedition_name: expeditionName || undefined,
        route_zone: routeZone || undefined,
        notes: notes || undefined,
      })

      setSuccessMsg(`Gelombang picking ${res.data.wave_number} berhasil dibuat dengan ${res.data.total_orders} pesanan!`)
      setActiveTab("list")
      await loadWaves()
      if (onWaveUpdated) onWaveUpdated()
    } catch (err: any) {
      setError(err?.message || "Gagal membuat gelombang picking. Pastikan ada pesanan DRAFT/CONFIRMED yang cocok.")
    } finally {
      setSubmitting(false)
    }
  }

  const handleReleaseWave = async (waveId: string) => {
    try {
      setSubmitting(true)
      setError(null)
      await api.wms.pickWaves.release(waveId)
      setSuccessMsg("Gelombang pengambilan berhasil dirilis ke tim gudang (IN_PROGRESS)!")
      await loadWaves()
      if (selectedWaveDetail && selectedWaveDetail.wave.id === waveId) {
        handleViewDetail(waveId)
      }
      if (onWaveUpdated) onWaveUpdated()
    } catch (err: any) {
      setError(err?.message || "Gagal merilis gelombang picking")
    } finally {
      setSubmitting(false)
    }
  }

  const handleViewDetail = async (waveId: string) => {
    try {
      setLoading(true)
      const res = await api.wms.pickWaves.get(waveId)
      setSelectedWaveDetail(res.data)
    } catch (err: any) {
      setError(err?.message || "Gagal memuat rincian wave")
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center overflow-y-auto bg-slate-900/60 p-4 backdrop-blur-xs">
      <div className="flex w-full max-w-4xl flex-col rounded-2xl bg-white shadow-2xl border border-slate-200 overflow-hidden max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50/80 px-6 py-4">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-50 border border-indigo-200 text-indigo-600">
              <Layers className="h-5 w-5" />
            </span>
            <div>
              <h2 className="text-base font-bold text-slate-900">
                Pelepasan Gelombang Picking (Wave Release - PDF-05)
              </h2>
              <p className="text-xs text-slate-500">
                Kelompokkan pesanan berdasarkan jenis order & kurir untuk efisiensi pengambilan batch massal
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-2 text-slate-400 hover:bg-slate-200 hover:text-slate-700 transition"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Tab Controls */}
        <div className="flex items-center justify-between border-b border-slate-200 px-6 pt-3 pb-0 bg-white">
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => {
                setActiveTab("list")
                setSelectedWaveDetail(null)
              }}
              className={`pb-3 px-3 text-xs font-semibold border-b-2 transition ${
                activeTab === "list"
                  ? "border-indigo-600 text-indigo-600"
                  : "border-transparent text-slate-500 hover:text-slate-800"
              }`}
            >
              Daftar Wave Aktif ({waves.length})
            </button>
            <button
              type="button"
              onClick={() => {
                setActiveTab("create")
                setSelectedWaveDetail(null)
              }}
              className={`pb-3 px-3 text-xs font-semibold border-b-2 transition ${
                activeTab === "create"
                  ? "border-indigo-600 text-indigo-600"
                  : "border-transparent text-slate-500 hover:text-slate-800"
              }`}
            >
              + Buat Wave Baru
            </button>
          </div>

          {/* Warehouse Selector */}
          <div className="flex items-center gap-2 pb-2">
            <Building className="h-3.5 w-3.5 text-slate-400" />
            <select
              value={selectedWarehouseId}
              onChange={(e) => setSelectedWarehouseId(e.target.value)}
              className="text-xs border border-slate-200 rounded-md px-2 py-1 bg-slate-50 text-slate-700 font-medium focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              {warehouses.map((wh) => (
                <option key={wh.id} value={wh.id}>
                  {wh.name}
                </option>
              ))}
            </select>
            <button
              type="button"
              onClick={loadWaves}
              disabled={loading}
              className="p-1 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition"
              title="Segarkan"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        {/* Status Alerts */}
        {error && (
          <div className="mx-6 mt-4 flex items-center gap-2 rounded-lg bg-rose-50 border border-rose-200 p-3 text-xs text-rose-700">
            <AlertCircle className="h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}
        {successMsg && (
          <div className="mx-6 mt-4 flex items-center gap-2 rounded-lg bg-emerald-50 border border-emerald-200 p-3 text-xs text-emerald-700">
            <CheckCircle2 className="h-4 w-4 shrink-0" />
            <span>{successMsg}</span>
          </div>
        )}

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto p-6">
          {activeTab === "create" ? (
            /* Formulir Pembuatan Wave */
            <form onSubmit={handleCreateWave} className="space-y-4 max-w-xl mx-auto">
              <div className="rounded-xl border border-indigo-100 bg-indigo-50/40 p-4 text-xs text-indigo-900 leading-relaxed">
                <span className="font-bold">Logika Pengelompokan Otomatis (OCA §1.3):</span>
                <p className="mt-1">
                  Sistem akan mengumpulkan seluruh Surat Jalan berstatus DRAFT atau CONFIRMED di gudang terpilih yang
                  sesuai dengan kriteria di bawah, lalu mengikatnya ke dalam satu batch instruksi picking.
                </p>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Tipe Pesanan / Saluran (Order Type) *
                </label>
                <select
                  value={orderType}
                  onChange={(e) => setOrderType(e.target.value)}
                  className="w-full text-xs border border-slate-200 rounded-lg p-2.5 bg-white text-slate-800 focus:ring-2 focus:ring-indigo-500 focus:outline-none"
                >
                  <option value="DIRECT_DO">Surat Jalan Langsung (DIRECT_DO)</option>
                  <option value="SALES_ORDER">Pesanan Penjualan B2B (SALES_ORDER)</option>
                  <option value="MARKETPLACE">E-Commerce / Marketplace (MARKETPLACE)</option>
                  <option value="TRANSFER">Transfer Antargudang (TRANSFER)</option>
                </select>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Kurir / Ekspedisi (Opsional)
                  </label>
                  <input
                    type="text"
                    value={expeditionName}
                    onChange={(e) => setExpeditionName(e.target.value)}
                    placeholder="Contoh: JNE, SICEPAT, ARMADA"
                    className="w-full text-xs border border-slate-200 rounded-lg p-2.5 bg-white text-slate-800 placeholder-slate-400 focus:ring-2 focus:ring-indigo-500 focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Zona Rute Pengiriman (Opsional)
                  </label>
                  <input
                    type="text"
                    value={routeZone}
                    onChange={(e) => setRouteZone(e.target.value)}
                    placeholder="Contoh: JABODETABEK, ZONA-BARAT"
                    className="w-full text-xs border border-slate-200 rounded-lg p-2.5 bg-white text-slate-800 placeholder-slate-400 focus:ring-2 focus:ring-indigo-500 focus:outline-none"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Catatan untuk Tim Picker (Opsional)
                </label>
                <textarea
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  rows={2}
                  placeholder="Catatan prioritas, batas waktu pengangkutan, dsb."
                  className="w-full text-xs border border-slate-200 rounded-lg p-2.5 bg-white text-slate-800 placeholder-slate-400 focus:ring-2 focus:ring-indigo-500 focus:outline-none"
                />
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setActiveTab("list")}
                  className="px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-100 rounded-lg transition"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={submitting}
                  className="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-semibold bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition disabled:opacity-50 shadow-sm"
                >
                  <Plus className="h-4 w-4" />
                  {submitting ? "Membuat Wave..." : "Rilis & Buat Wave"}
                </button>
              </div>
            </form>
          ) : selectedWaveDetail ? (
            /* Tampilan Detail Wave & Picking Tasks */
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b border-slate-200 pb-3">
                <div>
                  <button
                    type="button"
                    onClick={() => setSelectedWaveDetail(null)}
                    className="text-xs text-indigo-600 hover:underline mb-1 inline-block"
                  >
                    &larr; Kembali ke daftar wave
                  </button>
                  <h3 className="text-sm font-bold text-slate-900 flex items-center gap-2">
                    {selectedWaveDetail.wave.wave_number}
                    <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-700 border border-indigo-200">
                      {selectedWaveDetail.wave.order_type}
                    </span>
                  </h3>
                </div>
                {selectedWaveDetail.wave.status === "OPEN" && (
                  <button
                    type="button"
                    onClick={() => handleReleaseWave(selectedWaveDetail.wave.id)}
                    disabled={submitting}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition shadow-sm"
                  >
                    <Play className="h-3.5 w-3.5" />
                    Rilis Wave (IN_PROGRESS)
                  </button>
                )}
              </div>

              <div className="grid grid-cols-4 gap-3 text-xs bg-slate-50 p-3 rounded-xl border border-slate-200">
                <div>
                  <span className="text-slate-400 block">Status</span>
                  <span className="font-bold text-slate-800">{selectedWaveDetail.wave.status}</span>
                </div>
                <div>
                  <span className="text-slate-400 block">Kurir / Rute</span>
                  <span className="font-bold text-slate-800">
                    {selectedWaveDetail.wave.expedition_name || "-"} / {selectedWaveDetail.wave.route_zone || "-"}
                  </span>
                </div>
                <div>
                  <span className="text-slate-400 block">Total DO</span>
                  <span className="font-bold text-slate-800">{selectedWaveDetail.wave.total_orders} Pesanan</span>
                </div>
                <div>
                  <span className="text-slate-400 block">Total Baris Rak</span>
                  <span className="font-bold text-slate-800">{selectedWaveDetail.wave.total_lines} Baris</span>
                </div>
              </div>

              {/* Tabel Pesanan dalam Wave */}
              <div>
                <h4 className="text-xs font-bold text-slate-800 mb-2">
                  Daftar Instruksi Pengambilan ({selectedWaveDetail.picking_tasks.length} DO)
                </h4>
                <div className="border border-slate-200 rounded-xl overflow-hidden divide-y divide-slate-200">
                  {selectedWaveDetail.picking_tasks.map((pt) => (
                    <div key={pt.task.id} className="p-3 bg-white hover:bg-slate-50 transition flex items-center justify-between text-xs">
                      <div className="space-y-1">
                        <div className="flex items-center gap-2">
                          <span className="font-mono font-bold text-indigo-600">{pt.task.task_number}</span>
                          <span className="text-slate-500 font-mono">({pt.delivery_order.do_number})</span>
                          <span className="px-1.5 py-0.5 rounded text-[10px] bg-slate-100 text-slate-700">
                            {pt.task.status}
                          </span>
                        </div>
                        <div className="text-slate-500">
                          Penerima: <span className="text-slate-800 font-medium">{pt.delivery_order.recipient_name || pt.delivery_order.customer_name || "-"}</span>
                          {" &bull; "}
                          {pt.items.length} item barang
                        </div>
                      </div>
                      <div className="text-right">
                        <span className="text-[11px] text-slate-500 block">
                          {pt.delivery_order.expedition_name || "Internal"}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            /* Daftar Wave Aktif */
            <div className="space-y-3">
              {loading ? (
                <div className="py-12 text-center text-xs text-slate-400">Memuat daftar gelombang picking...</div>
              ) : waves.length === 0 ? (
                <div className="py-12 text-center text-xs text-slate-400 space-y-2">
                  <Boxes className="h-8 w-8 mx-auto text-slate-300" />
                  <p>Belum ada gelombang picking untuk gudang ini.</p>
                  <button
                    type="button"
                    onClick={() => setActiveTab("create")}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition"
                  >
                    <Plus className="h-3.5 w-3.5" />
                    Buat Wave Pertama
                  </button>
                </div>
              ) : (
                waves.map((w) => (
                  <div
                    key={w.id}
                    className="rounded-xl border border-slate-200 bg-white p-4 hover:border-indigo-300 transition shadow-xs flex items-center justify-between gap-4"
                  >
                    <div className="space-y-1">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold text-sm text-slate-900">{w.wave_number}</span>
                        <span className={`text-[10px] font-bold px-2 py-0.5 rounded-full ${
                          w.status === "OPEN"
                            ? "bg-amber-50 text-amber-700 border border-amber-200"
                            : w.status === "RELEASED" || w.status === "IN_PROGRESS"
                            ? "bg-indigo-50 text-indigo-700 border border-indigo-200"
                            : "bg-emerald-50 text-emerald-700 border border-emerald-200"
                        }`}>
                          {w.status}
                        </span>
                        <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-100 text-slate-600">
                          {w.order_type}
                        </span>
                      </div>
                      <div className="text-xs text-slate-500 flex items-center gap-3">
                        <span>Kurir: <strong className="text-slate-700">{w.expedition_name || "Semua"}</strong></span>
                        {w.route_zone && <span>Zona: <strong className="text-slate-700">{w.route_zone}</strong></span>}
                        <span>Pesanan: <strong className="text-indigo-600">{w.total_orders} DO</strong></span>
                      </div>
                    </div>

                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => handleViewDetail(w.id)}
                        className="px-3 py-1.5 text-xs font-medium text-slate-700 border border-slate-200 rounded-lg hover:bg-slate-50 transition"
                      >
                        Lihat DO ({w.total_orders})
                      </button>
                      {w.status === "OPEN" && (
                        <button
                          type="button"
                          onClick={() => handleReleaseWave(w.id)}
                          disabled={submitting}
                          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition shadow-xs disabled:opacity-50"
                        >
                          <Play className="h-3.5 w-3.5" />
                          Rilis Wave
                        </button>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
