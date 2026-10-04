import type { CSSProperties } from "react"

interface LogoProps {
  /** Pixel size of the square mark. Default 28. */
  size?: number
  className?: string
}

/**
 * Tayooli brand mark — solid teal-ink rounded-square tile with a white,
 * precisely-drawn "T" monogram (asymmetric crossbar + baseline accent).
 *
 * No glow, no gradients: flat, calm, finance-grade. Pure SVG so it stays
 * crisp at any size; the tile reads on both light and dark surfaces.
 */
export function Logo({ size = 28, className }: LogoProps) {
  const style: CSSProperties = {
    display: "block",
  }

  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      className={className}
      style={style}
      aria-hidden="true"
    >
      {/* Tile — brand primary */}
      <rect x="1" y="1" width="30" height="30" rx="8" fill="#1e6b58" />
      {/* Hairline rim so the tile lifts off dark surfaces too */}
      <rect
        x="1"
        y="1"
        width="30"
        height="30"
        rx="8"
        fill="none"
        stroke="rgba(255,255,255,0.16)"
        strokeWidth="1"
      />
      {/* "T" monogram — crossbar */}
      <rect x="9.6" y="10.4" width="13.6" height="3.6" rx="1.8" fill="#ffffff" />
      {/* "T" monogram — stem (slightly right of centre) */}
      <rect x="17.7" y="10.4" width="3.6" height="11.8" rx="1.8" fill="#ffffff" />
      {/* Baseline accent — lighter teal under the stem */}
      <rect x="16.4" y="24.2" width="6.2" height="1.7" rx="0.85" fill="#7cc9ab" />
    </svg>
  )
}
