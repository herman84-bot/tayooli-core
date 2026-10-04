'use client'

import { useState, useRef, useEffect } from 'react'
import { useAuth } from '@/hooks/useAuth'
import { useTeamMembers } from '@/hooks/useTeamMembers'
import { Settings, Users, Building2, Shield, UserPlus, MoreHorizontal, Trash2, RefreshCw, Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

type Tab = 'team' | 'profile'

const tabs: { id: Tab; label: string; icon: React.ElementType }[] = [
  { id: 'team', label: 'Tim', icon: Users },
  { id: 'profile', label: 'Perusahaan', icon: Building2 },
]

type DisplayMember = { id: string; name: string; email: string; role: string; status: string }

const roleColors: Record<string, string> = {
  owner: 'bg-primary/10 text-primary',
  admin: 'bg-blue-50 text-blue-600',
  member: 'bg-zinc-100 text-zinc-600',
  accountant: 'bg-purple-50 text-purple-600',
  approver: 'bg-amber-50 text-amber-600',
}

const roleLabels: Record<string, string> = {
  owner: 'Owner',
  admin: 'Admin',
  member: 'Member',
  accountant: 'Accountant',
  approver: 'Approver',
}

function MemberActions({ member, isOwner, onChangeRole, onRemove }: {
  member: DisplayMember
  isOwner: boolean
  onChangeRole: (userId: string, newRole: string) => void
  onRemove: (userId: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [showRoles, setShowRoles] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false)
        setShowRoles(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  if (isOwner) return null

  const roles = ['admin', 'member']

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => { setOpen(!open); setShowRoles(false) }}
        className="p-1 rounded text-muted-foreground hover:bg-muted transition-colors"
        aria-label="More options"
      >
        <MoreHorizontal className="h-4 w-4" />
      </button>

      {open && (
        <div className="absolute right-0 top-full mt-1 z-20 w-44 bg-background border border-border rounded-lg shadow-md py-1">
          {!showRoles ? (
            <>
              <button
                onClick={() => setShowRoles(true)}
                className="w-full flex items-center gap-2 px-3 py-2 text-sm text-foreground hover:bg-muted transition-colors"
              >
                <RefreshCw className="h-3.5 w-3.5 text-muted-foreground" />
                Ubah Role
              </button>
              <button
                onClick={() => { onRemove(member.id); setOpen(false) }}
                className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50 transition-colors"
              >
                <Trash2 className="h-3.5 w-3.5" />
                Hapus Anggota
              </button>
            </>
          ) : (
            <>
              <button
                onClick={() => setShowRoles(false)}
                className="w-full flex items-center gap-2 px-3 py-2 text-sm text-muted-foreground hover:bg-muted transition-colors"
              >
                ← Kembali
              </button>
              {roles.map((role) => (
                <button
                  key={role}
                  onClick={() => { onChangeRole(member.id, role); setOpen(false); setShowRoles(false) }}
                  className={cn(
                    'w-full flex items-center gap-2 px-3 py-2 text-sm transition-colors',
                    member.role === role ? 'text-primary font-medium bg-primary/5' : 'text-foreground hover:bg-muted'
                  )}
                >
                  {member.role === role && <span className="h-1.5 w-1.5 rounded-full bg-primary" />}
                  <span className={member.role === role ? '' : 'ml-[10px]'}>{roleLabels[role] || role}</span>
                </button>
              ))}
            </>
          )}
        </div>
      )}
    </div>
  )
}

function TeamTab() {
  const { members, isLoading, error, changeRole, removeMember } = useTeamMembers()
  const [showInvite, setShowInvite] = useState(false)

  const displayMembers: DisplayMember[] = members.map(m => ({
    id: m.id,
    name: m.full_name || m.email,
    email: m.email,
    role: m.role,
    status: 'active',
  }))

  const handleChangeRole = (userId: string, newRole: string) => {
    changeRole.mutate({ userId, role: newRole })
  }

  const handleRemove = (userId: string) => {
    removeMember.mutate(userId)
  }

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-sm font-semibold text-foreground">Anggota Tim</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            {isLoading ? 'Memuat...' : `${displayMembers.length} anggota`}
          </p>
        </div>
        <button
          onClick={() => setShowInvite(!showInvite)}
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors"
        >
          <UserPlus className="h-3.5 w-3.5" />
          Undang Anggota
        </button>
      </div>

      {showInvite && (
        <div className="mb-4 p-3 bg-muted/50 rounded-lg border border-border">
          <p className="text-xs font-medium text-foreground mb-2">Undang melalui email</p>
          <div className="flex gap-2">
            <input
              type="email"
              placeholder="email@perusahaan.com"
              className="flex-1 text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
            />
            <button className="px-3 py-1.5 text-xs font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors">
              Kirim
            </button>
          </div>
          <p className="text-[11px] text-muted-foreground mt-1.5">
            Undangan dikirim via email. Anggota akan mendapat akses ke workspace ini.
          </p>
        </div>
      )}

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
          Gagal memuat data tim. Coba muat ulang halaman.
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          <span className="ml-2 text-sm text-muted-foreground">Memuat anggota tim...</span>
        </div>
      ) : (
        <div className="border border-border rounded-lg">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="text-left px-3 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Nama</th>
                <th className="text-left px-3 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Email</th>
                <th className="text-left px-3 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Role</th>
                <th className="text-left px-3 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Status</th>
                <th className="w-10 px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {displayMembers.map((m) => (
                <tr key={m.id} className="border-b border-border last:border-0">
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-2">
                      <div className="h-7 w-7 rounded-full bg-primary/10 flex items-center justify-center text-[11px] font-semibold text-primary">
                        {m.name.charAt(0).toUpperCase()}
                      </div>
                      <span className="text-foreground font-medium">{m.name}</span>
                    </div>
                  </td>
                  <td className="px-3 py-2.5 text-muted-foreground">{m.email}</td>
                  <td className="px-3 py-2.5">
                    <span className={cn('inline-flex px-2 py-0.5 rounded text-[11px] font-medium', roleColors[m.role] || 'bg-zinc-100 text-zinc-600')}>
                      {roleLabels[m.role] || m.role}
                    </span>
                  </td>
                  <td className="px-3 py-2.5">
                    <span className="flex items-center gap-1.5 text-xs text-green-600">
                      <span className="h-1.5 w-1.5 rounded-full bg-green-500" />
                      Aktif
                    </span>
                  </td>
                  <td className="px-3 py-2.5">
                    <MemberActions
                      member={m}
                      isOwner={m.role === 'owner'}
                      onChangeRole={handleChangeRole}
                      onRemove={handleRemove}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-4 p-3 bg-muted/30 rounded-lg">
        <div className="flex items-start gap-2">
          <Shield className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
          <div>
            <p className="text-xs font-medium text-foreground">Tentang Role</p>
            <p className="text-[11px] text-muted-foreground mt-0.5 leading-relaxed">
              <strong>Owner</strong> — akses penuh, tidak bisa dihapus.{' '}
              <strong>Admin</strong> — bisa undang/hapus member, kelola pengaturan.{' '}
              <strong>Member</strong> — akses data, tidak bisa manage tim.
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

function ProfileTab() {
  return (
    <div>
      <h2 className="text-sm font-semibold text-foreground mb-1">Profil Perusahaan</h2>
      <p className="text-xs text-muted-foreground mb-4">Informasi ini ditampilkan di invoice dan dokumen.</p>

      <div className="space-y-3 max-w-lg">
        <div>
          <label className="block text-xs font-medium text-foreground mb-1">Nama Perusahaan</label>
          <input
            type="text"
            defaultValue="PT Tayooli Indonesia"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-foreground mb-1">Email</label>
          <input
            type="email"
            defaultValue="finance@tayooli.com"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-foreground mb-1">Telepon</label>
          <input
            type="tel"
            defaultValue="+62 21 555 0123"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-foreground mb-1">Alamat</label>
          <textarea
            rows={2}
            defaultValue="Jl. Sudirman No. 123, Jakarta Selatan"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50 resize-none"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-foreground mb-1">NPWP</label>
          <input
            type="text"
            defaultValue="12.345.678.9-013.000"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
          />
        </div>
        <button className="px-4 py-1.5 text-xs font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 transition-colors">
          Simpan Perubahan
        </button>
      </div>
    </div>
  )
}

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<Tab>('team')

  return (
    <div className="p-6 max-w-4xl">
      <div className="flex items-center gap-2.5 mb-5">
        <Settings className="h-5 w-5 text-muted-foreground" />
        <h1 className="text-lg font-semibold text-foreground">Settings</h1>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border mb-5">
        {tabs.map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setActiveTab(id)}
            className={cn(
              'flex items-center gap-1.5 px-3 py-2 text-sm font-medium border-b-2 transition-colors -mb-px',
              activeTab === id
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground',
            )}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      {activeTab === 'team' && <TeamTab />}
      {activeTab === 'profile' && <ProfileTab />}
    </div>
  )
}
