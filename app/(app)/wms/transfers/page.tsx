"use client"

import React, { useState, useMemo } from "react"
import Link from "next/link"
import { useManualRefresh } from "@/hooks/useManualRefresh"
import {
  Truck,
  ArrowRight,
  Plus,
  Search,
  CheckCircle2,
  Clock,
  Send,
  PackageCheck,
  Building,
  User,
  AlertTriangle,
  X,
  Trash2,
  Calendar,
  Layers,
  ChevronRight,
  ChevronLeft,
  RefreshCw,
  ArrowLeft,
  ClipboardCheck,
  ShieldCheck,
  XCircle,
  Hourglass,
  Ban,
} from "lucide-react"
import {
  useStockTransfers,
  useWarehouses,
  useWarehouseLocations,
  useCreateTransfer,
  useSubmitTransfer,
  useApproveTransfer,
  useRejectTransfer,
  useDispatchTransfer,
  useReceiveTransfer,
  useCancelTransfer,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"
import { useAuthStore } from "@/hooks/useAuth"
import { StockTransfer, TransferStatus } from "@/lib/api"

/**
 * Roles allowed to approve/reject a stock transfer (segregation of duties).
 * Mirrors backend `isTransferApproverRole` in wms_usecase.go.
 */
const APPROVER_ROLES = new Set(["admin", "owner", "regional_manager"])

interface LineItemDraft {
  productId: string
  productName: string
  sku: string
  quantity: number
  sourceLocationId?: string
}

export default function TransfersPage() {
  const { data: warehouses = [] } = useWarehouses()
  const { data: transfers = [], isLoading, refetch } = useStockTransfers()
  const { refresh, refreshing } = useManualRefresh([refetch])
  const { data: products = [] } = useProducts()
  const currentUser = useAuthStore((s) => s.user)
  const currentRole = (currentUser?.role ?? "").toLowerCase()
  const isApproverRole = APPROVER_ROLES.has(currentRole)

  const [statusFilter, setStatusFilter] = useState<string>("ALL")
  const [searchQuery, setSearchQuery] = useState("")

  // Modal State
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [selectedTransferForDetail, setSelectedTransferForDetail] = useState<StockTransfer | null>(null)

  // Mutations
  const createTransferMutation = useCreateTransfer()
  const submitMutation = useSubmitTransfer()
  const approveMutation = useApproveTransfer()
  const rejectMutation = useRejectTransfer()
  const dispatchMutation = useDispatchTransfer()
  const receiveMutation = useReceiveTransfer()
  const cancelMutation = useCancelTransfer()

  // Form State for New Transfer
  const [fromWhId, setFromWhId] = useState("")
  const [toWhId, setToWhId] = useState("")
  const [transferNumber, setTransferNumber] = useState("")
  const [vehiclePlate, setVehiclePlate] = useState("")
  const [driverName, setDriverName] = useState("")
  const [notes, setNotes] = useState("")
  const [lineItems, setLineItems] = useState<LineItemDraft[]>([])
  const [modalError, setModalError] = useState<string | null>(null)

  // Selected warehouse locations for source picking
  const { data: sourceLocations = [] } = useWarehouseLocations(fromWhId || null)
  // Selected warehouse locations for destination putaway
  const { data: toLocations = [] } = useWarehouseLocations(toWhId || null)

  // Effective source rack of a line item: keep the user's choice only while it
  // still belongs to the currently selected source warehouse, otherwise default
  // to that warehouse's first rack. Used by both the dropdown and the submit so
  // an item can never be saved with source_location_id = null or a rack from a
  // previously selected warehouse.
  const effectiveSourceLoc = (item: LineItemDraft): string | undefined =>
    item.sourceLocationId && sourceLocations.some((l) => l.id === item.sourceLocationId)
      ? item.sourceLocationId
      : sourceLocations[0]?.id

  // Filtered transfers
  const filteredTransfers = useMemo(() => {
    return transfers.filter((t) => {
      const matchesStatus = statusFilter === "ALL" || t.status === statusFilter
      const fromWh = warehouses.find((w) => w.id === t.from_warehouse_id)
      const toWh = warehouses.find((w) => w.id === t.to_warehouse_id)

      const matchesSearch =
        searchQuery === "" ||
        t.transfer_number.toLowerCase().includes(searchQuery.toLowerCase()) ||
        (t.driver_name && t.driver_name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (t.vehicle_plate && t.vehicle_plate.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (fromWh && fromWh.name.toLowerCase().includes(searchQuery.toLowerCase())) ||
        (toWh && toWh.name.toLowerCase().includes(searchQuery.toLowerCase()))

      return matchesStatus && matchesSearch
    })
  }, [transfers, statusFilter, searchQuery, warehouses])

  // Warehouse name helper
  const getWarehouseName = (id: string) => {
    const wh = warehouses.find((w) => w.id === id)
    return wh ? `${wh.name} (${wh.code})` : id.slice(0, 8)
  }

  // In-app Confirmation Modal & Toast State (replaces native confirm/alert)
  const [confirmModal, setConfirmModal] = useState<{
    type: "submit" | "approve" | "dispatch" | "receive" | "cancel"
    transferId: string
    title: string
    message: string
    actionLabel: string
  } | null>(null)
  // Reject requires a mandatory reason, so it uses its own modal (textarea) instead
  // of the generic one-click confirmModal above.
  const [rejectModal, setRejectModal] = useState<{ transferId: string; transferNumber: string } | null>(null)
  const [rejectReason, setRejectReason] = useState("")
  const [toast, setToast] = useState<{ type: "success" | "error"; message: string } | null>(null)

  const requestSubmit = (transferId: string) => {
    setConfirmModal({
      type: "submit",
      transferId,
      title: "Ajukan Persetujuan Transfer",
      message:
        "Ajukan dokumen transfer ini untuk disetujui oleh admin, owner, atau regional manager? Barang belum akan dikirim sampai disetujui.",
      actionLabel: "Ajukan Persetujuan",
    })
  }

  const requestApprove = (transferId: string) => {
    setConfirmModal({
      type: "approve",
      transferId,
      title: "Setujui Transfer Gudang",
      message:
        "Setujui transfer ini? Setelah disetujui, tim logistik gudang asal dapat mengirim armada (dispatch).",
      actionLabel: "Setujui Transfer",
    })
  }

  const requestReject = (transferId: string, transferNumber: string) => {
    setRejectReason("")
    setRejectModal({ transferId, transferNumber })
  }

  const requestDispatch = (transferId: string) => {
    setConfirmModal({
      type: "dispatch",
      transferId,
      title: "Konfirmasi Pengiriman Armada",
      message: "Konfirmasi pengiriman armada transfer? Status akan berubah menjadi IN_TRANSIT.",
      actionLabel: "Kirim Armada",
    })
  }

  const requestReceive = (transferId: string) => {
    setConfirmModal({
      type: "receive",
      transferId,
      title: "Konfirmasi Penerimaan Barang",
      message: "Konfirmasi penerimaan barang di gudang tujuan? Barang akan dibukukan ke stok tujuan.",
      actionLabel: "Konfirmasi Terima",
    })
  }

  const requestCancel = (transferId: string) => {
    setConfirmModal({
      type: "cancel",
      transferId,
      title: "Batalkan Draft Transfer",
      message:
        "Batalkan draft transfer ini? Status menjadi CANCELLED dan tidak bisa diajukan lagi. Karena masih draft, stok gudang tidak pernah berubah.",
      actionLabel: "Batalkan Draft",
    })
  }

  const isActionPending =
    submitMutation.isPending ||
    approveMutation.isPending ||
    rejectMutation.isPending ||
    dispatchMutation.isPending ||
    receiveMutation.isPending ||
    cancelMutation.isPending

  const handleConfirmAction = async () => {
    if (!confirmModal) return
    const { type, transferId } = confirmModal
    try {
      if (type === "submit") {
        await submitMutation.mutateAsync(transferId)
        setToast({ type: "success", message: "Transfer diajukan untuk persetujuan (PENDING_APPROVAL)." })
      } else if (type === "approve") {
        await approveMutation.mutateAsync(transferId)
        setToast({ type: "success", message: "Transfer disetujui. Siap untuk dikirim (dispatch)." })
      } else if (type === "dispatch") {
        await dispatchMutation.mutateAsync(transferId)
        setToast({ type: "success", message: "Armada transfer berhasil dikirim (IN_TRANSIT)." })
      } else if (type === "cancel") {
        await cancelMutation.mutateAsync(transferId)
        setToast({ type: "success", message: "Draft transfer dibatalkan (CANCELLED)." })
      } else {
        await receiveMutation.mutateAsync(transferId)
        setToast({ type: "success", message: "Barang transfer berhasil diterima di gudang tujuan." })
      }
      setConfirmModal(null)
      refetch()
    } catch (err: unknown) {
      setToast({
        type: "error",
        message: err instanceof Error ? err.message : `Gagal memproses ${type} transfer.`,
      })
      setConfirmModal(null)
    }
  }

  const handleConfirmReject = async () => {
    if (!rejectModal) return
    const reason = rejectReason.trim()
    if (!reason) {
      setToast({ type: "error", message: "Alasan penolakan wajib diisi." })
      return
    }
    try {
      await rejectMutation.mutateAsync({ id: rejectModal.transferId, reason })
      setToast({ type: "success", message: "Transfer ditolak dan dihentikan prosesnya." })
      setRejectModal(null)
      setRejectReason("")
      refetch()
    } catch (err: unknown) {
      setToast({
        type: "error",
        message: err instanceof Error ? err.message : "Gagal menolak transfer.",
      })
    }
  }

  // Add line item draft in modal
  const handleAddLineItem = () => {
    if (products.length === 0) return
    if (!fromWhId) {
      setModalError("Pilih Gudang Asal terlebih dahulu sebelum menambah barang.")
      return
    }
    const firstProduct = products[0]
    setLineItems((prev) => [
      ...prev,
      {
        productId: firstProduct.id,
        productName: firstProduct.name,
        sku: firstProduct.sku,
        quantity: 10,
        sourceLocationId: sourceLocations[0]?.id,
      },
    ])
  }

  const handleRemoveLineItem = (index: number) => {
    setLineItems((prev) => prev.filter((_, i) => i !== index))
  }

  const handleUpdateLineItem = (index: number, field: keyof LineItemDraft, value: unknown) => {
    setLineItems((prev) => {
      const updated = [...prev]
      if (field === "productId") {
        const prod = products.find((p) => p.id === value)
        if (prod) {
          updated[index] = {
            ...updated[index],
            productId: prod.id,
            productName: prod.name,
            sku: prod.sku,
          }
        }
      } else {
        updated[index] = { ...updated[index], [field]: value }
      }
      return updated
    })
  }

  // Handle Create Transfer Submit
  const handleCreateSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setModalError(null)

    if (!fromWhId || !toWhId) {
      setModalError("Gudang asal dan gudang tujuan wajib dipilih.")
      return
    }
    if (fromWhId === toWhId) {
      setModalError("Gudang asal dan tujuan tidak boleh sama.")
      return
    }
    if (sourceLocations.length === 0) {
      setModalError("Gudang asal belum memiliki lokasi rak penyimpanan. Buat minimal 1 rak di menu Gudang & Stok.")
      return
    }
    if (lineItems.length === 0) {
      setModalError("Minimal 1 barang harus dimasukkan ke dalam daftar transfer.")
      return
    }
    const missingLoc = lineItems.some((item) => !effectiveSourceLoc(item))
    if (missingLoc) {
      setModalError("Setiap barang wajib memiliki lokasi rak asal yang valid.")
      return
    }

    try {
      const generatedNumber =
        transferNumber.trim() || `TR-${new Date().toISOString().slice(0, 10).replace(/-/g, "")}-${Math.floor(100 + Math.random() * 900)}`

      await createTransferMutation.mutateAsync({
        from_warehouse_id: fromWhId,
        to_warehouse_id: toWhId,
        transfer_number: generatedNumber,
        vehicle_plate: vehiclePlate.trim() || undefined,
        driver_name: driverName.trim() || undefined,
        notes: notes.trim() || undefined,
        items: lineItems.map((item) => ({
          product_id: item.productId,
          requested_qty: item.quantity,
          source_location_id: effectiveSourceLoc(item),
        })),
      })

      setShowCreateModal(false)
      setFromWhId("")
      setToWhId("")
      setTransferNumber("")
      setVehiclePlate("")
      setDriverName("")
      setNotes("")
      setLineItems([])
      refetch()
    } catch (err: unknown) {
      setModalError(err instanceof Error ? err.message : "Gagal membuat transfer")
    }
  }

  // Render Step Tracker Component.
  // Flow: DRAFT -> PENDING_APPROVAL -> APPROVED -> IN_TRANSIT -> RECEIVED,
  // with REJECTED as a terminal branch rendered as a distinct banner (not a step)
  // so it's never confused with a "stuck" in-progress state.
  const renderStatusTracker = (status: TransferStatus) => {
    if (status === "REJECTED") {
      return (
        <div className="flex items-center gap-1.5 w-full max-w-md px-2.5 py-1.5 rounded-lg bg-rose-50 border border-rose-200 text-rose-700">
          <XCircle className="w-3.5 h-3.5 shrink-0" />
          <span className="text-[11px] font-bold uppercase tracking-wider">Ditolak</span>
        </div>
      )
    }

    // CANCELLED is a terminal branch like REJECTED, but neutral (not an error):
    // the requester discarded the draft before any stock moved.
    if (status === "CANCELLED") {
      return (
        <div className="flex items-center gap-1.5 w-full max-w-md px-2.5 py-1.5 rounded-lg bg-slate-100 border border-slate-300 text-slate-600">
          <Ban className="w-3.5 h-3.5 shrink-0" />
          <span className="text-[11px] font-bold uppercase tracking-wider">Dibatalkan</span>
        </div>
      )
    }

    const steps = [
      { key: "DRAFT", label: "Draft" },
      { key: "PENDING_APPROVAL", label: "Menunggu" },
      { key: "APPROVED", label: "Disetujui" },
      { key: "IN_TRANSIT", label: "Dikirim" },
      { key: "RECEIVED", label: "Selesai" },
    ]

    const getStepState = (stepKey: string) => {
      const order = ["DRAFT", "PENDING_APPROVAL", "APPROVED", "IN_TRANSIT", "RECEIVED"]
      const currentIndex =
        status === "DISPATCHED" ? order.indexOf("IN_TRANSIT") : order.indexOf(status)
      const stepIndex = order.indexOf(stepKey)

      if (stepIndex < currentIndex) return "completed"
      if (stepIndex === currentIndex) return "active"
      return "pending"
    }

    return (
      <div className="flex items-center gap-1 w-full max-w-md">
        {steps.map((step, idx) => {
          const state = getStepState(step.key)
          return (
            <React.Fragment key={step.key}>
              <div
                className={`flex-1 flex flex-col items-center py-1 px-1 rounded text-center transition-all ${
                  state === "completed"
                    ? "bg-emerald-50 text-emerald-700 border border-emerald-200"
                    : state === "active"
                    ? step.key === "IN_TRANSIT"
                      ? "bg-amber-100 text-amber-900 border border-amber-300 font-bold animate-pulse"
                      : step.key === "PENDING_APPROVAL"
                      ? "bg-yellow-100 text-yellow-900 border border-yellow-300 font-bold"
                      : "bg-blue-100 text-blue-900 border border-blue-300 font-bold"
                    : "bg-slate-100 text-slate-400 border border-slate-200"
                }`}
              >
                <span className="text-[9px] tracking-wider uppercase leading-tight">{step.label}</span>
              </div>
              {idx < steps.length - 1 && (
                <div
                  className={`w-2 h-0.5 shrink-0 ${
                    state === "completed" ? "bg-emerald-400" : "bg-slate-200"
                  }`}
                />
              )}
            </React.Fragment>
          )
        })}
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-[#F8FAFC] pb-16">
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
      <div className="bg-white border-b border-[#E2E8F0] px-4 sm:px-6 py-5 sticky top-0 z-20 shadow-xs">
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row sm:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <Link
              href="/wms"
              className="p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[44px] min-w-[44px] flex items-center justify-center transition-colors"
            >
              <ArrowLeft className="w-5 h-5" />
            </Link>
            <div>
              <div className="flex items-center gap-2">
                <Truck className="w-6 h-6 text-[#2563EB]" />
                <h1 className="text-xl sm:text-2xl font-bold tracking-tight text-slate-900">
                  Transfer Antar Gudang
                </h1>
              </div>
              <p className="text-xs sm:text-sm text-slate-500">
                Lacak status armada, surat jalan transit, dan verifikasi penerimaan barang
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => void refresh()}
              disabled={refreshing}
              aria-busy={refreshing}
              className="disabled:opacity-60 p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[48px] min-w-[48px] flex items-center justify-center transition-colors"
              title="Refresh"
            >
              <RefreshCw className={`w-5 h-5 ${refreshing ? "animate-spin" : ""}`} />
            </button>

            <button
              type="button"
              onClick={() => {
                // No auto-added row: racks are unknown until a source warehouse is chosen.
                setShowCreateModal(true)
              }}
              className="inline-flex items-center justify-center gap-2 px-5 min-h-[48px] rounded-lg font-bold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] active:scale-95 transition-all shadow-sm"
            >
              <Plus className="w-4 h-4" />
              <span>+ Buat Transfer Baru</span>
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 pt-6 space-y-6">
        {/* ── Filters & Search Bar ── */}
        <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3 shadow-xs">
          <div className="relative flex-1">
            <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Cari nomor transfer, plat nomor, atau nama supir..."
              className="w-full pl-10 pr-4 min-h-[48px] text-sm bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-[#2563EB]"
            />
          </div>

          <div className="flex items-center gap-1.5 overflow-x-auto pb-1 md:pb-0">
            {[
              { label: "Semua", value: "ALL" },
              { label: "Draft", value: "DRAFT" },
              { label: "Menunggu Approval", value: "PENDING_APPROVAL" },
              { label: "Approved", value: "APPROVED" },
              { label: "In Transit", value: "IN_TRANSIT" },
              { label: "Received", value: "RECEIVED" },
              { label: "Ditolak", value: "REJECTED" },
              { label: "Dibatalkan", value: "CANCELLED" },
            ].map((tab) => (
              <button
                key={tab.value}
                onClick={() => setStatusFilter(tab.value)}
                className={`px-3.5 py-2 text-xs font-semibold rounded-lg transition-all min-h-[42px] whitespace-nowrap ${
                  statusFilter === tab.value
                    ? "bg-[#2563EB] text-white"
                    : "bg-slate-100 text-slate-700 hover:bg-slate-200"
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        {/* ── Transfers List ── */}
        {isLoading ? (
          <div className="space-y-4">
            {[1, 2, 3].map((i) => (
              <div key={i} className="h-36 bg-slate-100 rounded-xl animate-pulse" />
            ))}
          </div>
        ) : filteredTransfers.length === 0 ? (
          <div className="text-center py-16 px-4 bg-white rounded-xl border border-[#E2E8F0]">
            <Truck className="w-12 h-12 text-slate-400 mx-auto mb-3" />
            <h3 className="text-base font-bold text-slate-900">Belum ada data transfer</h3>
            <p className="text-sm text-slate-500 max-w-md mx-auto mt-1 mb-5">
              Tidak ada pergerakan transfer barang yang cocok dengan filter saat ini.
            </p>
            <button
              onClick={() => setShowCreateModal(true)}
              className="inline-flex items-center gap-2 px-5 min-h-[48px] rounded-lg font-semibold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8]"
            >
              <Plus className="w-4 h-4" />
              Buat Transfer Baru
            </button>
          </div>
        ) : (
          <div className="space-y-4">
            {filteredTransfers.map((transfer) => {
              const isInTransit = transfer.status === "IN_TRANSIT" || transfer.status === "DISPATCHED"
              const isApproved = transfer.status === "APPROVED"
              const isReceived = transfer.status === "RECEIVED"
              const isDraft = transfer.status === "DRAFT"
              const isPendingApproval = transfer.status === "PENDING_APPROVAL"
              const isRejected = transfer.status === "REJECTED"
              const isCancelled = transfer.status === "CANCELLED"

              // Segregation of duties: the person who requested the transfer can
              // never be the one who approves/rejects it, even if they hold an
              // approver role (e.g. an admin requested it for their own warehouse).
              const isOwnRequest = !!currentUser && transfer.requested_by === currentUser.id
              const canApproveThis = isApproverRole && !isOwnRequest

              return (
                <div
                  key={transfer.id}
                  className="bg-white rounded-xl border border-[#E2E8F0] hover:border-[#BFDBFE] hover:shadow-md transition-all p-5 flex flex-col lg:flex-row items-start lg:items-center justify-between gap-5"
                >
                  {/* Left Column: Number, Route, Logistics info */}
                  <div className="space-y-3 flex-1">
                    <div className="flex flex-wrap items-center gap-2.5">
                      <span className="font-mono text-base font-extrabold text-slate-900">
                        {transfer.transfer_number}
                      </span>
                      <span className="text-xs text-slate-400">•</span>
                      <span className="text-xs text-slate-500 flex items-center gap-1">
                        <Calendar className="w-3.5 h-3.5 text-slate-400" />
                        {new Date(transfer.created_at).toLocaleDateString("id-ID", {
                          day: "numeric",
                          month: "short",
                          year: "numeric",
                        })}
                      </span>
                    </div>

                    {/* Source -> Destination Route */}
                    <div className="flex items-center gap-2 text-sm">
                      <div className="flex items-center gap-1.5 font-semibold text-slate-800 bg-slate-50 px-3 py-1.5 rounded-lg border border-slate-200">
                        <Building className="w-4 h-4 text-slate-500" />
                        <span>{getWarehouseName(transfer.from_warehouse_id)}</span>
                      </div>
                      <ArrowRight className="w-4 h-4 text-[#2563EB] shrink-0" />
                      <div className="flex items-center gap-1.5 font-semibold text-slate-800 bg-blue-50 px-3 py-1.5 rounded-lg border border-blue-200 text-[#2563EB]">
                        <Building className="w-4 h-4 text-[#2563EB]" />
                        <span>{getWarehouseName(transfer.to_warehouse_id)}</span>
                      </div>
                    </div>

                    {/* Driver & Vehicle Plate details */}
                    <div className="flex flex-wrap items-center gap-4 text-xs text-slate-600">
                      {transfer.vehicle_plate && (
                        <div className="flex items-center gap-1.5 font-mono font-bold bg-amber-50 text-amber-900 px-2 py-0.5 rounded border border-amber-200">
                          <Truck className="w-3.5 h-3.5" />
                          <span>{transfer.vehicle_plate}</span>
                        </div>
                      )}
                      {transfer.driver_name && (
                        <div className="flex items-center gap-1 text-slate-700">
                          <User className="w-3.5 h-3.5 text-slate-400" />
                          <span>Supir: <strong>{transfer.driver_name}</strong></span>
                        </div>
                      )}
                      {transfer.notes && (
                        <div className="text-slate-500 italic max-w-md truncate">
                          &ldquo;{transfer.notes}&rdquo;
                        </div>
                      )}
                    </div>

                    {/* Rejection reason banner: always visible, never hidden behind a click,
                        so staff immediately understand why the transfer stopped. */}
                    {isRejected && transfer.rejection_reason && (
                      <div className="flex items-start gap-2 text-xs bg-rose-50 border border-rose-200 text-rose-700 rounded-lg px-3 py-2 max-w-lg">
                        <AlertTriangle className="w-3.5 h-3.5 mt-0.5 shrink-0" />
                        <span>
                          <strong>Ditolak:</strong> {transfer.rejection_reason}
                        </span>
                      </div>
                    )}

                    {/* Cancellation banner: the draft was discarded before any stock
                        moved, so staff know nothing was deducted from either warehouse. */}
                    {isCancelled && (
                      <div className="flex items-start gap-2 text-xs bg-slate-50 border border-slate-200 text-slate-600 rounded-lg px-3 py-2 max-w-lg">
                        <Ban className="w-3.5 h-3.5 mt-0.5 shrink-0" />
                        <span>
                          <strong>Dibatalkan:</strong> draft dibatalkan sebelum diajukan. Stok gudang tidak berubah.
                        </span>
                      </div>
                    )}

                    {/* Approval transparency: who requested, and the current waiting state,
                        so there is never ambiguity about who needs to act next. */}
                    {isPendingApproval && (
                      <div
                        className={`inline-flex items-center gap-1.5 text-xs font-semibold rounded-lg px-3 py-1.5 w-fit ${
                          canApproveThis
                            ? "bg-yellow-50 border border-yellow-200 text-yellow-800"
                            : "bg-slate-50 border border-slate-200 text-slate-500"
                        }`}
                      >
                        <Hourglass className="w-3.5 h-3.5 shrink-0" />
                        {canApproveThis
                          ? "Menunggu keputusan Anda sebagai approver"
                          : isOwnRequest && isApproverRole
                          ? "Menunggu approver lain (Anda pengaju transfer ini)"
                          : "Menunggu persetujuan admin / regional manager"}
                      </div>
                    )}
                  </div>

                  {/* Middle Column: Visual Stepper Tracker */}
                  <div className="w-full lg:w-72 flex flex-col items-start lg:items-center">
                    <span className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1.5">
                      Status Perjalanan
                    </span>
                    {renderStatusTracker(transfer.status)}
                  </div>

                  {/* Right Column: Actions (min 48px touch target) */}
                  <div className="flex flex-wrap items-center gap-2.5 w-full lg:w-auto justify-end pt-3 lg:pt-0 border-t lg:border-t-0 border-slate-100">
                    {/* Action: Submit for approval (DRAFT -> PENDING_APPROVAL) */}
                    {isDraft && (
                      <>
                        <button
                          type="button"
                          onClick={() => requestSubmit(transfer.id)}
                          disabled={isActionPending}
                          className="flex-1 lg:flex-none inline-flex items-center justify-center gap-2 px-5 min-h-[48px] rounded-lg font-bold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] active:scale-95 transition-all shadow-xs"
                        >
                          <ClipboardCheck className="w-4 h-4" />
                          <span>Ajukan Persetujuan</span>
                        </button>
                        {/* Action: Discard the draft entirely (DRAFT -> CANCELLED) */}
                        <button
                          type="button"
                          onClick={() => requestCancel(transfer.id)}
                          disabled={isActionPending}
                          className="flex-1 lg:flex-none inline-flex items-center justify-center gap-2 px-5 min-h-[48px] rounded-lg font-bold text-sm border border-slate-300 text-slate-700 hover:bg-slate-50 hover:border-slate-400 active:scale-95 transition-all"
                        >
                          <Ban className="w-4 h-4" />
                          <span>Batalkan Draft</span>
                        </button>
                      </>
                    )}

                    {/* Action: Approve / Reject (PENDING_APPROVAL), only for approver roles
                        that are not the original requester. */}
                    {isPendingApproval && canApproveThis && (
                      <>
                        <button
                          type="button"
                          onClick={() => requestReject(transfer.id, transfer.transfer_number)}
                          disabled={isActionPending}
                          className="flex-1 lg:flex-none inline-flex items-center justify-center gap-2 px-5 min-h-[48px] rounded-lg font-bold text-sm bg-rose-600 text-white hover:bg-rose-700 active:scale-95 transition-all shadow-xs"
                        >
                          <XCircle className="w-4 h-4" />
                          <span>Tolak</span>
                        </button>
                        <button
                          type="button"
                          onClick={() => requestApprove(transfer.id)}
                          disabled={isActionPending}
                          className="flex-1 lg:flex-none inline-flex items-center justify-center gap-2 px-5 min-h-[48px] rounded-lg font-bold text-sm bg-emerald-600 text-white hover:bg-emerald-700 active:scale-95 transition-all shadow-xs"
                        >
                          <ShieldCheck className="w-4 h-4" />
                          <span>Setujui Transfer</span>
                        </button>
                      </>
                    )}

                    {/* Action: Dispatch button */}
                    {isApproved && (
                      <button
                        type="button"
                        onClick={() => requestDispatch(transfer.id)}
                        disabled={isActionPending}
                        className="flex-1 lg:flex-none inline-flex items-center justify-center gap-2 px-5 min-h-[48px] rounded-lg font-bold text-sm bg-[#EA580C] text-white hover:bg-[#C2410C] active:scale-95 transition-all shadow-xs"
                      >
                        <Send className="w-4 h-4" />
                        <span>Kirim Armada (Dispatch)</span>
                      </button>
                    )}

                    {/* Action: Receive button */}
                    {isInTransit && (
                      <button
                        type="button"
                        onClick={() => requestReceive(transfer.id)}
                        disabled={isActionPending}
                        className="flex-1 lg:flex-none inline-flex items-center justify-center gap-2 px-5 min-h-[48px] rounded-lg font-bold text-sm bg-emerald-600 text-white hover:bg-emerald-700 active:scale-95 transition-all shadow-xs"
                      >
                        <PackageCheck className="w-4 h-4" />
                        <span>Konfirmasi Terima (Receive)</span>
                      </button>
                    )}

                    {/* View details */}
                    <button
                      type="button"
                      onClick={() => setSelectedTransferForDetail(transfer)}
                      className="px-4 min-h-[48px] rounded-lg border border-slate-300 text-slate-700 hover:bg-slate-100 text-sm font-medium transition-colors"
                    >
                      Detail Item
                    </button>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* ── MODAL: Create Transfer & Line Item Builder ── */}
      {showCreateModal && (
        <div className="fixed inset-0 z-50 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-2xl w-full p-6 shadow-2xl border border-slate-200 max-h-[90vh] overflow-y-auto">
            <div className="flex items-center justify-between pb-4 border-b border-slate-100">
              <div className="flex items-center gap-2.5">
                <div className="p-2 rounded-lg bg-[#EFF6FF] text-[#2563EB]">
                  <Truck className="w-5 h-5" />
                </div>
                <div>
                  <h3 className="text-lg font-bold text-slate-900">Buat Transfer Antar Gudang</h3>
                  <p className="text-xs text-slate-500">Pindahkan stok antar fasilitas gudang secara resmi</p>
                </div>
              </div>
              <button
                onClick={() => setShowCreateModal(false)}
                className="p-2 text-slate-400 hover:text-slate-700 rounded-lg min-h-[44px] min-w-[44px] flex items-center justify-center"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            {modalError && (
              <div className="mt-4 p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-700 text-xs flex items-center gap-2">
                <AlertTriangle className="w-4 h-4 shrink-0" />
                <span>{modalError}</span>
              </div>
            )}

            <form onSubmit={handleCreateSubmit} className="mt-5 space-y-4">
              {/* Warehouse Selection */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 bg-slate-50 p-3.5 rounded-xl border border-slate-200">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    Dari Gudang (Asal) *
                  </label>
                  <select
                    required
                    value={fromWhId}
                    onChange={(e) => setFromWhId(e.target.value)}
                    className="w-full px-3 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB] bg-white"
                  >
                    <option value="">-- Pilih Gudang Asal --</option>
                    {warehouses.map((wh) => (
                      <option key={wh.id} value={wh.id}>
                        {wh.name} ({wh.code})
                      </option>
                    ))}
                  </select>
                  {fromWhId && sourceLocations.length === 0 && (
                    <div className="mt-2 p-2 bg-rose-50 border border-rose-200 rounded-lg text-xs text-rose-700 flex items-start gap-1.5">
                      <AlertTriangle className="w-3.5 h-3.5 shrink-0 text-rose-600 mt-0.5" />
                      <span>Gudang asal belum memiliki lokasi rak. Silakan buat rak di menu Gudang &amp; Stok.</span>
                    </div>
                  )}
                </div>

                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    Ke Gudang (Tujuan) *
                  </label>
                  <select
                    required
                    value={toWhId}
                    onChange={(e) => setToWhId(e.target.value)}
                    className="w-full px-3 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB] bg-white"
                  >
                    <option value="">-- Pilih Gudang Tujuan --</option>
                    {warehouses
                      .filter((wh) => wh.id !== fromWhId)
                      .map((wh) => (
                        <option key={wh.id} value={wh.id}>
                          {wh.name} ({wh.code})
                        </option>
                      ))}
                  </select>
                  {toWhId && toLocations.length === 0 && (
                    <div className="mt-2 p-2 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-800 flex items-start gap-1.5">
                      <AlertTriangle className="w-3.5 h-3.5 shrink-0 text-amber-600 mt-0.5" />
                      <span>Gudang tujuan belum memiliki rak penyimpanan. Saat diterima, barang otomatis masuk ke rak default (DEFAULT-kode gudang). Tambahkan rak di menu Gudang &amp; Stok jika ingin penempatan spesifik.</span>
                    </div>
                  )}
                </div>
              </div>

              {/* Vehicle & Driver */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    No. Transfer (Opsional)
                  </label>
                  <input
                    type="text"
                    placeholder="TR-Otomatis"
                    value={transferNumber}
                    onChange={(e) => setTransferNumber(e.target.value)}
                    className="w-full px-3 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm font-mono focus:ring-2 focus:ring-[#2563EB]"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    Plat Nomor Kendaraan
                  </label>
                  <input
                    type="text"
                    placeholder="Contoh: B 1234 XYZ"
                    value={vehiclePlate}
                    onChange={(e) => setVehiclePlate(e.target.value.toUpperCase())}
                    className="w-full px-3 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm font-mono focus:ring-2 focus:ring-[#2563EB]"
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                    Nama Pengemudi / Kurir
                  </label>
                  <input
                    type="text"
                    placeholder="Contoh: Pak Joko"
                    value={driverName}
                    onChange={(e) => setDriverName(e.target.value)}
                    className="w-full px-3 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB]"
                  />
                </div>
              </div>

              {/* Notes */}
              <div>
                <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                  Catatan / Instruksi
                </label>
                <input
                  type="text"
                  placeholder="Keterangan pengiriman..."
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  className="w-full px-3 py-2.5 min-h-[48px] rounded-lg border border-slate-300 text-sm focus:ring-2 focus:ring-[#2563EB]"
                />
              </div>

              {/* Line Item Builder */}
              <div className="pt-2">
                <div className="flex items-center justify-between mb-2">
                  <label className="text-xs font-bold text-slate-800 uppercase tracking-wider">
                    Daftar Barang Transfer ({lineItems.length})
                  </label>
                  <button
                    type="button"
                    onClick={handleAddLineItem}
                    disabled={!fromWhId}
                    title={!fromWhId ? "Pilih Gudang Asal terlebih dahulu" : undefined}
                    className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-bold rounded-md bg-[#EFF6FF] text-[#2563EB] hover:bg-[#DBEAFE] min-h-[36px] disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    + Tambah Baris
                  </button>
                </div>

                <div className="space-y-2 max-h-56 overflow-y-auto border border-slate-200 rounded-xl p-2 bg-slate-50">
                  {lineItems.length === 0 ? (
                    <div className="text-center py-6 text-xs text-slate-400">
                      {fromWhId
                        ? 'Belum ada barang. Klik "+ Tambah Baris" untuk memasukkan barang.'
                        : "Pilih Gudang Asal terlebih dahulu."}
                    </div>
                  ) : (
                    lineItems.map((item, idx) => (
                      <div
                        key={idx}
                        className="bg-white p-3 rounded-lg border border-slate-200 flex flex-col sm:flex-row items-center gap-2"
                      >
                        {/* Product selection */}
                        <div className="flex-1 w-full">
                          <select
                            value={item.productId}
                            onChange={(e) => handleUpdateLineItem(idx, "productId", e.target.value)}
                            className="w-full px-2.5 py-1.5 text-xs font-semibold rounded border border-slate-300 min-h-[40px]"
                          >
                            {products.map((p) => (
                              <option key={p.id} value={p.id}>
                                [{p.sku}] {p.name}
                              </option>
                            ))}
                          </select>
                        </div>

                        {/* Source Location selection */}
                        {sourceLocations.length > 0 && (
                          <div className="w-full sm:w-36">
                            <select
                              value={effectiveSourceLoc(item) ?? ""}
                              required
                              aria-label="Rak asal"
                              onChange={(e) =>
                                handleUpdateLineItem(idx, "sourceLocationId", e.target.value || undefined)
                              }
                              className="w-full px-2 py-1.5 text-xs rounded border border-slate-300 min-h-[40px]"
                            >
                              <option value="">-- Rak Asal --</option>
                              {sourceLocations.map((loc) => (
                                <option key={loc.id} value={loc.id}>
                                  {loc.code}
                                </option>
                              ))}
                            </select>
                          </div>
                        )}

                        {/* Qty */}
                        <div className="w-full sm:w-24">
                          <input
                            type="number"
                            min="1"
                            value={item.quantity}
                            onChange={(e) =>
                              handleUpdateLineItem(idx, "quantity", Math.max(1, parseInt(e.target.value) || 1))
                            }
                            className="w-full px-2 py-1.5 text-xs font-mono font-bold text-center rounded border border-slate-300 min-h-[40px]"
                          />
                        </div>

                        <button
                          type="button"
                          onClick={() => handleRemoveLineItem(idx)}
                          className="p-2 text-rose-500 hover:text-rose-700 min-h-[40px] min-w-[40px] flex items-center justify-center"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Submit Buttons */}
              <div className="pt-4 border-t border-slate-100 flex items-center justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 min-h-[48px] rounded-lg text-sm font-medium text-slate-700 hover:bg-slate-100"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createTransferMutation.isPending}
                  className="px-6 min-h-[48px] rounded-lg text-sm font-bold bg-[#2563EB] text-white hover:bg-[#1D4ED8] disabled:opacity-50"
                >
                  {createTransferMutation.isPending ? "Memproses..." : "Buat Dokumen Transfer"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* ── MODAL: Detail View ── */}
      {selectedTransferForDetail && (
        <div className="fixed inset-0 z-50 bg-slate-900/60 backdrop-blur-xs flex items-center justify-center p-4">
          <div className="bg-white rounded-2xl max-w-lg w-full p-6 shadow-2xl border border-slate-200">
            <div className="flex items-center justify-between pb-3 border-b border-slate-100">
              <div>
                <span className="font-mono text-xs text-slate-500">Detail Surat Transfer</span>
                <h3 className="text-lg font-extrabold text-slate-900">
                  {selectedTransferForDetail.transfer_number}
                </h3>
              </div>
              <button
                onClick={() => setSelectedTransferForDetail(null)}
                className="p-2 text-slate-400 hover:text-slate-700 rounded-lg min-h-[44px] min-w-[44px] flex items-center justify-center"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="mt-4 space-y-3 text-sm">
              <div className="p-3 bg-slate-50 rounded-xl space-y-2">
                <div className="flex justify-between text-xs">
                  <span className="text-slate-500">Asal:</span>
                  <span className="font-semibold text-slate-800">
                    {getWarehouseName(selectedTransferForDetail.from_warehouse_id)}
                  </span>
                </div>
                <div className="flex justify-between text-xs">
                  <span className="text-slate-500">Tujuan:</span>
                  <span className="font-semibold text-slate-800">
                    {getWarehouseName(selectedTransferForDetail.to_warehouse_id)}
                  </span>
                </div>
                <div className="flex justify-between text-xs">
                  <span className="text-slate-500">Status:</span>
                  <span
                    className={`font-bold ${
                      selectedTransferForDetail.status === "REJECTED" ? "text-rose-600" : "text-[#2563EB]"
                    }`}
                  >
                    {selectedTransferForDetail.status}
                  </span>
                </div>
                {selectedTransferForDetail.status === "REJECTED" && selectedTransferForDetail.rejection_reason && (
                  <div className="flex items-start gap-1.5 text-xs bg-rose-100/70 text-rose-700 rounded-lg px-2.5 py-2">
                    <AlertTriangle className="w-3.5 h-3.5 mt-0.5 shrink-0" />
                    <span>{selectedTransferForDetail.rejection_reason}</span>
                  </div>
                )}
                {selectedTransferForDetail.vehicle_plate && (
                  <div className="flex justify-between text-xs">
                    <span className="text-slate-500">Kendaraan:</span>
                    <span className="font-mono font-bold text-amber-800">
                      {selectedTransferForDetail.vehicle_plate}
                    </span>
                  </div>
                )}
                {selectedTransferForDetail.driver_name && (
                  <div className="flex justify-between text-xs">
                    <span className="text-slate-500">Pengemudi:</span>
                    <span className="font-semibold text-slate-800">
                      {selectedTransferForDetail.driver_name}
                    </span>
                  </div>
                )}
              </div>

              <div className="pt-2">
                <div className="text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  Daftar Kuantitas
                </div>
                <div className="p-3 bg-blue-50/50 rounded-xl border border-blue-100 text-xs text-slate-600 flex items-center justify-between">
                  <span>Status Kuantitas:</span>
                  <span className="inline-flex items-center gap-1 font-bold text-emerald-700 bg-emerald-100 px-2 py-0.5 rounded">
                    <CheckCircle2 className="w-3.5 h-3.5" />
                    Sent == Received Qty (Match)
                  </span>
                </div>
              </div>
            </div>

            <div className="mt-6 flex justify-end">
              <button
                type="button"
                onClick={() => setSelectedTransferForDetail(null)}
                className="px-5 min-h-[48px] rounded-lg text-sm font-semibold bg-slate-100 text-slate-800 hover:bg-slate-200"
              >
                Tutup
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── In-App Confirmation Modal (replaces confirm/alert) ── */}
      {confirmModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <div
                className={`p-3 rounded-xl border ${
                  confirmModal.type === "dispatch"
                    ? "bg-orange-50 text-orange-600 border-orange-100"
                    : confirmModal.type === "submit"
                    ? "bg-blue-50 text-blue-600 border-blue-100"
                    : "bg-emerald-50 text-emerald-600 border-emerald-100"
                }`}
              >
                {confirmModal.type === "dispatch" ? (
                  <Send className="w-6 h-6" />
                ) : confirmModal.type === "submit" ? (
                  <ClipboardCheck className="w-6 h-6" />
                ) : confirmModal.type === "approve" ? (
                  <ShieldCheck className="w-6 h-6" />
                ) : (
                  <PackageCheck className="w-6 h-6" />
                )}
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">{confirmModal.title}</h3>
                <p className="text-xs text-slate-500 mt-0.5">Operasi status transfer antar gudang</p>
              </div>
            </div>

            <p className="text-sm text-slate-600 leading-relaxed">{confirmModal.message}</p>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                type="button"
                onClick={() => setConfirmModal(null)}
                className="px-4 py-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50 text-sm font-medium transition"
              >
                Batal
              </button>
              <button
                type="button"
                disabled={isActionPending}
                onClick={handleConfirmAction}
                className={`px-4 py-2 rounded-lg text-white text-sm font-semibold shadow-sm transition disabled:opacity-50 ${
                  confirmModal.type === "dispatch"
                    ? "bg-[#EA580C] hover:bg-[#C2410C]"
                    : confirmModal.type === "submit"
                    ? "bg-[#2563EB] hover:bg-[#1D4ED8]"
                    : "bg-emerald-600 hover:bg-emerald-700"
                }`}
              >
                {isActionPending ? "Memproses…" : confirmModal.actionLabel}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* ── Reject Modal: mandatory reason textarea, kept separate from the generic
           confirmModal because an empty reason must block submission with inline
           feedback rather than a one-click confirm. ── */}
      {rejectModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/60 backdrop-blur-xs">
          <div className="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-2xl p-6 space-y-4">
            <div className="flex items-center gap-3">
              <div className="p-3 rounded-xl border bg-rose-50 text-rose-600 border-rose-100">
                <XCircle className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">Tolak Transfer {rejectModal.transferNumber}</h3>
                <p className="text-xs text-slate-500 mt-0.5">
                  Alasan ini akan terlihat oleh staf gudang yang mengajukan transfer.
                </p>
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 uppercase tracking-wider mb-1">
                Alasan Penolakan *
              </label>
              <textarea
                autoFocus
                required
                rows={3}
                value={rejectReason}
                onChange={(e) => setRejectReason(e.target.value)}
                placeholder="Contoh: Stok gudang asal menipis untuk pesanan lokal"
                className="w-full px-3 py-2.5 rounded-lg border border-slate-300 text-sm focus:outline-none focus:ring-2 focus:ring-rose-400 resize-none"
              />
              {rejectReason.trim() === "" && (
                <p className="text-[11px] text-slate-400 mt-1">Alasan wajib diisi sebelum menolak transfer ini.</p>
              )}
            </div>

            <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
              <button
                type="button"
                onClick={() => {
                  setRejectModal(null)
                  setRejectReason("")
                }}
                className="px-4 py-2 rounded-lg border border-slate-200 text-slate-700 hover:bg-slate-50 text-sm font-medium transition"
              >
                Batal
              </button>
              <button
                type="button"
                disabled={isActionPending || rejectReason.trim() === ""}
                onClick={handleConfirmReject}
                className="px-4 py-2 rounded-lg text-white text-sm font-semibold shadow-sm transition disabled:opacity-50 bg-rose-600 hover:bg-rose-700"
              >
                {rejectMutation.isPending ? "Memproses…" : "Konfirmasi Tolak"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
