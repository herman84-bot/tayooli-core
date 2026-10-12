"use client"

import { useEffect, useState } from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api, type PaginatedResponse, type Product, type ProductSearchParams } from "@/lib/api"

/**
 * useProducts loads the default un-paged product list (capped at 1000 by backend).
 * Maintained for existing picker/lookup callers that operate on the full list.
 */
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

/**
 * useProductsPaged powers server-paginated tables and views.
 * placeholderData keeps the previous page visible while the next loads.
 */
export function useProductsPaged(params: ProductSearchParams) {
  const page = params.page ?? 1
  const perPage = params.per_page ?? 20
  const q = params.q?.trim() ?? ""
  const category = params.category?.trim() ?? ""

  return useQuery<PaginatedResponse<Product>>({
    queryKey: ["products", "paged", { page, perPage, q, category }],
    queryFn: async (): Promise<PaginatedResponse<Product>> => {
      const res = await api.products.list({
        page,
        per_page: perPage,
        q: q || undefined,
        category: category || undefined,
      })
      if (res && Array.isArray(res.data)) {
        return {
          data: res.data,
          total: typeof res.total === "number" ? res.total : res.data.length,
          page: typeof res.page === "number" ? res.page : page,
          per_page: typeof res.per_page === "number" ? res.per_page : perPage,
        }
      }
      return { data: [], total: 0, page, per_page: perPage }
    },
    placeholderData: (prev) => prev,
  })
}

const PRODUCT_SEARCH_DEBOUNCE_MS = 250

/**
 * useProductSearch returns up to `limit` products matching `query` (name or
 * SKU) via server-side search, so pickers work on catalogs larger than the
 * 1000-row cap of useProducts(). An empty query returns no results.
 */
export function useProductSearch(query: string, limit = 8) {
  const trimmed = query.trim()
  const [debounced, setDebounced] = useState(trimmed)

  useEffect(() => {
    const handle = setTimeout(() => setDebounced(trimmed), PRODUCT_SEARCH_DEBOUNCE_MS)
    return () => clearTimeout(handle)
  }, [trimmed])

  return useQuery<Product[]>({
    queryKey: ["products", "search", { q: debounced, limit }],
    queryFn: async (): Promise<Product[]> => {
      const res = await api.products.list({ q: debounced, page: 1, per_page: limit })
      return Array.isArray(res?.data) ? res.data : []
    },
    enabled: debounced.length > 0,
    staleTime: 30_000,
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
