"use client"

import { useQuery } from "@tanstack/react-query"
import { api, type StockMovement, type StockSummary } from "@/lib/api"

export function useWMSMovements(params?: { product_id?: string; location_id?: string; limit?: number }) {
  return useQuery<StockMovement[]>({
    queryKey: ["wms-movements", params],
    queryFn: async (): Promise<StockMovement[]> => {
      const res = await api.wms.movements.list(params)
      return res.data ?? []
    },
  })
}

export function useWMSStock(warehouseId?: string) {
  return useQuery<StockSummary[]>({
    queryKey: ["wms-stock", warehouseId],
    queryFn: async (): Promise<StockSummary[]> => {
      const res = await api.wms.stock.list(warehouseId)
      return res.data ?? []
    },
  })
}
