import React from "react"

export interface ShimmerSkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  className?: string
}

/**
 * ShimmerSkeleton — Pure CSS GPU-accelerated skeleton wave inspired by Binance.
 * Zero JS overhead, zero layout shift, seamless in both light and dark modes.
 */
export function ShimmerSkeleton({ className = "", ...props }: ShimmerSkeletonProps) {
  return (
    <div
      className={`shimmer-wave rounded-md ${className}`}
      {...props}
    />
  )
}

/**
 * MetricCardSkeleton — Placeholder skeleton for KPI/Stat cards.
 */
export function MetricCardSkeleton({ className = "" }: { className?: string }) {
  return (
    <div className={`border border-border/60 rounded-xl p-4 bg-card shadow-xs space-y-3 ${className}`}>
      <div className="flex items-center gap-3">
        <ShimmerSkeleton className="h-9 w-9 rounded-lg shrink-0" />
        <div className="space-y-1.5 flex-1">
          <ShimmerSkeleton className="h-5 w-24 rounded" />
          <ShimmerSkeleton className="h-3 w-16 rounded" />
        </div>
      </div>
      <ShimmerSkeleton className="h-2.5 w-full rounded mt-2" />
    </div>
  )
}

/**
 * TableSkeleton — Multi-row table skeleton placeholder with header and data rows.
 */
export function TableSkeleton({
  rows = 5,
  cols = 5,
  className = "",
}: {
  rows?: number
  cols?: number
  className?: string
}) {
  return (
    <div className={`w-full overflow-hidden rounded-xl border border-border/60 bg-card ${className}`}>
      {/* Table Header Placeholder */}
      <div className="bg-slate-50/70 border-b border-border/60 px-6 py-3.5 flex items-center justify-between gap-4">
        {Array.from({ length: cols }).map((_, i) => (
          <ShimmerSkeleton key={i} className="h-3.5 w-20 rounded" />
        ))}
      </div>

      {/* Table Rows Placeholder */}
      <div className="divide-y divide-slate-100">
        {Array.from({ length: rows }).map((_, rIdx) => (
          <div key={rIdx} className="px-6 py-4 flex items-center justify-between gap-4">
            <ShimmerSkeleton className="h-4 w-24 rounded" />
            <ShimmerSkeleton className="h-4 w-36 rounded" />
            <ShimmerSkeleton className="h-4 w-20 rounded" />
            <ShimmerSkeleton className="h-4 w-28 rounded" />
            <ShimmerSkeleton className="h-7 w-16 rounded-md" />
          </div>
        ))}
      </div>
    </div>
  )
}

/**
 * BinanceArcSpinner — Smooth curved SVG arc spinner with variable-speed acceleration
 * and smooth deceleration (Binance Pro signature spinner).
 */
export function BinanceArcSpinner({
  size = 20,
  strokeColor = "#F0B90B",
  strokeWidth = 3.5,
  className = "",
}: {
  size?: number
  strokeColor?: string
  strokeWidth?: number
  className?: string
}) {
  return (
    <svg
      className={`animate-arc-spin shrink-0 ${className}`}
      width={size}
      height={size}
      viewBox="25 25 50 50"
    >
      <circle
        className="animate-arc-dash"
        cx="50"
        cy="50"
        r="20"
        fill="none"
        stroke={strokeColor}
        strokeWidth={strokeWidth}
      />
    </svg>
  )
}
