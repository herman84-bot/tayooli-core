import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '@/lib/api/client'
import { GRSchema, type GR, type CreateGRInput } from '@/lib/schemas/gr'
import { z } from 'zod'

const GR_KEY = ['goods-receipts'] as const

export interface GRList {
  data: GR[]
  total: number
  page: number
  per_page: number
}

interface UseGRsParams {
  page?: number
  perPage?: number
}

// ── List goods receipts (paginated) ─────────────────────────────────────────
export function useGRs({ page = 1, perPage = 20 }: UseGRsParams = {}) {
  return useQuery({
    queryKey: [...GR_KEY, { page, perPage }] as const,
    queryFn: async (): Promise<GRList> => {
      const res = await apiClient.get<unknown>('/api/v1/goods-receipts', {
        params: { page, per_page: perPage },
      })
      // Backend returns { data, total, page, per_page }; tolerate bare array too.
      const payload = res.data as unknown
      const arr = Array.isArray(payload) ? payload : (payload as { data?: unknown[] })?.data ?? []
      const meta = Array.isArray(payload) ? {} : (payload as { total?: number; page?: number; per_page?: number })
      return {
        data: z.array(GRSchema).parse(arr),
        total: meta.total ?? arr.length,
        page: meta.page ?? page,
        per_page: meta.per_page ?? perPage,
      }
    },
    retry: 1,
  })
}

// ── Single goods receipt ─────────────────────────────────────────────────────
export function useGR(id: string) {
  return useQuery({
    queryKey: [...GR_KEY, id] as const,
    queryFn: async (): Promise<GR> => {
      const res = await apiClient.get<unknown>(`/api/v1/goods-receipts/${id}`)
      return GRSchema.parse(res.data)
    },
    enabled: Boolean(id),
    retry: 1,
  })
}

// ── Create mutation ──────────────────────────────────────────────────────────
export function useCreateGR() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: CreateGRInput): Promise<GR> => {
      const res = await apiClient.post<unknown>('/api/v1/goods-receipts', data)
      return GRSchema.parse(res.data)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: GR_KEY })
    },
  })
}
