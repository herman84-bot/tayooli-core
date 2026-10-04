"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api, type Approval } from "@/lib/api"

export interface ApprovalList {
  data: Approval[]
  total: number
  page: number
  per_page: number
}

interface UseApprovalsParams {
  page?: number
  perPage?: number
}

export function useApprovals({ page = 1, perPage = 20 }: UseApprovalsParams = {}) {
  return useQuery({
    queryKey: ["approvals", { page, perPage }],
    queryFn: async (): Promise<ApprovalList> => {
      const res = await api.approvals.list({ page, per_page: perPage })
      // Backend returns a paginated envelope { data, total, page, per_page };
      // tolerate a bare array too (older clients/demo auth).
      const payload = res.data as
        | Approval[]
        | { data?: Approval[]; total?: number; page?: number; per_page?: number }
      const arr = Array.isArray(payload) ? payload : (payload.data ?? [])
      const meta = Array.isArray(payload) ? {} : payload
      return {
        data: arr,
        total: meta.total ?? arr.length,
        page: meta.page ?? page,
        per_page: meta.per_page ?? perPage,
      }
    },
    retry: 1,
  })
}

export function useApprovalActions() {
  const qc = useQueryClient()
  const approve = useMutation({
    mutationFn: (id: string) => api.approvals.approve(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["approvals"] }),
  })
  const reject = useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.approvals.reject(id, reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["approvals"] }),
  })
  return { approve, reject }
}
