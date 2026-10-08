"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  api,
  type AddLPNItemInput,
  type AppointmentStatus,
  type CreateAppointmentInput,
  type CreateDockInput,
  type CreateLPNInput,
  type DockAppointment,
  type DockStatus,
  type InboundDock,
  type MoveLPNInput,
  type StockLPN,
  type StockLPNDetail,
  type UpdateAppointmentStatusInput,
  type UpdateDockStatusInput,
} from "@/lib/api"

/**
 * Fetch inbound docks for a warehouse, optionally filtered by dock status.
 */
export function useInboundDocks(warehouseId?: string | null, status?: string) {
  return useQuery<InboundDock[]>({
    queryKey: ["inbound-docks", warehouseId, status],
    queryFn: async () => {
      if (!warehouseId) return []
      const res = await api.wms.docks.list(warehouseId, status)
      return res.data ?? []
    },
    enabled: !!warehouseId,
  })
}

/**
 * Mutation to create a new dock bay.
 * Invalidates ["inbound-docks"].
 */
export function useCreateDock() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateDockInput) => api.wms.docks.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["inbound-docks"] })
    },
  })
}

/**
 * Mutation to update dock bay operational status.
 * Invalidates ["inbound-docks"] and ["dock-appointments"].
 */
export function useUpdateDockStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars:
      | { id: string; data: UpdateDockStatusInput }
      | { id: string; status: DockStatus; notes?: string }
    ) => {
      const data: UpdateDockStatusInput =
        "status" in vars ? { status: vars.status, notes: vars.notes } : vars.data
      return api.wms.docks.updateStatus(vars.id, data)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["inbound-docks"] })
      qc.invalidateQueries({ queryKey: ["dock-appointments"] })
    },
  })
}

/**
 * Fetch dock appointments for a warehouse, optionally filtered by status.
 */
export function useDockAppointments(warehouseId?: string | null, status?: string) {
  return useQuery<DockAppointment[]>({
    queryKey: ["dock-appointments", warehouseId, status],
    queryFn: async () => {
      if (!warehouseId) return []
      const res = await api.wms.dockAppointments.list(warehouseId, status)
      return res.data ?? []
    },
    enabled: !!warehouseId,
  })
}

/**
 * Mutation to schedule a new dock appointment.
 * Invalidates ["dock-appointments"].
 */
export function useCreateAppointment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateAppointmentInput) => api.wms.dockAppointments.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dock-appointments"] })
    },
  })
}

/**
 * Mutation to assign a dock bay to an appointment.
 * Invalidates ["dock-appointments"] and ["inbound-docks"].
 */
export function useAssignDock() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars:
      | { id: string; dockId: string }
      | { appointmentId: string; dockId: string }
      | { id: string; dock_id: string }
      | { appointmentId: string; dock_id: string }
    ) => {
      const id = "appointmentId" in vars ? vars.appointmentId : vars.id
      const dockId = "dockId" in vars ? vars.dockId : vars.dock_id
      return api.wms.dockAppointments.assign(id, dockId)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dock-appointments"] })
      qc.invalidateQueries({ queryKey: ["inbound-docks"] })
    },
  })
}

/**
 * Mutation to advance dock appointment lifecycle status.
 * Invalidates ["dock-appointments"] and ["inbound-docks"].
 */
export function useUpdateAppointmentStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars:
      | { id: string; data: UpdateAppointmentStatusInput }
      | { id: string; status: AppointmentStatus; notes?: string }
      | { appointmentId: string; status: AppointmentStatus; notes?: string }
      | { appointmentId: string; data: UpdateAppointmentStatusInput }
    ) => {
      const id = "appointmentId" in vars ? vars.appointmentId : vars.id
      const data: UpdateAppointmentStatusInput =
        "status" in vars ? { status: vars.status, notes: vars.notes } : vars.data
      return api.wms.dockAppointments.updateStatus(id, data)
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["dock-appointments"] })
      qc.invalidateQueries({ queryKey: ["inbound-docks"] })
    },
  })
}

/**
 * Fetch stock LPNs (License Plate Numbers) for a warehouse.
 */
export function useStockLPNs(warehouseId?: string | null, status?: string) {
  return useQuery<StockLPN[]>({
    queryKey: ["stock-lpns", warehouseId, status],
    queryFn: async () => {
      if (!warehouseId) return []
      const res = await api.wms.lpns.list(warehouseId, status)
      return res.data ?? []
    },
    enabled: !!warehouseId,
  })
}

/**
 * Fetch LPN details with packaged line items.
 */
export function useStockLPNDetail(id?: string | null) {
  return useQuery<StockLPNDetail | null>({
    queryKey: ["stock-lpn", id],
    queryFn: async () => {
      if (!id) return null
      const res = await api.wms.lpns.get(id)
      return res.data ?? null
    },
    enabled: !!id,
  })
}

/**
 * Mutation to create a new pallet LPN container.
 * Invalidates ["stock-lpns"].
 */
export function useCreateLPN() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateLPNInput) => api.wms.lpns.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["stock-lpns"] })
    },
  })
}

/**
 * Mutation to pack a product batch quantity into an LPN container.
 * Invalidates ["stock-lpn", id] and ["stock-lpns"].
 */
export function useAddLPNItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars:
      | { id: string; data: AddLPNItemInput }
      | { lpnId: string; data: AddLPNItemInput }
      | { id: string; product_id: string; batch_id: string; quantity: number | string }
      | { lpnId: string; product_id: string; batch_id: string; quantity: number | string }
    ) => {
      const id = "lpnId" in vars ? vars.lpnId : vars.id
      const data: AddLPNItemInput =
        "product_id" in vars
          ? { product_id: vars.product_id, batch_id: vars.batch_id, quantity: vars.quantity }
          : vars.data
      return api.wms.lpns.addItem(id, data)
    },
    onSuccess: (_data, vars) => {
      const id = "lpnId" in vars ? vars.lpnId : vars.id
      if (id) {
        qc.invalidateQueries({ queryKey: ["stock-lpn", id] })
      }
      qc.invalidateQueries({ queryKey: ["stock-lpns"] })
    },
  })
}

/**
 * Mutation to move an LPN container atomically to target rack location.
 * Invalidates ["stock-lpn", id], ["stock-lpns"], ["wms-stock"], ["wms-locations"].
 */
export function useMoveLPN() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (vars:
      | { id: string; data: MoveLPNInput }
      | { lpnId: string; data: MoveLPNInput }
      | { id: string; target_location_id: string; notes?: string }
      | { lpnId: string; target_location_id: string; notes?: string }
    ) => {
      const id = "lpnId" in vars ? vars.lpnId : vars.id
      const data: MoveLPNInput =
        "target_location_id" in vars
          ? { target_location_id: vars.target_location_id, notes: vars.notes }
          : vars.data
      return api.wms.lpns.move(id, data)
    },
    onSuccess: (_data, vars) => {
      const id = "lpnId" in vars ? vars.lpnId : vars.id
      if (id) {
        qc.invalidateQueries({ queryKey: ["stock-lpn", id] })
      }
      qc.invalidateQueries({ queryKey: ["stock-lpns"] })
      qc.invalidateQueries({ queryKey: ["wms-stock"] })
      qc.invalidateQueries({ queryKey: ["wms-locations"] })
      qc.invalidateQueries({ queryKey: ["wms", "stock"] })
      qc.invalidateQueries({ queryKey: ["wms", "locations"] })
    },
  })
}
