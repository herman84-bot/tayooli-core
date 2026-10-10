/**
 * Route prefixes that require a session. Kept free of `next/server` imports so
 * jest can unit-test it (middleware.ts imports this).
 */
export const PROTECTED_PREFIXES = [
  "/dashboard",
  "/products",
  "/pos",
  "/wms",
  "/settings",
  "/help",
  "/approvals",
  "/accounting",
  "/billing",
  "/customers",
  "/sales-invoices",
  "/sales-orders",
  "/onboarding",
] as const
// Intentionally public: /payments/* (gateway return pages), /login, /register,
// /forgot-password, /reset-password, /verify-email.

export function isProtectedPath(pathname: string): boolean {
  return PROTECTED_PREFIXES.some((p) => pathname === p || pathname.startsWith(p + "/"))
}
