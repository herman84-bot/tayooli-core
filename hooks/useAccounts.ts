"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import type { CreateAccountInput } from "@/lib/api"

export function useAccounts() {
  return useQuery({
    queryKey: ["accounts"],
    queryFn: async () => {
      const res = await api.accounts.list()
      return res.data
    },
    retry: 1,
  })
}

export function useAccount(id: string) {
  return useQuery({ queryKey: ["accounts", id], queryFn: () => api.accounts.get(id) })
}

export function useCreateAccount() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateAccountInput) => api.accounts.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["accounts"] })
    },
  })
}
