"use client"

import React from "react"
import { RefreshCw, Check, AlertCircle } from "lucide-react"
import type { RefreshStatus } from "@/hooks/useManualRefresh"

interface RefreshButtonProps {
  status: RefreshStatus
  onClick: () => void
  error?: string | null
  /** Show text label next to the icon. Icon-only buttons still get an accessible name. */
  showLabel?: boolean
  className?: string
  iconClassName?: string
  disabled?: boolean
}

const LABEL: Record<RefreshStatus, string> = {
  idle: "Segarkan",
  refreshing: "Menyegarkan...",
  done: "Diperbarui",
  error: "Gagal",
}

/**
 * Refresh control with visible state: spinning icon while fetching (min 700ms),
 * green check on success, red alert on failure. State is also announced to
 * screen readers via aria-live.
 */
export function RefreshButton({
  status,
  onClick,
  error,
  showLabel = false,
  className = "",
  iconClassName = "w-4 h-4",
  disabled,
}: RefreshButtonProps) {
  const refreshing = status === "refreshing"
  const tone =
    status === "done"
      ? "!border-emerald-300 !bg-emerald-50 !text-emerald-700"
      : status === "error"
      ? "!border-rose-300 !bg-rose-50 !text-rose-700"
      : ""
  const title = status === "error" && error ? `Gagal menyegarkan: ${error}` : LABEL[status]

  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled || refreshing}
      aria-busy={refreshing}
      aria-label={showLabel ? undefined : title}
      title={title}
      className={`${className} ${tone} transition-colors duration-200 active:scale-95 disabled:cursor-wait disabled:opacity-80`}
    >
      {status === "done" ? (
        <Check className={iconClassName} aria-hidden="true" />
      ) : status === "error" ? (
        <AlertCircle className={iconClassName} aria-hidden="true" />
      ) : (
        <RefreshCw className={`${iconClassName} ${refreshing ? "animate-spin" : ""}`} aria-hidden="true" />
      )}
      {showLabel && <span>{LABEL[status]}</span>}
      <span className="sr-only" aria-live="polite">
        {status === "idle" ? "" : title}
      </span>
    </button>
  )
}
