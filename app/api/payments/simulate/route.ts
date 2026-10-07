import { NextRequest, NextResponse } from 'next/server'
import { firstForwardedClientIp } from '@/lib/api/forward-headers'

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8081'

/**
 * Demo-only payment simulation.
 *
 * Reached by the customer-facing demo checkout page (/payments/demo/[orderId]),
 * which is what the QR of a demo-mode QRIS intent encodes. It forwards to the
 * backend's POS simulate endpoint with the caller's session, so it can never
 * settle a payment by itself: the backend owns the decision and refuses the
 * call as soon as the tenant has real gateway credentials (ADR-008 Model B).
 *
 * The route exists (rather than the page calling the API directly) so the
 * page only needs an origin-relative URL and the session cookies stay with the
 * proxy layer like every other backend call.
 */
export async function POST(req: NextRequest) {
  const body = await req.json().catch(() => ({} as Record<string, unknown>))
  const orderId = String(body?.orderId ?? body?.order_id ?? '').trim()
  if (!orderId) {
    return NextResponse.json({ error: 'order_id wajib diisi' }, { status: 400 })
  }

  const headers = new Headers({ 'content-type': 'application/json' })
  const cookie = req.headers.get('cookie')
  if (cookie) headers.set('cookie', cookie)
  const authorization = req.headers.get('authorization')
  if (authorization) headers.set('authorization', authorization)
  const clientIp = firstForwardedClientIp(req.headers.get('x-forwarded-for'))
  if (clientIp) headers.set('x-forwarded-for', clientIp)

  let res: Response
  try {
    res = await fetch(
      `${BACKEND_URL}/api/v1/pos/payments/${encodeURIComponent(orderId)}/simulate`,
      { method: 'POST', headers, signal: AbortSignal.timeout(10_000) }
    )
  } catch {
    return NextResponse.json({ error: 'backend unreachable' }, { status: 502 })
  }

  const text = res.status === 204 ? null : await res.text()
  return new NextResponse(text, {
    status: res.status,
    statusText: res.statusText,
    headers: {
      'content-type': res.headers.get('content-type') ?? 'application/json',
    },
  })
}
