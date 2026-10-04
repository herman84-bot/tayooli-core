"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { api, type Subscription, type Plan, type SubscriptionInvoice, type UsageResponse, type PlanLimits } from "@/lib/api"

/* ── Subscription ──────────────────────────────────────────────────────────── */

export function useSubscription() {
  return useQuery<Subscription>({
    queryKey: ["subscription"],
    queryFn: () => api.subscription.get(),
    retry: false,
    staleTime: 60_000,
  })
}

export function useCreateSubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ plan, period }: { plan: string; period: string }) =>
      api.subscription.create(plan, period),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["subscription"] }),
  })
}

export function useUpdateSubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ plan, period }: { plan: string; period: string }) =>
      api.subscription.update(plan, period),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["subscription"] }),
  })
}

export function useCancelSubscription() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.subscription.cancel(),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["subscription"] }),
  })
}

/* ── Plans ─────────────────────────────────────────────────────────────────── */

export function usePlans() {
  return useQuery<Plan[]>({
    queryKey: ["plans"],
    queryFn: () => api.plans.list(),
    staleTime: 300_000,
  })
}

export function usePlanLimits(plan: string | undefined) {
  return useQuery<PlanLimits>({
    queryKey: ["plan-limits", plan],
    queryFn: () => api.plans.limits(plan!),
    enabled: !!plan,
    staleTime: 300_000,
  })
}

/* ── Usage ─────────────────────────────────────────────────────────────────── */

export function useUsage() {
  return useQuery<UsageResponse>({
    queryKey: ["usage"],
    queryFn: () => api.usage.get(),
    retry: false,
    staleTime: 60_000,
  })
}

/* ── Invoices ──────────────────────────────────────────────────────────────── */

export function useSubscriptionInvoices() {
  return useQuery<SubscriptionInvoice[]>({
    queryKey: ["subscription-invoices"],
    queryFn: () => api.subscription.invoices(),
    retry: false,
    staleTime: 30_000,
  })
}

export function useGeneratePaymentLink() {
  return useMutation({
    mutationFn: (invoiceId: string) => api.subscription.pay(invoiceId),
  })
}
