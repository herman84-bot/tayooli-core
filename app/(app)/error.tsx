"use client"

import { ErrorFallback } from "@/components/errors/ErrorFallback"

// Lives inside app/(app)/layout.tsx, so the sidebar stays usable when a page crashes.
export default function AppError({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  return <ErrorFallback error={error} reset={reset} />
}
