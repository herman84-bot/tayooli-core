"use client"

import React, { useState, useEffect, useRef, useMemo } from "react"
import {
  ShieldCheck,
  Barcode,
  CheckCircle2,
  AlertTriangle,
  AlertCircle,
  XCircle,
  Search,
  RefreshCw,
  Box,
  Truck,
  Printer,
  ClipboardList,
  UserCheck,
  Building,
  CheckSquare,
  History,
  FileCheck,
  Scale,
  PackageCheck
} from "lucide-react"
import {
  api,
  DeliveryOrder,
  DeliveryOrderItem,
  OutboundQCInspection
} from "@/lib/api"
import { useWarehouses } from "@/hooks/useWMS"
import {
  usePassOutboundQC,
  useRejectOutboundQC,
  useOutboundQCInspections
} from "@/hooks/useWMSManifests"
import { useAuth } from "@/hooks/useAuth"
import { PrintThermalAWB } from "@/components/wms/PrintThermalAWB"

interface OutboundQCSubViewProps {
  warehouseId?: string
}

const QC_STATUS_CONFIG: Record<
  string,
  { label: string; cls: string; bgBadge: string }
> = {
  PACKED: {
    label: "Menunggu QC",
    cls: "text-amber-700 border-amber-200 bg-amber-50",
    bgBadge: "bg-amber-100 text-amber-800",
  },
  QC_PASSED: {
    label: "Lolos QC",
    cls: "text-emerald-700 border-emerald-200 bg-emerald-50",
    bgBadge: "bg-emerald-100 text-emerald-800",
  },
  QC_FAILED: {
    label: "Ditolak (Kemasi Ulang)",
    cls: "text-rose-700 border-rose-200 bg-rose-50",
    bgBadge: "bg-rose-100 text-rose-800",
  },
}

const COMMON_REJECT_REASONS = [
  "Label Resi AWB tidak terbaca / cacat barcode",
  "Kardus / kemasan luar sobek atau basah",
  "Berat paket tidak sesuai dengan fisik",
  "Jumlah item fisik kurang / lebih dari Surat Jalan",
  "Salah varian / SKU produk dalam kemasan",
  "Lakban segel rusak atau terbuka",
]

export function OutboundQCSubView({ warehouseId }: OutboundQCSubViewProps) {
  const { user } = useAuth()
  const { data: warehouses = [] } = useWarehouses()
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>(
    warehouseId || ""
  )

  // Filters & State
  const [filterTab, setFilterTab] = useState<"READY" | "PASSED" | "FAILED" | "ALL">("READY")
  const [searchQuery, setSearchQuery] = useState("")
  const [orders, setOrders] = useState<DeliveryOrder[]>([])
  const [loadingOrders, setLoadingOrders] = useState(false)
  const [ordersError, setOrdersError] = useState<string | null>(null)

  // Selected Order for QC
  const [selectedOrder, setSelectedOrder] = useState<DeliveryOrder | null>(null)
  const [orderItems, setOrderItems] = useState<DeliveryOrderItem[]>([])
  const [loadingItems, setLoadingItems] = useState(false)

  // Scan & QC Verification
  const [scannedAWBInput, setScannedAWBInput] = useState("")
  const [awbVerified, setAwbVerified] = useState(false)
  const [awbMismatch, setAwbMismatch] = useState(false)
  const [qcNotes, setQcNotes] = useState("")

  // Checkboxes
  const [checkPackageCondition, setCheckPackageCondition] = useState(false)
  const [checkWeightMatch, setCheckWeightMatch] = useState(false)

  // Modals & Panels
  const [showRejectModal, setShowRejectModal] = useState(false)
  const [rejectReason, setRejectReason] = useState(COMMON_REJECT_REASONS[0])
  const [customRejectReason, setCustomRejectReason] = useState("")
  const [showThermalModal, setShowThermalModal] = useState(false)
  const [showHistoryModal, setShowHistoryModal] = useState(false)

  // Audio & Feedback
  const [audioEnabled, setAudioEnabled] = useState(true)
  const [feedbackMessage, setFeedbackMessage] = useState<{
    type: "success" | "error"
    text: string
  } | null>(null)

  const awbInputRef = useRef<HTMLInputElement>(null)
  const audioCtxRef = useRef<AudioContext | null>(null)

  // Mutations
  const passQCMutation = usePassOutboundQC()
  const rejectQCMutation = useRejectOutboundQC()
  const { data: historyList = [], refetch: refetchHistory, isLoading: loadingHistory } =
    useOutboundQCInspections({
      warehouse_id: selectedWarehouseId || undefined,
    })

  useEffect(() => {
    if (warehouseId) {
      setSelectedWarehouseId(warehouseId)
    }
  }, [warehouseId])

  useEffect(() => {
    loadOrders()
  }, [selectedWarehouseId])

  useEffect(() => {
    return () => {
      if (audioCtxRef.current && audioCtxRef.current.state !== "closed") {
        audioCtxRef.current.close().catch(() => {})
      }
    }
  }, [])

  const loadOrders = async () => {
    try {
      setLoadingOrders(true)
      setOrdersError(null)
      const res = await api.wms.deliveryOrders.list(selectedWarehouseId || undefined)
      // Only show orders that have reached at least PACKED, or QC_PASSED, or QC_FAILED
      const qcRelevant = (res.data || []).filter(
        (o) =>
          o.status === "PACKED" ||
          o.status === "QC_PASSED" ||
          o.status === "QC_FAILED"
      )
      setOrders(qcRelevant)
    } catch (err: any) {
      setOrdersError(err?.message || "Gagal memuat daftar pesanan untuk QC")
    } finally {
      setLoadingOrders(false)
    }
  }

  const playSound = (isSuccess: boolean) => {
    if (!audioEnabled || typeof window === "undefined") return
    try {
      if (!audioCtxRef.current) {
        const AudioCtx = window.AudioContext || (window as any).webkitAudioContext
        if (AudioCtx) audioCtxRef.current = new AudioCtx()
      }
      const ctx = audioCtxRef.current
      if (!ctx) return
      if (ctx.state === "suspended") {
        ctx.resume().catch(() => {})
      }
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.connect(gain)
      gain.connect(ctx.destination)

      if (isSuccess) {
        osc.frequency.setValueAtTime(880, ctx.currentTime)
        osc.frequency.setValueAtTime(1174.66, ctx.currentTime + 0.08)
        gain.gain.setValueAtTime(0.15, ctx.currentTime)
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.22)
        osc.start(ctx.currentTime)
        osc.stop(ctx.currentTime + 0.22)
      } else {
        osc.frequency.setValueAtTime(220, ctx.currentTime)
        osc.frequency.setValueAtTime(164.81, ctx.currentTime + 0.12)
        gain.gain.setValueAtTime(0.25, ctx.currentTime)
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.35)
        osc.start(ctx.currentTime)
        osc.stop(ctx.currentTime + 0.35)
      }
    } catch {
      // Audio context may fail if no user gesture
    }
  }

  const handleSelectOrder = async (order: DeliveryOrder) => {
    try {
      setLoadingItems(true)
      setFeedbackMessage(null)
      setSelectedOrder(order)
      setScannedAWBInput("")
      setAwbVerified(false)
      setAwbMismatch(false)
      setCheckPackageCondition(false)
      setCheckWeightMatch(false)
      setQcNotes(order.qc_notes || "")

      const res = await api.wms.deliveryOrders.get(order.id)
      setOrderItems(res.items || [])
    } catch (err: any) {
      setFeedbackMessage({
        type: "error",
        text: err?.message || "Gagal memuat rincian item pesanan",
      })
    } finally {
      setLoadingItems(false)
      setTimeout(() => {
        awbInputRef.current?.focus()
      }, 60)
    }
  }

  // Barcode / AWB scan verification
  const handleVerifyAWB = (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedOrder) return

    const input = scannedAWBInput.trim()
    if (!input) return

    const expectedAWB = (selectedOrder.tracking_number || "").trim()
    const expectedDO = (selectedOrder.do_number || "").trim()

    const isMatch =
      (expectedAWB !== "" && input.toLowerCase() === expectedAWB.toLowerCase()) ||
      (expectedDO !== "" && input.toLowerCase() === expectedDO.toLowerCase())

    if (isMatch) {
      setAwbVerified(true)
      setAwbMismatch(false)
      playSound(true)
      setFeedbackMessage({
        type: "success",
        text: `AWB / No. Surat Jalan terverifikasi cocok (${input})`,
      })
    } else {
      setAwbVerified(false)
      setAwbMismatch(true)
      playSound(false)
      setFeedbackMessage({
        type: "error",
        text: `Barcode AWB tidak cocok! Terbaca: ${input}. Harusnya: ${expectedAWB || expectedDO}`,
      })
    }
  }

  // Segregation of duties: Petugas QC != Petugas Packing
  const isSameActorViolation = useMemo(() => {
    if (!selectedOrder || !user) return false
    return !!(selectedOrder.packed_by && selectedOrder.packed_by === user.id)
  }, [selectedOrder, user])

  // Can pass QC?
  const canPassQC = useMemo(() => {
    if (!selectedOrder || selectedOrder.status !== "PACKED") return false
    if (isSameActorViolation) return false
    return true
  }, [selectedOrder, isSameActorViolation])

  const handlePassQC = async () => {
    if (!selectedOrder) return
    if (isSameActorViolation) {
      setFeedbackMessage({
        type: "error",
        text: "Pelanggaran Segregation of Duties: Petugas QC tidak boleh orang yang sama dengan yang mengemas paket!",
      })
      playSound(false)
      return
    }

    try {
      setFeedbackMessage(null)
      const res = await passQCMutation.mutateAsync({
        id: selectedOrder.id,
        payload: {
          scanned_awb: scannedAWBInput.trim() || undefined,
          notes: qcNotes.trim() || undefined,
        },
      })
      playSound(true)
      setFeedbackMessage({
        type: "success",
        text: `Surat Jalan ${selectedOrder.do_number} LOLOS QC OUTBOUND! Status berubah menjadi QC_PASSED (siap masuk Staging/Manifes).`,
      })
      setSelectedOrder(res.data)
      loadOrders()
      refetchHistory()
    } catch (err: any) {
      playSound(false)
      setFeedbackMessage({
        type: "error",
        text: err?.message || "Gagal menyetujui QC Surat Jalan",
      })
    }
  }

  const handleOpenReject = () => {
    setCustomRejectReason("")
    setRejectReason(COMMON_REJECT_REASONS[0])
    setShowRejectModal(true)
  }

  const handleConfirmReject = async () => {
    if (!selectedOrder) return
    if (isSameActorViolation) {
      setFeedbackMessage({
        type: "error",
        text: "Pelanggaran Segregation of Duties: Petugas QC tidak boleh orang yang sama dengan yang mengemas paket!",
      })
      setShowRejectModal(false)
      playSound(false)
      return
    }

    const finalReason = customRejectReason.trim() || rejectReason
    if (!finalReason.trim()) {
      alert("Alasan penolakan wajib diisi!")
      return
    }

    try {
      setFeedbackMessage(null)
      const res = await rejectQCMutation.mutateAsync({
        id: selectedOrder.id,
        payload: {
          reason: finalReason.trim(),
          notes: qcNotes.trim() || undefined,
        },
      })
      setShowRejectModal(false)
      playSound(false)
      setFeedbackMessage({
        type: "error",
        text: `Surat Jalan ${selectedOrder.do_number} DITOLAK QC (QC_FAILED)! Paket harus dikemas ulang.`,
      })
      setSelectedOrder(res.data)
      loadOrders()
      refetchHistory()
    } catch (err: any) {
      setFeedbackMessage({
        type: "error",
        text: err?.message || "Gagal menolak QC Surat Jalan",
      })
    }
  }

  // Filter orders by tab and search
  const filteredOrders = useMemo(() => {
    return orders.filter((o) => {
      if (filterTab === "READY" && o.status !== "PACKED") return false
      if (filterTab === "PASSED" && o.status !== "QC_PASSED") return false
      if (filterTab === "FAILED" && o.status !== "QC_FAILED") return false

      if (searchQuery.trim()) {
        const q = searchQuery.toLowerCase()
        const matchDO = o.do_number.toLowerCase().includes(q)
        const matchCust = (o.customer_name || o.recipient_name || "").toLowerCase().includes(q)
        const matchTracking = (o.tracking_number || "").toLowerCase().includes(q)
        const matchExp = (o.expedition_name || "").toLowerCase().includes(q)
        return matchDO || matchCust || matchTracking || matchExp
      }
      return true
    })
  }, [orders, filterTab, searchQuery])

  const readyCount = useMemo(() => orders.filter((o) => o.status === "PACKED").length, [orders])
  const passedCount = useMemo(() => orders.filter((o) => o.status === "QC_PASSED").length, [orders])
  const failedCount = useMemo(() => orders.filter((o) => o.status === "QC_FAILED").length, [orders])

  return (
    <div className="space-y-6">
      {/* ── Top Bar Filter & Stats ── */}
      <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-2xs flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-3 flex-wrap">
          <div className="flex items-center gap-2">
            <Building className="h-4 w-4 text-slate-500" />
            <span className="text-xs font-semibold text-slate-700">Gudang:</span>
          </div>
          <select
            value={selectedWarehouseId}
            onChange={(e) => setSelectedWarehouseId(e.target.value)}
            className="px-3 py-1.5 bg-slate-50 border border-slate-200 rounded-lg text-xs font-medium text-slate-800 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value="">Semua Gudang</option>
            {warehouses.map((w) => (
              <option key={w.id} value={w.id}>
                {w.name} ({w.code})
              </option>
            ))}
          </select>

          <div className="h-4 w-px bg-slate-200 hidden sm:block" />

          {/* Tab Filter Button Group */}
          <div className="flex items-center gap-1 bg-slate-100 p-1 rounded-lg text-xs font-medium">
            <button
              onClick={() => setFilterTab("READY")}
              className={`px-3 py-1 rounded-md transition ${
                filterTab === "READY"
                  ? "bg-white text-blue-700 font-bold shadow-2xs"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              Perlu QC ({readyCount})
            </button>
            <button
              onClick={() => setFilterTab("PASSED")}
              className={`px-3 py-1 rounded-md transition ${
                filterTab === "PASSED"
                  ? "bg-white text-emerald-700 font-bold shadow-2xs"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              Lolos ({passedCount})
            </button>
            <button
              onClick={() => setFilterTab("FAILED")}
              className={`px-3 py-1 rounded-md transition ${
                filterTab === "FAILED"
                  ? "bg-white text-rose-700 font-bold shadow-2xs"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              Ditolak ({failedCount})
            </button>
            <button
              onClick={() => setFilterTab("ALL")}
              className={`px-3 py-1 rounded-md transition ${
                filterTab === "ALL"
                  ? "bg-white text-slate-900 font-bold shadow-2xs"
                  : "text-slate-600 hover:text-slate-900"
              }`}
            >
              Semua ({orders.length})
            </button>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setShowHistoryModal(true)}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-slate-50 hover:bg-slate-100 border border-slate-200 rounded-lg text-xs font-semibold text-slate-700 transition"
          >
            <History className="h-3.5 w-3.5 text-indigo-600" />
            <span>Riwayat Audit QC</span>
          </button>
          <button
            type="button"
            onClick={loadOrders}
            disabled={loadingOrders}
            className="p-1.5 rounded-lg border border-slate-200 hover:bg-slate-50 text-slate-600 transition"
            title="Segarkan data pesanan"
          >
            <RefreshCw className={`h-4 w-4 ${loadingOrders ? "animate-spin" : ""}`} />
          </button>
        </div>
      </div>

      {/* ── Main Layout: Left Antrean & Right Workbench ── */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        {/* ── Left Column: Antrean Paket Siap QC ── */}
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-2xs">
          <div className="p-3.5 border-b border-slate-200 bg-slate-50/70">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-bold text-slate-900 flex items-center gap-1.5">
                <ShieldCheck className="h-4 w-4 text-blue-600" />
                <span>Antrean Paket ({filteredOrders.length})</span>
              </h3>
              <span className="text-[11px] text-slate-500 tabular-nums">
                {readyCount} menunggu
              </span>
            </div>
            <div className="mt-2.5 relative">
              <Search className="h-3.5 w-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="Cari No. SJ, resi, penerima…"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-8 pr-2.5 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </div>
          </div>

          <div className="divide-y divide-slate-100 max-h-[640px] overflow-y-auto">
            {loadingOrders ? (
              <div className="p-8 text-center text-xs text-slate-400">Memuat paket…</div>
            ) : ordersError ? (
              <div className="p-6 text-center text-xs text-rose-600 space-y-2">
                <AlertCircle className="h-6 w-6 mx-auto text-rose-500" />
                <p className="font-semibold">{ordersError}</p>
                <button
                  type="button"
                  onClick={loadOrders}
                  className="px-2.5 py-1 text-xs border border-rose-300 rounded bg-white text-rose-700"
                >
                  Coba Lagi
                </button>
              </div>
            ) : filteredOrders.length === 0 ? (
              <div className="p-8 text-center text-xs text-slate-400 space-y-1">
                <Box className="h-8 w-8 mx-auto text-slate-300 mb-1" />
                <p className="font-semibold text-slate-700">Tidak ada paket dalam antrean</p>
                <p>Seluruh Surat Jalan yang selesai dikemas telah diinspeksi.</p>
              </div>
            ) : (
              filteredOrders.map((order) => {
                const isSelected = selectedOrder?.id === order.id
                const isMyPack = user && order.packed_by === user.id
                const conf = QC_STATUS_CONFIG[order.status] || {
                  label: order.status,
                  cls: "text-slate-600 border-slate-200 bg-slate-50",
                  bgBadge: "bg-slate-100 text-slate-700",
                }

                return (
                  <div
                    key={order.id}
                    onClick={() => handleSelectOrder(order)}
                    className={`p-3.5 cursor-pointer transition-colors ${
                      isSelected
                        ? "bg-blue-50/70 ring-1 ring-inset ring-blue-600/30"
                        : "hover:bg-slate-50"
                    }`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <span className="font-mono font-bold text-xs text-slate-900">
                            {order.do_number}
                          </span>
                          <span
                            className={`text-[9px] font-bold px-1.5 py-0.5 rounded border ${conf.cls}`}
                          >
                            {conf.label}
                          </span>
                        </div>
                        <div className="text-xs font-semibold text-slate-800 mt-0.5">
                          {order.customer_name || order.recipient_name || "Pelanggan Umum"}
                        </div>
                      </div>
                      <div className="text-right shrink-0">
                        <span className="text-[10px] font-medium text-slate-500 block">
                          {order.expedition_name || "Internal"}
                        </span>
                        {order.package_weight_kg && (
                          <span className="text-[10px] tabular-nums text-slate-700 block">
                            {order.package_weight_kg} kg
                          </span>
                        )}
                      </div>
                    </div>

                    <div className="mt-2 flex items-center justify-between text-[11px] text-slate-500 border-t border-slate-100 pt-1.5">
                      <div>
                        {order.tracking_number ? (
                          <span className="tabular-nums text-slate-700 font-medium">
                            Resi: {order.tracking_number}
                          </span>
                        ) : (
                          <span className="text-amber-600 font-medium">Belum ada resi</span>
                        )}
                      </div>
                      <div className="flex items-center gap-1 text-[10px]">
                        <span>Kemas:</span>
                        <span
                          className={`font-semibold ${
                            isMyPack ? "text-amber-700 underline" : "text-slate-800"
                          }`}
                        >
                          {order.packed_by_name || "Petugas Kemas"}
                        </span>
                      </div>
                    </div>

                    {isMyPack && (
                      <div className="mt-1 flex items-center gap-1 text-[10px] text-amber-700 bg-amber-50 px-2 py-0.5 rounded border border-amber-200">
                        <AlertTriangle className="h-3 w-3 shrink-0" />
                        <span>Dikemas oleh Anda (Segregation of Duties)</span>
                      </div>
                    )}
                  </div>
                )
              })
            )}
          </div>
        </div>

        {/* ── Right Column: Meja Periksa QC Outbound Workbench ── */}
        <div className="lg:col-span-2 bg-white rounded-xl border border-slate-200 overflow-hidden shadow-2xs p-6 space-y-6">
          {!selectedOrder ? (
            <div className="py-24 text-center text-slate-400 space-y-2">
              <PackageCheck className="h-12 w-12 mx-auto text-slate-300" />
              <h4 className="text-sm font-bold text-slate-700">
                Pilih Paket Surat Jalan untuk Mulai Inspeksi QC
              </h4>
              <p className="text-xs max-w-md mx-auto text-slate-400">
                Pilih paket dari daftar di sebelah kiri untuk memverifikasi kondisi fisik kemasan,
                kesesuaian koli/berat, serta pemindaian barcode resi AWB sebelum paket masuk ke area
                staging dan manifes pengiriman.
              </p>
            </div>
          ) : (
            <>
              {/* Header Info Paket */}
              <div className="flex flex-col sm:flex-row sm:items-center justify-between pb-4 border-b border-slate-200 gap-3">
                <div>
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-mono font-black text-lg text-slate-900">
                      {selectedOrder.do_number}
                    </span>
                    <span
                      className={`text-[10px] font-bold px-2 py-0.5 rounded-full border ${
                        QC_STATUS_CONFIG[selectedOrder.status]?.cls ||
                        "bg-slate-100 text-slate-700 border-slate-200"
                      }`}
                    >
                      {QC_STATUS_CONFIG[selectedOrder.status]?.label || selectedOrder.status}
                    </span>
                  </div>
                  <p className="text-xs text-slate-500 mt-1">
                    Penerima: <strong className="text-slate-800">{selectedOrder.customer_name || selectedOrder.recipient_name}</strong>
                    {" • "}
                    Kurir: <strong className="text-slate-800">{selectedOrder.expedition_name || "Internal"}</strong>
                    {selectedOrder.tracking_number && (
                      <>
                        {" • "}
                        Resi AWB: <strong className="font-mono text-blue-700">{selectedOrder.tracking_number}</strong>
                      </>
                    )}
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setAudioEnabled(!audioEnabled)}
                    className={`p-2 rounded-lg border text-xs flex items-center gap-1.5 transition ${
                      audioEnabled
                        ? "bg-blue-50 border-blue-200 text-blue-700"
                        : "bg-slate-50 border-slate-200 text-slate-400"
                    }`}
                    title={audioEnabled ? "Matikan Audio Beeper" : "Aktifkan Audio Beeper"}
                  >
                    <span>{audioEnabled ? "Audio Nyala" : "Audio Mati"}</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setShowThermalModal(true)}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-slate-900 hover:bg-slate-800 text-white rounded-lg text-xs font-semibold shadow-2xs transition"
                  >
                    <Printer className="h-3.5 w-3.5" />
                    <span>Cetak Resi AWB</span>
                  </button>
                </div>
              </div>

              {/* Segregation of Duties Violation Warning */}
              {isSameActorViolation && (
                <div className="p-4 bg-amber-50 border border-amber-300 rounded-xl flex items-start gap-3 text-xs text-amber-900">
                  <AlertTriangle className="h-5 w-5 text-amber-600 mt-0.5 shrink-0" />
                  <div>
                    <h5 className="font-bold text-sm">Peringatan Segregation of Duties (Pemisahan Tugas ISO)</h5>
                    <p className="mt-0.5">
                      Paket ini dikemas oleh akun Anda (<strong>{selectedOrder.packed_by_name || "Anda"}</strong>).
                      Sesuai aturan keamanan pergudangan (ISO/OCA-WMS), verifikasi dan persetujuan QC wajib dilakukan
                      oleh staf atau supervisor yang berbeda.
                    </p>
                    <p className="mt-1 font-semibold text-amber-800">
                      Silakan minta rekan pemeriksa QC lain untuk memeriksa dan menyetujui Surat Jalan ini.
                    </p>
                  </div>
                </div>
              )}

              {/* Feedback Message */}
              {feedbackMessage && (
                <div
                  className={`p-3 rounded-xl border text-xs flex items-center gap-2 ${
                    feedbackMessage.type === "success"
                      ? "bg-emerald-50 text-emerald-800 border-emerald-200"
                      : "bg-rose-50 text-rose-800 border-rose-200"
                  }`}
                >
                  {feedbackMessage.type === "success" ? (
                    <CheckCircle2 className="h-4 w-4 text-emerald-600 shrink-0" />
                  ) : (
                    <AlertCircle className="h-4 w-4 text-rose-600 shrink-0" />
                  )}
                  <span>{feedbackMessage.text}</span>
                </div>
              )}

              {/* ── SECTION 1: Pemindaian Barcode Resi AWB ── */}
              <div className="bg-slate-50/70 p-4 rounded-xl border border-slate-200 space-y-3">
                <div className="flex items-center justify-between">
                  <label className="text-xs font-bold text-slate-800 flex items-center gap-1.5">
                    <Barcode className="h-4 w-4 text-blue-600" />
                    <span>Langkah 1: Pindai Barcode Resi AWB / No. Surat Jalan</span>
                  </label>
                  {awbVerified ? (
                    <span className="inline-flex items-center gap-1 text-[11px] font-bold text-emerald-700 bg-emerald-50 border border-emerald-200 px-2 py-0.5 rounded-full">
                      <CheckCircle2 className="h-3 w-3" /> Resi Cocok
                    </span>
                  ) : awbMismatch ? (
                    <span className="inline-flex items-center gap-1 text-[11px] font-bold text-rose-700 bg-rose-50 border border-rose-200 px-2 py-0.5 rounded-full">
                      <XCircle className="h-3 w-3" /> Tidak Cocok
                    </span>
                  ) : (
                    <span className="text-[11px] text-slate-400">Wajib dipindai</span>
                  )}
                </div>

                <form onSubmit={handleVerifyAWB} className="flex items-center gap-2">
                  <div className="relative flex-1">
                    <input
                      ref={awbInputRef}
                      type="text"
                      placeholder={`Arahkan scanner ke barcode resi (${selectedOrder.tracking_number || selectedOrder.do_number})...`}
                      value={scannedAWBInput}
                      onChange={(e) => {
                        setScannedAWBInput(e.target.value)
                        setAwbVerified(false)
                        setAwbMismatch(false)
                      }}
                      className="w-full pl-3 pr-3 py-2 bg-white border border-slate-300 rounded-lg text-xs font-mono text-slate-900 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                  </div>
                  <button
                    type="submit"
                    className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-xs font-bold shadow-2xs transition"
                  >
                    Verifikasi
                  </button>
                </form>
                <p className="text-[11px] text-slate-500">
                  Target kecocokan: <span className="font-mono font-semibold text-slate-700">{selectedOrder.tracking_number || selectedOrder.do_number}</span>
                </p>
              </div>

              {/* ── SECTION 2: Verifikasi Dimensi, Kemasan & Checklist Fisik ── */}
              <div className="space-y-4">
                <h4 className="text-xs font-bold text-slate-800 flex items-center gap-1.5">
                  <CheckSquare className="h-4 w-4 text-indigo-600" />
                  <span>Langkah 2: Pemeriksaan Fisik & Kemasan Paket</span>
                </h4>

                {/* Packaging Details Summary */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-white p-3.5 rounded-lg border border-slate-200 text-xs">
                  <div>
                    <span className="text-[10px] text-slate-500 block">Berat Paket</span>
                    <span className="tabular-nums font-bold text-slate-900 text-sm">
                      {selectedOrder.package_weight_kg ? `${selectedOrder.package_weight_kg} kg` : "1.0 kg"}
                    </span>
                  </div>
                  <div>
                    <span className="text-[10px] text-slate-500 block">Dimensi (P x L x T)</span>
                    <span className="tabular-nums font-bold text-slate-900 text-sm">
                      {selectedOrder.package_length_cm || 30} x {selectedOrder.package_width_cm || 20} x {selectedOrder.package_height_cm || 15} cm
                    </span>
                  </div>
                  <div>
                    <span className="text-[10px] text-slate-500 block">Jenis Kemasan</span>
                    <span className="font-bold text-slate-900 text-sm">
                      {selectedOrder.packaging_type || "KARTON"}
                    </span>
                  </div>
                  <div>
                    <span className="text-[10px] text-slate-500 block">Petugas Kemas</span>
                    <span className="font-bold text-indigo-700 text-sm">
                      {selectedOrder.packed_by_name || "Petugas Packing"}
                    </span>
                  </div>
                </div>

                {/* Physical Checkboxes */}
                <div className="space-y-2 bg-slate-50/50 p-3 rounded-lg border border-slate-200 text-xs">
                  <label className="flex items-center gap-2 cursor-pointer text-slate-700">
                    <input
                      type="checkbox"
                      checked={checkPackageCondition}
                      onChange={(e) => setCheckPackageCondition(e.target.checked)}
                      className="rounded text-blue-600 focus:ring-blue-500 h-4 w-4"
                    />
                    <span>Kondisi kemasan luar rapi, lakban segel utuh, dan tidak ada kerusakan fisik.</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer text-slate-700">
                    <input
                      type="checkbox"
                      checked={checkWeightMatch}
                      onChange={(e) => setCheckWeightMatch(e.target.checked)}
                      className="rounded text-blue-600 focus:ring-blue-500 h-4 w-4"
                    />
                    <span>Berat aktual timbangan sesuai dengan deklarasi resi kurir.</span>
                  </label>
                </div>

                {/* Items in Pack Table */}
                <div className="border border-slate-200 rounded-lg overflow-hidden">
                  <div className="bg-slate-50 px-3 py-2 border-b border-slate-200 flex items-center justify-between text-xs font-semibold text-slate-700">
                    <span>Daftar Item dalam Paket ({orderItems.length})</span>
                    <span className="text-[11px] text-slate-500">100% Terkemas di Stasiun Kemas</span>
                  </div>
                  {loadingItems ? (
                    <div className="p-4 text-center text-xs text-slate-400">Memuat rincian item…</div>
                  ) : orderItems.length === 0 ? (
                    <div className="p-4 text-center text-xs text-slate-400">Tidak ada data item</div>
                  ) : (
                    <table className="w-full text-left text-xs">
                      <thead className="bg-slate-50/50 text-[10px] text-slate-500 font-semibold border-b border-slate-100">
                        <tr>
                          <th className="py-2 px-3">Nama Produk / SKU</th>
                          <th className="py-2 px-3">Batch & Lokasi</th>
                          <th className="py-2 px-3 text-right">Jumlah Dikemas</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100">
                        {orderItems.map((it) => (
                          <tr key={it.id} className="hover:bg-slate-50/50">
                            <td className="py-2.5 px-3">
                              <div className="font-semibold text-slate-900">{it.product_name}</div>
                              <div className="text-[10px] font-mono text-slate-500">{it.product_sku}</div>
                            </td>
                            <td className="py-2.5 px-3">
                              <span className="inline-block px-1.5 py-0.5 rounded bg-slate-100 font-mono text-[10px] text-slate-700">
                                {it.batch_number || "BATCH-DEFAULT"}
                              </span>
                            </td>
                            <td className="py-2.5 px-3 text-right tabular-nums font-bold text-slate-900">
                              {Number(it.packed_qty || it.quantity).toLocaleString("id-ID")} unit
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              </div>

              {/* ── SECTION 3: Catatan Inspeksi QC ── */}
              <div>
                <label className="text-xs font-semibold text-slate-700 block mb-1">
                  Catatan Petugas QC (Opsional):
                </label>
                <input
                  type="text"
                  placeholder="Contoh: Kemasan rapi, bubble wrap ekstra dipasang, resi terbaca jelas."
                  value={qcNotes}
                  onChange={(e) => setQcNotes(e.target.value)}
                  className="w-full px-3 py-2 bg-white border border-slate-200 rounded-lg text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>

              {/* ── SECTION 4: Tombol Keputusan QC ── */}
              <div className="flex flex-col sm:flex-row items-center justify-between gap-3 pt-4 border-t border-slate-200">
                <div className="text-xs text-slate-500">
                  {selectedOrder.status === "QC_PASSED" ? (
                    <span className="text-emerald-700 font-semibold flex items-center gap-1">
                      <CheckCircle2 className="h-4 w-4" /> Surat Jalan ini sudah lolos QC Outbound
                    </span>
                  ) : selectedOrder.status === "QC_FAILED" ? (
                    <span className="text-rose-700 font-semibold flex items-center gap-1">
                      <AlertCircle className="h-4 w-4" /> Paket dalam status Ditolak QC. Harap kemas ulang di Stasiun Pengemasan.
                    </span>
                  ) : (
                    <span>Pastikan fisik kemasan dan AWB telah terverifikasi sebelum meloloskan.</span>
                  )}
                </div>

                <div className="flex items-center gap-2 w-full sm:w-auto">
                  {selectedOrder.status === "PACKED" && (
                    <>
                      <button
                        type="button"
                        onClick={handleOpenReject}
                        disabled={rejectQCMutation.isPending || isSameActorViolation}
                        className="flex-1 sm:flex-initial inline-flex items-center justify-center gap-1.5 px-4 py-2.5 rounded-lg border border-rose-300 bg-rose-50 hover:bg-rose-100 text-rose-700 font-bold text-xs shadow-2xs transition disabled:opacity-50"
                      >
                        <XCircle className="h-4 w-4" />
                        <span>Tolak QC (Reject)</span>
                      </button>

                      <button
                        type="button"
                        onClick={handlePassQC}
                        disabled={passQCMutation.isPending || isSameActorViolation}
                        className="flex-1 sm:flex-initial inline-flex items-center justify-center gap-1.5 px-5 py-2.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-bold text-xs shadow-2xs transition disabled:opacity-50"
                      >
                        <CheckCircle2 className="h-4 w-4" />
                        <span>{passQCMutation.isPending ? "Memproses..." : "Loloskan QC (QC_PASSED)"}</span>
                      </button>
                    </>
                  )}
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* ── Modal Penolakan QC ── */}
      {showRejectModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-xl space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-slate-200">
              <h3 className="text-sm font-bold text-rose-700 flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-rose-600" />
                <span>Tolak QC & Kemas Ulang</span>
              </h3>
              <button
                type="button"
                onClick={() => setShowRejectModal(false)}
                className="text-slate-400 hover:text-slate-600 text-lg leading-none"
              >
                &times;
              </button>
            </div>

            <p className="text-xs text-slate-600">
              Paket Surat Jalan <strong className="font-mono text-slate-900">{selectedOrder?.do_number}</strong> akan
              ditandai <strong className="text-rose-700">QC_FAILED</strong> dan dikembalikan ke Stasiun Pengemasan
              untuk diperiksa serta dikemas ulang.
            </p>

            <div>
              <label className="text-xs font-bold text-slate-800 block mb-1">
                Pilih Alasan Penolakan:
              </label>
              <select
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                className="w-full px-3 py-2 bg-slate-50 border border-slate-300 rounded-lg text-xs font-medium text-slate-800 focus:outline-none focus:ring-1 focus:ring-rose-500"
              >
                {COMMON_REJECT_REASONS.map((r) => (
                  <option key={r} value={r}>
                    {r}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="text-xs font-bold text-slate-800 block mb-1">
                Atau Tulis Alasan Tambahan:
              </label>
              <textarea
                rows={3}
                placeholder="Rincian kerusakan / ketidaksesuaian..."
                value={customRejectReason}
                onChange={(e) => setCustomRejectReason(e.target.value)}
                className="w-full px-3 py-2 bg-white border border-slate-300 rounded-lg text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-1 focus:ring-rose-500"
              />
            </div>

            <div className="flex items-center justify-end gap-2 pt-3 border-t border-slate-200">
              <button
                type="button"
                onClick={() => setShowRejectModal(false)}
                className="px-4 py-2 border border-slate-300 rounded-lg text-xs font-semibold text-slate-700 hover:bg-slate-50"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={handleConfirmReject}
                disabled={rejectQCMutation.isPending}
                className="px-4 py-2 bg-rose-600 hover:bg-rose-700 text-white rounded-lg text-xs font-bold shadow-2xs transition"
              >
                {rejectQCMutation.isPending ? "Menolak..." : "Konfirmasi Tolak QC"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Modal Riwayat Audit QC ── */}
      {showHistoryModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white rounded-xl max-w-3xl w-full p-6 shadow-xl space-y-4 max-h-[85vh] flex flex-col">
            <div className="flex items-center justify-between pb-3 border-b border-slate-200">
              <h3 className="text-sm font-bold text-slate-900 flex items-center gap-2">
                <History className="h-5 w-5 text-indigo-600" />
                <span>Riwayat Audit Inspeksi QC Outbound</span>
              </h3>
              <button
                type="button"
                onClick={() => setShowHistoryModal(false)}
                className="text-slate-400 hover:text-slate-600 text-lg leading-none"
              >
                &times;
              </button>
            </div>

            <div className="flex-1 overflow-y-auto divide-y divide-slate-100">
              {loadingHistory ? (
                <div className="p-8 text-center text-xs text-slate-400">Memuat riwayat…</div>
              ) : historyList.length === 0 ? (
                <div className="p-8 text-center text-xs text-slate-400">
                  Belum ada catatan inspeksi QC outbound di gudang ini.
                </div>
              ) : (
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 text-[10px] text-slate-500 font-semibold sticky top-0 border-b border-slate-200">
                    <tr>
                      <th className="py-2 px-3">Tanggal & Waktu</th>
                      <th className="py-2 px-3">No. Surat Jalan</th>
                      <th className="py-2 px-3">Pemeriksa QC</th>
                      <th className="py-2 px-3">Status</th>
                      <th className="py-2 px-3">Resi AWB</th>
                      <th className="py-2 px-3">Catatan / Alasan</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {historyList.map((h) => (
                      <tr key={h.id} className="hover:bg-slate-50">
                        <td className="py-2.5 px-3 text-slate-600 tabular-nums text-[11px]">
                          {new Date(h.created_at).toLocaleString("id-ID")}
                        </td>
                        <td className="py-2.5 px-3 font-mono font-bold text-slate-900">
                          {h.do_number}
                        </td>
                        <td className="py-2.5 px-3 text-slate-800 font-medium">
                          {h.inspector_name || "Petugas QC"}
                        </td>
                        <td className="py-2.5 px-3">
                          <span
                            className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                              h.status === "QC_PASSED"
                                ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                                : "bg-rose-50 text-rose-700 border border-rose-200"
                            }`}
                          >
                            {h.status === "QC_PASSED" ? "Lolos" : "Ditolak"}
                          </span>
                        </td>
                        <td className="py-2.5 px-3 font-mono text-slate-600">
                          {h.scanned_awb || "-"}
                        </td>
                        <td className="py-2.5 px-3 text-slate-600 max-w-xs truncate">
                          {h.rejection_reason || h.notes || "-"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
            </div>

            <div className="pt-3 border-t border-slate-200 text-right">
              <button
                type="button"
                onClick={() => setShowHistoryModal(false)}
                className="px-4 py-2 bg-slate-100 hover:bg-slate-200 text-slate-800 rounded-lg text-xs font-semibold"
              >
                Tutup
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Modal Cetak Resi Thermal AWB ── */}
      {showThermalModal && selectedOrder && (
        <PrintThermalAWB
          order={selectedOrder}
          items={orderItems}
          onClose={() => setShowThermalModal(false)}
        />
      )}
    </div>
  )
}
