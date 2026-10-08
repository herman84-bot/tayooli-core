"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  api,
  type CreateShippingManifestInput,
  type DispatchShippingManifestInput,
  type ShippingManifest,
  type ShippingManifestDetail,
  type WMSOutboundKPISummary,
} from "@/lib/api"

/**
 * Fetch shipping manifests filtered by warehouse, status, or expedition.
 */
export function useShippingManifests(params?: {
  warehouse_id?: string
  status?: string
  expedition_name?: string
}) {
  return useQuery<ShippingManifest[]>({
    queryKey: ["shipping-manifests", params],
    queryFn: async () => {
      const res = await api.wms.manifests.list(params)
      return res.data ?? []
    },
  })
}

/**
 * Fetch detailed shipping manifest with line items.
 */
export function useShippingManifestDetail(id?: string) {
  return useQuery<ShippingManifestDetail | null>({
    queryKey: ["shipping-manifest", id],
    queryFn: async () => {
      if (!id) return null
      const res = await api.wms.manifests.get(id)
      return res.data ?? null
    },
    enabled: !!id,
  })
}

/**
 * Mutation to create a new shipping manifest.
 * Invalidates ["shipping-manifests"] and ["delivery-orders"].
 */
export function useCreateShippingManifest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (payload: CreateShippingManifestInput) => api.wms.manifests.create(payload),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["shipping-manifests"] })
      qc.invalidateQueries({ queryKey: ["delivery-orders"] })
    },
  })
}

/**
 * Mutation to scan a delivery order onto the shipping manifest during loading.
 * Invalidates ["shipping-manifest", id] and ["shipping-manifests"].
 */
export function useScanLoadingDO() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, barcode }: { id: string; barcode: string }) =>
      api.wms.manifests.scanLoading(id, barcode),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ["shipping-manifest", variables.id] })
      qc.invalidateQueries({ queryKey: ["shipping-manifests"] })
    },
  })
}

export const useLoadingScanDO = useScanLoadingDO

/**
 * Mutation to dispatch a shipping manifest with driver digital signature.
 * Invalidates ["shipping-manifests"], ["delivery-orders"], ["wms-stock"], and ["wms-outbound-kpi"].
 */
export function useDispatchShippingManifest() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (variables: {
      id: string
      payload?: DispatchShippingManifestInput
      driver_signature_svg?: string
      notes?: string
    }) => {
      const payload: DispatchShippingManifestInput = variables.payload ?? {
        driver_signature_svg: variables.driver_signature_svg ?? "",
        notes: variables.notes,
      }
      return api.wms.manifests.dispatch(variables.id, payload)
    },
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ["shipping-manifest", variables.id] })
      qc.invalidateQueries({ queryKey: ["shipping-manifests"] })
      qc.invalidateQueries({ queryKey: ["delivery-orders"] })
      qc.invalidateQueries({ queryKey: ["wms-stock"] })
      qc.invalidateQueries({ queryKey: ["wms-outbound-kpi"] })
    },
  })
}

/**
 * Fetch WMS outbound KPI summary metrics.
 */
export function useWMSOutboundKPI(warehouseId?: string) {
  return useQuery<WMSOutboundKPISummary | null>({
    queryKey: ["wms-outbound-kpi", warehouseId],
    queryFn: async () => {
      const res = await api.wms.manifests.kpis(warehouseId)
      return res.data ?? null
    },
  })
}

export const useOutboundKPI = useWMSOutboundKPI
