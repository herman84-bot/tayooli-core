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

    expect(screen.getByText('Indikator Kinerja Pergudangan')).toBeInTheDocument()
    expect(screen.getByText('Waktu Bongkar ke Rak')).toBeInTheDocument()
    expect(screen.getByText('38.5')).toBeInTheDocument()
    expect(screen.getByText('Akurasi Penerimaan')).toBeInTheDocument()
    expect(screen.getByText('99.9')).toBeInTheDocument()
    expect(screen.getByText('Kepatuhan Pesanan (PO)')).toBeInTheDocument()
    expect(screen.getByText('98.2')).toBeInTheDocument()
    expect(screen.getByText('Antrean Barang Masuk')).toBeInTheDocument()
    expect(screen.getByText('Waktu Pesanan ke Kirim')).toBeInTheDocument()
    expect(screen.getByText('1.8')).toBeInTheDocument()
    expect(screen.getByText('100')).toBeInTheDocument()
    expect(screen.getByText('99.2')).toBeInTheDocument()
    expect(screen.getByText('Antrean Barang Keluar')).toBeInTheDocument()
  })

  it('new tenant (all rates null): shows "Belum ada data", never 100 or a pass badge', () => {
    mockKPIResult = {
      data: {
        window_days: 30,
        dock_to_stock_avg_minutes: null,
        receiving_accuracy_pct: null,
        po_compliance_pct: null,
        backlog_inbound_count: 0,
        order_to_dispatch_avg_hours: null,
        picking_accuracy_pct: null,
        on_time_shipment_pct: null,
        backlog_outbound_count: 0,
      },
    }
    render(<DashboardPage />)
    expect(screen.getAllByText('Belum ada data')).toHaveLength(6)
    expect(screen.queryByText('100')).not.toBeInTheDocument()
    // only the two backlog cards (real 0 counts) may carry a status badge
    expect(screen.getAllByText('Memenuhi')).toHaveLength(2)
    expect(screen.queryByText('Perhatian')).not.toBeInTheDocument()
    expect(screen.getByText(/30 hari terakhir/)).toBeInTheDocument()
  })

  it('labels the two proxy metrics by what they actually measure', () => {
    mockKPIResult = { data: mockKPIs }
    render(<DashboardPage />)
    expect(screen.getByText('Picking Tanpa Kerusakan')).toBeInTheDocument()
    expect(screen.getByText('Surat Jalan Terkirim')).toBeInTheDocument()
    expect(screen.queryByText('On-Time Shipment')).not.toBeInTheDocument()
    expect(screen.queryByText('Picking Accuracy')).not.toBeInTheDocument()
  })

  it('shows sample size so 100% of 1 record is distinguishable', () => {
    mockKPIResult = { data: { ...mockKPIs, picking_samples: 1 } }
    render(<DashboardPage />)
    expect(screen.getByText(/dari 1 data/)).toBeInTheDocument()
  })

  it('renders fallback zero defaults when kpi data is undefined', () => {
    mockKPIResult = { data: undefined }
    render(<DashboardPage />)

    expect(screen.getByText('Waktu Bongkar ke Rak')).toBeInTheDocument()
    expect(screen.getByText('Akurasi Penerimaan')).toBeInTheDocument()
    expect(screen.getByText('Kepatuhan Pesanan (PO)')).toBeInTheDocument()
    expect(screen.getByText('Waktu Pesanan ke Kirim')).toBeInTheDocument()
    // still loading/failed: no fabricated values, no pass/fail verdicts
    expect(screen.queryByText('Memenuhi')).not.toBeInTheDocument()
    expect(screen.queryByText('Perhatian')).not.toBeInTheDocument()
  })
})
