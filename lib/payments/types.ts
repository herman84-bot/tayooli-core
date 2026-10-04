/**
 * Shared types for the multi-gateway payment abstraction (lib/payments).
 *
 * Tayooli supports more than one payment gateway (currently Pakasir and
 * Midtrans). Every gateway is implemented behind the `PaymentProvider`
 * interface and registered in `lib/payments/registry.ts`, so the route
 * handlers, webhook receiver and UI are provider-agnostic.
 *
 * Multi-tenant model (Model B): each tenant owns their own gateway account /
 * project. All provider functions receive a `PaymentProviderConfig` resolved
 * per-tenant; process-level env vars act only as a global fallback for
 * development/demo mode.
 */

export type PaymentProviderId = 'pakasir' | 'midtrans'

export type PaymentMethod = 'qris' | 'va'

export type PaymentStatus = 'pending' | 'completed' | 'expired' | 'failed'

/**
 * Flattened per-tenant provider configuration. Only the fields relevant to
 * the selected `provider` are used.
 */
export interface PaymentProviderConfig {
  provider: PaymentProviderId
  /** Pakasir: public project slug (safe to expose). */
  slug?: string
  /** Pakasir: secret per-project API key (server-only). */
  apiKey?: string
  /** Midtrans: secret server key (server-only). */
  serverKey?: string
  /** Midtrans: public client key (used by snap.js). */
  clientKey?: string
  /** Midtrans: production vs sandbox mode. */
  isProduction?: boolean
  /** Hosted checkout base URL override. */
  baseUrl?: string
  /** Server-to-server API base URL override. */
  apiBaseUrl?: string
}

/** A payment transaction stored by the demo store (one per invoice). */
export interface PaymentTransaction {
  invoiceId: string
  /** Owning tenant id (isolates webhook handling per tenant). */
  tenantId: string
  /** Gateway that processed this transaction. */
  provider: PaymentProviderId
  orderId: string
  amount: number
  method: PaymentMethod
  paymentLink: string
  status: PaymentStatus
  createdAt: string
  completedAt?: string
  paymentMethod?: string
}

/** Provider-agnostic webhook payload normalized by each provider. */
export interface NormalizedWebhook {
  provider: PaymentProviderId
  orderId: string
  amount: number
  status: string
  paymentMethod?: string
  completedAt?: string
  /** Provider-specific raw fields (signature, project, status_code, …). */
  raw: Record<string, unknown>
}

export interface CreatePaymentParams {
  config: PaymentProviderConfig
  invoiceId: string
  orderId: string
  amount: number
  method: PaymentMethod
  redirect?: string
}

export interface CreatePaymentResult {
  paymentLink: string
  /** True when no real credentials are configured (simulate enabled). */
  demo: boolean
}

export interface VerifiedStatus {
  status: string
  amount?: number
}

/** Contract every payment gateway must implement. */
export interface PaymentProvider {
  id: PaymentProviderId
  name: string
  description: string

  /** True when real credentials are available for the given config. */
  isConfigured(config: PaymentProviderConfig): boolean

  /** Process-level env fallback config (demo / single-tenant). */
  globalConfig(): PaymentProviderConfig

  /**
   * Create a payment and return the hosted checkout link. In demo mode
   * (no credentials) returns a link to the local demo checkout page.
   */
  createPayment(params: CreatePaymentParams): Promise<CreatePaymentResult>

  /** Parse and validate a webhook body; null when malformed. */
  parseWebhook(raw: unknown): NormalizedWebhook | null

  /**
   * Provider-specific authenticity check (signature / project slug match).
   * Returns true when the webhook is genuine or when verification is not
   * possible (demo mode — the caller falls back to amount matching).
   */
  verifyWebhook(config: PaymentProviderConfig, webhook: NormalizedWebhook): boolean

  /** Re-verify a transaction server-to-server. Null when not possible. */
  verifyTransactionStatus(
    config: PaymentProviderConfig,
    orderId: string,
    amount: number
  ): Promise<VerifiedStatus | null>

  /** True when the provider status string means the payment succeeded. */
  isPaidStatus(status: string): boolean

  /** Link to the local demo checkout page (used in demo mode). */
  demoPaymentLink(params: { orderId: string; amount: number; method: PaymentMethod }): string
}
