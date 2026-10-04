"use client"

import { useQuery } from "@tanstack/react-query"
import { api } from "@/lib/api"

export function useInvoices({ page = 1, perPage = 20 }: { page?: number; perPage?: number } = {}) {
  return useQuery({
    queryKey: ["invoices", { page, perPage }],
    queryFn: async () => {
      const res = await api.invoices.list({ page, per_page: perPage })
      return res
    },
    retry: 1,
  })
}

export function useInvoice(id: string) {

  return useQuery({ queryKey: ["invoices", id], queryFn: () => api.invoices.get(id), enabled: Boolean(id), retry: 1 })
}
