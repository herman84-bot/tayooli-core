"use client"

import React, { useCallback, useEffect, useState } from "react"
import { motion, useReducedMotion } from "motion/react"

/**
 * useBoop — Exact 1:1 port of Railway's `Te` (`useBoop`) hook from `useCommands-C-c1VReq.js`:
 *
 * ```js
 * we = { stiffness: 300, damping: 10 },
 * Te = ({ x = 0, y = 0, rotate = 0, scale = 1, timing = 150 }) => {
 *   let a = useReducedMotion(),
 *       [o, s] = useState(false),
 *       c = o && !a ? { x, y, rotate, scale } : { x: 0, y: 0, rotate: 0, scale: 1 };
 *   useEffect(() => {
 *     if (!o) return;
 *     let id = setTimeout(() => { s(false) }, timing);
 *     return () => { clearTimeout(id) };
 *   }, [o]);
 *   return [c, we, useCallback(() => { s(true) }, [])];
 * }
 * ```
 *
 * In Railway's ProjectSettings sidebar (`ka` in `ProjectSettings-NMxbXMjS.js`),
 * each item calls `useBoop({ rotate: 35 })` and triggers `onMouseEnter={trigger}`.
 */

const SPRING_TRANSITION = {
  type: "spring" as const,
  stiffness: 300,
  damping: 10,
}

export interface BoopConfig {
  x?: number
  y?: number
  rotate?: number
  scale?: number
  timing?: number
}

export function useBoop({
  x = 0,
  y = 0,
  rotate = 35,
  scale = 1,
  timing = 150,
}: BoopConfig = {}) {
  const prefersReducedMotion = useReducedMotion()
  const [isBooped, setIsBooped] = useState(false)

  useEffect(() => {
    if (!isBooped) return
    const timeoutId = window.setTimeout(() => {
      setIsBooped(false)
    }, timing)
    return () => {
      window.clearTimeout(timeoutId)
    }
  }, [isBooped, timing])

  const trigger = useCallback(() => {
    setIsBooped(true)
  }, [])

  const animateState =
    isBooped && !prefersReducedMotion
      ? { x, y, rotate, scale }
      : { x: 0, y: 0, rotate: 0, scale: 1 }

  return [animateState, SPRING_TRANSITION, trigger] as const
}

export interface BoopIconProps {
  icon: React.ElementType
  className?: string
  animateState: { x: number; y: number; rotate: number; scale: number }
  transition?: typeof SPRING_TRANSITION
}

export function BoopIcon({
  icon: Icon,
  className,
  animateState,
  transition = SPRING_TRANSITION,
}: BoopIconProps) {
  return (
    <motion.span
      initial={{ x: 0, y: 0, rotate: 0, scale: 1 }}
      animate={animateState}
      transition={transition}
      className="inline-flex shrink-0 items-center justify-center focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring active:scale-95 disabled:opacity-50"
      aria-hidden="true"
    >
      <Icon className={className} />
    </motion.span>
  )
}
