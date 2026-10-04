"use client"

import { useState, useEffect } from "react"
import { useCopilot } from "@/hooks/useCopilot"
import { useQueryClient } from "@tanstack/react-query"
import { Undo2, X, CheckCircle2, Loader2, AlertTriangle } from "lucide-react"

export function UndoToast() {
  const { lastUndoSnapshot, setLastUndoSnapshot } = useCopilot()
  const queryClient = useQueryClient()
  const [timeLeft, setTimeLeft] = useState(15)
  const [isUndoing, setIsUndoing] = useState(false)
  const [undoError, setUndoError] = useState<string | null>(null)

  // Countdown only
  useEffect(() => {
    if (!lastUndoSnapshot) return

    setTimeLeft(15)
    setUndoError(null)
    setIsUndoing(false)
    const interval = setInterval(() => {
      setTimeLeft((prev) => (prev <= 1 ? 0 : prev - 1))
    }, 1000)

    return () => clearInterval(interval)
  }, [lastUndoSnapshot])

  // Auto-dismiss once the countdown reaches zero
  useEffect(() => {
    if (timeLeft === 0 && lastUndoSnapshot && !isUndoing) {
      setLastUndoSnapshot(null)
    }
  }, [timeLeft, lastUndoSnapshot, isUndoing, setLastUndoSnapshot])

  if (!lastUndoSnapshot) return null

  const handleUndo = async () => {
    if (!lastUndoSnapshot) return

    // If no server undo endpoint is available, simply dismiss
    if (!lastUndoSnapshot.undoPath) {
      setLastUndoSnapshot(null)
      return
    }

    setIsUndoing(true)
    setUndoError(null)

    try {
      const res = await fetch(lastUndoSnapshot.undoPath, {
        method: lastUndoSnapshot.undoMethod || "DELETE",
        credentials: "include",
      })

      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err?.error?.message || err?.message || "Gagal membatalkan aksi di server")
      }

      // Invalidate relevant queries so UI reflects deletion/reversion immediately
      await queryClient.invalidateQueries()
      setLastUndoSnapshot(null)
    } catch (err: any) {
      setUndoError(err?.message || "Gagal mengurungkan aksi.")
      setIsUndoing(false)
    }
  }

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 animate-in fade-in slide-in-from-bottom-3 duration-200">
      <div className="flex flex-col gap-1.5 px-4 py-2.5 rounded-xl border border-border bg-card text-card-foreground shadow-lg text-xs">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-emerald-600 dark:text-emerald-400 shrink-0" />
            <span className="font-medium text-foreground">{lastUndoSnapshot.description} diterapkan</span>
          </div>

          <div className="h-4 w-[1px] bg-border" />

          <button
            onClick={handleUndo}
            disabled={isUndoing}
            className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md bg-muted hover:bg-muted/80 text-foreground font-medium transition-colors cursor-pointer disabled:opacity-50"
          >
            {isUndoing ? (
              <Loader2 className="h-3 w-3 animate-spin text-primary" />
            ) : (
              <Undo2 className="h-3 w-3" />
            )}
            <span>{isUndoing ? "Mengurungkan..." : `Urungkan (${timeLeft}s)`}</span>
          </button>

          <button
            onClick={() => setLastUndoSnapshot(null)}
            className="p-1 rounded text-muted-foreground hover:text-foreground cursor-pointer"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>

        {undoError && (
          <div className="flex items-center gap-1 text-[11px] text-destructive pt-1 border-t border-border">
            <AlertTriangle className="h-3 w-3 shrink-0" />
            <span>{undoError}</span>
          </div>
        )}
      </div>
    </div>
  )
}
