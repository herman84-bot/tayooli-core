"use client"

import { useMutation, useQuery } from "@tanstack/react-query"
import { api } from "@/lib/api"

/** Triggers AI analysis (anomaly detection + GL suggestion) for an invoice. */
export function useTriggerInference() {
  return useMutation({
    mutationFn: api.inference.ingest,
  })
}

/**
 * Polls the inference status for an invoice until the job reaches a terminal
 * state ("completed" | "failed").
 */
export function useInferenceStatus(invoiceId: string | null) {
  return useQuery({
    queryKey: ["inference", invoiceId],
    queryFn: () => api.inference.status(invoiceId as string),
    enabled: !!invoiceId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === "completed" || status === "failed" ? false : 2000
    },
  })
}
