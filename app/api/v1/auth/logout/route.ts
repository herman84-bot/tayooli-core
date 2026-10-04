import { NextRequest, NextResponse } from "next/server"
import { proxyAuth } from "@/lib/auth/auth-proxy"
import {
  AUTH_COOKIE,
  backendBaseUrl,
  demoAuthEnabled,
} from "@/lib/auth/demo-auth"

export const dynamic = "force-dynamic"

/**
 * POST /api/v1/auth/logout
 *
 * - backend configured → proxy
 * - demo fallback → clears the demo `tayooli_auth` cookie
 *
 * Response: 200 { ok: true }
 */
export async function POST(request: NextRequest) {
  if (backendBaseUrl()) {
    return proxyAuth("/api/v1/auth/logout", request)
  }

  const res = NextResponse.json({ ok: true })
  if (demoAuthEnabled()) {
    res.cookies.set(AUTH_COOKIE, "", {
      httpOnly: true,
      path: "/",
      sameSite: "lax",
      maxAge: 0,
    })
  }
  return res
}
