import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import apiClient from '@/lib/api/client'
import {
  PaymentOrderSchema,
  PaymentOrderListSchema,
  type PaymentOrder,
  type PaymentOrderList,
  type CreatePaymentOrderInput,
} from '@/lib/schemas/payment-order'

const PAYMENT_ORDERS_KEY = ['payment-orders'] as const

interface UsePaymentOrdersParams {
  page?: number
  perPage?: number
}

// ── List all payment orders (paginated) ──────────────────────────────────────
export function usePaymentOrders({ page = 1, perPage = 20 }: UsePaymentOrdersParams = {}) {
  return useQuery({
    queryKey: [...PAYMENT_ORDERS_KEY, { page, perPage }] as const,
    queryFn: async (): Promise<PaymentOrderList> => {
      const res = await apiClient.get<unknown>('/api/v1/payment-orders', {
        params: { page, per_page: perPage },
      })
      return PaymentOrderListSchema.parse(res.data)
    },
    retry: 1,
  })
}

// ── Single payment order ─────────────────────────────────────────────────────
export function usePaymentOrder(id: string) {
  return useQuery({
    queryKey: [...PAYMENT_ORDERS_KEY, id] as const,
    queryFn: async (): Promise<PaymentOrder> => {
      const res = await apiClient.get<unknown>(`/api/v1/payment-orders/${id}`)
      return PaymentOrderSchema.parse(res.data)
    },
    enabled: Boolean(id),
    retry: 1,
  })
}

// ── Create mutation ──────────────────────────────────────────────────────────
export function useCreatePaymentOrder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (data: CreatePaymentOrderInput): Promise<PaymentOrder> => {
      const res = await apiClient.post<unknown>('/api/v1/payment-orders', data)
      return PaymentOrderSchema.parse(res.data)
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: PAYMENT_ORDERS_KEY })
    },
  })
}

// ── Approve mutation ─────────────────────────────────────────────────────────
export function useApprovePaymentOrder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string): Promise<PaymentOrder> => {
      const res = await apiClient.post<unknown>(`/api/v1/payment-orders/${id}/approve`)
      return PaymentOrderSchema.parse(res.data)
    },
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({ queryKey: PAYMENT_ORDERS_KEY })
      void queryClient.invalidateQueries({ queryKey: [...PAYMENT_ORDERS_KEY, id] })
    },
  })
}

// ── Pay mutation ─────────────────────────────────────────────────────────────
export function usePayPaymentOrder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string): Promise<PaymentOrder> => {
      const res = await apiClient.post<unknown>(`/api/v1/payment-orders/${id}/pay`)
      return PaymentOrderSchema.parse(res.data)
    },
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({ queryKey: PAYMENT_ORDERS_KEY })
      void queryClient.invalidateQueries({ queryKey: [...PAYMENT_ORDERS_KEY, id] })
    },
  })
}

// ── Reject mutation ──────────────────────────────────────────────────────────
export function useRejectPaymentOrder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async (id: string): Promise<PaymentOrder> => {
      const res = await apiClient.post<unknown>(`/api/v1/payment-orders/${id}/reject`)
      return PaymentOrderSchema.parse(res.data)
    },
    onSuccess: (_data, id) => {
      void queryClient.invalidateQueries({ queryKey: PAYMENT_ORDERS_KEY })
      void queryClient.invalidateQueries({ queryKey: [...PAYMENT_ORDERS_KEY, id] })
    },
  })
}
