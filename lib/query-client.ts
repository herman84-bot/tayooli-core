import { QueryClient } from '@tanstack/react-query'

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
        retry: 2,
        retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 10_000),
      },
      mutations: {
        retry: 0,
      },
    },
  })
}
