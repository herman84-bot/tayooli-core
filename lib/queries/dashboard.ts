import { useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/client'
import { DashboardSummarySchema, type DashboardSummary } from '@/lib/schemas/dashboard'

export function useDashboardSummary(period: string = '30d') {
  return useQuery({
    queryKey: ['dashboard', period] as const,
    queryFn: async (): Promise<DashboardSummary> => {
      const res = await apiClient.get<unknown>(`/api/v1/dashboard/summary?period=${encodeURIComponent(period)}`)
      return DashboardSummarySchema.parse(res.data)
    },
    // Retry transient failures (e.g. auth cookie not yet hydrated on first load),
    // but not a real 401 — retrying that only delays the "please log in" banner.
    retry: (failureCount, error) => {
      const status = (error as { response?: { status?: number } } | null)?.response?.status
      if (status === 401 || status === 403) return false
      return failureCount < 2
    },
    retryDelay: 1000,
    staleTime: 30_000, // 30 seconds — dashboard data doesn't change rapidly
  })
}
