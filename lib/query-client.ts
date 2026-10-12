import { QueryClient } from '@tanstack/react-query'

type MaybeHttpError = { status?: number; retryAfterSeconds?: number }

const MAX_QUERY_RETRIES = 2
const MAX_RETRY_AFTER_MS = 30_000

/**
 * Retry reads only when repeating can help: network errors, 5xx, and 429.
 * Other 4xx (bad input, auth, not found, conflict) fail the same way every
 * time, so retrying only adds load during an incident.
 */
export function shouldRetryQuery(failureCount: number, error: unknown): boolean {
  if (failureCount >= MAX_QUERY_RETRIES) return false
  const status = (error as MaybeHttpError | null)?.status
  if (typeof status === 'number' && status >= 400 && status < 500 && status !== 429) return false
  return true
}

/** Honour the server's Retry-After on 429 (capped); otherwise exponential backoff. */
export function queryRetryDelay(attemptIndex: number, error: unknown): number {
  const e = error as MaybeHttpError | null
  if (e?.status === 429 && e.retryAfterSeconds) {
    return Math.min(e.retryAfterSeconds * 1000, MAX_RETRY_AFTER_MS)
  }
  return Math.min(1000 * 2 ** attemptIndex, 10_000)
}

/**
 * Shared QueryClient factory.
 *
 * mutations.retry = 0: writes are NOT idempotent by default. If a write times
 * out at the proxy while the backend still commits it, an automatic retry would
 * create a duplicate document / double stock movement. TanStack Query's own
 * default is to not retry mutations; we keep that explicitly.
 */
export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        retry: shouldRetryQuery,
        retryDelay: queryRetryDelay,
      },
      mutations: {
        retry: 0,
      },
    },
  })
}
