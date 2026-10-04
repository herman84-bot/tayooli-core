import { NextRequest, NextResponse } from 'next/server';
import { firstForwardedClientIp } from '@/lib/api/forward-headers';

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8081';

async function proxy(req: NextRequest, method: string, path: string, body?: unknown) {
  const clientIp = firstForwardedClientIp(req.headers.get('x-forwarded-for'));
  const backendRes = await fetch(`${BACKEND_URL}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(req.headers.get('cookie')
        ? { cookie: req.headers.get('cookie') ?? '' }
        : {}),
      ...(req.headers.get('authorization')
        ? { authorization: req.headers.get('authorization') ?? '' }
        : {}),
      ...(clientIp ? { 'x-forwarded-for': clientIp } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
    cache: 'no-store',
  });

  const text = await backendRes.text();
  const data = text ? JSON.parse(text) : null;

  if (!backendRes.ok) {
    const message =
      (data && typeof data === 'object' && 'error' in data && typeof data.error === 'string'
        ? data.error
        : null) ?? backendRes.statusText;
    return NextResponse.json({ error: message }, { status: backendRes.status });
  }

  return NextResponse.json(data);
}

export async function GET(req: NextRequest) {
  try {
    return await proxy(req, 'GET', '/api/v1/sales-orders');
  } catch (error) {
    console.error('Failed to proxy GET /api/v1/sales-orders', error);
    return NextResponse.json(
      { error: 'Sales orders service is unavailable' },
      { status: 502 }
    );
  }
}

export async function POST(req: NextRequest) {
  try {
    const body = await req.json();
    return await proxy(req, 'POST', '/api/v1/sales-orders', body);
  } catch (error) {
    console.error('Failed to proxy POST /api/v1/sales-orders', error);
    return NextResponse.json(
      { error: 'Sales orders service is unavailable' },
      { status: 502 }
    );
  }
}
