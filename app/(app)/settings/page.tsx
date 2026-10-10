'use client'

import { useState, useRef, useEffect } from 'react'
import { useAuth } from '@/hooks/useAuth'
import { useTeamMembers } from '@/hooks/useTeamMembers'
import { useQuery } from '@tanstack/react-query'
import { api, type Warehouse } from '@/lib/api'
import { type TeamMember } from '@/hooks/useTeamMembers'
import { ASSIGNABLE_ROLES, ROLE_COLORS, ROLE_DESCRIPTIONS, ROLE_LABELS, requiresWarehouse, type AppRole } from '@/lib/rbac'
import { usePaymentConfig } from '@/hooks/usePaymentConfig'
import { useCompanyProfile } from '@/hooks/useCompanyProfile'
import {
  Settings,
  Users,
  Building2,
  Shield,
  UserPlus,
  MoreHorizontal,
  Trash2,
  RefreshCw,
  Loader2,
  CreditCard,
  CheckCircle2,
  Copy,
  Check,
  Eye,
  EyeOff,
  ExternalLink,
  AlertCircle,
  Info,
  Lock,
} from 'lucide-react'
import { cn } from '@/lib/utils'

type Tab = 'team' | 'profile' | 'payments'

const tabs: { id: Tab; label: string; icon: React.ElementType }[] = [
  { id: 'team', label: 'Tim', icon: Users },
  { id: 'profile', label: 'Perusahaan', icon: Building2 },
  { id: 'payments', label: 'Metode Pembayaran', icon: CreditCard },
]

function WarehousePicker({ warehouses, selected, onChange, loading }: {
  warehouses: Warehouse[]
  selected: string[]
  onChange: (ids: string[]) => void
  loading: boolean
}) {
  if (loading) return <p className="text-[11px] text-muted-foreground">Memuat daftar gudang...</p>
  if (warehouses.length === 0) {
    return <p className="text-[11px] text-amber-600">Belum ada gudang. Buat gudang dulu di menu Warehouse &amp; Stock.</p>
  }
  const toggle = (id: string) =>
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id])
  return (
    <fieldset className="mt-2">
      <legend className="text-[11px] font-medium text-foreground mb-1">Gudang yang ditugaskan (wajib minimal 1)</legend>
      <div className="grid gap-1 sm:grid-cols-2 max-h-40 overflow-auto">
        {warehouses.map((w) => (
          <label key={w.id} className="flex items-center gap-2 text-xs text-foreground px-2 py-1 rounded hover:bg-muted cursor-pointer">
            <input type="checkbox" checked={selected.includes(w.id)} onChange={() => toggle(w.id)} />
            <span>{w.name} <span className="text-muted-foreground">({w.code})</span></span>
          </label>
        ))}
      </div>
    </fieldset>
  )
}

function RoleSelect({ value, onChange, id }: { value: string; onChange: (v: string) => void; id: string }) {
  return (
    <div>
      <select
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30"
      >
        {ASSIGNABLE_ROLES.map((r) => (
          <option key={r} value={r}>{ROLE_LABELS[r]}</option>
        ))}
      </select>
      <p className="text-[11px] text-muted-foreground mt-1">{ROLE_DESCRIPTIONS[value]}</p>
    </div>
  )
}

function useWarehouseOptions(enabled: boolean) {
  return useQuery({
    queryKey: ['wms-warehouses-options'],
    queryFn: async () => (await api.wms.warehouses.list()).data ?? [],
    enabled,
    staleTime: 60_000,
  })
}

function EditRoleDialog({ member, onClose }: { member: TeamMember; onClose: () => void }) {
  const { changeRole } = useTeamMembers()
  const [role, setRole] = useState(ASSIGNABLE_ROLES.includes(member.role as AppRole) ? member.role : 'member')
  const [whIds, setWhIds] = useState<string[]>((member.assigned_warehouses ?? []).map((w) => w.id))
  const needsWh = requiresWarehouse(role)
  const warehouses = useWarehouseOptions(needsWh)
  const invalid = needsWh && whIds.length === 0

  const submit = () => {
    if (invalid) return
    changeRole.mutate(
      { userId: member.id, role, warehouseIds: needsWh ? whIds : [] },
      { onSuccess: onClose },
    )
  }

  return (
    <div role="dialog" aria-modal="true" aria-labelledby="edit-role-title" className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4">
      <div className="w-full max-w-md rounded-lg border border-border bg-background p-4 shadow-lg">
        <h3 id="edit-role-title" className="text-sm font-semibold text-foreground">Ubah Role</h3>
        <p className="text-xs text-muted-foreground mb-3">{member.email}</p>
        <label htmlFor="edit-role" className="text-[11px] font-medium text-foreground">Role</label>
        <RoleSelect id="edit-role" value={role} onChange={setRole} />
        {needsWh && (
          <WarehousePicker warehouses={warehouses.data ?? []} loading={warehouses.isLoading} selected={whIds} onChange={setWhIds} />
        )}
        {changeRole.error && <p role="alert" className="mt-2 text-xs text-red-600">{(changeRole.error as Error).message}</p>}
        <div className="mt-4 flex justify-end gap-2">
          <button onClick={onClose} className="px-3 py-1.5 text-xs rounded-md border border-border hover:bg-muted">Batal</button>
          <button
            onClick={submit}
            disabled={invalid || changeRole.isPending}
            className="px-3 py-1.5 text-xs font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
          >
            {changeRole.isPending ? 'Menyimpan...' : 'Simpan'}
          </button>
        </div>
      </div>
    </div>
  )
}

function MemberActions({ member, onEdit, onRemove }: {
  member: TeamMember
  onEdit: (m: TeamMember) => void
  onRemove: (userId: string) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  if (member.role === 'owner') return null

  return (
    <div className="relative" ref={ref}>
      <button
        onClick={() => setOpen(!open)}
        className="p-1 rounded text-muted-foreground hover:bg-muted transition-colors"
        aria-label="More options"
      >
        <MoreHorizontal className="h-4 w-4" />
      </button>
      {open && (
        <div className="absolute right-0 top-full mt-1 z-20 w-44 bg-background border border-border rounded-lg shadow-md py-1">
          <button
            onClick={() => { onEdit(member); setOpen(false) }}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-foreground hover:bg-muted transition-colors"
          >
            <RefreshCw className="h-3.5 w-3.5 text-muted-foreground" />
            Ubah Role &amp; Gudang
          </button>
          <button
            onClick={() => {
              if (window.confirm(`Hapus ${member.email} dari tim?`)) onRemove(member.id)
              setOpen(false)
            }}
            className="w-full flex items-center gap-2 px-3 py-2 text-sm text-red-600 hover:bg-red-50 transition-colors"
          >
            <Trash2 className="h-3.5 w-3.5" />
            Hapus Anggota
          </button>
        </div>
      )}
    </div>
  )
}

function InviteForm({ onDone }: { onDone: () => void }) {
  const { inviteMember } = useTeamMembers()
  const [email, setEmail] = useState('')
  const [role, setRole] = useState<string>('warehouse')
  const [whIds, setWhIds] = useState<string[]>([])
  const needsWh = requiresWarehouse(role)
  const warehouses = useWarehouseOptions(needsWh)
  const emailValid = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())
  const invalid = !emailValid || (needsWh && whIds.length === 0)

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (invalid) return
    inviteMember.mutate(
      { email: email.trim(), role, warehouseIds: needsWh ? whIds : [] },
      { onSuccess: () => { setEmail(''); setWhIds([]); onDone() } },
    )
  }

  return (
    <form onSubmit={submit} className="mb-4 p-3 bg-muted/50 rounded-lg border border-border space-y-2">
      <p className="text-xs font-medium text-foreground">Undang anggota baru</p>
      <div className="grid gap-2 sm:grid-cols-2">
        <div>
          <label htmlFor="invite-email" className="text-[11px] font-medium text-foreground">Email</label>
          <input
            id="invite-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="email@perusahaan.com"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground placeholder:text-muted-foreground outline-none focus:ring-1 focus:ring-primary/30"
          />
        </div>
        <div>
          <label htmlFor="invite-role" className="text-[11px] font-medium text-foreground">Role</label>
          <RoleSelect id="invite-role" value={role} onChange={setRole} />
        </div>
      </div>
      {needsWh && (
        <WarehousePicker warehouses={warehouses.data ?? []} loading={warehouses.isLoading} selected={whIds} onChange={setWhIds} />
      )}
      {inviteMember.error && <p role="alert" className="text-xs text-red-600">{(inviteMember.error as Error).message}</p>}
      <div className="flex justify-end">
        <button
          type="submit"
          disabled={invalid || inviteMember.isPending}
          className="px-3 py-1.5 text-xs font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50"
        >
          {inviteMember.isPending ? 'Mengirim...' : 'Undang'}
        </button>
      </div>
    </form>
  )
}

function TeamTab() {
  const { members, isLoading, error, removeMember } = useTeamMembers()
  const [showInvite, setShowInvite] = useState(false)
  const [editing, setEditing] = useState<TeamMember | null>(null)

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-sm font-semibold text-foreground">Anggota Tim</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            {isLoading ? 'Memuat...' : `${members.length} anggota`}
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

      {showInvite && <InviteForm onDone={() => setShowInvite(false)} />}

      {error && (
        <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
          Gagal memuat data tim. Coba muat ulang halaman.
        </div>
      )}
      {removeMember.error && (
        <div role="alert" className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600">
          {(removeMember.error as Error).message}
        </div>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          <span className="ml-2 text-sm text-muted-foreground">Memuat anggota tim...</span>
        </div>
      ) : (
        <div className="border border-border rounded-lg overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="text-left px-3 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Nama</th>
                <th className="text-left px-3 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Role</th>
                <th className="text-left px-3 py-2 text-[11px] font-medium text-muted-foreground uppercase tracking-wider">Gudang</th>
                <th className="w-10 px-3 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {members.map((m) => (
                <tr key={m.id} className="border-b border-border last:border-0">
                  <td className="px-3 py-2.5">
                    <div className="flex items-center gap-2">
                      <div className="h-7 w-7 rounded-full bg-primary/10 flex items-center justify-center text-[11px] font-semibold text-primary">
                        {(m.full_name || m.email).charAt(0).toUpperCase()}
                      </div>
                      <div>
                        <p className="text-foreground font-medium">{m.full_name || m.email}</p>
                        <p className="text-[11px] text-muted-foreground">{m.email}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={cn('inline-flex px-2 py-0.5 rounded text-[11px] font-medium', ROLE_COLORS[m.role] || 'bg-zinc-100 text-zinc-600')}>
                      {ROLE_LABELS[m.role] || m.role}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-xs text-muted-foreground">
                    {m.assigned_warehouses && m.assigned_warehouses.length > 0
                      ? m.assigned_warehouses.map((w) => w.name).join(', ')
                      : requiresWarehouse(m.role) ? <span className="text-amber-600">Belum ditugaskan</span> : 'Semua / tidak relevan'}
                  </td>
                  <td className="px-3 py-2.5">
                    <MemberActions member={m} onEdit={setEditing} onRemove={(id) => removeMember.mutate(id)} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {editing && <EditRoleDialog member={editing} onClose={() => setEditing(null)} />}

      <div className="mt-4 p-3 bg-muted/30 rounded-lg">
        <div className="flex items-start gap-2">
          <Shield className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
          <div>
            <p className="text-xs font-medium text-foreground">Tentang Role</p>
            <ul className="text-[11px] text-muted-foreground mt-0.5 leading-relaxed space-y-0.5">
              <li><strong>Owner</strong>: akses penuh, tidak bisa dihapus atau diubah.</li>
              {ASSIGNABLE_ROLES.map((r) => (
                <li key={r}><strong>{ROLE_LABELS[r]}</strong>: {ROLE_DESCRIPTIONS[r]}</li>
              ))}
            </ul>
          </div>
        </div>
      </div>
    </div>
  )
}


function ProfileTab() {
  const { user } = useAuth()
  const { profile, isLoading, updateProfile } = useCompanyProfile()

  const [companyName, setCompanyName] = useState('')
  const [division, setDivision] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [address, setAddress] = useState('')
  const [taxId, setTaxId] = useState('')
  const [website, setWebsite] = useState('')
  const [saveSuccess, setSaveSuccess] = useState(false)
  const [saveError, setSaveError] = useState<string | null>(null)

  const isAuthorized = user?.role === 'owner' || user?.role === 'admin'

  useEffect(() => {
    if (profile) {
      setCompanyName(profile.name || profile.company_name || '')
      setDivision(profile.division || '')
      setEmail(profile.email || '')
      setPhone(profile.phone || '')
      setAddress(profile.address || '')
      setTaxId(profile.tax_id || '')
      setWebsite(profile.website || '')
    }
  }, [profile])

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaveError(null)
    setSaveSuccess(false)

    const trimmedName = companyName.trim()
    if (trimmedName.length < 2) {
      setSaveError('Nama perusahaan minimal 2 karakter.')
      return
    }

    try {
      await updateProfile.mutateAsync({
        name: trimmedName,
        division: division.trim() || undefined,
        email: email.trim() || undefined,
        phone: phone.trim() || undefined,
        address: address.trim() || undefined,
        tax_id: taxId.trim() || undefined,
        website: website.trim() || undefined,
      })
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 4000)
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err.message : 'Gagal menyimpan profil perusahaan')
    }
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-12">
        <Loader2 className="h-6 w-6 animate-spin text-primary" />
        <span className="ml-2 text-xs text-muted-foreground">Memuat data perusahaan...</span>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-semibold text-foreground mb-1">Profil Bisnis / Perusahaan</h2>
        <p className="text-xs text-muted-foreground">
          Informasi ini digunakan otomatis pada kop Surat Jalan (Delivery Order), Faktur Penjualan, Berita Acara, dan Struk Kasir.
        </p>
      </div>

      {/* Mini Live Preview Banner */}
      <div className="rounded-lg border border-border/80 bg-muted/20 p-4 max-w-xl">
        <div className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground mb-2 flex items-center gap-1.5">
          <Eye className="h-3.5 w-3.5 text-primary" /> Pratinjau Kop Surat Cetak
        </div>
        <div className="rounded border border-border bg-card p-3 shadow-xs flex items-start gap-3">
          <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-zinc-900 text-white font-black text-lg">
            {companyName.trim().charAt(0).toUpperCase() || 'P'}
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-xs font-bold text-foreground truncate uppercase">
              {companyName.trim() || 'NAMA PERUSAHAAN ANDA'}
            </div>
            <div className="text-[11px] font-medium text-muted-foreground truncate">
              {division.trim() || 'Divisi Logistik & Pergudangan Terpadu'}
            </div>
            <div className="text-[10px] text-muted-foreground/80 truncate mt-0.5">
              {address.trim() || 'Alamat operasional perusahaan'} &bull; Telp: {phone.trim() || '—'}
            </div>
          </div>
        </div>
      </div>

      <form onSubmit={handleSave} className="space-y-4 max-w-lg">
        {saveSuccess && (
          <div className="flex items-center gap-2 rounded-lg border border-emerald-500/20 bg-emerald-500/10 p-3 text-xs text-emerald-700 dark:text-emerald-400">
            <CheckCircle2 className="h-4 w-4 shrink-0" />
            <span>Profil perusahaan berhasil disimpan dan disinkronkan ke seluruh dokumen cetak.</span>
          </div>
        )}

        {saveError && (
          <div className="flex items-center gap-2 rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
            <AlertCircle className="h-4 w-4 shrink-0" />
            <span>{saveError}</span>
          </div>
        )}

        <div>
          <label className="block text-xs font-medium text-foreground mb-1">
            Nama Perusahaan / Toko <span className="text-destructive">*</span>
          </label>
          <input
            type="text"
            required
            disabled={!isAuthorized}
            value={companyName}
            onChange={(e) => setCompanyName(e.target.value)}
            placeholder="PT Maju Logistik Sentral"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50 disabled:opacity-60"
          />
        </div>

        <div>
          <label className="block text-xs font-medium text-foreground mb-1">Divisi / Unit Operasional</label>
          <input
            type="text"
            disabled={!isAuthorized}
            value={division}
            onChange={(e) => setDivision(e.target.value)}
            placeholder="Divisi Logistik & Pergudangan Terpadu"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50 disabled:opacity-60"
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-foreground mb-1">Email Resmi</label>
            <input
              type="email"
              disabled={!isAuthorized}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="kontak@perusahaan.co.id"
              className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50 disabled:opacity-60"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-foreground mb-1">Telepon Kantor</label>
            <input
              type="tel"
              disabled={!isAuthorized}
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="(021) 555-0123"
              className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50 disabled:opacity-60"
            />
          </div>
        </div>

        <div>
          <label className="block text-xs font-medium text-foreground mb-1">Alamat Kantor / Gudang</label>
          <textarea
            rows={2}
            disabled={!isAuthorized}
            value={address}
            onChange={(e) => setAddress(e.target.value)}
            placeholder="Kawasan Industri Pulogadung Blok B No. 12, Jakarta Timur"
            className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50 resize-none disabled:opacity-60"
          />
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-foreground mb-1">NPWP / Tax ID</label>
            <input
              type="text"
              disabled={!isAuthorized}
              value={taxId}
              onChange={(e) => setTaxId(e.target.value)}
              placeholder="01.234.567.8-901.000"
              className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50 disabled:opacity-60"
            />
          </div>
          <div>
            <label className="block text-xs font-medium text-foreground mb-1">Website</label>
            <input
              type="text"
              disabled={!isAuthorized}
              value={website}
              onChange={(e) => setWebsite(e.target.value)}
              placeholder="www.perusahaan.co.id"
              className="w-full text-sm px-3 py-1.5 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50 disabled:opacity-60"
            />
          </div>
        </div>

        {isAuthorized ? (
          <button
            type="submit"
            disabled={updateProfile.isPending}
            className="inline-flex items-center gap-1.5 px-4 py-2 text-xs font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-60 transition-colors shadow-xs"
          >
            {updateProfile.isPending ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                <span>Menyimpan...</span>
              </>
            ) : (
              <span>Simpan Perubahan</span>
            )}
          </button>
        ) : (
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground p-2 rounded bg-muted/40 border border-border">
            <Lock className="h-3.5 w-3.5 text-muted-foreground" />
            <span>Hanya Administrator atau Pemilik Akun yang berwenang memperbarui profil bisnis.</span>
          </div>
        )}
      </form>
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
      {activeTab === 'payments' && <PaymentsTab />}
    </div>
  )
}

function PaymentsTab() {
  const [provider, setProvider] = useState<'midtrans' | 'pakasir'>('midtrans')
  const { config, isLoading, saveConfig } = usePaymentConfig(provider)

  const [serverKey, setServerKey] = useState('')
  const [clientKey, setClientKey] = useState('')
  const [slug, setSlug] = useState('')
  const [apiKey, setApiKey] = useState('')
  const [isProduction, setIsProduction] = useState(false)
  const [isActive, setIsActive] = useState(true)
  const [bankName, setBankName] = useState('')
  const [bankAccount, setBankAccount] = useState('')
  const [holderName, setHolderName] = useState('')

  const [showKey, setShowKey] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState(false)
  const [copiedWebhook, setCopiedWebhook] = useState(false)

  useEffect(() => {
    if (config) {
      setClientKey(config.client_key || '')
      setSlug(config.slug || '')
      setIsProduction(Boolean(config.is_production))
      setIsActive(config.is_active ?? true)
      setBankName(config.settlement_bank_name || '')
      setBankAccount(config.settlement_bank_account || '')
      setHolderName(config.settlement_holder_name || '')
      setServerKey('')
      setApiKey('')
    }
  }, [config, provider])

  const handleCopyWebhook = () => {
    const origin = typeof window !== 'undefined' ? window.location.origin : 'https://tayooli.my.id'
    const url = `${origin}/api/v1/webhooks/midtrans`
    navigator.clipboard.writeText(url)
    setCopiedWebhook(true)
    setTimeout(() => setCopiedWebhook(false), 2500)
  }

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaveSuccess(false)
    try {
      await saveConfig.mutateAsync({
        provider,
        server_key: serverKey ? serverKey.trim() : undefined,
        client_key: clientKey ? clientKey.trim() : undefined,
        slug: slug ? slug.trim() : undefined,
        api_key: apiKey ? apiKey.trim() : undefined,
        is_production: isProduction,
        is_active: isActive,
        settlement_bank_name: bankName ? bankName.trim() : undefined,
        settlement_bank_account: bankAccount ? bankAccount.trim() : undefined,
        settlement_holder_name: holderName ? holderName.trim() : undefined,
        gateway_fee_percent: 0.7,
      })
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 4000)
    } catch {
      // Error handled by saveConfig.error
    }
  }

  const hasConfig = Boolean(config?.hasCredentials)
  const isLive = hasConfig && config?.is_active

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-semibold text-foreground">Integrasi Payment Gateway (BYO Settlement)</h2>
        <p className="text-xs text-muted-foreground mt-0.5">
          Hubungkan akun payment gateway Anda sendiri. Dana penjualan kasir (QRIS) langsung disalurkan oleh gateway ke rekening bank toko Anda (ADR-008 Model B).
        </p>
      </div>

      {/* Provider Selector Tabs */}
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => setProvider('midtrans')}
          className={cn(
            'flex items-center gap-2 px-4 py-2 text-xs font-medium rounded-lg border transition-all',
            provider === 'midtrans'
              ? 'bg-primary/5 border-primary text-primary font-semibold shadow-xs'
              : 'border-border bg-card text-muted-foreground hover:bg-muted'
          )}
        >
          <span className="w-2 h-2 rounded-full bg-blue-500" />
          Midtrans (QRIS & Core API)
        </button>
        <button
          type="button"
          onClick={() => setProvider('pakasir')}
          className={cn(
            'flex items-center gap-2 px-4 py-2 text-xs font-medium rounded-lg border transition-all',
            provider === 'pakasir'
              ? 'bg-primary/5 border-primary text-primary font-semibold shadow-xs'
              : 'border-border bg-card text-muted-foreground hover:bg-muted'
          )}
        >
          <span className="w-2 h-2 rounded-full bg-emerald-500" />
          Pakasir (Fallback UMKM)
        </button>
      </div>

      {/* Gateway Status Badge Banner */}
      <div className={cn(
        'p-3.5 rounded-xl border flex items-start gap-3',
        isLive
          ? 'bg-emerald-50/60 border-emerald-200 text-emerald-900'
          : 'bg-amber-50/60 border-amber-200 text-amber-900'
      )}>
        {isLive ? (
          <CheckCircle2 className="h-5 w-5 text-emerald-600 shrink-0 mt-0.5" />
        ) : (
          <AlertCircle className="h-5 w-5 text-amber-600 shrink-0 mt-0.5" />
        )}
        <div className="text-xs">
          <p className="font-semibold">
            {isLive
              ? `Gateway ${provider === 'midtrans' ? 'Midtrans' : 'Pakasir'} Aktif (${config?.is_production ? 'Live Production' : 'Sandbox Testing'})`
              : 'Mode Demo Aktif'}
          </p>
          <p className="mt-0.5 leading-relaxed opacity-90">
            {isLive
              ? 'Pembayaran QRIS di kasir POS akan membuat tagihan riil dan dicek langsung ke sistem gateway.'
              : 'Belum ada kredensial aktif. Kasir POS saat ini menjalankan simulasi pembayaran instan (Mode Demo) agar kasir dapat diuji tanpa uang riil.'}
          </p>
        </div>
      </div>

      {isLoading ? (
        <div className="flex items-center justify-center py-8">
          <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
          <span className="ml-2 text-xs text-muted-foreground">Memuat konfigurasi pembayaran...</span>
        </div>
      ) : (
        <form onSubmit={handleSave} className="space-y-5 max-w-xl">
          {/* Active Status & Environment */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 p-3.5 rounded-xl border border-border bg-muted/20">
            <label className="flex items-center gap-2.5 cursor-pointer text-xs font-medium text-foreground">
              <input
                type="checkbox"
                checked={isActive}
                onChange={(e) => setIsActive(e.target.checked)}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/30"
              />
              <span>Aktifkan Gateway Ini</span>
            </label>

            <label className="flex items-center gap-2.5 cursor-pointer text-xs font-medium text-foreground">
              <input
                type="checkbox"
                checked={isProduction}
                onChange={(e) => setIsProduction(e.target.checked)}
                className="h-4 w-4 rounded border-border text-primary focus:ring-primary/30"
              />
              <span>Gunakan Mode Live (Production)</span>
            </label>
          </div>

          {/* Credentials Inputs */}
          {provider === 'midtrans' ? (
            <div className="space-y-3">
              <div>
                <div className="flex items-center justify-between mb-1">
                  <label className="text-xs font-medium text-foreground">Server Key Midtrans</label>
                  <a
                    href="https://dashboard.midtrans.com/"
                    target="_blank"
                    rel="noreferrer"
                    className="text-[11px] text-primary hover:underline flex items-center gap-1"
                  >
                    Buka Midtrans Dashboard <ExternalLink className="h-3 w-3" />
                  </a>
                </div>
                <div className="relative">
                  <input
                    type={showKey ? 'text' : 'password'}
                    value={serverKey}
                    onChange={(e) => setServerKey(e.target.value)}
                    placeholder={hasConfig ? '•••••••••••••••• (tersimpan, isi jika ingin ubah)' : 'Contoh: SB-Mid-server-xxxx (Sandbox) atau Mid-server-xxxx'}
                    className="w-full text-xs font-mono px-3 py-2 pr-10 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
                  />
                  <button
                    type="button"
                    onClick={() => setShowKey(!showKey)}
                    className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                    title={showKey ? 'Sembunyikan' : 'Tampilkan'}
                  >
                    {showKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                <p className="text-[11px] text-muted-foreground mt-1">
                  Didapat dari menu <em>Settings &gt; Access Keys</em> di dashboard Midtrans merchant Anda.
                </p>
              </div>

              <div>
                <label className="block text-xs font-medium text-foreground mb-1">Client Key Midtrans</label>
                <input
                  type="text"
                  value={clientKey}
                  onChange={(e) => setClientKey(e.target.value)}
                  placeholder="Contoh: SB-Mid-client-xxxx"
                  className="w-full text-xs font-mono px-3 py-2 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
                />
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-foreground mb-1">Project Slug Pakasir</label>
                <input
                  type="text"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value)}
                  placeholder="Contoh: toko-berkah-jaya"
                  className="w-full text-xs px-3 py-2 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-foreground mb-1">API Key Pakasir</label>
                <input
                  type="password"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder={hasConfig ? '•••••••••••••••• (tersimpan, isi jika ingin ubah)' : 'Masukkan API Key Pakasir'}
                  className="w-full text-xs font-mono px-3 py-2 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
                />
              </div>
            </div>
          )}

          {/* Webhook notification helper */}
          {provider === 'midtrans' && (
            <div className="p-3 bg-muted/40 rounded-lg border border-border space-y-1.5">
              <div className="flex items-center justify-between">
                <span className="text-xs font-semibold text-foreground flex items-center gap-1.5">
                  <Lock className="h-3.5 w-3.5 text-primary" /> Webhook Notification URL
                </span>
                <button
                  type="button"
                  onClick={handleCopyWebhook}
                  className="inline-flex items-center gap-1 text-[11px] font-medium text-primary hover:underline"
                >
                  {copiedWebhook ? (
                    <>
                      <Check className="h-3 w-3 text-emerald-600" />
                      <span className="text-emerald-600">Disalin!</span>
                    </>
                  ) : (
                    <>
                      <Copy className="h-3 w-3" />
                      Salin URL
                    </>
                  )}
                </button>
              </div>
              <p className="text-[11px] text-muted-foreground">
                Tempel URL berikut pada dashboard Midtrans (<em>Settings &gt; Configuration &gt; Payment Notification URL</em>) agar kasir menerima notifikasi lunas instan:
              </p>
              <div className="font-mono text-[11px] bg-background px-2.5 py-1.5 rounded border border-border select-all break-all">
                {typeof window !== 'undefined' ? `${window.location.origin}/api/v1/webhooks/midtrans` : 'https://tayooli.my.id/api/v1/webhooks/midtrans'}
              </div>
            </div>
          )}

          {/* Settlement Bank Details */}
          <div className="space-y-3 pt-2 border-t border-border">
            <h3 className="text-xs font-semibold text-foreground">Rekening Bank Pencairan (Settlement)</h3>
            <p className="text-[11px] text-muted-foreground">
              Rekening tujuan di mana Midtrans mencairkan saldo QRIS Anda secara otomatis.
            </p>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-foreground mb-1">Nama Bank</label>
                <input
                  type="text"
                  value={bankName}
                  onChange={(e) => setBankName(e.target.value)}
                  placeholder="Contoh: BCA / Mandiri / BRI"
                  className="w-full text-xs px-3 py-2 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
                />
              </div>

              <div>
                <label className="block text-xs font-medium text-foreground mb-1">Nomor Rekening</label>
                <input
                  type="text"
                  value={bankAccount}
                  onChange={(e) => setBankAccount(e.target.value)}
                  placeholder="Contoh: 1234567890"
                  className="w-full text-xs font-mono px-3 py-2 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-medium text-foreground mb-1">Nama Pemilik Rekening</label>
              <input
                type="text"
                value={holderName}
                onChange={(e) => setHolderName(e.target.value)}
                placeholder="Contoh: PT Toko Berkah Jaya / Budi Santoso"
                className="w-full text-xs px-3 py-2 rounded-md border border-border bg-background text-foreground outline-none focus:ring-1 focus:ring-primary/30 focus:border-primary/50"
              />
            </div>
          </div>

          {/* ADR-008 Assurance Callout */}
          <div className="p-3 bg-blue-50/60 border border-blue-100 rounded-lg text-blue-900 text-xs flex gap-2.5">
            <Info className="h-4 w-4 text-blue-600 shrink-0 mt-0.5" />
            <div className="leading-relaxed text-[11px]">
              <strong>Jaminan Keamanan Dana:</strong> Tayooli menerapkan model <em>Bring-Your-Own Settlement</em>. Kami tidak memotong, menyimpan, atau mengelola dana Anda. Seluruh pembayaran mengalir langsung dari pelanggan ke rekening Midtrans Anda.
            </div>
          </div>

          {/* Feedback states */}
          {saveSuccess && (
            <div className="p-3 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-lg text-xs flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-emerald-600" />
              <span>Pengaturan payment gateway berhasil disimpan!</span>
            </div>
          )}

          {saveConfig.isError && (
            <div className="p-3 bg-red-50 border border-red-200 text-red-800 rounded-lg text-xs flex items-center gap-2">
              <AlertCircle className="h-4 w-4 text-red-600" />
              <span>Gagal menyimpan pengaturan: {(saveConfig.error as Error)?.message || 'Terjadi kesalahan sistem.'}</span>
            </div>
          )}

          {/* Submit button */}
          <button
            type="submit"
            disabled={saveConfig.isPending}
            className="flex items-center justify-center gap-2 px-4 py-2 text-xs font-semibold bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors disabled:opacity-50"
          >
            {saveConfig.isPending ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin" /> Menyimpan...
              </>
            ) : (
              'Simpan Pengaturan Gateway'
            )}
          </button>
        </form>
      )}
    </div>
  )
}
