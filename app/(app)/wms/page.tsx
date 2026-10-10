"use client"

import React, { useState, useMemo } from "react"
import Link from "next/link"
import { useManualRefresh } from "@/hooks/useManualRefresh"
import { RefreshButton } from "@/components/ui/RefreshButton"
import {
  Warehouse as WarehouseIcon,
  Plus,
  Search,
  Box,
  Layers,
  Truck,
  ScanLine,
  ArrowRightLeft,
  ArrowUpRight,
  ArrowDownLeft,
  CheckCircle2,
  AlertCircle,
  X,
  Building,
  MapPin,
  Barcode,
  PackageCheck,
  RefreshCw,
  History,
} from "lucide-react"
import {
  useWarehouses,
  useWarehouseLocations,
  useCreateWarehouse,
  useCreateLocation,
  useStockTransfers,
} from "@/hooks/useWMS"
import { useWMSMovements, useWMSStock } from "@/hooks/useWMSLedger"
import { Warehouse, WarehouseLocation, LocationType, StockSummary } from "@/lib/api"
import { ExportModal, ExportButton } from "@/components/ui/ExportModal"
import { ActivityTimelineDrawer } from "@/components/wms/ActivityTimelineDrawer"

export default function WMSDashboardPage() {
  const { data: warehouses = [], isLoading: loadingWarehouses, refetch: refetchWarehouses } = useWarehouses()
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>("ALL")

  const activeWarehouseFilter = selectedWarehouseId === "ALL" ? null : selectedWarehouseId
  const {
    data: locations = [],
    isLoading: loadingLocations,
    refetch: refetchLocations,
  } = useWarehouseLocations(activeWarehouseFilter)
  const { data: transfers = [] } = useStockTransfers(activeWarehouseFilter)

  // Real WMS Movements Ledger & Warehouse Stock Summary
  const {
    data: movements = [],
    isLoading: loadingMovements,
    refetch: refetchMovements,
  } = useWMSMovements({ limit: 50 })

  const {
    data: stockSummary = [],
    isLoading: loadingStock,
    refetch: refetchStock,
    dataUpdatedAt,
  } = useWMSStock(activeWarehouseFilter ?? undefined)
  const { refresh, status: refreshStatus, refreshError } = useManualRefresh([refetchWarehouses, refetchLocations, refetchMovements, refetchStock])

  // Modals state
  const [showCreateWhModal, setShowCreateWhModal] = useState(false)
  const [showCreateLocModal, setShowCreateLocModal] = useState(false)
  const [selectedMovementForAudit, setSelectedMovementForAudit] = useState<any | null>(null)

  // Filters & Tabs
  const [activeTab, setActiveTab] = useState<"locations" | "movements">("locations")
  const [searchQuery, setSearchQuery] = useState("")
  const [typeFilter, setTypeFilter] = useState<string>("ALL")

  // Mutations
  const createWarehouseMutation = useCreateWarehouse()
  const createLocationMutation = useCreateLocation()

  // Form states for Create Warehouse
  const [newWhCode, setNewWhCode] = useState("")
  const [newWhName, setNewWhName] = useState("")
  const [newWhAddress, setNewWhAddress] = useState("")
  const [whFormError, setWhFormError] = useState<string | null>(null)

  // Form states for Create Location
  const [newLocWarehouseId, setNewLocWarehouseId] = useState("")
  const [newLocCode, setNewLocCode] = useState("")
  const [newLocName, setNewLocName] = useState("")
  const [newLocBarcode, setNewLocBarcode] = useState("")
  const [newLocType, setNewLocType] = useState<LocationType>("INTERNAL")
  const [newLocIsPallet, setNewLocIsPallet] = useState(false)
  const [newLocPalletNumber, setNewLocPalletNumber] = useState("")
  const [newLocCapacity, setNewLocCapacity] = useState("1000")
  const [locFormError, setLocFormError] = useState<string | null>(null)

  // Active warehouse object
  const activeWarehouse = useMemo(() => {
    if (selectedWarehouseId === "ALL") return null
    return warehouses.find((w) => w.id === selectedWarehouseId) || null
  }, [warehouses, selectedWarehouseId])

  // Filtered locations
  const filteredLocations = useMemo(() => {
    return locations.filter((loc) => {
      const matchesSearch =
        searchQuery === "" ||
        loc.code.toLowerCase().includes(searchQuery.toLowerCase()) ||
        loc.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (loc.barcode && loc.barcode.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (loc.pallet_number && loc.pallet_number.toLowerCase().includes(searchQuery.toLowerCase()))

      const matchesType =
        typeFilter === "ALL" ||
        (typeFilter === "PALLET" ? loc.is_pallet : loc.type === typeFilter)

      return matchesSearch && matchesType
    })
  }, [locations, searchQuery, typeFilter])

  // Map real stock quantity per location from stock summary
  const stockByLocation = useMemo(() => {
    const map = new Map<string, number>()
    for (const item of stockSummary) {
      if (item.location_id) {
        map.set(item.location_id, (map.get(item.location_id) || 0) + Number(item.quantity || 0))
      }
    }
    return map
  }, [stockSummary])

  const [exportingStock, setExportingStock] = useState(false)

  // Handle warehouse creation
  const handleCreateWarehouse = async (e: React.FormEvent) => {
    e.preventDefault()
    setWhFormError(null)

    if (!newWhCode.trim() || !newWhName.trim()) {
      setWhFormError("Kode gudang dan nama gudang wajib diisi.")
      return
    }

    try {
      await createWarehouseMutation.mutateAsync({
        code: newWhCode.trim(),
        name: newWhName.trim(),
        address: newWhAddress.trim() || undefined,
        is_active: true,
      })
      setShowCreateWhModal(false)
      setNewWhCode("")
      setNewWhName("")
      setNewWhAddress("")
      refetchWarehouses()
    } catch (err: unknown) {
      setWhFormError(err instanceof Error ? err.message : "Gagal membuat gudang baru.")
    }
  }

  // Handle location creation
  const handleCreateLocation = async (e: React.FormEvent) => {
    e.preventDefault()
    setLocFormError(null)

    const targetWhId = newLocWarehouseId || (selectedWarehouseId !== "ALL" ? selectedWarehouseId : warehouses[0]?.id)
    if (!targetWhId) {
      setLocFormError("Pilih gudang tujuan terlebih dahulu.")
      return
    }
    if (!newLocCode.trim() || !newLocName.trim()) {
      setLocFormError("Kode lokasi dan nama lokasi wajib diisi.")
      return
    }

    try {
      await createLocationMutation.mutateAsync({
        warehouse_id: targetWhId,
        code: newLocCode.trim().toUpperCase(),
        name: newLocName.trim(),
        barcode: newLocBarcode.trim() || undefined,
        type: newLocType,
        is_pallet: newLocIsPallet,
        pallet_number: newLocIsPallet && newLocPalletNumber.trim() ? newLocPalletNumber.trim() : undefined,
        max_capacity: parseFloat(newLocCapacity) || 1000,
      })
      setShowCreateLocModal(false)
      setNewLocCode("")
      setNewLocName("")
      setNewLocBarcode("")
      setNewLocIsPallet(false)
      setNewLocPalletNumber("")
      setNewLocCapacity("1000")
      refetchLocations()
    } catch (err: unknown) {
      setLocFormError(err instanceof Error ? err.message : "Gagal menambahkan lokasi.")
    }
  }

  // Helper for virtual location style
  const renderLocationBadge = (locName: string) => {
    if (locName.startsWith("@VENDOR")) {
      return (
        <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-semibold bg-emerald-100 text-emerald-800 border border-emerald-300">
          <ArrowDownLeft className="w-3.5 h-3.5" />
          @VENDOR (Pemasok)
        </span>
      )
    }
    if (locName.startsWith("@TRANSIT")) {
      return (
        <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-semibold bg-amber-100 text-amber-800 border border-amber-300">
          <Truck className="w-3.5 h-3.5" />
          @TRANSIT (Dalam Perjalanan)
        </span>
      )
    }
    if (locName.startsWith("@CUSTOMER") || locName === "Pelanggan") {
      return (
        <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-semibold bg-blue-100 text-blue-800 border border-blue-300">
          <ArrowUpRight className="w-3.5 h-3.5" />
          Pelanggan Tujuan
        </span>
      )
    }
    if (locName.startsWith("@SCRAP") || locName.startsWith("@LOSS")) {
      return (
        <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-semibold bg-rose-100 text-rose-800 border border-rose-300">
          <AlertCircle className="w-3.5 h-3.5" />
          {locName}
        </span>
      )
    }
    return (
      <span className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium bg-slate-100 text-slate-800 border border-slate-200">
        <Box className="w-3.5 h-3.5 text-slate-500" />
        {locName}
      </span>
    )
  }

  return (
    <div className="min-h-screen bg-[#F8FAFC] pb-16">
      {/* ── Top Bar / Header ── */}
      <div className="bg-white border-b border-[#E2E8F0] px-4 sm:px-6 py-5 sticky top-0 z-20 shadow-xs">
        <div className="max-w-7xl mx-auto flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2.5">
              <div className="p-2 rounded-lg bg-[#EFF6FF] border border-[#BFDBFE] text-[#2563EB]">
                <WarehouseIcon className="w-6 h-6" />
              </div>
              <div>
                <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
                  Warehouse Management System
                </h1>
                <p className="text-xs sm:text-sm text-slate-500">
                  Monitoring rak penyimpanan, kapasitas multi-gudang, dan pergerakan stok
                </p>
              </div>
            </div>
          </div>

          {/* Quick Action Navigation Buttons (min 48px touch target) */}
          <div className="flex flex-wrap items-center gap-2">
            <Link
              href="/wms/scanner"
              className="inline-flex items-center justify-center gap-2 px-4 min-h-[48px] rounded-lg font-medium text-sm bg-white border border-[#BFDBFE] text-[#2563EB] hover:bg-[#EFF6FF] active:scale-95 transition-all shadow-xs"
            >
              <ScanLine className="w-4 h-4 text-[#2563EB]" />
              <span>Barcode Scanner</span>
            </Link>

            <Link
              href="/wms/transfers"
              className="inline-flex items-center justify-center gap-2 px-4 min-h-[48px] rounded-lg font-medium text-sm bg-white border border-[#E2E8F0] text-slate-700 hover:bg-slate-50 active:scale-95 transition-all shadow-xs"
            >
              <ArrowRightLeft className="w-4 h-4 text-slate-600" />
              <span>Transfer Gudang</span>
            </Link>

            <ExportButton onClick={() => setExportingStock(true)} disabled={stockSummary.length === 0} />

            <button
              type="button"
              onClick={() => setShowCreateLocModal(true)}
              className="inline-flex items-center justify-center gap-2 px-4 min-h-[48px] rounded-lg font-medium text-sm bg-white border border-slate-300 text-slate-800 hover:bg-slate-100 active:scale-95 transition-all shadow-xs"
            >
              <Plus className="w-4 h-4" />
              <span>+ Tambah Rak/Lokasi</span>
            </button>

            <button
              type="button"
              onClick={() => setShowCreateWhModal(true)}
              className="inline-flex items-center justify-center gap-2 px-4 min-h-[48px] rounded-lg font-semibold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] active:scale-95 transition-all shadow-sm"
            >
              <Building className="w-4 h-4" />
              <span>+ Gudang Baru</span>
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 pt-6 space-y-6">
        {/* ── Active Warehouse Selector & Quick Metrics ── */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {/* Warehouse Selector Card */}
          <div className="md:col-span-2 bg-white rounded-xl border border-[#E2E8F0] p-4 sm:p-5 shadow-xs flex flex-col justify-between">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <span className="text-xs font-semibold uppercase tracking-wider text-slate-500">
                Pilih Gudang Aktif
              </span>
              <RefreshButton
              updatedAt={dataUpdatedAt}
                status={refreshStatus}
                error={refreshError}
                onClick={() => void refresh()}
                showLabel
                iconClassName="w-3.5 h-3.5"
                className="text-xs text-[#2563EB] hover:bg-blue-50 rounded-md px-2 border border-transparent flex items-center gap-1 min-h-[36px]"
              />
            </div>

            <div className="mt-3">
              <select
                aria-label="Pilih Gudang Aktif"
                value={selectedWarehouseId}
                onChange={(e) => setSelectedWarehouseId(e.target.value)}
                className="w-full min-h-[48px] px-3.5 py-2.5 bg-slate-50 border border-slate-300 rounded-lg text-sm font-semibold text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB] focus:border-transparent transition-all"
              >
                <option value="ALL">🌐 Semua Gudang (Regional Multi-Warehouse View)</option>
                {warehouses.map((wh) => (
                  <option key={wh.id} value={wh.id}>
                    📦 [{wh.code}] {wh.name} {wh.address ? `— ${wh.address}` : ""}
                  </option>
                ))}
              </select>
            </div>

            <div className="mt-3 text-xs text-slate-500 flex items-center gap-2">
              <MapPin className="w-3.5 h-3.5 text-slate-400" />
              <span>
                {activeWarehouse
                  ? `Lokasi Operasional: ${activeWarehouse.name} (${activeWarehouse.address || "Belum ada alamat"})`
                  : "Menampilkan agregasi seluruh fasilitas gudang dan regional"}
              </span>
            </div>
          </div>

          {/* Metric 1: Total Locations / Racks */}
          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 sm:p-5 shadow-xs flex flex-col justify-between">
            <div className="flex items-center justify-between text-slate-500">
              <span className="text-xs font-semibold uppercase tracking-wider">Total Rak & Bin</span>
              <Layers className="w-5 h-5 text-[#2563EB]" />
            </div>
            <div className="mt-2">
              <div className="text-3xl font-extrabold text-slate-900">
                {loadingLocations ? "..." : locations.length}
              </div>
              <p className="text-xs text-slate-500 mt-1">
                {locations.filter((l) => l.is_pallet).length} Pallet LPN terdaftar
              </p>
            </div>
          </div>

          {/* Metric 2: Active Transfers */}
          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 sm:p-5 shadow-xs flex flex-col justify-between">
            <div className="flex items-center justify-between text-slate-500">
              <span className="text-xs font-semibold uppercase tracking-wider">Transfer Dalam Transit</span>
              <Truck className="w-5 h-5 text-[#EA580C]" />
            </div>
            <div className="mt-2">
              <div className="text-3xl font-extrabold text-slate-900">
                {transfers.filter((t) => t.status === "IN_TRANSIT" || t.status === "DISPATCHED").length}
              </div>
              <p className="text-xs text-slate-500 mt-1">
                {transfers.filter((t) => t.status === "APPROVED").length} transfer siap dispatch
              </p>
            </div>
          </div>
        </div>

        {/* ── Main Tabbed View ── */}
        <div className="bg-white rounded-xl border border-[#E2E8F0] shadow-xs overflow-hidden">
          {/* Tabs bar */}
          <div className="flex items-center border-b border-[#E2E8F0] px-4 sm:px-6 bg-slate-50/70">
            <button
              onClick={() => setActiveTab("locations")}
              className={`flex items-center gap-2 py-4 px-4 border-b-2 font-semibold text-sm transition-all min-h-[48px] ${
                activeTab === "locations"
                  ? "border-[#2563EB] text-[#2563EB] bg-white rounded-t-lg"
                  : "border-transparent text-slate-600 hover:text-slate-900"
              }`}
            >
              <Layers className="w-4 h-4" />
              <span>Struktur Rak & Lokasi Penyimpanan ({filteredLocations.length})</span>
            </button>

            <button
              onClick={() => setActiveTab("movements")}
              className={`flex items-center gap-2 py-4 px-4 border-b-2 font-semibold text-sm transition-all min-h-[48px] ${
                activeTab === "movements"
                  ? "border-[#2563EB] text-[#2563EB] bg-white rounded-t-lg"
                  : "border-transparent text-slate-600 hover:text-slate-900"
              }`}
            >
              <ArrowRightLeft className="w-4 h-4" />
              <span>Riwayat Mutasi Stok</span>
            </button>
          </div>

          {/* TAB 1: Locations & Racks Grid */}
          {activeTab === "locations" && (
            <div className="p-4 sm:p-6 space-y-5">
              {/* Search & Type Filter Bar */}
              <div className="flex flex-col sm:flex-row gap-3 items-center justify-between">
                <div className="relative w-full sm:w-80">
                  <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    placeholder="Cari kode rak, nama, atau barcode..."
                    className="w-full pl-10 pr-4 min-h-[48px] text-sm bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                  />
                </div>

                <div className="flex items-center gap-2 w-full sm:w-auto overflow-x-auto pb-1">
                  <span className="text-xs font-medium text-slate-500 whitespace-nowrap">Filter:</span>
                  {[
                    { label: "Semua Tipe", value: "ALL" },
                    { label: "Internal Bin", value: "INTERNAL" },
                    { label: "Pallet LPN", value: "PALLET" },
                    { label: "Transit", value: "TRANSIT" },
                    { label: "Pemasok", value: "VENDOR" },
                  ].map((filter) => (
                    <button
                      key={filter.value}
                      onClick={() => setTypeFilter(filter.value)}
                      className={`px-3 py-2 text-xs font-semibold rounded-lg transition-all min-h-[40px] whitespace-nowrap ${
                        typeFilter === filter.value
                          ? "bg-[#2563EB] text-white"
                          : "bg-slate-100 text-slate-700 hover:bg-slate-200"
                      }`}
                    >
                      {filter.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Grid Cards */}
              {loadingLocations ? (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {[1, 2, 3, 4, 5, 6].map((i) => (
                    <div key={i} className="h-44 bg-slate-100 rounded-xl animate-pulse" />
                  ))}
                </div>
              ) : filteredLocations.length === 0 ? (
                <div className="text-center py-16 px-4 bg-slate-50 rounded-xl border border-dashed border-slate-300">
                  <Box className="w-12 h-12 text-slate-400 mx-auto mb-3" />
                  <h3 className="text-base font-bold text-slate-900">Belum ada lokasi penyimpanan</h3>
                  <p className="text-sm text-slate-500 max-w-md mx-auto mt-1 mb-5">
                    Buat struktur rak, bin, atau pallet LPN pertama Anda untuk mengoptimalkan penempatan barang
                    (putaway).
                  </p>
                  <button
                    onClick={() => setShowCreateLocModal(true)}
                    className="inline-flex items-center gap-2 px-5 min-h-[48px] rounded-lg font-semibold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8]"
                  >
                    <Plus className="w-4 h-4" />
                    + Tambah Rak / Lokasi
                  </button>
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {filteredLocations.map((loc) => {
                    const capacityNum = loc.max_capacity ? Number(loc.max_capacity) : 1000
                    const currentQty = stockByLocation.get(loc.id) || 0
                    const occupancyPercent = capacityNum > 0 ? Math.min(100, Math.round((currentQty / capacityNum) * 100)) : 0

                    return (
                      <div
                        key={loc.id}
                        className="bg-white rounded-xl border border-[#E2E8F0] p-4.5 hover:border-[#BFDBFE] hover:shadow-md transition-all flex flex-col justify-between"
                      >
                        <div>
                          {/* Card Header: Code & Badges */}
                          <div className="flex items-start justify-between gap-2">
                            <div>
                              <span className="font-mono text-base font-extrabold text-slate-900 tracking-tight">
                                {loc.code}
                              </span>
                              <h4 className="text-sm font-medium text-slate-700 mt-0.5 line-clamp-1">
                                {loc.name}
                              </h4>
                            </div>

                            <div className="flex flex-col items-end gap-1">
                              {loc.is_pallet ? (
                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-bold bg-amber-100 text-amber-800 border border-amber-300">
                                  PALLET LPN
                                </span>
                              ) : (
                                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-bold bg-blue-100 text-blue-800 border border-blue-200">
                                  {loc.type}
                                </span>
                              )}
                            </div>
                          </div>

                          {/* Barcode & Pallet metadata */}
                          <div className="mt-3.5 space-y-1.5 text-xs text-slate-600 bg-slate-50 p-2.5 rounded-lg border border-slate-100">
                            {loc.barcode && (
                              <div className="flex items-center gap-1.5">
                                <Barcode className="w-3.5 h-3.5 text-slate-500" />
                                <span className="font-mono">{loc.barcode}</span>
                              </div>
                            )}
                            {loc.pallet_number && (
                              <div className="flex items-center gap-1.5 text-amber-900 font-semibold">
                                <Box className="w-3.5 h-3.5 text-amber-600" />
                                <span>No. Pallet: {loc.pallet_number}</span>
                              </div>
                            )}
                            <div className="flex items-center justify-between text-slate-500 pt-1">
                              <span>Kapasitas Max:</span>
                              <span className="font-semibold text-slate-800">{capacityNum.toLocaleString()} unit</span>
                            </div>
                            <div className="flex items-center justify-between text-slate-500">
                              <span>Stok Fisik Saat Ini:</span>
                              <span className="font-semibold text-slate-800">{currentQty.toLocaleString()} unit</span>
                            </div>
                          </div>
                        </div>

                        {/* Capacity Utilization Bar */}
                        <div className="mt-4 pt-3 border-t border-slate-100">
                          <div className="flex items-center justify-between text-xs mb-1.5">
                            <span className="text-slate-500">Utilisasi Kapasitas</span>
                            <span
                              className={`font-bold ${
                                occupancyPercent > 85
                                  ? "text-rose-600"
                                  : occupancyPercent > 60
                                  ? "text-amber-600"
                                  : "text-emerald-600"
                              }`}
                            >
                              {occupancyPercent}%
                            </span>
                          </div>
                          <div className="w-full bg-slate-100 rounded-full h-2 overflow-hidden">
                            <div
                              className={`h-full rounded-full transition-all ${
                                occupancyPercent > 85
                                  ? "bg-rose-500"
                                  : occupancyPercent > 60
                                  ? "bg-amber-500"
                                  : "bg-emerald-500"
                              }`}
                              style={{ width: `${occupancyPercent}%` }}
                            />
                          </div>
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )}

          {/* TAB 2: Stock Movements History */}
          {activeTab === "movements" && (
            <div className="p-4 sm:p-6 space-y-4">
              <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 pb-2">
                <div>
                  <h3 className="text-base font-bold text-slate-900">
                    Riwayat Mutasi Stok Gudang
                  </h3>
                  <p className="text-xs text-slate-500">
                    Catatan perpindahan fisik barang antar gudang, rak simpan, dan area transit
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  <span className="inline-flex items-center gap-1.5 text-xs font-semibold px-2.5 py-1 rounded bg-slate-100 text-slate-700">
                    <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />
                    Tercatat Otomatis
                  </span>
                </div>
              </div>

              {/* Table */}
              <div className="overflow-x-auto border border-slate-200 rounded-xl">
                <table className="w-full text-left text-sm">
                  <thead className="bg-slate-100 text-xs font-semibold uppercase text-slate-600 border-b border-slate-200">
                    <tr>
                      <th className="px-4 py-3">No. Mutasi / Waktu</th>
                      <th className="px-4 py-3">Produk & SKU</th>
                      <th className="px-4 py-3">Lokasi Asal</th>
                      <th className="px-4 py-3">Lokasi Tujuan</th>
                      <th className="px-4 py-3 text-right">Kuantitas</th>
                      <th className="px-4 py-3 text-center">Status</th>
                      <th className="px-4 py-3">Dibuat / Disetujui / Operator</th>
                      <th className="px-4 py-3 text-right">Riwayat</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-200 bg-white">
                    {loadingMovements ? (
                      <tr>
                        <td colSpan={8} className="px-4 py-8 text-center text-slate-500 text-sm">
                          Memuat riwayat mutasi stok...
                        </td>
                      </tr>
                    ) : movements.length === 0 ? (
                      <tr>
                        <td colSpan={8} className="px-4 py-12 text-center text-slate-500 text-sm">
                          Belum ada riwayat pergerakan stok barang.
                        </td>
                      </tr>
                    ) : (
                      movements.map((mov) => (
                        <tr key={mov.id} className="hover:bg-slate-50/80 transition-colors">
                          <td className="px-4 py-3 font-mono text-xs text-slate-900">
                            <div className="font-bold text-[#2563EB]">{mov.movement_number}</div>
                            <div className="text-[11px] text-slate-400">
                              {new Date(mov.created_at).toLocaleString("id-ID", {
                                day: "numeric",
                                month: "short",
                                year: "numeric",
                                hour: "2-digit",
                                minute: "2-digit",
                              })}
                            </div>
                          </td>
                          <td className="px-4 py-3">
                            <div className="font-semibold text-slate-900">{mov.product_name || "Produk"}</div>
                            <div className="font-mono text-xs text-slate-500">{mov.sku || "-"}</div>
                          </td>
                          <td className="px-4 py-3">
                            {renderLocationBadge(mov.source_location_code || mov.source_location_id || "-")}
                          </td>
                          <td className="px-4 py-3">
                            {renderLocationBadge(mov.dest_location_code || mov.dest_location_id || "-")}
                          </td>
                          <td className="px-4 py-3 text-right font-mono font-bold text-slate-900">
                            {Number(mov.quantity) > 0 ? `+${Number(mov.quantity)}` : Number(mov.quantity)}{" "}
                            <span className="text-xs font-normal text-slate-500">unit</span>
                          </td>
                          <td className="px-4 py-3 text-center">
                            <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-bold bg-emerald-50 text-emerald-700 border border-emerald-200">
                              <CheckCircle2 className="w-3 h-3" />
                              {mov.status}
                            </span>
                          </td>
                          <td className="px-4 py-3 text-xs text-slate-600">
                            <div className="font-medium text-slate-800">
                              {mov.executed_by_name || (mov.executed_by ? `Petugas (${mov.executed_by.slice(0, 8)})` : "Sistem / Otomatis")}
                            </div>
                            <div className="text-[11px] text-slate-400">
                              Ref: <span className="font-mono font-medium text-slate-600">{mov.reference_type || "INTERNAL"}</span>
                            </div>
                          </td>
                          <td className="px-4 py-3 text-right">
                            <button
                              type="button"
                              onClick={() => setSelectedMovementForAudit(mov)}
                              className="inline-flex items-center gap-1 px-2.5 py-1 rounded-lg border border-slate-200 bg-white text-slate-700 hover:bg-slate-50 transition text-xs font-medium shadow-2xs"
                              title="Lihat Riwayat Aktivitas & Jejak Audit"
                            >
                              <History className="w-3.5 h-3.5 text-indigo-600" />
                              <span>Riwayat</span>
                            </button>
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
      </div>

      {/* ── DRAWER: Activity Timeline & Audit Trail (CR-05b) ── */}
      {selectedMovementForAudit && (
        <ActivityTimelineDrawer
          isOpen={!!selectedMovementForAudit}
          onClose={() => setSelectedMovementForAudit(null)}
          title={selectedMovementForAudit.movement_number}
          subtitle={`Mutasi Stok (${selectedMovementForAudit.reference_type || "WMS Ledger"})`}
          entityType="stock_movement"
          entityId={selectedMovementForAudit.id}
          actors={{
            executed_by_name: selectedMovementForAudit.executed_by_name || "Operator Gudang",
            executed_at: selectedMovementForAudit.created_at,
            created_at: selectedMovementForAudit.created_at,
            created_by_name: selectedMovementForAudit.executed_by_name || "Sistem WMS",
          }}
          metadata={{
            status: selectedMovementForAudit.status,
            reference_type: selectedMovementForAudit.reference_type,
            reference_number: selectedMovementForAudit.reference_id,
            product_name: selectedMovementForAudit.product_name,
            sku: selectedMovementForAudit.sku,
            source_location: selectedMovementForAudit.source_location_code || selectedMovementForAudit.source_location_id,
            dest_location: selectedMovementForAudit.dest_location_code || selectedMovementForAudit.dest_location_id,
            quantity: selectedMovementForAudit.quantity,
            batch_number: selectedMovementForAudit.batch_number,
          }}
        />
      )}

      {/* ── MODAL 1: Create Warehouse ── */}
      {showCreateWhModal && (
        <div className="fixed inset-0 z-50 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-2xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150">
            <div className="flex items-center justify-between pb-4 border-b border-slate-100">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-lg bg-[#EFF6FF] text-[#2563EB]">
                  <Building className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-slate-900">Tambah Gudang Baru</h3>
                  <p className="text-xs text-slate-500">Daftarkan entitas gudang fisik atau regional baru</p>
                </div>
              </div>
              <button
                onClick={() => setShowCreateWhModal(false)}
                className="p-2 text-slate-400 hover:text-slate-700 rounded-lg min-h-[44px] min-w-[44px] flex items-center justify-center"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {whFormError && (
              <div className="mt-4 p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-700 text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{whFormError}</span>
              </div>
            )}

            <form onSubmit={handleCreateWarehouse} className="mt-5 space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Kode Gudang *
                </label>
                <input
                  type="text"
                  required
                  placeholder="Misal: GUDANG-CKG-01"
                  value={newWhCode}
                  onChange={(e) => setNewWhCode(e.target.value.toUpperCase())}
                  className="w-full px-3.5 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm font-mono focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Nama Gudang *
                </label>
                <input
                  type="text"
                  required
                  placeholder="Misal: Gudang Distribusi Cakung"
                  value={newWhName}
                  onChange={(e) => setNewWhName(e.target.value)}
                  className="w-full px-3.5 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Alamat / Wilayah
                </label>
                <textarea
                  rows={3}
                  placeholder="Alamat lengkap fasilitas gudang..."
                  value={newWhAddress}
                  onChange={(e) => setNewWhAddress(e.target.value)}
                  className="w-full px-3.5 py-2.5 rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                />
              </div>

              <div className="pt-4 border-t border-slate-100 flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowCreateWhModal(false)}
                  className="px-4 min-h-[48px] rounded-lg text-sm font-medium text-slate-700 hover:bg-slate-100"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createWarehouseMutation.isPending}
                  className="px-5 min-h-[48px] rounded-lg text-sm font-semibold bg-[#2563EB] text-white hover:bg-[#1D4ED8] disabled:opacity-50"
                >
                  {createWarehouseMutation.isPending ? "Menyimpan..." : "Simpan Gudang"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── MODAL 2: Create Location / Rack ── */}
      {showCreateLocModal && (
        <div className="fixed inset-0 z-50 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-2xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150">
            <div className="flex items-center justify-between pb-4 border-b border-slate-100">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-lg bg-[#EFF6FF] text-[#2563EB]">
                  <Layers className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-slate-900">Tambah Rak / Lokasi Baru</h3>
                  <p className="text-xs text-slate-500">Konfigurasi koordinat Rak, Bin, atau Pallet LPN</p>
                </div>
              </div>
              <button
                onClick={() => setShowCreateLocModal(false)}
                className="p-2 text-slate-400 hover:text-slate-700 rounded-lg min-h-[44px] min-w-[44px] flex items-center justify-center"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {locFormError && (
              <div className="mt-4 p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-700 text-xs flex items-center gap-2">
                <AlertCircle className="w-4 h-4 shrink-0" />
                <span>{locFormError}</span>
              </div>
            )}

            <form onSubmit={handleCreateLocation} className="mt-5 space-y-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Pilih Gudang *
                </label>
                <select
                  value={newLocWarehouseId || (selectedWarehouseId !== "ALL" ? selectedWarehouseId : "")}
                  onChange={(e) => setNewLocWarehouseId(e.target.value)}
                  className="w-full px-3.5 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                >
                  <option value="">-- Pilih Gudang --</option>
                  {warehouses.map((wh) => (
                    <option key={wh.id} value={wh.id}>
                      [{wh.code}] {wh.name}
                    </option>
                  ))}
                </select>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    Kode Lokasi *
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="Contoh: RAK-A1-BIN04"
                    value={newLocCode}
                    onChange={(e) => setNewLocCode(e.target.value.toUpperCase())}
                    className="w-full px-3.5 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm font-mono focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    Nama / Deskripsi *
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="Contoh: Rak Susun Makanan"
                    value={newLocName}
                    onChange={(e) => setNewLocName(e.target.value)}
                    className="w-full px-3.5 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    Barcode Lokasi (Opsional)
                  </label>
                  <input
                    type="text"
                    placeholder="LOC-XXXXX"
                    value={newLocBarcode}
                    onChange={(e) => setNewLocBarcode(e.target.value)}
                    className="w-full px-3.5 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm font-mono focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    Tipe Lokasi
                  </label>
                  <select
                    value={newLocType}
                    onChange={(e) => setNewLocType(e.target.value as LocationType)}
                    className="w-full px-3.5 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                  >
                    <option value="INTERNAL">INTERNAL (Rak/Bin Biasa)</option>
                    <option value="TRANSIT">TRANSIT (Staging Area)</option>
                    <option value="VENDOR">Pemasok (Area Penerimaan)</option>
                    <option value="CUSTOMER">CUSTOMER (Pengiriman)</option>
                    <option value="SCRAP">SCRAP (Barang Rusak)</option>
                  </select>
                </div>
              </div>

              {/* Pallet Checkbox */}
              <div className="p-3 bg-slate-50 rounded-xl border border-slate-200 space-y-3">
                <label className="flex items-center gap-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={newLocIsPallet}
                    onChange={(e) => setNewLocIsPallet(e.target.checked)}
                    className="w-5 h-5 rounded text-[#2563EB] focus:ring-[#2563EB]"
                  />
                  <span className="text-sm font-semibold text-slate-800">
                    Ini adalah Pallet LPN Bergerak (License Plate Number)
                  </span>
                </label>

                {newLocIsPallet && (
                  <div>
                    <label className="block text-xs font-semibold text-slate-600 mb-1">
                      Nomor Pallet (LPN ID)
                    </label>
                    <input
                      type="text"
                      placeholder="Misal: PLT-JKT-0042"
                      value={newLocPalletNumber}
                      onChange={(e) => setNewLocPalletNumber(e.target.value.toUpperCase())}
                      className="w-full px-3 py-2 min-h-[44px] rounded-lg border border-slate-300 text-sm font-mono focus:ring-2 focus:ring-[#2563EB]"
                    />
                  </div>
                )}
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Kapasitas Maksimum Unit
                </label>
                <input
                  type="number"
                  value={newLocCapacity}
                  onChange={(e) => setNewLocCapacity(e.target.value)}
                  className="w-full px-3.5 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm font-mono focus:ring-2 focus:ring-[#2563EB] focus:outline-none"
                />
              </div>

              <div className="pt-4 border-t border-slate-100 flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowCreateLocModal(false)}
                  className="px-4 min-h-[48px] rounded-lg text-sm font-medium text-slate-700 hover:bg-slate-100"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createLocationMutation.isPending}
                  className="px-5 min-h-[48px] rounded-lg text-sm font-semibold bg-[#2563EB] text-white hover:bg-[#1D4ED8] disabled:opacity-50"
                >
                  {createLocationMutation.isPending ? "Menyimpan..." : "Simpan Lokasi"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <ExportModal<StockSummary>
        open={exportingStock}
        onClose={() => setExportingStock(false)}
        title="Stok & Lokasi Gudang"
        filename="stok-gudang"
        allRows={stockSummary}
        visibleRows={stockSummary}
        visibleSummary={[`Gudang: ${warehouses.find((w) => w.id === selectedWarehouseId)?.name ?? "Semua"}`]}
        columns={[
          { header: "SKU", value: (s) => s.sku, width: 16 },
          { header: "Nama Produk", value: (s) => s.product_name, width: 32 },
          { header: "Gudang", value: (s) => s.warehouse_name || "-", width: 22 },
          { header: "Lokasi", value: (s) => s.location_code || "-", width: 18 },
          { header: "Qty", value: (s) => Number(s.quantity) || 0, width: 12, align: "right" },
        ]}
        filters={[
          {
            type: "select",
            id: "wh",
            label: "Gudang",
            options: warehouses.map((w) => ({ value: w.id, label: w.name })),
            match: (s, v) => s.warehouse_id === v,
          },
          {
            type: "select",
            id: "qty",
            label: "Kondisi stok",
            options: [
              { value: "zero", label: "Habis (0)" },
              { value: "low", label: "Menipis (1–10)" },
              { value: "ok", label: "Aman (> 10)" },
            ],
            match: (s, v) => {
              const q = Number(s.quantity) || 0
              return v === "zero" ? q <= 0 : v === "low" ? q > 0 && q <= 10 : q > 10
            },
          },
        ]}
      />
    </div>
  )
}
