"use client"

import React, { useState, useMemo, useEffect, useRef } from "react"
import Link from "next/link"
import {
  ClipboardCheck,
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
  ArrowRight,
  TrendingDown,
  TrendingUp,
  MinusCircle,
  Check,
  Sparkles,
  Lock,
  Volume2,
  VolumeX,
} from "lucide-react"
import {
  useStockOpnames,
  useStockOpname,
  useCreateStockOpname,
  useAddOpnameItem,
  useCompleteStockOpname,
  useWarehouses,
  useWarehouseLocations,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import { useBarcodeScanner } from "@/hooks/useBarcodeScanner"
import {
  StockOpname,
  StockOpnameItem,
  StockOpnameStatus,
  Product,
  WarehouseLocation,
} from "@/lib/api"

export default function StockOpnamePage() {
  const { data: warehouses = [], isLoading: loadingWarehouses, refetch: refetchWarehouses } = useWarehouses()
  const { data: products = [], isLoading: loadingProducts } = useProducts()

  // Filters
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>("ALL")
  const [statusFilter, setStatusFilter] = useState<string>("ALL")
  const [searchQuery, setSearchQuery] = useState("")

  const activeWarehouseFilter = selectedWarehouseId === "ALL" ? null : selectedWarehouseId
  const {
    data: opnames = [],
    isLoading: loadingOpnames,
    refetch: refetchOpnames,
  } = useStockOpnames(activeWarehouseFilter)

  // Drawer / Active Counting Session State
  const [activeOpnameId, setActiveOpnameId] = useState<string | null>(null)
  const {
    data: activeOpnameDetail,
    isLoading: loadingActiveOpname,
    refetch: refetchActiveOpname,
  } = useStockOpname(activeOpnameId)

  // Modals state
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showPostingConfirmModal, setShowPostingConfirmModal] = useState(false)

  // Notification Toast state
  const [toast, setToast] = useState<{
    type: "success" | "error" | "info"
    title: string
    message: string
  } | null>(null)

  // Mutations
  const createOpnameMutation = useCreateStockOpname()
  const addItemMutation = useAddOpnameItem()
  const completeOpnameMutation = useCompleteStockOpname()

  // Form states for Create Opname Modal
  const [newWarehouseId, setNewWarehouseId] = useState("")
  const [newOpnameNumber, setNewOpnameNumber] = useState("")
  const [newNotes, setNewNotes] = useState("")
  const [createModalError, setCreateModalError] = useState<string | null>(null)

  // Counting session inputs
  const [scanInput, setScanInput] = useState("")
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null)
  const [selectedLocationId, setSelectedLocationId] = useState<string>("")
  const [physicalCountInput, setPhysicalCountInput] = useState<string>("1")
  const [systemCountInput, setSystemCountInput] = useState<number>(0)
  const [countNotes, setCountNotes] = useState("")
  const [countingError, setCountingError] = useState<string | null>(null)
  const [soundEnabled, setSoundEnabled] = useState(true)
  const [drawerSearch, setDrawerSearch] = useState("")

  // Focus input ref
  const scanInputRef = useRef<HTMLInputElement>(null)

  // Auto-dismiss toast after 6 seconds
  useEffect(() => {
    if (toast) {
      const timer = setTimeout(() => setToast(null), 6000)
      return () => clearTimeout(timer)
    }
  }, [toast])

  // Active Opname header & warehouse
  const activeOpname = activeOpnameDetail?.opname ?? opnames.find((o) => o.id === activeOpnameId) ?? null
  const activeOpnameWarehouseId = activeOpname?.warehouse_id ?? null
  const { data: opnameLocations = [] } = useWarehouseLocations(activeOpnameWarehouseId)

  // Default internal location when locations load
  const defaultLocationId = useMemo(() => {
    if (opnameLocations.length === 0) return ""
    const firstInternal = opnameLocations.find((l) => l.type === "INTERNAL") || opnameLocations[0]
    return firstInternal?.id || ""
  }, [opnameLocations])

  const effectiveLocationId = selectedLocationId || defaultLocationId

  // Barcode detection handler
  const handleBarcodeDetected = (code: string) => {
    const cleanCode = code.trim().toLowerCase()
    if (!cleanCode) return

    // 1. Check if matches location barcode
    const matchedLoc = opnameLocations.find(
      (l) => l.barcode?.toLowerCase() === cleanCode || l.code.toLowerCase() === cleanCode
    )
    if (matchedLoc) {
      setSelectedLocationId(matchedLoc.id)
      setToast({
        type: "info",
        title: "Lokasi Rak Dipilih",
        message: `Lokasi penghitungan dialihkan ke: ${matchedLoc.name} (${matchedLoc.code})`,
      })
      setScanInput("")
      return
    }

    // 2. Check if matches product SKU or name
    const matchedProd = products.find(
      (p) =>
        p.sku.toLowerCase() === cleanCode ||
        p.name.toLowerCase().includes(cleanCode) ||
        p.id.toLowerCase() === cleanCode
    )

    if (matchedProd) {
      setSelectedProduct(matchedProd)
      // Check if product already exists in counted items to prefill or suggest
      const existingItem = activeOpnameDetail?.items.find((item) => item.product_id === matchedProd.id)
      if (existingItem) {
        setSystemCountInput(Number(existingItem.system_qty) || 0)
        setPhysicalCountInput(String(Number(existingItem.physical_qty) + 1))
      } else {
        // Mock standard theoretical stock from warehouse
        setSystemCountInput(50)
        setPhysicalCountInput("1")
      }
      setScanInput("")
      setToast({
        type: "success",
        title: "Barcode Terdeteksi",
        message: `Produk: ${matchedProd.name} (${matchedProd.sku})`,
      })
    } else {
      setToast({
        type: "error",
        title: "Produk Tidak Ditemukan",
        message: `Barcode "${code}" tidak cocok dengan SKU produk maupun lokasi rak.`,
      })
    }
  }

  // Hook for hardware scanner & programmatic trigger
  const { triggerScan } = useBarcodeScanner({
    onScan: handleBarcodeDetected,
    soundFeedback: soundEnabled,
    hapticFeedback: true,
    enabled: !!activeOpnameId && activeOpname?.status !== "COMPLETED" && activeOpname?.status !== "CANCELLED",
  })

  // Manual Scan Input Submit
  const handleManualScanSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!scanInput.trim()) return
    triggerScan(scanInput.trim())
  }

  // Real-time discrepancy calculation
  const parsedPhysicalQty = parseFloat(physicalCountInput) || 0
  const realTimeDiscrepancy = parsedPhysicalQty - systemCountInput

  // Record Count Line Item
  const handleRecordCount = async (e: React.FormEvent) => {
    e.preventDefault()
    setCountingError(null)

    if (!activeOpnameId) return
    if (!selectedProduct) {
      setCountingError("Pilih atau scan barcode produk terlebih dahulu.")
      return
    }
    if (!effectiveLocationId) {
      setCountingError("Pilih lokasi rak tempat barang dihitung.")
      return
    }
    if (isNaN(parsedPhysicalQty) || parsedPhysicalQty < 0) {
      setCountingError("Jumlah fisik harus berupa angka lebih besar atau sama dengan 0.")
      return
    }

    try {
      await addItemMutation.mutateAsync({
        id: activeOpnameId,
        data: {
          product_id: selectedProduct.id,
          location_id: effectiveLocationId,
          physical_qty: parsedPhysicalQty,
          notes: countNotes.trim() || undefined,
        },
      })

      setToast({
        type: "success",
        title: "Penghitungan Disimpan",
        message: `${selectedProduct.name}: Fisik ${parsedPhysicalQty} (Selisih: ${realTimeDiscrepancy >= 0 ? "+" : ""}${realTimeDiscrepancy})`,
      })

      // Reset product & count inputs for next fast scan
      setSelectedProduct(null)
      setPhysicalCountInput("1")
      setCountNotes("")
      setScanInput("")
      if (scanInputRef.current) {
        scanInputRef.current.focus()
      }
    } catch (err: unknown) {
      setCountingError(err instanceof Error ? err.message : "Gagal menyimpan item opname.")
    }
  }

  // Finalize / Post Adjustment to @LOSS
  const handleConfirmPosting = async () => {
    if (!activeOpnameId) return
    setShowPostingConfirmModal(false)

    try {
      await completeOpnameMutation.mutateAsync(activeOpnameId)
      setToast({
        type: "success",
        title: "Posting Penyesuaian Berhasil (@LOSS)",
        message: "Sesi opname telah dikunci. Mutasi penyeimbang otomatis diposting ke ledger virtual @LOSS.",
      })
      refetchOpnames()
      refetchActiveOpname()
    } catch (err: unknown) {
      setToast({
        type: "error",
        title: "Gagal Posting Opname",
        message: err instanceof Error ? err.message : "Terjadi kesalahan saat memposting penyesuaian.",
      })
    }
  }

  // Create Opname Submit
  const handleCreateOpname = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreateModalError(null)

    if (!newWarehouseId) {
      setCreateModalError("Pilih gudang tempat dilakukannya stock opname.")
      return
    }

    try {
      const res = await createOpnameMutation.mutateAsync({
        warehouse_id: newWarehouseId,
        opname_number: newOpnameNumber.trim() || undefined,
        notes: newNotes.trim() || undefined,
      })

      setShowCreateModal(false)
      setNewWarehouseId("")
      setNewOpnameNumber("")
      setNewNotes("")
      setToast({
        type: "success",
        title: "Sesi Opname Dibuat",
        message: `Sesi opname ${res.opname_number || "baru"} siap dilakukan.`,
      })

      // Automatically open drawer for the newly created opname
      setActiveOpnameId(res.id)
    } catch (err: unknown) {
      setCreateModalError(err instanceof Error ? err.message : "Gagal membuat sesi stock opname.")
    }
  }

  // Filtered opname sessions list
  const filteredOpnames = useMemo(() => {
    return opnames.filter((o) => {
      const matchesStatus = statusFilter === "ALL" || o.status === statusFilter
      const wh = warehouses.find((w) => w.id === o.warehouse_id)

      const matchesSearch =
        searchQuery === "" ||
        o.opname_number.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (o.notes && o.notes.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (o.conducted_by && o.conducted_by.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (wh && wh.name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (wh && wh.code.toLowerCase().includes(searchQuery.toLowerCase()))

      return matchesStatus && matchesSearch
    })
  }, [opnames, statusFilter, searchQuery, warehouses])

  // Counted items in active opname with calculations
  const countedItems = useMemo(() => activeOpnameDetail?.items ?? [], [activeOpnameDetail?.items])
  const filteredCountedItems = useMemo(() => {
    if (!drawerSearch.trim()) return countedItems
    const q = drawerSearch.toLowerCase()
    return countedItems.filter((item) => {
      const prod = products.find((p) => p.id === item.product_id)
      const loc = opnameLocations.find((l) => l.id === item.location_id)
      return (
        (prod && (prod.name.toLowerCase().includes(q) || prod.sku.toLowerCase().includes(q))) ||
        (loc && (loc.name.toLowerCase().includes(q) || loc.code.toLowerCase().includes(q))) ||
        (item.notes && item.notes.toLowerCase().includes(q))
      )
    })
  }, [countedItems, drawerSearch, products, opnameLocations])

  // Discrepancy stats for active opname
  const activeStats = useMemo(() => {
    let matchCount = 0
    let surplusCount = 0
    let deficitCount = 0
    let totalDiscrepancyQty = 0

    countedItems.forEach((item) => {
      const disc = Number(item.discrepancy_qty) || 0
      totalDiscrepancyQty += disc
      if (disc === 0) matchCount++
      else if (disc > 0) surplusCount++
      else deficitCount++
    })

    return {
      totalItems: countedItems.length,
      matchCount,
      surplusCount,
      deficitCount,
      totalDiscrepancyQty,
    }
  }, [countedItems])

  // Helper for warehouse name
  const getWarehouseName = (id: string) => {
    const wh = warehouses.find((w) => w.id === id)
    return wh ? `${wh.name} (${wh.code})` : id.slice(0, 8)
  }

  // Helper for location name
  const getLocationName = (id: string) => {
    const loc = opnameLocations.find((l) => l.id === id)
    return loc ? `${loc.name} [${loc.code}]` : id.slice(0, 8)
  }

  // Helper for product details
  const getProduct = (id: string) => {
    return products.find((p) => p.id === id)
  }

  // Helper for Status Badge
  const renderStatusBadge = (status: StockOpnameStatus) => {
    switch (status) {
      case "DRAFT":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold bg-slate-100 text-slate-700 border border-slate-300">
            <span className="w-1.5 h-1.5 rounded-full bg-slate-400" />
            Draft
          </span>
        )
      case "IN_PROGRESS":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold bg-amber-50 text-amber-800 border border-amber-300 animate-pulse">
            <span className="w-1.5 h-1.5 rounded-full bg-amber-500" />
            Sedang Dihitung
          </span>
        )
      case "COMPLETED":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-800 border border-emerald-300">
            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />
            Selesai & Diposting
          </span>
        )
      case "CANCELLED":
        return (
          <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold bg-rose-50 text-rose-800 border border-rose-300">
            <X className="w-3.5 h-3.5 text-rose-600" />
            Dibatalkan
          </span>
        )
    }
  }

  // Visual Discrepancy indicator badge
  const renderDiscrepancyBadge = (discrepancy: number) => {
    if (discrepancy === 0) {
      return (
        <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-300">
          <Check className="w-3.5 h-3.5 text-emerald-600" />
          Sesuai (0)
        </span>
      )
    }
    if (discrepancy > 0) {
      return (
        <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-300">
          <TrendingUp className="w-3.5 h-3.5 text-blue-600" />
          +{discrepancy} Surplus
        </span>
      )
    }
    return (
      <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-semibold bg-rose-50 text-rose-700 border border-rose-300">
        <TrendingDown className="w-3.5 h-3.5 text-rose-600" />
        {discrepancy} Hilang/Kurang
      </span>
    )
  }

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
              <Sparkles className="w-5 h-5" />
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
                <ClipboardCheck className="w-6 h-6 text-[#2563EB]" />
                <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
                  Stock Opname & Penyesuaian Fisik
                </h1>
              </div>
              <p className="text-xs sm:text-sm text-slate-500">
                Penghitungan fisik berkala, deteksi selisih otomatis, dan posting penyeimbang ke ledger @LOSS
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => {
                refetchOpnames()
                refetchWarehouses()
              }}
              className="p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[48px] min-w-[48px] flex items-center justify-center transition-colors"
              title="Refresh Data"
            >
              <RefreshCw className="w-5 h-5" />
            </button>
            <button
              onClick={() => {
                setNewWarehouseId(warehouses[0]?.id || "")
                setNewOpnameNumber(`OPN-${new Date().toISOString().slice(0, 10).replace(/-/g, "")}-${String(opnames.length + 1).padStart(3, "0")}`)
                setShowCreateModal(true)
              }}
              className="flex-1 sm:flex-none flex items-center justify-center gap-2 px-4 py-2.5 bg-[#2563EB] hover:bg-blue-700 text-white font-medium rounded-lg shadow-sm transition-all min-h-[48px]"
            >
              <Plus className="w-5 h-5" />
              <span>Audit Baru (New Opname)</span>
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 pt-6 space-y-6">
        {/* ── Summary Stats ── */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4">
          <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-xs">
            <span className="text-xs font-medium text-slate-500">Total Sesi Opname</span>
            <div className="flex items-baseline justify-between mt-2">
              <span className="text-2xl font-bold text-slate-900">{opnames.length}</span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
                Semua Periode
              </span>
            </div>
          </div>

          <div className="bg-white p-4 rounded-xl border border-amber-200 bg-amber-50/20 shadow-xs">
            <span className="text-xs font-medium text-amber-700">Sedang Dihitung (In Progress)</span>
            <div className="flex items-baseline justify-between mt-2">
              <span className="text-2xl font-bold text-amber-700">
                {opnames.filter((o) => o.status === "IN_PROGRESS").length}
              </span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-amber-100 text-amber-800">
                Aktif
              </span>
            </div>
          </div>

          <div className="bg-white p-4 rounded-xl border border-emerald-200 bg-emerald-50/20 shadow-xs">
            <span className="text-xs font-medium text-emerald-700">Telah Diposting (@LOSS)</span>
            <div className="flex items-baseline justify-between mt-2">
              <span className="text-2xl font-bold text-emerald-700">
                {opnames.filter((o) => o.status === "COMPLETED").length}
              </span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-emerald-100 text-emerald-800">
                Kelar
              </span>
            </div>
          </div>

          <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-xs">
            <span className="text-xs font-medium text-slate-500">Draft Menunggu Audit</span>
            <div className="flex items-baseline justify-between mt-2">
              <span className="text-2xl font-bold text-slate-800">
                {opnames.filter((o) => o.status === "DRAFT").length}
              </span>
              <span className="text-xs font-semibold px-2 py-0.5 rounded-full bg-slate-100 text-slate-600">
                Siap
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
                placeholder="Cari nomor opname, catatan, atau petugas..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-10 pr-4 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#2563EB] focus:border-transparent min-h-[48px]"
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
                className="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#2563EB] bg-white min-h-[48px]"
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

          {/* Status Filter Tabs */}
          <div className="flex items-center gap-1 overflow-x-auto pb-1 md:pb-0 scrollbar-none">
            {(["ALL", "IN_PROGRESS", "DRAFT", "COMPLETED"] as const).map((st) => (
              <button
                key={st}
                onClick={() => setStatusFilter(st)}
                className={`px-3 py-2 rounded-lg text-xs font-semibold whitespace-nowrap transition-colors min-h-[48px] flex items-center ${
                  statusFilter === st
                    ? "bg-[#2563EB] text-white"
                    : "bg-slate-100 text-slate-600 hover:bg-slate-200"
                }`}
              >
                {st === "ALL" && "Semua Status"}
                {st === "IN_PROGRESS" && "Sedang Dihitung"}
                {st === "DRAFT" && "Draft"}
                {st === "COMPLETED" && "Selesai"}
              </button>
            ))}
          </div>
        </div>

        {/* ── Opnames List / Table ── */}
        <div className="bg-white rounded-xl border border-slate-200 shadow-xs overflow-hidden">
          {loadingOpnames ? (
            <div className="p-12 text-center">
              <RefreshCw className="w-8 h-8 text-[#2563EB] animate-spin mx-auto mb-3" />
              <p className="text-sm font-medium text-slate-600">Memuat sesi stock opname...</p>
            </div>
          ) : filteredOpnames.length === 0 ? (
            <div className="p-12 text-center">
              <ClipboardCheck className="w-12 h-12 text-slate-300 mx-auto mb-3" />
              <h3 className="text-base font-semibold text-slate-800">Tidak ada sesi stock opname ditemukan</h3>
              <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto">
                {searchQuery || statusFilter !== "ALL" || selectedWarehouseId !== "ALL"
                  ? "Coba ubah filter atau kata kunci pencarian Anda."
                  : "Mulai penghitungan stok fisik pertama dengan membuat audit opname baru."}
              </p>
              <button
                onClick={() => setShowCreateModal(true)}
                className="mt-4 inline-flex items-center gap-2 px-4 py-2.5 bg-[#2563EB] text-white text-sm font-medium rounded-lg hover:bg-blue-700 min-h-[48px]"
              >
                <Plus className="w-4 h-4" />
                <span>Buat Sesi Opname Baru</span>
              </button>
            </div>
          ) : (
            <div className="divide-y divide-slate-100">
              {/* Desktop Table View */}
              <div className="hidden md:block overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead className="bg-slate-50 border-b border-slate-200 text-xs uppercase font-semibold text-slate-600 tracking-wider">
                    <tr>
                      <th className="px-6 py-3.5">Nomor Audit</th>
                      <th className="px-6 py-3.5">Gudang</th>
                      <th className="px-6 py-3.5">Status</th>
                      <th className="px-6 py-3.5">Petugas / Conductor</th>
                      <th className="px-6 py-3.5">Tanggal Dibuat</th>
                      <th className="px-6 py-3.5 text-right">Aksi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {filteredOpnames.map((o) => {
                      const isOpened = activeOpnameId === o.id
                      return (
                        <tr
                          key={o.id}
                          className={`hover:bg-slate-50/80 transition-colors ${
                            isOpened ? "bg-blue-50/40" : ""
                          }`}
                        >
                          <td className="px-6 py-4">
                            <div className="font-semibold text-slate-900">{o.opname_number}</div>
                            {o.notes && <div className="text-xs text-slate-500 truncate max-w-xs">{o.notes}</div>}
                          </td>
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-1.5 text-slate-800 font-medium">
                              <Building className="w-4 h-4 text-slate-400" />
                              <span>{getWarehouseName(o.warehouse_id)}</span>
                            </div>
                          </td>
                          <td className="px-6 py-4">{renderStatusBadge(o.status)}</td>
                          <td className="px-6 py-4">
                            <div className="flex items-center gap-1.5 text-slate-700 text-xs">
                              <User className="w-3.5 h-3.5 text-slate-400" />
                              <span>
                                {o.conducted_by
                                  ? o.conducted_by.length > 20
                                    ? "Petugas Operasional"
                                    : o.conducted_by
                                  : "Petugas Gudang"}
                              </span>
                            </div>
                          </td>
                          <td className="px-6 py-4 text-xs text-slate-500">
                            {new Date(o.created_at).toLocaleDateString("id-ID", {
                              day: "numeric",
                              month: "short",
                              year: "numeric",
                              hour: "2-digit",
                              minute: "2-digit",
                            })}
                          </td>
                          <td className="px-6 py-4 text-right">
                            <button
                              onClick={() => setActiveOpnameId(o.id)}
                              className={`inline-flex items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-semibold transition-all min-h-[48px] ${
                                isOpened
                                  ? "bg-[#2563EB] text-white shadow-xs"
                                  : "border border-slate-300 text-slate-700 hover:bg-slate-100"
                              }`}
                            >
                              <span>{o.status === "COMPLETED" ? "Lihat Hasil Hitung" : "Buka Penghitungan"}</span>
                              <ArrowRight className="w-3.5 h-3.5" />
                            </button>
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>

              {/* Mobile Cards View */}
              <div className="md:hidden divide-y divide-slate-100">
                {filteredOpnames.map((o) => (
                  <div key={o.id} className="p-4 space-y-3">
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <span className="font-bold text-slate-900 text-base">{o.opname_number}</span>
                        <div className="text-xs text-slate-500 mt-0.5">{getWarehouseName(o.warehouse_id)}</div>
                      </div>
                      {renderStatusBadge(o.status)}
                    </div>
                    {o.notes && <p className="text-xs text-slate-600 bg-slate-50 p-2 rounded-md">{o.notes}</p>}
                    <div className="flex items-center justify-between pt-1">
                      <span className="text-[11px] text-slate-400">
                        {new Date(o.created_at).toLocaleDateString("id-ID", {
                          day: "numeric",
                          month: "short",
                          year: "numeric",
                        })}
                      </span>
                      <button
                        onClick={() => setActiveOpnameId(o.id)}
                        className="inline-flex items-center gap-1.5 px-4 py-2.5 rounded-lg text-xs font-semibold bg-[#2563EB] text-white shadow-xs min-h-[48px]"
                      >
                        <span>{o.status === "COMPLETED" ? "Lihat Hasil" : "Mulai Hitung"}</span>
                        <ArrowRight className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ── ACTIVE COUNTING DRAWER / VIEW ── */}
      {activeOpnameId && activeOpname && (
        <div className="fixed inset-0 z-40 flex justify-end bg-black/40 backdrop-blur-xs transition-opacity animate-in fade-in duration-200">
          <div className="w-full max-w-3xl bg-white h-full shadow-2xl flex flex-col overflow-hidden animate-in slide-in-from-right duration-300">
            {/* Drawer Header */}
            <div className="bg-slate-900 text-white p-4 sm:p-6 border-b border-slate-800 shrink-0">
              <div className="flex items-start justify-between gap-4">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold uppercase tracking-wider text-blue-400">
                      Sesi Penghitungan Fisik
                    </span>
                    {renderStatusBadge(activeOpname.status)}
                  </div>
                  <h2 className="text-lg sm:text-xl font-bold tracking-tight text-white mt-1">
                    {activeOpname.opname_number}
                  </h2>
                  <div className="flex items-center gap-3 text-xs text-slate-300 mt-1">
                    <span className="flex items-center gap-1">
                      <Building className="w-3.5 h-3.5 text-slate-400" />
                      {getWarehouseName(activeOpname.warehouse_id)}
                    </span>
                    <span>•</span>
                    <span className="flex items-center gap-1">
                      <User className="w-3.5 h-3.5 text-slate-400" />
                      {activeOpname.conducted_by || "Petugas Gudang"}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    onClick={() => setSoundEnabled(!soundEnabled)}
                    className="p-2 rounded-lg bg-slate-800 text-slate-300 hover:text-white hover:bg-slate-700 min-h-[48px] min-w-[48px] flex items-center justify-center transition-colors"
                    title={soundEnabled ? "Audio beep aktif" : "Audio beep senyap"}
                  >
                    {soundEnabled ? <Volume2 className="w-5 h-5 text-emerald-400" /> : <VolumeX className="w-5 h-5" />}
                  </button>
                  <button
                    onClick={() => setActiveOpnameId(null)}
                    className="p-2 rounded-lg bg-slate-800 text-slate-300 hover:text-white hover:bg-slate-700 min-h-[48px] min-w-[48px] flex items-center justify-center transition-colors"
                    title="Tutup Panel"
                  >
                    <X className="w-6 h-6" />
                  </button>
                </div>
              </div>

              {/* Counting Stats Bar */}
              <div className="grid grid-cols-4 gap-2 mt-4 pt-4 border-t border-slate-800 text-center">
                <div className="bg-slate-800/80 p-2 rounded-lg">
                  <span className="text-[10px] text-slate-400 uppercase font-medium">Total Dihitung</span>
                  <div className="text-lg font-bold text-white mt-0.5">{activeStats.totalItems}</div>
                </div>
                <div className="bg-emerald-950/40 border border-emerald-800/50 p-2 rounded-lg">
                  <span className="text-[10px] text-emerald-400 uppercase font-medium">Sesuai (0)</span>
                  <div className="text-lg font-bold text-emerald-400 mt-0.5">{activeStats.matchCount}</div>
                </div>
                <div className="bg-blue-950/40 border border-blue-800/50 p-2 rounded-lg">
                  <span className="text-[10px] text-blue-400 uppercase font-medium">Surplus (+)</span>
                  <div className="text-lg font-bold text-blue-400 mt-0.5">+{activeStats.surplusCount}</div>
                </div>
                <div className="bg-rose-950/40 border border-rose-800/50 p-2 rounded-lg">
                  <span className="text-[10px] text-rose-400 uppercase font-medium">Defisit (-)</span>
                  <div className="text-lg font-bold text-rose-400 mt-0.5">-{activeStats.deficitCount}</div>
                </div>
              </div>
            </div>

            {/* Drawer Body */}
            <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-6">
              {/* If Session is DRAFT or IN_PROGRESS: Show Barcode Input & Entry Form */}
              {activeOpname.status !== "COMPLETED" && activeOpname.status !== "CANCELLED" ? (
                <div className="bg-blue-50/50 border border-blue-200 rounded-xl p-4 sm:p-5 space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Barcode className="w-5 h-5 text-[#2563EB]" />
                      <h3 className="text-sm font-bold text-slate-900">
                        Scan Barcode Fisik (Continuous Scanner)
                      </h3>
                    </div>
                    <span className="text-[11px] font-semibold text-emerald-700 bg-emerald-100/80 px-2 py-0.5 rounded-full">
                      Scanner Aktif (&le;30ms)
                    </span>
                  </div>

                  {/* Fast Scan Input Bar */}
                  <form onSubmit={handleManualScanSubmit} className="flex gap-2">
                    <div className="relative flex-1">
                      <ScanLine className="w-5 h-5 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
                      <input
                        ref={scanInputRef}
                        type="text"
                        placeholder="Scan barcode SKU / nomor rak atau ketik..."
                        value={scanInput}
                        onChange={(e) => setScanInput(e.target.value)}
                        className="w-full pl-10 pr-4 py-2.5 rounded-lg border border-slate-300 text-sm focus:outline-none focus:ring-2 focus:ring-[#2563EB] bg-white min-h-[48px]"
                      />
                    </div>
                    <button
                      type="submit"
                      className="px-5 py-2.5 bg-[#EA580C] hover:bg-orange-700 text-white font-semibold text-sm rounded-lg shadow-xs transition-colors min-h-[48px] shrink-0"
                    >
                      Scan / Cari
                    </button>
                  </form>

                  {/* Active Count Entry Card */}
                  <form onSubmit={handleRecordCount} className="bg-white p-4 rounded-lg border border-slate-200 space-y-3">
                    {countingError && (
                      <div className="p-3 rounded-lg bg-rose-50 border border-rose-200 text-xs text-rose-700 flex items-center gap-2">
                        <AlertCircle className="w-4 h-4 shrink-0" />
                        <span>{countingError}</span>
                      </div>
                    )}

                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      {/* Product Selector */}
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 mb-1">
                          Produk Master <span className="text-rose-500">*</span>
                        </label>
                        <select
                          value={selectedProduct?.id || ""}
                          onChange={(e) => {
                            const found = products.find((p) => p.id === e.target.value) || null
                            setSelectedProduct(found)
                            setSystemCountInput(50) // Default baseline system stock
                          }}
                          className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#2563EB] bg-white min-h-[48px]"
                        >
                          <option value="">-- Pilih atau scan produk --</option>
                          {products.map((p) => (
                            <option key={p.id} value={p.id}>
                              {p.name} [{p.sku}]
                            </option>
                          ))}
                        </select>
                      </div>

                      {/* Location Selector */}
                      <div>
                        <label className="block text-xs font-semibold text-slate-700 mb-1">
                          Lokasi Rak / Bin <span className="text-rose-500">*</span>
                        </label>
                        <select
                          value={effectiveLocationId}
                          onChange={(e) => setSelectedLocationId(e.target.value)}
                          className="w-full px-3 py-2 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#2563EB] bg-white min-h-[48px]"
                        >
                          <option value="">-- Pilih lokasi penyimpanan --</option>
                          {opnameLocations.map((l) => (
                            <option key={l.id} value={l.id}>
                              {l.name} ({l.code})
                            </option>
                          ))}
                        </select>
                      </div>
                    </div>

                    {/* Quantities & Discrepancy Preview */}
                    <div className="bg-slate-50 p-3 rounded-lg border border-slate-200 grid grid-cols-3 gap-3 items-center">
                      <div>
                        <span className="block text-[11px] font-medium text-slate-500">Stok Sistem</span>
                        <div className="text-base font-bold text-slate-800 mt-1">{systemCountInput}</div>
                      </div>

                      <div>
                        <span className="block text-[11px] font-semibold text-[#2563EB]">Jumlah Fisik</span>
                        <div className="flex items-center gap-1 mt-1">
                          <input
                            type="number"
                            min="0"
                            step="any"
                            value={physicalCountInput}
                            onChange={(e) => setPhysicalCountInput(e.target.value)}
                            className="w-full px-2 py-1 text-base font-bold text-slate-900 border border-slate-300 rounded focus:ring-2 focus:ring-[#2563EB] min-h-[40px]"
                          />
                        </div>
                      </div>

                      <div>
                        <span className="block text-[11px] font-medium text-slate-500">Prediksi Selisih</span>
                        <div className="mt-1">{renderDiscrepancyBadge(realTimeDiscrepancy)}</div>
                      </div>
                    </div>

                    {/* Quick increment buttons */}
                    <div className="flex items-center gap-1.5 overflow-x-auto pb-1">
                      <span className="text-[11px] text-slate-400 mr-1">Quick:</span>
                      {[1, 5, 10, 50].map((inc) => (
                        <button
                          key={inc}
                          type="button"
                          onClick={() => {
                            const cur = parseFloat(physicalCountInput) || 0
                            setPhysicalCountInput(String(cur + inc))
                          }}
                          className="px-2.5 py-1 text-xs font-semibold rounded bg-slate-100 hover:bg-slate-200 text-slate-700 min-h-[36px]"
                        >
                          +{inc}
                        </button>
                      ))}
                      <button
                        type="button"
                        onClick={() => {
                          const cur = parseFloat(physicalCountInput) || 0
                          if (cur > 0) setPhysicalCountInput(String(Math.max(0, cur - 1)))
                        }}
                        className="px-2.5 py-1 text-xs font-semibold rounded bg-slate-100 hover:bg-slate-200 text-slate-700 min-h-[36px]"
                      >
                        -1
                      </button>
                      <button
                        type="button"
                        onClick={() => setPhysicalCountInput(String(systemCountInput))}
                        className="px-2.5 py-1 text-xs font-semibold rounded bg-emerald-50 hover:bg-emerald-100 text-emerald-700 border border-emerald-200 min-h-[36px]"
                      >
                        Samakan Sistem
                      </button>
                    </div>

                    {/* Notes Field */}
                    <div>
                      <input
                        type="text"
                        placeholder="Catatan kondisi (opsional, misal: kemasan sobek, hilang, tercecer)..."
                        value={countNotes}
                        onChange={(e) => setCountNotes(e.target.value)}
                        className="w-full px-3 py-2 text-xs rounded-lg border border-slate-200 focus:outline-none focus:ring-1 focus:ring-[#2563EB]"
                      />
                    </div>

                    {/* Submit Record Count Button */}
                    <button
                      type="submit"
                      disabled={addItemMutation.isPending || !selectedProduct}
                      className="w-full flex items-center justify-center gap-2 py-3 bg-[#2563EB] hover:bg-blue-700 disabled:opacity-50 text-white font-bold rounded-lg shadow-sm transition-all min-h-[48px]"
                    >
                      {addItemMutation.isPending ? (
                        <>
                          <RefreshCw className="w-5 h-5 animate-spin" />
                          <span>Menyimpan ke Daftar...</span>
                        </>
                      ) : (
                        <>
                          <Plus className="w-5 h-5" />
                          <span>Catat Hasil Hitung (Record Item)</span>
                        </>
                      )}
                    </button>
                  </form>
                </div>
              ) : (
                <div className="p-4 rounded-xl bg-emerald-50 border border-emerald-200 text-emerald-800 flex items-start gap-3">
                  <CheckCircle2 className="w-6 h-6 text-emerald-600 shrink-0 mt-0.5" />
                  <div>
                    <h4 className="text-sm font-bold">Sesi Telah Selesai & Terkunci</h4>
                    <p className="text-xs text-emerald-700 mt-1">
                      Semua selisih telah dibukukan secara permanen ke buku besar akun penyeimbang (@LOSS). Data fisik
                      tidak dapat diubah kembali.
                    </p>
                    {activeOpname.approved_by && (
                      <div className="text-[11px] text-emerald-600 mt-2 font-medium">
                        Disetujui oleh: {activeOpname.approved_by}
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* ── Counted Items List Section ── */}
              <div className="space-y-3">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <Layers className="w-5 h-5 text-slate-600" />
                    <h3 className="text-sm font-bold text-slate-900">
                      Rincian Barang Dihitung ({countedItems.length})
                    </h3>
                  </div>

                  <div className="relative w-full sm:w-64">
                    <Search className="w-4 h-4 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400" />
                    <input
                      type="text"
                      placeholder="Cari item di sesi ini..."
                      value={drawerSearch}
                      onChange={(e) => setDrawerSearch(e.target.value)}
                      className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg border border-slate-200 focus:outline-none focus:ring-1 focus:ring-[#2563EB] min-h-[36px]"
                    />
                  </div>
                </div>

                {loadingActiveOpname ? (
                  <div className="p-8 text-center text-slate-500 text-xs">Memuat detail barang...</div>
                ) : filteredCountedItems.length === 0 ? (
                  <div className="p-8 text-center bg-slate-50 rounded-xl border border-dashed border-slate-200 text-slate-400 text-xs">
                    Belum ada barang yang dicatat pada sesi audit ini. Scan barcode atau pilih produk di atas.
                  </div>
                ) : (
                  <div className="space-y-2">
                    {filteredCountedItems.map((item) => {
                      const prod = getProduct(item.product_id)
                      const disc = Number(item.discrepancy_qty) || 0
                      return (
                        <div
                          key={item.id}
                          className="bg-white p-3.5 rounded-xl border border-slate-200 shadow-2xs hover:border-slate-300 transition-colors flex flex-col sm:flex-row sm:items-center justify-between gap-3"
                        >
                          <div className="space-y-1 flex-1">
                            <div className="flex items-center gap-2">
                              <span className="font-bold text-slate-900 text-sm">
                                {prod ? prod.name : `Produk [${item.product_id.slice(0, 8)}]`}
                              </span>
                              {prod && (
                                <span className="text-[11px] font-mono px-1.5 py-0.5 rounded bg-slate-100 text-slate-600">
                                  {prod.sku}
                                </span>
                              )}
                            </div>
                            <div className="flex items-center gap-2 text-xs text-slate-500">
                              <Building className="w-3.5 h-3.5 text-slate-400" />
                              <span>{getLocationName(item.location_id)}</span>
                              {item.notes && (
                                <>
                                  <span>•</span>
                                  <span className="italic text-slate-600 truncate max-w-xs">{item.notes}</span>
                                </>
                              )}
                            </div>
                          </div>

                          <div className="flex items-center gap-4 shrink-0 justify-between sm:justify-end border-t sm:border-t-0 pt-2 sm:pt-0">
                            <div className="text-right text-xs">
                              <span className="text-slate-400 text-[10px] block">Sistem / Fisik</span>
                              <span className="font-semibold text-slate-700">
                                {item.system_qty} &rarr;{" "}
                                <span className="font-bold text-slate-900">{item.physical_qty}</span>
                              </span>
                            </div>

                            <div>{renderDiscrepancyBadge(disc)}</div>
                          </div>
                        </div>
                      )
                    })}
                  </div>
                )}
              </div>
            </div>

            {/* Drawer Footer with Posting Action Button */}
            {activeOpname.status !== "COMPLETED" && activeOpname.status !== "CANCELLED" && (
              <div className="bg-slate-50 p-4 sm:p-5 border-t border-slate-200 shrink-0 space-y-3">
                <div className="flex items-center justify-between text-xs text-slate-600">
                  <span>Total baris: <strong>{countedItems.length}</strong></span>
                  <span>
                    Total selisih:{" "}
                    <strong
                      className={
                        activeStats.totalDiscrepancyQty === 0
                          ? "text-emerald-600"
                          : activeStats.totalDiscrepancyQty > 0
                          ? "text-blue-600"
                          : "text-rose-600"
                      }
                    >
                      {activeStats.totalDiscrepancyQty >= 0 ? "+" : ""}
                      {activeStats.totalDiscrepancyQty}
                    </strong>
                  </span>
                </div>

                <button
                  type="button"
                  onClick={() => setShowPostingConfirmModal(true)}
                  disabled={countedItems.length === 0}
                  className="w-full flex items-center justify-center gap-2 py-3 px-4 bg-[#EA580C] hover:bg-orange-700 disabled:opacity-50 text-white font-bold text-sm rounded-lg shadow-md transition-all min-h-[48px]"
                >
                  <Lock className="w-5 h-5" />
                  <span>Posting Penyesuaian (@LOSS)</span>
                </button>
              </div>
            )}
          </div>
        </div>
      )}

      {/* ── MODAL: Posting Confirmation (@LOSS) ── */}
      {showPostingConfirmModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs">
          <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-2xl border border-slate-100 space-y-4 animate-in zoom-in-95 duration-200">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-xl bg-orange-100 text-[#EA580C]">
                <AlertCircle className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-lg font-bold text-slate-900">Konfirmasi Posting Penyesuaian (@LOSS)</h3>
                <p className="text-xs text-slate-500">Tindakan ini permanen dan tidak dapat dibatalkan</p>
              </div>
            </div>

            <div className="bg-slate-50 p-4 rounded-xl text-xs space-y-2 text-slate-700 border border-slate-200">
              <p>
                Sistem akan secara otomatis melakukan <strong>double-entry ledger balancing</strong>:
              </p>
              <ul className="list-disc pl-4 space-y-1 text-slate-600">
                <li>
                  Barang <strong>Surplus ({activeStats.surplusCount} item)</strong>: Mutasi masuk dari lokasi virtual{" "}
                  <code>@LOSS</code> ke rak gudang.
                </li>
                <li>
                  Barang <strong>Defisit ({activeStats.deficitCount} item)</strong>: Mutasi keluar dari rak gudang ke
                  lokasi virtual <code>@LOSS</code>.
                </li>
                <li>
                  Sesi ini akan disetujui & berstatus <code>COMPLETED</code>.
                </li>
              </ul>
            </div>

            <div className="flex items-center gap-2 pt-2">
              <button
                type="button"
                onClick={() => setShowPostingConfirmModal(false)}
                className="flex-1 py-2.5 px-4 rounded-lg border border-slate-200 text-slate-700 font-semibold text-sm hover:bg-slate-50 min-h-[48px]"
              >
                Batal
              </button>
              <button
                type="button"
                onClick={handleConfirmPosting}
                disabled={completeOpnameMutation.isPending}
                className="flex-1 py-2.5 px-4 rounded-lg bg-[#EA580C] hover:bg-orange-700 text-white font-bold text-sm shadow-sm transition-all min-h-[48px] flex items-center justify-center gap-2"
              >
                {completeOpnameMutation.isPending ? (
                  <>
                    <RefreshCw className="w-4 h-4 animate-spin" />
                    <span>Memposting...</span>
                  </>
                ) : (
                  <>
                    <CheckCircle2 className="w-4 h-4" />
                    <span>Ya, Posting Sekarang</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── MODAL: Create New Stock Opname ── */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-2xl border border-slate-100 space-y-4 animate-in zoom-in-95 duration-200">
            <div className="flex items-center justify-between pb-2 border-b border-slate-100">
              <div className="flex items-center gap-2">
                <ClipboardCheck className="w-5 h-5 text-[#2563EB]" />
                <h3 className="text-base font-bold text-slate-900">Audit Baru (New Opname)</h3>
              </div>
              <button
                onClick={() => setShowCreateModal(false)}
                className="p-1.5 rounded-md text-slate-400 hover:text-slate-600"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleCreateOpname} className="space-y-4">
              {createModalError && (
                <div className="p-3 rounded-lg bg-rose-50 border border-rose-200 text-xs text-rose-700 flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{createModalError}</span>
                </div>
              )}

              {/* Warehouse selector */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Target Gudang <span className="text-rose-500">*</span>
                </label>
                <select
                  value={newWarehouseId}
                  onChange={(e) => setNewWarehouseId(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#2563EB] bg-white min-h-[48px]"
                  required
                >
                  <option value="">-- Pilih Gudang Audit --</option>
                  {warehouses.map((w) => (
                    <option key={w.id} value={w.id}>
                      {w.name} ({w.code})
                    </option>
                  ))}
                </select>
              </div>

              {/* Opname Number */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Nomor Referensi Opname
                </label>
                <input
                  type="text"
                  placeholder="Contoh: OPN-20260408-001"
                  value={newOpnameNumber}
                  onChange={(e) => setNewOpnameNumber(e.target.value)}
                  className="w-full px-3 py-2.5 rounded-lg border border-slate-200 text-sm focus:outline-none focus:ring-2 focus:ring-[#2563EB] min-h-[48px]"
                />
                <span className="text-[11px] text-slate-400 mt-1 block">
                  Kosongkan untuk membuat nomor otomatis secara berurutan.
                </span>
              </div>

              {/* Scope & Notes */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1">
                  Cakupan & Catatan Audit
                </label>
                <textarea
                  rows={3}
                  placeholder="Misal: Audit Siklus Bulanan Rak Depan & Sembako..."
                  value={newNotes}
                  onChange={(e) => setNewNotes(e.target.value)}
                  className="w-full px-3 py-2 text-sm rounded-lg border border-slate-200 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                />
              </div>

              <div className="flex items-center gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="flex-1 py-2.5 px-4 rounded-lg border border-slate-200 text-slate-700 font-semibold text-sm hover:bg-slate-50 min-h-[48px]"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createOpnameMutation.isPending}
                  className="flex-1 py-2.5 px-4 rounded-lg bg-[#2563EB] hover:bg-blue-700 text-white font-bold text-sm shadow-sm transition-all min-h-[48px] flex items-center justify-center gap-2"
                >
                  {createOpnameMutation.isPending ? (
                    <>
                      <RefreshCw className="w-4 h-4 animate-spin" />
                      <span>Membuat Sesi...</span>
                    </>
                  ) : (
                    <>
                      <Plus className="w-4 h-4" />
                      <span>Buat Sesi Opname</span>
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
