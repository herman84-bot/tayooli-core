import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '@/lib/api/client'
import { InvoiceSchema, InvoiceListSchema, type Invoice, type InvoiceList, type CreateInvoiceInput } from '@/lib/schemas/invoice'
import { z } from 'zod'

const INVOICES_KEY = ['invoices'] as const

interface UseInvoicesParams {
  page?: number
  perPage?: number
}

// ── List all invoices (paginated) ───────────────────────────────────────────
export function useInvoices({ page = 1, perPage = 20 }: UseInvoicesParams = {}) {
  return useQuery({
    queryKey: [...INVOICES_KEY, { page, perPage }] as const,
    queryFn: async (): Promise<InvoiceList> => {
      const res = await apiClient.get<unknown>('/api/v1/invoices', {
        params: { page, per_page: perPage },
      })
      // API returns { data, total, page, per_page } envelope
      return InvoiceListSchema.parse(res.data)
    },
    retry: 1,
  })
}

// ── Single invoice ───────────────────────────────────────────────────────────
export function useInvoice(id: string) {
  return useQuery({
    queryKey: [...INVOICES_KEY, id] as const,
    queryFn: async (): Promise<Invoice> => {
      const res = await apiClient.get<unknown>(`/api/v1/invoices/${id}`)
      return InvoiceSchema.parse(res.data)
    },
    enabled: Boolean(id),
    retry: 1,
  })
}

// ── Approve mutation ─────────────────────────────────────────────────────────
export function useApproveInvoice() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string): Promise<Invoice> => {
      const res = await apiClient.post<unknown>(`/api/v1/invoices/${id}/approve`)
      return InvoiceSchema.parse(res.data)
    },
    onSuccess: (_data, id) => {
      // Invalidate list and the specific invoice cache
      void queryClient.invalidateQueries({ queryKey: INVOICES_KEY })
      void queryClient.invalidateQueries({ queryKey: [...INVOICES_KEY, id] })
    },
  })
}

// ── Reject mutation ──────────────────────────────────────────────────────────
export function useRejectInvoice() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string): Promise<Invoice> => {
      const res = await apiClient.post<unknown>(`/api/v1/invoices/${id}/reject`)
      return InvoiceSchema.parse(res.data)
    },
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({ queryKey: INVOICES_KEY })
      void queryClient.invalidateQueries({ queryKey: [...INVOICES_KEY, id] })
    },
  })
}

// ── Create mutation ──────────────────────────────────────────────────────────
export function useCreateInvoice() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreateInvoiceInput): Promise<Invoice> => {
      const res = await apiClient.post<unknown>('/api/v1/invoices', data)
      return InvoiceSchema.parse(res.data)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: INVOICES_KEY })
    },
  })
}
