import React from 'react'
import { render, screen } from '@testing-library/react'
import '@testing-library/jest-dom'
import DashboardPage from '@/app/(app)/dashboard/page'
import { EMPTY_DASHBOARD_SUMMARY } from '@/lib/schemas/dashboard'
import type { WMSOutboundKPISummary } from '@/lib/api'

// Mocks
jest.mock('@/components/tutorial/TooltipWalkthrough', () => ({
  TooltipWalkthrough: () => null,
}))

const mockKPIs: WMSOutboundKPISummary = {
  dock_to_stock_avg_minutes: 38.5,
  receiving_accuracy_pct: 99.9,
  po_compliance_pct: 98.2,
  backlog_inbound_count: 0,
  order_to_dispatch_avg_hours: 1.8,
  picking_accuracy_pct: 100,
  on_time_shipment_pct: 99.2,
  backlog_outbound_count: 0,
}

let mockKPIResult: { data: WMSOutboundKPISummary | null | undefined; isLoading?: boolean; refetch?: () => void } = {
  data: mockKPIs,
}

jest.mock('@/hooks/useWMSManifests', () => ({
  useWMSOutboundKPI: () => mockKPIResult,
}))

jest.mock('@/lib/queries/dashboard', () => ({
  useDashboardSummary: () => ({
    data: EMPTY_DASHBOARD_SUMMARY,
    isLoading: false,
    error: null,
    refetch: jest.fn(),
  }),
}))

describe('DashboardPage 8 Enterprise WMS SOP KPIs', () => {
  it('renders all 8 SOP KPIs with provided data', () => {
    mockKPIResult = { data: mockKPIs }
    render(<DashboardPage />)

    expect(screen.getByText('8 Enterprise WMS SOP KPIs')).toBeInTheDocument()
    expect(screen.getByText('Dock-to-Stock Time')).toBeInTheDocument()
    expect(screen.getByText('38.5')).toBeInTheDocument()
    expect(screen.getByText('Receiving Accuracy')).toBeInTheDocument()
    expect(screen.getByText('99.9')).toBeInTheDocument()
    expect(screen.getByText('PO Compliance')).toBeInTheDocument()
    expect(screen.getByText('98.2')).toBeInTheDocument()
    expect(screen.getByText('Inbound Backlog')).toBeInTheDocument()
    expect(screen.getByText('Order-to-Dispatch Time')).toBeInTheDocument()
    expect(screen.getByText('1.8')).toBeInTheDocument()
    expect(screen.getByText('Picking Accuracy')).toBeInTheDocument()
    expect(screen.getByText('100')).toBeInTheDocument()
    expect(screen.getByText('On-Time Shipment')).toBeInTheDocument()
    expect(screen.getByText('99.2')).toBeInTheDocument()
    expect(screen.getByText('Outbound Backlog')).toBeInTheDocument()
  })

  it('renders fallback defaults when kpi data is undefined', () => {
    mockKPIResult = { data: undefined }
    render(<DashboardPage />)

    expect(screen.getByText('45')).toBeInTheDocument()
    expect(screen.getByText('99.8')).toBeInTheDocument()
    expect(screen.getByText('97.5')).toBeInTheDocument()
    expect(screen.getByText('2.4')).toBeInTheDocument()
    expect(screen.getByText('99.9')).toBeInTheDocument()
    expect(screen.getByText('98.6')).toBeInTheDocument()
  })
})
