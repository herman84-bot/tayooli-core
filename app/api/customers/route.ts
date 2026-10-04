import { NextRequest, NextResponse } from 'next/server';
import { firstForwardedClientIp } from '@/lib/api/forward-headers';

const BACKEND_URL = process.env.BACKEND_URL ?? 'http://localhost:8081';

// Proxy ke backend Go (/api/v1/customers). Semua error dari backend
// diteruskan apa adanya (status + message) agar frontend bisa menampilkan
// pesan yang sebenarnya, mis. 401 unauthenticated / 403 forbidden.
async function proxy(req: NextRequest): Promise<NextResponse> {
  const url = `${BACKEND_URL}/api/v1/customers`;

  const headers = new Headers();
  // Teruskan sesi: cookie JWT HttpOnly (preferred) dan Authorization header.
  const cookie = req.headers.get('cookie');
  if (cookie) headers.set('cookie', cookie);
  const authorization = req.headers.get('authorization');
  if (authorization) headers.set('authorization', authorization);
  const contentType = req.headers.get('content-type');
  if (contentType) headers.set('content-type', contentType);
  // Forward the real client IP (first hop only) for the backend rate limiter.
  const clientIp = firstForwardedClientIp(req.headers.get('x-forwarded-for'));
  if (clientIp) headers.set('x-forwarded-for', clientIp);

  const isPost = req.method === 'POST';

  let res: Response;
  try {
    res = await fetch(url, {
      method: req.method,
      headers,
      body: isPost ? await req.text() : undefined,
      cache: 'no-store',
    });
  } catch (err) {
    console.error('Failed to reach backend', err);
    return NextResponse.json(
      { error: `Backend unavailable: ${BACKEND_URL}` },
      { status: 502 }
    );
  }

  // Teruskan body dan status backend tanpa perubahan.
  const resBody = await res.text();
  return new NextResponse(resBody, {
    status: res.status,
    headers: {
      'Content-Type': res.headers.get('content-type') ?? 'application/json',
    },
  });
}

export async function GET(req: NextRequest) {
  return proxy(req);
}

export async function POST(req: NextRequest) {
  return proxy(req);
}
