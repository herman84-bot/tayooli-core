'use client'

import React from 'react'
import { QueryClient, QueryClientContext, useQuery, useMutation } from '@tanstack/react-query'
import { api, type CompanyProfile } from '@/lib/api'

// Fallback singleton query client for isolated environments (e.g. unit tests without QueryClientProvider)
const fallbackQueryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
    },
  },
})

export function useCompanyProfile() {
  const clientInContext = React.useContext(QueryClientContext)
  const client = clientInContext ?? fallbackQueryClient

  const query = useQuery<CompanyProfile>(
    {
      queryKey: ['company-profile'],
      queryFn: async () => {
        const res = await api.settings.getProfile()
        return res.data
      },
      staleTime: 60_000,
    },
    client
  )

  const updateProfile = useMutation(
    {
      mutationFn: (payload: Partial<CompanyProfile>) => api.settings.updateProfile(payload),
      onSuccess: (data) => {
        if (data?.data) {
          client.setQueryData(['company-profile'], data.data)
        }
        client.invalidateQueries({ queryKey: ['company-profile'] })
      },
    },
    client
  )

  return {
    profile: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
    updateProfile,
  }
}
