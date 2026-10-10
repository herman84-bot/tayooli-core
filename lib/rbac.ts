/**
 * Enterprise RBAC — sumber tunggal aturan role di frontend.
 * Spesifikasi: docs/superpowers/specs/2026-10-10-enterprise-rbac-hierarchy-design.md
 *
 * Catatan: ini hanya lapisan UX (menyembunyikan menu & mengalihkan rute).
 * Penegakan keamanan sebenarnya ada di backend Go (RequireRole/DenyRole +
 * ValidateWarehouseWriteAccess).
 */

export type AppRole =
  | 'owner'
  | 'admin'
  | 'warehouse_manager'
  | 'warehouse'
  | 'cashier'
  | 'auditor'
  | 'member'

/** Role yang bisa dipilih saat undang / ubah role (owner tidak bisa diberikan). */
export const ASSIGNABLE_ROLES: AppRole[] = [
  'admin',
  'warehouse_manager',
  'warehouse',
  'cashier',
  'auditor',
  'member',
]

export const ROLE_LABELS: Record<string, string> = {
  owner: 'Owner',
  admin: 'Admin Operasional',
  warehouse_manager: 'Kepala Gudang',
  regional_manager: 'Kepala Gudang (Regional)',
  warehouse: 'Staf Gudang',
  cashier: 'Kasir',
  auditor: 'Auditor',
  member: 'Member',
  accountant: 'Accountant',
  approver: 'Approver',
}

export const ROLE_DESCRIPTIONS: Record<string, string> = {
  admin: 'Kelola produk, harga, persetujuan transfer & scrap, marketplace.',
  warehouse_manager: 'Setujui transfer & validasi opname di gudang yang ditugaskan.',
  warehouse: 'Scan, terima, putaway, picking, packing di gudang yang ditugaskan.',
  cashier: 'Hanya layar kasir (POS).',
  auditor: 'Hanya lihat laporan & mutasi, tidak bisa mengubah data.',
  member: 'Akses dasar tanpa kelola tim.',
}

export const ROLE_COLORS: Record<string, string> = {
  owner: 'bg-primary/10 text-primary',
  admin: 'bg-blue-50 text-blue-600',
  warehouse_manager: 'bg-indigo-50 text-indigo-600',
  regional_manager: 'bg-indigo-50 text-indigo-600',
  warehouse: 'bg-orange-50 text-orange-600',
  cashier: 'bg-emerald-50 text-emerald-600',
  auditor: 'bg-zinc-100 text-zinc-600',
  member: 'bg-zinc-100 text-zinc-600',
  accountant: 'bg-purple-50 text-purple-600',
  approver: 'bg-amber-50 text-amber-600',
}

/** Role yang wajib memiliki minimal satu gudang yang ditugaskan. */
export function requiresWarehouse(role: string): boolean {
  return role === 'warehouse' || role === 'warehouse_manager'
}

const ALL = '*'

/**
 * Peta prefix rute → role yang boleh melihat menu & membuka halaman.
 * Urutan penting: prefix paling spesifik di atas.
 */
const ROUTE_RULES: { prefix: string; roles: string[] | typeof ALL }[] = [
  { prefix: '/help', roles: ALL },
  { prefix: '/pos', roles: ['owner', 'admin', 'cashier', 'member'] },
  { prefix: '/settings', roles: ['owner', 'admin'] },
  { prefix: '/wms/marketplace', roles: ['owner', 'admin', 'auditor'] },
  { prefix: '/wms/scanner', roles: ['owner', 'admin', 'warehouse_manager', 'regional_manager', 'warehouse', 'member'] },
  { prefix: '/wms', roles: ['owner', 'admin', 'warehouse_manager', 'regional_manager', 'warehouse', 'auditor', 'member'] },
  { prefix: '/products', roles: ['owner', 'admin', 'warehouse_manager', 'regional_manager', 'warehouse', 'auditor', 'member'] },
  { prefix: '/dashboard', roles: ['owner', 'admin', 'warehouse_manager', 'regional_manager', 'auditor', 'member'] },
]

/**
 * Apakah role boleh membuka path tertentu.
 * Role tidak dikenal / rute tidak terdaftar: diizinkan (backend tetap menjaga),
 * kecuali kasir yang dikunci ketat ke rute yang terdaftar untuknya.
 */
export function canAccessRoute(role: string | undefined | null, pathname: string): boolean {
  if (!role) return true
  const rule = ROUTE_RULES.find(
    (r) => pathname === r.prefix || pathname.startsWith(r.prefix + '/'),
  )
  if (!rule) return role !== 'cashier'
  if (rule.roles === ALL) return true
  // Role warisan (accountant/approver dsb.) tetap diperlakukan seperti member.
  const effective = rule.roles.includes(role) ? role : isKnownRole(role) ? role : 'member'
  return rule.roles.includes(effective)
}

function isKnownRole(role: string): boolean {
  return ['owner', 'admin', 'warehouse_manager', 'regional_manager', 'warehouse', 'cashier', 'auditor', 'member'].includes(role)
}

/** Halaman awal setelah login / saat akses ditolak. */
export function homeRouteFor(role: string | undefined | null): string {
  if (role === 'cashier') return '/pos'
  if (role === 'warehouse') return '/wms/scanner'
  return '/dashboard'
}
