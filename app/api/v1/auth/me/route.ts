import { NextRequest, NextResponse } from "next/server"
import { demoUserFromRequest, proxyAuth } from "@/lib/auth/auth-proxy"
import { backendBaseUrl, demoAuthEnabled } from "@/lib/auth/demo-auth"

export const dynamic = "force-dynamic"

/**
 * GET /api/v1/auth/me
 *
 * Returns the current session user:
 *  - backend configured → proxy (relays the `tayooli_auth` cookie)
 *  - demo fallback → verifies the signed demo cookie
 *
 * Response: 200 { user } | 401 unauthenticated
 */
export async function GET(request: NextRequest) {
  if (backendBaseUrl()) {
    // A backend is configured → it is the source of truth. Relay the result
    // as-is. A stale/fake local cookie is NOT accepted here: passing a demo
    // cookie to a real backend only produces a 401 + empty data downstream,
    // which hides the real problem (see README.md: dashboard must be real).
    return proxyAuth("/api/v1/auth/me", request)
  }

  // Demo fallback (used when no backend is configured, or proxy failed).
  if (process.env.NODE_ENV === "production" && process.env.AUTH_DEMO !== "true") {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 })
  }

  const user = demoUserFromRequest(request)
  if (!user) {
    return NextResponse.json({ error: "unauthenticated" }, { status: 401 })
  }

  return NextResponse.json({ user })
}
