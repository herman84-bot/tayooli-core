/**
 * In-memory payment store for the multi-gateway demo vertical slice.
 *
 * The ERP's production persistence is PostgreSQL via the Go backend, but the
 * sandbox cannot run Go + Postgres + Kafka, so this module simulates the
 * persistence layer with a process-global Map (survives HMR via globalThis).
 *
 * Multi-tenant note: mirrors `tenant_payment_configs` (see schema_pakasir.sql).
 * Every transaction records its owning tenant AND the payment provider, so
 * webhooks can be isolated and verified per tenant per gateway (Model B —
 * money settles directly to the tenant's bank).
 */

import type {
  PaymentMethod,
  PaymentProviderConfig,
  PaymentProviderId,
  PaymentTransaction,
} from '@/lib/payments/types'

export interface TenantPaymentConfig extends PaymentProviderConfig {
  tenantId: string
  settlementBankName?: string
  settlementBankAccount?: string
  settlementHolderName?: string
  gatewayFeePercent?: number
  isActive: boolean
}

interface PaymentStore {
  /** tenantId -> provider config */
  tenantConfigs: Map<string, TenantPaymentConfig>
  /** invoiceId -> active transaction */
  transactions: Map<string, PaymentTransaction>
  /** invoiceId -> last recorded status from webhooks */
  invoiceStatus: Map<string, 'Unpaid' | 'Paid' | 'Overdue'>
}

const GLOBAL_KEY = '__payment_demo_store__' as const

function createStore(): PaymentStore {
  return {
    tenantConfigs: new Map<string, TenantPaymentConfig>(),
    transactions: new Map<string, PaymentTransaction>(),
    invoiceStatus: new Map<string, 'Unpaid' | 'Paid' | 'Overdue'>(),
  }
}

function getStore(): PaymentStore {
  const g = globalThis as typeof globalThis & { [GLOBAL_KEY]?: PaymentStore }
  if (!g[GLOBAL_KEY]) {
    g[GLOBAL_KEY] = createStore()
  }
  return g[GLOBAL_KEY]
}

// ── Per-tenant provider config ───────────────────────────────────────────────

/** Upsert a tenant's payment provider config. */
export function setTenantPaymentConfig(config: TenantPaymentConfig): void {
  getStore().tenantConfigs.set(config.tenantId, config)
}

/** Resolve a tenant's stored provider config (may be for any provider). */
export function getTenantPaymentConfig(
  tenantId: string
): TenantPaymentConfig | undefined {
  return getStore().tenantConfigs.get(tenantId)
}

/** List all configured tenants (used by the demo config admin page). */
export function listTenantPaymentConfigs(): TenantPaymentConfig[] {
  return Array.from(getStore().tenantConfigs.values())
}

// ── Transactions ─────────────────────────────────────────────────────────────

export function createTransaction(params: {
  tenantId: string
  provider: PaymentProviderId
  invoiceId: string
  amount: number
  method: PaymentMethod
  paymentLink: string
}): PaymentTransaction {
  const store = getStore()
  const tx: PaymentTransaction = {
    tenantId: params.tenantId,
    provider: params.provider,
    invoiceId: params.invoiceId,
    orderId: `${params.invoiceId}-${Date.now().toString(36).toUpperCase()}`,
    amount: Math.round(params.amount),
    method: params.method,
    paymentLink: params.paymentLink,
    status: 'pending',
    createdAt: new Date().toISOString(),
  }
  store.transactions.set(params.invoiceId, tx)
  return tx
}

export function getTransaction(invoiceId: string): PaymentTransaction | undefined {
  return getStore().transactions.get(invoiceId)
}

/** Look up a transaction by its order id (used by webhooks). */
export function getTransactionByOrderId(
  orderId: string
): PaymentTransaction | undefined {
  const store = getStore()
  for (const tx of store.transactions.values()) {
    if (tx.orderId === orderId) {
      return tx
    }
  }
  return undefined
}

/** Mark an invoice as paid (called from the webhook handler after verify). */
export function markInvoicePaid(params: {
  invoiceId: string
  paymentMethod?: string
  completedAt?: string
}): void {
  const store = getStore()
  const existing = store.transactions.get(params.invoiceId)
  if (existing) {
    existing.status = 'completed'
    existing.paymentMethod = params.paymentMethod
    existing.completedAt = params.completedAt ?? new Date().toISOString()
  }
  store.invoiceStatus.set(params.invoiceId, 'Paid')
}

export function getInvoiceStatus(
  invoiceId: string
): 'Unpaid' | 'Paid' | 'Overdue' | undefined {
  return getStore().invoiceStatus.get(invoiceId)
}

/** Reset the store (used by tests). */
export function resetPaymentStore(): void {
  const g = globalThis as typeof globalThis & { [GLOBAL_KEY]?: PaymentStore }
  delete g[GLOBAL_KEY]
}
