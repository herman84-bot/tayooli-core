/**
 * @jest-environment node
 */
import { NextRequest } from 'next/server'
import { GET } from '@/app/api/v1/[...path]/route'

// The backend's per-tenant limiter answers 429 + Retry-After. The proxy must
// pass Retry-After through, or the browser cannot back off correctly.
describe('proxy forwards Retry-After', () => {
  const originalFetch = global.fetch
  afterEach(() => {
    global.fetch = originalFetch
  })

  it('keeps status 429 and the Retry-After header', async () => {
    global.fetch = jest.fn().mockResolvedValue(
      new Response(JSON.stringify({ error: 'Terlalu banyak permintaan.' }), {
        status: 429,
        headers: { 'content-type': 'application/json', 'retry-after': '3' },
      }),
    ) as unknown as typeof fetch

    const req = new NextRequest('http://localhost:3000/api/v1/products')
    const res = await GET(req, { params: Promise.resolve({ path: ['products'] }) })

    expect(res.status).toBe(429)
    expect(res.headers.get('retry-after')).toBe('3')
  })
})
