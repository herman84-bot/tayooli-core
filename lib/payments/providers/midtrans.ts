/**
 * Midtrans payment provider — implements the PaymentProvider interface.
 *
 * Midtrans (Midtrans.com, GoTo Financial) is a licensed Indonesian payment
 * gateway. Integration uses the Snap API: we create a transaction
 * server-side with the tenant's Server Key, get a `redirect_url`, and point
 * the customer at Midtrans' hosted Snap page (QRIS, Virtual Account,
 * e-wallet, etc.). Midtrans notifies us via webhook; we verify the SHA512
 * signature and re-check the status server-side before marking the invoice
 * paid.
 *
 * Settlement: each tenant has their own Midtrans merchant account (KYC) and
 * funds settle to the tenant's own bank account — Model B. See ADR-008.
 */
import { createHash } from 'crypto'
import type {
  CreatePaymentParams,
  CreatePaymentResult,
  NormalizedWebhook,
  PaymentProvider,
  PaymentProviderConfig,
  VerifiedStatus,
} from '../types'

export const MIDTRANS_SANDBOX_BASE_URL = 'https://app.sandbox.midtrans.com'
export const MIDTRANS_PRODUCTION_BASE_URL = 'https://app.midtrans.com'
export const MIDTRANS_SANDBOX_API_BASE_URL = 'https://api.sandbox.midtrans.com'
export const MIDTRANS_PRODUCTION_API_BASE_URL = 'https://api.midtrans.com'

function basicAuth(serverKey: string): string {
  return `Basic ${Buffer.from(`${serverKey}:`).toString('base64')}`
}

function snapBaseUrl(config: PaymentProviderConfig): string {
  if (config.baseUrl) return config.baseUrl
  return config.isProduction ? MIDTRANS_PRODUCTION_BASE_URL : MIDTRANS_SANDBOX_BASE_URL
}

function apiBaseUrl(config: PaymentProviderConfig): string {
  if (config.apiBaseUrl) return config.apiBaseUrl
  return config.isProduction ? MIDTRANS_PRODUCTION_API_BASE_URL : MIDTRANS_SANDBOX_API_BASE_URL
}

export const midtransProvider: PaymentProvider = {
  id: 'midtrans',
  name: 'Midtrans',
  description: 'Snap — QRIS, Virtual Account & e-wallet (GoTo Financial)',

  isConfigured: (config) => Boolean(config.serverKey),

  globalConfig: (): PaymentProviderConfig => ({
    provider: 'midtrans',
    serverKey: process.env.MIDTRANS_SERVER_KEY ?? '',
    clientKey: process.env.MIDTRANS_CLIENT_KEY ?? '',
    isProduction: process.env.MIDTRANS_IS_PRODUCTION === 'true',
    baseUrl: process.env.MIDTRANS_BASE_URL,
    apiBaseUrl: process.env.MIDTRANS_API_BASE_URL,
  }),

  createPayment: async (params: CreatePaymentParams): Promise<CreatePaymentResult> => {
    const { config, invoiceId, orderId, amount, method, redirect } = params
    if (config.serverKey) {
      const res = await fetch(`${snapBaseUrl(config)}/snap/v1/transactions`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          Authorization: basicAuth(config.serverKey),
        },
        body: JSON.stringify({
          transaction_details: {
            order_id: orderId,
            gross_amount: Math.round(amount),
          },
          item_details: [
            {
              id: invoiceId,
              price: Math.round(amount),
              quantity: 1,
              name: `Invoice ${invoiceId}`,
            },
          ],
          credit_card: { secure: true },
          ...(redirect ? { callbacks: { finish: redirect } } : {}),
        }),
        cache: 'no-store',
      })
      if (!res.ok) {
        throw new Error(`Midtrans Snap API error ${res.status}`)
      }
      const data = (await res.json()) as Record<string, unknown>
      const paymentLink =
        typeof data.redirect_url === 'string' ? data.redirect_url : undefined
      if (!paymentLink) {
        throw new Error('Midtrans Snap API returned no redirect_url')
      }
      return { paymentLink, demo: false }
    }
    // Demo mode: point at the local demo checkout page.
    return { paymentLink: midtransProvider.demoPaymentLink({ orderId, amount, method }), demo: true }
  },

  parseWebhook: (raw: unknown): NormalizedWebhook | null => {
    if (typeof raw !== 'object' || raw === null) return null
    const body = raw as Record<string, unknown>
    const orderId = typeof body.order_id === 'string' ? body.order_id.trim() : ''
    const grossAmount = typeof body.gross_amount === 'string' ? body.gross_amount.trim() : ''
    const status = typeof body.transaction_status === 'string' ? body.transaction_status.trim().toLowerCase() : ''
    const amount = Number(grossAmount)

    if (!orderId || !Number.isFinite(amount) || amount <= 0 || !status) {
      return null
    }
    return {
      provider: 'midtrans',
      orderId,
      amount,
      status,
      paymentMethod: typeof body.payment_type === 'string' ? body.payment_type : undefined,
      completedAt:
        typeof body.transaction_time === 'string' ? body.transaction_time : undefined,
      raw: body,
    }
  },

  verifyWebhook: (config, webhook): boolean => {
    // No server key → cannot verify (demo mode; caller falls back to amount).
    if (!config.serverKey) return true
    const body = webhook.raw
    const statusCode = typeof body.status_code === 'string' ? body.status_code : ''
    const grossAmount = typeof body.gross_amount === 'string' ? body.gross_amount : ''
    const receivedSignature = typeof body.signature_key === 'string' ? body.signature_key : ''
    if (!statusCode || !grossAmount || !receivedSignature) return false
    // Midtrans signature: SHA512(order_id + status_code + gross_amount + server_key)
    const expected = createHash('sha512')
      .update(`${webhook.orderId}${statusCode}${grossAmount}${config.serverKey}`)
      .digest('hex')
    return expected === receivedSignature
  },

  verifyTransactionStatus: async (config, orderId, amount): Promise<VerifiedStatus | null> => {
    if (!config.serverKey) return null
    const url = `${apiBaseUrl(config)}/v2/${encodeURIComponent(orderId)}/status`
    const res = await fetch(url, {
      headers: { Authorization: basicAuth(config.serverKey) },
      cache: 'no-store',
    })
    if (!res.ok) return null
    const data = (await res.json()) as Record<string, unknown>
    const status = typeof data.transaction_status === 'string' ? data.transaction_status.toLowerCase() : ''
    const amountFromApi =
      typeof data.gross_amount === 'string' ? Number(data.gross_amount) : undefined
    return { status, amount: Number.isFinite(amountFromApi) ? amountFromApi : undefined }
  },

  isPaidStatus: (status) => status === 'capture' || status === 'settlement',

  demoPaymentLink: ({ orderId, amount, method }) =>
    `/payments/demo/${encodeURIComponent(orderId)}?provider=midtrans&amount=${Math.round(amount)}&method=${method}`,
}
