'use client'

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

type TeamMember = {
  id: string
  email: string
  full_name: string
  role: string
  created_at: string
}

export function useTeamMembers() {
  const queryClient = useQueryClient()

  const members = useQuery<TeamMember[]>({
    queryKey: ['team-members'],
    queryFn: async () => {
      const res = await fetch('/api/v1/settings/team', { credentials: 'include' })
      if (!res.ok) throw new Error('Gagal memuat anggota tim')
      const data = await res.json()
      return data.members ?? []
    },
  })

  const changeRole = useMutation({
    mutationFn: async ({ userId, role }: { userId: string; role: string }) => {
      const res = await fetch(`/api/v1/settings/team/${userId}/role`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ role }),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error?.message || 'Gagal mengubah role')
      }
      return res.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['team-members'] })
    },
  })

  const removeMember = useMutation({
    mutationFn: async (userId: string) => {
      const res = await fetch(`/api/v1/settings/team/${userId}`, {
        method: 'DELETE',
        credentials: 'include',
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        throw new Error(err.error?.message || 'Gagal menghapus anggota')
      }
      return res.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['team-members'] })
    },
  })

  return {
    members: members.data ?? [],
    isLoading: members.isLoading,
    error: members.error,
    changeRole,
    removeMember,
  }
}
