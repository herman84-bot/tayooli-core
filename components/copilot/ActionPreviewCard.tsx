"use client"

import { useState } from "react"
import { ProposedAction, useCopilot } from "@/hooks/useCopilot"
import { executeToolCall } from "@/lib/copilot/executor"
import { useRouter } from "next/navigation"
import { useQueryClient } from "@tanstack/react-query"
import { Check, X, AlertTriangle, ShieldAlert, Loader2, ArrowRight, RotateCcw } from "lucide-react"
import { cn } from "@/lib/utils"

interface ActionPreviewCardProps {
  action: ProposedAction
}

export function ActionPreviewCard({ action }: ActionPreviewCardProps) {
  const router = useRouter()
  const queryClient = useQueryClient()
  const { updateActionStatus, setLastUndoSnapshot } = useCopilot()
  const [confirmInput, setConfirmInput] = useState("")
  const [unsupportedAck, setUnsupportedAck] = useState(false)

  const isConfirmed = !action.requiresConfirmationText || confirmInput.trim().toUpperCase() === action.requiresConfirmationText
  const isBlocked = !!(action.blockedReasons && action.blockedReasons.length > 0)
  const hasUnackedFields = !!(action.unsupportedFields && action.unsupportedFields.length > 0 && !unsupportedAck)
  const canExecute = isConfirmed && !isBlocked && !hasUnackedFields

  const handleExecute = async () => {
    if (!canExecute) return

    updateActionStatus(action.id, "executing")
    try {
      const res = await executeToolCall(action.toolName, action.payload, (path) => router.push(path))

      // Invalidate relevant queries to keep frontend strictly in sync
      if (res.targetModule) {
        queryClient.invalidateQueries({ queryKey: [res.targetModule] })
        if (res.targetModule === "settings") {
          queryClient.invalidateQueries({ queryKey: ["team-members"] })
          queryClient.invalidateQueries({ queryKey: ["payments-config"] })
        }
      }

      // Record snapshot for undo toast
      setLastUndoSnapshot({
        id: action.id,
        toolName: action.toolName,
        timestamp: Date.now(),
        restorePayload: action.payload,
        description: action.title,
        undoPath: res.undo?.path,
        undoMethod: res.undo?.method,
        entityId: res.undo ? (res.data?.id || res.data?.member?.id) : undefined,
      })

      updateActionStatus(action.id, "executed")
    } catch (err: any) {
      updateActionStatus(action.id, "failed", err?.message || "Gagal mengeksekusi aksi.")
    }
  }

  const handleCancel = () => {
    updateActionStatus(action.id, "cancelled")
  }

  const handleRetry = () => {
    // Re-open the proposal: back to pending so the user can adjust intent
    // (or the underlying data) and approve again without re-typing the prompt.
    updateActionStatus(action.id, "pending")
  }

  // Risk badge styling with safe fallback
  const riskLevels = {
    low: {
      label: "Rendah",
      className: "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-300 dark:border-emerald-800",
      icon: Check,
    },
    medium: {
      label: "Sedang",
      className: "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-950/40 dark:text-amber-300 dark:border-amber-800",
      icon: AlertTriangle,
    },
    high: {
      label: "Kritis",
      className: "bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-950/40 dark:text-rose-300 dark:border-rose-800",
      icon: ShieldAlert,
    },
  }

  const levelKey = (action.riskLevel && action.riskLevel in riskLevels)
    ? (action.riskLevel as "low" | "medium" | "high")
    : "low"
  const riskBadge = riskLevels[levelKey]
  const RiskIcon = riskBadge.icon

  return (
    <div className="my-3 rounded-xl border border-border bg-card text-card-foreground shadow-xs overflow-hidden">
      {/* Card Header */}
      <div className="flex items-center justify-between px-3.5 py-2.5 bg-muted/40 border-b border-border">
        <div className="flex items-center gap-2">
          <span className="text-xs font-semibold text-foreground tracking-tight">{action.title}</span>
        </div>
        <div className={cn("inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10px] font-medium border", riskBadge.className)}>
          <RiskIcon className="h-3 w-3" />
          <span>Risiko: {riskBadge.label}</span>
        </div>
      </div>

      {/* Description */}
      <div className="px-3.5 py-2.5 text-xs text-muted-foreground leading-relaxed">
        {action.description}
      </div>

      {/* Diff Table */}
      {action.diff && action.diff.length > 0 && (
        <div className="px-3.5 pb-2.5">
          <div className="rounded-lg border border-border bg-background overflow-hidden">
            <table className="w-full text-left text-xs">
              <thead className="bg-muted/50 text-[11px] text-muted-foreground border-b border-border font-medium">
                <tr>
                  <th className="py-1.5 px-2.5">Field</th>
                  <th className="py-1.5 px-2.5">Sebelumnya</th>
                  <th className="py-1.5 px-2.5">Perubahan</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {action.diff.map((d, idx) => (
                  <tr key={idx} className="hover:bg-muted/20">
                    <td className="py-1.5 px-2.5 font-medium text-foreground">{d.field}</td>
                    <td className="py-1.5 px-2.5 text-muted-foreground line-through font-mono text-[11px]">
                      {String(d.oldValue || "-")}
                    </td>
                    <td className="py-1.5 px-2.5 font-mono text-[11px] text-primary font-medium">
                      <div className="inline-flex items-center gap-1">
                        <ArrowRight className="h-3 w-3 text-muted-foreground" />
                        <span>{String(d.newValue || "-")}</span>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Source Quote */}
      {action.sourceQuote && (
        <div className="px-3.5 py-1.5 bg-muted/30 border-b border-border text-[11px] text-muted-foreground italic">
          Kutipan pesan: &ldquo;{action.sourceQuote}&rdquo;
        </div>
      )}

      {/* Blocked Reasons Warning */}
      {action.blockedReasons && action.blockedReasons.length > 0 && (
        <div className="px-3.5 py-2 border-t border-border bg-rose-50/70 dark:bg-rose-950/40 text-rose-800 dark:text-rose-300 text-xs">
          <div className="font-semibold flex items-center gap-1.5 mb-1 text-rose-900 dark:text-rose-200">
            <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
            <span>Proposal Diblokir (Data Tidak Valid / Kurang):</span>
          </div>
          <ul className="list-disc pl-5 space-y-0.5 text-[11px]">
            {action.blockedReasons.map((reason, idx) => (
              <li key={idx}>{reason}</li>
            ))}
          </ul>
        </div>
      )}

      {/* Unsupported Fields Warning */}
      {action.unsupportedFields && action.unsupportedFields.length > 0 && (
        <div className="px-3.5 py-2 border-t border-border bg-amber-50/70 dark:bg-amber-950/40 text-amber-800 dark:text-amber-300 text-xs">
          <div className="font-semibold flex items-center gap-1.5 mb-1 text-amber-900 dark:text-amber-200">
            <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
            <span>Field Tidak Didukung (Akan Dibuang):</span>
          </div>
          <ul className="list-disc pl-5 space-y-0.5 text-[11px]">
            {action.unsupportedFields.map((f, idx) => (
              <li key={idx}>
                <span className="font-mono font-medium">{f.field}</span>: {f.reason}
              </li>
            ))}
          </ul>
          <label className="flex items-center gap-2 mt-2 cursor-pointer text-[11px] font-medium text-amber-900 dark:text-amber-200">
            <input
              type="checkbox"
              checked={unsupportedAck}
              onChange={(e) => setUnsupportedAck(e.target.checked)}
              className="rounded border-amber-400"
            />
            <span>Saya mengerti field di atas tidak didukung dan akan dibuang</span>
          </label>
        </div>
      )}

      {/* Confirmation text input for High Risk actions */}
      {action.requiresConfirmationText && action.status === "pending" && (
        <div className="px-3.5 py-2 border-t border-border bg-rose-50/40 dark:bg-rose-950/20">
          <label className="block text-[11px] font-medium text-rose-800 dark:text-rose-300 mb-1">
            Ketik <span className="font-mono font-bold">{action.requiresConfirmationText}</span> untuk mengizinkan:
          </label>
          <input
            type="text"
            value={confirmInput}
            onChange={(e) => setConfirmInput(e.target.value)}
            placeholder={action.requiresConfirmationText}
            className="w-full px-2.5 py-1 text-xs rounded-md border border-rose-300 dark:border-rose-800 bg-background text-foreground font-mono focus:outline-hidden focus:ring-1 focus:ring-rose-500"
          />
        </div>
      )}

      {/* Footer / Status / Actions */}
      <div className="px-3.5 py-2.5 bg-muted/20 border-t border-border flex items-center justify-between gap-2">
        {action.status === "pending" && (
          <div className="flex items-center justify-end gap-2 w-full">
            <button
              onClick={handleCancel}
              className="px-3 py-1.5 text-xs rounded-md border border-border bg-background text-foreground hover:bg-muted font-medium transition-colors"
            >
              Batalkan
            </button>
            <button
              onClick={handleExecute}
              disabled={!canExecute}
              className={cn(
                "inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md font-medium text-primary-foreground transition-colors shadow-xs",
                canExecute
                  ? "bg-primary hover:bg-primary/90 cursor-pointer"
                  : "bg-muted text-muted-foreground cursor-not-allowed"
              )}
            >
              Setujui & Jalankan
            </button>
          </div>
        )}

        {action.status === "executing" && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground font-medium py-1">
            <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />
            <span>Mengeksekusi ke backend Go...</span>
          </div>
        )}

        {action.status === "executed" && (
          <div className="flex items-center gap-1.5 text-xs text-emerald-600 dark:text-emerald-400 font-medium py-1">
            <Check className="h-4 w-4" />
            <span>Aksi berhasil dieksekusi</span>
          </div>
        )}

        {action.status === "cancelled" && (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground py-1">
            <X className="h-3.5 w-3.5" />
            <span>Dibatalkan oleh pengguna</span>
          </div>
        )}

        {action.status === "failed" && (
          <div className="flex flex-col gap-1 text-xs text-destructive py-1 w-full">
            <div className="flex items-center gap-1 font-medium">
              <AlertTriangle className="h-3.5 w-3.5" />
              <span>Eksekusi Gagal</span>
            </div>
            {action.error && <span className="text-[11px] text-muted-foreground">{action.error}</span>}
            <div className="flex items-center justify-end gap-2 mt-0.5">
              <button
                onClick={handleCancel}
                className="px-3 py-1.5 text-xs rounded-md border border-border bg-background text-foreground hover:bg-muted font-medium transition-colors"
              >
                Batalkan
              </button>
              <button
                onClick={handleRetry}
                className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-md font-medium text-primary-foreground transition-colors shadow-xs bg-primary hover:bg-primary/90 cursor-pointer"
              >
                <RotateCcw className="h-3 w-3" />
                Coba Lagi
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
