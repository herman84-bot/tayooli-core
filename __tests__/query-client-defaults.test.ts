import { createQueryClient, shouldRetryQuery, queryRetryDelay } from '@/lib/query-client'

const httpError = (status: number, retryAfterSeconds?: number) =>
  Object.assign(new Error(`HTTP ${status}`), { status, retryAfterSeconds })

// Regression: mutations (POST/PUT/DELETE) must never auto-retry. A retry after a
// proxy timeout re-sends a write the backend may already have committed,
// creating duplicate documents / double stock movements.
// Ref: TanStack Query default is "will not retry a mutation on error".
describe('createQueryClient defaults', () => {
  it('does not retry mutations', () => {
    const qc = createQueryClient()
    expect(qc.getDefaultOptions().mutations?.retry).toBe(0)
  })

  it('uses the status-aware retry policy for queries', () => {
    const qc = createQueryClient()
    expect(qc.getDefaultOptions().queries?.retry).toBe(shouldRetryQuery)
  })
})

describe('shouldRetryQuery', () => {
  it('retries network errors and 5xx up to 2 times', () => {
    expect(shouldRetryQuery(0, new TypeError('Failed to fetch'))).toBe(true)
    expect(shouldRetryQuery(1, httpError(503))).toBe(true)
    expect(shouldRetryQuery(2, httpError(502))).toBe(false)
  })

  it('never retries client errors (400/401/403/404/409/422): repeating cannot fix them', () => {
    for (const s of [400, 401, 403, 404, 409, 422]) {
      expect(shouldRetryQuery(0, httpError(s))).toBe(false)
    }
  })

  it('retries 429 (rate limited) after backing off', () => {
    expect(shouldRetryQuery(0, httpError(429, 3))).toBe(true)
  })
})

describe('queryRetryDelay', () => {
  it('honours Retry-After on 429', () => {
    expect(queryRetryDelay(0, httpError(429, 3))).toBe(3000)
  })

  it('caps Retry-After so the UI is not stuck', () => {
    expect(queryRetryDelay(0, httpError(429, 600))).toBe(30_000)
  })

  it('falls back to exponential backoff', () => {
    expect(queryRetryDelay(0, httpError(503))).toBe(1000)
    expect(queryRetryDelay(1, httpError(503))).toBe(2000)
    expect(queryRetryDelay(10, httpError(503))).toBe(10_000)
  })
})
