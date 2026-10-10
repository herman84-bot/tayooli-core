"use client"

import React, { useState, useMemo, useEffect, useRef } from "react"
import Link from "next/link"
import { useManualRefresh } from "@/hooks/useManualRefresh"
import { RefreshButton } from "@/components/ui/RefreshButton"
import {
  AlertTriangle,
  Plus,
  Search,
  CheckCircle2,
  AlertCircle,
  X,
  Building,
  Barcode,
  RefreshCw,
  ArrowLeft,
  ScanLine,
  Layers,
  Calendar,
  User,
  ShieldAlert,
  ArrowRight,
  Flame,
  FileSpreadsheet,
  Lock,
  Volume2,
  VolumeX,
  PackageX,
  Info,
} from "lucide-react"
import {
  useStockScraps,
  useCreateStockScrap,
  useWarehouses,
  useWarehouseLocations,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import { useBarcodeScanner } from "@/hooks/useBarcodeScanner"
import { useAuthStore } from "@/hooks/useAuth"
import { Product, WarehouseLocation, StockScrap } from "@/lib/api"

export default function StockScrapPage() {
  const currentUser = useAuthStore((s) => s.user)
  const currentRole = (currentUser?.role ?? "").toLowerCase()

  const { data: warehouses = [], isLoading: loadingWarehouses, refetch: refetchWarehouses } = useWarehouses()
  const { data: products = [], isLoading: loadingProducts } = useProducts()

  // Filter States
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>("ALL")
  const [searchQuery, setSearchQuery] = useState("")
  const [destinationFilter, setDestinationFilter] = useState<string>("ALL")

  const activeWarehouseFilter = selectedWarehouseId === "ALL" ? null : selectedWarehouseId
  const {
    data: scraps = [],
    isLoading: loadingScraps,
    refetch: refetchScraps,
    dataUpdatedAt,
  } = useStockScraps(activeWarehouseFilter)
  const { refresh, status: refreshStatus, refreshError } = useManualRefresh([refetchScraps, refetchWarehouses])

  // Modal State
  const [showQuarantineModal, setShowQuarantineModal] = useState(false)
  const [modalWarehouseId, setModalWarehouseId] = useState("")
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null)
  const [sourceLocationId, setSourceLocationId] = useState("")
  const [scrapDestinationType, setScrapDestinationType] = useState<"VIRTUAL" | "PHYSICAL">("VIRTUAL")
  const [targetQuarantineLocationId, setTargetQuarantineLocationId] = useState("")
  const [quantityInput, setQuantityInput] = useState("1")
  const [reasonPreset, setReasonPreset] = useState("")
  const [reasonDetail, setReasonDetail] = useState("")
  const [scrapNumberInput, setScrapNumberInput] = useState("")
  const [modalError, setModalError] = useState<string | null>(null)
  const [barcodeSearchInput, setBarcodeSearchInput] = useState("")
  const [soundEnabled, setSoundEnabled] = useState(true)

  // Toast Notification State
  const [toast, setToast] = useState<{
    type: "success" | "error" | "info"
    title: string
    message: string
  } | null>(null)

  // Mutations
  const createScrapMutation = useCreateStockScrap()

  // Selected warehouse locations for modal
  const effectiveModalWhId = modalWarehouseId || (warehouses[0]?.id ?? "")
  const { data: modalLocations = [] } = useWarehouseLocations(effectiveModalWhId)

  // Reference for focus
  const barcodeInputRef = useRef<HTMLInputElement>(null)

  // Auto-dismiss toast after 6 seconds
  useEffect(() => {
    if (toast) {
      const timer = setTimeout(() => setToast(null), 6000)
      return () => clearTimeout(timer)
    }
  }, [toast])

  // Preset reason options
  const PRESET_REASONS = [
    "Kemasan Bocor / Rusak",
    "Kadaluwarsa / Expired",
    "Jatuh saat Forklift / Handling",
    "Cacat Pabrik / Defective",
    "Kotor / Terkontaminasi",
    "Segel Terbuka / Rusak",
  ]

  // Barcode Scan Handler
  const handleBarcodeScanned = (code: string) => {
    const clean = code.trim().toLowerCase()
    if (!clean) return

    // 1. Check if matches location barcode
    const matchedLoc = modalLocations.find(
      (l) => l.barcode?.toLowerCase() === clean || l.code.toLowerCase() === clean
    )
    if (matchedLoc) {
      setSourceLocationId(matchedLoc.id)
      setToast({
        type: "info",
        title: "Lokasi Sumber Terdeteksi",
        message: `Rak asal diatur ke: ${matchedLoc.name} (${matchedLoc.code})`,
      })
      setBarcodeSearchInput("")
      return
    }

    // 2. Check if matches product
    const matchedProd = products.find(
      (p) =>
        p.sku.toLowerCase() === clean ||
        p.name.toLowerCase().includes(clean) ||
        p.id.toLowerCase() === clean
    )
    if (matchedProd) {
      setSelectedProduct(matchedProd)
      setBarcodeSearchInput("")
      setToast({
        type: "success",
        title: "Produk Terdeteksi",
        message: `${matchedProd.name} (${matchedProd.sku})`,
      })
    } else {
      setToast({
        type: "error",
        title: "Barcode Tidak Ditemukan",
        message: `Barcode "${code}" tidak cocok dengan SKU atau Lokasi.`,
      })
    }
  }

  // Hook for hardware barcode scanner
  const { triggerScan } = useBarcodeScanner({
    onScan: handleBarcodeScanned,
    soundFeedback: soundEnabled,
    hapticFeedback: true,
    enabled: showQuarantineModal,
  })

  // Manual Scan Bar Submit
  const handleManualScanSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!barcodeSearchInput.trim()) return
    triggerScan(barcodeSearchInput.trim())
  }

  // Submit Quarantine / Scrap
  const handleSubmitScrap = async (e: React.FormEvent) => {
    e.preventDefault()
    setModalError(null)

    if (!modalWarehouseId) {
      setModalError("Gudang asal wajib dipilih.")
      return
    }
    if (!selectedProduct) {
      setModalError("Pilih atau scan barcode produk yang rusak.")
      return
    }
    if (!sourceLocationId) {
      setModalError("Pilih lokasi rak asal tempat barang rusak diambil.")
      return
    }

    const qty = parseFloat(quantityInput)
    if (isNaN(qty) || qty <= 0) {
      setModalError("Jumlah barang rusak harus berupa angka lebih besar dari 0.")
      return
    }

    const fullReason = [reasonPreset, reasonDetail.trim()].filter(Boolean).join(" - ")
    if (!fullReason) {
      setModalError("Pilih atau isi alasan kerusakan/karantina.")
      return
    }
    if (fullReason.length < 10) {
      setModalError("Alasan pemusnahan/scrap barang harus minimal 10 karakter.")
      return
    }
    if (qty > 10 && currentRole !== "admin" && currentRole !== "owner") {
      setModalError("Pemusnahan stok di atas ambang batas (10 unit) wajib dilakukan oleh pengguna dengan peran Admin atau Owner.")
      return
    }

    try {
      const res = await createScrapMutation.mutateAsync({
        warehouse_id: modalWarehouseId,
        product_id: selectedProduct.id,
        source_location_id: sourceLocationId,
        scrap_location_id:
          scrapDestinationType === "PHYSICAL" && targetQuarantineLocationId
            ? targetQuarantineLocationId
            : undefined,
        quantity: qty,
        reason: fullReason,
        scrap_number: scrapNumberInput.trim() || undefined,
      })

      setShowQuarantineModal(false)
      setSelectedProduct(null)
      setQuantityInput("1")
      setReasonPreset("")
      setReasonDetail("")
      setScrapNumberInput("")

      setToast({
        type: "success",
        title: "Karantina Barang Berhasil Dibukukan",
        message: `Stok telah dipotong dari rak asal ke ${
          scrapDestinationType === "VIRTUAL" ? "ledger virtual @SCRAP" : "rak karantina fisik"
        } (No: ${res.scrap_number || "SCRAP"}).`,
      })

      refetchScraps()
    } catch (err: unknown) {
      setModalError(
        err instanceof Error
          ? err.message
          : "Gagal memproses karantina barang rusak. Pastikan stok mencukupi."
      )
    }
  }

  // Helpers
  const getWarehouseName = (id: string) => {
    const wh = warehouses.find((w) => w.id === id)
    return wh ? `${wh.name} (${wh.code})` : id.slice(0, 8)
  }

  const getProduct = (id: string) => {
    return products.find((p) => p.id === id)
  }

  const getLocation = (id: string) => {
    return modalLocations.find((l) => l.id === id)
  }

  // Filtered Scraps List
  const filteredScraps = useMemo(() => {
    return scraps.filter((s) => {
      const wh = warehouses.find((w) => w.id === s.warehouse_id)
      const prod = products.find((p) => p.id === s.product_id)

      const matchesSearch =
        searchQuery === "" ||
        s.scrap_number.toLowerCase().includes(searchQuery.toLowerCase()) ||
        s.reason.toLowerCase().includes(searchQuery.toLowerCase()) ||
        s.reported_by.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (wh && wh.name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (prod && prod.name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (prod && prod.sku.toLowerCase().includes(searchQuery.toLowerCase()))

      const isVirtual = !s.scrap_location_id || s.scrap_location_id.toLowerCase().includes("scrap")
      const matchesDest =
        destinationFilter === "ALL" ||
        (destinationFilter === "VIRTUAL" && isVirtual) ||
        (destinationFilter === "PHYSICAL" && !isVirtual)

      return matchesSearch && matchesDest
    })
  }, [scraps, searchQuery, destinationFilter, warehouses, products])

  // Summary Metrics
  const metrics = useMemo(() => {
    let totalQty = 0
    let virtualCount = 0
    let physicalCount = 0

    scraps.forEach((s) => {
      totalQty += Number(s.quantity) || 0
      const isVirtual = !s.scrap_location_id || s.scrap_location_id.toLowerCase().includes("scrap")
      if (isVirtual) virtualCount++
      else physicalCount++
    })

    return {
      totalRecords: scraps.length,
      totalQty,
      virtualCount,
      physicalCount,
    }
  }, [scraps])

  return (
    <div className="min-h-screen bg-[#F8FAFC] pb-16">
      {/* ── Toast Notification Banner ── */}
      {toast && (
        <div className="fixed top-4 right-4 z-50 max-w-md w-full p-4 rounded-xl shadow-lg border transition-all duration-300 flex items-start gap-3 bg-white border-slate-200">
          {toast.type === "success" && (
            <div className="p-2 rounded-lg bg-emerald-100 text-emerald-700 shrink-0">
              <CheckCircle2 className="w-5 h-5" />
            </div>
          )}
          {toast.type === "error" && (
            <div className="p-2 rounded-lg bg-rose-100 text-rose-700 shrink-0">
              <AlertCircle className="w-5 h-5" />
            </div>
          )}
          {toast.type === "info" && (
            <div className="p-2 rounded-lg bg-blue-100 text-[#2563EB] shrink-0">
              <Info className="w-5 h-5" />
            </div>
          )}
          <div className="flex-1">
            <h4 className="text-sm font-semibold text-slate-900">{toast.title}</h4>
            <p className="text-xs text-slate-600 mt-0.5">{toast.message}</p>
          </div>
          <button
            onClick={() => setToast(null)}
            className="p-1.5 text-slate-400 hover:text-slate-600 rounded-md transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* ── Top Header Bar ── */}
      <div className="bg-white border-b border-[#E2E8F0] px-4 sm:px-6 py-5 sticky top-0 z-20 shadow-xs">
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <Link
              href="/wms"
              className="p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[48px] min-w-[48px] flex items-center justify-center transition-colors"
              title="Kembali ke Dashboard WMS"
            >
              <ArrowLeft className="w-5 h-5" />
            </Link>
            <div>
              <div className="flex items-center gap-2">
                <AlertTriangle className="w-6 h-6 text-[#EA580C]" />
                <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
                  Barang Rusak & Karantina (Scrap)
                </h1>
              </div>
              <p className="text-xs sm:text-sm text-slate-500">
                Pencatatan barang cacat, pemindahan ke rak karantina, atau pemusnahan ke ledger @SCRAP
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <RefreshButton
              updatedAt={dataUpdatedAt}
              status={refreshStatus}
              error={refreshError}
              onClick={() => void refresh()}
              iconClassName="w-5 h-5"
              className="p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[48px] min-w-[48px] flex items-center justify-center"
            />
            <button
              onClick={() => {
                setModalWarehouseId(warehouses[0]?.id || "")
                setShowQuarantineModal(true)
              }}
              className="flex-1 sm:flex-none flex items-center justify-center gap-2 px-4 py-2.5 bg-[#EA580C] hover:bg-orange-700 text-white font-medium rounded-lg shadow-sm transition-all min-h-[48px]"
            >
              <Plus className="w-5 h-5" />
              <span>Karantina Barang Rusak</span>
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 pt-6 space-y-6">
        {/* ── Summary Stats ── */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
          <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-xs">
            <span className="text-xs font-medium text-slate-500">Total Insiden Scrap</span>
            <div className="flex items-baseline justify-between mt-2">
              <span className="text-2xl font-bold text-slate-900">{metrics.totalRecords}</span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
                Laporan
              </span>
            </div>
          </div>

          <div className="bg-white p-4 rounded-xl border border-rose-200 bg-rose-50/20 shadow-xs">
            <span className="text-xs font-medium text-rose-700">Total Unit Rusak</span>
            <div className="flex items-baseline justify-between mt-2">
              <span className="text-2xl font-bold text-rose-700">{metrics.totalQty}</span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-rose-100 text-rose-800">
                Pcs / Unit
              </span>
            </div>
          </div>

          <div className="bg-white p-4 rounded-xl border border-orange-200 bg-orange-50/20 shadow-xs">
            <span className="text-xs font-medium text-orange-700">Pemusnahan Virtual (@SCRAP)</span>
            <div className="flex items-baseline justify-between mt-2">
              <span className="text-2xl font-bold text-orange-700">{metrics.virtualCount}</span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-orange-100 text-orange-800">
                Write-off
              </span>
            </div>
          </div>

          <div className="bg-white p-4 rounded-xl border border-blue-200 bg-blue-50/20 shadow-xs">
            <span className="text-xs font-medium text-[#2563EB]">Karantina Fisik (Bay)</span>
            <div className="flex items-baseline justify-between mt-2">
              <span className="text-2xl font-bold text-[#2563EB]">{metrics.physicalCount}</span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-blue-100 text-blue-800">
                Isolasi
              </span>
            </div>
          </div>
        </div>

        {/* ── Filters & Search Toolbar ── */}
        <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-xs flex flex-col md:flex-row gap-3 items-stretch md:items-center justify-between">
          <div className="flex flex-col sm:flex-row gap-2 flex-1">
            {/* Search Input */}
            <div className="relative flex-1">
              <Search className="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="Cari nomor scrap, produk, SKU, alasan, atau pelapor..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#EA580C] focus:border-transparent min-h-[48px]"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery("")}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                >
                  <X className="w-4 h-4" />
                </button>
              )}
            </div>

            {/* Warehouse Filter */}
            <div className="w-full sm:w-64">
              <select
                value={selectedWarehouseId}
                onChange={(e) => setSelectedWarehouseId(e.target.value)}
                className="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#EA580C] bg-white min-h-[48px]"
              >
                <option value="ALL">Semua Gudang</option>
                {warehouses.map((w) => (
                  <option key={w.id} value={w.id}>
                    {w.name} ({w.code})
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Destination Filter Tabs */}
          <div className="flex items-center gap-1 overflow-x-auto pb-1 md:pb-0 scrollbar-none">
            {(["ALL", "VIRTUAL", "PHYSICAL"] as const).map((dest) => (
              <button
                key={dest}
                onClick={() => setDestinationFilter(dest)}
                className={`px-3 py-2 rounded-lg text-xs font-semibold whitespace-nowrap transition-colors min-h-[48px] flex items-center ${
                  destinationFilter === dest
                    ? "bg-[#EA580C] text-white"
                    : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                }`}
              >
                {dest === "ALL" && "Semua Destinasi"}
                {dest === "VIRTUAL" && "Virtual @SCRAP"}
                {dest === "PHYSICAL" && "Karantina Fisik"}
              </button>
            ))}
          </div>
        </div>

        {/* ── Scrap Ledger Table ── */}
        <div className="bg-white rounded-xl border border-slate-200 shadow-xs overflow-hidden">
          {loadingScraps ? (
            <div className="p-12 text-center">
              <RefreshCw className="w-8 h-8 text-[#EA580C] animate-spin mx-auto mb-3" />
              <p className="text-sm font-medium text-slate-600">Memuat riwayat barang rusak...</p>
            </div>
          ) : filteredScraps.length === 0 ? (
            <div className="p-12 text-center">
              <PackageX className="w-12 h-12 text-slate-300 mx-auto mb-3" />
              <h3 className="text-base font-semibold text-slate-800">Tidak ada catatan barang rusak</h3>
              <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto">
                {searchQuery || destinationFilter !== "ALL" || selectedWarehouseId !== "ALL"
                  ? "Coba ubah filter atau kata kunci pencarian Anda."
                  : "Belum ada laporan barang rusak yang dicatat ke sistem."}
              </p>
              <button
                onClick={() => {
                  setModalWarehouseId(warehouses[0]?.id || "")
                  setShowQuarantineModal(true)
                }}
                className="mt-4 inline-flex items-center gap-2 px-4 py-2.5 bg-[#EA580C] text-white text-sm font-medium rounded-lg hover:bg-orange-700 min-h-[48px]"
              >
                <Plus className="w-4 h-4" />
                <span>Karantina Barang Rusak Sekarang</span>
              </button>
            </div>
          ) : (
            <div className="divide-y divide-slate-100">
              {/* Desktop View */}
              <div className="hidden md:block overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead className="bg-slate-50 border-b border-slate-200 text-xs uppercase font-semibold text-slate-600 tracking-wider">
                    <tr>
                      <th className="px-6 py-3.5">Nomor Scrap</th>
                      <th className="px-6 py-3.5">Tanggal</th>
                      <th className="px-6 py-3.5">Produk & SKU</th>
                      <th className="px-6 py-3.5">Lokasi Asal</th>
                      <th className="px-6 py-3.5">Tujuan</th>
                      <th className="px-6 py-3.5">Jumlah</th>
                      <th className="px-6 py-3.5">Alasan Kerusakan</th>
                      <th className="px-6 py-3.5">Pelapor</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {filteredScraps.map((s) => {
                      const prod = getProduct(s.product_id)
                      const isVirtual =
                        !s.scrap_location_id || s.scrap_location_id.toLowerCase().includes("scrap")
                      return (
                        <tr key={s.id} className="hover:bg-slate-50/80 transition-colors">
                          <td className="px-6 py-4 font-mono font-semibold text-slate-900 text-xs">
                            {s.scrap_number}
                          </td>
                          <td className="px-6 py-4 text-xs text-slate-500 whitespace-nowrap">
                            {new Date(s.created_at).toLocaleDateString("id-ID", {
                              day: "numeric",
                              month: "short",
                              year: "numeric",
                              hour: "2-digit",
                              minute: "2-digit",
                            })}
                          </td>
                          <td className="px-6 py-4">
                            <div className="font-semibold text-slate-900 text-xs">
                              {prod ? prod.name : `Produk [${s.product_id.slice(0, 8)}]`}
                            </div>
                            {prod && (
                              <div className="text-[11px] font-mono text-slate-500">{prod.sku}</div>
                            )}
                          </td>
                          <td className="px-6 py-4 text-xs">
                            <span className="font-medium text-slate-700 bg-slate-100 px-2 py-0.5 rounded">
                              {s.source_location_id.slice(0, 8)}
                            </span>
                          </td>
                          <td className="px-6 py-4 text-xs">
                            {isVirtual ? (
                              <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-rose-50 text-rose-700 border border-rose-200">
                                <Flame className="w-3 h-3 text-rose-500" />
                                Virtual @SCRAP
                              </span>
                            ) : (
                              <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-blue-50 text-blue-700 border border-blue-200">
                                <ShieldAlert className="w-3 h-3 text-blue-500" />
                                Karantina Fisik
                              </span>
                            )}
                          </td>
                          <td className="px-6 py-4">
                            <span className="inline-flex items-center px-2 py-0.5 rounded font-bold text-xs bg-rose-100 text-rose-800">
                              {s.quantity} pcs
                            </span>
                          </td>
                          <td className="px-6 py-4 text-xs text-slate-700 max-w-xs truncate" title={s.reason}>
                            {s.reason}
                          </td>
                          <td className="px-6 py-4 text-xs text-slate-600">
                            <div className="flex items-center gap-1">
                              <User className="w-3.5 h-3.5 text-slate-400" />
                              <span>{s.reported_by || "Petugas"}</span>
                            </div>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>

              {/* Mobile View */}
              <div className="md:hidden divide-y divide-slate-100">
                {filteredScraps.map((s) => {
                  const prod = getProduct(s.product_id)
                  const isVirtual =
                    !s.scrap_location_id || s.scrap_location_id.toLowerCase().includes("scrap")
                  return (
                    <div key={s.id} className="p-4 space-y-2.5">
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <div className="font-mono font-bold text-xs text-slate-800">{s.scrap_number}</div>
                          <div className="font-semibold text-slate-900 text-sm mt-0.5">
                            {prod ? prod.name : `Produk [${s.product_id.slice(0, 8)}]`}
                          </div>
                        </div>
                        <span className="inline-flex items-center px-2 py-1 rounded font-bold text-xs bg-rose-100 text-rose-800 shrink-0">
                          {s.quantity} unit
                        </span>
                      </div>

                      <div className="flex flex-wrap items-center gap-2 text-xs">
                        <span className="text-slate-500">Asal:</span>
                        <span className="font-medium bg-slate-100 px-1.5 py-0.5 rounded text-slate-700">
                          {s.source_location_id.slice(0, 8)}
                        </span>
                        <span className="text-slate-300">&rarr;</span>
                        {isVirtual ? (
                          <span className="font-semibold text-rose-700 bg-rose-50 px-2 py-0.5 rounded-full border border-rose-200">
                            @SCRAP
                          </span>
                        ) : (
                          <span className="font-semibold text-blue-700 bg-blue-50 px-2 py-0.5 rounded-full border border-blue-200">
                            Karantina Fisik
                          </span>
                        )}
                      </div>

                      <div className="p-2 rounded bg-slate-50 border border-slate-200 text-xs text-slate-700">
                        <strong>Alasan:</strong> {s.reason}
                      </div>

                      <div className="flex items-center justify-between text-[11px] text-slate-400 pt-1">
                        <span>Oleh: {s.reported_by || "Petugas"}</span>
                        <span>
                          {new Date(s.created_at).toLocaleDateString("id-ID", {
                            day: "numeric",
                            month: "short",
                            year: "numeric",
                          })}
                        </span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ── MODAL: Quarantine Damaged Goods (Karantina Barang Rusak) ── */}
      {showQuarantineModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs">
          <div className="bg-white rounded-2xl max-w-xl w-full p-5 sm:p-6 shadow-2xl border border-slate-100 space-y-4 animate-in zoom-in-95 duration-200 max-h-[92vh] overflow-y-auto">
            {/* Modal Header */}
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <div className="p-2 rounded-lg bg-orange-100 text-[#EA580C]">
                  <AlertTriangle className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900">
                    Karantina Barang Rusak (Quarantine Damaged Goods)
                  </h3>
                  <p className="text-xs text-slate-500">
                    Diverifikasi secara atomik dengan advisory lock untuk mencegah stok negatif
                  </p>
                </div>
              </div>
              <button
                onClick={() => setShowQuarantineModal(false)}
                className="p-1.5 rounded-md text-slate-400 hover:text-slate-600"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Error Message */}
            {modalError && (
              <div className="p-3 rounded-lg bg-rose-50 border border-rose-200 text-xs text-rose-700 flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{modalError}</span>
              </div>
            )}

            {/* Fast Barcode Scanning Helper */}
            <form onSubmit={handleManualScanSubmit} className="bg-slate-50 p-3 rounded-xl border border-slate-200 space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="font-semibold text-slate-700 flex items-center gap-1.5">
                  <ScanLine className="w-4 h-4 text-[#EA580C]" />
                  Scan Barcode Produk / Rak Asal
                </span>
                <span className="text-[10px] text-emerald-600 font-medium">Scanner Hardware Aktif</span>
              </div>
              <div className="flex gap-2">
                <input
                  ref={barcodeInputRef}
                  type="text"
                  placeholder="Scan SKU barcode atau kode rak..."
                  value={barcodeSearchInput}
                  onChange={(e) => setBarcodeSearchInput(e.target.value)}
                  className="flex-1 px-3 py-1.5 text-xs rounded-lg border border-slate-300 focus:outline-none focus:ring-2 focus:ring-[#EA580C] min-h-[44px]"
                />
                <button
                  type="submit"
                  className="px-4 py-2 bg-[#EA580C] hover:bg-orange-700 text-white font-semibold text-xs rounded-lg min-h-[44px]"
                >
                  Scan
                </button>
              </div>
            </form>

            <form onSubmit={handleSubmitScrap} className="space-y-4">
              {/* Warehouse selector */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Gudang <span className="text-rose-500">*</span>
                </label>
                <select
                  value={modalWarehouseId}
                  onChange={(e) => {
                    setModalWarehouseId(e.target.value)
                    setSourceLocationId("")
                  }}
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#EA580C] bg-white min-h-[48px]"
                  required
                >
                  <option value="">-- Pilih Gudang --</option>
                  {warehouses.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name} ({w.code})
                    </option>
                  ))}
                </select>
              </div>

              {/* Product Selector */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Produk Master Rusak <span className="text-rose-500">*</span>
                </label>
                <select
                  value={selectedProduct?.id || ""}
                  onChange={(e) => {
                    const found = products.find((p) => p.id === e.target.value) || null
                    setSelectedProduct(found)
                  }}
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#EA580C] bg-white min-h-[48px]"
                  required
                >
                  <option value="">-- Pilih Produk --</option>
                  {products.map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name} [{p.sku}]
                    </option>
                  ))}
                </select>
              </div>

              {/* Source Rack Location */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Lokasi Rak Asal (Source Rack) <span className="text-rose-500">*</span>
                </label>
                <select
                  value={sourceLocationId}
                  onChange={(e) => setSourceLocationId(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#EA580C] bg-white min-h-[48px]"
                  required
                >
                  <option value="">-- Pilih Rak Asal --</option>
                  {modalLocations
                    .filter((l) => l.type === "INTERNAL")
                    .map((l) => (
                      <option key={l.id} value={l.id}>
                        {l.name} ({l.code})
                      </option>
                    ))}
                </select>
              </div>

              {/* Destination Type Selector */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Tujuan Karantina / Tindakan <span className="text-rose-500">*</span>
                </label>
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={() => setScrapDestinationType("VIRTUAL")}
                    className={`p-3 rounded-xl border text-left transition-all min-h-[48px] ${
                      scrapDestinationType === "VIRTUAL"
                        ? "border-rose-400 bg-rose-50/60 ring-2 ring-rose-300"
                        : "border-slate-200 bg-white hover:bg-slate-50"
                    }`}
                  >
                    <div className="flex items-center gap-1.5 font-bold text-xs text-rose-800">
                      <Flame className="w-4 h-4 text-rose-500" />
                      <span>Virtual @SCRAP</span>
                    </div>
                    <p className="text-[11px] text-slate-500 mt-1">
                      Pemusnahan fisik / Pengurangan stok langsung
                    </p>
                  </button>

                  <button
                    type="button"
                    onClick={() => setScrapDestinationType("PHYSICAL")}
                    className={`p-3 rounded-xl border text-left transition-all min-h-[48px] ${
                      scrapDestinationType === "PHYSICAL"
                        ? "border-blue-400 bg-blue-50/60 ring-2 ring-blue-300"
                        : "border-slate-200 bg-white hover:bg-slate-50"
                    }`}
                  >
                    <div className="flex items-center gap-1.5 font-bold text-xs text-[#2563EB]">
                      <ShieldAlert className="w-4 h-4 text-blue-500" />
                      <span>Rak Karantina Fisik</span>
                    </div>
                    <p className="text-[11px] text-slate-500 mt-1">
                      Isolasi fisik untuk inspeksi / retur supplier
                    </p>
                  </button>
                </div>
              </div>

              {/* If Physical Quarantine: Target Location Picker */}
              {scrapDestinationType === "PHYSICAL" && (
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">
                    Rak Karantina Fisik Tujuan <span className="text-rose-500">*</span>
                  </label>
                  <select
                    value={targetQuarantineLocationId}
                    onChange={(e) => setTargetQuarantineLocationId(e.target.value)}
                    className="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#2563EB] bg-white min-h-[48px]"
                    required
                  >
                    <option value="">-- Pilih Rak Karantina --</option>
                    {modalLocations.map((l) => (
                      <option key={l.id} value={l.id}>
                        {l.name} ({l.code}) - {l.type}
                      </option>
                    ))}
                  </select>
                </div>
              )}

              {/* Quantity */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Jumlah Rusak (Quantity) <span className="text-rose-500">*</span>
                </label>
                <div className="flex items-center gap-2">
                  <input
                    type="number"
                    min="1"
                    step="any"
                    value={quantityInput}
                    onChange={(e) => setQuantityInput(e.target.value)}
                    className="flex-1 px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#EA580C] min-h-[48px]"
                    required
                  />
                  <div className="flex gap-1">
                    {[1, 5, 10].map((num) => (
                      <button
                        key={num}
                        type="button"
                        onClick={() => setQuantityInput(String(num))}
                        className="px-3 py-2 rounded-lg border border-slate-200 text-xs font-semibold hover:bg-slate-100 min-h-[48px]"
                      >
                        {num}
                      </button>
                    ))}
                  </div>
                </div>
              </div>

              {/* Preset Reason Chips */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Kategori Kerusakan <span className="text-rose-500">*</span>
                </label>
                <div className="flex flex-wrap gap-1.5 mb-2">
                  {PRESET_REASONS.map((p) => (
                    <button
                      key={p}
                      type="button"
                      onClick={() => setReasonPreset(p)}
                      className={`px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all ${
                        reasonPreset === p
                          ? "bg-[#EA580C] text-white"
                          : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                      }`}
                    >
                      {p}
                    </button>
                  ))}
                </div>
                <textarea
                  rows={2}
                  placeholder="Detail kronologi kerusakan (misal: pallet miring saat loading truk)..."
                  value={reasonDetail}
                  onChange={(e) => setReasonDetail(e.target.value)}
                  className="w-full px-3 py-2 text-xs rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-[#EA580C]"
                />
              </div>

              {/* Custom Scrap Number (Optional) */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Nomor Referensi Scrap (Opsional)
                </label>
                <input
                  type="text"
                  placeholder="Otomatis: SCRAP-YYYYMMDD-XXX"
                  value={scrapNumberInput}
                  onChange={(e) => setScrapNumberInput(e.target.value)}
                  className="w-full px-3 py-2 text-xs rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-[#EA580C]"
                />
              </div>

              {/* Advisory Lock Information */}
              <div className="p-3 rounded-xl bg-amber-50 border border-amber-200 text-[11px] text-amber-800 flex items-start gap-2">
                <Lock className="w-4 h-4 text-amber-600 shrink-0 mt-0.5" />
                <span>
                  Sistem menerapkan <strong>pg_advisory_xact_lock</strong> pada kombinasi SKU dan lokasi rak.
                  Jika stok fisik di rak kurang dari jumlah yang diminta, transaksi akan dibatalkan secara atomik.
                </span>
              </div>

              {/* Actions */}
              <div className="flex items-center gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowQuarantineModal(false)}
                  className="flex-1 py-2.5 px-4 rounded-lg border border-slate-200 text-slate-700 font-semibold text-sm hover:bg-slate-50 min-h-[48px]"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createScrapMutation.isPending}
                  className="flex-1 py-2.5 px-4 rounded-lg bg-[#EA580C] hover:bg-orange-700 text-white font-bold text-sm shadow-sm transition-all min-h-[48px] flex items-center justify-center gap-2"
                >
                  {createScrapMutation.isPending ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      <span>Memproses Karantina...</span>
                    </>
                  ) : (
                    <>
                      <CheckCircle2 className="w-4 h-4" />
                      <span>Karantina & Catat Scrap</span>
                    </>
                  )}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
