"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api, type Product } from "@/lib/api"

export function useProducts() {
  return useQuery<Product[]>({
    queryKey: ["products"],
    queryFn: async (): Promise<Product[]> => {
      const res = await api.products.list()
      if (Array.isArray(res)) return res as Product[]
      if (res && Array.isArray((res as any).data)) return (res as any).data as Product[]
      return []
    },
    retry: 1,
  })
}

export function useProductInventory(productId: string | null) {
  return useQuery({
    queryKey: ["inventory", productId],
    queryFn: () => api.products.inventory(productId as string),
    enabled: !!productId,
  })
}

export function useCreateProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.products.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: ["products"] }),
  })
}

export function useUpdateProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: Parameters<typeof api.products.update>[1] }) =>
      api.products.update(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["products"] }),
  })
}

export function useDeleteProduct() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.products.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["products"] }),
  })
}

export function useAdjustInventory(productId: string | null) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: { inventoryId: string; delta: number }) =>
      api.products.adjustInventory(productId as string, input.inventoryId, input.delta),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["inventory", productId] }),
  })
}
