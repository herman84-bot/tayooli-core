'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

export type AssignedWarehouse = { id: string; code: string; name: string }

export type TeamMember = {
  id: string
  email: string
  full_name: string
  role: string
  assigned_warehouses?: AssignedWarehouse[]
  created_at: string
}

export type InviteResponse = {
  message: string
  email_sent: boolean
  email_error?: string
  member: TeamMember
}

async function readError(res: Response, fallback: string): Promise<string> {
  const err = await res.json().catch(() => ({}))
  if (typeof err?.error === 'string' && err.error) return err.error
  return err?.error?.message || err?.message || fallback
}

export function useTeamMembers() {
  const queryClient = useQueryClient()
  const invalidate = () => queryClient.invalidateQueries({ queryKey: ['team-members'] })

  const members = useQuery<TeamMember[]>({
    queryKey: ['team-members'],
    queryFn: async () => {
      const res = await fetch('/api/v1/settings/team', { credentials: 'include' })
      if (!res.ok) throw new Error('Gagal memuat anggota tim')
      const data = await res.json()
      return data.members ?? []
    },
  })

  const inviteMember = useMutation({
    mutationFn: async ({ email, role, warehouseIds = [] }: { email: string; role: string; warehouseIds?: string[] }): Promise<InviteResponse> => {
      const res = await fetch('/api/v1/settings/team', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ email, role, warehouse_ids: warehouseIds }),
      })
      if (!res.ok) throw new Error(await readError(res, 'Gagal mengundang anggota'))
      return res.json()
    },
    onSuccess: invalidate,
  })

  const resendInvite = useMutation({
    mutationFn: async (userId: string): Promise<InviteResponse> => {
      const res = await fetch(`/api/v1/settings/team/${userId}/resend-invite`, {
        method: 'POST',
        credentials: 'include',
      })
      if (!res.ok) throw new Error(await readError(res, 'Gagal mengirim ulang undangan'))
      return res.json()
    },
  })

  const changeRole = useMutation({
    mutationFn: async ({ userId, role, warehouseIds = [] }: { userId: string; role: string; warehouseIds?: string[] }) => {
      const res = await fetch(`/api/v1/settings/team/${userId}/role`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ role, warehouse_ids: warehouseIds }),
      })
      if (!res.ok) throw new Error(await readError(res, 'Gagal mengubah role'))
      return res.json()
    },
    onSuccess: invalidate,
  })

  const removeMember = useMutation({
    mutationFn: async (userId: string) => {
      const res = await fetch(`/api/v1/settings/team/${userId}`, {
        method: 'DELETE',
        credentials: 'include',
      })
      if (!res.ok) throw new Error(await readError(res, 'Gagal menghapus anggota'))
      return res.json()
    },
    onSuccess: invalidate,
  })

  return {
    members: members.data ?? [],
    isLoading: members.isLoading,
    error: members.error,
    inviteMember,
    resendInvite,
    changeRole,
    removeMember,
  }
}
