"use client"

import React, { useState, useMemo, useEffect, useRef } from "react"
import Link from "next/link"
import { useManualRefresh, AUTO_REFRESH_MS } from "@/hooks/useManualRefresh"
import { RefreshButton } from "@/components/ui/RefreshButton"
import {
  UploadCloud,
  CheckCircle2,
  AlertTriangle,
  AlertCircle,
  X,
  Search,
  Filter,
  ArrowRight,
  Plus,
  RefreshCw,
  ShoppingBag,
  Boxes,
  FileText,
  Layers,
  Copy,
  Check,
  Truck,
  Warehouse as WarehouseIcon,
  PackageCheck,
  PackageX,
  ChevronDown,
  ChevronUp,
  Info,
  Calendar,
  DollarSign,
  Globe,
  Tag,
  Clock,
  Send,
} from "lucide-react"
import {
  useWarehouses,
  useWarehouseLocations,
  useMarketplaceBatches,
  useDecideMarketplaceBatch,
  useMarketplaceOrders,
  useMarketplaceOrder,
  useMarketplaceSKUMappings,
  useImportMarketplaceOrders,
  useCreateSKUMapping,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import {
  MarketplaceChannel,
  MarketplaceOrderStatus,
  MarketplaceOrder,
  MarketplaceOrderItem,
  MarketplaceSKUMapping,
  Product,
  Warehouse,
  CreateSKUMappingPayload,
} from "@/lib/api"
import {
  parseMarketplaceCSV,
  SAMPLE_CSV_TEMPLATES,
  ParsedOrderPreview,
} from "@/lib/marketplace-parser"

// ── Channels Config ─────────────────────────────────────────────────────────────

interface ChannelConfig {
  label: string
  color: string
  bg: string
  border: string
  badgeClass: string
  dotClass: string
}

const CHANNEL_CONFIGS: Record<MarketplaceChannel, ChannelConfig> = {
  SHOPEE: {
    label: "Shopee",
    color: "#EE4D2D",
    bg: "bg-orange-50",
    border: "border-orange-200",
    badgeClass: "bg-orange-50 text-orange-700 border-orange-200",
    dotClass: "bg-orange-500",
  },
  TOKOPEDIA: {
    label: "Tokopedia",
    color: "#03AC0E",
    bg: "bg-emerald-50",
    border: "border-emerald-200",
    badgeClass: "bg-emerald-50 text-emerald-700 border-emerald-200",
    dotClass: "bg-emerald-500",
  },
  TIKTOK: {
    label: "TikTok Shop",
    color: "#000000",
    bg: "bg-zinc-100",
    border: "border-zinc-300",
    badgeClass: "bg-zinc-900 text-zinc-100 border-zinc-700",
    dotClass: "bg-zinc-900",
  },
  LAZADA: {
    label: "Lazada",
    color: "#0F146D",
    bg: "bg-blue-50",
    border: "border-blue-200",
    badgeClass: "bg-blue-50 text-blue-700 border-blue-200",
    dotClass: "bg-blue-600",
  },
  BLIBLI: {
    label: "Blibli",
    color: "#0095DA",
    bg: "bg-sky-50",
    border: "border-sky-200",
    badgeClass: "bg-sky-50 text-sky-700 border-sky-200",
    dotClass: "bg-sky-500",
  },
  OTHER: {
    label: "Lainnya / CSV",
    color: "#64748B",
    bg: "bg-slate-50",
    border: "border-slate-200",
    badgeClass: "bg-slate-100 text-slate-700 border-slate-300",
    dotClass: "bg-slate-500",
  },
}

// ── Currency Formatter ──────────────────────────────────────────────────────────

function formatIDR(amount: number | string | undefined | null): string {
  const num = typeof amount === "number" ? amount : Number(amount) || 0
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(num)
}

function formatDate(dateStr: string | undefined | null): string {
  if (!dateStr) return "-"
  try {
    const d = new Date(dateStr)
    return new Intl.DateTimeFormat("id-ID", {
      day: "numeric",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    }).format(d)
  } catch {
    return dateStr
  }
}

// ── Main Page Component ─────────────────────────────────────────────────────────

export default function MarketplacePage() {
  // Query data
  const { data: warehouses = [], isLoading: loadingWarehouses } = useWarehouses()
  const { data: products = [], isLoading: loadingProducts } = useProducts()
  const {
    data: batches = [],
    isLoading: loadingBatches,
    refetch: refetchBatches,
  } = useMarketplaceBatches()
  const decideBatch = useDecideMarketplaceBatch()
  const pendingBatches = batches.filter((b) => b.status === "PENDING_APPROVAL")
  const {
    data: orders = [],
    isLoading: loadingOrders,
    refetch: refetchOrders,
    dataUpdatedAt,
  } = useMarketplaceOrders()
  const {
    data: skuMappings = [],
    isLoading: loadingMappings,
    refetch: refetchMappings,
  } = useMarketplaceSKUMappings()
  const { refresh, status: refreshStatus, refreshError } = useManualRefresh([refetchBatches, refetchOrders, refetchMappings], { autoRefreshMs: AUTO_REFRESH_MS })

  // Mutations
  const importOrdersMutation = useImportMarketplaceOrders()
  const createMappingMutation = useCreateSKUMapping()

  // Navigation tab state: 'import' | 'orders' | 'mappings'
  const [activeTab, setActiveTab] = useState<"import" | "orders" | "mappings">("import")

  // Target warehouse state for import
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>("")
  const targetWarehouseId = selectedWarehouseId || warehouses[0]?.id || ""

  // M5: stock is deducted from an explicit INTERNAL rack of the chosen warehouse.
  const { data: warehouseLocations = [] } = useWarehouseLocations(targetWarehouseId || null)
  const sourceRacks = useMemo(
    () => warehouseLocations.filter((l) => l.type === "INTERNAL" && l.warehouse_id === targetWarehouseId),
    [warehouseLocations, targetWarehouseId]
  )
  const [selectedSourceLocationId, setSelectedSourceLocationId] = useState<string>("")
  const sourceLocationId = sourceRacks.some((l) => l.id === selectedSourceLocationId)
    ? selectedSourceLocationId
    : ""

  // Toast / notification state
  const [notification, setNotification] = useState<{
    type: "success" | "error" | "info"
    title: string
    message: string
  } | null>(null)

  const showNotification = (
    type: "success" | "error" | "info",
    title: string,
    message: string
  ) => {
    setNotification({ type, title, message })
    setTimeout(() => setNotification(null), 6000)
  }

  // ── Tab 1: Import Orders State ────────────────────────────────────────────────
  const [importChannel, setImportChannel] = useState<MarketplaceChannel | "AUTO">("AUTO")
  const [fileToImport, setFileToImport] = useState<File | null>(null)
  const [rawCsvText, setRawCsvText] = useState<string>("")
  const [fileName, setFileName] = useState<string>("")
  const [isDragOver, setIsDragOver] = useState(false)
  const [expandedPreviewRows, setExpandedPreviewRows] = useState<Record<string, boolean>>({})

  // Parsing result
  const parsedPreview = useMemo(() => {
    if (!rawCsvText.trim()) return null
    const targetChannel = importChannel === "AUTO" ? undefined : importChannel
    return parseMarketplaceCSV(rawCsvText, targetChannel)
  }, [rawCsvText, importChannel])

  // Import Result Banner State
  const [lastImportResult, setLastImportResult] = useState<{
    batchNumber: string
    total: number
    processed: number
    failed: number
    unmapped: number
    channel: MarketplaceChannel
  } | null>(null)

  // ── Tab 2: Orders Ledger Filter State ─────────────────────────────────────────
  const [orderChannelFilter, setOrderChannelFilter] = useState<string>("ALL")
  const [orderStatusFilter, setOrderStatusFilter] = useState<string>("ALL")
  const [orderWarehouseFilter, setOrderWarehouseFilter] = useState<string>("ALL")
  const [orderSearchQuery, setOrderSearchQuery] = useState<string>("")

  // Order Detail Drawer State
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null)
  const { data: selectedOrderDetail, isLoading: loadingOrderDetail } =
    useMarketplaceOrder(selectedOrderId)

  // ── Tab 3: SKU Mapping Filter & Modal State ───────────────────────────────────
  const [mappingChannelFilter, setMappingChannelFilter] = useState<string>("ALL")
  const [mappingSearchQuery, setMappingSearchQuery] = useState<string>("")
  const [showMappingModal, setShowMappingModal] = useState<boolean>(false)

  // Modal form states
  const [mapChannel, setMapChannel] = useState<MarketplaceChannel>("SHOPEE")
  const [mapExternalSku, setMapExternalSku] = useState<string>("")
  const [mapExternalName, setMapExternalName] = useState<string>("")
  const [mapProductId, setMapProductId] = useState<string>("")
  const [mapMultiplier, setMapMultiplier] = useState<string>("1")
  const [mapModalError, setMapModalError] = useState<string | null>(null)

  // Copied indicator state for tracking numbers
  const [copiedTracking, setCopiedTracking] = useState<string | null>(null)
  const handleCopyTracking = (trackNum: string) => {
    navigator.clipboard.writeText(trackNum)
    setCopiedTracking(trackNum)
    setTimeout(() => setCopiedTracking(null), 2000)
  }

  // ── Summary KPI Calculations ──────────────────────────────────────────────────
  const summaryKpis = useMemo(() => {
    const totalBatches = batches.length
    const successfulOrders = orders.filter((o) => o.status === "COMPLETED").length
    const netRevenue = orders
      .filter((o) => o.status === "COMPLETED")
      .reduce((acc, curr) => acc + (Number(curr.net_amount) || 0), 0)

    // Unmapped SKUs from orders and batches
    const unmappedOrders = orders.filter((o) => o.status === "UNMAPPED_SKU")
    const unmappedSkuList = new Set<string>()
    unmappedOrders.forEach((o) => {
      o.items?.forEach((it) => {
        if (!it.is_mapped) {
          unmappedSkuList.add(it.external_sku)
        }
      })
    })

    return {
      totalBatches,
      successfulOrders,
      netRevenue,
      unmappedOrdersCount: unmappedOrders.length,
      unmappedSkuCount: unmappedSkuList.size,
      unmappedSkuList: Array.from(unmappedSkuList),
    }
  }, [batches, orders])

  // ── File Handlers ─────────────────────────────────────────────────────────────
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleFileSelect = (file: File) => {
    if (!file) return
    if (file.size > 5 * 1024 * 1024) {
      showNotification(
        "error",
        "Ukuran File Terlalu Besar",
        "Batas maksimum upload adalah 5 MB sesuai kebijakan API WMS."
      )
      return
    }

    setFileToImport(file)
    setFileName(file.name)

    const reader = new FileReader()
    reader.onload = (e) => {
      const text = e.target?.result as string
      setRawCsvText(text || "")
    }
    reader.onerror = () => {
      showNotification("error", "Gagal Membaca File", "Tidak dapat membaca konten file.")
    }
    reader.readAsText(file)
  }

  const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragOver(false)
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFileSelect(e.dataTransfer.files[0])
    }
  }

  const loadSampleTemplate = (channel: keyof typeof SAMPLE_CSV_TEMPLATES) => {
    const csv = SAMPLE_CSV_TEMPLATES[channel]
    setRawCsvText(csv)
    setFileName(`sample_${channel.toLowerCase()}_orders.csv`)
    setFileToImport(null)
    setImportChannel(channel)
    showNotification(
      "info",
      `Contoh Data ${channel} Dimuat`,
      `Memuat sampel pesanan penjualan ${channel} untuk pengujian cepat.`
    )
  }

  const handleClearImport = () => {
    setFileToImport(null)
    setRawCsvText("")
    setFileName("")
    setExpandedPreviewRows({})
    if (fileInputRef.current) fileInputRef.current.value = ""
  }

  // ── Submit Order Import ────────────────────────────────────────────────────────
  const handleExecuteImport = async () => {
    if (!targetWarehouseId) {
      showNotification("error", "Pilih Gudang Tujuan", "Pilih gudang tujuan sebelum impor.")
      return
    }
    if (!sourceLocationId) {
      showNotification("error", "Pilih Rak Sumber", "Pilih rak sumber pemotongan stok sebelum impor.")
      return
    }
    if (!parsedPreview || parsedPreview.orders.length === 0) {
      showNotification("error", "Data Pesanan Kosong", "Unggah file CSV dengan data pesanan valid.")
      return
    }

    const channelToUse =
      importChannel === "AUTO" ? parsedPreview.detectedChannel : importChannel

    try {
      if (fileToImport) {
        // Use FormData for direct file upload
        const formData = new FormData()
        formData.append("file", fileToImport)
        formData.append("warehouse_id", targetWarehouseId)
        formData.append("source_location_id", sourceLocationId)
        formData.append("channel", channelToUse)

        const res = await importOrdersMutation.mutateAsync(formData)
        const batch = res.data?.batch || res.batch
        setLastImportResult({
          batchNumber: batch?.batch_number || `BATCH-MKT-${Date.now()}`,
          total: parsedPreview.orders.length,
          processed: batch?.processed_orders ?? parsedPreview.validOrderCount,
          failed: batch?.failed_orders ?? parsedPreview.errorOrderCount,
          unmapped: batch?.unmapped_skus ?? 0,
          channel: channelToUse,
        })
      } else {
        // Use Structured JSON payload
        const payload = {
          warehouse_id: targetWarehouseId,
          source_location_id: sourceLocationId,
          channel: channelToUse,
          file_name: fileName || "manual_csv_import.csv",
          orders: parsedPreview.orders.filter((o) => o.isValid),
        }

        const res = await importOrdersMutation.mutateAsync(payload)
        const batch = res.data?.batch || res.batch
        setLastImportResult({
          batchNumber: batch?.batch_number || `BATCH-MKT-${Date.now()}`,
          total: parsedPreview.orders.length,
          processed: batch?.processed_orders ?? parsedPreview.validOrderCount,
          failed: batch?.failed_orders ?? parsedPreview.errorOrderCount,
          unmapped: batch?.unmapped_skus ?? 0,
          channel: channelToUse,
        })
      }

      showNotification(
        "success",
        "Impor Pesanan Selesai",
        `${parsedPreview.validOrderCount} pesanan tersimpan dan menunggu persetujuan owner/admin lain sebelum stok dipotong.`
      )
      handleClearImport()
      refetchBatches()
      refetchOrders()
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : "Terjadi kesalahan saat memproses impor."
      showNotification("error", "Gagal Memproses Impor", errMsg)
    }
  }

  // ── Open Mapping Modal (Optionally Pre-filled) ─────────────────────────────────
  const openCreateMapping = (sku?: string, channel?: MarketplaceChannel) => {
    setMapChannel(channel || "SHOPEE")
    setMapExternalSku(sku || "")
    setMapExternalName("")
    setMapProductId(products[0]?.id || "")
    setMapMultiplier("1")
    setMapModalError(null)
    setShowMappingModal(true)
  }

  // ── Submit SKU Mapping ─────────────────────────────────────────────────────────
  const handleSaveMapping = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!mapExternalSku.trim()) {
      setMapModalError("Kode SKU Marketplace wajib diisi.")
      return
    }
    if (!mapProductId) {
      setMapModalError("Pilih Produk Master Internal.")
      return
    }
    const multNum = parseFloat(mapMultiplier)
    if (isNaN(multNum) || multNum <= 0) {
      setMapModalError("Faktor pengali (multiplier) harus berupa angka positif > 0.")
      return
    }

    try {
      const payload: CreateSKUMappingPayload = {
        channel_name: mapChannel,
        external_sku: mapExternalSku.trim(),
        external_name: mapExternalName.trim() || undefined,
        product_id: mapProductId,
        multiplier: multNum,
        mapping_type: "MARKETPLACE",
      }

      await createMappingMutation.mutateAsync(payload)
      showNotification(
        "success",
        "Pemetaan SKU Berhasil Disimpan",
        `SKU ${mapExternalSku} terhubung ke produk internal. Pesanan tertunda akan otomatis di-reproses.`
      )
      setShowMappingModal(false)
      refetchMappings()
      refetchOrders()
      refetchBatches()
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : "Gagal menyimpan pemetaan SKU."
      setMapModalError(errMsg)
    }
  }

  // ── Filtered Orders Ledger ────────────────────────────────────────────────────
  const filteredOrders = useMemo(() => {
    return orders.filter((o) => {
      const matchesChannel =
        orderChannelFilter === "ALL" || o.channel === orderChannelFilter
      const matchesStatus =
        orderStatusFilter === "ALL" || o.status === orderStatusFilter
      const matchesWarehouse =
        orderWarehouseFilter === "ALL" || o.warehouse_id === orderWarehouseFilter
      const query = orderSearchQuery.toLowerCase().trim()
      const matchesSearch =
        query === "" ||
        o.external_order_id.toLowerCase().includes(query) ||
        (o.customer_name && o.customer_name.toLowerCase().includes(query)) ||
        (o.tracking_number && o.tracking_number.toLowerCase().includes(query)) ||
        (o.courier && o.courier.toLowerCase().includes(query)) ||
        (o.items &&
          o.items.some(
            (it) =>
              it.external_sku.toLowerCase().includes(query) ||
              it.item_name.toLowerCase().includes(query)
          ))

      return matchesChannel && matchesStatus && matchesWarehouse && matchesSearch
    })
  }, [orders, orderChannelFilter, orderStatusFilter, orderWarehouseFilter, orderSearchQuery])

  // ── Filtered SKU Mappings ─────────────────────────────────────────────────────
  const filteredMappings = useMemo(() => {
    return skuMappings.filter((m) => {
      const matchesChannel =
        mappingChannelFilter === "ALL" || m.channel_name === mappingChannelFilter
      const query = mappingSearchQuery.toLowerCase().trim()
      const matchesSearch =
        query === "" ||
        m.external_sku.toLowerCase().includes(query) ||
        (m.external_name && m.external_name.toLowerCase().includes(query)) ||
        (m.product_sku && m.product_sku.toLowerCase().includes(query)) ||
        (m.product_name && m.product_name.toLowerCase().includes(query))

      return matchesChannel && matchesSearch
    })
  }, [skuMappings, mappingChannelFilter, mappingSearchQuery])

  // Status Badge Component
  const renderStatusBadge = (status: MarketplaceOrderStatus) => {
    switch (status) {
      case "COMPLETED":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-600" />
            COMPLETED
          </span>
        )
      case "UNMAPPED_SKU":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-amber-50 text-amber-700 border border-amber-200">
            <span className="w-1.5 h-1.5 rounded-full bg-amber-600 animate-pulse" />
            UNMAPPED_SKU
          </span>
        )
      case "STOCK_INSUFFICIENT":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-rose-50 text-rose-700 border border-rose-200">
            <span className="w-1.5 h-1.5 rounded-full bg-rose-600" />
            STOK KURANG
          </span>
        )
      case "FAILED":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-red-50 text-red-700 border border-red-200">
            <span className="w-1.5 h-1.5 rounded-full bg-red-600" />
            FAILED
          </span>
        )
      case "PROCESSING":
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-200">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-600 animate-spin" />
            PROCESSING
          </span>
        )
      default:
        return (
          <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-semibold bg-slate-50 text-slate-700 border border-slate-200">
            <span className="w-1.5 h-1.5 rounded-full bg-slate-400" />
            {status}
          </span>
        )
    }
  }

  // Channel Badge Component
  const renderChannelBadge = (channel: MarketplaceChannel) => {
    const conf = CHANNEL_CONFIGS[channel] || CHANNEL_CONFIGS.OTHER
    return (
      <span
        className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded text-xs font-semibold border ${conf.badgeClass}`}
      >
        <span className={`w-1.5 h-1.5 rounded-full ${conf.dotClass}`} />
        {conf.label}
      </span>
    )
  }

  return (
    <div className="min-h-screen bg-slate-50/60 pb-16">
      {/* ── Notification Banner ────────────────────────────────────────────── */}
      {notification && (
        <div
          role="status"
          aria-live="polite"
          className={`fixed top-4 right-4 z-50 max-w-md p-4 rounded-xl border shadow-xl flex items-start gap-3 transition-all ${
            notification.type === "success"
              ? "bg-emerald-50 text-emerald-900 border-emerald-300"
              : notification.type === "error"
              ? "bg-rose-50 text-rose-900 border-rose-300"
              : "bg-blue-50 text-blue-900 border-blue-300"
          }`}
        >
          {notification.type === "success" ? (
            <CheckCircle2 className="w-5 h-5 text-emerald-600 shrink-0 mt-0.5" />
          ) : notification.type === "error" ? (
            <AlertCircle className="w-5 h-5 text-rose-600 shrink-0 mt-0.5" />
          ) : (
            <Info className="w-5 h-5 text-blue-600 shrink-0 mt-0.5" />
          )}
          <div className="flex-1 min-w-0">
            <h4 className="font-semibold text-sm">{notification.title}</h4>
            <p className="text-xs text-slate-600 mt-0.5 leading-relaxed">
              {notification.message}
            </p>
          </div>
          <button
            onClick={() => setNotification(null)}
            className="p-1 rounded text-slate-400 hover:text-slate-600 hover:bg-black/5"
            aria-label="Tutup notifikasi"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      )}

      {/* ── Page Header ────────────────────────────────────────────────────── */}
      <div className="bg-white border-b border-slate-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
            <div>
              <div className="flex items-center gap-2 text-xs text-slate-500 font-medium mb-1">
                <Link href="/wms" className="hover:text-blue-600 transition-colors">
                  Warehouse & Stock
                </Link>
                <span>/</span>
                <span className="text-slate-900 font-semibold">Omnichannel Marketplace</span>
              </div>
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-xl bg-blue-600 text-white shadow-sm shadow-blue-500/20">
                  <ShoppingBag className="w-6 h-6" />
                </div>
                <div>
                  <h1 className="text-2xl font-bold text-slate-900 tracking-tight">
                    Marketplace Sales Import & SKU Mapping
                  </h1>
                  <p className="text-sm text-slate-500">
                    Impor pesanan Shopee, Tokopedia, TikTok Shop, Lazada, resolusi packaging multiplier & potong stok otomatis untuk pelanggan.
                  </p>
                </div>
              </div>
            </div>

            {/* Quick Action Buttons */}
            <div className="flex items-center gap-2">
              <RefreshButton
              updatedAt={dataUpdatedAt}
                status={refreshStatus}
                error={refreshError}
                onClick={() => void refresh()}
                showLabel
                iconClassName="w-4 h-4"
                className="inline-flex items-center justify-center gap-2 min-h-[48px] px-3.5 py-2 rounded-lg text-sm font-medium text-slate-700 bg-white border border-slate-300 hover:bg-slate-50"
              />

              <button
                onClick={() => openCreateMapping()}
                className="inline-flex items-center justify-center min-h-[48px] px-4 py-2 rounded-lg text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 active:bg-blue-800 shadow-sm shadow-blue-500/20 transition-colors"
              >
                <Plus className="w-4 h-4 mr-2" />
                Tambah Pemetaan SKU
              </button>
            </div>
          </div>

          {/* ── Top Summary Cards ──────────────────────────────────────────── */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mt-6">
            {/* Card 1: Total Sesi Impor */}
            <div className="bg-slate-50/80 rounded-xl border border-slate-200/80 p-4 transition-all hover:bg-white hover:shadow-sm">
              <div className="flex items-center justify-between text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
                <span>Total Sesi Impor</span>
                <span className="p-1.5 rounded-lg bg-blue-50 text-blue-600">
                  <FileText className="w-4 h-4" />
                </span>
              </div>
              <div className="text-2xl font-bold text-slate-900">
                {summaryKpis.totalBatches} Sesi
              </div>
              <div className="text-xs text-slate-500 mt-1 flex items-center gap-1.5">
                <span className="inline-block w-2 h-2 rounded-full bg-emerald-500" />
                <span>Terverifikasi dengan Idempotency Batch</span>
              </div>
            </div>

            {/* Card 2: Pesanan Sukses */}
            <div className="bg-slate-50/80 rounded-xl border border-slate-200/80 p-4 transition-all hover:bg-white hover:shadow-sm">
              <div className="flex items-center justify-between text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
                <span>Pesanan Sukses</span>
                <span className="p-1.5 rounded-lg bg-emerald-50 text-emerald-600">
                  <PackageCheck className="w-4 h-4" />
                </span>
              </div>
              <div className="text-2xl font-bold text-emerald-600">
                {summaryKpis.successfulOrders} Pesanan
              </div>
              <div className="text-xs text-slate-500 mt-1">
                Stok terpotong untuk{" "}
                <span className="font-semibold text-emerald-700 bg-emerald-50 px-1 py-0.5 rounded">
                  Pelanggan Tujuan
                </span>
              </div>
            </div>

            {/* Card 3: Omset Bersih */}
            <div className="bg-slate-50/80 rounded-xl border border-slate-200/80 p-4 transition-all hover:bg-white hover:shadow-sm">
              <div className="flex items-center justify-between text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
                <span>Omset Bersih Marketplace</span>
                <span className="p-1.5 rounded-lg bg-indigo-50 text-indigo-600">
                  <DollarSign className="w-4 h-4" />
                </span>
              </div>
              <div className="text-2xl font-bold text-slate-900">
                {formatIDR(summaryKpis.netRevenue)}
              </div>
              <div className="text-xs text-slate-500 mt-1">
                Total belanja bersih setelah fee channel
              </div>
            </div>

            {/* Card 4: SKU Belum Terpetakan (*Unmapped Alert*) */}
            <div
              className={`rounded-xl border p-4 transition-all ${
                summaryKpis.unmappedSkuCount > 0
                  ? "bg-amber-50/70 border-amber-300 shadow-sm"
                  : "bg-slate-50/80 border-slate-200/80"
              }`}
            >
              <div className="flex items-center justify-between text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2">
                <span
                  className={
                    summaryKpis.unmappedSkuCount > 0 ? "text-amber-800 font-bold" : ""
                  }
                >
                  SKU Belum Terpetakan
                </span>
                <span
                  className={`p-1.5 rounded-lg ${
                    summaryKpis.unmappedSkuCount > 0
                      ? "bg-amber-200/70 text-amber-800 animate-pulse"
                      : "bg-slate-100 text-slate-500"
                  }`}
                >
                  <AlertTriangle className="w-4 h-4" />
                </span>
              </div>
              <div className="flex items-baseline justify-between">
                <div
                  className={`text-2xl font-bold ${
                    summaryKpis.unmappedSkuCount > 0 ? "text-amber-700" : "text-slate-900"
                  }`}
                >
                  {summaryKpis.unmappedSkuCount} SKU
                </div>
                {summaryKpis.unmappedSkuCount > 0 && (
                  <button
                    onClick={() => setActiveTab("mappings")}
                    className="text-xs font-semibold text-amber-800 hover:text-amber-900 underline underline-offset-2 min-h-[32px] inline-flex items-center"
                  >
                    Petakan Sekarang &rarr;
                  </button>
                )}
              </div>
              <div className="text-xs text-slate-500 mt-1">
                {summaryKpis.unmappedOrdersCount > 0
                  ? `${summaryKpis.unmappedOrdersCount} pesanan menunggu pemetaan SKU`
                  : "Semua channel SKU terhubung ke produk internal"}
              </div>
            </div>
          </div>

          {/* ── 3 Main Navigation Tabs ─────────────────────────────────────── */}
          <div className="flex border-b border-slate-200 mt-8 gap-2 overflow-x-auto pb-px">
            <button
              onClick={() => setActiveTab("import")}
              className={`min-h-[48px] px-5 py-2.5 font-semibold text-sm rounded-t-lg transition-colors flex items-center gap-2 border-b-2 shrink-0 ${
                activeTab === "import"
                  ? "border-blue-600 text-blue-600 bg-blue-50/50"
                  : "border-transparent text-slate-600 hover:text-slate-900 hover:bg-slate-50"
              }`}
            >
              <UploadCloud className="w-4 h-4" />
              <span>Tab 1: Impor Pesanan (Import Orders)</span>
            </button>

            <button
              onClick={() => setActiveTab("orders")}
              className={`min-h-[48px] px-5 py-2.5 font-semibold text-sm rounded-t-lg transition-colors flex items-center gap-2 border-b-2 shrink-0 ${
                activeTab === "orders"
                  ? "border-blue-600 text-blue-600 bg-blue-50/50"
                  : "border-transparent text-slate-600 hover:text-slate-900 hover:bg-slate-50"
              }`}
            >
              <ShoppingBag className="w-4 h-4" />
              <span>Tab 2: Daftar Pesanan (Orders Ledger)</span>
              <span className="ml-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-200 text-slate-700">
                {orders.length}
              </span>
            </button>

            <button
              onClick={() => setActiveTab("mappings")}
              className={`min-h-[48px] px-5 py-2.5 font-semibold text-sm rounded-t-lg transition-colors flex items-center gap-2 border-b-2 shrink-0 ${
                activeTab === "mappings"
                  ? "border-blue-600 text-blue-600 bg-blue-50/50"
                  : "border-transparent text-slate-600 hover:text-slate-900 hover:bg-slate-50"
              }`}
            >
              <Boxes className="w-4 h-4" />
              <span>Tab 3: Pemetaan SKU & Resolver</span>
              {summaryKpis.unmappedSkuCount > 0 ? (
                <span className="ml-1 px-2 py-0.5 rounded-full text-xs font-bold bg-amber-100 text-amber-800 border border-amber-300 animate-pulse">
                  {summaryKpis.unmappedSkuCount} Unmapped
                </span>
              ) : (
                <span className="ml-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-200 text-slate-700">
                  {skuMappings.length}
                </span>
              )}
            </button>
          </div>
        </div>
      </div>

      {/* ── Main Workspace Body ────────────────────────────────────────────── */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 pt-6">
        {/* =================================================================== */}
        {/* TAB 1: IMPOR PESANAN                                                */}
        {/* =================================================================== */}
        {activeTab === "import" && (
          <div className="space-y-6">
            {/* Last Import Success/Warning Result Banner */}
            {lastImportResult && (
              <div
                className={`p-4 rounded-xl border shadow-sm ${
                  lastImportResult.unmapped > 0
                    ? "bg-amber-50 border-amber-300 text-amber-900"
                    : "bg-emerald-50 border-emerald-300 text-emerald-900"
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex items-start gap-3">
                    {lastImportResult.unmapped > 0 ? (
                      <AlertTriangle className="w-6 h-6 text-amber-600 shrink-0 mt-0.5" />
                    ) : (
                      <CheckCircle2 className="w-6 h-6 text-emerald-600 shrink-0 mt-0.5" />
                    )}
                    <div>
                      <h3 className="font-bold text-base">
                        Sesi Impor Selesai: {lastImportResult.batchNumber}
                      </h3>
                      <p className="text-sm mt-1 leading-relaxed">
                        Total {lastImportResult.total} pesanan diproses. Sukses:{" "}
                        <span className="font-bold text-emerald-700">
                          {lastImportResult.processed}
                        </span>
                        , Gagal / Duplikat:{" "}
                        <span className="font-bold text-rose-700">
                          {lastImportResult.failed}
                        </span>
                        , SKU Belum Terpetakan:{" "}
                        <span className="font-bold text-amber-800">
                          {lastImportResult.unmapped}
                        </span>
                        .
                      </p>
                      {lastImportResult.unmapped > 0 && (
                        <div className="mt-3 flex items-center gap-3">
                          <button
                            onClick={() => setActiveTab("mappings")}
                            className="inline-flex items-center min-h-[48px] px-4 py-2 rounded-lg text-sm font-bold text-white bg-amber-600 hover:bg-amber-700 transition-colors shadow-sm"
                          >
                            Petakan SKU Belum Terdaftar Sekarang &rarr;
                          </button>
                        </div>
                      )}
                    </div>
                  </div>
                  <button
                    onClick={() => setLastImportResult(null)}
                    className="p-1 rounded text-slate-400 hover:text-slate-600"
                    aria-label="Tutup ringkasan impor"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              </div>
            )}

            {/* Import Configuration Card */}
            <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-6">
              <h2 className="text-lg font-bold text-slate-900 mb-4 flex items-center gap-2">
                <UploadCloud className="w-5 h-5 text-blue-600" />
                <span>Upload Berkas Penjualan Marketplace</span>
              </h2>

              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-6">
                {/* Warehouse Selector */}
                <div>
                  <label
                    htmlFor="warehouse-select"
                    className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5"
                  >
                    Gudang Pemotongan Stok <span className="text-rose-500">*</span>
                  </label>
                  <div className="relative">
                    <select
                      id="warehouse-select"
                      value={targetWarehouseId}
                      onChange={(e) => setSelectedWarehouseId(e.target.value)}
                      disabled={loadingWarehouses}
                      className="w-full min-h-[48px] px-3.5 py-2.5 rounded-lg bg-white border border-slate-300 text-slate-900 text-sm font-medium focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                    >
                      {warehouses.map((wh) => (
                        <option key={wh.id} value={wh.id}>
                          {wh.name} ({wh.code})
                        </option>
                      ))}
                    </select>
                  </div>
                  <p className="text-xs text-slate-500 mt-1">
                    Stok barang yang terpetakan akan otomatis dipotong dari gudang ini untuk pesanan pelanggan.
                  </p>
                </div>

                {/* Source Rack Selector (M5) */}
                <div>
                  <label
                    htmlFor="source-rack-select"
                    className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5"
                  >
                    Rak Sumber <span className="text-rose-500">*</span>
                  </label>
                  <select
                    id="source-rack-select"
                    value={sourceLocationId}
                    onChange={(e) => setSelectedSourceLocationId(e.target.value)}
                    className="w-full min-h-[48px] px-3.5 py-2.5 rounded-lg bg-white border border-slate-300 text-slate-900 text-sm font-medium focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                  >
                    <option value="">Pilih rak...</option>
                    {sourceRacks.map((l) => (
                      <option key={l.id} value={l.id}>
                        {l.name} ({l.code})
                      </option>
                    ))}
                  </select>
                  <p className="text-xs text-slate-500 mt-1">
                    {sourceRacks.length === 0
                      ? "Gudang ini belum punya rak internal."
                      : "Stok pesanan dipotong dari rak ini."}
                  </p>
                </div>

                {/* Channel Selector */}
                <div>
                  <label
                    htmlFor="channel-select"
                    className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5"
                  >
                    Kanal Penjualan (Marketplace)
                  </label>
                  <select
                    id="channel-select"
                    value={importChannel}
                    onChange={(e) =>
                      setImportChannel(e.target.value as MarketplaceChannel | "AUTO")
                    }
                    className="w-full min-h-[48px] px-3.5 py-2.5 rounded-lg bg-white border border-slate-300 text-slate-900 text-sm font-medium focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                  >
                    <option value="AUTO">✨ Deteksi Otomatis (Auto-Detect Header)</option>
                    <option value="SHOPEE">Shopee (Ekspor Pesanan Massal)</option>
                    <option value="TOKOPEDIA">Tokopedia (Daftar Pesanan Tokopedia)</option>
                    <option value="TIKTOK">TikTok Shop (Order Export)</option>
                    <option value="LAZADA">Lazada (Lazada Seller Center)</option>
                    <option value="BLIBLI">Blibli Omnichannel</option>
                    <option value="OTHER">Generic CSV / TSV Standar</option>
                  </select>
                  <p className="text-xs text-slate-500 mt-1">
                    Sistem dapat mendeteksi template Shopee, Tokopedia, TikTok, Lazada otomatis dari nama kolom.
                  </p>
                </div>
              </div>

              {/* Drag and Drop Zone */}
              <div
                onDragOver={(e) => {
                  e.preventDefault()
                  setIsDragOver(true)
                }}
                onDragLeave={() => setIsDragOver(false)}
                onDrop={handleDrop}
                className={`relative border-2 border-dashed rounded-xl p-8 text-center transition-all ${
                  isDragOver
                    ? "border-blue-500 bg-blue-50/40"
                    : rawCsvText
                    ? "border-emerald-400 bg-emerald-50/20"
                    : "border-slate-300 hover:border-slate-400 bg-slate-50/50"
                }`}
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  id="csv-file-input"
                  accept=".csv,.tsv,.txt,.xlsx,.xls,.json"
                  onChange={(e) => {
                    if (e.target.files && e.target.files[0]) {
                      handleFileSelect(e.target.files[0])
                    }
                  }}
                  className="hidden"
                />

                <div className="flex flex-col items-center justify-center gap-2">
                  <div className="p-3 rounded-full bg-blue-100 text-blue-600 mb-1">
                    <UploadCloud className="w-8 h-8" />
                  </div>

                  {fileName ? (
                    <div className="space-y-1">
                      <div className="inline-flex items-center gap-2 px-3 py-1 rounded-md bg-emerald-100 text-emerald-800 text-sm font-semibold">
                        <Check className="w-4 h-4" />
                        <span>{fileName}</span>
                      </div>
                      <p className="text-xs text-slate-500">
                        {parsedPreview?.orders.length || 0} baris pesanan siap diproses.
                      </p>
                    </div>
                  ) : (
                    <>
                      <h3 className="text-base font-semibold text-slate-900">
                        Tarik & Lepas File Pesanan CSV/Excel ke Sini
                      </h3>
                      <p className="text-xs text-slate-500 max-w-md">
                        Mendukung format ekspor resmi dari Shopee, Tokopedia, TikTok Shop, Lazada, atau file CSV generik (Maksimal 5 MB).
                      </p>
                    </>
                  )}

                  <div className="flex items-center gap-3 mt-4">
                    <label
                      htmlFor="csv-file-input"
                      className="cursor-pointer min-h-[48px] px-5 py-2.5 rounded-lg text-sm font-semibold text-white bg-blue-600 hover:bg-blue-700 active:bg-blue-800 shadow-sm transition-colors inline-flex items-center"
                    >
                      <FileText className="w-4 h-4 mr-2" />
                      {fileName ? "Ganti File" : "Pilih File dari Komputer"}
                    </label>

                    {rawCsvText && (
                      <button
                        type="button"
                        onClick={handleClearImport}
                        className="min-h-[48px] px-4 py-2 rounded-lg text-sm font-medium text-slate-600 hover:text-slate-800 hover:bg-slate-200/60 transition-colors"
                      >
                        Hapus Pilihan
                      </button>
                    )}
                  </div>
                </div>
              </div>

              {/* Sample 1-Click Loaders */}
              <div className="mt-4 pt-4 border-t border-slate-100 flex flex-wrap items-center justify-between gap-2">
                <span className="text-xs text-slate-500 font-medium">
                  Atau uji coba cepat dengan data sampel resmi:
                </span>
                <div className="flex flex-wrap items-center gap-2">
                  <button
                    type="button"
                    onClick={() => loadSampleTemplate("SHOPEE")}
                    className="min-h-[36px] px-3 py-1 rounded text-xs font-semibold bg-orange-50 text-orange-700 border border-orange-200 hover:bg-orange-100 transition-colors"
                  >
                    + Sampel Shopee
                  </button>
                  <button
                    type="button"
                    onClick={() => loadSampleTemplate("TOKOPEDIA")}
                    className="min-h-[36px] px-3 py-1 rounded text-xs font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200 hover:bg-emerald-100 transition-colors"
                  >
                    + Sampel Tokopedia
                  </button>
                  <button
                    type="button"
                    onClick={() => loadSampleTemplate("TIKTOK")}
                    className="min-h-[36px] px-3 py-1 rounded text-xs font-semibold bg-zinc-100 text-zinc-800 border border-zinc-300 hover:bg-zinc-200 transition-colors"
                  >
                    + Sampel TikTok
                  </button>
                  <button
                    type="button"
                    onClick={() => loadSampleTemplate("LAZADA")}
                    className="min-h-[36px] px-3 py-1 rounded text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-200 hover:bg-blue-100 transition-colors"
                  >
                    + Sampel Lazada
                  </button>
                </div>
              </div>
            </div>

            {/* ── Client-Side CSV Preview & Validation Table ───────────────── */}
            {parsedPreview && (
              <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
                {/* Preview Toolbar */}
                <div className="p-4 sm:p-6 border-b border-slate-200 bg-slate-50/60 flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                  <div>
                    <div className="flex items-center gap-2.5">
                      <span className="text-base font-bold text-slate-900">
                        Pratinjau Data Pesanan ({parsedPreview.orders.length} Pesanan)
                      </span>
                      {renderChannelBadge(parsedPreview.detectedChannel)}
                    </div>
                    <div className="flex flex-wrap items-center gap-3 text-xs text-slate-500 mt-1">
                      <span className="text-emerald-700 font-semibold flex items-center gap-1">
                        <Check className="w-3.5 h-3.5 text-emerald-600" />
                        {parsedPreview.validOrderCount} Valid
                      </span>
                      {parsedPreview.errorOrderCount > 0 && (
                        <span className="text-rose-600 font-semibold flex items-center gap-1">
                          <AlertCircle className="w-3.5 h-3.5 text-rose-600" />
                          {parsedPreview.errorOrderCount} Bermasalah
                        </span>
                      )}
                      <span>&bull;</span>
                      <span>
                        Estimasi Omset:{" "}
                        <strong className="text-slate-900 font-mono">
                          {formatIDR(parsedPreview.totalRevenue)}
                        </strong>
                      </span>
                    </div>
                  </div>

                  {/* One-Click Import Execution Button */}
                  <button
                    onClick={handleExecuteImport}
                    disabled={
                      importOrdersMutation.isPending || parsedPreview.validOrderCount === 0
                    }
                    className="min-h-[48px] px-6 py-2.5 rounded-lg text-sm font-bold text-white bg-orange-600 hover:bg-orange-700 active:bg-orange-800 disabled:opacity-50 disabled:cursor-not-allowed shadow-md shadow-orange-600/20 transition-all flex items-center justify-center shrink-0"
                  >
                    {importOrdersMutation.isPending ? (
                      <>
                        <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                        Memproses Impor & Mutasi Stok...
                      </>
                    ) : (
                      <>
                        <Send className="w-4 h-4 mr-2" />
                        Proses Impor & Potong Stok ({parsedPreview.validOrderCount})
                      </>
                    )}
                  </button>
                </div>

                {/* Validation Errors Notice if any */}
                {parsedPreview.errors.length > 0 && (
                  <div className="p-4 bg-rose-50 border-b border-rose-200 text-rose-800 text-xs flex items-center gap-2">
                    <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
                    <span>{parsedPreview.errors.join("; ")}</span>
                  </div>
                )}

                {/* Preview Table */}
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-50 text-slate-500 font-semibold text-xs uppercase tracking-wider border-b border-slate-200">
                      <tr>
                        <th className="px-4 py-3">Order ID Marketplace</th>
                        <th className="px-4 py-3">Pembeli & Kontak</th>
                        <th className="px-4 py-3">Kurir & Resi</th>
                        <th className="px-4 py-3">Barang (Items)</th>
                        <th className="px-4 py-3 text-right">Total Belanja</th>
                        <th className="px-4 py-3 text-center">Status Validasi</th>
                        <th className="px-4 py-3 text-right">Detail</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 font-normal">
                      {parsedPreview.orders.map((ord) => {
                        const isExpanded = expandedPreviewRows[ord.external_order_id] || false
                        const hasUnmapped = ord.items.some((it) => {
                          const mapping = skuMappings.find(
                            (m) =>
                              m.external_sku.toLowerCase() === it.external_sku.toLowerCase() &&
                              m.channel_name === ord.detectedChannel
                          )
                          return !mapping
                        })

                        return (
                          <React.Fragment key={ord.external_order_id}>
                            <tr className="hover:bg-slate-50/70 transition-colors">
                              <td className="px-4 py-3">
                                <div className="font-mono text-xs font-bold text-slate-900">
                                  {ord.external_order_id}
                                </div>
                                <div className="text-[11px] text-slate-400">
                                  {formatDate(ord.order_date)}
                                </div>
                              </td>

                              <td className="px-4 py-3">
                                <div className="font-medium text-slate-800 text-xs">
                                  {ord.customer_name || "Pelanggan Marketplace"}
                                </div>
                                {ord.customer_phone && (
                                  <div className="text-[11px] text-slate-500">
                                    {ord.customer_phone}
                                  </div>
                                )}
                              </td>

                              <td className="px-4 py-3">
                                <div className="text-xs text-slate-700 font-medium">
                                  {ord.courier || "Standard"}
                                </div>
                                {ord.tracking_number && (
                                  <div className="font-mono text-[11px] text-blue-600">
                                    {ord.tracking_number}
                                  </div>
                                )}
                              </td>

                              <td className="px-4 py-3">
                                <div className="text-xs text-slate-800 font-medium">
                                  {ord.items.length} item
                                </div>
                                <div className="text-[11px] text-slate-500 truncate max-w-[200px]">
                                  {ord.items[0]?.item_name || ord.items[0]?.external_sku}
                                  {ord.items.length > 1 && ` (+${ord.items.length - 1} lainnya)`}
                                </div>
                              </td>

                              <td className="px-4 py-3 text-right">
                                <div className="font-mono text-xs font-bold text-slate-900">
                                  {formatIDR(ord.total_amount)}
                                </div>
                                {Number(ord.shipping_fee) > 0 && (
                                  <div className="text-[10px] text-slate-400">
                                    Ongkir: {formatIDR(ord.shipping_fee)}
                                  </div>
                                )}
                              </td>

                              <td className="px-4 py-3 text-center">
                                {ord.isValid ? (
                                  hasUnmapped ? (
                                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-amber-50 text-amber-700 border border-amber-200">
                                      <AlertTriangle className="w-3 h-3 text-amber-600" />
                                      SKU Belum Petakan
                                    </span>
                                  ) : (
                                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
                                      <Check className="w-3 h-3 text-emerald-600" />
                                      Valid & Terpetakan
                                    </span>
                                  )
                                ) : (
                                  <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-rose-50 text-rose-700 border border-rose-200">
                                    <X className="w-3 h-3 text-rose-600" />
                                    Error
                                  </span>
                                )}
                              </td>

                              <td className="px-4 py-3 text-right">
                                <button
                                  type="button"
                                  onClick={() =>
                                    setExpandedPreviewRows((prev) => ({
                                      ...prev,
                                      [ord.external_order_id]: !isExpanded,
                                    }))
                                  }
                                  className="min-h-[44px] px-2 py-1 text-xs font-semibold text-blue-600 hover:text-blue-800 hover:underline"
                                >
                                  {isExpanded ? "Tutup" : "Lihat Item"}
                                </button>
                              </td>
                            </tr>

                            {/* Expanded items row */}
                            {isExpanded && (
                              <tr className="bg-slate-50/50">
                                <td colSpan={7} className="p-4 pl-8 border-y border-slate-100">
                                  <div className="text-xs font-semibold text-slate-700 mb-2">
                                    Rincian Produk dalam Pesanan #{ord.external_order_id}:
                                  </div>
                                  <div className="space-y-1.5">
                                    {ord.items.map((it, idx) => {
                                      const matchedMap = skuMappings.find(
                                        (m) =>
                                          m.external_sku.toLowerCase() ===
                                            it.external_sku.toLowerCase() &&
                                          m.channel_name === ord.detectedChannel
                                      )

                                      return (
                                        <div
                                          key={idx}
                                          className="flex flex-wrap items-center justify-between p-2.5 rounded-lg bg-white border border-slate-200 text-xs"
                                        >
                                          <div className="flex items-center gap-2">
                                            <span className="font-mono font-bold text-slate-800">
                                              {it.external_sku}
                                            </span>
                                            <span className="text-slate-500">&mdash;</span>
                                            <span className="text-slate-700 font-medium">
                                              {it.item_name}
                                            </span>
                                          </div>

                                          <div className="flex items-center gap-4 text-slate-600">
                                            <span>Qty: {it.quantity}</span>
                                            <span>@{formatIDR(it.unit_price)}</span>
                                            <span className="font-semibold text-slate-900">
                                              Subtotal: {formatIDR(it.subtotal)}
                                            </span>

                                            {matchedMap ? (
                                              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
                                                <Check className="w-3 h-3 text-emerald-600" />
                                                Petakan: {matchedMap.product_name || matchedMap.product_sku} (x{matchedMap.multiplier})
                                              </span>
                                            ) : (
                                              <button
                                                onClick={() =>
                                                  openCreateMapping(
                                                    it.external_sku,
                                                    ord.detectedChannel
                                                  )
                                                }
                                                className="inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-bold bg-amber-100 text-amber-800 border border-amber-300 hover:bg-amber-200 transition-colors"
                                              >
                                                <Plus className="w-3 h-3" />
                                                Petakan Sekarang
                                              </button>
                                            )}
                                          </div>
                                        </div>
                                      )
                                    })}
                                  </div>
                                </td>
                              </tr>
                            )}
                          </React.Fragment>
                        )
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}

        {/* =================================================================== */}
        {/* TAB 2: DAFTAR PESANAN (ORDERS LEDGER)                               */}
        {/* =================================================================== */}
        {activeTab === "orders" && (
          <div className="space-y-4">
            {pendingBatches.length > 0 && (
              <section aria-label="Batch menunggu persetujuan" className="bg-amber-50 rounded-xl border border-amber-200 p-4 space-y-3">
                <h3 className="text-sm font-semibold text-amber-900">
                  Batch Menunggu Persetujuan ({pendingBatches.length})
                </h3>
                <p className="text-xs text-amber-800">
                  Stok baru dipotong setelah batch disetujui owner/admin yang bukan pengunggah.
                </p>
                <ul className="divide-y divide-amber-200">
                  {pendingBatches.map((b) => (
                    <li key={b.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
                      <div className="text-xs">
                        <div className="font-mono font-semibold text-slate-800">{b.batch_number}</div>
                        <div className="text-slate-600">
                          {b.channel} · {b.processed_orders} pesanan · {b.file_name}
                        </div>
                      </div>
                      <div className="flex gap-2">
                        <button
                          type="button"
                          disabled={decideBatch.isPending}
                          onClick={() =>
                            decideBatch.mutate(
                              { id: b.id, action: "approve" },
                              {
                                onSuccess: () => showNotification("success", "Batch Disetujui", `${b.batch_number}: stok telah dipotong.`),
                                onError: (err) => showNotification("error", "Gagal Menyetujui", err instanceof Error ? err.message : "Gagal menyetujui batch"),
                              }
                            )
                          }
                          className="min-h-[40px] px-3 rounded-lg bg-emerald-600 text-white text-xs font-semibold disabled:opacity-50"
                        >
                          Setujui & Potong Stok
                        </button>
                        <button
                          type="button"
                          disabled={decideBatch.isPending}
                          onClick={() =>
                            decideBatch.mutate(
                              { id: b.id, action: "reject" },
                              {
                                onSuccess: () => showNotification("success", "Batch Ditolak", `${b.batch_number}: tidak ada stok yang dipotong.`),
                                onError: (err) => showNotification("error", "Gagal Menolak", err instanceof Error ? err.message : "Gagal menolak batch"),
                              }
                            )
                          }
                          className="min-h-[40px] px-3 rounded-lg border border-rose-300 text-rose-700 text-xs font-semibold disabled:opacity-50"
                        >
                          Tolak
                        </button>
                      </div>
                    </li>
                  ))}
                </ul>
              </section>
            )}
            {/* Filter Controls Toolbar */}
            <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                {/* Search query */}
                <div className="relative">
                  <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
                  <input
                    type="text"
                    placeholder="Cari No. Pesanan, Resi, Pembeli..."
                    value={orderSearchQuery}
                    onChange={(e) => setOrderSearchQuery(e.target.value)}
                    className="w-full min-h-[48px] pl-9 pr-3.5 py-2 rounded-lg bg-slate-50 border border-slate-200 text-sm focus:bg-white focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none"
                  />
                  {orderSearchQuery && (
                    <button
                      onClick={() => setOrderSearchQuery("")}
                      className="absolute right-3 top-1/2 -translate-y-1/2 text-slate-400 hover:text-slate-600"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  )}
                </div>

                {/* Channel Filter */}
                <div>
                  <select
                    value={orderChannelFilter}
                    onChange={(e) => setOrderChannelFilter(e.target.value)}
                    className="w-full min-h-[48px] px-3.5 py-2 rounded-lg bg-slate-50 border border-slate-200 text-sm font-medium text-slate-700 focus:bg-white focus:ring-2 focus:ring-blue-500 outline-none"
                  >
                    <option value="ALL">Semua Kanal Marketplace</option>
                    <option value="SHOPEE">Shopee</option>
                    <option value="TOKOPEDIA">Tokopedia</option>
                    <option value="TIKTOK">TikTok Shop</option>
                    <option value="LAZADA">Lazada</option>
                    <option value="BLIBLI">Blibli</option>
                    <option value="OTHER">Lainnya</option>
                  </select>
                </div>

                {/* Status Filter */}
                <div>
                  <select
                    value={orderStatusFilter}
                    onChange={(e) => setOrderStatusFilter(e.target.value)}
                    className="w-full min-h-[48px] px-3.5 py-2 rounded-lg bg-slate-50 border border-slate-200 text-sm font-medium text-slate-700 focus:bg-white focus:ring-2 focus:ring-blue-500 outline-none"
                  >
                    <option value="ALL">Semua Status Pesanan</option>
                    <option value="COMPLETED">COMPLETED (Sukses & Potong Stok)</option>
                    <option value="UNMAPPED_SKU">UNMAPPED_SKU (Perlu Pemetaan)</option>
                    <option value="STOCK_INSUFFICIENT">STOCK_INSUFFICIENT (Stok Kurang)</option>
                    <option value="PROCESSING">PROCESSING</option>
                    <option value="FAILED">FAILED</option>
                  </select>
                </div>

                {/* Warehouse Filter */}
                <div>
                  <select
                    value={orderWarehouseFilter}
                    onChange={(e) => setOrderWarehouseFilter(e.target.value)}
                    className="w-full min-h-[48px] px-3.5 py-2 rounded-lg bg-slate-50 border border-slate-200 text-sm font-medium text-slate-700 focus:bg-white focus:ring-2 focus:ring-blue-500 outline-none"
                  >
                    <option value="ALL">Semua Gudang</option>
                    {warehouses.map((wh) => (
                      <option key={wh.id} value={wh.id}>
                        {wh.name}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            </div>

            {/* Orders Table & Mobile Cards */}
            <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
              {loadingOrders ? (
                <div className="p-12 text-center text-slate-500">
                  <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-blue-600" />
                  <p className="text-sm font-medium">Memuat daftar pesanan marketplace...</p>
                </div>
              ) : filteredOrders.length === 0 ? (
                <div className="p-12 text-center text-slate-500">
                  <ShoppingBag className="w-10 h-10 mx-auto mb-3 text-slate-300" />
                  <h3 className="font-semibold text-slate-800 text-base">
                    Tidak ada pesanan yang sesuai filter
                  </h3>
                  <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto">
                    Coba sesuaikan kata kunci pencarian atau ubah filter kanal dan status di atas.
                  </p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-50 text-slate-500 font-semibold text-xs uppercase tracking-wider border-b border-slate-200">
                      <tr>
                        <th className="px-4 py-3">Kanal & Order ID</th>
                        <th className="px-4 py-3">Pembeli</th>
                        <th className="px-4 py-3">Kurir & Resi</th>
                        <th className="px-4 py-3">Produk & Kuantitas</th>
                        <th className="px-4 py-3 text-right">Omset Bersih</th>
                        <th className="px-4 py-3 text-center">Status</th>
                        <th className="px-4 py-3 text-right">Aksi</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 font-normal">
                      {filteredOrders.map((ord) => (
                        <tr
                          key={ord.id}
                          className="hover:bg-slate-50/70 transition-colors"
                        >
                          {/* Channel & External Order ID */}
                          <td className="px-4 py-3">
                            <div className="mb-1.5">{renderChannelBadge(ord.channel)}</div>
                            <div className="font-mono text-xs font-bold text-slate-900">
                              {ord.external_order_id}
                            </div>
                            <div className="text-[11px] text-slate-400">
                              {formatDate(ord.order_date || ord.created_at)}
                            </div>
                          </td>

                          {/* Customer */}
                          <td className="px-4 py-3">
                            <div className="font-medium text-slate-900 text-xs">
                              {ord.customer_name || "Pelanggan Marketplace"}
                            </div>
                            {ord.customer_phone && (
                              <div className="text-[11px] text-slate-500">
                                {ord.customer_phone}
                              </div>
                            )}
                            <div className="text-[10px] text-slate-400 truncate max-w-[180px]">
                              {ord.shipping_address || "Alamat tercatat"}
                            </div>
                          </td>

                          {/* Courier & Tracking */}
                          <td className="px-4 py-3">
                            <div className="text-xs text-slate-700 font-medium">
                              {ord.courier || "Standard"}
                            </div>
                            {ord.tracking_number ? (
                              <div className="flex items-center gap-1 mt-0.5">
                                <span className="font-mono text-[11px] text-blue-600 bg-blue-50 px-1 py-0.5 rounded">
                                  {ord.tracking_number}
                                </span>
                                <button
                                  onClick={() => handleCopyTracking(ord.tracking_number!)}
                                  className="p-1 rounded text-slate-400 hover:text-slate-600"
                                  title="Salin nomor resi"
                                >
                                  {copiedTracking === ord.tracking_number ? (
                                    <Check className="w-3 h-3 text-emerald-600" />
                                  ) : (
                                    <Copy className="w-3 h-3" />
                                  )}
                                </button>
                              </div>
                            ) : (
                              <span className="text-[11px] text-slate-400">-</span>
                            )}
                          </td>

                          {/* Items Preview */}
                          <td className="px-4 py-3">
                            <div className="text-xs text-slate-800 font-semibold">
                              {ord.items?.length || 1} item
                            </div>
                            <div className="text-[11px] text-slate-600 truncate max-w-[220px]">
                              {ord.items?.[0]?.item_name || ord.items?.[0]?.external_sku || "Barang"}
                              {ord.items && ord.items.length > 1 && (
                                <span className="text-slate-400 font-normal">
                                  {" "}
                                  (+{ord.items.length - 1} lainnya)
                                </span>
                              )}
                            </div>
                            {ord.warehouse_name && (
                              <div className="text-[10px] text-slate-400 mt-0.5 flex items-center gap-1">
                                <WarehouseIcon className="w-3 h-3 text-slate-400" />
                                <span>{ord.warehouse_name}</span>
                              </div>
                            )}
                          </td>

                          {/* Net Revenue */}
                          <td className="px-4 py-3 text-right">
                            <div className="font-mono text-xs font-bold text-slate-900">
                              {formatIDR(ord.net_amount || ord.total_amount)}
                            </div>
                            {Number(ord.marketplace_fee) > 0 && (
                              <div className="text-[10px] text-slate-400">
                                Fee: {formatIDR(ord.marketplace_fee)}
                              </div>
                            )}
                          </td>

                          {/* Status */}
                          <td className="px-4 py-3 text-center">
                            {renderStatusBadge(ord.status)}
                          </td>

                          {/* Detail Action */}
                          <td className="px-4 py-3 text-right">
                            <button
                              onClick={() => setSelectedOrderId(ord.id)}
                              className="min-h-[48px] px-3.5 py-1.5 rounded-lg text-xs font-semibold text-blue-600 hover:text-blue-800 hover:bg-blue-50 transition-colors inline-flex items-center"
                            >
                              Detail &rarr;
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}

        {/* =================================================================== */}
        {/* TAB 3: PEMETAAN SKU & RESOLVER                                      */}
        {/* =================================================================== */}
        {activeTab === "mappings" && (
          <div className="space-y-6">
            {/* Unmapped SKU Alert Banner if any pending orders need mapping */}
            {summaryKpis.unmappedSkuList.length > 0 && (
              <div className="bg-amber-50 rounded-xl border border-amber-300 p-5 shadow-sm">
                <div className="flex items-start gap-3">
                  <div className="p-2 rounded-lg bg-amber-100 text-amber-800 mt-0.5">
                    <AlertTriangle className="w-5 h-5" />
                  </div>
                  <div className="flex-1">
                    <h3 className="text-base font-bold text-amber-950">
                      Terdapat {summaryKpis.unmappedSkuList.length} External SKU Belum Terpetakan!
                    </h3>
                    <p className="text-xs text-amber-800 mt-1 leading-relaxed">
                      Pesanan penjualan yang memiliki SKU di bawah ini ditahan sementara. Segera petakan ke produk master internal agar stok gudang dapat dipotong otomatis untuk pesanan pelanggan.
                    </p>

                    {/* Quick action buttons for unmapped SKUs */}
                    <div className="mt-3 flex flex-wrap gap-2">
                      {summaryKpis.unmappedSkuList.map((sku) => (
                        <div
                          key={sku}
                          className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-white border border-amber-300 text-xs shadow-sm"
                        >
                          <span className="font-mono font-bold text-slate-800">{sku}</span>
                          <button
                            onClick={() => openCreateMapping(sku)}
                            className="min-h-[32px] px-2.5 py-1 rounded bg-amber-600 hover:bg-amber-700 text-white font-semibold text-xs transition-colors flex items-center gap-1"
                          >
                            <Plus className="w-3 h-3" />
                            Petakan Sekarang
                          </button>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* Catalog Toolbar */}
            <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-4">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
                <div className="flex flex-wrap items-center gap-3 flex-1">
                  {/* Search SKU */}
                  <div className="relative min-w-[240px] flex-1 sm:max-w-xs">
                    <Search className="w-4 h-4 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
                    <input
                      type="text"
                      placeholder="Cari SKU Marketplace atau Produk Internal..."
                      value={mappingSearchQuery}
                      onChange={(e) => setMappingSearchQuery(e.target.value)}
                      className="w-full min-h-[48px] pl-9 pr-3.5 py-2 rounded-lg bg-slate-50 border border-slate-200 text-sm focus:bg-white focus:ring-2 focus:ring-blue-500 outline-none"
                    />
                  </div>

                  {/* Channel Filter */}
                  <select
                    value={mappingChannelFilter}
                    onChange={(e) => setMappingChannelFilter(e.target.value)}
                    className="min-h-[48px] px-3.5 py-2 rounded-lg bg-slate-50 border border-slate-200 text-sm font-medium text-slate-700 focus:bg-white focus:ring-2 focus:ring-blue-500 outline-none"
                  >
                    <option value="ALL">Semua Kanal</option>
                    <option value="SHOPEE">Shopee</option>
                    <option value="TOKOPEDIA">Tokopedia</option>
                    <option value="TIKTOK">TikTok Shop</option>
                    <option value="LAZADA">Lazada</option>
                    <option value="BLIBLI">Blibli</option>
                  </select>
                </div>

                <button
                  onClick={() => openCreateMapping()}
                  className="min-h-[48px] px-4 py-2.5 rounded-lg text-sm font-bold text-white bg-blue-600 hover:bg-blue-700 active:bg-blue-800 shadow-sm transition-colors flex items-center justify-center shrink-0"
                >
                  <Plus className="w-4 h-4 mr-2" />
                  Tambah Pemetaan Baru
                </button>
              </div>
            </div>

            {/* Active SKU Mappings Table */}
            <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
              {loadingMappings ? (
                <div className="p-12 text-center text-slate-500">
                  <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-blue-600" />
                  <p className="text-sm font-medium">Memuat katalog pemetaan SKU...</p>
                </div>
              ) : filteredMappings.length === 0 ? (
                <div className="p-12 text-center text-slate-500">
                  <Boxes className="w-10 h-10 mx-auto mb-3 text-slate-300" />
                  <h3 className="font-semibold text-slate-800 text-base">
                    Belum ada pemetaan SKU yang sesuai
                  </h3>
                  <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto">
                    Klik tombol &quot;Tambah Pemetaan Baru&quot; di atas untuk menghubungkan SKU marketplace ke produk internal gudang.
                  </p>
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="bg-slate-50 text-slate-500 font-semibold text-xs uppercase tracking-wider border-b border-slate-200">
                      <tr>
                        <th className="px-4 py-3">Kanal</th>
                        <th className="px-4 py-3">SKU Marketplace (External SKU)</th>
                        <th className="px-4 py-3">Produk Master Internal</th>
                        <th className="px-4 py-3 text-center">Faktor Pengali (Multiplier)</th>
                        <th className="px-4 py-3 text-center">Status Resolver</th>
                        <th className="px-4 py-3 text-right">Tanggal Dibuat</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-100 font-normal">
                      {filteredMappings.map((map) => (
                        <tr
                          key={map.id}
                          className="hover:bg-slate-50/70 transition-colors"
                        >
                          {/* Channel */}
                          <td className="px-4 py-3">
                            {renderChannelBadge(map.channel_name as MarketplaceChannel)}
                          </td>

                          {/* External SKU & Name */}
                          <td className="px-4 py-3">
                            <div className="font-mono text-xs font-bold text-slate-900">
                              {map.external_sku}
                            </div>
                            {map.external_name && (
                              <div className="text-[11px] text-slate-500 truncate max-w-xs">
                                {map.external_name}
                              </div>
                            )}
                          </td>

                          {/* Internal Product */}
                          <td className="px-4 py-3">
                            <div className="font-medium text-slate-900 text-xs">
                              {map.product_name || "Produk Internal"}
                            </div>
                            <div className="font-mono text-[11px] text-blue-600">
                              {map.product_sku || map.product_id}
                            </div>
                          </td>

                          {/* Multiplier Badge */}
                          <td className="px-4 py-3 text-center">
                            <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-bold bg-blue-50 text-blue-700 border border-blue-200 font-mono">
                              &times; {map.multiplier}{" "}
                              {Number(map.multiplier) > 1 ? "Pcs (Bundle)" : "Pcs"}
                            </span>
                          </td>

                          {/* Resolver Status */}
                          <td className="px-4 py-3 text-center">
                            {map.status === "PENDING" ? (
                              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-amber-50 text-amber-700 border border-amber-200">
                                <Clock className="w-3 h-3 text-amber-600" />
                                Menunggu Persetujuan
                              </span>
                            ) : map.status === "REJECTED" ? (
                              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-rose-50 text-rose-700 border border-rose-200">
                                <AlertTriangle className="w-3 h-3 text-rose-600" />
                                Ditolak
                              </span>
                            ) : (
                              <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
                                <Check className="w-3 h-3 text-emerald-600" />
                                Terhubung & Aktif
                              </span>
                            )}
                          </td>

                          {/* Created At */}
                          <td className="px-4 py-3 text-right text-xs text-slate-400">
                            {formatDate(map.created_at)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        )}
      </main>

      {/* =================================================================== */}
      {/* SLIDE-OVER DETAIL DRAWER: ORDER DETAILS                             */}
      {/* =================================================================== */}
      {selectedOrderId && (
        <div className="fixed inset-0 z-50 flex justify-end">
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-black/40 backdrop-blur-sm transition-opacity"
            onClick={() => setSelectedOrderId(null)}
            aria-hidden="true"
          />

          {/* Drawer Panel */}
          <div className="relative z-50 w-full max-w-2xl bg-white shadow-2xl flex flex-col border-l border-slate-200 h-full overflow-hidden">
            {/* Drawer Header */}
            <div className="p-6 border-b border-slate-200 bg-slate-50 flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="p-2.5 rounded-xl bg-blue-100 text-blue-700">
                  <ShoppingBag className="w-5 h-5" />
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <h2 className="text-lg font-bold text-slate-900">
                      Pesanan #{selectedOrderDetail?.external_order_id || selectedOrderId}
                    </h2>
                    {selectedOrderDetail &&
                      renderChannelBadge(selectedOrderDetail.channel)}
                  </div>
                  <p className="text-xs text-slate-500">
                    Waktu Pesanan: {formatDate(selectedOrderDetail?.order_date)}
                  </p>
                </div>
              </div>

              <button
                onClick={() => setSelectedOrderId(null)}
                className="min-h-[48px] min-w-[48px] p-2 rounded-lg text-slate-400 hover:text-slate-600 hover:bg-slate-200 transition-colors flex items-center justify-center"
                aria-label="Tutup detail pesanan"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {/* Drawer Content */}
            <div className="flex-1 overflow-y-auto p-6 space-y-6">
              {loadingOrderDetail ? (
                <div className="p-12 text-center text-slate-500">
                  <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-blue-600" />
                  <p className="text-sm font-medium">Memuat detail pesanan...</p>
                </div>
              ) : selectedOrderDetail ? (
                <>
                  {/* Status Banner */}
                  <div className="flex items-center justify-between p-4 rounded-xl border bg-slate-50">
                    <div>
                      <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider block mb-1">
                        Status Pemenuhan WMS
                      </span>
                      {renderStatusBadge(selectedOrderDetail.status)}
                    </div>
                    {selectedOrderDetail.status === "UNMAPPED_SKU" && (
                      <button
                        onClick={() => {
                          const unmappedItem = selectedOrderDetail.items?.find((i) => !i.is_mapped)
                          openCreateMapping(
                            unmappedItem?.external_sku,
                            selectedOrderDetail.channel
                          )
                        }}
                        className="min-h-[48px] px-4 py-2 rounded-lg text-xs font-bold text-white bg-amber-600 hover:bg-amber-700 transition-colors shadow-sm inline-flex items-center"
                      >
                        <Plus className="w-4 h-4 mr-1.5" />
                        Petakan SKU Ini Sekarang
                      </button>
                    )}
                  </div>

                  {/* Virtual Double-Entry Movement Card */}
                  <div className="p-4 rounded-xl border border-emerald-200 bg-emerald-50/50">
                    <div className="flex items-start gap-2.5">
                      <CheckCircle2 className="w-5 h-5 text-emerald-600 shrink-0 mt-0.5" />
                      <div>
                        <h4 className="font-bold text-xs text-emerald-950 uppercase tracking-wider">
                          Referensi Mutasi Stok Double-Entry: Pelanggan Tujuan
                        </h4>
                        <p className="text-xs text-emerald-800 mt-1 leading-relaxed">
                          Saat pesanan berstatus <span className="font-semibold">COMPLETED</span>, mutasi stok WMS diterbitkan secara otomatis dari gudang utama ke lokasi virtual <span className="font-semibold">Pelanggan Tujuan</span> dengan referensi <span className="font-mono font-bold">MARKETPLACE</span>.
                        </p>
                      </div>
                    </div>
                  </div>

                  {/* Customer & Shipping Details */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="p-4 rounded-xl border border-slate-200 bg-white space-y-1.5">
                      <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                        Informasi Penerima
                      </span>
                      <div className="text-sm font-bold text-slate-900">
                        {selectedOrderDetail.customer_name || "Pelanggan Marketplace"}
                      </div>
                      <div className="text-xs text-slate-600">
                        {selectedOrderDetail.customer_phone || "Nomor telepon tidak tersedia"}
                      </div>
                      <div className="text-xs text-slate-500 pt-1 leading-relaxed">
                        {selectedOrderDetail.shipping_address || "Alamat tercatat"}
                      </div>
                    </div>

                    <div className="p-4 rounded-xl border border-slate-200 bg-white space-y-1.5">
                      <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">
                        Ekspedisi & Gudang
                      </span>
                      <div className="text-sm font-bold text-slate-900 flex items-center gap-1.5">
                        <Truck className="w-4 h-4 text-slate-500" />
                        <span>{selectedOrderDetail.courier || "Standard Courier"}</span>
                      </div>
                      <div className="flex items-center gap-2 pt-0.5">
                        <span className="text-xs text-slate-500">No. Resi:</span>
                        <span className="font-mono text-xs font-bold text-blue-600">
                          {selectedOrderDetail.tracking_number || "-"}
                        </span>
                      </div>
                      <div className="text-xs text-slate-500 flex items-center gap-1 pt-1">
                        <WarehouseIcon className="w-3.5 h-3.5 text-slate-400" />
                        <span>{selectedOrderDetail.warehouse_name || "Gudang Distribusi"}</span>
                      </div>
                    </div>
                  </div>

                  {/* Financial Breakdown */}
                  <div className="p-4 rounded-xl border border-slate-200 bg-white space-y-2">
                    <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider block mb-2">
                      Rincian Keuangan Pesanan
                    </span>
                    <div className="flex justify-between text-xs text-slate-600">
                      <span>Total Harga Produk</span>
                      <span className="font-mono font-medium">
                        {formatIDR(selectedOrderDetail.total_amount)}
                      </span>
                    </div>
                    <div className="flex justify-between text-xs text-slate-600">
                      <span>Ongkos Kirim Dibayar Pembeli</span>
                      <span className="font-mono font-medium">
                        {formatIDR(selectedOrderDetail.shipping_fee)}
                      </span>
                    </div>
                    <div className="flex justify-between text-xs text-slate-600">
                      <span>Biaya Layanan / Marketplace Fee</span>
                      <span className="font-mono text-rose-600 font-medium">
                        - {formatIDR(selectedOrderDetail.marketplace_fee)}
                      </span>
                    </div>
                    <div className="pt-2 border-t border-slate-100 flex justify-between text-sm font-bold text-slate-900">
                      <span>Omset Bersih Diterima</span>
                      <span className="font-mono text-emerald-600">
                        {formatIDR(selectedOrderDetail.net_amount)}
                      </span>
                    </div>
                  </div>

                  {/* Line Items Table */}
                  <div className="space-y-3">
                    <h3 className="text-sm font-bold text-slate-900 flex items-center gap-2">
                      <Boxes className="w-4 h-4 text-blue-600" />
                      <span>Barang Dipesan & Resolusi SKU</span>
                    </h3>

                    <div className="border border-slate-200 rounded-xl overflow-hidden">
                      <table className="w-full text-left text-xs">
                        <thead className="bg-slate-50 text-slate-500 font-semibold border-b border-slate-200">
                          <tr>
                            <th className="px-3.5 py-2.5">External SKU & Barang</th>
                            <th className="px-3.5 py-2.5">Qty Pesanan</th>
                            <th className="px-3.5 py-2.5">Produk Master & Multiplier</th>
                            <th className="px-3.5 py-2.5 text-right">Subtotal</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                          {selectedOrderDetail.items?.map((item) => (
                            <tr key={item.id} className="hover:bg-slate-50/50">
                              <td className="px-3.5 py-3">
                                <div className="font-mono font-bold text-slate-900">
                                  {item.external_sku}
                                </div>
                                <div className="text-slate-600 mt-0.5">{item.item_name}</div>
                              </td>

                              <td className="px-3.5 py-3 font-semibold text-slate-800">
                                {item.quantity}
                              </td>

                              <td className="px-3.5 py-3">
                                {item.is_mapped ? (
                                  <div>
                                    <div className="font-medium text-emerald-700 flex items-center gap-1">
                                      <Check className="w-3.5 h-3.5" />
                                      <span>{item.product_name || "Produk Terpetakan"}</span>
                                    </div>
                                    <div className="text-[11px] font-mono text-slate-500 mt-0.5">
                                      SKU: {item.product_sku || item.product_id} &times;{" "}
                                      {item.multiplier || 1} Pcs
                                    </div>
                                  </div>
                                ) : (
                                  <div className="flex items-center gap-2">
                                    <span className="inline-flex items-center gap-1 text-amber-700 bg-amber-50 px-2 py-0.5 rounded font-semibold border border-amber-200">
                                      <AlertTriangle className="w-3 h-3 text-amber-600" />
                                      Belum Terpetakan
                                    </span>
                                    <button
                                      onClick={() =>
                                        openCreateMapping(
                                          item.external_sku,
                                          selectedOrderDetail.channel
                                        )
                                      }
                                      className="min-h-[36px] px-2 py-1 rounded bg-amber-600 hover:bg-amber-700 text-white font-bold text-[11px]"
                                    >
                                      Petakan
                                    </button>
                                  </div>
                                )}
                              </td>

                              <td className="px-3.5 py-3 text-right font-mono font-bold text-slate-900">
                                {formatIDR(item.subtotal)}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                </>
              ) : null}
            </div>

            {/* Drawer Footer */}
            <div className="p-4 border-t border-slate-200 bg-slate-50 flex justify-end gap-3">
              <button
                onClick={() => setSelectedOrderId(null)}
                className="min-h-[48px] px-5 py-2.5 rounded-lg text-sm font-semibold text-slate-700 bg-white border border-slate-300 hover:bg-slate-100 transition-colors"
              >
                Tutup
              </button>
            </div>
          </div>
        </div>
      )}

      {/* =================================================================== */}
      {/* MODAL: TAMBAH PEMETAAN SKU (1-CLICK IN-PLACE RESOLVER)              */}
      {/* =================================================================== */}
      {showMappingModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
          {/* Backdrop */}
          <div
            className="fixed inset-0 bg-black/40 backdrop-blur-sm"
            onClick={() => setShowMappingModal(false)}
            aria-hidden="true"
          />

          {/* Modal Card */}
          <div className="relative z-50 w-full max-w-lg bg-white rounded-2xl shadow-2xl border border-slate-200 overflow-hidden">
            <div className="p-6 border-b border-slate-100 flex items-center justify-between bg-slate-50">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-xl bg-blue-100 text-blue-600">
                  <Boxes className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-base font-bold text-slate-900">
                    Tambah Pemetaan SKU Marketplace
                  </h3>
                  <p className="text-xs text-slate-500">
                    Hubungkan SKU Marketplace ke Produk Master Internal Gudang
                  </p>
                </div>
              </div>
              <button
                onClick={() => setShowMappingModal(false)}
                className="min-h-[48px] min-w-[48px] p-2 rounded-lg text-slate-400 hover:text-slate-600"
                aria-label="Tutup modal pemetaan SKU"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <form onSubmit={handleSaveMapping} className="p-6 space-y-4">
              {mapModalError && (
                <div className="p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-700 text-xs flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{mapModalError}</span>
                </div>
              )}

              {/* Channel Selector */}
              <div>
                <label
                  htmlFor="map-channel-select"
                  className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5"
                >
                  Kanal Marketplace <span className="text-rose-500">*</span>
                </label>
                <select
                  id="map-channel-select"
                  value={mapChannel}
                  onChange={(e) => setMapChannel(e.target.value as MarketplaceChannel)}
                  className="w-full min-h-[48px] px-3.5 py-2.5 rounded-lg border border-slate-300 text-slate-900 text-sm font-medium focus:ring-2 focus:ring-blue-500 outline-none"
                >
                  <option value="SHOPEE">Shopee</option>
                  <option value="TOKOPEDIA">Tokopedia</option>
                  <option value="TIKTOK">TikTok Shop</option>
                  <option value="LAZADA">Lazada</option>
                  <option value="BLIBLI">Blibli</option>
                  <option value="OTHER">Lainnya / Generic</option>
                </select>
              </div>

              {/* External SKU */}
              <div>
                <label
                  htmlFor="map-external-sku"
                  className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5"
                >
                  Kode SKU di Marketplace (External SKU){" "}
                  <span className="text-rose-500">*</span>
                </label>
                <input
                  id="map-external-sku"
                  type="text"
                  placeholder="Contoh: KOPISUSU-DUS-24"
                  value={mapExternalSku}
                  onChange={(e) => setMapExternalSku(e.target.value)}
                  className="w-full min-h-[48px] px-3.5 py-2.5 rounded-lg border border-slate-300 text-slate-900 text-sm font-mono focus:ring-2 focus:ring-blue-500 outline-none"
                />
              </div>

              {/* External Name (Optional) */}
              <div>
                <label
                  htmlFor="map-external-name"
                  className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5"
                >
                  Nama Produk di Marketplace (Opsional)
                </label>
                <input
                  id="map-external-name"
                  type="text"
                  placeholder="Contoh: Kopi Susu Dus Karton 24 pcs"
                  value={mapExternalName}
                  onChange={(e) => setMapExternalName(e.target.value)}
                  className="w-full min-h-[48px] px-3.5 py-2.5 rounded-lg border border-slate-300 text-slate-900 text-sm focus:ring-2 focus:ring-blue-500 outline-none"
                />
              </div>

              {/* Internal Product Selector */}
              <div>
                <label
                  htmlFor="map-product-select"
                  className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5"
                >
                  Pilih Produk Master Internal Gudang{" "}
                  <span className="text-rose-500">*</span>
                </label>
                <select
                  id="map-product-select"
                  value={mapProductId}
                  onChange={(e) => setMapProductId(e.target.value)}
                  disabled={loadingProducts}
                  className="w-full min-h-[48px] px-3.5 py-2.5 rounded-lg border border-slate-300 text-slate-900 text-sm font-medium focus:ring-2 focus:ring-blue-500 outline-none"
                >
                  {products.length === 0 ? (
                    <option value="">Memuat produk master...</option>
                  ) : (
                    products.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name} ({p.sku}) &mdash; {formatIDR(p.price)}
                      </option>
                    ))
                  )}
                </select>
              </div>

              {/* Multiplier */}
              <div>
                <label
                  htmlFor="map-multiplier"
                  className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1.5"
                >
                  Faktor Pengali Kuantitas (Packaging Multiplier)
                </label>
                <div className="flex items-center gap-3">
                  <input
                    id="map-multiplier"
                    type="number"
                    min="1"
                    step="1"
                    placeholder="1"
                    value={mapMultiplier}
                    onChange={(e) => setMapMultiplier(e.target.value)}
                    className="w-32 min-h-[48px] px-3.5 py-2.5 rounded-lg border border-slate-300 text-slate-900 text-sm font-bold text-center focus:ring-2 focus:ring-blue-500 outline-none"
                  />
                  <div className="text-xs text-slate-500 leading-tight">
                    1 unit SKU Marketplace ={" "}
                    <strong className="text-slate-900 font-bold">
                      {mapMultiplier || 1} Pcs
                    </strong>{" "}
                    di gudang. (Misal: 1 Dus = 24 Pcs).
                  </div>
                </div>
              </div>

              {/* Explanatory callout */}
              <div className="p-3.5 rounded-xl bg-blue-50/80 border border-blue-200 text-xs text-blue-900 flex items-start gap-2.5">
                <Info className="w-4 h-4 text-blue-600 shrink-0 mt-0.5" />
                <p className="leading-relaxed">
                  <strong>Otomatisasi Reproses:</strong> Saat disimpan, sistem akan langsung mencari semua pesanan berstatus <span className="font-mono">UNMAPPED_SKU</span>, menautkan produk internal, dan memotong stok untuk pesanan pelanggan secara otomatis.
                </p>
              </div>

              {/* Modal Actions */}
              <div className="pt-4 border-t border-slate-100 flex items-center justify-end gap-3">
                <button
                  type="button"
                  onClick={() => setShowMappingModal(false)}
                  className="min-h-[48px] px-4 py-2.5 rounded-lg text-sm font-semibold text-slate-700 hover:bg-slate-100 transition-colors"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createMappingMutation.isPending}
                  className="min-h-[48px] px-6 py-2.5 rounded-lg text-sm font-bold text-white bg-blue-600 hover:bg-blue-700 active:bg-blue-800 disabled:opacity-50 transition-colors shadow-sm"
                >
                  {createMappingMutation.isPending
                    ? "Menyimpan & Reproses..."
                    : "Simpan & Hubungkan"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
