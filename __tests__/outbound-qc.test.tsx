import React from "react"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { OutboundQCSubView } from "@/components/wms/OutboundQCSubView"
import { api, DeliveryOrder } from "@/lib/api"

// Mock hooks
const mockUser = {
  id: "user-qc-1",
  email: "qc@tayooli.com",
  role: "warehouse",
  tenantId: "t-1",
  full_name: "Budi QC",
}

jest.mock("@/hooks/useAuth", () => ({
  useAuth: () => ({
    user: mockUser,
    isAuthenticated: true,
    isLoading: false,
  }),
}))

jest.mock("@/hooks/useWMS", () => ({
  useWarehouses: () => ({
    data: [
      { id: "wh-1", code: "WH-MAIN", name: "Gudang Utama", is_active: true },
    ],
    isLoading: false,
  }),
}))

const mockMutatePassQC = jest.fn()
const mockMutateRejectQC = jest.fn()
const mockRefetchHistory = jest.fn()

jest.mock("@/hooks/useWMSManifests", () => ({
  usePassOutboundQC: () => ({
    mutateAsync: mockMutatePassQC,
    isPending: false,
  }),
  useRejectOutboundQC: () => ({
    mutateAsync: mockMutateRejectQC,
    isPending: false,
  }),
  useOutboundQCInspections: () => ({
    data: [
      {
        id: "insp-1",
        tenant_id: "t-1",
        delivery_order_id: "do-1",
        do_number: "DO/20261101/0001",
        warehouse_id: "wh-1",
        inspector_id: "user-qc-1",
        inspector_name: "Budi QC",
        status: "QC_PASSED",
        scanned_awb: "AWB-SPX-12345",
        awb_matched: true,
        package_qty_matched: true,
        created_at: "2026-11-01T10:00:00Z",
      },
    ],
    isLoading: false,
    refetch: mockRefetchHistory,
  }),
}))

const mockDOs: DeliveryOrder[] = [
  {
    id: "do-1",
    tenant_id: "t-1",
    warehouse_id: "wh-1",
    do_number: "DO/20261101/0001",
    status: "PACKED",
    customer_name: "Toko Berkah",
    expedition_name: "Shopee Xpress",
    tracking_number: "AWB-SPX-12345",
    packed_by: "user-packer-2", // Different user -> Segregation OK
    packed_by_name: "Joko Packer",
    package_weight_kg: 2.5,
    package_length_cm: 30,
    package_width_cm: 20,
    package_height_cm: 15,
    packaging_type: "KARTON",
    sales_order_id: null,
    created_at: "2026-11-01T08:00:00Z",
    updated_at: "2026-11-01T08:30:00Z",
  },
  {
    id: "do-2",
    tenant_id: "t-1",
    warehouse_id: "wh-1",
    do_number: "DO/20261101/0002",
    status: "PACKED",
    customer_name: "CV Mandiri",
    expedition_name: "JNE",
    tracking_number: "JNE-99999",
    packed_by: "user-qc-1", // Same user as mockUser -> Segregation Violation!
    packed_by_name: "Budi QC",
    package_weight_kg: 5.0,
    packaging_type: "KARTON",
    sales_order_id: null,
    created_at: "2026-11-01T08:15:00Z",
    updated_at: "2026-11-01T08:45:00Z",
  },
]

describe("OutboundQCSubView Component", () => {
  beforeEach(() => {
    jest.clearAllMocks()
    jest.spyOn(api.wms.deliveryOrders, "list").mockResolvedValue({
      data: mockDOs,
    } as any)
    jest.spyOn(api.wms.deliveryOrders, "get").mockResolvedValue({
      delivery_order: mockDOs[0],
      items: [
        {
          id: "item-1",
          tenant_id: "t-1",
          delivery_order_id: "do-1",
          product_id: "prod-1",
          product_name: "Minyak Goreng 1L",
          product_sku: "MYK-01",
          quantity: 2,
          packed_qty: 2,
          location_id: "loc-1",
          batch_number: "BATCH-2026-01",
          created_at: "2026-11-01T08:00:00Z",
        },
      ],
    } as any)
  })

  it("renders order queue and selects an order", async () => {
    render(<OutboundQCSubView warehouseId="wh-1" />)

    await waitFor(() => {
      expect(screen.getByText("DO/20261101/0001")).toBeInTheDocument()
      expect(screen.getByText("DO/20261101/0002")).toBeInTheDocument()
    })

    // Click DO-1 to select
    fireEvent.click(screen.getByText("DO/20261101/0001"))

    await waitFor(() => {
      expect(screen.getByText("Minyak Goreng 1L")).toBeInTheDocument()
      expect(screen.getByText("MYK-01")).toBeInTheDocument()
      expect(screen.getAllByText("Joko Packer").length).toBeGreaterThan(0)
    })
  })

  it("verifies AWB scan matching and mismatch", async () => {
    render(<OutboundQCSubView warehouseId="wh-1" />)

    await waitFor(() => {
      expect(screen.getByText("DO/20261101/0001")).toBeInTheDocument()
    })
    fireEvent.click(screen.getByText("DO/20261101/0001"))

    await waitFor(() => {
      expect(screen.getByPlaceholderText(/Arahkan scanner ke barcode resi/i)).toBeInTheDocument()
    })

    const awbInput = screen.getByPlaceholderText(/Arahkan scanner ke barcode resi/i)

    // 1. Scan mismatch barcode
    fireEvent.change(awbInput, { target: { value: "WRONG-AWB-CODE" } })
    fireEvent.click(screen.getByRole("button", { name: "Verifikasi" }))

    await waitFor(() => {
      expect(screen.getByText(/Barcode AWB tidak cocok!/i)).toBeInTheDocument()
      expect(screen.getByText("Tidak Cocok")).toBeInTheDocument()
    })

    // 2. Scan correct AWB barcode
    fireEvent.change(awbInput, { target: { value: "AWB-SPX-12345" } })
    fireEvent.click(screen.getByRole("button", { name: "Verifikasi" }))

    await waitFor(() => {
      expect(screen.getByText(/AWB \/ No. Surat Jalan terverifikasi cocok/i)).toBeInTheDocument()
      expect(screen.getByText("Resi Cocok")).toBeInTheDocument()
    })
  })

  it("enforces Segregation of Duties: warns and disables approval if packed by same user", async () => {
    render(<OutboundQCSubView warehouseId="wh-1" />)

    await waitFor(() => {
      expect(screen.getByText("DO/20261101/0002")).toBeInTheDocument()
    })

    // Select DO-2 (which was packed by user-qc-1 = mockUser)
    fireEvent.click(screen.getByText("DO/20261101/0002"))

    await waitFor(() => {
      expect(
        screen.getByText(/Peringatan Segregation of Duties/i)
      ).toBeInTheDocument()
    })

    const passBtn = screen.getByRole("button", { name: /Loloskan QC/i })
    expect(passBtn).toBeDisabled()

    const rejectBtn = screen.getByRole("button", { name: /Tolak QC/i })
    expect(rejectBtn).toBeDisabled()
  })

  it("submits QC Pass for valid order", async () => {
    mockMutatePassQC.mockResolvedValueOnce({
      data: { ...mockDOs[0], status: "QC_PASSED" },
    })

    render(<OutboundQCSubView warehouseId="wh-1" />)

    await waitFor(() => {
      expect(screen.getByText("DO/20261101/0001")).toBeInTheDocument()
    })
    fireEvent.click(screen.getByText("DO/20261101/0001"))

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Loloskan QC/i })).toBeEnabled()
    })

    const passBtn = screen.getByRole("button", { name: /Loloskan QC/i })
    fireEvent.click(passBtn)

    await waitFor(() => {
      expect(mockMutatePassQC).toHaveBeenCalledTimes(1)
      expect(mockMutatePassQC).toHaveBeenCalledWith({
        id: "do-1",
        payload: expect.any(Object),
      })
    })
  })

  it("submits QC Reject with reason", async () => {
    mockMutateRejectQC.mockResolvedValueOnce({
      data: { ...mockDOs[0], status: "QC_FAILED" },
    })

    render(<OutboundQCSubView warehouseId="wh-1" />)

    await waitFor(() => {
      expect(screen.getByText("DO/20261101/0001")).toBeInTheDocument()
    })
    fireEvent.click(screen.getByText("DO/20261101/0001"))

    await waitFor(() => {
      expect(screen.getByRole("button", { name: /Tolak QC/i })).toBeEnabled()
    })

    // Click Tolak QC to open modal
    fireEvent.click(screen.getByRole("button", { name: /Tolak QC/i }))

    expect(screen.getByText(/Tolak QC & Kemas Ulang/i)).toBeInTheDocument()

    // Confirm rejection
    const confirmRejectBtn = screen.getByRole("button", { name: /Konfirmasi Tolak QC/i })
    fireEvent.click(confirmRejectBtn)

    await waitFor(() => {
      expect(mockMutateRejectQC).toHaveBeenCalledTimes(1)
      expect(mockMutateRejectQC).toHaveBeenCalledWith({
        id: "do-1",
        payload: expect.objectContaining({
          reason: expect.any(String),
        }),
      })
    })
  })

  it("opens audit history modal", async () => {
    render(<OutboundQCSubView warehouseId="wh-1" />)

    const historyBtn = screen.getByRole("button", { name: /Riwayat Audit QC/i })
    fireEvent.click(historyBtn)

    expect(
      screen.getByText("Riwayat Audit Inspeksi QC Outbound")
    ).toBeInTheDocument()
    expect(screen.getByText("AWB-SPX-12345")).toBeInTheDocument()
  })
})
