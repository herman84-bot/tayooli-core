import { NextResponse } from "next/server"
import type { NextRequest } from "next/server"
import { isProtectedPath } from "@/lib/auth/protected-routes"

const AUTH_COOKIE = "tayooli_auth"

/**
 * Server-side route guard. A missing session cookie redirects to /login
 * before any protected page renders. Presence check only — the Go backend
 * stays the source of truth (signature, expiry, logout revocation are
 * enforced on every API call, and the client layout re-checks /auth/me).
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl
  const authCookie = request.cookies.get(AUTH_COOKIE)?.value

  if (pathname === "/") {
    const isLandingDisabled =
      process.env.NEXT_PUBLIC_DISABLE_LANDING_PAGE !== "false"
    if (isLandingDisabled) {
      return NextResponse.redirect(new URL(authCookie ? "/dashboard" : "/login", request.url))
    }
    return NextResponse.next()
  }

  if (isProtectedPath(pathname) && !authCookie) {
    const res = NextResponse.redirect(new URL("/login", request.url))
    res.headers.set("Cache-Control", "no-store")
    return res
  }

  const res = NextResponse.next()
  // Back button must not serve a protected page from cache after logout.
  if (isProtectedPath(pathname)) res.headers.set("Cache-Control", "no-store")
  return res
}

// Must stay in sync with PROTECTED_PREFIXES (enforced by __tests__/protected-routes.test.ts).
export const config = {
  matcher: [
    "/",
    "/dashboard/:path*",
    "/products/:path*",
    "/pos/:path*",
    "/wms/:path*",
    "/settings/:path*",
    "/help/:path*",
    "/approvals/:path*",
    "/accounting/:path*",
    "/billing/:path*",
    "/customers/:path*",
    "/sales-invoices/:path*",
    "/sales-orders/:path*",
  ],
}
