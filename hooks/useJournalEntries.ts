"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api } from "@/lib/api"
import type { CreateJournalEntryPayload } from "@/lib/api"

export function useJournalEntries() {
  return useQuery({
    queryKey: ["journal-entries"],
    queryFn: async () => {
      const res = await api.journalEntries.list()
      return res.data
    },
    retry: 1,
  })
}

export function useJournalEntry(id: string) {
  return useQuery({ queryKey: ["journal-entries", id], queryFn: () => api.journalEntries.get(id) })
}

export function useCreateJournalEntry() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateJournalEntryPayload) => api.journalEntries.create(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["journal-entries"] })
    },
  })
}
