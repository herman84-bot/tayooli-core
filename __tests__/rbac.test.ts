import { canAccessRoute, homeRouteFor, requiresWarehouse, ASSIGNABLE_ROLES } from '@/lib/rbac'

const MENU = [
  '/dashboard', '/products', '/wms/arus-barang', '/wms', '/wms/marketplace', '/wms/transfers',
  '/wms/opname', '/wms/scrap', '/wms/scanner', '/pos', '/settings', '/help',
]

const visible = (role: string) => MENU.filter((m) => canAccessRoute(role, m))

describe('RBAC route matrix', () => {
  it('owner & admin see all 12 menus', () => {
    expect(visible('owner')).toEqual(MENU)
    expect(visible('admin')).toEqual(MENU)
  })

  it('cashier sees only POS and Help', () => {
    expect(visible('cashier')).toEqual(['/pos', '/help'])
  })

  it('cashier is blocked from deep and unknown routes', () => {
    expect(canAccessRoute('cashier', '/wms/transfers/123')).toBe(false)
    expect(canAccessRoute('cashier', '/products/abc')).toBe(false)
    expect(canAccessRoute('cashier', '/some-unlisted-page')).toBe(false)
    expect(canAccessRoute('cashier', '/pos/history')).toBe(true)
  })

  it('warehouse staff: no dashboard, marketplace, POS, settings', () => {
    const v = visible('warehouse')
    for (const m of ['/dashboard', '/wms/marketplace', '/pos', '/settings']) expect(v).not.toContain(m)
    for (const m of ['/wms/scanner', '/wms/arus-barang', '/wms/opname', '/products']) expect(v).toContain(m)
  })

  it('warehouse_manager: dashboard yes, POS/settings/marketplace no', () => {
    const v = visible('warehouse_manager')
    expect(v).toContain('/dashboard')
    for (const m of ['/pos', '/settings', '/wms/marketplace']) expect(v).not.toContain(m)
  })

  it('auditor: no scanner, POS, settings', () => {
    const v = visible('auditor')
    for (const m of ['/wms/scanner', '/pos', '/settings']) expect(v).not.toContain(m)
  })

  it('prefix match does not leak (/wmsx is not /wms)', () => {
    expect(canAccessRoute('cashier', '/posx')).toBe(false)
  })

  it('home routes', () => {
    expect(homeRouteFor('cashier')).toBe('/pos')
    expect(homeRouteFor('warehouse')).toBe('/wms/scanner')
    expect(homeRouteFor('admin')).toBe('/dashboard')
  })

  it('home route is always accessible for its role', () => {
    for (const r of [...ASSIGNABLE_ROLES, 'owner']) expect(canAccessRoute(r, homeRouteFor(r))).toBe(true)
  })

  it('owner is not assignable; warehouse roles require warehouse', () => {
    expect(ASSIGNABLE_ROLES).not.toContain('owner')
    expect(requiresWarehouse('warehouse')).toBe(true)
    expect(requiresWarehouse('warehouse_manager')).toBe(true)
    expect(requiresWarehouse('cashier')).toBe(false)
  })
})
