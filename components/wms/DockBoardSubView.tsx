"use client"

import React, { useState } from "react"
import {
  Anchor,
  Clock,
  CheckCircle2,
  AlertCircle,
  AlertTriangle,
  Truck,
  Plus,
  Wrench,
  Search,
  X,
  Warehouse as WarehouseIcon,
  Play,
  Check,
  RotateCcw,
  Calendar,
  Building2,
  Phone,
  User,
  FileText,
} from "lucide-react"
import {
  useInboundDocks,
  useCreateDock,
  useUpdateDockStatus,
  useDockAppointments,
  useCreateAppointment,
  useAssignDock,
  useUpdateAppointmentStatus,
} from "@/hooks/useWMSDocksAndLPNs"
import type {
  InboundDock,
  DockAppointment,
  DockStatus,
  DockType,
  AppointmentStatus,
  CreateDockInput,
  CreateAppointmentInput,
} from "@/lib/api"

export interface DockBoardSubViewProps {
  warehouseId?: string | null
}

const DOCK_STATUS_BADGE: Record<DockStatus, { label: string; cls: string }> = {
  AVAILABLE: {
    label: "AVAILABLE",
    cls: "bg-emerald-50 text-emerald-700 border-emerald-200",
  },
  OCCUPIED: {
    label: "OCCUPIED",
    cls: "bg-amber-50 text-amber-700 border-amber-200",
  },
  MAINTENANCE: {
    label: "MAINTENANCE",
    cls: "bg-rose-50 text-rose-700 border-rose-200",
  },
}

const APPOINTMENT_STATUS_BADGE: Record<AppointmentStatus, { label: string; cls: string }> = {
  SCHEDULED: {
    label: "Terjadwal",
    cls: "bg-slate-100 text-slate-700 border-slate-200",
  },
  ARRIVED: {
    label: "Tiba",
    cls: "bg-blue-50 text-blue-700 border-blue-200",
  },
  UNLOADING: {
    label: "Bongkar",
    cls: "bg-amber-50 text-amber-700 border-amber-200",
  },
  COMPLETED: {
    label: "Selesai",
    cls: "bg-emerald-50 text-emerald-700 border-emerald-200",
  },
  CANCELLED: {
    label: "Dibatalkan",
    cls: "bg-rose-50 text-rose-700 border-rose-200",
  },
}

function formatDate(iso?: string): string {
  if (!iso) return "-"
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return "-"
  return d.toLocaleString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function DockBoardSubView({ warehouseId }: DockBoardSubViewProps) {
  // Data queries
  const {
    data: docks = [],
    isLoading: loadingDocks,
  } = useInboundDocks(warehouseId)

  const {
    data: appointments = [],
    isLoading: loadingAppointments,
  } = useDockAppointments(warehouseId)

  // Mutations
  const createDockMutation = useCreateDock()
  const updateDockStatusMutation = useUpdateDockStatus()
  const createAppointmentMutation = useCreateAppointment()
  const assignDockMutation = useAssignDock()
  const updateAppointmentStatusMutation = useUpdateAppointmentStatus()

  // UI state
  const [appointmentFilter, setAppointmentFilter] = useState<
    "ALL" | "SCHEDULED" | "ARRIVED" | "UNLOADING" | "COMPLETED"
  >("ALL")
  const [searchQuery, setSearchQuery] = useState("")

  // Modals
  const [isCreateDockOpen, setIsCreateDockOpen] = useState(false)
  const [isCreateAppointmentOpen, setIsCreateAppointmentOpen] = useState(false)
  const [assigningAppointment, setAssigningAppointment] = useState<DockAppointment | null>(null)
  const [selectedDockIdToAssign, setSelectedDockIdToAssign] = useState<string>("")

  // Alerts & Messages
  const [alertError, setAlertError] = useState<string | null>(null)
  const [alertSuccess, setAlertSuccess] = useState<string | null>(null)

  // Form states - Create Dock
  const [newDockCode, setNewDockCode] = useState("")
  const [newDockName, setNewDockName] = useState("")
  const [newDockType, setNewDockType] = useState<DockType>("INBOUND")
  const [newMaxTonnage, setNewMaxTonnage] = useState<number>(20)
  const [newDockNotes, setNewDockNotes] = useState("")

  // Form states - Create Appointment
  const [vendorName, setVendorName] = useState("")
  const [vehiclePlate, setVehiclePlate] = useState("")
  const [driverName, setDriverName] = useState("")
  const [driverPhone, setDriverPhone] = useState("")
  const [poReference, setPoReference] = useState("")
  const [estimatedArrival, setEstimatedArrival] = useState(() => {
    const d = new Date()
    d.setMinutes(d.getMinutes() - d.getTimezoneOffset())
    return d.toISOString().slice(0, 16)
  })
  const [initialDockId, setInitialDockId] = useState("")
  const [appointmentNotes, setAppointmentNotes] = useState("")

  // Metrics
  const totalDocks = docks.length
  const availableDocks = docks.filter((d) => d.status === "AVAILABLE").length
  const unloadingDocks = docks.filter(
    (d) => d.status === "OCCUPIED"
  ).length
  const waitingQueue = appointments.filter(
    (a) => a.status === "SCHEDULED" || a.status === "ARRIVED"
  ).length

  // Handlers for Dock Card Actions
  const handleToggleMaintenance = async (dock: InboundDock) => {
    try {
      setAlertError(null)
      const nextStatus: DockStatus =
        dock.status === "MAINTENANCE" ? "AVAILABLE" : "MAINTENANCE"
      await updateDockStatusMutation.mutateAsync({
        id: dock.id,
        status: nextStatus,
      })
      setAlertSuccess(
        `Status ${dock.dock_name} berhasil diubah ke ${nextStatus}`
      )
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal mengubah status dermaga"
      setAlertError(msg)
    }
  }

  const handleReleaseDock = async (dock: InboundDock) => {
    try {
      setAlertError(null)
      // If there is an active appointment occupying this dock, complete it
      const activeAppt = appointments.find(
        (a) =>
          a.dock_id === dock.id &&
          (a.status === "UNLOADING" || a.status === "ARRIVED")
      )
      if (activeAppt) {
        await updateAppointmentStatusMutation.mutateAsync({
          appointmentId: activeAppt.id,
          status: "COMPLETED",
        })
      }
      await updateDockStatusMutation.mutateAsync({
        id: dock.id,
        status: "AVAILABLE",
      })
      setAlertSuccess(`Dermaga ${dock.dock_name} berhasil dilepas dan kembali tersedia.`)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal melepas dermaga"
      setAlertError(msg)
    }
  }

  // Handlers for Appointment Actions
  const handleUpdateAppointmentStatus = async (
    appointmentId: string,
    nextStatus: AppointmentStatus
  ) => {
    try {
      setAlertError(null)
      await updateAppointmentStatusMutation.mutateAsync({
        appointmentId,
        status: nextStatus,
      })
      setAlertSuccess(`Status janji armada berhasil diperbarui ke ${nextStatus}`)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal memperbarui status janji"
      setAlertError(msg)
    }
  }

  const handleOpenAssignModal = (appt: DockAppointment) => {
    setAssigningAppointment(appt)
    setSelectedDockIdToAssign(appt.dock_id || "")
    setAlertError(null)
  }

  const handleConfirmAssignDock = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!assigningAppointment || !selectedDockIdToAssign) return

    try {
      setAlertError(null)
      await assignDockMutation.mutateAsync({
        appointmentId: assigningAppointment.id,
        dockId: selectedDockIdToAssign,
      })
      setAlertSuccess("Dermaga berhasil dialokasikan untuk armada ini.")
      setAssigningAppointment(null)
      setSelectedDockIdToAssign("")
    } catch (err: unknown) {
      const rawMsg = err instanceof Error ? err.message : String(err)
      // Catch ErrDockOccupied (409)
      if (
        rawMsg.toLowerCase().includes("occupied") ||
        rawMsg.toLowerCase().includes("undergoing unloading") ||
        rawMsg.includes("409") ||
        rawMsg.toLowerCase().includes("digunakan")
      ) {
        setAlertError(
          "Dermaga ini sedang digunakan oleh armada lain! Silakan pilih dermaga lain."
        )
      } else {
        setAlertError(rawMsg || "Gagal mengalokasikan dermaga")
      }
    }
  }

  // Form Submissions
  const handleCreateDockSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!warehouseId) {
      setAlertError("Pilih gudang terlebih dahulu.")
      return
    }
    try {
      setAlertError(null)
      const input: CreateDockInput = {
        warehouse_id: warehouseId,
        dock_code: newDockCode.trim() || undefined,
        dock_name: newDockName.trim(),
        dock_type: newDockType,
        max_tonnage: Number(newMaxTonnage) || 20,
        notes: newDockNotes.trim() || undefined,
      }
      await createDockMutation.mutateAsync(input)
      setAlertSuccess(`Dermaga ${newDockName} berhasil ditambahkan.`)
      setIsCreateDockOpen(false)
      setNewDockCode("")
      setNewDockName("")
      setNewDockNotes("")
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Gagal menambahkan dermaga"
      setAlertError(msg)
    }
  }

  const handleCreateAppointmentSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!warehouseId) {
      setAlertError("Pilih gudang terlebih dahulu.")
      return
    }
    try {
      setAlertError(null)
      const input: CreateAppointmentInput = {
        warehouse_id: warehouseId,
        vendor_name: vendorName.trim(),
        vehicle_plate: vehiclePlate.trim(),
        driver_name: driverName.trim(),
        driver_phone: driverPhone.trim() || undefined,
        po_reference: poReference.trim() || undefined,
        estimated_arrival: new Date(estimatedArrival).toISOString(),
        dock_id: initialDockId.trim() || undefined,
        notes: appointmentNotes.trim() || undefined,
      }
      await createAppointmentMutation.mutateAsync(input)
      setAlertSuccess("Jadwal armada berhasil dibuat.")
      setIsCreateAppointmentOpen(false)
      setVendorName("")
      setVehiclePlate("")
      setDriverName("")
      setDriverPhone("")
      setPoReference("")
      setInitialDockId("")
      setAppointmentNotes("")
    } catch (err: unknown) {
      const rawMsg = err instanceof Error ? err.message : String(err)
      if (
        rawMsg.toLowerCase().includes("occupied") ||
        rawMsg.toLowerCase().includes("undergoing unloading") ||
        rawMsg.includes("409") ||
        rawMsg.toLowerCase().includes("digunakan")
      ) {
        setAlertError(
          "Dermaga ini sedang digunakan oleh armada lain! Silakan pilih dermaga lain."
        )
      } else {
        setAlertError(rawMsg || "Gagal membuat jadwal armada")
      }
    }
  }

  // Filtered Appointments
  const filteredAppointments = appointments.filter((appt) => {
    if (appointmentFilter !== "ALL" && appt.status !== appointmentFilter) {
      return false
    }
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase()
      const matchNumber = appt.appointment_number?.toLowerCase().includes(q)
      const matchVendor = appt.vendor_name?.toLowerCase().includes(q)
      const matchPlate = appt.vehicle_plate?.toLowerCase().includes(q)
      const matchDriver = appt.driver_name?.toLowerCase().includes(q)
      const matchPO = appt.po_reference?.toLowerCase().includes(q)
      if (!matchNumber && !matchVendor && !matchPlate && !matchDriver && !matchPO) {
        return false
      }
    }
    return true
  })

  return (
    <div className="space-y-6">
      {/* Alert Banner */}
      {alertError && (
        <div
          role="alert"
          className="flex items-center justify-between rounded-xl border border-rose-200 bg-rose-50 p-4 text-xs text-rose-800 shadow-sm"
        >
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 flex-shrink-0 text-rose-600" />
            <span className="font-semibold">{alertError}</span>
          </div>
          <button
            onClick={() => setAlertError(null)}
            className="rounded p-1 hover:bg-rose-100 text-rose-600"
            aria-label="Tutup pesan error"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {alertSuccess && (
        <div className="flex items-center justify-between rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-xs text-emerald-800 shadow-sm">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 flex-shrink-0 text-emerald-600" />
            <span>{alertSuccess}</span>
          </div>
          <button
            onClick={() => setAlertSuccess(null)}
            className="rounded p-1 hover:bg-emerald-100 text-emerald-600"
            aria-label="Tutup pesan sukses"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      {/* Header Toolbar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-white p-4 rounded-xl border border-slate-200 shadow-sm">
        <div>
          <h2 className="text-base font-bold text-slate-900 flex items-center gap-2">
            <WarehouseIcon className="h-5 w-5 text-emerald-600" />
            Papan Dermaga & Antrean Armada (Dock Management)
          </h2>
          <p className="text-xs text-slate-500 mt-0.5">
            Manajemen alokasi dermaga bongkar muat dan jadwal antrean armada pemasok masuk secara real-time.
          </p>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => setIsCreateDockOpen(true)}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-white border border-slate-300 text-slate-700 hover:bg-slate-50 text-xs font-semibold shadow-sm transition-colors"
          >
            <Plus className="h-3.5 w-3.5" />
            Tambah Dermaga
          </button>
          <button
            onClick={() => setIsCreateAppointmentOpen(true)}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white text-xs font-semibold shadow-sm transition-colors"
          >
            <Truck className="h-3.5 w-3.5" />
            Jadwalkan Armada Baru
          </button>
        </div>
      </div>

      {/* Metrics Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-500">Total Dermaga</span>
            <div className="p-2 rounded-lg bg-slate-100 text-slate-600">
              <Anchor className="h-4 w-4" />
            </div>
          </div>
          <div className="mt-2 text-2xl font-bold text-slate-900">{totalDocks}</div>
          <p className="mt-1 text-[11px] text-slate-400">Total kapasitas dermaga</p>
        </div>

        <div className="rounded-xl border border-emerald-100 bg-white p-4 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-emerald-700">Dermaga Tersedia</span>
            <div className="p-2 rounded-lg bg-emerald-50 text-emerald-600">
              <CheckCircle2 className="h-4 w-4" />
            </div>
          </div>
          <div className="mt-2 text-2xl font-bold text-emerald-600">{availableDocks}</div>
          <p className="mt-1 text-[11px] text-emerald-600/80">Siap menerima armada</p>
        </div>

        <div className="rounded-xl border border-amber-100 bg-white p-4 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-amber-700">Sedang Bongkar</span>
            <div className="p-2 rounded-lg bg-amber-50 text-amber-600">
              <Truck className="h-4 w-4" />
            </div>
          </div>
          <div className="mt-2 text-2xl font-bold text-amber-600">{unloadingDocks}</div>
          <p className="mt-1 text-[11px] text-amber-600/80">Dermaga terisi / proses bongkar</p>
        </div>

        <div className="rounded-xl border border-purple-100 bg-white p-4 shadow-sm">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-purple-700">Antrean Menunggu</span>
            <div className="p-2 rounded-lg bg-purple-50 text-purple-600">
              <Clock className="h-4 w-4" />
            </div>
          </div>
          <div className="mt-2 text-2xl font-bold text-purple-600">{waitingQueue}</div>
          <p className="mt-1 text-[11px] text-purple-600/80">Terjadwal & armada tiba</p>
        </div>
      </div>

      {/* Docks Grid Section */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
        <div className="border-b border-slate-200 p-4 flex items-center justify-between bg-slate-50/50">
          <div>
            <h3 className="text-sm font-bold text-slate-800 flex items-center gap-2">
              <Anchor className="h-4 w-4 text-slate-500" />
              Status Dermaga (Dock Bays)
            </h3>
            <p className="text-xs text-slate-500">Visual pemanfaatan setiap pintu dermaga gudang</p>
          </div>
          <span className="text-xs font-medium px-2.5 py-0.5 rounded-full bg-slate-100 text-slate-600 border border-slate-200">
            {docks.length} Dermaga Terdaftar
          </span>
        </div>

        <div className="p-4">
          {loadingDocks ? (
            <div className="py-8 text-center text-xs text-slate-400">Memuat status dermaga...</div>
          ) : docks.length === 0 ? (
            <div className="py-10 text-center">
              <Anchor className="mx-auto h-8 w-8 text-slate-300 mb-2" />
              <p className="text-xs font-semibold text-slate-700">Belum Ada Dermaga Terdaftar</p>
              <p className="text-xs text-slate-400 mt-0.5">
                Tambahkan dermaga pertama untuk gudang ini untuk mulai menjadwalkan alokasi armada.
              </p>
              <button
                onClick={() => setIsCreateDockOpen(true)}
                className="mt-3 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-emerald-600 text-white text-xs font-semibold shadow-sm hover:bg-emerald-700"
              >
                <Plus className="h-3.5 w-3.5" />
                Tambah Dermaga Sekarang
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
              {docks.map((dock) => {
                const activeAppt = appointments.find(
                  (a) =>
                    a.dock_id === dock.id &&
                    (a.status === "UNLOADING" ||
                      a.status === "ARRIVED" ||
                      a.status === "SCHEDULED")
                )
                const badge = DOCK_STATUS_BADGE[dock.status] || {
                  label: dock.status,
                  cls: "bg-slate-100 text-slate-700 border-slate-200",
                }

                return (
                  <div
                    key={dock.id}
                    className="flex flex-col justify-between rounded-xl border border-slate-200 bg-white p-4 shadow-sm hover:border-slate-300 transition-all"
                  >
                    <div>
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <span className="text-[11px] font-mono font-semibold text-slate-500 uppercase tracking-wide">
                            {dock.dock_code}
                          </span>
                          <h4 className="text-sm font-bold text-slate-900 mt-0.5">
                            {dock.dock_name}
                          </h4>
                        </div>
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold border ${badge.cls}`}
                        >
                          {badge.label}
                        </span>
                      </div>

                      <div className="mt-2 flex items-center gap-2 text-[11px] text-slate-500">
                        <span className="px-1.5 py-0.5 rounded bg-slate-100 text-slate-600 font-medium">
                          {dock.dock_type}
                        </span>
                        <span>•</span>
                        <span>Max {dock.max_tonnage} Ton</span>
                      </div>

                      {/* Active docked appointment info */}
                      {activeAppt ? (
                        <div className="mt-3 rounded-lg border border-slate-200 bg-slate-50/80 p-2.5 text-xs">
                          <div className="flex items-center justify-between text-[11px] font-semibold text-slate-700">
                            <span className="flex items-center gap-1">
                              <Truck className="h-3 w-3 text-slate-500" />
                              {activeAppt.vehicle_plate}
                            </span>
                            <span
                              className={`px-1.5 py-0.2 rounded text-[10px] font-medium border ${
                                APPOINTMENT_STATUS_BADGE[activeAppt.status]?.cls || ""
                              }`}
                            >
                              {activeAppt.status}
                            </span>
                          </div>
                          <p className="mt-1 text-[11px] text-slate-600 truncate font-medium">
                            {activeAppt.vendor_name}
                          </p>
                          <p className="text-[10px] text-slate-500 truncate">
                            Sopir: {activeAppt.driver_name}
                          </p>
                        </div>
                      ) : (
                        <div className="mt-3 py-3 text-center rounded-lg border border-dashed border-slate-200 text-[11px] text-slate-400">
                          Tidak ada armada aktif
                        </div>
                      )}
                    </div>

                    {/* Actions on dock card */}
                    <div className="mt-4 pt-3 border-t border-slate-100 flex items-center justify-between gap-2">
                      <button
                        onClick={() => handleToggleMaintenance(dock)}
                        disabled={updateDockStatusMutation.isPending}
                        className="inline-flex items-center gap-1 text-[11px] font-medium text-slate-600 hover:text-slate-900 p-1 rounded hover:bg-slate-100 transition-colors"
                        title={
                          dock.status === "MAINTENANCE"
                            ? "Selesai Maintenance & Aktifkan"
                            : "Ubah ke Maintenance"
                        }
                      >
                        <Wrench className="h-3 w-3" />
                        {dock.status === "MAINTENANCE" ? "Selesai Maint." : "Maintenance"}
                      </button>

                      {dock.status === "OCCUPIED" && (
                        <button
                          onClick={() => handleReleaseDock(dock)}
                          disabled={updateDockStatusMutation.isPending}
                          className="inline-flex items-center gap-1 text-[11px] font-semibold text-rose-600 hover:text-rose-700 p-1 rounded hover:bg-rose-50 transition-colors"
                        >
                          <RotateCcw className="h-3 w-3" />
                          Lepas Dermaga
                        </button>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>

      {/* Inbound Appointments Table & Schedule Manager */}
      <div className="rounded-xl border border-slate-200 bg-white shadow-sm overflow-hidden">
        <div className="border-b border-slate-200 p-4">
          <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-bold text-slate-800 flex items-center gap-2">
                <Clock className="h-4 w-4 text-slate-500" />
                Jadwal & Antrean Armada (Inbound Queue)
              </h3>
              <p className="text-xs text-slate-500">
                Pencatatan kedatangan armada pemasok, alokasi pintu dermaga, dan progres bongkar koli.
              </p>
            </div>

            <div className="w-full md:w-64">
              <div className="relative">
                <Search className="absolute left-2.5 top-2.5 h-3.5 w-3.5 text-slate-400" />
                <input
                  type="text"
                  placeholder="Cari PO, Plat, Sopir, Vendor..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full pl-8 pr-3 py-1.5 rounded-lg border border-slate-300 text-xs focus:outline-none focus:ring-2 focus:ring-emerald-500/20 focus:border-emerald-500"
                />
              </div>
            </div>
          </div>

          {/* Filter tabs */}
          <div className="mt-4 flex items-center gap-1 overflow-x-auto text-xs border-t border-slate-100 pt-3">
            <button
              onClick={() => setAppointmentFilter("ALL")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                appointmentFilter === "ALL"
                  ? "bg-slate-900 text-white font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Semua
            </button>
            <button
              onClick={() => setAppointmentFilter("SCHEDULED")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                appointmentFilter === "SCHEDULED"
                  ? "bg-slate-900 text-white font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Terjadwal (SCHEDULED)
            </button>
            <button
              onClick={() => setAppointmentFilter("ARRIVED")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                appointmentFilter === "ARRIVED"
                  ? "bg-slate-900 text-white font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Tiba (ARRIVED)
            </button>
            <button
              onClick={() => setAppointmentFilter("UNLOADING")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                appointmentFilter === "UNLOADING"
                  ? "bg-slate-900 text-white font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Bongkar (UNLOADING)
            </button>
            <button
              onClick={() => setAppointmentFilter("COMPLETED")}
              className={`px-3 py-1.5 rounded-lg font-medium transition-colors ${
                appointmentFilter === "COMPLETED"
                  ? "bg-slate-900 text-white font-semibold"
                  : "text-slate-600 hover:bg-slate-100"
              }`}
            >
              Selesai (COMPLETED)
            </button>
          </div>
        </div>

        {/* Table */}
        <div className="overflow-x-auto">
          {loadingAppointments ? (
            <div className="py-12 text-center text-xs text-slate-400">
              Memuat daftar antrean armada...
            </div>
          ) : filteredAppointments.length === 0 ? (
            <div className="py-12 text-center">
              <Truck className="mx-auto h-8 w-8 text-slate-300 mb-2" />
              <p className="text-xs font-semibold text-slate-700">Tidak Ada Antrean Armada</p>
              <p className="text-xs text-slate-400 mt-0.5">
                Tidak ada data antrean armada yang sesuai filter saat ini.
              </p>
            </div>
          ) : (
            <table className="w-full text-left text-xs text-slate-600">
              <thead className="bg-slate-50 text-[11px] font-semibold text-slate-700 uppercase tracking-wider border-b border-slate-200">
                <tr>
                  <th className="px-4 py-3">No. Janji</th>
                  <th className="px-4 py-3">Pemasok & Ref PO</th>
                  <th className="px-4 py-3">Armada & Sopir</th>
                  <th className="px-4 py-3">Estimasi Tiba (ETA)</th>
                  <th className="px-4 py-3">Dermaga Bay</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3 text-right">Aksi</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {filteredAppointments.map((appt) => {
                  const badge = APPOINTMENT_STATUS_BADGE[appt.status] || {
                    label: appt.status,
                    cls: "bg-slate-100 text-slate-700 border-slate-200",
                  }
                  const assignedDock = docks.find((d) => d.id === appt.dock_id)

                  return (
                    <tr key={appt.id} className="hover:bg-slate-50/70 transition-colors">
                      <td className="px-4 py-3 font-mono font-semibold text-slate-900">
                        {appt.appointment_number}
                      </td>
                      <td className="px-4 py-3">
                        <div className="font-semibold text-slate-900">{appt.vendor_name}</div>
                        <div className="text-[11px] text-slate-400">
                          Ref: {appt.po_reference || "-"}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="font-semibold text-slate-800">{appt.vehicle_plate}</div>
                        <div className="text-[11px] text-slate-500">
                          {appt.driver_name} {appt.driver_phone ? `(${appt.driver_phone})` : ""}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="text-slate-700">{formatDate(appt.estimated_arrival)}</div>
                        {appt.actual_arrival && (
                          <div className="text-[11px] text-emerald-600 font-medium">
                            Aktual: {formatDate(appt.actual_arrival)}
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        {assignedDock || appt.dock_name ? (
                          <span className="inline-flex items-center gap-1 font-semibold text-slate-800">
                            <Anchor className="h-3 w-3 text-emerald-600" />
                            {assignedDock?.dock_name || appt.dock_name} (
                            {assignedDock?.dock_code || appt.dock_code})
                          </span>
                        ) : (
                          <span className="italic text-slate-400">Belum dialokasikan</span>
                        )}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold border ${badge.cls}`}
                        >
                          {badge.label}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex items-center justify-end gap-1.5">
                          {/* SCHEDULED actions */}
                          {appt.status === "SCHEDULED" && (
                            <>
                              <button
                                onClick={() =>
                                  handleUpdateAppointmentStatus(appt.id, "ARRIVED")
                                }
                                disabled={updateAppointmentStatusMutation.isPending}
                                className="px-2.5 py-1 rounded bg-blue-50 text-blue-700 hover:bg-blue-100 font-semibold text-[11px] transition-colors"
                              >
                                Tiba
                              </button>
                              <button
                                onClick={() => handleOpenAssignModal(appt)}
                                className="px-2.5 py-1 rounded bg-slate-100 text-slate-700 hover:bg-slate-200 font-semibold text-[11px] transition-colors"
                              >
                                Alokasikan Dock
                              </button>
                            </>
                          )}

                          {/* ARRIVED actions */}
                          {appt.status === "ARRIVED" && (
                            <>
                              {appt.dock_id ? (
                                <button
                                  onClick={() =>
                                    handleUpdateAppointmentStatus(appt.id, "UNLOADING")
                                  }
                                  disabled={updateAppointmentStatusMutation.isPending}
                                  className="px-2.5 py-1 rounded bg-emerald-600 text-white hover:bg-emerald-700 font-semibold text-[11px] transition-colors shadow-sm"
                                >
                                  Mulai Bongkar
                                </button>
                              ) : (
                                <button
                                  onClick={() => handleOpenAssignModal(appt)}
                                  className="px-2.5 py-1 rounded bg-amber-50 text-amber-800 hover:bg-amber-100 font-semibold text-[11px] transition-colors"
                                >
                                  Alokasikan Dock
                                </button>
                              )}
                              {appt.dock_id && (
                                <button
                                  onClick={() => handleOpenAssignModal(appt)}
                                  className="px-2 py-1 rounded bg-slate-100 text-slate-600 hover:bg-slate-200 font-medium text-[11px] transition-colors"
                                  title="Ganti Dermaga"
                                >
                                  Ganti Dock
                                </button>
                              )}
                            </>
                          )}

                          {/* UNLOADING actions */}
                          {appt.status === "UNLOADING" && (
                            <button
                              onClick={() =>
                                handleUpdateAppointmentStatus(appt.id, "COMPLETED")
                              }
                              disabled={updateAppointmentStatusMutation.isPending}
                              className="px-2.5 py-1 rounded bg-emerald-600 text-white hover:bg-emerald-700 font-semibold text-[11px] transition-colors shadow-sm"
                            >
                              Selesai
                            </button>
                          )}

                          {/* COMPLETED badge */}
                          {appt.status === "COMPLETED" && (
                            <span className="text-[11px] text-emerald-600 font-medium flex items-center gap-1">
                              <Check className="h-3.5 w-3.5" /> Selesai
                            </span>
                          )}
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          )}
        </div>
      </div>

      {/* Modal: Tambah Dermaga */}
      {isCreateDockOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
          <div className="w-full max-w-md rounded-2xl bg-white p-5 shadow-2xl border border-slate-100">
            <div className="flex items-center justify-between border-b border-slate-100 pb-3">
              <h3 className="text-sm font-bold text-slate-800 flex items-center gap-2">
                <Anchor className="h-4 w-4 text-emerald-600" />
                Tambah Dermaga Baru
              </h3>
              <button
                onClick={() => setIsCreateDockOpen(false)}
                className="text-slate-400 hover:text-slate-600"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleCreateDockSubmit} className="mt-4 space-y-3.5 text-xs">
              <div>
                <label className="block font-medium text-slate-700 mb-1">
                  Kode Dermaga <span className="text-rose-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="Mis: DOCK-01"
                  value={newDockCode}
                  onChange={(e) => setNewDockCode(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                />
              </div>

              <div>
                <label className="block font-medium text-slate-700 mb-1">
                  Nama Dermaga <span className="text-rose-500">*</span>
                </label>
                <input
                  type="text"
                  required
                  placeholder="Mis: Dermaga Inbound Barat 1"
                  value={newDockName}
                  onChange={(e) => setNewDockName(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                />
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block font-medium text-slate-700 mb-1">Tipe Dermaga</label>
                  <select
                    value={newDockType}
                    onChange={(e) => setNewDockType(e.target.value as DockType)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  >
                    <option value="INBOUND">INBOUND</option>
                    <option value="OUTBOUND">OUTBOUND</option>
                    <option value="CROSS_DOCK">CROSS_DOCK</option>
                  </select>
                </div>

                <div>
                  <label className="block font-medium text-slate-700 mb-1">
                    Kapasitas (Tonase)
                  </label>
                  <input
                    type="number"
                    min="1"
                    step="0.5"
                    value={newMaxTonnage}
                    onChange={(e) => setNewMaxTonnage(Number(e.target.value))}
                    className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>
              </div>

              <div>
                <label className="block font-medium text-slate-700 mb-1">Catatan</label>
                <textarea
                  rows={2}
                  placeholder="Catatan operasional dermaga..."
                  value={newDockNotes}
                  onChange={(e) => setNewDockNotes(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                />
              </div>

              <div className="mt-5 flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setIsCreateDockOpen(false)}
                  className="px-3 py-1.5 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50 font-medium"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createDockMutation.isPending}
                  className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-semibold shadow-sm transition-colors"
                >
                  {createDockMutation.isPending ? "Menyimpan..." : "Simpan Dermaga"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Modal: Jadwalkan Armada Baru */}
      {isCreateAppointmentOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
          <div className="w-full max-w-lg rounded-2xl bg-white p-5 shadow-2xl border border-slate-100">
            <div className="flex items-center justify-between border-b border-slate-100 pb-3">
              <h3 className="text-sm font-bold text-slate-800 flex items-center gap-2">
                <Truck className="h-4 w-4 text-emerald-600" />
                Jadwalkan Armada Pemasok Baru
              </h3>
              <button
                onClick={() => setIsCreateAppointmentOpen(false)}
                className="text-slate-400 hover:text-slate-600"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <form onSubmit={handleCreateAppointmentSubmit} className="mt-4 space-y-3.5 text-xs">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block font-medium text-slate-700 mb-1">
                    Nama Pemasok / Vendor <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="Mis: PT Pangan Sentosa"
                    value={vendorName}
                    onChange={(e) => setVendorName(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>

                <div>
                  <label className="block font-medium text-slate-700 mb-1">
                    No. Plat Kendaraan <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="Mis: B 9123 UXZ"
                    value={vehiclePlate}
                    onChange={(e) => setVehiclePlate(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block font-medium text-slate-700 mb-1">
                    Nama Sopir <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="text"
                    required
                    placeholder="Nama sopir"
                    value={driverName}
                    onChange={(e) => setDriverName(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>

                <div>
                  <label className="block font-medium text-slate-700 mb-1">No. Telp Sopir</label>
                  <input
                    type="text"
                    placeholder="Mis: 08123456789"
                    value={driverPhone}
                    onChange={(e) => setDriverPhone(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block font-medium text-slate-700 mb-1">Ref PO / Dokumen</label>
                  <input
                    type="text"
                    placeholder="Mis: PO-202610-091"
                    value={poReference}
                    onChange={(e) => setPoReference(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>

                <div>
                  <label className="block font-medium text-slate-700 mb-1">
                    Estimasi Tiba (ETA) <span className="text-rose-500">*</span>
                  </label>
                  <input
                    type="datetime-local"
                    required
                    value={estimatedArrival}
                    onChange={(e) => setEstimatedArrival(e.target.value)}
                    className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                  />
                </div>
              </div>

              <div>
                <label className="block font-medium text-slate-700 mb-1">
                  Pilih Dermaga Awal (Opsional)
                </label>
                <select
                  value={initialDockId}
                  onChange={(e) => setInitialDockId(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                >
                  <option value="">-- Alokasikan Nanti saat Tiba --</option>
                  {docks.map((d) => (
                    <option key={d.id} value={d.id} disabled={d.status !== "AVAILABLE"}>
                      {d.dock_name} ({d.dock_code}) - {d.status}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="block font-medium text-slate-700 mb-1">Catatan</label>
                <textarea
                  rows={2}
                  placeholder="Catatan barang muatan atau instruksi bongkar..."
                  value={appointmentNotes}
                  onChange={(e) => setAppointmentNotes(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-1.5 focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                />
              </div>

              <div className="mt-5 flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setIsCreateAppointmentOpen(false)}
                  className="px-3 py-1.5 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50 font-medium"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={createAppointmentMutation.isPending}
                  className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-semibold shadow-sm transition-colors"
                >
                  {createAppointmentMutation.isPending ? "Menyimpan..." : "Buat Jadwal Armada"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Modal: Alokasikan Dermaga */}
      {assigningAppointment && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 backdrop-blur-sm p-4">
          <div className="w-full max-w-md rounded-2xl bg-white p-5 shadow-2xl border border-slate-100">
            <div className="flex items-center justify-between border-b border-slate-100 pb-3">
              <h3 className="text-sm font-bold text-slate-800 flex items-center gap-2">
                <Anchor className="h-4 w-4 text-emerald-600" />
                Alokasikan Dermaga Bongkar
              </h3>
              <button
                onClick={() => setAssigningAppointment(null)}
                className="text-slate-400 hover:text-slate-600"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="my-3 rounded-lg bg-slate-50 p-3 text-xs space-y-1">
              <div className="font-semibold text-slate-900">
                {assigningAppointment.appointment_number} - {assigningAppointment.vendor_name}
              </div>
              <div className="text-slate-500">
                Armada: <span className="font-medium text-slate-700">{assigningAppointment.vehicle_plate}</span> | Sopir: {assigningAppointment.driver_name}
              </div>
            </div>

            <form onSubmit={handleConfirmAssignDock} className="space-y-4 text-xs">
              <div>
                <label className="block font-medium text-slate-700 mb-1">
                  Pilih Dermaga <span className="text-rose-500">*</span>
                </label>
                <select
                  required
                  value={selectedDockIdToAssign}
                  onChange={(e) => setSelectedDockIdToAssign(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 px-3 py-2 text-xs focus:border-emerald-500 focus:outline-none focus:ring-2 focus:ring-emerald-500/20"
                >
                  <option value="">-- Pilih Dermaga Tersedia --</option>
                  {docks.map((d) => (
                    <option
                      key={d.id}
                      value={d.id}
                      disabled={d.status !== "AVAILABLE" && d.id !== assigningAppointment.dock_id}
                    >
                      {d.dock_name} ({d.dock_code}) - {d.status} (Max {d.max_tonnage} Ton)
                    </option>
                  ))}
                </select>
              </div>

              <div className="flex items-center justify-end gap-2 pt-2 border-t border-slate-100">
                <button
                  type="button"
                  onClick={() => setAssigningAppointment(null)}
                  className="px-3 py-1.5 rounded-lg border border-slate-300 text-slate-600 hover:bg-slate-50 font-medium"
                >
                  Batal
                </button>
                <button
                  type="submit"
                  disabled={!selectedDockIdToAssign || assignDockMutation.isPending}
                  className="px-3.5 py-1.5 rounded-lg bg-emerald-600 hover:bg-emerald-700 text-white font-semibold shadow-sm transition-colors"
                >
                  {assignDockMutation.isPending ? "Mengalokasikan..." : "Konfirmasi Alokasi"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
