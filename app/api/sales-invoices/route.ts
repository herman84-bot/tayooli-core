import { NextRequest, NextResponse } from 'next/server';
import { firstForwardedClientIp } from '@/lib/api/forward-headers';

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8081';

// Proxy ke backend Go: meneruskan cookie/authorization dari request asli agar
// auth middleware backend (session/tenant) dapat memvalidasi pengguna.
async function proxy(req: NextRequest, path: string, init?: RequestInit) {
  const headers = new Headers(init?.headers);
  const cookie = req.headers.get('cookie');
  if (cookie) headers.set('cookie', cookie);
  const authorization = req.headers.get('authorization');
  if (authorization) headers.set('authorization', authorization);
  // Forward the real client IP (first hop only) for the backend rate limiter.
  const clientIp = firstForwardedClientIp(req.headers.get('x-forwarded-for'));
  if (clientIp) headers.set('x-forwarded-for', clientIp);

  const res = await fetch(`${BACKEND_URL}/api/v1${path}`, { ...init, headers });
  const body = res.status === 204 ? null : await res.text();

  return new NextResponse(body, {
    status: res.status,
    statusText: res.statusText,
    headers: {
      'content-type': res.headers.get('content-type') ?? 'application/json',
    },
  });
}

export async function GET(req: NextRequest) {
  return proxy(req, '/sales-invoices');
}

export async function POST(req: NextRequest) {
  return proxy(req, '/sales-invoices', {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(await req.json()),
  });
}
