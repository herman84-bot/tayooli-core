"use client"

import React, { useState, useMemo } from "react"
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
} from "lucide-react"
import {
  useDeliveryOrders,
  useWarehouses,
  useWarehouseLocations,
  useCreateDeliveryOrder,
  useDispatchDeliveryOrder,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import { api, DeliveryOrder, DeliveryOrderItem, DeliveryOrderStatus } from "@/lib/api"
import { PrintDeliveryOrder } from "@/components/wms/PrintDeliveryOrder"

interface LineItemDraft {
  productId: string
  productName: string
  sku: string
  quantity: number
  locationId: string
  locationCode?: string
}

export default function DeliveryOrdersPage() {
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
  const [expeditionName, setExpeditionName] = useState("")
  const [trackingNumber, setTrackingNumber] = useState("")
  const [driverName, setDriverName] = useState("")
  const [vehiclePlate, setVehiclePlate] = useState("")
  const [recipientName, setRecipientName] = useState("")
  const [lineItems, setLineItems] = useState<LineItemDraft[]>([])
  const [modalError, setModalError] = useState<string | null>(null)

  // Locations for selected warehouse in create modal
  const { data: modalLocations = [] } = useWarehouseLocations(modalWarehouseId || null)

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
    setSalesOrderId(`so-${Math.random().toString(36).substring(2, 10)}`)
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

    for (let i = 0; i < lineItems.length; i++) {
      const it = lineItems[i]
      if (it.quantity <= 0) {
        setModalError(`Baris ${i + 1}: Kuantitas barang harus lebih besar dari 0.`)
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
        sales_order_id: salesOrderId || "SO-DIRECT",
        do_number: doNumber.trim(),
        expedition_name: expeditionName.trim() || undefined,
        tracking_number: trackingNumber.trim() || undefined,
        driver_name: driverName.trim() || undefined,
        vehicle_plate: vehiclePlate.trim() || undefined,
        recipient_name: recipientName.trim() || undefined,
        items: lineItems.map((item) => ({
          product_id: item.productId,
          quantity: item.quantity,
          location_id: item.locationId,
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
          <div className="space-y-1">
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
            <button
              onClick={() => refetch()}
              disabled={isFetching}
              className="p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[44px] min-w-[44px] flex items-center justify-center transition-colors disabled:opacity-50"
              title="Segarkan data"
            >
              <RefreshCw className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`} />
            </button>
            <button
              onClick={handleOpenCreateModal}
              className="inline-flex items-center gap-2 px-4 py-2.5 min-h-[44px] rounded-lg font-semibold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] active:scale-95 transition-all shadow-sm"
            >
              <Plus className="h-4 w-4" />
              <span>Buat Surat Jalan Baru</span>
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 pt-6 space-y-6">
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
                    <th className="py-3.5 px-4">Ref. Sales Order</th>
                    <th className="py-3.5 px-4">Ekspedisi & Plat</th>
                    <th className="py-3.5 px-4">Pengemudi / Supir</th>
                    <th className="py-3.5 px-4">Penerima</th>
                    <th className="py-3.5 px-4">Status</th>
                    <th className="py-3.5 px-4 text-right">Aksi</th>
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
                        <td className="py-3.5 px-4 font-mono text-slate-500">
                          {order.sales_order_id ? order.sales_order_id.slice(0, 12) : "—"}
                        </td>
                        <td className="py-3.5 px-4">
                          <div className="text-slate-800 font-medium">{order.expedition_name || "Internal"}</div>
                          <div className="text-[10px] font-mono text-slate-500">{order.vehicle_plate || "—"}</div>
                        </td>
                        <td className="py-3.5 px-4 text-slate-700">
                          {order.driver_name || "—"}
                        </td>
                        <td className="py-3.5 px-4 text-slate-900 font-medium">
                          {order.recipient_name || "Customer Retail"}
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
                          <div className="flex items-center justify-end gap-1.5">
                            <button
                              onClick={() => handleOpenPrint(order)}
                              title="Cetak Surat Jalan Resmi (A4)"
                              className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-200 bg-white text-slate-700 hover:bg-slate-50 hover:text-slate-900 transition text-[11px] font-medium shadow-2xs"
                            >
                              <Printer className="h-3.5 w-3.5 text-[#2563EB]" />
                              Cetak DO
                            </button>

                            {canDispatch && (
                              <button
                                onClick={(e) => requestDispatch(order, e)}
                                disabled={dispatchDoMutation.isPending}
                                title="Kirim Surat Jalan & Potong Stok"
                                className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-emerald-600 text-white hover:bg-emerald-700 transition text-[11px] font-medium shadow-2xs disabled:opacity-50"
                              >
                                <Send className="h-3.5 w-3.5" />
                                Dispatch
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
      </div>

      {/* ── In-App Confirmation Modal for Dispatch (replaces confirm/alert) ── */}
      {orderToDispatch && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4">
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
          <div className="w-full max-w-2xl bg-white border border-slate-200 rounded-2xl shadow-2xl overflow-hidden my-8">
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
                  <label className="text-slate-700 font-semibold">Ref. Sales Order ID *</label>
                  <input
                    type="text"
                    required
                    value={salesOrderId}
                    onChange={(e) => setSalesOrderId(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 font-mono focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                    placeholder="e.g. SO-2026-0045"
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
                  <label className="text-slate-700 font-semibold">Penerima / Customer *</label>
                  <input
                    type="text"
                    required
                    value={recipientName}
                    onChange={(e) => setRecipientName(e.target.value)}
                    className="w-full bg-white border border-slate-200 rounded-lg px-3 py-2 text-slate-900 focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
                    placeholder="e.g. PT Nusantara Retail Makmur"
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
                        <div className="col-span-4">
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
                            <option value="">Pilih Rak/Bin...</option>
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
                            value={item.quantity}
                            onChange={(e) => {
                              const updated = [...lineItems]
                              updated[idx].quantity = Number(e.target.value)
                              setLineItems(updated)
                            }}
                            className="w-full bg-white border border-slate-200 rounded-lg px-2 py-1.5 text-slate-900 text-right font-mono"
                          />
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
                  disabled={createDoMutation.isPending}
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
    </div>
  )
}
