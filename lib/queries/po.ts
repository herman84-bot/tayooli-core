import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '@/lib/api/client'
import { POSchema, type PO, type CreatePOInput } from '@/lib/schemas/po'
import { z } from 'zod'

const PO_KEY = ['purchase-orders'] as const

export interface POList {
  data: PO[]
  total: number
  page: number
  per_page: number
}

interface UsePOsParams {
  page?: number
  perPage?: number
}

// ── List purchase orders (paginated) ────────────────────────────────────────
export function usePOs({ page = 1, perPage = 20 }: UsePOsParams = {}) {
  return useQuery({
    queryKey: [...PO_KEY, { page, perPage }] as const,
    queryFn: async (): Promise<POList> => {
      const res = await apiClient.get<unknown>('/api/v1/purchase-orders', {
        params: { page, per_page: perPage },
      })
      // Backend returns { data, total, page, per_page }; tolerate bare array too.
      const payload = res.data as unknown
      const arr = Array.isArray(payload) ? payload : (payload as { data?: unknown[] })?.data ?? []
      const meta = Array.isArray(payload) ? {} : (payload as { total?: number; page?: number; per_page?: number })
      return {
        data: z.array(POSchema).parse(arr),
        total: meta.total ?? arr.length,
        page: meta.page ?? page,
        per_page: meta.per_page ?? perPage,
      }
    },
    retry: 1,
  })
}

// ── Single purchase order ────────────────────────────────────────────────────
export function usePO(id: string) {
  return useQuery({
    queryKey: [...PO_KEY, id] as const,
    queryFn: async (): Promise<PO> => {
      const res = await apiClient.get<unknown>(`/api/v1/purchase-orders/${id}`)
      return POSchema.parse(res.data)
    },
    enabled: Boolean(id),
    retry: 1,
  })
}

// ── Create mutation ──────────────────────────────────────────────────────────
export function useCreatePO() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: CreatePOInput): Promise<PO> => {
      const res = await apiClient.post<unknown>('/api/v1/purchase-orders', data)
      return POSchema.parse(res.data)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: PO_KEY })
    },
  })
}
