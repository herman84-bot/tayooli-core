"use client"

import { Icon } from "@/components/ui/icon"

interface AnomalyBadgeProps {
  score: number
  detected: boolean
}

export function AnomalyBadge({ score, detected }: AnomalyBadgeProps) {
  if (!detected) {
    return (
      <span className="flex items-center gap-1 text-green-600 text-sm">
        <Icon name="CheckCircle" size="sm" />
        Normal
      </span>
    )
  }

  return (
    <span className="flex items-center gap-1 text-red-600 text-sm">
      <Icon name="AlertTriangle" size="sm" />
      Anomaly ({score.toFixed(1)}%)
    </span>
  )
}
