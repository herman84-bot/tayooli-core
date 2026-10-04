/**
 * Demo Mode & Billing Toggle Configuration
 *
 * When DISABLE_BILLING is true (default: true for client demo):
 * - Trial countdown banner is completely disabled.
 * - Sidebar Billing navigation is hidden.
 * - Settings Billing tab is hidden.
 * - Workspace onboarding redirects straight to /dashboard (bypassing /onboarding/trial).
 * - Trial page immediately redirects to /dashboard.
 * - System behaves as unlimited Enterprise license with zero paywalls.
 *
 * To RE-ENABLE billing & trial commercial system:
 * Set NEXT_PUBLIC_DISABLE_BILLING="false" in .env.local OR change DEFAULT_DISABLE_BILLING to false.
 */

const DEFAULT_DISABLE_BILLING = true

export const DISABLE_BILLING =
  typeof process !== 'undefined' && process.env.NEXT_PUBLIC_DISABLE_BILLING !== undefined
    ? process.env.NEXT_PUBLIC_DISABLE_BILLING === 'true'
    : DEFAULT_DISABLE_BILLING

export const IS_DEMO_MODE =
  typeof process !== 'undefined' && process.env.NEXT_PUBLIC_DEMO_MODE !== undefined
    ? process.env.NEXT_PUBLIC_DEMO_MODE !== 'false'
    : true
