/**
 * @jest-environment node
 */
import { NextRequest } from 'next/server'
import { GET } from '@/app/api/v1/[...path]/route'

// Regression: a proxy timeout must surface as 504 (outcome unknown, backend may
// still be committing), not 502 "backend unreachable".
describe('api/v1 proxy timeout', () => {
  const realFetch = global.fetch
  afterEach(() => {
    global.fetch = realFetch
  })

  it('returns 504 when the backend call times out', async () => {
    global.fetch = jest.fn().mockRejectedValue(
      Object.assign(new Error('timed out'), { name: 'TimeoutError' }),
    ) as unknown as typeof fetch
    const req = new NextRequest('http://localhost:3000/api/v1/products')
    const res = await GET(req, { params: Promise.resolve({ path: ['products'] }) })
    expect(res.status).toBe(504)
  })

  it('returns 502 when the backend is unreachable', async () => {
    global.fetch = jest.fn().mockRejectedValue(new TypeError('fetch failed')) as unknown as typeof fetch
    const req = new NextRequest('http://localhost:3000/api/v1/products')
    const res = await GET(req, { params: Promise.resolve({ path: ['products'] }) })
    expect(res.status).toBe(502)
  })
})
