import {
  buildPaymentLink,
  buildOrderId,
  parseWebhookPayload,
  globalProviderConfig,
  isProviderConfigured,
} from '@/lib/pakasir'
import {
  createTransaction,
  getTransaction,
  getTransactionByOrderId,
  markInvoicePaid,
  getInvoiceStatus,
  setTenantPaymentConfig,
  getTenantPaymentConfig,
  listTenantPaymentConfigs,
  resetPaymentStore,
} from '@/lib/payment-store'

describe('lib/pakasir — payment link builder', () => {
  it('builds a hosted checkout link with slug, amount and order_id', () => {
    const link = buildPaymentLink({
      slug: 'my-project',
      amount: 125000,
      orderId: 'INV-2001-ABC1',
    })
    expect(link).toBe(
      'https://app.pakasir.com/pay/my-project/125000?order_id=INV-2001-ABC1'
    )
  })

  it('rounds fractional amounts to an integer', () => {
    const link = buildPaymentLink({
      slug: 'p',
      amount: 1250.5,
      orderId: 'O1',
    })
    expect(link).toContain('/pay/p/1251')
  })

  it('appends the redirect param when provided', () => {
    const link = buildPaymentLink({
      slug: 'p',
      amount: 100,
      orderId: 'O1',
      redirect: 'https://example.com/success',
    })
    expect(link).toContain('redirect=https%3A%2F%2Fexample.com%2Fsuccess')
  })

  it('rejects non-positive amounts', () => {
    expect(() =>
      buildPaymentLink({ slug: 'p', amount: 0, orderId: 'O1' })
    ).toThrow('positive integer')
  })

  it('rejects missing slug', () => {
    expect(() =>
      buildPaymentLink({ slug: '', amount: 100, orderId: 'O1' })
    ).toThrow('slug')
  })

  it('uses a custom base URL when provided', () => {
    const link = buildPaymentLink({
      slug: 'p',
      amount: 100,
      orderId: 'O1',
      baseUrl: 'https://sandbox.pakasir.test',
    })
    expect(link).toContain('https://sandbox.pakasir.test/pay/p/100')
  })

  it('detects live vs demo mode from the api key', () => {
    expect(isProviderConfigured({ slug: 'p' })).toBe(false)
    expect(isProviderConfigured({ slug: 'p', apiKey: 'sk_live_1' })).toBe(true)
  })

  it('builds a provider config from env fallbacks', () => {
    const cfg = globalProviderConfig()
    expect(typeof cfg.slug).toBe('string')
    expect(typeof cfg.apiKey).toBe('string')
  })
})

describe('lib/pakasir — webhook payload parsing', () => {
  it('parses a valid Pakasir webhook payload', () => {
    const payload = parseWebhookPayload({
      amount: 22000,
      order_id: '240910HDE7C9',
      project: 'depodomain',
      status: 'completed',
      payment_method: 'qris',
      completed_at: '2024-09-10T08:07:02.819+07:00',
    })
    expect(payload).toEqual({
      amount: 22000,
      order_id: '240910HDE7C9',
      project: 'depodomain',
      status: 'completed',
      payment_method: 'qris',
      completed_at: '2024-09-10T08:07:02.819+07:00',
    })
  })

  it('rejects malformed payloads', () => {
    expect(parseWebhookPayload(null)).toBeNull()
    expect(parseWebhookPayload('nope')).toBeNull()
    expect(parseWebhookPayload({ amount: 'x' })).toBeNull()
    expect(parseWebhookPayload({ amount: -5, order_id: 'o', project: 'p' })).toBeNull()
    expect(
      parseWebhookPayload({ amount: 100, order_id: '', project: 'p', status: 'x' })
    ).toBeNull()
  })

  it('lowercases and trims the status', () => {
    const payload = parseWebhookPayload({
      amount: 100,
      order_id: 'o1',
      project: 'p',
      status: ' COMPLETED ',
    })
    expect(payload?.status).toBe('completed')
  })
})

describe('lib/pakasir — order id generation', () => {
  it('appends a URL-safe random suffix to the invoice id', () => {
    const orderId = buildOrderId('INV-2001')
    expect(orderId).toMatch(/^INV-2001-[A-Z0-9]{4}$/)
  })

  it('sanitizes unsafe characters', () => {
    const orderId = buildOrderId('INV 2001/₩')
    expect(orderId).not.toContain(' ')
    expect(orderId).not.toContain('/')
  })
})

describe('lib/payment-store — per-tenant config (Model B)', () => {
  beforeEach(() => {
    resetPaymentStore()
  })

  it('stores and resolves a tenant provider config', () => {
    setTenantPaymentConfig({
      tenantId: 'tenant-acme',
      provider: 'pakasir',
      slug: 'acme-project',
      apiKey: 'sk_test_abc',
      settlementBankName: 'BCA',
      settlementBankAccount: '123456',
      settlementHolderName: 'PT Acme',
      isActive: true,
    })
    const cfg = getTenantPaymentConfig('tenant-acme')
    expect(cfg?.slug).toBe('acme-project')
    expect(cfg?.apiKey).toBe('sk_test_abc')
    expect(cfg?.settlementBankName).toBe('BCA')
  })

  it('isolates configs between tenants', () => {
    setTenantPaymentConfig({
      tenantId: 't1',
      provider: 'pakasir',
      slug: 'p1',
      isActive: true,
    })
    setTenantPaymentConfig({
      tenantId: 't2',
      provider: 'midtrans',
      serverKey: 'sk_2',
      isActive: true,
    })
    expect(getTenantPaymentConfig('t1')?.slug).toBe('p1')
    expect(getTenantPaymentConfig('t2')?.serverKey).toBe('sk_2')
    expect(listTenantPaymentConfigs()).toHaveLength(2)
  })

  it('overwrites config on upsert', () => {
    setTenantPaymentConfig({ tenantId: 't1', provider: 'pakasir', slug: 'p1', isActive: true })
    setTenantPaymentConfig({ tenantId: 't1', provider: 'pakasir', slug: 'p1v2', isActive: true })
    expect(getTenantPaymentConfig('t1')?.slug).toBe('p1v2')
    expect(listTenantPaymentConfigs()).toHaveLength(1)
  })

  it('returns undefined for unknown tenant', () => {
    expect(getTenantPaymentConfig('nope')).toBeUndefined()
  })
})

describe('lib/payment-store — in-memory payment store', () => {
  beforeEach(() => {
    resetPaymentStore()
  })

  it('creates a pending transaction keyed by invoice id with tenant + provider', () => {
    const tx = createTransaction({
      tenantId: 'tenant-acme',
      provider: 'pakasir',
      invoiceId: 'INV-2001',
      amount: 125000,
      method: 'qris',
      paymentLink: 'https://app.pakasir.com/pay/p/125000?order_id=INV-2001-X1',
    })
    expect(tx.status).toBe('pending')
    expect(tx.amount).toBe(125000)
    expect(tx.tenantId).toBe('tenant-acme')
    expect(tx.provider).toBe('pakasir')
    expect(getTransaction('INV-2001')).toBe(tx)
  })

  it('looks up transactions by order id (webhook path)', () => {
    const tx = createTransaction({
      tenantId: 'tenant-acme',
      provider: 'midtrans',
      invoiceId: 'INV-2001',
      amount: 5000,
      method: 'va',
      paymentLink: 'https://example.test/pay',
    })
    expect(getTransactionByOrderId(tx.orderId)?.invoiceId).toBe('INV-2001')
    expect(getTransactionByOrderId(tx.orderId)?.tenantId).toBe('tenant-acme')
    expect(getTransactionByOrderId(tx.orderId)?.provider).toBe('midtrans')
  })

  it('marks an invoice paid and records payment metadata', () => {
    const tx = createTransaction({
      tenantId: 'tenant-acme',
      provider: 'pakasir',
      invoiceId: 'INV-2002',
      amount: 3400500,
      method: 'qris',
      paymentLink: 'https://example.test/pay',
    })
    markInvoicePaid({
      invoiceId: 'INV-2002',
      paymentMethod: 'qris',
      completedAt: '2024-09-10T08:07:02Z',
    })
    expect(getTransaction('INV-2002')?.status).toBe('completed')
    expect(getTransaction('INV-2002')?.paymentMethod).toBe('qris')
    expect(getInvoiceStatus('INV-2002')).toBe('Paid')
    expect(tx.completedAt).toBe('2024-09-10T08:07:02Z')
  })

  it('returns undefined for unknown invoices', () => {
    expect(getTransaction('NOPE')).toBeUndefined()
    expect(getInvoiceStatus('NOPE')).toBeUndefined()
  })
})
