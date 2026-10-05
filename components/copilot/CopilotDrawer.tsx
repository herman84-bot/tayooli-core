"use client"

import { useState, useRef, useEffect } from "react"
import { useCopilot, CopilotMessage, ProposedAction } from "@/hooks/useCopilot"
import { useAIPermissions } from "@/hooks/useAIPermissions"
import { useAuth } from "@/hooks/useAuth"
import { usePathname, useRouter } from "next/navigation"
import { useQueryClient } from "@tanstack/react-query"
import { executeToolCall } from "@/lib/copilot/executor"
import { ActionPreviewCard } from "./ActionPreviewCard"
import { ProviderStatusBadge } from "./ProviderStatusBadge"
import { AutonomySettingsModal } from "./AutonomySettingsModal"
import {
  X,
  Send,
  Loader2,
  Settings2,
  Trash2,
  Bot,
  User,
  ShieldAlert,
  Sparkles,
} from "lucide-react"
import { cn } from "@/lib/utils"

function splitTableRow(line: string): string[] {
  let content = line.trim()
  if (content.startsWith("|")) content = content.slice(1)
  if (content.endsWith("|")) content = content.slice(0, -1)
  return content.split("|").map((cell) => cell.trim())
}

function isTableRow(line: string): boolean {
  const trimmed = line.trim()
  return trimmed.includes("|") && trimmed.split("|").length >= 3
}

function isTableSeparator(line: string): boolean {
  const trimmed = line.trim()
  if (!trimmed.includes("|")) return false
  const cells = splitTableRow(trimmed)
  return cells.length > 0 && cells.every((c) => /^:?-+:?$/.test(c.trim()))
}

function renderInlineMarkdown(cleanLine: string, keyPrefix: string) {
  const parts = []
  const regex = /(\*\*.*?\*\*|`.*?`|\*.*?\*)/g
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = regex.exec(cleanLine)) !== null) {
    if (match.index > lastIndex) {
      parts.push(cleanLine.slice(lastIndex, match.index))
    }
    const token = match[0]
    if (token.startsWith("**") && token.endsWith("**")) {
      parts.push(
        <strong key={`${keyPrefix}-${match.index}`} className="font-semibold text-foreground">
          {token.slice(2, -2)}
        </strong>
      )
    } else if (token.startsWith("`") && token.endsWith("`")) {
      parts.push(
        <code key={`${keyPrefix}-${match.index}`} className="px-1 py-0.5 rounded bg-muted font-mono text-[11px] text-foreground">
          {token.slice(1, -1)}
        </code>
      )
    } else if (token.startsWith("*") && token.endsWith("*")) {
      parts.push(
        <span key={`${keyPrefix}-${match.index}`} className="text-foreground font-medium">
          {token.slice(1, -1)}
        </span>
      )
    }
    lastIndex = regex.lastIndex
  }

  if (lastIndex < cleanLine.length) {
    parts.push(cleanLine.slice(lastIndex))
  }

  return parts.length > 0 ? parts : cleanLine
}

export function renderFormattedMessage(text: string) {
  if (!text) return null

  const lines = text.split("\n")
  const elements: React.ReactNode[] = []
  let i = 0

  while (i < lines.length) {
    const line = lines[i]

    // Detect Markdown Table: header with separator OR consecutive table rows
    const hasSep = i + 1 < lines.length && isTableSeparator(lines[i + 1])
    const hasNextRow = i + 1 < lines.length && isTableRow(lines[i + 1])

    if (isTableRow(line) && (hasSep || hasNextRow)) {
      const headers = splitTableRow(line)
      const dataRows: string[][] = []
      i += hasSep ? 2 : 1 // skip header and separator lines (or just header)

      while (i < lines.length && isTableRow(lines[i]) && !isTableSeparator(lines[i])) {
        dataRows.push(splitTableRow(lines[i]))
        i++
      }

      elements.push(
        <div key={`table-${i}`} className="my-2.5 overflow-x-auto rounded-lg border border-border bg-card/60 shadow-sm">
          <table className="w-full text-left text-xs border-collapse">
            <thead className="bg-muted/70 border-b border-border text-foreground font-semibold">
              <tr>
                {headers.map((h, hIdx) => (
                  <th key={hIdx} className="px-3 py-2 text-foreground font-semibold whitespace-nowrap">
                    {renderInlineMarkdown(h, `th-${i}-${hIdx}`)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-border/50">
              {dataRows.map((row, rIdx) => (
                <tr key={rIdx} className="hover:bg-muted/30 transition-colors">
                  {row.map((cell, cIdx) => (
                    <td key={cIdx} className="px-3 py-2 text-foreground/90 align-top">
                      {renderInlineMarkdown(cell, `td-${i}-${rIdx}-${cIdx}`)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )
      continue
    }

    if (!line.trim()) {
      elements.push(<div key={`blank-${i}`} className="h-1.5" />)
      i++
      continue
    }

    // Process bullet points
    const isBullet = line.trim().startsWith("- ") || line.trim().startsWith("* ")
    const cleanLine = isBullet ? line.trim().slice(2) : line

    elements.push(
      <div key={`line-${i}`} className={cn("leading-relaxed", isBullet && "flex items-start gap-1.5 pl-2")}>
        {isBullet && <span className="text-muted-foreground mt-0.5">•</span>}
        <div>{renderInlineMarkdown(cleanLine, `inline-${i}`)}</div>
      </div>
    )
    i++
  }

  return elements
}

type ProviderStatus = "gemini" | "secondary" | "offline"

// mapProvider normalizes the backend's providerUsed string to the union type
// used by the status badge. Gemini/"gemini" → gemini; "groq/..." becomes
// secondary; anything else/empty → secondary (online) unless offline.
function mapProvider(raw: string | undefined): ProviderStatus {
  if (!raw) return "secondary"
  const s = raw.toLowerCase()
  if (s === "offline") return "offline"
  if (s.startsWith("gemini")) return "gemini"
  return "secondary"
}

export function CopilotDrawer() {
  const router = useRouter()
  const queryClient = useQueryClient()
  const {
    isOpen,
    close,
    openSettings,
    messages,
    addMessage,
    clearMessages,
    isStreaming,
    setIsStreaming,
    activeProvider,
    setActiveProvider,
    autonomyLevel,
    updateActionStatus,
    setLastUndoSnapshot,
    cooldownSeconds,
    setCooldownSeconds,
  } = useCopilot()

  const { data: permissions } = useAIPermissions()
  const { user } = useAuth()
  const pathname = usePathname()

  const [input, setInput] = useState("")
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  // Auto-execute low-risk actions in Autopilot mode
  const executeAutopilotAction = async (action: ProposedAction) => {
    updateActionStatus(action.id, "executing")
    try {
      const res = await executeToolCall(action.toolName, action.payload, (path) => router.push(path))

      if (res.targetModule) {
        queryClient.invalidateQueries({ queryKey: [res.targetModule] })
        if (res.targetModule === "settings") {
          queryClient.invalidateQueries({ queryKey: ["team-members"] })
          queryClient.invalidateQueries({ queryKey: ["payments-config"] })
        }
      }

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
      updateActionStatus(action.id, "failed", err?.message || "Gagal mengeksekusi aksi otomatis.")
    }
  }

  // Auto-scroll to bottom
  useEffect(() => {
    if (isOpen) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
    }
  }, [messages, isOpen])

  // Focus input on open
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 100)
    }
  }, [isOpen])

  // Cooldown countdown timer
  useEffect(() => {
    if (cooldownSeconds <= 0) return
    const interval = setInterval(() => {
      setCooldownSeconds(cooldownSeconds - 1)
    }, 1000)
    return () => clearInterval(interval)
  }, [cooldownSeconds, setCooldownSeconds])

  const handleSend = async (textToSend?: string) => {
    const query = (textToSend || input).trim()
    if (!query || isStreaming) return

    setInput("")

    const userMsg: CopilotMessage = {
      id: `user-${Date.now()}`,
      role: "user",
      content: query,
      timestamp: new Date().toISOString(),
    }
    addMessage(userMsg)
    setIsStreaming(true)

    try {
      const effectiveAutonomy = autonomyLevel || permissions?.autonomy_level || "assisted"

      const res = await fetch("/api/v1/copilot/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          message: query,
          context: {
            tenantId: user?.tenantId || "default",
            userRole: user?.role || "admin",
            activeRoute: pathname,
            activeModule: pathname.split("/")[1] || "dashboard",
            autonomyLevel: effectiveAutonomy,
            allowedScopes: permissions?.allowed_scopes || ["workspace.read"],
            emergencyStop: permissions?.emergency_stop || false,
          },
        }),
      })

      if (!res.ok) {
        throw new Error(`HTTP error ${res.status}`)
      }

      const data = await res.json()

      setActiveProvider(mapProvider(data.providerUsed))

      // In Advisory mode: block mutating actions (only allow navigation)
      const filteredActions: ProposedAction[] = (data.actions || []).filter((action: ProposedAction) => {
        if (effectiveAutonomy === "advisory" && action.toolName !== "navigate_to_module") {
          return false
        }
        return true
      })

      const assistantMsg: CopilotMessage = {
        id: `assistant-${Date.now()}`,
        role: "assistant",
        content: data.reply || "",
        timestamp: new Date().toISOString(),
        proposedActions: filteredActions,
        providerUsed: data.providerUsed,
      }
      addMessage(assistantMsg)

      // In Autopilot mode: auto-execute low-risk actions without waiting for user click
      if (effectiveAutonomy === "autopilot") {
        for (const action of filteredActions) {
          if (action.riskLevel === "low") {
            // Asynchronously auto-execute
            executeAutopilotAction(action)
          }
        }
      }
    } catch (err: any) {
      console.error("Copilot fetch error:", err)
      setActiveProvider("offline")
      setCooldownSeconds(30)

      addMessage({
        id: `assistant-err-${Date.now()}`,
        role: "assistant",
        content:
          "Terjadi kendala saat menghubungi server AI. Sistem beralih ke mode navigasi lokal. Anda tetap dapat menggunakan menu navigasi langsung.",
        timestamp: new Date().toISOString(),
        providerUsed: "offline",
      })
    } finally {
      setIsStreaming(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const quickPrompts = [
    "Ubah nama perusahaan",
    "Cek stok produk",
    "Buka kasir POS",
    "Buka Stock Opname",
  ]

  if (!isOpen) return null

  return (
    <>
      <div
        className="fixed inset-0 z-40 bg-black/20 backdrop-blur-xs transition-opacity"
        onClick={close}
      />

      <aside
        className={cn(
          "fixed top-0 right-0 z-50 h-full w-[420px] max-w-[calc(100vw-1.5rem)]",
          "bg-card text-card-foreground border-l border-border shadow-2xl",
          "flex flex-col animate-in slide-in-from-right duration-200"
        )}
      >
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-muted/30">
          <div className="flex items-center gap-2.5">
            <div className="h-7 w-7 rounded-lg bg-primary/10 flex items-center justify-center text-primary">
              <Bot className="h-4 w-4" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-xs font-semibold text-foreground tracking-tight">Tayooli Copilot</h3>
                <ProviderStatusBadge provider={activeProvider} cooldownSeconds={cooldownSeconds} />
              </div>
              <p className="text-[10px] text-muted-foreground">Autonomous Workspace Assistant</p>
            </div>
          </div>

          <div className="flex items-center gap-1">
            <button
              onClick={openSettings}
              title="Pengaturan Wewenang AI"
              className="p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            >
              <Settings2 className="h-4 w-4" />
            </button>
            <button
              onClick={clearMessages}
              title="Reset Percakapan"
              className="p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            >
              <Trash2 className="h-4 w-4" />
            </button>
            <button
              onClick={close}
              title="Tutup Panel"
              className="p-1.5 rounded-md text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* Emergency Stop Alert Banner if active */}
        {permissions?.emergency_stop && (
          <div className="px-4 py-2 bg-rose-50 dark:bg-rose-950/40 border-b border-rose-200 dark:border-rose-900 flex items-center gap-2 text-rose-800 dark:text-rose-200 text-xs">
            <ShieldAlert className="h-4 w-4 shrink-0 text-rose-600" />
            <span>Emergency Kill-Switch aktif: Eksekusi otomatis dinonaktifkan.</span>
          </div>
        )}

        {/* Message History */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 text-xs">
          {messages.map((m) => (
            <div
              key={m.id}
              className={cn(
                "flex flex-col gap-1.5 max-w-[92%]",
                m.role === "user" ? "ml-auto items-end" : "mr-auto items-start"
              )}
            >
              <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground px-1">
                {m.role === "user" ? (
                  <>
                    <span>Anda</span>
                    <User className="h-3 w-3" />
                  </>
                ) : (
                  <>
                    <Bot className="h-3 w-3 text-primary" />
                    <span>Copilot</span>
                  </>
                )}
              </div>

              <div
                className={cn(
                  "p-3 rounded-xl leading-relaxed text-xs",
                  m.role === "user"
                    ? "bg-primary text-primary-foreground font-medium rounded-tr-xs"
                    : "bg-muted/50 text-foreground border border-border rounded-tl-xs"
                )}
              >
                {renderFormattedMessage(m.content)}
              </div>

              {/* Render Action Preview Cards if any */}
              {m.proposedActions && m.proposedActions.length > 0 && (
                <div className="w-full">
                  {m.proposedActions.map((action) => (
                    <ActionPreviewCard key={action.id} action={action} />
                  ))}
                </div>
              )}
            </div>
          ))}

          {isStreaming && (
            <div className="flex items-center gap-2 text-muted-foreground text-xs p-2">
              <Loader2 className="h-3.5 w-3.5 animate-spin text-primary" />
              <span>Memproses instruksi workspace...</span>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Quick Suggestion Chips */}
        <div className="px-3 py-2 border-t border-border bg-muted/10 overflow-x-auto flex items-center gap-1.5 scrollbar-none">
          {quickPrompts.map((prompt, idx) => (
            <button
              key={idx}
              onClick={() => handleSend(prompt)}
              className="shrink-0 text-[11px] px-2.5 py-1 rounded-full border border-border bg-background hover:bg-muted text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            >
              {prompt}
            </button>
          ))}
        </div>

        {/* Input Bar */}
        <div className="p-3 border-t border-border bg-card">
          <div className="relative flex items-center">
            <input
              ref={inputRef}
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Ketik perintah (contoh: cek stok barang, buka kasir POS)..."
              disabled={isStreaming}
              className="w-full pl-3 pr-10 py-2.5 text-xs rounded-lg border border-border bg-background text-foreground placeholder:text-muted-foreground focus:outline-hidden focus:ring-1 focus:ring-primary disabled:opacity-50"
            />
            <button
              onClick={() => handleSend()}
              disabled={!input.trim() || isStreaming}
              className={cn(
                "absolute right-1.5 p-1.5 rounded-md text-primary-foreground transition-colors cursor-pointer",
                input.trim() && !isStreaming
                  ? "bg-primary hover:bg-primary/90"
                  : "bg-muted text-muted-foreground cursor-not-allowed"
              )}
            >
              <Send className="h-3.5 w-3.5" />
            </button>
          </div>
          <div className="flex items-center justify-between mt-1.5 px-1 text-[10px] text-muted-foreground">
            <span>Tekan Enter untuk mengirim</span>
            <span>Esc / Klik luar untuk tutup</span>
          </div>
        </div>
      </aside>

      <AutonomySettingsModal />
    </>
  )
}
