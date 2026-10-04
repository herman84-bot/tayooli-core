/**
 * Payment provider registry — the single place that knows which gateways are
 * available. Route handlers, the webhook receiver and the UI resolve a
 * provider by id through this module, so adding a gateway later is a matter
 * of implementing `PaymentProvider` and registering it here.
 */
import { pakasirProvider } from './providers/pakasir'
import { midtransProvider } from './providers/midtrans'
import { getTenantPaymentConfig } from '@/lib/payment-store'
import type {
  PaymentProvider,
  PaymentProviderConfig,
  PaymentProviderId,
} from './types'

export const PAYMENT_PROVIDERS: Record<PaymentProviderId, PaymentProvider> = {
  pakasir: pakasirProvider,
  midtrans: midtransProvider,
}

export const PAYMENT_PROVIDER_IDS = Object.keys(PAYMENT_PROVIDERS) as PaymentProviderId[]

export function isPaymentProviderId(value: unknown): value is PaymentProviderId {
  return typeof value === 'string' && value in PAYMENT_PROVIDERS
}

export function getProvider(providerId: PaymentProviderId): PaymentProvider {
  const provider = PAYMENT_PROVIDERS[providerId]
  if (!provider) {
    throw new Error(`Unknown payment provider: ${providerId}`)
  }
  return provider
}

/**
 * Resolve the provider config for a tenant:
 *  1. per-tenant config stored in the DB mirror (Model B), else
 *  2. the provider's process-level env fallback (single-tenant / demo).
 */
export function resolveProviderConfig(
  tenantId: string,
  providerId: PaymentProviderId
): { config: PaymentProviderConfig; demo: boolean } {
  const tenantCfg = getTenantPaymentConfig(tenantId)
  if (tenantCfg && tenantCfg.isActive !== false) {
    // The stored config may be for any provider; keep only the fields this
    // provider cares about by re-hydrating a fresh config object.
    const config: PaymentProviderConfig = {
      provider: providerId,
      slug: tenantCfg.slug,
      apiKey: tenantCfg.apiKey,
      serverKey: tenantCfg.serverKey,
      clientKey: tenantCfg.clientKey,
      isProduction: tenantCfg.isProduction,
      baseUrl: tenantCfg.baseUrl,
      apiBaseUrl: tenantCfg.apiBaseUrl,
    }
    // Only treat the tenant config as usable when it belongs to this provider
    // (has at least the provider's primary credential field).
    const provider = getProvider(providerId)
    const hasProviderFields =
      providerId === 'pakasir'
        ? Boolean(config.slug || config.apiKey)
        : Boolean(config.serverKey || config.clientKey)
    if (hasProviderFields) {
      return { config, demo: !provider.isConfigured(config) }
    }
  }
  const provider = getProvider(providerId)
  const config = provider.globalConfig()
  return { config, demo: !provider.isConfigured(config) }
}
