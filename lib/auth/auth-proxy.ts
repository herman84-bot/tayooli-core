import { NextRequest, NextResponse } from "next/server"
import { firstForwardedClientIp } from "@/lib/api/forward-headers"
import {
  AUTH_COOKIE,
  backendBaseUrl,
  verifyDemoToken,
} from "@/lib/auth/demo-auth"

/**
 * Next.js-specific auth helpers used by the `/api/v1/auth/*` route handlers.
 * Kept separate from `lib/auth/demo-auth.ts` so the pure demo helpers stay
 * unit-testable in jest (importing `next/server` fails outside the runtime).
 */

/** Same default the browser-side apiClient uses when no env override is set. */
export const DEFAULT_API_BASE = "http://localhost:8081"

/** Extracts the `tayooli_auth=<jwt>` value from raw Set-Cookie header(s). */
export function jwtFromSetCookie(setCookies: string[]): string | null {
  for (const c of setCookies) {
    const m = c.match(/tayooli_auth=([^;]+)/)
    if (m) return m[1]
  }
  return null
}

/** Reads `tenant_id` out of a JWT payload (JWT is base64url JSON). */
export function tenantIdFromToken(token: string): string | null {
  try {
    const payload = JSON.parse(
      Buffer.from(token.split(".")[1] ?? "", "base64url").toString("utf8")
    ) as { tenant_id?: string }
    return payload.tenant_id ?? null
  } catch {
    return null
  }
}

/**
 * Best-effort: exchanges demo credentials for a real backend JWT.
 * Used when the Go API is running but not configured via env (local E2E).
 * Returns null when the backend is unreachable or rejects the login.
 */
export async function mintBackendToken(
  email: string,
  password: string
): Promise<{ token: string; tenantId: string } | null> {
  const base = backendBaseUrl() ?? DEFAULT_API_BASE
  try {
    const res = await fetch(`${base}/api/v1/auth/login`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ email, password }),
      cache: "no-store",
      redirect: "manual",
      // Never let the backend mint delay the login response: fail fast so
      // the demo session still completes the navigation in time.
      signal: AbortSignal.timeout(10000),
    })
    if (!res.ok) return null
    const setCookies = res.headers.getSetCookie
      ? res.headers.getSetCookie()
      : res.headers.get("set-cookie")
        ? [res.headers.get("set-cookie") as string]
        : []
    const token = jwtFromSetCookie(setCookies)
    if (!token) return null
    return { token, tenantId: tenantIdFromToken(token) ?? "" }
  } catch {
    return null
  }
}

/**
 * Proxies an auth request to the configured Go backend, relaying the
 * `Set-Cookie` headers so the browser keeps the real session cookie.
 * Returns a 502 when the backend is unreachable.
 */
export async function proxyAuth(
  path: string,
  request: NextRequest,
  parsedBody?: unknown
): Promise<Response> {
  const base = backendBaseUrl()
  if (!base) {
    return NextResponse.json({ error: "auth backend not configured" }, { status: 503 })
  }

  const cookieHeader = request.headers.get("cookie")
  let requestBody: string | undefined = undefined
  if (request.method !== "GET" && request.method !== "HEAD") {
    if (parsedBody !== undefined) {
      requestBody = JSON.stringify(parsedBody)
    } else {
      try {
        requestBody = await request.text()
      } catch {
        requestBody = undefined
      }
    }
  }

  const clientIp = firstForwardedClientIp(request.headers.get("x-forwarded-for"))

  try {
    const res = await fetch(`${base}${path}`, {
      method: request.method,
      headers: {
        "content-type": "application/json",
        ...(cookieHeader ? { cookie: cookieHeader } : {}),
        ...(clientIp ? { "x-forwarded-for": clientIp } : {}),
      },
      body: requestBody,
      cache: "no-store",
      redirect: "manual",
      signal: AbortSignal.timeout(10000),
    })

    const setCookies = res.headers.getSetCookie
      ? res.headers.getSetCookie()
      : res.headers.get("set-cookie")
        ? [res.headers.get("set-cookie") as string]
        : []

    // When this is a login that set a real session cookie, surface the JWT in
    // the JSON body too so the client can attach it as `Authorization: Bearer`
    // on direct apiClient calls to the Go API.
    const jwt = jwtFromSetCookie(setCookies)
    let text = await res.text()
    if (jwt) {
      try {
        const body = JSON.parse(text) as Record<string, unknown>
        body.token = jwt
        body.tenantId = tenantIdFromToken(jwt) ?? ""
        text = JSON.stringify(body)
      } catch {
        // Non-JSON upstream body: leave as-is.
      }
    }

    const out = new Response(text, {
      status: res.status,
      headers: {
        "content-type": res.headers.get("content-type") ?? "application/json",
      },
    })
    for (const c of setCookies) {
      out.headers.append("set-cookie", c)
    }
    return out
  } catch (err) {
    return NextResponse.json(
      { error: `auth backend unreachable: ${err instanceof Error ? err.message : String(err)}` },
      { status: 502 }
    )
  }
}

/** Returns the demo-mode auth user from the request cookie, or null. */
export function demoUserFromRequest(request: NextRequest): {
  id: string
  email: string
  role: string
  tenantId: string
} | null {
  return verifyDemoToken(request.cookies.get(AUTH_COOKIE)?.value)
}
