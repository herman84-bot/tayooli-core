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
    const proxyResult = await proxyAuth("/api/v1/auth/me", request)
    // If proxy succeeded (not 502), relay it. Otherwise fall through to demo.
    if (proxyResult.status !== 502) {
      // In dev mode, if remote backend returned 401, check if we have a valid demo cookie session
      if (process.env.NODE_ENV !== "production" && proxyResult.status === 401) {
        const demoUser = demoUserFromRequest(request)
        if (demoUser) {
          return NextResponse.json({ user: demoUser })
        }
      }
      return proxyResult
    }
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
