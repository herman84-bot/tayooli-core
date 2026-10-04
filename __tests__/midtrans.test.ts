import { createHash } from 'crypto'
import { midtransProvider } from '@/lib/payments/providers/midtrans'
import {
  getProvider,
  isPaymentProviderId,
  resolveProviderConfig,
} from '@/lib/payments/registry'
import {
  resetPaymentStore,
  setTenantPaymentConfig,
} from '@/lib/payment-store'

const SERVER_KEY = 'Midtrans-Server-Key-123'

function sign(payload: {
  orderId: string
  statusCode: string
  grossAmount: string
}): string {
  return createHash('sha512')
    .update(`${payload.orderId}${payload.statusCode}${payload.grossAmount}${SERVER_KEY}`)
    .digest('hex')
}

describe('lib/payments/providers/midtrans — config', () => {
  it('detects live vs demo mode from the server key', () => {
    expect(midtransProvider.isConfigured({ provider: 'midtrans' })).toBe(false)
    expect(
      midtransProvider.isConfigured({ provider: 'midtrans', serverKey: SERVER_KEY })
    ).toBe(true)
  })

  it('builds a config from env fallbacks', () => {
    const cfg = midtransProvider.globalConfig()
    expect(cfg.provider).toBe('midtrans')
    expect(typeof cfg.serverKey).toBe('string')
  })
})

describe('lib/payments/providers/midtrans — webhook parsing', () => {
  const basePayload = {
    order_id: 'INV-2001-AB12',
    status_code: '200',
    gross_amount: '125000.00',
    transaction_status: 'settlement',
    payment_type: 'qris',
    transaction_time: '2026-08-07T10:00:00.000+07:00',
    signature_key: 'ignored-in-parse',
  }

  it('parses a valid Midtrans webhook payload', () => {
    const webhook = midtransProvider.parseWebhook(basePayload)
    expect(webhook).not.toBeNull()
    expect(webhook?.provider).toBe('midtrans')
    expect(webhook?.orderId).toBe('INV-2001-AB12')
    expect(webhook?.amount).toBe(125000)
    expect(webhook?.status).toBe('settlement')
    expect(webhook?.paymentMethod).toBe('qris')
  })

  it('rejects malformed payloads', () => {
    expect(midtransProvider.parseWebhook(null)).toBeNull()
    expect(midtransProvider.parseWebhook('nope')).toBeNull()
    expect(midtransProvider.parseWebhook({ order_id: 'x' })).toBeNull()
    expect(
      midtransProvider.parseWebhook({ order_id: 'x', gross_amount: 'abc' })
    ).toBeNull()
    expect(
      midtransProvider.parseWebhook({
        order_id: '',
        gross_amount: '100',
        transaction_status: 'pending',
      })
    ).toBeNull()
  })

  it('accepts capture as a paid status', () => {
    const webhook = midtransProvider.parseWebhook({
      ...basePayload,
      transaction_status: 'capture',
      fraud_status: 'accept',
    })
    expect(webhook?.status).toBe('capture')
    expect(midtransProvider.isPaidStatus('capture')).toBe(true)
    expect(midtransProvider.isPaidStatus('settlement')).toBe(true)
    expect(midtransProvider.isPaidStatus('pending')).toBe(false)
    expect(midtransProvider.isPaidStatus('expire')).toBe(false)
  })
})

describe('lib/payments/providers/midtrans — SHA512 signature verification', () => {
  it('accepts a webhook with a valid signature', () => {
    const body = {
      order_id: 'INV-2001-AB12',
      status_code: '200',
      gross_amount: '125000.00',
      transaction_status: 'settlement',
      signature_key: sign({
        orderId: 'INV-2001-AB12',
        statusCode: '200',
        grossAmount: '125000.00',
      }),
    }
    const webhook = midtransProvider.parseWebhook(body)
    expect(webhook).not.toBeNull()
    expect(
      midtransProvider.verifyWebhook(
        { provider: 'midtrans', serverKey: SERVER_KEY },
        webhook as NonNullable<typeof webhook>
      )
    ).toBe(true)
  })

  it('rejects a webhook with a tampered signature', () => {
    const body = {
      order_id: 'INV-2001-AB12',
      status_code: '200',
      gross_amount: '125000.00',
      transaction_status: 'settlement',
      signature_key: 'deadbeef',
    }
    const webhook = midtransProvider.parseWebhook(body)
    expect(
      midtransProvider.verifyWebhook(
        { provider: 'midtrans', serverKey: SERVER_KEY },
        webhook as NonNullable<typeof webhook>
      )
    ).toBe(false)
  })

  it('returns true (unverifiable) when no server key is configured — demo mode', () => {
    const body = {
      order_id: 'INV-2001-AB12',
      status_code: '200',
      gross_amount: '125000.00',
      transaction_status: 'settlement',
    }
    const webhook = midtransProvider.parseWebhook(body)
    expect(
      midtransProvider.verifyWebhook(
        { provider: 'midtrans' },
        webhook as NonNullable<typeof webhook>
      )
    ).toBe(true)
  })
})

describe('lib/payments/providers/midtrans — demo link', () => {
  it('points at the local demo checkout page in demo mode', () => {
    const link = midtransProvider.demoPaymentLink({
      orderId: 'INV-2001-AB12',
      amount: 125000,
      method: 'qris',
    })
    expect(link).toContain('/payments/demo/INV-2001-AB12')
    expect(link).toContain('provider=midtrans')
    expect(link).toContain('amount=125000')
    expect(link).toContain('method=qris')
  })
})

describe('lib/payments/registry', () => {
  it('registers both providers', () => {
    expect(getProvider('pakasir').name).toBe('Pakasir')
    expect(getProvider('midtrans').name).toBe('Midtrans')
    expect(isPaymentProviderId('pakasir')).toBe(true)
    expect(isPaymentProviderId('midtrans')).toBe(true)
    expect(isPaymentProviderId('unknown')).toBe(false)
  })

  it('resolves the tenant config for the matching provider (Model B)', () => {
    resetPaymentStore()
    setTenantPaymentConfig({
      tenantId: 'tenant-acme',
      provider: 'midtrans',
      serverKey: SERVER_KEY,
      clientKey: 'client-1',
      isActive: true,
    })
    const { config, demo } = resolveProviderConfig('tenant-acme', 'midtrans')
    expect(config.serverKey).toBe(SERVER_KEY)
    expect(demo).toBe(false)

    // The same tenant has no Pakasir config → falls back to demo/global.
    const pakasirCfg = resolveProviderConfig('tenant-acme', 'pakasir')
    expect(pakasirCfg.config.provider).toBe('pakasir')
    expect(pakasirCfg.config.slug).toBe('')
  })

  it('falls back to demo mode when the tenant has no credentials', () => {
    resetPaymentStore()
    setTenantPaymentConfig({
      tenantId: 'tenant-new',
      provider: 'midtrans',
      isActive: true,
    })
    const { demo } = resolveProviderConfig('tenant-new', 'midtrans')
    expect(demo).toBe(true)
  })
})
