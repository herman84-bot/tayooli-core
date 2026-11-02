/**
 * @jest-environment jsdom
 */
import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import SettingsPage from '@/app/(app)/settings/page'

jest.mock('@/hooks/useAuth', () => ({
  useAuth: () => ({
    user: { role: 'admin', email: 'admin@tayooli.com' },
  }),
}))

jest.mock('@/hooks/useTeamMembers', () => ({
  useTeamMembers: () => ({
    members: [
      { id: 'm-1', email: 'staff@tayooli.com', full_name: 'Staff Logistik', role: 'warehouse_staff' },
    ],
    isLoading: false,
    inviteMember: jest.fn(),
    updateRole: jest.fn(),
    removeMember: jest.fn(),
    isInviting: false,
    resendInvite: { mutate: jest.fn(), isPending: false, error: null },
  }),
}))

jest.mock('@tanstack/react-query', () => ({
  useQuery: () => ({ data: [], isLoading: false }),
}))

jest.mock('@/hooks/usePaymentConfig', () => ({
  usePaymentConfig: () => ({
    config: null,
    isLoading: false,
    isSaving: false,
    save: jest.fn(),
  }),
}))

jest.mock('@/hooks/useCompanyProfile', () => ({
  useCompanyProfile: () => ({
    profile: {
      company_name: 'PT Tayooli Logistik',
      division: 'Divisi Logistik',
      address: 'Jl. Sudirman No. 1',
      phone: '08123456789',
    },
    isLoading: false,
    isSaving: false,
    save: jest.fn(),
  }),
}))

describe('SettingsPage UI and Typography Quality', () => {
  it('renders team table headers in sentence case without uppercase tracking-wider', () => {
    render(<SettingsPage />)

    const nameHeader = screen.getByRole('columnheader', { name: /Nama/i })
    expect(nameHeader.className).not.toMatch(/uppercase/)
    expect(nameHeader.className).not.toMatch(/tracking-wider/)
  })
})
