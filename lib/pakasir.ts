/**
 * Pakasir payment gateway helpers.
 *
 * Pakasir (pakasir.com) is an Indonesian payment-link platform (PT Geksa).
 * It supports QRIS and Virtual Account channels (BRI, BNI, Permata, CIMB,
 * Maybank, BNC, etc.) and notifies merchants via a webhook when a customer
 * completes a payment.
 *
 * Multi-tenant model (Model B): every tenant owns its own Pakasir project
 * (slug + api_key). All functions therefore accept a `PakasirProviderConfig`
 * argument resolved per-tenant; the process-level env vars act only as a
 * global fallback for development/demo mode.
 */

/** Per-tenant Pakasir provider configuration (persisted per tenant). */
export interface PakasirProviderConfig {
  /** Public project slug created in the tenant's Pakasir dashboard. */
  slug: string
  /** Secret per-project API key. Optional (demo mode without it). */
  apiKey?: string
  /** Hosted checkout base URL override. */
  baseUrl?: string
  /** API base URL override (used for server-side verification). */
  apiBaseUrl?: string
}

export const DEFAULT_PAKASIR_BASE_URL = 'https://app.pakasir.com'
export const DEFAULT_PAKASIR_API_BASE_URL = 'https://app.pakasir.com/api'

/** Process-level fallback config (demo/development or single-tenant mode). */
export function globalProviderConfig(): PakasirProviderConfig {
  return {
    slug: process.env.PAKASIR_PROJECT_SLUG ?? '',
    apiKey: process.env.PAKASIR_API_KEY ?? '',
    baseUrl: process.env.PAKASIR_BASE_URL,
    apiBaseUrl: process.env.PAKASIR_API_BASE_URL,
  }
}

/** True when a real API key is available for the given config. */
export function isProviderConfigured(config: PakasirProviderConfig): boolean {
  return Boolean(config.apiKey)
}

export type PakasirMethod = 'qris' | 'va'

export interface PakasirTransaction {
  /** Internal ERP invoice id this payment belongs to. */
  invoiceId: string
  /** Owning tenant id (used to isolate webhook handling). */
  tenantId: string
  /** Pakasir order id, unique per transaction. */
  orderId: string
  amount: number
  method: PakasirMethod
  paymentLink: string
  status: 'pending' | 'completed' | 'expired' | 'failed'
  createdAt: string
  completedAt?: string
  paymentMethod?: string
}

/** Webhook payload sent by Pakasir to the merchant webhook URL. */
export interface PakasirWebhookPayload {
  amount: number
  order_id: string
  project: string
  status: string
  payment_method?: string
  completed_at?: string
}

/**
 * Build the hosted checkout link.
 * amount must be a positive integer (Pakasir does not accept decimals/dots).
 */
export function buildPaymentLink(params: {
  slug: string
  amount: number
  orderId: string
  redirect?: string
  baseUrl?: string
}): string {
  const { slug, amount, orderId, redirect, baseUrl } = params
  const amountInt = Math.round(amount)
  if (amountInt <= 0) {
    throw new Error('Pakasir amount must be a positive integer')
  }
  if (!slug) {
    throw new Error('Pakasir project slug is not configured')
  }
  const url = new URL(
    `${baseUrl ?? DEFAULT_PAKASIR_BASE_URL}/pay/${encodeURIComponent(slug)}/${amountInt}`
  )
  url.searchParams.set('order_id', orderId)
  if (redirect) {
    url.searchParams.set('redirect', redirect)
  }
  return url.toString()
}

/** Parse a webhook body defensively. Returns null when malformed. */
export function parseWebhookPayload(raw: unknown): PakasirWebhookPayload | null {
  if (typeof raw !== 'object' || raw === null) {
    return null
  }
  const body = raw as Record<string, unknown>
  const amount = Number(body.amount)
  const orderId = typeof body.order_id === 'string' ? body.order_id.trim() : ''
  const project = typeof body.project === 'string' ? body.project.trim() : ''
  const status = typeof body.status === 'string' ? body.status.trim().toLowerCase() : ''

  if (!Number.isFinite(amount) || amount <= 0 || !orderId || !project || !status) {
    return null
  }
  return {
    amount,
    order_id: orderId,
    project,
    status,
    payment_method:
      typeof body.payment_method === 'string' ? body.payment_method : undefined,
    completed_at:
      typeof body.completed_at === 'string' ? body.completed_at : undefined,
  }
}

/**
 * Verify a transaction server-to-server using the Pakasir transaction detail
 * endpoint. This is the official best practice — never trust the webhook body
 * alone. Returns null when verification cannot be performed (e.g. no API key
 * configured) so callers can decide how to handle it.
 */
export async function verifyTransactionStatus(params: {
  config: PakasirProviderConfig
  orderId: string
  amount: number
}): Promise<{ status: string; amount?: number } | null> {
  const { config, orderId, amount } = params
  if (!isProviderConfigured(config) || !config.slug) {
    return null
  }
  const url = new URL(`${config.apiBaseUrl ?? DEFAULT_PAKASIR_API_BASE_URL}/transactiondetail`)
  url.searchParams.set('project', config.slug)
  url.searchParams.set('amount', String(Math.round(amount)))
  url.searchParams.set('order_id', orderId)
  url.searchParams.set('api_key', config.apiKey as string)

  const res = await fetch(url.toString(), { cache: 'no-store' })
  if (!res.ok) {
    return null
  }
  const data = (await res.json()) as Record<string, unknown>
  const status = typeof data.status === 'string' ? data.status : ''
  const amountFromApi = typeof data.amount === 'number' ? data.amount : undefined
  return { status: status.toLowerCase(), amount: amountFromApi }
}

/** Generate a short, URL-safe order id like "INV-2001-P3K" for a payment. */
export function buildOrderId(invoiceId: string): string {
  const clean = invoiceId.replace(/[^A-Za-z0-9-]/g, '').slice(0, 40)
  const suffix = Math.random().toString(36).slice(2, 6).toUpperCase()
  return `${clean}-${suffix}`
}
