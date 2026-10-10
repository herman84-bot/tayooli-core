"use client"

import { useCallback, useEffect, useRef, useState } from "react"

type RefetchFn = () => Promise<unknown> | unknown

/**
 * Wraps one or more TanStack Query `refetch` functions for a manual "Refresh"
 * button. Unlike `isLoading` (true only on the very first load), `refreshing`
 * stays true until every refetch promise settles, so the button can spin,
 * disable itself, and report the outcome. Repeated clicks while a refresh is
 * running are ignored.
 */
export function useManualRefresh(refetchers: RefetchFn[]) {
  const [refreshing, setRefreshing] = useState(false)
  const [lastRefreshedAt, setLastRefreshedAt] = useState<Date | null>(null)
  const [refreshError, setRefreshError] = useState<string | null>(null)
  const inFlight = useRef(false)
  const fnsRef = useRef(refetchers)
  useEffect(() => {
    fnsRef.current = refetchers
  })

  const refresh = useCallback(async () => {
    if (inFlight.current) return
    inFlight.current = true
    setRefreshing(true)
    setRefreshError(null)
    try {
      const results = await Promise.all(fnsRef.current.map((fn) => Promise.resolve(fn())))
      // TanStack refetch resolves (not rejects) with { isError, error } on failure.
      const failed = results.find(
        (r) => r && typeof r === "object" && "isError" in r && (r as { isError: boolean }).isError
      ) as { error?: unknown } | undefined
      if (failed) {
        setRefreshError(failed.error instanceof Error ? failed.error.message : "Gagal memuat ulang data")
      } else {
        setLastRefreshedAt(new Date())
      }
    } catch (err) {
      setRefreshError(err instanceof Error ? err.message : "Gagal memuat ulang data")
    } finally {
      inFlight.current = false
      setRefreshing(false)
    }
  }, [])

  return { refresh, refreshing, lastRefreshedAt, refreshError }
}
