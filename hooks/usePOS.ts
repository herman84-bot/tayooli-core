"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api, type POSOrder } from "@/lib/api"

export function usePOSOrders(limit = 50) {
  return useQuery<POSOrder[]>({
    queryKey: ["pos-orders", limit],
    queryFn: async (): Promise<POSOrder[]> => {
      const res = await api.pos.orders(limit)
      return res.data ?? []
    },
  })
}

export function usePOSCheckout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.pos.checkout,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["pos-orders"] })
      qc.invalidateQueries({ queryKey: ["products"] })
      qc.invalidateQueries({ queryKey: ["wms-movements"] })
      qc.invalidateQueries({ queryKey: ["wms-stock"] })
    },
  })
}
