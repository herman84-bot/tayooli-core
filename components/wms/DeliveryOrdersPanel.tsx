"use client"

import React, { useState, useMemo, useEffect } from "react"
import Link from "next/link"
import {
  FileCheck,
  Truck,
  Plus,
  Search,
  CheckCircle2,
  Clock,
  Send,
  Printer,
  Building,
  AlertTriangle,
  X,
  Trash2,
  Layers,
  RefreshCw,
  ArrowLeft,
  Barcode,
  ClipboardList,
  QrCode,
  Gift,
  UserCheck,
  UserPlus,
  History,
} from "lucide-react"
import {
  useDeliveryOrders,
  useWarehouses,
  useWarehouseLocations,
  useCreateDeliveryOrder,
  useDispatchDeliveryOrder,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import {
  api,
  DeliveryOrder,
  DeliveryOrderItem,
  DeliveryOrderStatus,
  Customer,
  PickingTaskDetail,
} from "@/lib/api"
import { PrintDeliveryOrder } from "@/components/wms/PrintDeliveryOrder"
import { PrintPickingList } from "@/components/wms/PrintPickingList"
import { PrintThermalAWB } from "@/components/wms/PrintThermalAWB"
import { PackStationModal } from "@/components/wms/PackStationModal"
import { WaveReleaseModal } from "@/components/wms/WaveReleaseModal"
import { ActivityTimelineDrawer } from "@/components/wms/ActivityTimelineDrawer"
import { useWMSStock } from "@/hooks/useWMSLedger"
import { availableFor, validateDOLineQty } from "@/lib/wms/validation"
import { ExportModal, ExportButton, type ExportFilter } from "@/components/ui/ExportModal"
import type { ExportColumn } from "@/lib/export"
import { ShippingManifestsPanel } from "@/components/wms/ShippingManifestsPanel"

// Backend stores sales_order_id as a UUID FK; anything else is rejected with 400.
const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i

interface LineItemDraft {
  productId: string
  productName: string
  sku: string
  quantity: number
  locationId: string
  locationCode?: string
  isFreeItem?: boolean
}

/**
 * Surat Jalan (DO) list + create/dispatch/print flow. Rendered standalone at
 * the legacy route and embedded (header hidden) inside /wms/arus-barang KELUAR.
 */
export default function DeliveryOrdersPanel({ embedded = false }: { embedded?: boolean } = {}) {
  const [mainTab, setMainTab] = useState<"DO" | "MANIFEST">("DO")
  const [selectedWarehouseFilter, setSelectedWarehouseFilter] = useState<string>("ALL")
  const activeWhId = selectedWarehouseFilter === "ALL" ? null : selectedWarehouseFilter

  const { data: warehouses = [] } = useWarehouses()
  const { data: deliveryOrders = [], isLoading, refetch, isFetching } = useDeliveryOrders(activeWhId)
  const { data: products = [] } = useProducts()

  const [statusFilter, setStatusFilter] = useState<string>("ALL")
  const [searchQuery, setSearchQuery] = useState("")

  // Print Modal State
  const [selectedDoForPrint, setSelectedDoForPrint] = useState<DeliveryOrder | null>(null)
  const [printItems, setPrintItems] = useState<DeliveryOrderItem[]>([])
  const [loadingPrintDetails, setLoadingPrintDetails] = useState(false)

  // Dispatch Confirmation Modal State (replaces native confirm/alert)
  const [orderToDispatch, setOrderToDispatch] = useState<DeliveryOrder | null>(null)
  const [toast, setToast] = useState<{ type: "success" | "error"; message: string } | null>(null)

  // Create DO Modal State
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [modalWarehouseId, setModalWarehouseId] = useState("")
  const [doNumber, setDoNumber] = useState("")
  const [salesOrderId, setSalesOrderId] = useState("")
  const [customerId, setCustomerId] = useState("")
  const [orderType, setOrderType] = useState("DIRECT_DO")
  const [expeditionName, setExpeditionName] = useState("")
  const [trackingNumber, setTrackingNumber] = useState("")
  const [driverName, setDriverName] = useState("")
  const [vehiclePlate, setVehiclePlate] = useState("")
  const [recipientName, setRecipientName] = useState("")
  const [lineItems, setLineItems] = useState<LineItemDraft[]>([])
  const [modalError, setModalError] = useState<string | null>(null)

  // Sprint 3 Modals State
  const [selectedDoForPicking, setSelectedDoForPicking] = useState<PickingTaskDetail | null>(null)
  const [loadingPicking, setLoadingPicking] = useState(false)

  const [selectedDoForPackStation, setSelectedDoForPackStation] = useState<DeliveryOrder | null>(null)
  const [packStationItems, setPackStationItems] = useState<DeliveryOrderItem[]>([])
  const [loadingPackStation, setLoadingPackStation] = useState(false)

  const [selectedDoForThermal, setSelectedDoForThermal] = useState<{
    order: DeliveryOrder
    items: DeliveryOrderItem[]
  } | null>(null)
  const [isWaveModalOpen, setIsWaveModalOpen] = useState(false)
  const [loadingThermal, setLoadingThermal] = useState(false)
  const [selectedDoForAudit, setSelectedDoForAudit] = useState<DeliveryOrder | null>(null)

  // Quick Add Customer Modal State (CR-02b)
  const [customers, setCustomers] = useState<Customer[]>([])
  const [showQuickCustomerModal, setShowQuickCustomerModal] = useState(false)
  const [newCustName, setNewCustName] = useState("")
  const [newCustPhone, setNewCustPhone] = useState("")
  const [newCustAddress, setNewCustAddress] = useState("")
  const [savingCustomer, setSavingCustomer] = useState(false)

  useEffect(() => {
    api.customers.list().then((res) => {
      if (res && res.data) setCustomers(res.data)
    }).catch(() => {})
  }, [])

  // Export Modal State
  const [exporting, setExporting] = useState(false)

  // Helpers must be declared before doExportColumns/doExportFilters reference them
  // (they are evaluated during render, before getStatusBadge section below).
  const getStatusLabel = (status: DeliveryOrderStatus): string => {
    const labels: Record<DeliveryOrderStatus, string> = {
      DRAFT: "Draf",
      CONFIRMED: "Dikonfirmasi",
      PICKED: "Diambil",
      PACKED: "Dikemas",
      SHIPPED: "Dikirim",
      DELIVERED: "Diterima",
      RETURNED: "Dikembalikan",
      CANCELLED: "Dibatalkan",
    }
    return labels[status] || status
  }

  const fmtDate = (iso?: string | null): string => {
    if (!iso) return ""
    const d = new Date(iso)
    return Number.isNaN(d.getTime())
      ? ""
      : d.toLocaleDateString("id-ID", { day: "2-digit", month: "short", year: "numeric" })
  }

  const doExportColumns: ExportColumn<DeliveryOrder>[] = [
    { header: "No. Surat Jalan", value: (d) => d.do_number, width: 22 },
    { header: "Tanggal", value: (d) => fmtDate(d.created_at), width: 20 },
    { header: "Gudang", value: (d) => getWarehouseName(d.warehouse_id), width: 22 },
    { header: "Status", value: (d) => getStatusLabel(d.status), width: 18 },
    { header: "Penerima", value: (d) => d.recipient_name || "", width: 22 },
    { header: "Driver", value: (d) => d.driver_name || "", width: 22 },
    { header: "Plat Kendaraan", value: (d) => d.vehicle_plate || "", width: 18 },
    { header: "Ekspedisi", value: (d) => d.expedition_name || "", width: 22 },
    { header: "No. Tracking", value: (d) => d.tracking_number || "", width: 22 },
    { header: "Tgl Diterima", value: (d) => fmtDate(d.received_date), width: 20 },
  ]
  const doExportFilters: ExportFilter<DeliveryOrder>[] = [
    { type: "dateRange", id: "created", label: "Tanggal terbit", getDate: (d) => d.created_at },
    { type: "dateRange", id: "received", label: "Tanggal diterima", getDate: (d) => d.received_date },
    {
      type: "select",
      id: "status",
      label: "Status",
      options: ["DRAFT", "CONFIRMED", "PICKED", "PACKED", "SHIPPED", "DELIVERED", "RETURNED", "CANCELLED"].map((s) => ({
        value: s,
        label: getStatusLabel(s as DeliveryOrderStatus),
      })),
      match: (d, v) => d.status === v,
    },
  ]

  // Locations for selected warehouse in create modal
  const { data: modalLocations = [] } = useWarehouseLocations(modalWarehouseId || null)
  // Live stock for availability pre-check (only fetched while the create modal has a warehouse).
  const {
    data: modalStock,
    isError: modalStockError,
    isLoading: modalStockLoading,
    isSuccess: modalStockSuccess,
  } = useWMSStock(
    showCreateModal && modalWarehouseId ? modalWarehouseId : undefined,
    showCreateModal && !!modalWarehouseId
  )

  // Remaining availability per line, accounting for earlier lines that draw from the same pool.
  const lineChecks = useMemo(() => {
    const used = new Map<string, number>()
    const stockReady = modalStockSuccess && !modalStockError && modalStock && !!modalWarehouseId
    return lineItems.map((item) => {
      const key = `${item.productId}|${item.locationId || "*"}`
      const avail = stockReady ? availableFor(modalStock, item.productId, item.locationId) : undefined
      const already = used.get(key) ?? 0
      const remaining = avail === undefined ? undefined : Math.max(0, avail - already)
      const error = validateDOLineQty(item.quantity, remaining)
      if (!error && Number.isFinite(item.quantity)) used.set(key, already + item.quantity)
      return { remaining, error }
    })
  }, [lineItems, modalStock, modalStockError, modalStockSuccess, modalWarehouseId])
  const hasLineErrors = lineChecks.some((c) => c.error)

  // Mutations
  const createDoMutation = useCreateDeliveryOrder()
  const dispatchDoMutation = useDispatchDeliveryOrder()

  // Filtered Delivery Orders
  const filteredOrders = useMemo(() => {
    return deliveryOrders.filter((order) => {
      const matchesStatus = statusFilter === "ALL" || order.status === statusFilter
      const wh = warehouses.find((w) => w.id === order.warehouse_id)

      const matchesSearch =
        searchQuery === "" ||
        order.do_number.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (order.recipient_name && order.recipient_name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (order.driver_name && order.driver_name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (order.vehicle_plate && order.vehicle_plate.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (order.expedition_name && order.expedition_name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (wh && wh.name.toLowerCase().includes(searchQuery.toLowerCase()))

      return matchesStatus && matchesSearch
    })
  }, [deliveryOrders, statusFilter, searchQuery, warehouses])

  // Summary Metrics
  const metrics = useMemo(() => {
    const total = deliveryOrders.length
    const draft = deliveryOrders.filter(
      (d) => d.status === "DRAFT" || d.status === "CONFIRMED" || d.status === "PICKED" || d.status === "PACKED"
    ).length
    const shipped = deliveryOrders.filter((d) => d.status === "SHIPPED").length
    const delivered = deliveryOrders.filter((d) => d.status === "DELIVERED").length
    return { total, draft, shipped, delivered }
  }, [deliveryOrders])

  // Open Print Modal with Item Details
  const handleOpenPrint = async (order: DeliveryOrder) => {
    setSelectedDoForPrint(order)
    setLoadingPrintDetails(true)
    try {
      const res = await api.wms.deliveryOrders.get(order.id)
      setPrintItems(res.items || [])
    } catch {
      setPrintItems([])
    } finally {
      setLoadingPrintDetails(false)
    }
  }

  // Trigger Dispatch Confirmation Modal
  const requestDispatch = (order: DeliveryOrder, e: React.MouseEvent) => {
    e.stopPropagation()
    setOrderToDispatch(order)
  }

  // Open Picking List (FE-06)
  const handleOpenPicking = async (order: DeliveryOrder) => {
    setLoadingPicking(true)
    try {
      const res = await api.wms.deliveryOrders.getPickingTask(order.id)
      setSelectedDoForPicking(res.data)
    } catch (err: any) {
      setToast({
        type: "error",
        message: err.message || "Gagal memuat Picking Task",
      })
    } finally {
      setLoadingPicking(false)
    }
  }

  // Open Pack Station (FE-07)
  const handleOpenPackStation = async (order: DeliveryOrder) => {
    setLoadingPackStation(true)
    try {
      const res = await api.wms.deliveryOrders.get(order.id)
      setSelectedDoForPackStation(res.delivery_order)
      setPackStationItems(res.items)
    } catch (err: any) {
      setToast({
        type: "error",
        message: err.message || "Gagal memuat item stasiun kemas",
      })
    } finally {
      setLoadingPackStation(false)
    }
  }

  // Open Thermal AWB Label (FE-08)
  const handleOpenThermal = async (order: DeliveryOrder) => {
    setLoadingThermal(true)
    try {
      const res = await api.wms.deliveryOrders.get(order.id)
      setSelectedDoForThermal({ order: res.delivery_order, items: res.items })
    } catch (err: any) {
      setToast({
        type: "error",
        message: err.message || "Gagal memuat data label thermal",
      })
    } finally {
      setLoadingThermal(false)
    }
  }

  // Quick Customer Create (CR-02b)
  const handleCreateQuickCustomer = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newCustName.trim()) return
    setSavingCustomer(true)
    try {
      const created = await api.customers.create({
        name: newCustName.trim(),
        phone: newCustPhone.trim() || undefined,
        address: newCustAddress.trim() || undefined,
      })
      setCustomers((prev) => [created, ...prev])
      setCustomerId(created.id)
      setRecipientName(created.name)
      setShowQuickCustomerModal(false)
      setNewCustName("")
      setNewCustPhone("")
      setNewCustAddress("")
      setToast({
        type: "success",
        message: `Pelanggan "${created.name}" berhasil ditambahkan.`,
      })
    } catch (err: any) {
      setToast({
        type: "error",
        message: err.message || "Gagal menambahkan pelanggan",
      })
    } finally {
      setSavingCustomer(false)
    }
  }

  // Execute Dispatch
  const confirmExecuteDispatch = async () => {
    if (!orderToDispatch) return
    const id = orderToDispatch.id
    try {
      await dispatchDoMutation.mutateAsync(id)
      setOrderToDispatch(null)
      setToast({
        type: "success",
        message: `Surat Jalan ${orderToDispatch.do_number} berhasil dikirim (stok dipotong).`,
      })
      refetch()
    } catch (err: unknown) {
      setToast({
        type: "error",
        message: err instanceof Error ? err.message : "Gagal melakukan dispatch Surat Jalan.",
      })
    }
  }

  // Open Create Modal
  const handleOpenCreateModal = () => {
    const randomSuffix = Math.floor(1000 + Math.random() * 9000)
    const todayStr = new Date().toISOString().slice(0, 10).replace(/-/g, "")
    setDoNumber(`DO/${todayStr}/${randomSuffix}`)
    setSalesOrderId("")
    setCustomerId("")
    setOrderType("DIRECT_DO")
    setModalWarehouseId(warehouses[0]?.id || "")
    setExpeditionName("JNE Trucking (JTR)")
    setTrackingNumber(`JTR${Math.floor(1000000000 + Math.random() * 9000000000)}`)
    setDriverName("")
    setVehiclePlate("")
    setRecipientName("")
    setLineItems([])
    setModalError(null)
    setShowCreateModal(true)
  }

  // Add Item to Draft
  const handleAddItem = () => {
    if (products.length === 0) {
      setModalError("Tidak ada data produk yang tersedia.")
      return
    }
    const defaultProduct = products[0]
    const defaultLocation = modalLocations[0]
    setLineItems((prev) => [
      ...prev,
      {
        productId: defaultProduct.id,
        productName: defaultProduct.name,
        sku: defaultProduct.sku || "",
        quantity: 1,
        locationId: defaultLocation?.id || "",
        locationCode: defaultLocation?.code || "DEFAULT",
      },
    ])
  }

  // Submit Create Delivery Order
  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setModalError(null)

    if (!modalWarehouseId) {
      setModalError("Gudang pemenuhan harus dipilih.")
      return
    }
    if (!doNumber.trim()) {
      setModalError("Nomor Surat Jalan wajib diisi.")
      return
    }
    if (lineItems.length === 0) {
      setModalError("Surat Jalan harus memiliki minimal 1 item barang.")
      return
    }
    const soRef = salesOrderId.trim()
    if (soRef && !UUID_RE.test(soRef)) {
      setModalError("Ref. Sales Order harus berupa ID Sales Order yang valid, atau kosongkan untuk Surat Jalan langsung.")
      return
    }

    for (let i = 0; i < lineItems.length; i++) {
      const it = lineItems[i]
      const qtyErr = lineChecks[i]?.error ?? validateDOLineQty(it.quantity)
      if (qtyErr) {
        setModalError(`Baris ${i + 1} (${it.productName || it.sku}): ${qtyErr}`)
        return
      }
      if (!it.locationId) {
        setModalError(`Baris ${i + 1}: Lokasi rak pengambilan harus ditentukan.`)
        return
      }
    }

    try {
      await createDoMutation.mutateAsync({
        warehouse_id: modalWarehouseId,
        sales_order_id: soRef || undefined,
        customer_id: customerId || undefined,
        order_type: orderType,
        do_number: doNumber.trim(),
        expedition_name: expeditionName.trim() || undefined,
        tracking_number: trackingNumber.trim() || undefined,
        driver_name: driverName.trim() || undefined,
        vehicle_plate: vehiclePlate.trim() || undefined,
        recipient_name: recipientName.trim() || undefined,
        items: lineItems.map((item) => ({
          product_id: item.productId,
          quantity: item.quantity,
          location_id: item.locationId || undefined,
          is_free_item: item.isFreeItem || false,
        })),
      })
      setShowCreateModal(false)
      setToast({
        type: "success",
        message: `Surat Jalan ${doNumber.trim()} berhasil diterbitkan.`,
      })
      refetch()
    } catch (err: unknown) {
      setModalError(err instanceof Error ? err.message : "Gagal membuat Surat Jalan.")
    }
  }

  const getWarehouseName = (id: string) => {
    const wh = warehouses.find((w) => w.id === id)
    return wh ? wh.name : id.slice(0, 8)
  }

  const getStatusBadge = (status: DeliveryOrderStatus) => {
    switch (status) {
      case "DRAFT":
        return "bg-slate-100 text-slate-700 border-slate-200"
      case "CONFIRMED":
      case "PICKED":
      case "PACKED":
        return "bg-amber-50 text-amber-700 border-amber-200"
      case "SHIPPED":
        return "bg-blue-50 text-[#2563EB] border-blue-200"
      case "DELIVERED":
        return "bg-emerald-50 text-emerald-700 border-emerald-200"
      case "CANCELLED":
      case "RETURNED":
        return "bg-rose-50 text-rose-700 border-rose-200"
      default:
        return "bg-slate-100 text-slate-600 border-slate-200"
    }
  }


  return (
    <div className="min-h-screen bg-[#F8FAFC] pb-16 text-slate-900">
      {/* ── Toast Notification ── */}
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

      {/* ── Top Header ── */}
      <div className="bg-white border-b border-[#E2E8F0] shadow-xs">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 py-6 flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className={embedded ? "sr-only" : "space-y-1"}>
            <div className="flex items-center gap-2 text-xs text-slate-500">
              <Link href="/wms" className="hover:text-slate-800 transition flex items-center gap-1">
                <ArrowLeft className="h-3 w-3" /> WMS Hub
              </Link>
              <span>/</span>
              <span className="text-slate-800 font-medium">Surat Jalan</span>
            </div>
            <h1 className="text-xl sm:text-2xl font-bold text-slate-900 flex items-center gap-2.5">
              <div className="p-2 bg-emerald-50 text-emerald-600 rounded-lg border border-emerald-100">
                <FileCheck className="h-6 w-6" />
              </div>
              Surat Jalan & Delivery Orders
            </h1>
            <p className="text-xs sm:text-sm text-slate-500">
              Penerbitan dokumen legal serah terima pengiriman, pelacakan armada ekspedisi, dan pemotongan stok otomatis.
            </p>
          </div>

          <div className="flex items-center gap-2">
            {mainTab === "DO" && (
              <>
                <button
                  onClick={() => refetch()}
                  disabled={isFetching}
                  className="p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[44px] min-w-[44px] flex items-center justify-center transition-colors disabled:opacity-50"
                  title="Segarkan data"
                >
                  <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
                </button>
                <ExportButton onClick={() => setExporting(true)} disabled={deliveryOrders.length === 0} />

                <button
                  onClick={() => setIsWaveModalOpen(true)}
                  className="inline-flex items-center gap-1.5 px-3.5 py-2.5 min-h-[44px] rounded-lg font-semibold text-xs border border-indigo-200 bg-indigo-50 text-indigo-700 hover:bg-indigo-100 active:scale-95 transition-all shadow-xs"
                  title="Kelompokkan DO menjadi batch picking (Wave Release - PDF-05)"
                >
                  <Layers className="h-4 w-4 text-indigo-600" />
                  <span>Wave Release (PDF-05)</span>
                </button>

                <button
                  onClick={handleOpenCreateModal}
                  className="inline-flex items-center gap-2 px-4 py-2.5 min-h-[44px] rounded-lg font-semibold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] active:scale-95 transition-all shadow-sm"
                >
                  <Plus className="h-4 w-4" />
                  <span>Buat Surat Jalan Baru</span>
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 pt-6 space-y-6">
        {/* ── Submodule Tab Toggle ── */}
        <div className="flex items-center gap-2 border-b border-slate-200 pb-3">
          <button
            type="button"
            onClick={() => setMainTab("DO")}
            className={`inline-flex items-center gap-2 px-4 py-2 rounded-xl text-xs sm:text-sm font-semibold transition ${
              mainTab === "DO"
                ? "bg-slate-900 text-white shadow-xs"
                : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
            }`}
          >
            <FileCheck className="w-4 h-4" />
            <span>Surat Jalan (DO)</span>
          </button>
          <button
            type="button"
            onClick={() => setMainTab("MANIFEST")}
            className={`inline-flex items-center gap-2 px-4 py-2 rounded-xl text-xs sm:text-sm font-semibold transition ${
              mainTab === "MANIFEST"
                ? "bg-slate-900 text-white shadow-xs"
                : "text-slate-600 hover:text-slate-900 hover:bg-slate-100"
            }`}
          >
            <Truck className="w-4 h-4" />
            <span>Manifest Ekspedisi</span>
          </button>
        </div>

        {mainTab === "MANIFEST" ? (
          <ShippingManifestsPanel
            warehouseId={selectedWarehouseFilter !== "ALL" ? selectedWarehouseFilter : undefined}
          />
        ) : (
          <>
            {/* ── Metrics Summary ── */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 shadow-xs">
            <div className="flex items-center justify-between text-slate-500 text-xs mb-1">
              <span>Total Surat Jalan</span>
              <Layers className="h-4 w-4 text-slate-400" />
            </div>
            <div className="text-2xl font-bold text-slate-900">{metrics.total}</div>
            <div className="text-[11px] text-slate-400 mt-1">Seluruh arsip pengiriman</div>
          </div>

          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 shadow-xs">
            <div className="flex items-center justify-between text-amber-600 text-xs mb-1">
              <span>Siap Dikirim (Draft)</span>
              <Clock className="h-4 w-4 text-amber-500" />
            </div>
            <div className="text-2xl font-bold text-amber-600">{metrics.draft}</div>
            <div className="text-[11px] text-slate-400 mt-1">Menunggu armada & dispatch</div>
          </div>

          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 shadow-xs">
            <div className="flex items-center justify-between text-[#2563EB] text-xs mb-1">
              <span>Dalam Perjalanan</span>
              <Truck className="h-4 w-4 text-[#2563EB]" />
            </div>
            <div className="text-2xl font-bold text-[#2563EB]">{metrics.shipped}</div>
            <div className="text-[11px] text-slate-400 mt-1">Armada sedang di jalan</div>
          </div>

          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 shadow-xs">
            <div className="flex items-center justify-between text-emerald-600 text-xs mb-1">
              <span>Diterima Pelanggan</span>
              <CheckCircle2 className="h-4 w-4 text-emerald-500" />
            </div>
            <div className="text-2xl font-bold text-emerald-600">{metrics.delivered}</div>
            <div className="text-[11px] text-slate-400 mt-1">Serah terima selesai</div>
          </div>
        </div>

        {/* ── Controls & Filter Bar ── */}
        <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3 shadow-xs">
          <div className="flex flex-wrap items-center gap-2.5">
            {/* Filter Warehouse */}
            <div className="flex items-center gap-1.5 text-xs text-slate-500">
              <Building className="h-3.5 w-3.5 text-slate-400" />
              <select
                value={selectedWarehouseFilter}
                onChange={(e) => setSelectedWarehouseFilter(e.target.value)}
                className="bg-slate-50 border border-slate-200 rounded-lg px-3 py-2 text-xs text-slate-800 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
              >
                <option value="ALL">Semua Gudang Asal</option>
                {warehouses.map((w) => (
                  <option key={w.id} value={w.id}>
                    {w.name} ({w.code})
                  </option>
                ))}
              </select>
            </div>

            {/* Filter Status */}
            <div className="flex items-center gap-1 bg-slate-100 p-0.5 rounded-lg border border-slate-200 text-xs">
              {[
                { label: "Semua", value: "ALL" },
                { label: "Draft", value: "DRAFT" },
                { label: "Shipped", value: "SHIPPED" },
                { label: "Delivered", value: "DELIVERED" },
              ].map((tab) => (
                <button
                  key={tab.value}
                  onClick={() => setStatusFilter(tab.value)}
                  className={`px-3 py-1.5 rounded-md font-medium transition ${
                    statusFilter === tab.value
                      ? "bg-white text-slate-900 shadow-xs"
                      : "text-slate-600 hover:text-slate-900"
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>

          {/* Search Box */}
          <div className="relative flex-1 max-w-sm">
            <Search className="h-4 w-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              placeholder="Cari No. DO, penerima, supir..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-2 bg-slate-50 border border-slate-200 rounded-lg text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-[#2563EB] focus:bg-white transition"
            />
          </div>
        </div>

        {/* ── Delivery Orders Table ── */}
        <div className="bg-white rounded-xl border border-[#E2E8F0] shadow-xs overflow-hidden">
          {isLoading ? (
            <div className="p-12 text-center text-slate-500 space-y-2">
              <RefreshCw className="h-6 w-6 animate-spin mx-auto text-[#2563EB]" />
              <p className="text-xs">Memuat data Surat Jalan...</p>
            </div>
          ) : filteredOrders.length === 0 ? (
            <div className="p-12 text-center text-slate-500 space-y-3">
              <Truck className="h-10 w-10 mx-auto text-slate-300 stroke-[1.5]" />
              <p className="text-sm font-medium text-slate-800">Tidak ada Surat Jalan yang ditemukan</p>
              <p className="text-xs text-slate-500 max-w-sm mx-auto">
                Silakan ubah filter atau terbitkan Surat Jalan pengiriman baru untuk memulai proses dispatch.
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs border-collapse">
                <thead>
                  <tr className="border-b border-slate-200 bg-slate-50/70 text-slate-500 font-semibold uppercase tracking-wider">
                    <th className="py-3.5 px-4">No. Surat Jalan</th>
                    <th className="py-3.5 px-4">Gudang Asal</th>
                    <th className="py-3.5 px-4">Pelanggan / Penerima</th>
                    <th className="py-3.5 px-4">Tipe & Ref</th>
                    <th className="py-3.5 px-4">Ekspedisi & Driver</th>
                    <th className="py-3.5 px-4">Pelaku (Dibuat / Disetujui / Dikemas / Dikirim)</th>
                    <th className="py-3.5 px-4">Status</th>
                    <th className="py-3.5 px-4 text-right">Aksi Outbound</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 bg-white">
                  {filteredOrders.map((order) => {
                    const canDispatch =
                      order.status === "DRAFT" ||
                      order.status === "CONFIRMED" ||
                      order.status === "PICKED" ||
                      order.status === "PACKED"
                    return (
                      <tr key={order.id} className="hover:bg-slate-50/80 transition-colors group">
                        <td className="py-3.5 px-4 font-mono font-semibold text-slate-900">
                          <div className="flex items-center gap-1.5">
                            <FileCheck className="h-4 w-4 text-[#2563EB]" />
                            <span>{order.do_number}</span>
                          </div>
                          <div className="text-[10px] text-slate-400 font-sans mt-0.5">
                            {new Date(order.created_at).toLocaleDateString("id-ID", {
                              day: "2-digit",
                              month: "short",
                              year: "numeric",
                            })}
                          </div>
                        </td>
                        <td className="py-3.5 px-4 text-slate-700">
                          {getWarehouseName(order.warehouse_id)}
                        </td>
                        <td className="py-3.5 px-4">
                          <div className="font-bold text-slate-900">{order.customer_name || order.recipient_name || "Pelanggan Retail"}</div>
                          {order.customer_name && order.recipient_name && (
                            <div className="text-[10px] text-slate-500">U.P: {order.recipient_name}</div>
                          )}
                        </td>
                        <td className="py-3.5 px-4">
                          <span className="inline-block px-2 py-0.5 rounded bg-slate-100 text-slate-700 font-mono text-[10px] font-bold">
                            {order.order_type || "DIRECT_DO"}
                          </span>
                          {order.sales_order_id && (
                            <div className="text-[10px] font-mono text-slate-400 mt-0.5">SO: {order.sales_order_id.slice(0, 8)}</div>
                          )}
                        </td>
                        <td className="py-3.5 px-4">
                          <div className="text-slate-800 font-medium">{order.expedition_name || "Internal"}</div>
                          <div className="text-[10px] text-slate-500">{order.driver_name ? `Supir: ${order.driver_name}` : (order.vehicle_plate || "—")}</div>
                        </td>
                        <td className="py-3.5 px-4 text-[11px] text-slate-600">
                          <div>Buat: <span className="font-medium text-slate-800">{order.created_by_name || "Admin"}</span></div>
                          {order.confirmed_by_name && (
                            <div className="text-indigo-700">Setuju: <span className="font-medium">{order.confirmed_by_name}</span></div>
                          )}
                          {order.packed_by_name && (
                            <div className="text-emerald-700">Kemas: <span className="font-medium">{order.packed_by_name}</span></div>
                          )}
                          {order.dispatched_by_name && (
                            <div className="text-blue-700">Kirim: <span className="font-medium">{order.dispatched_by_name}</span></div>
                          )}
                        </td>
                        <td className="py-3.5 px-4">
                          <span
                            className={`inline-block px-2.5 py-0.5 rounded-full text-[10px] font-bold border tracking-wider uppercase ${getStatusBadge(
                              order.status
                            )}`}
                          >
                            {order.status}
                          </span>
                        </td>
                        <td className="py-3.5 px-4 text-right">
                          <div className="flex items-center justify-end gap-1 flex-wrap">
                            <button
                              onClick={() => setSelectedDoForAudit(order)}
                              title="Lihat Riwayat Aktivitas & Jejak Audit (CR-05b)"
                              className="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-slate-200 bg-white text-slate-700 hover:bg-slate-50 text-[11px] font-medium shadow-2xs"
                            >
                              <History className="h-3 w-3 text-indigo-600" />
                              Riwayat
                            </button>
                            <button
                              onClick={() => handleOpenPrint(order)}
                              title="Cetak Surat Jalan A4"
                              className="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-slate-200 bg-white text-slate-700 hover:bg-slate-50 text-[11px] font-medium shadow-2xs"
                            >
                              <Printer className="h-3 w-3 text-[#2563EB]" />
                              A4
                            </button>
                            <button
                              onClick={() => handleOpenPicking(order)}
                              title="Cetak Picking List (Terurut Rak)"
                              className="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-indigo-200 bg-indigo-50/50 text-indigo-700 hover:bg-indigo-100 text-[11px] font-medium shadow-2xs"
                            >
                              <ClipboardList className="h-3 w-3 text-indigo-600" />
                              Picking
                            </button>
                            {(order.status === "DRAFT" || order.status === "CONFIRMED" || order.status === "PICKED" || order.status === "PACKED") && (
                              <button
                                onClick={() => handleOpenPackStation(order)}
                                title="Buka Meja Kemas Barcode Scanner"
                                className="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-amber-200 bg-amber-50 text-amber-800 hover:bg-amber-100 text-[11px] font-semibold shadow-2xs"
                              >
                                <Barcode className="h-3 w-3 text-amber-700" />
                                Kemas
                              </button>
                            )}
                            {(order.status === "PACKED" || order.status === "SHIPPED") && (
                              <button
                                onClick={() => handleOpenThermal(order)}
                                title="Cetak Label Resi Thermal 100x150 mm"
                                className="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-purple-200 bg-purple-50 text-purple-700 hover:bg-purple-100 text-[11px] font-medium shadow-2xs"
                              >
                                <QrCode className="h-3 w-3 text-purple-600" />
                                AWB
                              </button>
                            )}
                            {canDispatch && (
                              <button
                                onClick={(e) => requestDispatch(order, e)}
                                disabled={dispatchDoMutation.isPending}
                                title="Kirim Surat Jalan & Potong Stok"
                                className="inline-flex items-center gap-1 px-2.5 py-1 rounded-md bg-emerald-600 text-white hover:bg-emerald-700 text-[11px] font-semibold shadow-2xs disabled:opacity-50"
                              >
                                <Send className="h-3 w-3" />
                                Kirim
                              </button>
                            )}
                          </div>
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
        </>
      )}
      </div>

      {/* ── Activity Timeline & Audit Drawer (CR-05b) ── */}
      {selectedDoForAudit && (
        <ActivityTimelineDrawer
          isOpen={!!selectedDoForAudit}
          onClose={() => setSelectedDoForAudit(null)}
          title={selectedDoForAudit.do_number}
          subtitle={`Surat Jalan Keluar (${selectedDoForAudit.order_type || "DIRECT_DO"})`}
          entityType="delivery_order"
          entityId={selectedDoForAudit.id}
          actors={{
            created_by_name: selectedDoForAudit.created_by_name || "Admin Pembuat",
            created_at: selectedDoForAudit.created_at,
            confirmed_by_name: selectedDoForAudit.confirmed_by_name,
            packed_by_name: selectedDoForAudit.packed_by_name,
            dispatched_by_name: selectedDoForAudit.dispatched_by_name,
          }}
          metadata={{
            status: selectedDoForAudit.status,
            reference_type: selectedDoForAudit.order_type,
            reference_number: selectedDoForAudit.sales_order_id || selectedDoForAudit.tracking_number,
            product_name: selectedDoForAudit.customer_name || selectedDoForAudit.recipient_name,
          }}
        />
      )}

      {/* ── In-App Confirmation Modal for Dispatch (replaces confirm/alert) ── */}
      {orderToDispatch && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div
            role="alertdialog"
            aria-modal="true"
            aria-label="Konfirmasi Pengiriman"
            className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4"
          >
            <div className="flex items-center gap-3">
              <div className="p-3 bg-emerald-50 text-emerald-600 rounded-xl border border-emerald-100">
                <Send className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">Konfirmasi Pengiriman</h3>
                <p className="text-xs text-slate-500 mt-0.5">Surat Jalan {orderToDispatch.do_number}</p>
              </div>
            </div>

            <p className="text-sm text-slate-600 leading-relaxed">
              Konfirmasi pengiriman Surat Jalan ini? Status akan diperbarui menjadi <strong>SHIPPED</strong> dan stok gudang otomatis dipotong untuk pelanggan tujuan.
            </p>
            <p className="rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs font-medium text-amber-800">
              Tindakan ini permanen: potongan stok di buku besar tidak dapat dibatalkan dari layar ini.
            </p>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                type="button"
                onClick={() => setOrderToDispatch(null)}
                className="px-4 py-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50 text-sm font-medium transition"
              >
                Batal
              </button>
              <button
                type="button"
                disabled={dispatchDoMutation.isPending}
                onClick={confirmExecuteDispatch}
                className="px-4 py-2 rounded-lg bg-emerald-600 text-white text-sm font-semibold hover:bg-emerald-700 shadow-sm transition disabled:opacity-50 flex items-center gap-1.5"
              >
                {dispatchDoMutation.isPending ? "Mengirim…" : "Kirim Surat Jalan"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Modal: Buat Surat Jalan Baru ── */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs overflow-y-auto">
          <div
            role="dialog"
            aria-modal="true"
            aria-label="Penerbitan Surat Jalan Baru"
            className="w-full max-w-2xl bg-white border border-slate-200 rounded-2xl shadow-2xl overflow-hidden my-8"
          >
            <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4 bg-slate-50/50">
              <div className="flex items-center gap-2.5">
                <div className="h-8 w-8 rounded-lg bg-emerald-50 border border-emerald-200 flex items-center justify-center text-emerald-600">
                  <FileCheck className="h-4 w-4" />
                </div>
                <div>
                  <h2 className="text-sm font-bold text-slate-900">Penerbitan Surat Jalan Baru</h2>
                  <p className="text-xs text-slate-500">Pengiriman barang dari gudang ke alamat pemesan</p>
                </div>
              </div>
              <button
                onClick={() => setShowCreateModal(false)}
                className="text-slate-400 hover:text-slate-700 p-1.5 rounded-lg hover:bg-slate-100 transition"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleCreateSubmit} className="p-6 space-y-4 text-xs">
              {modalError && (
                <div className="p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-700 flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4 shrink-0 text-rose-500" />
                  <span>{modalError}</span>
                </div>
              )}

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-slate-700 font-semibold">Nomor Surat Jalan (DO) *</label>
                  <input
                    type="text"
                    required
                    value={doNumber}
                    onChange={(e) => setDoNumber(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 font-mono focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                    placeholder="e.g. DO/2026/09/0001"
                  />
                </div>

                <div className="space-y-1">
                  <label htmlFor="do-sales-order-ref" className="text-slate-700 font-semibold">
                    Ref. Sales Order <span className="font-normal text-slate-400">(opsional)</span>
                  </label>
                  <input
                    id="do-sales-order-ref"
                    type="text"
                    value={salesOrderId}
                    onChange={(e) => setSalesOrderId(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 font-mono focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                    placeholder="Kosongkan untuk Surat Jalan langsung"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-slate-700 font-semibold">Gudang Pemenuhan (Asal) *</label>
                  <select
                    value={modalWarehouseId}
                    onChange={(e) => setModalWarehouseId(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                  >
                    {warehouses.map((w) => (
                      <option key={w.id} value={w.id}>
                        {w.name} ({w.code})
                      </option>
                    ))}
                  </select>
                </div>

                <div className="space-y-1">
                  <div className="flex items-center justify-between">
                    <label className="text-slate-700 font-semibold">Pilih Pelanggan (Customer)</label>
                    <button
                      type="button"
                      onClick={() => setShowQuickCustomerModal(true)}
                      className="text-[#2563EB] hover:text-[#1D4ED8] font-bold text-[11px] inline-flex items-center gap-1"
                    >
                      <UserPlus className="h-3 w-3" /> + Cepat
                    </button>
                  </div>
                  <select
                    value={customerId}
                    onChange={(e) => {
                      setCustomerId(e.target.value)
                      const c = customers.find((cust) => cust.id === e.target.value)
                      if (c) setRecipientName(c.name)
                    }}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                  >
                    <option value="">Pilih dari database pelanggan...</option>
                    {customers.map((c) => (
                      <option key={c.id} value={c.id}>
                        {c.name} {c.phone ? `(${c.phone})` : ""}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-1">
                  <label className="text-slate-700 font-semibold">Tipe Pesanan / Alur Keluar</label>
                  <select
                    value={orderType}
                    onChange={(e) => setOrderType(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB] font-medium"
                  >
                    <option value="DIRECT_DO">Surat Jalan Langsung (DIRECT_DO)</option>
                    <option value="SALES_ORDER">Pesanan Penjualan B2B (SALES_ORDER)</option>
                    <option value="MARKETPLACE">Pesanan Marketplace Omnichannel (MARKETPLACE)</option>
                    <option value="TRANSFER">Transfer Antar Gudang / Cabang (TRANSFER)</option>
                  </select>
                </div>

                <div className="space-y-1">
                  <label className="text-slate-700 font-semibold">Nama Penerima / U.P *</label>
                  <input
                    type="text"
                    required
                    value={recipientName}
                    onChange={(e) => setRecipientName(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                    placeholder="e.g. Toko Berkah Mandiri / Bpk. Hendra"
                  />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4 pt-1">
                <div className="space-y-1">
                  <label className="text-slate-700 font-semibold">Jasa Ekspedisi / Kurir</label>
                  <input
                    type="text"
                    value={expeditionName}
                    onChange={(e) => setExpeditionName(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                    placeholder="e.g. JNE Trucking"
                  />
                </div>

                <div className="space-y-1">
                  <label className="text-slate-700 font-semibold">Nama Sopir / Driver</label>
                  <input
                    type="text"
                    value={driverName}
                    onChange={(e) => setDriverName(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                    placeholder="e.g. Bambang Sudarsono"
                  />
                </div>

                <div className="space-y-1">
                  <label className="text-slate-700 font-semibold">Nomor Plat Polisi</label>
                  <input
                    type="text"
                    value={vehiclePlate}
                    onChange={(e) => setVehiclePlate(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 font-mono focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                    placeholder="e.g. B 9482 TKL"
                  />
                </div>
              </div>

              {/* ── Line Items ── */}
              <div className="border-t border-slate-200 pt-4 space-y-3">
                <div className="flex items-center justify-between">
                  <label className="text-slate-800 font-semibold flex items-center gap-1.5">
                    <Barcode className="h-4 w-4 text-[#2563EB]" />
                    Daftar Barang yang Diserahkan ({lineItems.length} item)
                  </label>
                  <button
                    type="button"
                    onClick={handleAddItem}
                    className="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-slate-100 text-slate-700 hover:bg-slate-200 border border-slate-200 transition font-medium text-xs"
                  >
                    <Plus className="h-3.5 w-3.5" /> Tambah Baris
                  </button>
                </div>

                {lineItems.length === 0 ? (
                  <div className="p-6 rounded-xl border border-dashed border-slate-200 text-center text-slate-500 bg-slate-50">
                    Belum ada item ditambahkan. Klik &quot;Tambah Baris&quot; untuk memasukkan barang.
                  </div>
                ) : (
                  <div className="space-y-2 max-h-48 overflow-y-auto pr-1">
                    {lineItems.map((item, idx) => (
                      <div
                        key={idx}
                        className="grid grid-cols-12 gap-2 items-center bg-slate-50 p-2.5 rounded-xl border border-slate-200"
                      >
                        {/* Pilih Produk */}
                        <div className="col-span-5">
                          <select
                            value={item.productId}
                            onChange={(e) => {
                              const found = products.find((p) => p.id === e.target.value)
                              const updated = [...lineItems]
                              updated[idx].productId = e.target.value
                              updated[idx].productName = found?.name || ""
                              updated[idx].sku = found?.sku || ""
                              setLineItems(updated)
                            }}
                            className="w-full bg-white border border-slate-200 rounded-lg px-2.5 py-1.5 text-slate-900"
                          >
                            {products.map((p) => (
                              <option key={p.id} value={p.id}>
                                {p.name} ({p.sku || "NO-SKU"})
                              </option>
                            ))}
                          </select>
                        </div>

                        {/* Pilih Rak Lokasi */}
                        <div className="col-span-3">
                          <select
                            value={item.locationId}
                            onChange={(e) => {
                              const foundLoc = modalLocations.find((l) => l.id === e.target.value)
                              const updated = [...lineItems]
                              updated[idx].locationId = e.target.value
                              updated[idx].locationCode = foundLoc?.code || "DEFAULT"
                              setLineItems(updated)
                            }}
                            className="w-full bg-white border border-slate-200 rounded-lg px-2.5 py-1.5 text-slate-900 font-mono text-[11px]"
                          >
                            <option value="">Auto FEFO (Semua Rak)</option>
                            {modalLocations.map((loc) => (
                              <option key={loc.id} value={loc.id}>
                                {loc.code} ({loc.name})
                              </option>
                            ))}
                          </select>
                        </div>

                        {/* Kuantitas */}
                        <div className="col-span-2">
                          <input
                            type="number"
                            min="1"
                            step="1"
                            inputMode="numeric"
                            aria-label={`Qty baris ${idx + 1}`}
                            aria-invalid={!!lineChecks[idx]?.error}
                            value={Number.isFinite(item.quantity) ? item.quantity : ""}
                            onChange={(e) => {
                              const raw = e.target.value
                              setLineItems((prev) =>
                                prev.map((li, i) => (i === idx ? { ...li, quantity: raw === "" ? NaN : Number(raw) } : li))
                              )
                            }}
                            className={`w-full bg-white border rounded-lg px-2 py-1.5 text-slate-900 text-right font-mono ${
                              lineChecks[idx]?.error ? "border-rose-400 bg-rose-50" : "border-slate-200"
                            }`}
                          />
                        </div>

                        {/* Availability feedback (pre-check) */}
                        <div className="col-span-12 -mt-1 text-[10px]">
                          {lineChecks[idx]?.error ? (
                            <span role="alert" className="text-rose-600 font-semibold">{lineChecks[idx].error}</span>
                          ) : lineChecks[idx]?.remaining !== undefined ? (
                            <span className="text-slate-500">
                              Sisa available {item.locationId ? "di rak ini" : "(semua rak)"}: {lineChecks[idx].remaining}
                            </span>
                          ) : modalStockLoading ? (
                            <span className="text-slate-400">Memeriksa stok gudang...</span>
                          ) : modalStockError ? (
                            <span className="text-amber-600">Data stok gagal dimuat — validasi akhir dilakukan server.</span>
                          ) : null}
                        </div>

                        {/* Bonus / Free Item Flag (PDF-01) */}
                        <div className="col-span-1 text-center">
                          <label className="cursor-pointer inline-flex flex-col items-center" title="Tandai sebagai Barang Bonus / Sampel Gratis">
                            <input
                              type="checkbox"
                              checked={!!item.isFreeItem}
                              onChange={(e) => {
                                const updated = [...lineItems]
                                updated[idx].isFreeItem = e.target.checked
                                setLineItems(updated)
                              }}
                              className="rounded border-slate-300 text-emerald-600 focus:ring-emerald-500"
                            />
                            <span className="text-[9px] font-bold text-emerald-700">Bonus</span>
                          </label>
                        </div>

                        {/* Hapus Baris */}
                        <div className="col-span-1 text-center">
                          <button
                            type="button"
                            onClick={() => {
                              setLineItems(lineItems.filter((_, i) => i !== idx))
                            }}
                            className="text-slate-400 hover:text-rose-600 p-1 rounded-md transition"
                          >
                            <Trash2 className="h-4 w-4" />
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* ── Modal Footer ── */}
              <div className="flex items-center justify-end gap-2 border-t border-slate-200 pt-4">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50 transition font-medium"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createDoMutation.isPending || hasLineErrors}
                  className="px-4 py-2 rounded-lg bg-[#2563EB] text-white font-semibold hover:bg-[#1D4ED8] shadow-sm disabled:opacity-50 transition"
                >
                  {createDoMutation.isPending ? "Menerbitkan..." : "Terbitkan Surat Jalan"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── Modal Pratinjau Cetak Dokumen (A4 Surat Jalan) ── */}
      {selectedDoForPrint && (
        <>
          {loadingPrintDetails ? (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 backdrop-blur-xs">
              <div className="bg-white border border-slate-200 rounded-xl p-6 text-center space-y-2 shadow-2xl">
                <RefreshCw className="h-6 w-6 animate-spin mx-auto text-[#2563EB]" />
                <p className="text-xs text-slate-600 font-medium">Menyiapkan dokumen cetak...</p>
              </div>
            </div>
          ) : (
            <PrintDeliveryOrder
              deliveryOrder={selectedDoForPrint}
              items={printItems}
              warehouseName={getWarehouseName(selectedDoForPrint.warehouse_id)}
              onClose={() => setSelectedDoForPrint(null)}
            />
          )}
        </>
      )}

      {/* ── Modal Pratinjau Cetak Picking List Terurut Rak (FE-06) ── */}
      {selectedDoForPicking && (
        <PrintPickingList
          detail={selectedDoForPicking}
          onClose={() => setSelectedDoForPicking(null)}
        />
      )}

      {/* ── Modal Stasiun Meja Kemas Barcode Scanner (FE-07) ── */}
      {selectedDoForPackStation && (
        <PackStationModal
          order={selectedDoForPackStation}
          initialItems={packStationItems}
          onClose={() => setSelectedDoForPackStation(null)}
          onSuccess={(updatedOrder) => {
            setSelectedDoForPackStation(null)
            setToast({
              type: "success",
              message: `Surat Jalan ${updatedOrder.do_number} telah selesai dikemas 100% (PACKED).`,
            })
            refetch()
          }}
        />
      )}

      {/* ── Modal Cetak Label Resi Thermal 100x150 mm (FE-08) ── */}
      {selectedDoForThermal && (
        <PrintThermalAWB
          order={selectedDoForThermal.order}
          items={selectedDoForThermal.items}
          onClose={() => setSelectedDoForThermal(null)}
        />
      )}

      {/* ── Modal Wave Release Grouping (PDF-05) ── */}
      <WaveReleaseModal
        isOpen={isWaveModalOpen}
        onClose={() => setIsWaveModalOpen(false)}
        warehouses={warehouses}
        defaultWarehouseId={selectedWarehouseFilter === "ALL" ? undefined : selectedWarehouseFilter}
        onWaveUpdated={refetch}
      />

      {/* ── Modal Cepat Tambah Pelanggan Baru (CR-02b) ── */}
      {showQuickCustomerModal && (
        <div className="fixed inset-0 z-60 flex items-center justify-center p-4 bg-slate-900/70 backdrop-blur-xs">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4">
            <div className="flex items-center justify-between border-b pb-3">
              <div className="flex items-center gap-2">
                <UserPlus className="h-5 w-5 text-[#2563EB]" />
                <h3 className="text-sm font-bold text-slate-900">Tambah Pelanggan Cepat</h3>
              </div>
              <button
                type="button"
                onClick={() => setShowQuickCustomerModal(false)}
                className="text-slate-400 hover:text-slate-600 p-1 rounded-lg"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleCreateQuickCustomer} className="space-y-3 text-xs">
              <div className="space-y-1">
                <label className="text-slate-700 font-semibold">Nama Toko / Pelanggan *</label>
                <input
                  type="text"
                  required
                  value={newCustName}
                  onChange={(e) => setNewCustName(e.target.value)}
                  placeholder="e.g. Toko Berkah Abadi"
                  className="w-full px-3 py-2 border border-slate-200 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                  autoFocus
                />
              </div>

              <div className="space-y-1">
                <label className="text-slate-700 font-semibold">Nomor Telepon / WhatsApp</label>
                <input
                  type="text"
                  value={newCustPhone}
                  onChange={(e) => setNewCustPhone(e.target.value)}
                  placeholder="e.g. 081234567890"
                  className="w-full px-3 py-2 border border-slate-200 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                />
              </div>

              <div className="space-y-1">
                <label className="text-slate-700 font-semibold">Alamat Lengkap Tujuan</label>
                <textarea
                  rows={2}
                  value={newCustAddress}
                  onChange={(e) => setNewCustAddress(e.target.value)}
                  placeholder="e.g. Jl. Raya Industri No. 45, Cikarang"
                  className="w-full px-3 py-2 border border-slate-200 rounded-lg text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                />
              </div>

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setShowQuickCustomerModal(false)}
                  className="px-3 py-1.5 border border-slate-200 rounded-lg text-slate-600 hover:bg-slate-50 font-medium"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={savingCustomer || !newCustName.trim()}
                  className="px-4 py-1.5 bg-[#2563EB] hover:bg-[#1D4ED8] text-white rounded-lg font-semibold disabled:opacity-50"
                >
                  {savingCustomer ? "Menyimpan..." : "Simpan & Pilih"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <ExportModal<DeliveryOrder>
        open={exporting}
        onClose={() => setExporting(false)}
        title="Surat Jalan"
        filename="surat-jalan"
        allRows={deliveryOrders}
        visibleRows={filteredOrders}
        visibleSummary={
          [
            statusFilter !== "ALL" ? `Status: ${getStatusLabel(statusFilter as DeliveryOrderStatus)}` : null,
            selectedWarehouseFilter !== "ALL" ? `Gudang: ${getWarehouseName(selectedWarehouseFilter)}` : null,
            searchQuery.trim() ? `Pencarian: "${searchQuery.trim()}"` : null,
          ].filter((x): x is string => x !== null)
        }
        columns={doExportColumns}
        filters={doExportFilters}
      />
    </div>
  )
}
