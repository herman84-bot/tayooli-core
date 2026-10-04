import { create } from 'zustand'

interface AuthState {
  token: string | null
  tenantId: string | null
  setToken: (token: string) => void
  setTenantId: (tenantId: string) => void
  clearToken: () => void
}

export const useAuthStore = create<AuthState>()((set) => ({
  token: null,
  tenantId: null,
  setToken: (token) => set({ token }),
  setTenantId: (tenantId) => set({ tenantId }),
  clearToken: () => set({ token: null, tenantId: null }),
}))