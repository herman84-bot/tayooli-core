"use client"

import { useQuery } from "@tanstack/react-query"
import { api, type ExpiryReport } from "@/lib/api"

/** Batch expiry control report (GET /wms/expiry-report). */
export function useExpiryReport(days: number, warehouseId?: string) {
  return useQuery<ExpiryReport | null>({
    queryKey: ["wms-expiry-report", days, warehouseId ?? null],
    queryFn: async () => {
      const res = await api.wms.expiry.report(days, warehouseId)
      return res.data ?? null
    },
  })
}
