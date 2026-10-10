"use client"

import { useCallback, useEffect, useRef, useState } from "react"

type RefetchFn = () => Promise<unknown> | unknown

/** Minimum time the spinner stays visible so a fast (<100ms) refetch is still perceptible. */
export const MIN_REFRESH_FEEDBACK_MS = 700
/** How long the "Diperbarui" success state stays after a refresh. */
export const REFRESH_DONE_VISIBLE_MS = 2000

export type RefreshStatus = "idle" | "refreshing" | "done" | "error"

/**
 * Wraps one or more TanStack Query `refetch` functions for a manual "Refresh"
 * button. `refreshing` stays true until every refetch settles AND at least
 * MIN_REFRESH_FEEDBACK_MS has elapsed, then `status` becomes "done" (or
 * "error") for REFRESH_DONE_VISIBLE_MS so the user sees confirmation.
 * Repeated clicks while a refresh is running are ignored.
 */
export function useManualRefresh(refetchers: RefetchFn[]) {
  const [status, setStatus] = useState<RefreshStatus>("idle")
  const [lastRefreshedAt, setLastRefreshedAt] = useState<Date | null>(null)
  const [refreshError, setRefreshError] = useState<string | null>(null)
  const inFlight = useRef(false)
  const fnsRef = useRef(refetchers)
  const resetTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const mounted = useRef(true)

  useEffect(() => {
    fnsRef.current = refetchers
  })
  useEffect(() => {
    mounted.current = true
    return () => {
      mounted.current = false
      if (resetTimer.current) clearTimeout(resetTimer.current)
    }
  }, [])

  const refresh = useCallback(async () => {
    if (inFlight.current) return
    inFlight.current = true
    if (resetTimer.current) clearTimeout(resetTimer.current)
    setStatus("refreshing")
    setRefreshError(null)
    const minDelay = new Promise((r) => setTimeout(r, MIN_REFRESH_FEEDBACK_MS))
    let errorMsg: string | null = null
    try {
      const [results] = await Promise.all([
        Promise.all(fnsRef.current.map((fn) => Promise.resolve(fn()))),
        minDelay,
      ])
      // TanStack refetch resolves (not rejects) with { isError, error } on failure.
      const failed = results.find(
        (r) => r && typeof r === "object" && "isError" in r && (r as { isError: boolean }).isError
      ) as { error?: unknown } | undefined
      if (failed) {
        errorMsg = failed.error instanceof Error ? failed.error.message : "Gagal memuat ulang data"
      }
    } catch (err) {
      await minDelay
      errorMsg = err instanceof Error ? err.message : "Gagal memuat ulang data"
    } finally {
      inFlight.current = false
    }
    if (!mounted.current) return
    if (errorMsg) {
      setRefreshError(errorMsg)
      setStatus("error")
    } else {
      setLastRefreshedAt(new Date())
      setStatus("done")
    }
    resetTimer.current = setTimeout(() => {
      if (mounted.current) setStatus("idle")
    }, REFRESH_DONE_VISIBLE_MS)
  }, [])

  return { refresh, refreshing: status === "refreshing", status, lastRefreshedAt, refreshError }
}
