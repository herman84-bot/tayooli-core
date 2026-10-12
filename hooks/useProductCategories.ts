"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api, type ProductCategory } from "@/lib/api"

export function useProductCategories() {
  return useQuery<ProductCategory[]>({
    queryKey: ["product-categories"],
    queryFn: async (): Promise<ProductCategory[]> => {
      const res = await api.productCategories.list()
      if (Array.isArray(res)) return res
      if (res && Array.isArray((res as any).data)) return (res as any).data
      return []
    },
    staleTime: 60_000,
  })
}

export function useCreateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.productCategories.create({ name }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["product-categories"] })
    },
  })
}

export function useRenameCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) =>
      api.productCategories.rename(id, name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["product-categories"] })
      qc.invalidateQueries({ queryKey: ["products"] })
    },
  })
}

export function useDeleteCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.productCategories.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["product-categories"] })
      qc.invalidateQueries({ queryKey: ["products"] })
    },
  })
}
