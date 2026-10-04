"use client"

import { useEffect } from "react"
import { useCopilot } from "@/hooks/useCopilot"
import { Bot, Sparkles, Command } from "lucide-react"
import { cn } from "@/lib/utils"

export function CopilotTrigger() {
  const { isOpen, toggle, open, pendingActions } = useCopilot()

  // Global Ctrl+K / Cmd+K listener
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault()
        toggle()
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [toggle])

  if (isOpen) return null

  const hasPending = pendingActions.filter((a) => a.status === "pending").length > 0

  return (
    <div className="fixed bottom-5 right-5 z-40 flex items-center gap-2">
      <button
        onClick={open}
        aria-label="Buka Tayooli Copilot"
        className={cn(
          "group relative flex items-center gap-2.5 px-3.5 py-2.5 rounded-full",
          "bg-primary text-primary-foreground shadow-md hover:shadow-lg",
          "border border-primary-foreground/15",
          "transition-all duration-150 cursor-pointer active:scale-95"
        )}
      >
        <div className="relative">
          <Bot className="h-4 w-4" />
          {hasPending && (
            <span className="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-amber-400 ring-2 ring-primary animate-pulse" />
          )}
        </div>
        <span className="text-xs font-semibold tracking-tight">Copilot</span>
        <span className="hidden sm:inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-[10px] font-mono bg-primary-foreground/15 text-primary-foreground">
          <Command className="h-2.5 w-2.5" />
          <span>K</span>
        </span>
      </button>
    </div>
  )
}
