import { createHmac, randomBytes, timingSafeEqual } from "node:crypto"

/**
 * Demo auth fallback for the Next.js preview.
 *
 * The Go API (auth via POST /api/v1/auth/login with a `tayooli_auth` cookie)
 * is not running inside the Next.js preview sandbox. These pure helpers back
 * the same relative `/api/v1/auth/*` endpoints with a small in-memory demo
 * session so the app can be exercised end-to-end without the backend.
 *
 * This module must stay free of `next/server` imports so it can be unit
 * tested in jest (route handlers live in `app/api/v1/auth/*` and import the
 * Next-specific proxy helper from `lib/auth/auth-proxy.ts`).
 *
 * Safety:
 *  - Demo mode is ONLY active when no backend base URL is configured
 *    (NEXT_PUBLIC_API_URL / API_BASE_URL) AND we are not in production
 *    (unless explicitly opted in with AUTH_DEMO=true).
 *  - Sessions are HMAC-signed with a per-boot secret (no forged tokens,
 *    invalidated on restart). No real data, no secrets.
 */

export const AUTH_COOKIE = "tayooli_auth"

const TENANT_ID = "550e8400-e29b-41d4-a716-446655440000"

interface DemoUser {
  id: string
  tenantId: string
  email: string
  password: string
  role: string
}

/** Mirrors backend/go-core/migrations/012_e2e_test_users.sql */
export const DEMO_USERS: DemoUser[] = [
  {
    id: "550e8400-e29b-41d4-a716-446655440001",
    tenantId: TENANT_ID,
    email: "admin@test.com",
    password: "password123",
    role: "admin",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440101",
    tenantId: TENANT_ID,
    email: "accountant@test.com",
    password: "password123",
    role: "accountant",
  },
  {
    id: "550e8400-e29b-41d4-a716-446655440102",
    tenantId: TENANT_ID,
    email: "approver@test.com",
    password: "password123",
    role: "approver",
  },
]

/**
 * Per-boot random secret shared by all auth route handlers (module cache).
 * Tokens can't be forged and are invalidated on server restart.
 */
const globalForDemo = globalThis as unknown as {
  __tayooliDemoSecret?: string
}
const SESSION_SECRET = process.env.AUTH_DEMO_SECRET ?? (globalForDemo.__tayooliDemoSecret ??= randomBytes(32).toString("hex"))

const SESSION_TTL_SECONDS = 60 * 60 // 1 hour, mirrors the Go cookie MaxAge

/** Backend base URL when a real Go API is configured, otherwise null. */
export function backendBaseUrl(): string | null {
  const raw = process.env.NEXT_PUBLIC_API_URL ?? process.env.API_BASE_URL ?? process.env.BACKEND_URL
  const trimmed = raw?.trim()
  return trimmed ? trimmed.replace(/\/+$/, "") : null
}

/**
 * Demo mode is enabled when:
 *  - no backend is configured, AND
 *  - not in production, unless explicitly opted in via AUTH_DEMO=true
 */
export function demoAuthEnabled(): boolean {
  if (backendBaseUrl()) return false
  if (process.env.AUTH_DEMO === "true") return true
  return process.env.NODE_ENV !== "production"
}

export function verifyDemoCredentials(
  email: string,
  password: string
): DemoUser | null {
  const user = DEMO_USERS.find((u) => u.email.toLowerCase() === email.toLowerCase())
  if (!user || user.password !== password) return null
  return user
}

export function publicUser(user: DemoUser): {
  id: string
  email: string
  role: string
} {
  return { id: user.id, email: user.email, role: user.role }
}

interface DemoSessionPayload {
  sub: string
  email: string
  role: string
  tenantId: string
  exp: number
}

function sign(input: string): string {
  return createHmac("sha256", SESSION_SECRET).update(input).digest("hex")
}

function encode(payload: DemoSessionPayload): string {
  const data = Buffer.from(JSON.stringify(payload)).toString("base64url")
  return `${data}.${sign(data)}`
}

export function signDemoToken(user: DemoUser): string {
  const payload: DemoSessionPayload = {
    sub: user.id,
    email: user.email,
    role: user.role,
    tenantId: user.tenantId,
    exp: Math.floor(Date.now() / 1000) + SESSION_TTL_SECONDS,
  }
  return encode(payload)
}

export function verifyDemoToken(
  token: string | undefined
): { id: string; email: string; role: string; tenantId: string } | null {
  if (!token) return null
  const parts = token.split(".")
  if (parts.length !== 2) return null
  const [data, sig] = parts
  // Strict: reject any token with a non-64-hex signature (e.g. junk appended
  // after a valid token) instead of silently truncating it.
  if (!data || !sig || !/^[0-9a-f]{64}$/.test(sig)) return null

  const expected = sign(data)
  const actual = Buffer.from(sig, "hex")
  const expectedBuf = Buffer.from(expected, "hex")
  if (actual.length !== expectedBuf.length || !timingSafeEqual(actual, expectedBuf)) {
    return null
  }

  try {
    const payload = JSON.parse(
      Buffer.from(data, "base64url").toString("utf8")
    ) as DemoSessionPayload
    if (typeof payload.exp !== "number" || payload.exp < Math.floor(Date.now() / 1000)) {
      return null
    }
    return {
      id: payload.sub,
      email: payload.email,
      role: payload.role,
      tenantId: payload.tenantId,
    }
  } catch {
    return null
  }
}
