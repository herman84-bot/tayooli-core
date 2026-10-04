/**
 * Shared webhook receiver logic for all payment providers.
 *
 * Every provider exposes its own endpoint (`/api/webhooks/pakasir`,
 * `/api/webhooks/midtrans`) that delegates here. The flow is gateway-
 * agnostic:
 *
 *  1. Parse the raw body through the provider's `parseWebhook`.
 *  2. Map `order_id` → stored transaction, which records its owning tenant
 *     and provider.
 *  3. Provider authenticity check (signature / project slug match).
 *  4. When the tenant has real credentials: re-verify server-to-server
 *     (official best practice — never trust the webhook body alone).
 *  5. Demo mode (no credentials): the amount must match the created
 *     transaction.
 *  6. If paid → mark the invoice `Paid`.
 */
import { NextResponse } from 'next/server'
import { getProvider, resolveProviderConfig } from './registry'
import {
  getTransactionByOrderId,
  markInvoicePaid,
} from '@/lib/payment-store'
import type { PaymentProviderId } from './types'

export async function handleProviderWebhook(
  providerId: PaymentProviderId,
  request: Request
): Promise<NextResponse> {
  const provider = getProvider(providerId)

  let raw: unknown
  try {
    raw = await request.json()
  } catch {
    return NextResponse.json({ error: 'invalid json body' }, { status: 400 })
  }

  const webhook = provider.parseWebhook(raw)
  if (!webhook) {
    return NextResponse.json(
      { error: 'malformed webhook payload' },
      { status: 400 }
    )
  }

  // Order id maps to the transaction, which carries tenant + provider.
  const orderId = webhook.orderId
  const tx = getTransactionByOrderId(orderId)
  const invoiceId = tx?.invoiceId

  if (!invoiceId || !tx) {
    // Unknown order — not necessarily an error (could be from another
    // system), but we must not mark anything paid. Acknowledge to avoid
    // retries.
    return NextResponse.json({ ok: true, ignored: true })
  }

  // Guard: a webhook for provider A must never settle a transaction created
  // through provider B.
  if (tx.provider !== providerId) {
    return NextResponse.json(
      { error: `webhook provider mismatch: expected ${tx.provider}` },
      { status: 403 }
    )
  }

  const { config, demo } = resolveProviderConfig(tx.tenantId, providerId)

  // 1. Authenticity (signature / project match) — always enforce when a
  //    provider-level identity exists.
  if (!provider.verifyWebhook(config, webhook)) {
    return NextResponse.json(
      { error: `${provider.name} webhook verification failed` },
      { status: 403 }
    )
  }

  // 2. Server-side verification when real credentials are configured.
  let isPaid = false
  if (!demo) {
    const verified = await provider.verifyTransactionStatus(
      config,
      orderId,
      tx.amount
    )
    if (!verified) {
      return NextResponse.json(
        { error: `could not verify transaction with ${provider.name}` },
        { status: 502 }
      )
    }
    isPaid = provider.isPaidStatus(verified.status)
  } else {
    // Demo mode: amount must match what we created for this tenant's invoice.
    if (Math.round(webhook.amount) !== tx.amount) {
      return NextResponse.json({ error: 'amount mismatch' }, { status: 400 })
    }
    isPaid = provider.isPaidStatus(webhook.status)
  }

  if (!isPaid) {
    return NextResponse.json({ ok: true, ignored: true })
  }

  markInvoicePaid({
    invoiceId,
    paymentMethod: webhook.paymentMethod,
    completedAt: webhook.completedAt,
  })

  return NextResponse.json({ ok: true, tenantId: tx.tenantId })
}
