"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { api, type POSOrder, type POSPaymentCharge, type POSPaymentStatus } from "@/lib/api"

export function usePOSOrders(limit = 50) {
  return useQuery<POSOrder[]>({
    queryKey: ["pos-orders", limit],
    queryFn: async (): Promise<POSOrder[]> => {
      const res = await api.pos.orders(limit)
      return res.data ?? []
    },
  })
}

export function usePOSCheckout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.pos.checkout,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["pos-orders"] })
      qc.invalidateQueries({ queryKey: ["products"] })
      qc.invalidateQueries({ queryKey: ["wms-movements"] })
      qc.invalidateQueries({ queryKey: ["wms-stock"] })
    },
  })
}

/** Starts a QRIS payment intent for the current cart total. */
export function usePOSCreatePayment() {
  return useMutation<POSPaymentCharge, Error, { amount: number; method: string }>({
    mutationFn: api.pos.createPayment,
  })
}

/**
 * Polls a payment intent until it is settled (or definitively not payable).
 *
 * The server is the source of truth: a pending payment is re-verified against
 * the gateway on every poll, so the cashier's screen follows the money rather
 * than deciding it. Polling stops on any terminal status.
 */
export function usePOSPaymentStatus(orderId: string | null, opts?: { enabled?: boolean }) {
  return useQuery<POSPaymentStatus>({
    queryKey: ["pos-payment", orderId],
    queryFn: () => api.pos.paymentStatus(orderId as string),
    enabled: !!orderId && (opts?.enabled ?? true),
    refetchInterval: (query) => {
      const status = query.state.data?.status
      if (!status) return 3000
      return status === "pending" ? 3000 : false
    },
  })
}

/**
 * Demo-mode only: settles a payment locally so the whole flow can be exercised
 * without a merchant account. The backend refuses this for tenants that have
 * real gateway credentials.
 */
export function usePOSSimulatePayment() {
  const qc = useQueryClient()
  return useMutation<{ status: string }, Error, string>({
    mutationFn: api.pos.simulatePayment,
    onSuccess: (_res, orderId) => {
      qc.invalidateQueries({ queryKey: ["pos-payment", orderId] })
    },
  })
}
