import { useQuery } from '@tanstack/react-query'
import apiClient from '@/lib/api/client'
import { DashboardSummarySchema, type DashboardSummary } from '@/lib/schemas/dashboard'

const DASHBOARD_KEY = ['dashboard'] as const

export function useDashboardSummary() {
  return useQuery({
    queryKey: DASHBOARD_KEY,
    queryFn: async (): Promise<DashboardSummary> => {
      const res = await apiClient.get<unknown>('/api/v1/dashboard/summary')
      return DashboardSummarySchema.parse(res.data)
    },
    retry: 1,
    staleTime: 30_000, // 30 seconds — dashboard data doesn't change rapidly
  })
}
