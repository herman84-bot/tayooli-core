import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '@/lib/api/client'
import {
  VendorSchema,
  VendorListSchema,
  VendorRatingSchema,
  type Vendor,
  type VendorList,
  type VendorRating,
  type CreateVendorInput,
  type UpdateVendorInput,
  type RateVendorInput,
} from '@/lib/schemas/vendor'

const VENDORS_KEY = ['vendors'] as const

interface UseVendorsParams {
  page?: number
  perPage?: number
  search?: string
}

// ── List all vendors (paginated + search) ───────────────────────────────────
export function useVendors({ page = 1, perPage = 20, search }: UseVendorsParams = {}) {
  return useQuery({
    queryKey: [...VENDORS_KEY, { page, perPage, search }] as const,
    queryFn: async (): Promise<VendorList> => {
      const params: Record<string, string | number> = { page, per_page: perPage }
      if (search) {
        params.q = search
      }
      const res = await apiClient.get<unknown>('/api/v1/vendors', { params })
      return VendorListSchema.parse(res.data)
    },
    retry: 1,
  })
}

// ── Single vendor ───────────────────────────────────────────────────────────
export function useVendor(id: string) {
  return useQuery({
    queryKey: [...VENDORS_KEY, id] as const,
    queryFn: async (): Promise<Vendor> => {
      const res = await apiClient.get<unknown>(`/api/v1/vendors/${id}`)
      return VendorSchema.parse(res.data)
    },
    enabled: Boolean(id),
    retry: 1,
  })
}

// ── Create mutation ─────────────────────────────────────────────────────────
export function useCreateVendor() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreateVendorInput): Promise<Vendor> => {
      const res = await apiClient.post<unknown>('/api/v1/vendors', data)
      return VendorSchema.parse(res.data)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: VENDORS_KEY })
    },
  })
}

// ── Update mutation ─────────────────────────────────────────────────────────
export function useUpdateVendor() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: UpdateVendorInput }): Promise<Vendor> => {
      const res = await apiClient.put<unknown>(`/api/v1/vendors/${id}`, data)
      return VendorSchema.parse(res.data)
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: VENDORS_KEY })
      void queryClient.invalidateQueries({ queryKey: [...VENDORS_KEY, variables.id] })
    },
  })
}

// ── Delete mutation ─────────────────────────────────────────────────────────
export function useDeleteVendor() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string): Promise<void> => {
      await apiClient.delete(`/api/v1/vendors/${id}`)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: VENDORS_KEY })
    },
  })
}

// ── Rate mutation ───────────────────────────────────────────────────────────
export function useRateVendor() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ id, data }: { id: string; data: RateVendorInput }): Promise<VendorRating> => {
      const res = await apiClient.post<unknown>(`/api/v1/vendors/${id}/rate`, data)
      return VendorRatingSchema.parse(res.data)
    },
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: VENDORS_KEY })
      void queryClient.invalidateQueries({ queryKey: [...VENDORS_KEY, variables.id] })
    },
  })
}
