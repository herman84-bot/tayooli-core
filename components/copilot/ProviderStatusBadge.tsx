"use client"

interface ProviderStatusBadgeProps {
  provider: "gemini" | "secondary" | "offline"
  cooldownSeconds?: number
}

export function ProviderStatusBadge({ cooldownSeconds = 0 }: ProviderStatusBadgeProps) {
  if (cooldownSeconds > 0) {
    return (
      <div className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-zinc-100 text-zinc-600 border border-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:border-zinc-700">
        <span className="h-1.5 w-1.5 rounded-full bg-zinc-400" />
        <span>Menghubungkan...</span>
      </div>
    )
  }

  return (
    <div className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-50 text-emerald-700 border border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:border-emerald-800">
      <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
      <span>Online</span>
    </div>
  )
}
