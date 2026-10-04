"use client"

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"

export interface AIPermissionsData {
  id: string
  tenant_id: string
  autonomy_level: "advisory" | "assisted" | "autopilot"
  allowed_scopes: string[]
  emergency_stop: boolean
  updated_by?: string
  created_at: string
  updated_at: string
}

export interface UpdateAIPermissionsPayload {
  autonomy_level?: "advisory" | "assisted" | "autopilot"
  allowed_scopes?: string[]
  emergency_stop?: boolean
}

export function useAIPermissions() {
  return useQuery<AIPermissionsData>({
    queryKey: ["ai-permissions"],
    queryFn: async () => {
      const res = await fetch("/api/v1/ai/permissions", {
        credentials: "include",
      })
      if (!res.ok) {
        // Fallback default for demo/offline
        return {
          id: "default",
          tenant_id: "default",
          autonomy_level: "assisted",
          allowed_scopes: ["workspace.read", "workspace.profile_write", "workspace.master_write"],
          emergency_stop: false,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        }
      }
      return res.json()
    },
    staleTime: 60 * 1000,
  })
}

export function useUpdateAIPermissions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (payload: UpdateAIPermissionsPayload) => {
      const res = await fetch("/api/v1/ai/permissions", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err?.error?.message || "Failed to update AI permissions")
      }
      return res.json()
    },
    onSuccess: (data) => {
      qc.setQueryData(["ai-permissions"], data)
      qc.invalidateQueries({ queryKey: ["ai-permissions"] })
    },
  })
}
