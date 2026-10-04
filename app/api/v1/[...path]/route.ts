import { NextRequest, NextResponse } from 'next/server';
import { getDemoResponse } from '@/lib/api/demo-data';
import { extractClientIp } from '@/lib/api/forward-headers';

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8081';

// Proxy catch-all untuk /api/v1/* → backend Go.
// Menangani auth (login/me/logout), invoices, approvals, accounting, dll.
// Meneruskan cookie + authorization dari request asli agar sesi backend valid.
// Ini adalah satu-satunya pintu frontend ke API Go; tanpa ini semua
// panggilan /api/v1/* di frontend akan 404.
export async function GET(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return proxy(req, await joinPath(params));
}

export async function POST(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  const path = await joinPath(params);
  const body = await req.json().catch(() => ({}));
  return proxy(req, path, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export async function PUT(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return proxy(req, await joinPath(params), {
    method: 'PUT',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(await req.json().catch(() => ({}))),
  });
}

export async function PATCH(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return proxy(req, await joinPath(params), {
    method: 'PATCH',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(await req.json().catch(() => ({}))),
  });
}

export async function DELETE(req: NextRequest, { params }: { params: Promise<{ path: string[] }> }) {
  return proxy(req, await joinPath(params), { method: 'DELETE' });
}

async function joinPath(params: Promise<{ path: string[] }>): Promise<string> {
  const { path } = await params;
  const safeSegments = (path ?? []).filter(
    (seg) => seg !== '..' && seg !== '.' && !seg.includes('/') && !seg.includes('\\')
  );
  return `/${safeSegments.join('/')}`;
}

/**
 * Handle inference demo responses when the Go backend is unreachable.
 * Returns a mock response for inference endpoints, or null if not an inference path.
 */
function getInferenceDemoResponse(path: string, method: string, body?: Record<string, unknown>): unknown | null {
  if (process.env.NODE_ENV === 'production') return null;

  // POST /api/v1/inference/ingest — return queued job
  if (path === '/inference/ingest' && method === 'POST' && body?.invoice_id) {
    return {
      job_id: body.invoice_id,
      status: 'queued',
    };
  }

  // GET /api/v1/inference/status/<id> — return completed result
  const statusMatch = path.match(/^\/inference\/status\/(.+)$/);
  if (statusMatch && method === 'GET') {
    const invoiceId = statusMatch[1];
    // Simulate anomaly detection based on invoice_id hash
    const hash = invoiceId.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0);
    const score = hash % 100;
    const detected = score >= 50;
    const glAccounts: Record<string, string> = {
      0: '5101 — Salaries',
      1: '5102 — Utilities',
      2: '5103 — Rent',
      3: '5104 — Office Supplies',
      4: '5105 — Marketing',
      5: '5106 — Transportation',
      6: '5107 — Insurance',
      7: '5108 — Professional Services',
      8: '5109 — Depreciation',
      9: '5110 — Miscellaneous Expense',
    };
    const glKey = score % 10;

    return {
      invoice_id: invoiceId,
      status: 'completed',
      anomaly_score: score,
      suggested_gl_account: glAccounts[glKey] ?? '5110 — Miscellaneous Expense',
      error: null,
    };
  }

  return null;
}

function parseBody(body: BodyInit | null | undefined): Record<string, unknown> | undefined {
  if (!body || typeof body !== 'string') return undefined;
  try {
    return JSON.parse(body);
  } catch {
    return undefined;
  }
}

async function proxy(req: NextRequest, path: string, init?: RequestInit) {
  const headers = new Headers(init?.headers);
  const cookie = req.headers.get('cookie');
  if (cookie) headers.set('cookie', cookie);
  const authorization = req.headers.get('authorization');
  if (authorization) headers.set('authorization', authorization);
  // Forward the real client IP (preferring X-Real-IP set by Nginx) so the backend's
  // trusted-proxy rate limiter keys on the actual client instead of 127.0.0.1.
  const clientIp = extractClientIp(req.headers.get('x-real-ip'), req.headers.get('x-forwarded-for'));
  if (clientIp) headers.set('x-forwarded-for', clientIp);

  let res: Response;
  try {
    // Teruskan query string asli (mis. ?page=2&per_page=20) agar pagination,
    // filter, dan search di backend tetap berfungsi.
    res = await fetch(`${BACKEND_URL}/api/v1${path}${req.nextUrl.search}`, {
      ...init,
      headers,
      signal: AbortSignal.timeout(10000),
    });
  } catch {
    // Backend unreachable — try inference demo, then generic demo data fallback.
    const method = init?.method ?? 'GET';
    const parsedBody = parseBody(init?.body);
    const inferDemo = getInferenceDemoResponse(path, method, parsedBody);
    if (inferDemo !== null) {
      return NextResponse.json(inferDemo);
    }
    const demo = getDemoResponse(path, req.nextUrl.search, parsedBody, method);
    if (demo !== null) {
      return NextResponse.json(demo);
    }
    return NextResponse.json({ error: 'backend unreachable' }, { status: 502 });
  }

  // If backend returns error for inference endpoints, fall back to demo data.
  // This handles: demo JWT invalid (401), AI worker down (500), etc.
  if (res.status >= 400 && path.startsWith('/inference/')) {
    const parsedBody = parseBody(init?.body);
    const inferDemo = getInferenceDemoResponse(path, init?.method ?? 'GET', parsedBody);
    if (inferDemo !== null) {
      return NextResponse.json(inferDemo);
    }
  }

  // In dev mode with demo auth: if remote backend rejected demo cookie with 401,
  // return simulated success for mutations so dev UI testing works seamlessly.
  if (process.env.NODE_ENV !== 'production' && res.status === 401) {
    const method = init?.method ?? 'GET';
    const parsedBody = parseBody(init?.body);
    const demo = getDemoResponse(path, req.nextUrl.search, parsedBody, method);
    if (demo !== null) {
      return NextResponse.json(demo);
    }
    if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
      return NextResponse.json({ success: true, ...(parsedBody || {}) });
    }
  }

  const body = res.status === 204 ? null : await res.text();

  // Teruskan Set-Cookie dari backend (mis. lumina_auth) ke browser.
  const setCookie = res.headers.get('set-cookie');
  return new NextResponse(body, {
    status: res.status,
    statusText: res.statusText,
    headers: {
      'content-type': res.headers.get('content-type') ?? 'application/json',
      ...(setCookie ? { 'set-cookie': setCookie } : {}),
    },
  });
}
