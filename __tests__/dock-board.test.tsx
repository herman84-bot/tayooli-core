import React from "react"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import "@testing-library/jest-dom"
import { DockBoardSubView } from "@/components/wms/DockBoardSubView"
import type { InboundDock, DockAppointment } from "@/lib/api"

const mockWarehouseId = "wh-cibitung-1"

const mockDocks: InboundDock[] = [
  {
    id: "dock-1",
    warehouse_id: mockWarehouseId,
    dock_code: "DCK-01",
    dock_name: "Dermaga Inbound 1",
    dock_type: "INBOUND",
    max_tonnage: 20,
    status: "AVAILABLE",
    notes: "Pintu dermaga sisi barat",
    created_at: "2026-10-08T00:00:00Z",
    updated_at: "2026-10-08T00:00:00Z",
  },
  {
    id: "dock-2",
    warehouse_id: mockWarehouseId,
    dock_code: "DCK-02",
    dock_name: "Dermaga Inbound 2",
    dock_type: "INBOUND",
    max_tonnage: 25,
    status: "OCCUPIED",
    notes: "Sedang proses bongkar kontainer",
    created_at: "2026-10-08T00:00:00Z",
    updated_at: "2026-10-08T00:00:00Z",
  },
  {
    id: "dock-3",
    warehouse_id: mockWarehouseId,
    dock_code: "DCK-03",
    dock_name: "Dermaga Perbaikan 3",
    dock_type: "CROSS_DOCK",
    max_tonnage: 15,
    status: "MAINTENANCE",
    notes: "Hydraulic leveler dalam servis",
    created_at: "2026-10-08T00:00:00Z",
    updated_at: "2026-10-08T00:00:00Z",
  },
]

const mockAppointments: DockAppointment[] = [
  {
    id: "appt-1",
    warehouse_id: mockWarehouseId,
    dock_id: "dock-2",
    dock_code: "DCK-02",
    dock_name: "Dermaga Inbound 2",
    appointment_number: "APT-20261008-001",
    vendor_name: "PT Sumber Makmur",
    vehicle_plate: "B 9182 KDA",
    driver_name: "Budi Santoso",
    driver_phone: "081234567890",
    po_reference: "PO-INB-9901",
    estimated_arrival: "2026-10-08T08:00:00Z",
    actual_arrival: "2026-10-08T08:15:00Z",
    start_unloading_at: "2026-10-08T08:30:00Z",
    status: "UNLOADING",
    created_at: "2026-10-08T07:00:00Z",
    updated_at: "2026-10-08T08:30:00Z",
  },
  {
    id: "appt-2",
    warehouse_id: mockWarehouseId,
    dock_id: undefined,
    appointment_number: "APT-20261008-002",
    vendor_name: "CV Berkah Abadi",
    vehicle_plate: "D 8821 ZZ",
    driver_name: "Asep Sunandar",
    driver_phone: "085678901234",
    po_reference: "PO-INB-9902",
    estimated_arrival: "2026-10-08T10:00:00Z",
    status: "SCHEDULED",
    created_at: "2026-10-08T07:15:00Z",
    updated_at: "2026-10-08T07:15:00Z",
  },
  {
    id: "appt-3",
    warehouse_id: mockWarehouseId,
    dock_id: "dock-1",
    dock_code: "DCK-01",
    dock_name: "Dermaga Inbound 1",
    appointment_number: "APT-20261008-003",
    vendor_name: "PT Logistik Sentral",
    vehicle_plate: "B 1234 ABC",
    driver_name: "Iwan Fals",
    driver_phone: "087788990011",
    po_reference: "PO-INB-9903",
    estimated_arrival: "2026-10-08T09:00:00Z",
    actual_arrival: "2026-10-08T08:50:00Z",
    status: "ARRIVED",
    created_at: "2026-10-08T07:30:00Z",
    updated_at: "2026-10-08T08:50:00Z",
  },
  {
    id: "appt-4",
    warehouse_id: mockWarehouseId,
    appointment_number: "APT-20261008-004",
    vendor_name: "PT Indah Sejahtera",
    vehicle_plate: "F 4321 XY",
    driver_name: "Doni Tata",
    po_reference: "PO-INB-9904",
    estimated_arrival: "2026-10-08T06:00:00Z",
    completed_at: "2026-10-08T07:30:00Z",
    status: "COMPLETED",
    created_at: "2026-10-08T05:00:00Z",
    updated_at: "2026-10-08T07:30:00Z",
  },
]

const mockCreateDockMutate = jest.fn().mockResolvedValue({})
const mockUpdateDockStatusMutate = jest.fn().mockResolvedValue({})
const mockCreateAppointmentMutate = jest.fn().mockResolvedValue({})
const mockAssignDockMutate = jest.fn().mockResolvedValue({})
const mockUpdateAppointmentStatusMutate = jest.fn().mockResolvedValue({})

jest.mock("@/hooks/useWMSDocksAndLPNs", () => ({
  useInboundDocks: () => ({
    data: mockDocks,
    isLoading: false,
    refetch: jest.fn(),
  }),
  useCreateDock: () => ({
    mutateAsync: mockCreateDockMutate,
    isPending: false,
  }),
  useUpdateDockStatus: () => ({
    mutateAsync: mockUpdateDockStatusMutate,
    isPending: false,
  }),
  useDockAppointments: () => ({
    data: mockAppointments,
    isLoading: false,
    refetch: jest.fn(),
  }),
  useCreateAppointment: () => ({
    mutateAsync: mockCreateAppointmentMutate,
    isPending: false,
  }),
  useAssignDock: () => ({
    mutateAsync: mockAssignDockMutate,
    isPending: false,
  }),
  useUpdateAppointmentStatus: () => ({
    mutateAsync: mockUpdateAppointmentStatusMutate,
    isPending: false,
  }),
}))

describe("DockBoardSubView Component", () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockAssignDockMutate.mockResolvedValue({})
  })

  it("renders dock metrics correctly", () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    // Total Dermaga = 3
    expect(screen.getByText("Total Dermaga")).toBeInTheDocument()
    expect(screen.getByText("3")).toBeInTheDocument()

    // Dermaga Tersedia = 1 (dock-1)
    expect(screen.getByText("Dermaga Tersedia")).toBeInTheDocument()
    // Sedang Bongkar = 1 (dock-2)
    expect(screen.getByText("Sedang Bongkar")).toBeInTheDocument()
    expect(screen.getAllByText("1").length).toBeGreaterThanOrEqual(2)

    // Antrean Menunggu = 2 (appt-2 SCHEDULED + appt-3 ARRIVED)
    expect(screen.getByText("Antrean Menunggu")).toBeInTheDocument()
    expect(screen.getByText("2")).toBeInTheDocument()
  })

  it("renders dock cards with status badges and details", () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    // Dock bays
    expect(screen.getByText("Dermaga Inbound 1")).toBeInTheDocument()
    expect(screen.getByText("DCK-01")).toBeInTheDocument()
    expect(screen.getByText("AVAILABLE")).toBeInTheDocument()

    expect(screen.getByText("Dermaga Inbound 2")).toBeInTheDocument()
    expect(screen.getByText("DCK-02")).toBeInTheDocument()
    expect(screen.getByText("OCCUPIED")).toBeInTheDocument()

    expect(screen.getByText("Dermaga Perbaikan 3")).toBeInTheDocument()
    expect(screen.getByText("DCK-03")).toBeInTheDocument()
    expect(screen.getByText("MAINTENANCE")).toBeInTheDocument()

    // Active appointment in occupied dock 2 (appears in dock card and table)
    expect(screen.getAllByText("B 9182 KDA").length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText(/PT Sumber Makmur/i).length).toBeGreaterThanOrEqual(1)
  })

  it("supports toggle maintenance on dock card", async () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    // dock-3 is in MAINTENANCE, button label is "Selesai Maint."
    const finishMaintBtn = screen.getByText("Selesai Maint.")
    fireEvent.click(finishMaintBtn)

    await waitFor(() => {
      expect(mockUpdateDockStatusMutate).toHaveBeenCalledWith({
        id: "dock-3",
        status: "AVAILABLE",
      })
    })
  })

  it("supports release dock action when occupied", async () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    const releaseBtn = screen.getByText("Lepas Dermaga")
    fireEvent.click(releaseBtn)

    await waitFor(() => {
      expect(mockUpdateAppointmentStatusMutate).toHaveBeenCalledWith({
        appointmentId: "appt-1",
        status: "COMPLETED",
      })
      expect(mockUpdateDockStatusMutate).toHaveBeenCalledWith({
        id: "dock-2",
        status: "AVAILABLE",
      })
    })
  })

  it("renders inbound appointments table with filter tabs", () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    expect(screen.getByRole("button", { name: /^Semua$/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Terjadwal \(SCHEDULED\)/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Tiba \(ARRIVED\)/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Bongkar \(UNLOADING\)/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /Selesai \(COMPLETED\)/i })).toBeInTheDocument()

    // Table rows
    expect(screen.getByText("APT-20261008-001")).toBeInTheDocument()
    expect(screen.getByText("APT-20261008-002")).toBeInTheDocument()
    expect(screen.getByText("APT-20261008-003")).toBeInTheDocument()
    expect(screen.getByText("APT-20261008-004")).toBeInTheDocument()
  })

  it("filters appointments when clicking status tab", () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    const scheduledTab = screen.getByRole("button", { name: /Terjadwal \(SCHEDULED\)/i })
    fireEvent.click(scheduledTab)

    expect(screen.getByText("APT-20261008-002")).toBeInTheDocument()
    expect(screen.queryByText("APT-20261008-001")).not.toBeInTheDocument()
    expect(screen.queryByText("APT-20261008-003")).not.toBeInTheDocument()
  })

  it("handles appointment lifecycle actions: Tiba, Mulai Bongkar, Selesai", async () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    // 1. Tiba for SCHEDULED (appt-2)
    const tibaBtn = screen.getByRole("button", { name: /^Tiba$/i })
    fireEvent.click(tibaBtn)
    await waitFor(() => {
      expect(mockUpdateAppointmentStatusMutate).toHaveBeenCalledWith({
        appointmentId: "appt-2",
        status: "ARRIVED",
      })
    })

    // 2. Mulai Bongkar for ARRIVED with dock assigned (appt-3)
    const mulaiBongkarBtn = screen.getByRole("button", { name: /^Mulai Bongkar$/i })
    fireEvent.click(mulaiBongkarBtn)
    await waitFor(() => {
      expect(mockUpdateAppointmentStatusMutate).toHaveBeenCalledWith({
        appointmentId: "appt-3",
        status: "UNLOADING",
      })
    })

    // 3. Selesai for UNLOADING (appt-1)
    const selesaiBtn = screen.getByRole("button", { name: /^Selesai$/i })
    fireEvent.click(selesaiBtn)
    await waitFor(() => {
      expect(mockUpdateAppointmentStatusMutate).toHaveBeenCalledWith({
        appointmentId: "appt-1",
        status: "COMPLETED",
      })
    })
  })

  it("opens modal and creates new dock bay", async () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    const tambahDermagaBtn = screen.getByRole("button", { name: /Tambah Dermaga/i })
    fireEvent.click(tambahDermagaBtn)

    expect(screen.getByText("Tambah Dermaga Baru")).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText(/Mis: DOCK-01/i), {
      target: { value: "DCK-04" },
    })
    fireEvent.change(screen.getByPlaceholderText(/Mis: Dermaga Inbound Barat 1/i), {
      target: { value: "Dermaga Inbound 4" },
    })

    const submitBtn = screen.getByRole("button", { name: /Simpan Dermaga/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockCreateDockMutate).toHaveBeenCalledWith(
        expect.objectContaining({
          warehouse_id: mockWarehouseId,
          dock_code: "DCK-04",
          dock_name: "Dermaga Inbound 4",
          dock_type: "INBOUND",
        })
      )
    })
  })

  it("opens modal and creates new appointment", async () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    const jadwalkanBtn = screen.getByRole("button", { name: /Jadwalkan Armada Baru/i })
    fireEvent.click(jadwalkanBtn)

    expect(screen.getByText("Jadwalkan Armada Pemasok Baru")).toBeInTheDocument()

    fireEvent.change(screen.getByPlaceholderText(/Mis: PT Pangan Sentosa/i), {
      target: { value: "PT Sinar Pagi" },
    })
    fireEvent.change(screen.getByPlaceholderText(/Mis: B 9123 UXZ/i), {
      target: { value: "B 5555 XYZ" },
    })
    fireEvent.change(screen.getByPlaceholderText(/Nama sopir/i), {
      target: { value: "Ahmad Dani" },
    })

    const submitBtn = screen.getByRole("button", { name: /Buat Jadwal Armada/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockCreateAppointmentMutate).toHaveBeenCalledWith(
        expect.objectContaining({
          warehouse_id: mockWarehouseId,
          vendor_name: "PT Sinar Pagi",
          vehicle_plate: "B 5555 XYZ",
          driver_name: "Ahmad Dani",
        })
      )
    })
  })

  it("assigns dock to appointment successfully", async () => {
    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    // appt-2 is SCHEDULED without dock -> click "Alokasikan Dock"
    const assignButtons = screen.getAllByRole("button", { name: /Alokasikan Dock/i })
    fireEvent.click(assignButtons[0])

    expect(screen.getByText("Alokasikan Dermaga Bongkar")).toBeInTheDocument()

    // Select available dock
    const select = screen.getByRole("combobox")
    fireEvent.change(select, { target: { value: "dock-1" } })

    const confirmBtn = screen.getByRole("button", { name: /Konfirmasi Alokasi/i })
    fireEvent.click(confirmBtn)

    await waitFor(() => {
      expect(mockAssignDockMutate).toHaveBeenCalledWith({
        appointmentId: "appt-2",
        dockId: "dock-1",
      })
    })
  })

  it("catches ErrDockOccupied (409) with friendly alert message", async () => {
    mockAssignDockMutate.mockRejectedValueOnce(
      new Error("conflict: dock is currently occupied or undergoing unloading")
    )

    render(<DockBoardSubView warehouseId={mockWarehouseId} />)

    const assignButtons = screen.getAllByRole("button", { name: /Alokasikan Dock/i })
    fireEvent.click(assignButtons[0])

    const select = screen.getByRole("combobox")
    fireEvent.change(select, { target: { value: "dock-1" } })

    const confirmBtn = screen.getByRole("button", { name: /Konfirmasi Alokasi/i })
    fireEvent.click(confirmBtn)

    await waitFor(() => {
      expect(
        screen.getByText("Dermaga ini sedang digunakan oleh armada lain! Silakan pilih dermaga lain.")
      ).toBeInTheDocument()
    })
  })
})
