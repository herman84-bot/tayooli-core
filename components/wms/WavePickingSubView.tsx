"use client"

import React, { useState, useEffect } from "react"
import {
  Layers,
  Plus,
  Play,
  CheckCircle2,
  Clock,
  Printer,
  Search,
  Building,
  RefreshCw,
  Boxes,
  FileText,
  AlertCircle,
  Truck,
  ChevronRight,
  UserCheck
} from "lucide-react"
import { api, PickWave, PickWaveDetail, PickingTaskDetail } from "@/lib/api"
import { useWarehouses } from "@/hooks/useWMS"
import { WaveReleaseModal } from "@/components/wms/WaveReleaseModal"
import { PrintPickingList } from "@/components/wms/PrintPickingList"

interface WavePickingSubViewProps {
  warehouseId?: string
}

export function WavePickingSubView({ warehouseId }: WavePickingSubViewProps) {
  const { data: warehouses = [] } = useWarehouses()
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>(
    warehouseId || ""
  )
  const [orderTypeFilter, setOrderTypeFilter] = useState<string>("ALL")
  const [statusFilter, setStatusFilter] = useState<string>("ALL")
  const [searchQuery, setSearchQuery] = useState<string>("")

  const [waves, setWaves] = useState<PickWave[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Modals state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [selectedWaveDetail, setSelectedWaveDetail] = useState<PickWaveDetail | null>(null)
  const [loadingDetail, setLoadingDetail] = useState(false)
  const [selectedTaskForPrint, setSelectedTaskForPrint] = useState<PickingTaskDetail | null>(null)
  const [releasingId, setReleasingId] = useState<string | null>(null)
  const [toastMsg, setToastMsg] = useState<{ type: "success" | "error"; text: string } | null>(null)

  useEffect(() => {
    if (warehouseId) {
      setSelectedWarehouseId(warehouseId)
    }
  }, [warehouseId])

  useEffect(() => {
    loadWaves()
  }, [selectedWarehouseId, orderTypeFilter, statusFilter])

  const loadWaves = async () => {
    try {
      setLoading(true)
      setError(null)
      const res = await api.wms.pickWaves.list({
        warehouse_id: selectedWarehouseId || undefined,
        order_type: orderTypeFilter !== "ALL" ? orderTypeFilter : undefined,
        status: statusFilter !== "ALL" ? statusFilter : undefined,
      })
      setWaves(res.data || [])
    } catch (err: any) {
      setError(err?.message || "Gagal memuat daftar gelombang picking")
    } finally {
      setLoading(false)
    }
  }

  const handleReleaseWave = async (waveId: string) => {
    try {
      setReleasingId(waveId)
      await api.wms.pickWaves.release(waveId)
      setToastMsg({ type: "success", text: "Gelombang picking berhasil dirilis (IN_PROGRESS)!" })
      await loadWaves()
      if (selectedWaveDetail && selectedWaveDetail.wave.id === waveId) {
        handleViewWaveDetail(waveId)
      }
    } catch (err: any) {
      setToastMsg({ type: "error", text: err?.message || "Gagal merilis gelombang picking" })
    } finally {
      setReleasingId(null)
    }
  }

  const handleViewWaveDetail = async (waveId: string) => {
    try {
      setLoadingDetail(true)
      const res = await api.wms.pickWaves.get(waveId)
      setSelectedWaveDetail(res.data)
    } catch (err: any) {
      setToastMsg({ type: "error", text: err?.message || "Gagal memuat rincian wave" })
    } finally {
      setLoadingDetail(false)
    }
  }

  // Filtered waves by search query
  const filteredWaves = waves.filter((w) => {
    if (!searchQuery) return true
    const q = searchQuery.toLowerCase()
    return (
      w.wave_number.toLowerCase().includes(q) ||
      (w.expedition_name && w.expedition_name.toLowerCase().includes(q)) ||
      (w.route_zone && w.route_zone.toLowerCase().includes(q))
    )
  })

  // Summary Metrics
  const totalWaves = waves.length
  const openWaves = waves.filter((w) => w.status === "OPEN").length
  const inProgressWaves = waves.filter((w) => w.status === "RELEASED" || w.status === "IN_PROGRESS").length
  const completedWaves = waves.filter((w) => w.status === "COMPLETED").length

  return (
    <div className="space-y-6">
      {/* ── Toast Notification ── */}
      {toastMsg && (
        <div
          className={`p-3 rounded-lg text-xs flex items-center justify-between border ${
            toastMsg.type === "success"
              ? "bg-emerald-50 text-emerald-800 border-emerald-200"
              : "bg-rose-50 text-rose-800 border-rose-200"
          }`}
        >
          <span>{toastMsg.text}</span>
          <button
            type="button"
            onClick={() => setToastMsg(null)}
            className="text-slate-400 hover:text-slate-700 ml-4 font-bold"
          >
            &times;
          </button>
        </div>
      )}

      {/* ── Summary Metrics Cards ── */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-xl border border-slate-200 p-4 shadow-2xs">
          <div className="flex items-center justify-between text-slate-500 text-xs mb-1">
            <span>Total Gelombang</span>
            <Layers className="h-4 w-4 text-slate-400" />
          </div>
          <div className="text-2xl font-bold text-slate-900">{totalWaves}</div>
          <div className="text-[11px] text-slate-400 mt-1">Seluruh batch picking</div>
        </div>

        <div className="bg-white rounded-xl border border-slate-200 p-4 shadow-2xs">
          <div className="flex items-center justify-between text-amber-600 text-xs mb-1">
            <span>Menunggu Rilis (OPEN)</span>
            <Clock className="h-4 w-4 text-amber-500" />
          </div>
          <div className="text-2xl font-bold text-amber-600">{openWaves}</div>
          <div className="text-[11px] text-slate-400 mt-1">Siap dilepas ke picker</div>
        </div>

        <div className="bg-white rounded-xl border border-slate-200 p-4 shadow-2xs">
          <div className="flex items-center justify-between text-indigo-600 text-xs mb-1">
            <span>Sedang Dipick</span>
            <Boxes className="h-4 w-4 text-indigo-500" />
          </div>
          <div className="text-2xl font-bold text-indigo-600">{inProgressWaves}</div>
          <div className="text-[11px] text-slate-400 mt-1">Petugas sedang mengambil</div>
        </div>

        <div className="bg-white rounded-xl border border-slate-200 p-4 shadow-2xs">
          <div className="flex items-center justify-between text-emerald-600 text-xs mb-1">
            <span>Selesai Dipick</span>
            <CheckCircle2 className="h-4 w-4 text-emerald-500" />
          </div>
          <div className="text-2xl font-bold text-emerald-600">{completedWaves}</div>
          <div className="text-[11px] text-slate-400 mt-1">Siap ke meja kemas</div>
        </div>
      </div>

      {/* ── Control Bar & Action ── */}
      <div className="bg-white rounded-xl border border-slate-200 p-4 flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3 shadow-2xs">
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Gudang */}
          <div className="flex items-center gap-1.5 text-xs text-slate-500">
            <Building className="h-3.5 w-3.5 text-slate-400" />
            <select
              value={selectedWarehouseId}
              onChange={(e) => setSelectedWarehouseId(e.target.value)}
              className="bg-slate-50 border border-slate-200 rounded-lg px-2.5 py-1.5 text-xs text-slate-800 font-medium focus:outline-none focus:ring-1 focus:ring-indigo-500"
            >
              <option value="">Semua Gudang</option>
              {warehouses.map((wh) => (
                <option key={wh.id} value={wh.id}>
                  {wh.name}
                </option>
              ))}
            </select>
          </div>

          {/* Tipe Order */}
          <select
            value={orderTypeFilter}
            onChange={(e) => setOrderTypeFilter(e.target.value)}
            className="bg-slate-50 border border-slate-200 rounded-lg px-2.5 py-1.5 text-xs text-slate-800 font-medium focus:outline-none focus:ring-1 focus:ring-indigo-500"
          >
            <option value="ALL">Semua Jenis Order</option>
            <option value="DIRECT_DO">Surat Jalan (DIRECT_DO)</option>
            <option value="SALES_ORDER">Pesanan Penjualan (SALES_ORDER)</option>
            <option value="MARKETPLACE">E-Commerce (MARKETPLACE)</option>
            <option value="TRANSFER">Transfer Antargudang (TRANSFER)</option>
          </select>

          {/* Status */}
          <div className="flex items-center gap-1 bg-slate-100 p-0.5 rounded-lg border border-slate-200 text-xs">
            {[
              { label: "Semua", value: "ALL" },
              { label: "Open", value: "OPEN" },
              { label: "Berjalan", value: "IN_PROGRESS" },
              { label: "Selesai", value: "COMPLETED" },
            ].map((t) => (
              <button
                key={t.value}
                onClick={() => setStatusFilter(t.value)}
                className={`px-2.5 py-1 rounded-md font-medium transition ${
                  statusFilter === t.value
                    ? "bg-white text-slate-900 shadow-2xs font-bold"
                    : "text-slate-600 hover:text-slate-900"
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>

          <button
            onClick={loadWaves}
            disabled={loading}
            className="p-2 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition"
            title="Segarkan data"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
          </button>
        </div>

        {/* Search & Buat Wave Button */}
        <div className="flex items-center gap-2">
          <div className="relative flex-1 sm:w-64">
            <Search className="h-3.5 w-3.5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              placeholder="Cari no. wave, kurir, zona..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-8 pr-3 py-1.5 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-1 focus:ring-indigo-500"
            />
          </div>

          <button
            onClick={() => setIsCreateModalOpen(true)}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg font-semibold text-xs bg-indigo-600 text-white hover:bg-indigo-700 active:scale-95 transition shadow-2xs shrink-0"
          >
            <Plus className="h-3.5 w-3.5" />
            <span>+ Buat Wave Baru</span>
          </button>
        </div>
      </div>

      {/* ── Waves Table & List ── */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-2xs">
        {loading ? (
          <div className="py-16 text-center text-xs text-slate-400">
            <RefreshCw className="h-6 w-6 animate-spin mx-auto text-slate-300 mb-2" />
            Memuat daftar gelombang picking...
          </div>
        ) : filteredWaves.length === 0 ? (
          <div className="py-16 text-center text-slate-400 space-y-2">
            <Boxes className="h-10 w-10 mx-auto text-slate-300" />
            <p className="text-sm font-semibold text-slate-700">Belum ada gelombang picking</p>
            <p className="text-xs text-slate-400 max-w-sm mx-auto">
              Kelompokkan Surat Jalan berstatus DRAFT/CONFIRMED menjadi satu gelombang picking (Wave) untuk
              memaksimalkan efisiensi rute penataan rak FEFO.
            </p>
            <button
              onClick={() => setIsCreateModalOpen(true)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition"
            >
              <Plus className="h-3.5 w-3.5" />
              Buat Wave Sekarang
            </button>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="border-b border-slate-200 bg-slate-50/70 text-slate-500 font-semibold uppercase tracking-wider">
                  <th className="py-3.5 px-4">No. Wave</th>
                  <th className="py-3.5 px-4">Tipe Order</th>
                  <th className="py-3.5 px-4">Kurir / Zona Rute</th>
                  <th className="py-3.5 px-4">Jumlah DO & Baris</th>
                  <th className="py-3.5 px-4">Picker</th>
                  <th className="py-3.5 px-4">Status</th>
                  <th className="py-3.5 px-4 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white">
                {filteredWaves.map((w) => (
                  <tr key={w.id} className="hover:bg-slate-50/80 transition group">
                    <td className="py-3.5 px-4">
                      <div className="font-mono font-bold text-slate-900">{w.wave_number}</div>
                      <div className="text-[10px] text-slate-400">
                        {new Date(w.created_at).toLocaleString("id-ID")}
                      </div>
                    </td>
                    <td className="py-3.5 px-4">
                      <span className="inline-block px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-indigo-50 text-indigo-700 border border-indigo-200">
                        {w.order_type}
                      </span>
                    </td>
                    <td className="py-3.5 px-4">
                      <div className="text-slate-800 font-medium">{w.expedition_name || "Semua Ekspedisi"}</div>
                      <div className="text-[10px] text-slate-400">{w.route_zone || "Semua Zona"}</div>
                    </td>
                    <td className="py-3.5 px-4">
                      <div className="font-bold text-slate-800">{w.total_orders} Surat Jalan</div>
                      <div className="text-[10px] text-slate-400">{w.total_lines} Baris Rak</div>
                    </td>
                    <td className="py-3.5 px-4 text-slate-600">
                      {w.picker_name ? (
                        <div className="flex items-center gap-1 text-slate-800 font-medium">
                          <UserCheck className="h-3 w-3 text-emerald-600" />
                          <span>{w.picker_name}</span>
                        </div>
                      ) : (
                        <span className="text-slate-400 italic">Belum ditugaskan</span>
                      )}
                    </td>
                    <td className="py-3.5 px-4">
                      <span
                        className={`inline-block px-2.5 py-0.5 rounded-full text-[10px] font-bold border tracking-wider uppercase ${
                          w.status === "OPEN"
                            ? "bg-amber-50 text-amber-700 border-amber-200"
                            : w.status === "RELEASED" || w.status === "IN_PROGRESS"
                            ? "bg-indigo-50 text-indigo-700 border-indigo-200"
                            : "bg-emerald-50 text-emerald-700 border-emerald-200"
                        }`}
                      >
                        {w.status}
                      </span>
                    </td>
                    <td className="py-3.5 px-4 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        <button
                          type="button"
                          onClick={() => handleViewWaveDetail(w.id)}
                          className="px-2.5 py-1 text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 transition text-[11px] font-medium"
                        >
                          Lihat DO ({w.total_orders})
                        </button>
                        {w.status === "OPEN" && (
                          <button
                            type="button"
                            onClick={() => handleReleaseWave(w.id)}
                            disabled={releasingId === w.id}
                            className="inline-flex items-center gap-1 px-2.5 py-1 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg transition text-[11px] font-semibold disabled:opacity-50 shadow-2xs"
                          >
                            <Play className="h-3 w-3" />
                            <span>Rilis</span>
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* ── Modal Pratinjau Detail Wave & Picking Tasks ── */}
      {selectedWaveDetail && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs overflow-y-auto">
          <div className="bg-white rounded-2xl border border-slate-200 shadow-2xl max-w-3xl w-full p-6 space-y-4 max-h-[85vh] flex flex-col">
            <div className="flex items-center justify-between border-b border-slate-200 pb-3">
              <div>
                <h3 className="text-base font-bold text-slate-900 flex items-center gap-2">
                  <span>{selectedWaveDetail.wave.wave_number}</span>
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-indigo-50 text-indigo-700 border border-indigo-200">
                    {selectedWaveDetail.wave.order_type}
                  </span>
                </h3>
                <p className="text-xs text-slate-500">
                  Kurir: {selectedWaveDetail.wave.expedition_name || "Semua"} &bull; Zona:{" "}
                  {selectedWaveDetail.wave.route_zone || "Semua"} &bull; {selectedWaveDetail.wave.total_orders} Pesanan
                </p>
              </div>
              <button
                type="button"
                onClick={() => setSelectedWaveDetail(null)}
                className="text-slate-400 hover:text-slate-700 p-1 rounded-lg text-lg font-bold"
              >
                &times;
              </button>
            </div>

            <div className="flex-1 overflow-y-auto space-y-3">
              <h4 className="text-xs font-bold text-slate-700">Daftar Instruksi Picking (FE-06):</h4>
              <div className="border border-slate-200 rounded-xl divide-y divide-slate-200 overflow-hidden">
                {selectedWaveDetail.picking_tasks.map((pt) => (
                  <div key={pt.task.id} className="p-3 bg-white hover:bg-slate-50 transition flex items-center justify-between text-xs">
                    <div className="space-y-0.5">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold text-indigo-600">{pt.task.task_number}</span>
                        <span className="text-slate-500 font-mono">({pt.delivery_order.do_number})</span>
                        <span className="px-1.5 py-0.5 rounded text-[10px] bg-slate-100 text-slate-700 font-medium">
                          {pt.task.status}
                        </span>
                      </div>
                      <div className="text-slate-500 text-[11px]">
                        Tujuan: <span className="text-slate-800 font-medium">{pt.delivery_order.recipient_name || pt.delivery_order.customer_name || "-"}</span>
                        {" &bull; "}
                        {pt.items.length} item barang
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <button
                        type="button"
                        onClick={() => setSelectedTaskForPrint(pt)}
                        className="inline-flex items-center gap-1 px-2.5 py-1 text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 transition text-[11px] font-medium"
                      >
                        <Printer className="h-3 w-3 text-indigo-600" />
                        <span>Cetak Picking List</span>
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-2 border-t border-slate-200">
              <button
                type="button"
                onClick={() => setSelectedWaveDetail(null)}
                className="px-4 py-2 text-xs font-medium text-slate-600 hover:bg-slate-100 rounded-lg transition"
              >
                Tutup
              </button>
              {selectedWaveDetail.wave.status === "OPEN" && (
                <button
                  type="button"
                  onClick={() => handleReleaseWave(selectedWaveDetail.wave.id)}
                  className="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-semibold bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition shadow-2xs"
                >
                  <Play className="h-3.5 w-3.5" />
                  <span>Rilis Seluruh Wave</span>
                </button>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ── Modal Buat Wave Baru ── */}
      <WaveReleaseModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        warehouses={warehouses}
        defaultWarehouseId={selectedWarehouseId}
        onWaveUpdated={loadWaves}
      />

      {/* ── Modal Cetak Dokumen Picking List A4 (FE-06) ── */}
      {selectedTaskForPrint && (
        <PrintPickingList
          detail={selectedTaskForPrint}
          onClose={() => setSelectedTaskForPrint(null)}
        />
      )}
    </div>
  )
}
