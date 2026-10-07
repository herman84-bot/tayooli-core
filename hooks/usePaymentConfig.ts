'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, TenantPaymentConfigResponse, UpsertPaymentConfigPayload } from '@/lib/api'

export function usePaymentConfig(provider: string = 'midtrans') {
  const queryClient = useQueryClient()

  const query = useQuery<TenantPaymentConfigResponse>({
    queryKey: ['payment-config', provider],
    queryFn: () => api.payments.getConfig(provider),
  })

  const saveConfig = useMutation({
    mutationFn: (payload: UpsertPaymentConfigPayload) => api.payments.upsertConfig(payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['payment-config'] })
    },
  })

  return {
    config: query.data,
    isLoading: query.isLoading,
    isError: query.isError,
    error: query.error,
    refetch: query.refetch,
    saveConfig,
  }
}
