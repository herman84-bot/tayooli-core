"use client"

import { type LucideIcon } from "lucide-react"
import * as Lucide from "lucide-react"

type IconName = keyof typeof Lucide

interface IconProps {
  name: IconName
  size?: "sm" | "md" | "lg"
  className?: string
}

const sizeMap = {
  sm: "w-4 h-4",
  md: "w-5 h-5",
  lg: "w-6 h-6",
} as const

export function Icon({ name, size = "md", className }: IconProps) {
  const IconComponent = Lucide[name] as LucideIcon | undefined
  if (!IconComponent) {
    console.warn(`Icon "${name}" not found in Lucide`)
    return null
  }
  return <IconComponent className={`${sizeMap[size]} text-current ${className ?? ""}`} strokeWidth={1.5} />
}
