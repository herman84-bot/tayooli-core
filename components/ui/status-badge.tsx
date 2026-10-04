import { cn } from '@/lib/utils'

const defaultColorMap: Record<string, string> = {
  pending: 'bg-amber-50 text-amber-700 border-amber-200',
  approved: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  accepted: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  matched: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  received: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  rejected: 'bg-rose-50 text-rose-600 border-rose-200',
  cancelled: 'bg-rose-50 text-rose-600 border-rose-200',
  pending_review: 'bg-orange-50 text-orange-700 border-orange-200',
  partially_received: 'bg-blue-50 text-blue-700 border-blue-200',
  open: 'bg-blue-50 text-blue-700 border-blue-200',
  ai_processed: 'bg-indigo-50 text-indigo-700 border-indigo-200',
  ai_failed: 'bg-red-50 text-red-700 border-red-200',
}

export interface StatusBadgeProps {
  status: string
  colorMap?: Record<string, string>
  label?: string
}

export function StatusBadge({ status, colorMap, label }: StatusBadgeProps) {
  const map = colorMap ?? defaultColorMap
  const classes = map[status] ?? 'bg-zinc-50 text-zinc-600 border-zinc-200'
  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold border',
        classes,
      )}
    >
      {label ?? status}
    </span>
  )
}
