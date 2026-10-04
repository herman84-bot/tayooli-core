/**
 * Pakasir payment provider — implements the PaymentProvider interface on top
 * of the existing Pakasir helpers in `lib/pakasir.ts`.
 *
 * Pakasir (pakasir.com, PT Geksa) is an Indonesian payment-link platform
 * supporting QRIS and Virtual Account channels with webhook notifications.
 */
import {
  buildPaymentLink,
  DEFAULT_PAKASIR_API_BASE_URL,
  DEFAULT_PAKASIR_BASE_URL,
  parseWebhookPayload,
  verifyTransactionStatus as verifyPakasirStatus,
} from '@/lib/pakasir'
import type {
  CreatePaymentParams,
  CreatePaymentResult,
  NormalizedWebhook,
  PaymentProvider,
  PaymentProviderConfig,
  VerifiedStatus,
} from '../types'

export const pakasirProvider: PaymentProvider = {
  id: 'pakasir',
  name: 'Pakasir',
  description: 'Payment link — QRIS & Virtual Account (PT Geksa)',

  isConfigured: (config) => Boolean(config.apiKey),

  globalConfig: (): PaymentProviderConfig => ({
    provider: 'pakasir',
    slug: process.env.PAKASIR_PROJECT_SLUG ?? '',
    apiKey: process.env.PAKASIR_API_KEY ?? '',
    baseUrl: process.env.PAKASIR_BASE_URL,
    apiBaseUrl: process.env.PAKASIR_API_BASE_URL,
  }),

  createPayment: async (params: CreatePaymentParams): Promise<CreatePaymentResult> => {
    const { config, invoiceId, orderId, amount, method, redirect } = params
    if (config.slug && config.apiKey) {
      const paymentLink = buildPaymentLink({
        slug: config.slug,
        amount,
        orderId,
        redirect,
        baseUrl: config.baseUrl,
      })
      return { paymentLink, demo: false }
    }
    // Demo mode: point at the local demo checkout page.
    return { paymentLink: pakasirProvider.demoPaymentLink({ orderId, amount, method }), demo: true }
  },

  parseWebhook: (raw: unknown): NormalizedWebhook | null => {
    const payload = parseWebhookPayload(raw)
    if (!payload) return null
    return {
      provider: 'pakasir',
      orderId: payload.order_id,
      amount: payload.amount,
      status: payload.status,
      paymentMethod: payload.payment_method,
      completedAt: payload.completed_at,
      raw: payload as unknown as Record<string, unknown>,
    }
  },

  verifyWebhook: (config, webhook): boolean => {
    // The `project` field must belong to the tenant's registered slug.
    if (config.slug && webhook.raw.project !== config.slug) return false
    return true
  },

  verifyTransactionStatus: async (config, orderId, amount): Promise<VerifiedStatus | null> => {
    if (!config.apiKey || !config.slug) return null
    // Narrow to the fields Pakasir's helper expects (slug is required there).
    const pakasirConfig = {
      slug: config.slug,
      apiKey: config.apiKey,
      baseUrl: config.baseUrl,
      apiBaseUrl: config.apiBaseUrl,
    }
    const result = await verifyPakasirStatus({ config: pakasirConfig, orderId, amount })
    return result ? { status: result.status, amount: result.amount } : null
  },

  isPaidStatus: (status) =>
    status === 'completed' || status === 'settlement' || status === 'paid',

  demoPaymentLink: ({ orderId, amount, method }) =>
    `/payments/demo/${encodeURIComponent(orderId)}?provider=pakasir&amount=${Math.round(amount)}&method=${method}`,
}

// Re-export the default base URLs so tests and docs stay consistent.
export { DEFAULT_PAKASIR_API_BASE_URL, DEFAULT_PAKASIR_BASE_URL }
