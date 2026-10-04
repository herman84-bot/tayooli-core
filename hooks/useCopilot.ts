"use client"

import { create } from "zustand"

export interface ActionDiff {
  field: string
  oldValue: any
  newValue: any
}

export interface ProposedAction {
  id: string
  toolName: string
  title: string
  description: string
  riskLevel: "low" | "medium" | "high"
  requiresConfirmationText?: string // e.g., "KONFIRMASI" for high risk
  diff: ActionDiff[]
  payload: Record<string, any>
  status: "pending" | "executing" | "executed" | "cancelled" | "failed"
  error?: string
  blockedReasons?: string[]
  unsupportedFields?: { field: string; value: any; reason: string }[]
  sourceQuote?: string
}

export interface CopilotMessage {
  id: string
  role: "user" | "assistant" | "system"
  content: string
  timestamp: string
  proposedActions?: ProposedAction[]
  providerUsed?: "gemini" | "secondary" | "offline"
}

export interface ExecutedActionSnapshot {
  id: string
  toolName: string
  timestamp: number
  restorePayload?: Record<string, any>
  description: string
  entityId?: string
  undoPath?: string
  undoMethod?: string
}

export type AutonomyLevel = "advisory" | "assisted" | "autopilot"

const getInitialAutonomyLevel = (): AutonomyLevel => {
  if (typeof window === "undefined") return "assisted"
  try {
    const stored = localStorage.getItem("tayooli_copilot_autonomy_level")
    if (stored === "advisory" || stored === "assisted" || stored === "autopilot") {
      return stored
    }
  } catch {
    // Ignore localStorage access errors
  }
  return "assisted"
}

interface CopilotState {
  isOpen: boolean
  isSettingsOpen: boolean
  isStreaming: boolean
  activeProvider: "gemini" | "secondary" | "offline"
  autonomyLevel: AutonomyLevel
  cooldownSeconds: number
  messages: CopilotMessage[]
  pendingActions: ProposedAction[]
  lastUndoSnapshot: ExecutedActionSnapshot | null

  // Actions
  open: () => void
  close: () => void
  toggle: () => void
  openSettings: () => void
  closeSettings: () => void
  setIsStreaming: (streaming: boolean) => void
  setActiveProvider: (provider: "gemini" | "secondary" | "offline") => void
  setAutonomyLevel: (level: AutonomyLevel) => void
  setCooldownSeconds: (sec: number) => void
  addMessage: (message: CopilotMessage) => void
  updateLastAssistantMessage: (content: string, actions?: ProposedAction[]) => void
  clearMessages: () => void
  addPendingAction: (action: ProposedAction) => void
  updateActionStatus: (id: string, status: ProposedAction["status"], error?: string) => void
  removePendingAction: (id: string) => void
  clearPendingActions: () => void
  setLastUndoSnapshot: (snapshot: ExecutedActionSnapshot | null) => void
}

export const useCopilot = create<CopilotState>((set) => ({
  isOpen: false,
  isSettingsOpen: false,
  isStreaming: false,
  activeProvider: "gemini",
  autonomyLevel: getInitialAutonomyLevel(),
  cooldownSeconds: 0,
  messages: [
    {
      id: "initial-system-msg",
      role: "assistant",
      content: "Halo! Saya Tayooli Copilot. Anda dapat menugaskan saya untuk mengatur workspace, mengundang anggota tim, konfigurasi profil PT, payment gateway, maupun membuat data master.",
      timestamp: new Date().toISOString(),
    },
  ],
  pendingActions: [],
  lastUndoSnapshot: null,

  open: () => set({ isOpen: true }),
  close: () => set({ isOpen: false }),
  toggle: () => set((state) => ({ isOpen: !state.isOpen })),
  openSettings: () => set({ isSettingsOpen: true }),
  closeSettings: () => set({ isSettingsOpen: false }),
  setIsStreaming: (isStreaming) => set({ isStreaming }),
  setActiveProvider: (activeProvider) => set({ activeProvider }),
  setAutonomyLevel: (autonomyLevel) => {
    if (typeof window !== "undefined") {
      try {
        localStorage.setItem("tayooli_copilot_autonomy_level", autonomyLevel)
      } catch {
        // Ignore localStorage quota or access errors
      }
    }
    set({ autonomyLevel })
  },
  setCooldownSeconds: (cooldownSeconds) => set({ cooldownSeconds }),

  addMessage: (message) =>
    set((state) => ({
      messages: [...state.messages, message],
    })),

  updateLastAssistantMessage: (content, actions) =>
    set((state) => {
      const messages = [...state.messages]
      const lastIdx = messages.findLastIndex((m) => m.role === "assistant")
      if (lastIdx !== -1) {
        messages[lastIdx] = {
          ...messages[lastIdx],
          content,
          ...(actions ? { proposedActions: actions } : {}),
        }
      }
      return { messages }
    }),

  clearMessages: () =>
    set({
      messages: [
        {
          id: "fresh-msg",
          role: "assistant",
          content: "Sesi obrolan direset. Ada yang bisa saya bantu di workspace Anda?",
          timestamp: new Date().toISOString(),
        },
      ],
      pendingActions: [],
    }),

  addPendingAction: (action) =>
    set((state) => ({
      pendingActions: [...state.pendingActions, action],
    })),

  updateActionStatus: (id, status, error) =>
    set((state) => ({
      pendingActions: state.pendingActions.map((a) =>
        a.id === id ? { ...a, status, error } : a
      ),
      messages: state.messages.map((m) => ({
        ...m,
        proposedActions: m.proposedActions?.map((a) =>
          a.id === id ? { ...a, status, error } : a
        ),
      })),
    })),

  removePendingAction: (id) =>
    set((state) => ({
      pendingActions: state.pendingActions.filter((a) => a.id !== id),
    })),

  clearPendingActions: () => set({ pendingActions: [] }),

  setLastUndoSnapshot: (lastUndoSnapshot) => set({ lastUndoSnapshot }),
}))
