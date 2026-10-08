"use client"

import React, { useState, useMemo } from "react"
import {
  Truck,
  Plus,
  Search,
  CheckCircle2,
  Clock,
  Printer,
  Barcode,
  Send,
  AlertTriangle,
  X,
  FileCheck,
  Check,
  RefreshCw,
  Eye,
  ShieldCheck,
  ChevronRight,
  PackageCheck,
  Building,
  Phone,
  Calendar,
} from "lucide-react"
import {
  useShippingManifests,
  useCreateShippingManifest,
  useShippingManifestDetail,
  useScanLoadingDO,
  useDispatchShippingManifest,
} from "@/hooks/useWMSManifests"
import { useDeliveryOrders, useWarehouses } from "@/hooks/useWMS"
import {
  api,
  type ShippingManifest,
  type ShippingManifestDetail,
  type ShippingManifestStatus,
  type DeliveryOrder,
} from "@/lib/api"
import { SignatureCanvas } from "@/components/wms/SignatureCanvas"
import { PrintShippingManifest } from "@/components/wms/PrintShippingManifest"

export interface ShippingManifestsPanelProps {
  warehouseId?: string
}

const COMMON_EXPEDITIONS = [
  "JNE Trucking (JTR)",
  "J&T Express",
  "SiCepat Cargo",
  "Shopee Xpress (SPX)",
  "GoSend Instant / Sameday",
  "GrabExpress",
  "Anteraja",
  "Indah Logistik",
  "Dakota Cargo",
  "Truk Internal / Armada Sendiri",
]

export function ShippingManifestsPanel({ warehouseId }: ShippingManifestsPanelProps) {
  // Filters
  const [statusFilter, setStatusFilter] = useState<string>("ALL")
  const [searchQuery, setSearchQuery] = useState("")

  // Queries
  const { data: warehouses = [] } = useWarehouses()
  const {
    data: manifests = [],
    isLoading: isManifestsLoading,
    refetch,
    isFetching,
  } = useShippingManifests({
    warehouse_id: warehouseId,
    status: statusFilter === "ALL" ? undefined : statusFilter,
  })

  // Mutations
  const createManifestMutation = useCreateShippingManifest()
  const scanLoadingMutation = useScanLoadingDO()
  const dispatchMutation = useDispatchShippingManifest()

  // Toast
  const [toast, setToast] = useState<{ type: "success" | "error"; message: string } | null>(null)

  // ── Create Manifest Modal State ──
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [selectedModalWhId, setSelectedModalWhId] = useState<string>(warehouseId || "")
  const [expeditionName, setExpeditionName] = useState(COMMON_EXPEDITIONS[0])
  const [customExpedition, setCustomExpedition] = useState("")
  const [driverName, setDriverName] = useState("")
  const [vehiclePlate, setVehiclePlate] = useState("")
  const [driverPhone, setDriverPhone] = useState("")
  const [notes, setNotes] = useState("")
  const [selectedDoIds, setSelectedDoIds] = useState<string[]>([])
  const [createError, setCreateError] = useState<string | null>(null)

  // Fetch DOs for Create Modal (filter PACKED/CONFIRMED unassigned)
  const effectiveWhForDO = selectedModalWhId || warehouseId || (warehouses[0]?.id ?? null)
  const { data: allDOs = [], isLoading: isDOsLoading } = useDeliveryOrders(effectiveWhForDO)

  // Filter DOs ready for staging/manifest (only PACKED DOs ready for dispatch manifest)
  const availableDOs = useMemo(() => {
    return allDOs.filter((d) => d.status === "PACKED")
  }, [allDOs])

  // ── Detail / Loading Scan Drawer State ──
  const [activeDetailId, setActiveDetailId] = useState<string | null>(null)
  const { data: activeDetail, isLoading: isDetailLoading, refetch: refetchDetail } =
    useShippingManifestDetail(activeDetailId || undefined)
  const [scanBarcodeInput, setScanBarcodeInput] = useState("")
  const [scanMessage, setScanMessage] = useState<{ type: "success" | "error"; text: string } | null>(
    null
  )

  // ── Dispatch & Signature Modal State ──
  const [dispatchManifestTarget, setDispatchManifestTarget] = useState<ShippingManifest | null>(null)
  const [signatureSvg, setSignatureSvg] = useState<string>("")
  const [dispatchNotes, setDispatchNotes] = useState("")
  const [dispatchError, setDispatchError] = useState<string | null>(null)

  // ── Print Modal State ──
  const [printDetail, setPrintDetail] = useState<ShippingManifestDetail | null>(null)
  const [loadingPrintId, setLoadingPrintId] = useState<string | null>(null)

  // Summary Metrics
  const metrics = useMemo(() => {
    const total = manifests.length
    const staged = manifests.filter((m) => m.status === "STAGED").length
    const loaded = manifests.filter((m) => m.status === "LOADED").length
    const dispatched = manifests.filter((m) => m.status === "DISPATCHED").length
    return { total, staged, loaded, dispatched }
  }, [manifests])

  // Filtered Manifests by search
  const filteredManifests = useMemo(() => {
    return manifests.filter((m) => {
      if (!searchQuery.trim()) return true
      const q = searchQuery.toLowerCase()
      return (
        m.manifest_number.toLowerCase().includes(q) ||
        m.expedition_name.toLowerCase().includes(q) ||
        m.driver_name.toLowerCase().includes(q) ||
        m.vehicle_plate.toLowerCase().includes(q)
      )
    })
  }, [manifests, searchQuery])

  // Open Create Modal
  const handleOpenCreateModal = () => {
    setSelectedModalWhId(warehouseId || warehouses[0]?.id || "")
    setExpeditionName(COMMON_EXPEDITIONS[0])
    setCustomExpedition("")
    setDriverName("")
    setVehiclePlate("")
    setDriverPhone("")
    setNotes("")
    setSelectedDoIds([])
    setCreateError(null)
    setShowCreateModal(true)
  }

  // Toggle DO selection
  const toggleSelectDO = (doId: string) => {
    setSelectedDoIds((prev) =>
      prev.includes(doId) ? prev.filter((id) => id !== doId) : [...prev, doId]
    )
  }

  const toggleSelectAllDOs = () => {
    if (selectedDoIds.length === availableDOs.length) {
      setSelectedDoIds([])
    } else {
      setSelectedDoIds(availableDOs.map((d) => d.id))
    }
  }

  // Calculate live sum of packages & weight for modal
  const selectedDOSummary = useMemo(() => {
    const selected = availableDOs.filter((d) => selectedDoIds.includes(d.id))
    const totalKoli = selected.length
    const totalWeight = selected.reduce(
      (sum, d) => sum + Number(d.package_weight_kg || 0),
      0
    )
    return { totalKoli, totalWeight }
  }, [availableDOs, selectedDoIds])

  // Submit Create Manifest
  const handleCreateManifest = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreateError(null)

    const finalExp = expeditionName === "Lainnya (Ketik Manual)" ? customExpedition.trim() : expeditionName
    if (!finalExp) {
      setCreateError("Nama ekspedisi wajib diisi.")
      return
    }
    if (!driverName.trim()) {
      setCreateError("Nama pengemudi / kurir wajib diisi.")
      return
    }
    if (!vehiclePlate.trim()) {
      setCreateError("Nomor plat kendaraan wajib diisi.")
      return
    }
    if (!selectedModalWhId) {
      setCreateError("Gudang asal wajib dipilih.")
      return
    }
    if (selectedDoIds.length === 0) {
      setCreateError("Pilih minimal 1 Surat Jalan untuk dimasukkan ke dalam manifest.")
      return
    }

    try {
      const res = await createManifestMutation.mutateAsync({
        warehouse_id: selectedModalWhId,
        expedition_name: finalExp,
        driver_name: driverName.trim(),
        vehicle_plate: vehiclePlate.trim(),
        driver_phone: driverPhone.trim() || undefined,
        delivery_order_ids: selectedDoIds,
        notes: notes.trim() || undefined,
      })
      setShowCreateModal(false)
      setToast({
        type: "success",
        message: `Manifest ${res.data.manifest_number} berhasil dibuat dengan ${selectedDoIds.length} Surat Jalan.`,
      })
      refetch()
    } catch (err: unknown) {
      setCreateError(err instanceof Error ? err.message : "Gagal membuat manifest pengiriman.")
    }
  }

  // Perform Loading Scan
  const handleLoadingScan = async (e?: React.FormEvent, directBarcode?: string) => {
    if (e) e.preventDefault()
    if (!activeDetailId) return
    const code = (directBarcode || scanBarcodeInput).trim()
    if (!code) return

    setScanMessage(null)
    try {
      await scanLoadingMutation.mutateAsync({
        id: activeDetailId,
        barcode: code,
      })
      setScanBarcodeInput("")
      setScanMessage({
        type: "success",
        text: `Surat Jalan "${code}" berhasil dipindai dan dimuat ke truk.`,
      })
      refetchDetail()
      refetch()
    } catch (err: unknown) {
      setScanMessage({
        type: "error",
        text: err instanceof Error ? err.message : "Barcode tidak cocok atau gagal memuat.",
      })
    }
  }

  // Open Dispatch Modal
  const handleOpenDispatchModal = (m: ShippingManifest) => {
    setDispatchManifestTarget(m)
    setSignatureSvg("")
    setDispatchNotes(m.notes || "")
    setDispatchError(null)
  }

  // Submit Dispatch
  const handleConfirmDispatch = async () => {
    if (!dispatchManifestTarget) return
    if (!signatureSvg.trim()) {
      setDispatchError("Tanda tangan digital pengemudi wajib dilengkapi sebelum dispatch.")
      return
    }

    setDispatchError(null)
    try {
      await dispatchMutation.mutateAsync({
        id: dispatchManifestTarget.id,
        driver_signature_svg: signatureSvg,
        notes: dispatchNotes.trim() || undefined,
      })
      const targetNum = dispatchManifestTarget.manifest_number
      setDispatchManifestTarget(null)
      if (activeDetailId === dispatchManifestTarget.id) {
        setActiveDetailId(null)
      }
      setToast({
        type: "success",
        message: `Manifest ${targetNum} berhasil di-dispatch! Armada telah resmi diberangkatkan.`,
      })
      refetch()
    } catch (err: unknown) {
      setDispatchError(err instanceof Error ? err.message : "Gagal melakukan dispatch manifest.")
    }
  }

  // Open Print Manifest
  const handleOpenPrint = async (manifestId: string) => {
    setLoadingPrintId(manifestId)
    try {
      const res = await api.wms.manifests.get(manifestId)
      if (res.data) {
        setPrintDetail(res.data)
      } else {
        setToast({ type: "error", message: "Gagal memuat data cetak manifest." })
      }
    } catch (err: unknown) {
      setToast({
        type: "error",
        message: err instanceof Error ? err.message : "Gagal memuat detail cetak manifest.",
      })
    } finally {
      setLoadingPrintId(null)
    }
  }

  // Status Badge Helper
  const renderStatusBadge = (status: ShippingManifestStatus) => {
    switch (status) {
      case "STAGED":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-amber-50 text-amber-700 border border-amber-200">
            <Clock className="w-3 h-3 text-amber-500" />
            STAGED (Siap Muat)
          </span>
        )
      case "LOADED":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-200">
            <PackageCheck className="w-3 h-3 text-blue-500" />
            LOADED (Terpindai)
          </span>
        )
      case "DISPATCHED":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
            <CheckCircle2 className="w-3 h-3 text-emerald-600" />
            DISPATCHED (Berangkat)
          </span>
        )
      case "CANCELLED":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-rose-50 text-rose-700 border border-rose-200">
            <X className="w-3 h-3 text-rose-500" />
            BATAL
          </span>
        )
      default:
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-700 border border-slate-200">
            {status}
          </span>
        )
    }
  }

  return (
    <div className="space-y-6">
      {/* Toast Notification */}
      {toast && (
        <div
          className={`fixed top-4 right-4 z-50 flex items-center gap-2 px-4 py-3 rounded-xl shadow-lg border text-sm transition-all ${
            toast.type === "success"
              ? "bg-emerald-50 border-emerald-200 text-emerald-800"
              : "bg-rose-50 border-rose-200 text-rose-800"
          }`}
        >
          {toast.type === "success" ? (
            <CheckCircle2 className="w-5 h-5 text-emerald-600 shrink-0" />
          ) : (
            <AlertTriangle className="w-5 h-5 text-rose-600 shrink-0" />
          )}
          <span>{toast.message}</span>
          <button
            onClick={() => setToast(null)}
            className="ml-2 text-slate-400 hover:text-slate-700 p-0.5"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* Metrics Summary */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-xl border border-slate-200 p-4 shadow-xs">
          <div className="flex items-center justify-between text-slate-500 text-xs mb-1">
            <span>Total Manifest</span>
            <Truck className="h-4 w-4 text-slate-400" />
          </div>
          <div className="text-2xl font-bold text-slate-900">{metrics.total}</div>
          <div className="text-[11px] text-slate-400 mt-1">Seluruh manifest pengiriman</div>
        </div>

        <div className="bg-white rounded-xl border border-slate-200 p-4 shadow-xs">
          <div className="flex items-center justify-between text-amber-600 text-xs mb-1">
            <span>Staged (Siap Muat)</span>
            <Clock className="h-4 w-4 text-amber-500" />
          </div>
          <div className="text-2xl font-bold text-amber-600">{metrics.staged}</div>
          <div className="text-[11px] text-slate-400 mt-1">Menunggu pemindaian ke bak truk</div>
        </div>

        <div className="bg-white rounded-xl border border-slate-200 p-4 shadow-xs">
          <div className="flex items-center justify-between text-blue-600 text-xs mb-1">
            <span>Loaded (Terpindai Penuh)</span>
            <PackageCheck className="h-4 w-4 text-blue-500" />
          </div>
          <div className="text-2xl font-bold text-blue-600">{metrics.loaded}</div>
          <div className="text-[11px] text-slate-400 mt-1">Siap serah terima &amp; TTD sopir</div>
        </div>

        <div className="bg-white rounded-xl border border-slate-200 p-4 shadow-xs">
          <div className="flex items-center justify-between text-emerald-600 text-xs mb-1">
            <span>Dispatched (Berangkat)</span>
            <CheckCircle2 className="h-4 w-4 text-emerald-600" />
          </div>
          <div className="text-2xl font-bold text-emerald-600">{metrics.dispatched}</div>
          <div className="text-[11px] text-slate-400 mt-1">Telah diserahterimakan ke armada</div>
        </div>
      </div>

      {/* Control Bar: Filters, Search & Create Button */}
      <div className="bg-white rounded-xl border border-slate-200 p-4 flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3 shadow-xs">
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Status Tabs */}
          <div className="flex items-center gap-1 bg-slate-100 p-0.5 rounded-lg border border-slate-200 text-xs">
            {[
              { label: "Semua", value: "ALL" },
              { label: "Staged", value: "STAGED" },
              { label: "Loaded", value: "LOADED" },
              { label: "Dispatched", value: "DISPATCHED" },
            ].map((tab) => (
              <button
                key={tab.value}
                onClick={() => setStatusFilter(tab.value)}
                className={`px-3 py-1.5 rounded-md font-medium transition ${
                  statusFilter === tab.value
                    ? "bg-white text-slate-900 shadow-xs font-semibold"
                    : "text-slate-600 hover:text-slate-900"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Search Box */}
          <div className="relative flex-1 sm:w-64">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-slate-400" />
            <input
              type="text"
              placeholder="Cari manifest, kurir, plat..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-1.5 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="p-2 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 flex items-center justify-center transition disabled:opacity-50"
            title="Segarkan data"
          >
            <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
          </button>

          <button
            onClick={handleOpenCreateModal}
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-lg font-semibold text-xs bg-blue-600 text-white hover:bg-blue-700 active:scale-95 transition shadow-xs"
          >
            <Plus className="h-4 w-4" />
            <span>Buat Manifest Baru</span>
          </button>
        </div>
      </div>

      {/* Manifest Table */}
      <div className="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-xs">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs border-collapse">
            <thead className="bg-slate-50 border-b border-slate-200 text-slate-600 uppercase font-semibold">
              <tr>
                <th className="py-3 px-4">No. Manifest</th>
                <th className="py-3 px-4">Ekspedisi</th>
                <th className="py-3 px-4">Sopir / Plat</th>
                <th className="py-3 px-4 text-center">Koli</th>
                <th className="py-3 px-4 text-right">Berat</th>
                <th className="py-3 px-4 text-center">Status</th>
                <th className="py-3 px-4">Tanggal Dibuat</th>
                <th className="py-3 px-4 text-right">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {isManifestsLoading ? (
                <tr>
                  <td colSpan={8} className="py-8 text-center text-slate-500">
                    <RefreshCw className="w-5 h-5 animate-spin mx-auto mb-2 text-slate-400" />
                    Memuat daftar manifest...
                  </td>
                </tr>
              ) : filteredManifests.length === 0 ? (
                <tr>
                  <td colSpan={8} className="py-12 text-center text-slate-500">
                    <Truck className="w-8 h-8 mx-auto mb-2 text-slate-300" />
                    <p className="font-medium text-slate-700">Belum ada manifest pengiriman</p>
                    <p className="text-slate-400 text-[11px] mt-0.5">
                      Kelompokkan Surat Jalan yang sudah dikemas menjadi manifest ekspedisi.
                    </p>
                  </td>
                </tr>
              ) : (
                filteredManifests.map((m) => (
                  <tr key={m.id} className="hover:bg-slate-50/80 transition">
                    <td className="py-3 px-4 font-mono font-semibold text-slate-900">
                      {m.manifest_number}
                    </td>
                    <td className="py-3 px-4 text-slate-800 font-medium">
                      {m.expedition_name}
                    </td>
                    <td className="py-3 px-4 text-slate-700">
                      <div className="font-medium text-slate-800">{m.driver_name}</div>
                      <div className="text-[11px] font-mono text-slate-500">{m.vehicle_plate}</div>
                    </td>
                    <td className="py-3 px-4 text-center font-bold text-slate-800">
                      {m.total_packages}
                    </td>
                    <td className="py-3 px-4 text-right font-mono text-slate-700">
                      {Number(m.total_weight_kg || 0).toFixed(2)} kg
                    </td>
                    <td className="py-3 px-4 text-center">
                      {renderStatusBadge(m.status)}
                    </td>
                    <td className="py-3 px-4 text-slate-500">
                      <div>{new Date(m.created_at).toLocaleDateString("id-ID")}</div>
                      <div className="text-[10px] text-slate-400">
                        {new Date(m.created_at).toLocaleTimeString("id-ID", {
                          hour: "2-digit",
                          minute: "2-digit",
                        })}
                      </div>
                    </td>
                    <td className="py-3 px-4 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        {/* Detail / Loading Scan */}
                        <button
                          type="button"
                          onClick={() => setActiveDetailId(m.id)}
                          className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-100 transition text-[11px] font-medium"
                          title="Periksa detail & scan koli"
                        >
                          <Barcode className="w-3.5 h-3.5 text-blue-600" />
                          <span>Scan / Detail</span>
                        </button>

                        {/* Dispatch Button if not dispatched */}
                        {m.status !== "DISPATCHED" && m.status !== "CANCELLED" && (
                          <button
                            type="button"
                            onClick={() => handleOpenDispatchModal(m)}
                            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-emerald-600 text-white hover:bg-emerald-700 transition text-[11px] font-medium shadow-xs"
                            title="Tanda tangan & berangkatkan armada"
                          >
                            <Send className="w-3.5 h-3.5" />
                            <span>Dispatch</span>
                          </button>
                        )}

                        {/* Cetak Manifest */}
                        <button
                          type="button"
                          onClick={() => handleOpenPrint(m.id)}
                          disabled={loadingPrintId === m.id}
                          className="p-1.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition"
                          title="Cetak Manifest A4"
                        >
                          <Printer className={`w-3.5 h-3.5 ${loadingPrintId === m.id ? "animate-spin" : ""}`} />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* ────────────────────────────────────────────────────────── */}
      {/* ── CREATE MANIFEST MODAL ───────────────────────────────── */}
      {/* ────────────────────────────────────────────────────────── */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs overflow-y-auto">
          <div className="bg-white rounded-2xl max-w-3xl w-full shadow-2xl border border-slate-200 overflow-hidden flex flex-col max-h-[92vh]">
            <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between bg-slate-50">
              <div className="flex items-center gap-2">
                <div className="p-2 bg-blue-50 text-blue-600 rounded-lg">
                  <Truck className="h-5 w-5" />
                </div>
                <div>
                  <h3 className="font-bold text-slate-900 text-base">Buat Manifest Ekspedisi Baru</h3>
                  <p className="text-xs text-slate-500">
                    Kelompokkan beberapa Surat Jalan menjadi satu berkas serah terima pengiriman
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setShowCreateModal(false)}
                className="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <form onSubmit={handleCreateManifest} className="flex-1 overflow-y-auto p-6 space-y-5">
              {createError && (
                <div className="flex items-center gap-2 p-3 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700">
                  <AlertTriangle className="h-4 w-4 shrink-0 text-rose-600" />
                  <span>{createError}</span>
                </div>
              )}

              {/* Form Metadata */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {/* Warehouse */}
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Gudang Pengirim <span className="text-rose-500">*</span>
                  </label>
                  <select
                    value={selectedModalWhId}
                    onChange={(e) => setSelectedModalWhId(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-800 focus:ring-2 focus:ring-blue-500"
                    required
                  >
                    {warehouses.map((w) => (
                      <option key={w.id} value={w.id}>
                        {w.name} ({w.code})
                      </option>
                    ))}
                  </select>
                </div>

                {/* Expedition Name */}
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Jasa Ekspedisi / Kurir <span className="text-rose-500">*</span>
                  </label>
                  <select
                    value={expeditionName}
                    onChange={(e) => setExpeditionName(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-800 focus:ring-2 focus:ring-blue-500"
                  >
                    {COMMON_EXPEDITIONS.map((exp) => (
                      <option key={exp} value={exp}>
                        {exp}
                      </option>
                    ))}
                    <option value="Lainnya (Ketik Manual)">Lainnya (Ketik Manual)</option>
                  </select>
                  {expeditionName === "Lainnya (Ketik Manual)" && (
                    <input
                      type="text"
                      placeholder="Masukkan nama ekspedisi..."
                      value={customExpedition}
                      onChange={(e) => setCustomExpedition(e.target.value)}
                      className="mt-1.5 w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-1.5 text-xs text-slate-800 focus:ring-2 focus:ring-blue-500"
                      required
                    />
                  )}
                </div>

                {/* Driver Name */}
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Nama Sopir / Pengemudi <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="text"
                    placeholder="Contoh: Budi Santoso"
                    value={driverName}
                    onChange={(e) => setDriverName(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-800 focus:ring-2 focus:ring-blue-500"
                    required
                  />
                </div>

                {/* Vehicle Plate */}
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Nomor Polisi Armada <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="text"
                    placeholder="Contoh: B 9482 TYN"
                    value={vehiclePlate}
                    onChange={(e) => setVehiclePlate(e.target.value.toUpperCase())}
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-xs font-mono uppercase text-slate-800 focus:ring-2 focus:ring-blue-500"
                    required
                  />
                </div>

                {/* Driver Phone */}
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    No. Telepon Sopir (Opsional)
                  </label>
                  <input
                    type="tel"
                    placeholder="Contoh: 081234567890"
                    value={driverPhone}
                    onChange={(e) => setDriverPhone(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-800 focus:ring-2 focus:ring-blue-500"
                  />
                </div>

                {/* Notes */}
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Catatan Khusus (Opsional)
                  </label>
                  <input
                    type="text"
                    placeholder="Contoh: Pengiriman batch sore dermaga 2"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-800 focus:ring-2 focus:ring-blue-500"
                  />
                </div>
              </div>

              {/* Delivery Orders Selection Table */}
              <div className="space-y-2 pt-2 border-t border-slate-200">
                <div className="flex items-center justify-between">
                  <div>
                    <h4 className="text-xs font-bold text-slate-800">
                      Pilih Surat Jalan Siap Muat ({availableDOs.length} tersedia)
                    </h4>
                    <p className="text-[11px] text-slate-500">
                      Centang Surat Jalan yang akan diserahkan ke armada pengangkut
                    </p>
                  </div>
                  <button
                    type="button"
                    onClick={toggleSelectAllDOs}
                    className="text-xs font-medium text-blue-600 hover:text-blue-800"
                  >
                    {selectedDoIds.length === availableDOs.length && availableDOs.length > 0
                      ? "Batal Pilih Semua"
                      : "Pilih Semua"}
                  </button>
                </div>

                <div className="border border-slate-200 rounded-xl overflow-hidden max-h-56 overflow-y-auto">
                  <table className="w-full text-left text-xs border-collapse">
                    <thead className="bg-slate-50 border-b border-slate-200 text-slate-600 sticky top-0">
                      <tr>
                        <th className="py-2 px-3 w-8">
                          <input
                            type="checkbox"
                            checked={
                              availableDOs.length > 0 &&
                              selectedDoIds.length === availableDOs.length
                            }
                            onChange={toggleSelectAllDOs}
                            className="rounded text-blue-600"
                          />
                        </th>
                        <th className="py-2 px-3">No. DO</th>
                        <th className="py-2 px-3">Pelanggan</th>
                        <th className="py-2 px-3 text-right">Berat (kg)</th>
                        <th className="py-2 px-3 text-center">Kemasan</th>
                        <th className="py-2 px-3 text-center">Status</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100">
                      {isDOsLoading ? (
                        <tr>
                          <td colSpan={6} className="py-4 text-center text-slate-400">
                            Memuat daftar Surat Jalan...
                          </td>
                        </tr>
                      ) : availableDOs.length === 0 ? (
                        <tr>
                          <td colSpan={6} className="py-6 text-center text-slate-400">
                            Tidak ada Surat Jalan siap muat (status PACKED) di gudang ini.
                          </td>
                        </tr>
                      ) : (
                        availableDOs.map((d) => {
                          const isChecked = selectedDoIds.includes(d.id)
                          return (
                            <tr
                              key={d.id}
                              onClick={() => toggleSelectDO(d.id)}
                              className={`cursor-pointer transition ${
                                isChecked ? "bg-blue-50/60 font-medium" : "hover:bg-slate-50"
                              }`}
                            >
                              <td className="py-2 px-3" onClick={(e) => e.stopPropagation()}>
                                <input
                                  type="checkbox"
                                  checked={isChecked}
                                  onChange={() => toggleSelectDO(d.id)}
                                  className="rounded text-blue-600"
                                />
                              </td>
                              <td className="py-2 px-3 font-mono font-semibold text-slate-900">
                                {d.do_number}
                              </td>
                              <td className="py-2 px-3 text-slate-700">
                                {d.customer_name || d.recipient_name || "-"}
                              </td>
                              <td className="py-2 px-3 text-right font-mono text-slate-800">
                                {Number(d.package_weight_kg || 0).toFixed(2)}
                              </td>
                              <td className="py-2 px-3 text-center text-slate-600">
                                {d.packaging_type || "Koli"}
                              </td>
                              <td className="py-2 px-3 text-center">
                                <span className="px-2 py-0.5 rounded-full text-[10px] font-semibold bg-amber-50 text-amber-700 border border-amber-200">
                                  {d.status}
                                </span>
                              </td>
                            </tr>
                          )
                        })
                      )}
                    </tbody>
                  </table>
                </div>

                {/* Summary bar */}
                <div className="bg-slate-100 rounded-lg p-3 flex items-center justify-between text-xs text-slate-700">
                  <span>
                    Terpilih: <strong>{selectedDOSummary.totalKoli} Surat Jalan</strong>
                  </span>
                  <span>
                    Estimasi Berat Total:{" "}
                    <strong>{selectedDOSummary.totalWeight.toFixed(2)} kg</strong>
                  </span>
                </div>
              </div>

              {/* Submit Buttons */}
              <div className="pt-3 border-t border-slate-200 flex items-center justify-end gap-2.5">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 rounded-lg border border-slate-200 text-xs font-medium text-slate-700 hover:bg-slate-100 transition"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createManifestMutation.isPending || selectedDoIds.length === 0}
                  className="inline-flex items-center gap-1.5 px-5 py-2 rounded-lg text-xs font-semibold bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50 transition shadow-xs"
                >
                  {createManifestMutation.isPending ? (
                    <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Check className="w-3.5 h-3.5" />
                  )}
                  <span>Buat Manifest</span>
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ────────────────────────────────────────────────────────── */}
      {/* ── DETAIL & LOADING SCAN DRAWER / MODAL ────────────────── */}
      {/* ────────────────────────────────────────────────────────── */}
      {activeDetailId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs overflow-y-auto">
          <div className="bg-white rounded-2xl max-w-4xl w-full shadow-2xl border border-slate-200 overflow-hidden flex flex-col max-h-[92vh]">
            <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between bg-slate-50">
              <div className="flex items-center gap-2">
                <div className="p-2 bg-blue-50 text-blue-600 rounded-lg">
                  <Barcode className="h-5 w-5" />
                </div>
                <div>
                  <h3 className="font-bold text-slate-900 text-base">
                    Loading Scan &amp; Verifikasi Koli Truk
                  </h3>
                  <p className="text-xs text-slate-500 font-mono">
                    Manifest: {activeDetail?.manifest.manifest_number || "..."} &bull;{" "}
                    {activeDetail?.manifest.expedition_name}
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => {
                  setActiveDetailId(null)
                  setScanMessage(null)
                }}
                className="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-6 space-y-6">
              {isDetailLoading || !activeDetail ? (
                <div className="py-12 text-center text-slate-400">
                  <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-slate-400" />
                  Memuat detail manifest...
                </div>
              ) : (
                <>
                  {/* Meta Information Cards */}
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-slate-50 p-3 rounded-xl border border-slate-200 text-xs">
                    <div>
                      <span className="text-slate-400 block text-[11px]">Sopir / Kurir</span>
                      <strong className="text-slate-800">{activeDetail.manifest.driver_name}</strong>
                    </div>
                    <div>
                      <span className="text-slate-400 block text-[11px]">No. Polisi Armada</span>
                      <strong className="text-slate-800 font-mono">
                        {activeDetail.manifest.vehicle_plate}
                      </strong>
                    </div>
                    <div>
                      <span className="text-slate-400 block text-[11px]">Total Koli &amp; Berat</span>
                      <strong className="text-slate-800">
                        {activeDetail.items.length} koli /{" "}
                        {Number(activeDetail.manifest.total_weight_kg || 0).toFixed(1)} kg
                      </strong>
                    </div>
                    <div>
                      <span className="text-slate-400 block text-[11px]">Status Manifest</span>
                      <div className="mt-0.5">{renderStatusBadge(activeDetail.manifest.status)}</div>
                    </div>
                  </div>

                  {/* Loading Scan Progress Bar */}
                  {(() => {
                    const totalItems = activeDetail.items.length
                    const scannedItems = activeDetail.items.filter((it) => it.scanned).length
                    const pct = totalItems > 0 ? Math.round((scannedItems / totalItems) * 100) : 0
                    const isAllLoaded = scannedItems === totalItems && totalItems > 0

                    return (
                      <div className="space-y-2 bg-blue-50/50 p-4 rounded-xl border border-blue-100">
                        <div className="flex items-center justify-between text-xs">
                          <span className="font-semibold text-slate-800 flex items-center gap-1.5">
                            <PackageCheck className="w-4 h-4 text-blue-600" />
                            Progress Pemuatan ke Truk:
                          </span>
                          <span className="font-mono font-bold text-blue-700">
                            {scannedItems} / {totalItems} Koli ({pct}%)
                          </span>
                        </div>
                        <div className="w-full bg-slate-200 h-2.5 rounded-full overflow-hidden">
                          <div
                            className={`h-full transition-all duration-300 rounded-full ${
                              isAllLoaded ? "bg-emerald-500" : "bg-blue-600"
                            }`}
                            style={{ width: `${pct}%` }}
                          />
                        </div>
                        {isAllLoaded && (
                          <div className="text-[11px] text-emerald-700 font-semibold flex items-center gap-1 mt-1">
                            <CheckCircle2 className="w-3.5 h-3.5" />
                            Seluruh koli telah terpindai lengkap dan siap di-dispatch!
                          </div>
                        )}
                      </div>
                    )
                  })()}

                  {/* Loading Scan Input Field */}
                  {activeDetail.manifest.status !== "DISPATCHED" && (
                    <form
                      onSubmit={(e) => handleLoadingScan(e)}
                      className="flex items-stretch gap-2"
                    >
                      <div className="relative flex-1">
                        <Barcode className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
                        <input
                          type="text"
                          placeholder="Scan barcode Surat Jalan (DO/xxx) atau ketik manual..."
                          value={scanBarcodeInput}
                          onChange={(e) => setScanBarcodeInput(e.target.value)}
                          className="w-full pl-9 pr-3 py-2 bg-slate-50 border border-slate-300 rounded-xl text-xs font-mono text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                          autoFocus
                        />
                      </div>
                      <button
                        type="submit"
                        disabled={scanLoadingMutation.isPending || !scanBarcodeInput.trim()}
                        className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl text-xs font-semibold bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-40 transition shadow-xs"
                      >
                        {scanLoadingMutation.isPending ? (
                          <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                        ) : (
                          <Check className="w-3.5 h-3.5" />
                        )}
                        <span>Verifikasi Muat</span>
                      </button>
                    </form>
                  )}

                  {/* Scan feedback alert */}
                  {scanMessage && (
                    <div
                      className={`flex items-center gap-2 p-3 rounded-xl text-xs ${
                        scanMessage.type === "success"
                          ? "bg-emerald-50 text-emerald-800 border border-emerald-200"
                          : "bg-rose-50 text-rose-800 border border-rose-200"
                      }`}
                    >
                      {scanMessage.type === "success" ? (
                        <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                      ) : (
                        <AlertTriangle className="w-4 h-4 text-rose-600 shrink-0" />
                      )}
                      <span>{scanMessage.text}</span>
                    </div>
                  )}

                  {/* Table of Items */}
                  <div className="border border-slate-200 rounded-xl overflow-hidden">
                    <table className="w-full text-left text-xs border-collapse">
                      <thead className="bg-slate-50 border-b border-slate-200 text-slate-600 uppercase font-semibold">
                        <tr>
                          <th className="py-2.5 px-3 w-10 text-center">No</th>
                          <th className="py-2.5 px-3">No. DO</th>
                          <th className="py-2.5 px-3">Pelanggan</th>
                          <th className="py-2.5 px-3">Kota Tujuan</th>
                          <th className="py-2.5 px-3 text-right">Berat</th>
                          <th className="py-2.5 px-3 text-center">Status Muat</th>
                          <th className="py-2.5 px-3 text-right">Aksi</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-100">
                        {activeDetail.items.map((item, idx) => (
                          <tr
                            key={item.delivery_order_id || idx}
                            className={`transition ${
                              item.scanned ? "bg-emerald-50/30" : "hover:bg-slate-50"
                            }`}
                          >
                            <td className="py-2 px-3 text-center text-slate-400 font-mono">
                              {idx + 1}
                            </td>
                            <td className="py-2 px-3 font-mono font-semibold text-slate-900">
                              {item.do_number}
                            </td>
                            <td className="py-2 px-3 text-slate-800">{item.customer_name}</td>
                            <td className="py-2 px-3 text-slate-600">
                              {item.destination_city || "-"}
                            </td>
                            <td className="py-2 px-3 text-right font-mono text-slate-700">
                              {Number(item.package_weight_kg || 0).toFixed(2)} kg
                            </td>
                            <td className="py-2 px-3 text-center">
                              {item.scanned ? (
                                <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200">
                                  <CheckCircle2 className="w-3 h-3 text-emerald-600" />
                                  Termuat
                                </span>
                              ) : (
                                <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-amber-700 bg-amber-50 px-2 py-0.5 rounded-full border border-amber-200">
                                  <Clock className="w-3 h-3 text-amber-600" />
                                  Menunggu
                                </span>
                              )}
                            </td>
                            <td className="py-2 px-3 text-right">
                              {!item.scanned && activeDetail.manifest.status !== "DISPATCHED" && (
                                <button
                                  type="button"
                                  onClick={() => handleLoadingScan(undefined, item.do_number)}
                                  disabled={scanLoadingMutation.isPending}
                                  className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg text-[11px] font-medium bg-blue-50 text-blue-700 hover:bg-blue-100 border border-blue-200 transition"
                                >
                                  <Check className="w-3 h-3" />
                                  <span>Tandai Termuat</span>
                                </button>
                              )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Action Footer */}
                  <div className="pt-4 border-t border-slate-200 flex items-center justify-between">
                    <button
                      type="button"
                      onClick={() => handleOpenPrint(activeDetail.manifest.id)}
                      className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-xs font-medium border border-slate-200 text-slate-700 hover:bg-slate-100 transition"
                    >
                      <Printer className="w-4 h-4" />
                      <span>Cetak Lembar Manifest</span>
                    </button>

                    {activeDetail.manifest.status !== "DISPATCHED" && (
                      <button
                        type="button"
                        onClick={() => handleOpenDispatchModal(activeDetail.manifest)}
                        className="inline-flex items-center gap-1.5 px-5 py-2 rounded-lg text-xs font-semibold bg-emerald-600 text-white hover:bg-emerald-700 shadow-xs transition"
                      >
                        <Send className="w-4 h-4" />
                        <span>Lanjut ke Dispatch &amp; TTD</span>
                      </button>
                    )}
                  </div>
                </>
              )}
            </div>
          </div>
        </div>
      )}

      {/* ────────────────────────────────────────────────────────── */}
      {/* ── DISPATCH & SIGNATURE MODAL ──────────────────────────── */}
      {/* ────────────────────────────────────────────────────────── */}
      {dispatchManifestTarget && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs overflow-y-auto">
          <div className="bg-white rounded-2xl max-w-xl w-full shadow-2xl border border-slate-200 overflow-hidden flex flex-col">
            <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between bg-slate-50">
              <div className="flex items-center gap-2">
                <div className="p-2 bg-emerald-50 text-emerald-600 rounded-lg">
                  <Send className="h-5 w-5" />
                </div>
                <div>
                  <h3 className="font-bold text-slate-900 text-base">
                    Dispatch &amp; Serah Terima Pengemudi
                  </h3>
                  <p className="text-xs text-slate-500 font-mono">
                    {dispatchManifestTarget.manifest_number} &bull; {dispatchManifestTarget.expedition_name}
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setDispatchManifestTarget(null)}
                className="text-slate-400 hover:text-slate-600 p-1.5 rounded-lg"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              {dispatchError && (
                <div className="flex items-center gap-2 p-3 bg-rose-50 border border-rose-200 rounded-xl text-xs text-rose-700">
                  <AlertTriangle className="h-4 w-4 shrink-0 text-rose-600" />
                  <span>{dispatchError}</span>
                </div>
              )}

              {/* Manifest Brief Info */}
              <div className="bg-slate-50 p-3 rounded-xl border border-slate-200 text-xs space-y-1">
                <div className="flex justify-between">
                  <span className="text-slate-500">Nama Pengemudi:</span>
                  <strong className="text-slate-800">{dispatchManifestTarget.driver_name}</strong>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">No. Polisi Armada:</span>
                  <strong className="text-slate-800 font-mono">
                    {dispatchManifestTarget.vehicle_plate}
                  </strong>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-500">Total Koli / Berat:</span>
                  <strong className="text-slate-800">
                    {dispatchManifestTarget.total_packages} koli /{" "}
                    {Number(dispatchManifestTarget.total_weight_kg || 0).toFixed(2)} kg
                  </strong>
                </div>
              </div>

              {/* Signature Canvas */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1.5">
                  Tanda Tangan Digital Pengemudi / Kurir <span className="text-rose-500">*</span>
                </label>
                <SignatureCanvas
                  onSave={(svgOrDataUrl) => {
                    setSignatureSvg(svgOrDataUrl)
                    setToast({
                      type: "success",
                      message: "Tanda tangan pengemudi berhasil direkam.",
                    })
                  }}
                  height={160}
                />
                {signatureSvg && (
                  <div className="mt-1.5 text-[11px] text-emerald-600 font-medium flex items-center gap-1">
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    Tanda tangan tersimpan dan siap diverifikasi.
                  </div>
                )}
              </div>

              {/* Notes */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Catatan Pengiriman Tambahan (Opsional)
                </label>
                <input
                  type="text"
                  placeholder="Contoh: Diserahkan dalam kondisi segel utuh"
                  value={dispatchNotes}
                  onChange={(e) => setDispatchNotes(e.target.value)}
                  className="w-full bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-800 focus:ring-2 focus:ring-emerald-500"
                />
              </div>

              {/* Submit Buttons */}
              <div className="pt-3 border-t border-slate-200 flex items-center justify-end gap-2.5">
                <button
                  type="button"
                  onClick={() => setDispatchManifestTarget(null)}
                  className="px-4 py-2 rounded-lg border border-slate-200 text-xs font-medium text-slate-700 hover:bg-slate-100 transition"
                >
                  Batal
                </button>
                <button
                  type="button"
                  onClick={handleConfirmDispatch}
                  disabled={dispatchMutation.isPending || !signatureSvg}
                  className="inline-flex items-center gap-1.5 px-5 py-2 rounded-lg text-xs font-semibold bg-emerald-600 text-white hover:bg-emerald-700 disabled:opacity-50 transition shadow-xs"
                >
                  {dispatchMutation.isPending ? (
                    <RefreshCw className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Send className="w-3.5 h-3.5" />
                  )}
                  <span>Konfirmasi &amp; Dispatch</span>
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* ────────────────────────────────────────────────────────── */}
      {/* ── PRINT SHIPPING MANIFEST MODAL ───────────────────────── */}
      {/* ────────────────────────────────────────────────────────── */}
      {printDetail && (
        <PrintShippingManifest
          manifestDetail={printDetail}
          onClose={() => setPrintDetail(null)}
        />
      )}
    </div>
  )
}
