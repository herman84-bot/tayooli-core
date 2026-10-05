import { NextRequest, NextResponse } from "next/server"
import { mintBackendToken, proxyAuth } from "@/lib/auth/auth-proxy"
import {
  AUTH_COOKIE,
  backendBaseUrl,
  demoAuthEnabled,
  publicUser,
  signDemoToken,
  verifyDemoCredentials,
} from "@/lib/auth/demo-auth"

export const dynamic = "force-dynamic"

interface LoginBody {
  email?: string
  password?: string
}

/**
 * POST /api/v1/auth/login
 *
 * Behavior:
 *  1. If a Go backend is configured (NEXT_PUBLIC_API_URL / API_BASE_URL),
 *     proxy the request and relay the session cookie.
 *  2. Otherwise serve the demo fallback (only outside production unless
 *     AUTH_DEMO=true): validates the seed users (admin/accountant/approver @
 *     test.com, password `password123`) and sets the same `tayooli_auth`
 *     HttpOnly cookie the backend would set.
 *
 * Response: 200 { user: { id, email, role } } | 401 invalid credentials
 */
export async function POST(request: NextRequest) {
  let body: LoginBody = {}
  try {
    body = (await request.json()) as LoginBody
  } catch {
    return NextResponse.json({ error: "invalid request body" }, { status: 400 })
  }

  const email = typeof body.email === "string" ? body.email.trim() : ""
  const password = typeof body.password === "string" ? body.password : ""
  if (!email || !password) {
    return NextResponse.json(
      { error: "email and password are required" },
      { status: 400 }
    )
  }
  if (password.length < 8) {
    return NextResponse.json(
      { error: "password must be at least 8 characters" },
      { status: 400 }
    )
  }

  // 1) Real backend configured → proxy, and relay its verdict as-is.
  // No silent demo session here: minting a local demo cookie when the real
  // backend rejects the login only produces a stuck state ("logged in" UI
  // with every subsequent data call 401ing). The user must fix the real
  // credentials instead.
  if (backendBaseUrl()) {
    return proxyAuth("/api/v1/auth/login", request, body)
  }

  // 2) Demo fallback (used when no backend is configured, or proxy failed).
  //    Allow demo auth even when a backend URL is set — the proxy may have
  //    failed (unreachable), so we should not block the fallback.
  if (process.env.NODE_ENV === "production" && process.env.AUTH_DEMO !== "true") {
    return NextResponse.json(
      { error: "auth backend not configured" },
      { status: 503 }
    )
  }

  const user = verifyDemoCredentials(email, password)
  if (!user) {
    return NextResponse.json({ error: "invalid credentials" }, { status: 401 })
  }

  // The demo session cookie is ALWAYS set (even when a backend JWT is also
  // minted below) so `/api/v1/auth/me` keeps restoring the session after a
  // hard navigation reload.
  const setDemoCookie = (r: NextResponse): NextResponse => {
    r.cookies.set(AUTH_COOKIE, signDemoToken(user), {
      httpOnly: true,
      path: "/",
      sameSite: "lax",
      maxAge: 60 * 60,
      secure: process.env.NODE_ENV === "production",
    })
    return r
  }

  // Best-effort: if a real Go backend happens to be reachable (e.g. local
  // E2E), also mint its JWT and return it so the client can call the API
  // with `Authorization: Bearer` via apiClient. Harmless when unreachable.
  const minted = await mintBackendToken(email, password)
  if (minted) {
    return setDemoCookie(
      NextResponse.json({
        user: publicUser(user),
        token: minted.token,
        tenantId: minted.tenantId,
      })
    )
  }
  return setDemoCookie(NextResponse.json({ user: publicUser(user) }))
}
