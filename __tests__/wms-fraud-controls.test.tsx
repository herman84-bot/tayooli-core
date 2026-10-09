import React from "react"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import "@testing-library/jest-dom"
import StockScrapPage from "@/app/(app)/wms/scrap/page"
import StockOpnamePage from "@/app/(app)/wms/opname/page"
import DeliveryOrdersPanel from "@/components/wms/DeliveryOrdersPanel"

// Mock hooks
const mockUseWarehouses = jest.fn()
const mockUseProducts = jest.fn()
const mockUseStockScraps = jest.fn()
const mockUseCreateStockScrap = jest.fn()
const mockUseStockOpnames = jest.fn()
const mockUseStockOpname = jest.fn()
const mockUseCompleteStockOpname = jest.fn()
const mockUseDeliveryOrders = jest.fn()
const mockUseDispatchDeliveryOrder = jest.fn()

let mockCurrentUser = {
  id: "user-warehouse-1",
  name: "Budi Warehouse",
  role: "warehouse",
}

jest.mock("@/hooks/useAuth", () => ({
  useAuthStore: (selector: any) => selector({ user: mockCurrentUser }),
}))

jest.mock("@/hooks/useWMS", () => ({
  useWarehouses: () => mockUseWarehouses(),
  useWarehouseLocations: () => ({ data: [{ id: "loc-1", name: "Rak A1", code: "RAK-A1", type: "INTERNAL" }] }),
  useStockScraps: () => mockUseStockScraps(),
  useCreateStockScrap: () => mockUseCreateStockScrap(),
  useStockOpnames: () => mockUseStockOpnames(),
  useStockOpname: () => mockUseStockOpname(),
  useCreateStockOpname: () => ({ mutateAsync: jest.fn() }),
  useAddOpnameItem: () => ({ mutateAsync: jest.fn() }),
  useCompleteStockOpname: () => mockUseCompleteStockOpname(),
  useCreateDeliveryOrder: () => ({
    mutateAsync: jest.fn().mockResolvedValue({}),
    isPending: false,
  }),
  useDeliveryOrders: () => mockUseDeliveryOrders(),
  useDispatchDeliveryOrder: () => mockUseDispatchDeliveryOrder(),
  useCustomers: () => ({ data: [] }),
}))

jest.mock("@/hooks/useWMSLedger", () => ({
  useWMSStock: () => ({
    data: [{ product_id: "p-1", location_id: "loc-1", quantity: 100, available_qty: 100 }],
    isLoading: false,
    isError: false,
  }),
  useWMSMovements: () => ({ data: [], isLoading: false }),
}))

jest.mock("@/hooks/useProducts", () => ({
  useProducts: () => mockUseProducts(),
}))

jest.mock("@/hooks/useBarcodeScanner", () => ({
  useBarcodeScanner: () => ({ triggerScan: jest.fn() }),
}))

jest.mock("@/lib/api", () => ({
  ...jest.requireActual("@/lib/api"),
  api: {
    customers: { list: jest.fn().mockResolvedValue({ data: [] }) },
    wms: {
      deliveryOrders: {
        getAuditTrail: jest.fn().mockResolvedValue({ data: [] }),
      },
    },
  },
}))

describe("WMS Frontend Fraud Controls", () => {
  beforeEach(() => {
    jest.clearAllMocks()
    mockUseWarehouses.mockReturnValue({
      data: [{ id: "wh-1", name: "Gudang Utama" }],
      isLoading: false,
    })
    mockUseProducts.mockReturnValue({
      data: [{ id: "prod-1", name: "Beras 5kg", sku: "SKU-BERAS-5K" }],
      isLoading: false,
    })
    mockUseStockScraps.mockReturnValue({
      data: [],
      isLoading: false,
      refetch: jest.fn(),
    })
    mockUseCreateStockScrap.mockReturnValue({
      mutateAsync: jest.fn().mockResolvedValue({ id: "scrap-1" }),
      isPending: false,
    })
    mockUseStockOpnames.mockReturnValue({
      data: [],
      isLoading: false,
      refetch: jest.fn(),
    })
    mockUseStockOpname.mockReturnValue({
      data: null,
      isLoading: false,
      refetch: jest.fn(),
    })
    mockUseCompleteStockOpname.mockReturnValue({
      mutateAsync: jest.fn().mockResolvedValue({ id: "opname-1" }),
      isPending: false,
    })
    mockUseDeliveryOrders.mockReturnValue({
      data: [],
      isLoading: false,
      refetch: jest.fn(),
    })
    mockUseDispatchDeliveryOrder.mockReturnValue({
      mutateAsync: jest.fn().mockResolvedValue({ id: "do-1" }),
      isPending: false,
    })
  })

  test("F2: Kirim button is rendered only for PACKED delivery orders, not DRAFT or CONFIRMED", () => {
    mockUseDeliveryOrders.mockReturnValue({
      data: [
        {
          id: "do-draft",
          do_number: "DO-DRAFT-01",
          status: "DRAFT",
          warehouse_id: "wh-1",
          recipient_name: "Customer A",
          created_at: new Date().toISOString(),
        },
        {
          id: "do-confirmed",
          do_number: "DO-CONF-01",
          status: "CONFIRMED",
          warehouse_id: "wh-1",
          recipient_name: "Customer B",
          created_at: new Date().toISOString(),
        },
        {
          id: "do-packed",
          do_number: "DO-PACKED-01",
          status: "PACKED",
          warehouse_id: "wh-1",
          recipient_name: "Customer C",
          created_at: new Date().toISOString(),
        },
      ],
      isLoading: false,
      refetch: jest.fn(),
    })

    render(<DeliveryOrdersPanel />)

    // Row DO-DRAFT-01 and DO-CONF-01 should not have Kirim button
    const kirimButtons = screen.queryAllByRole("button", { name: /Kirim/i })
    expect(kirimButtons.length).toBe(1)
    expect(screen.getByText("DO-PACKED-01")).toBeInTheDocument()
  })

  test("F3: Scrap validation enforces minimum 10 characters for reason and blocks >10 units for warehouse role", async () => {
    mockCurrentUser = { id: "user-1", name: "Staff", role: "warehouse" }

    render(<StockScrapPage />)

    // Open scrap modal
    fireEvent.click(screen.getAllByRole("button", { name: /Karantina Barang Rusak/i })[0])

    // Fill product and location
    fireEvent.change(screen.getByDisplayValue("-- Pilih Produk --"), { target: { value: "prod-1" } })
    fireEvent.change(screen.getByDisplayValue("-- Pilih Rak Asal --"), { target: { value: "loc-1" } })

    // Fill form with reason < 10 chars
    fireEvent.change(screen.getByDisplayValue("1"), { target: { value: "5" } })
    const reasonInput = screen.getByPlaceholderText(/Detail kronologi kerusakan/i)
    fireEvent.change(reasonInput, {
      target: { value: "rusak" }, // 5 chars
    })

    fireEvent.click(screen.getByRole("button", { name: /Karantina & Catat Scrap/i }))

    expect(await screen.findByText(/Alasan pemusnahan\/scrap barang harus minimal 10 karakter/i)).toBeInTheDocument()

    // Now try quantity > 10 for warehouse role
    fireEvent.change(screen.getByDisplayValue("5"), { target: { value: "15" } })
    fireEvent.change(reasonInput, {
      target: { value: "Kardus penyok parah terkena banjir" }, // > 10 chars
    })

    fireEvent.click(screen.getByRole("button", { name: /Karantina & Catat Scrap/i }))

    expect(await screen.findByText(/Pemusnahan stok di atas ambang batas \(10 unit\) wajib dilakukan oleh pengguna dengan peran Admin atau Owner/i)).toBeInTheDocument()
  })

  test("F3: Opname completion shows 'Ajukan untuk Persetujuan' for warehouse role", () => {
    mockCurrentUser = { id: "user-1", name: "Staff Gudang", role: "warehouse" }
    mockUseStockOpname.mockReturnValue({
      data: {
        opname: {
          id: "opname-1",
          opname_number: "OPN-2026-001",
          status: "IN_PROGRESS",
          warehouse_id: "wh-1",
          conducted_by: "user-1",
        },
        items: [
          {
            id: "item-1",
            product_id: "prod-1",
            location_id: "loc-1",
            system_qty: 10,
            physical_qty: 12,
            discrepancy_qty: 2,
          },
        ],
      },
      isLoading: false,
      refetch: jest.fn(),
    })

    mockUseStockOpnames.mockReturnValue({
      data: [
        {
          id: "opname-1",
          opname_number: "OPN-2026-001",
          status: "IN_PROGRESS",
          warehouse_id: "wh-1",
          created_at: new Date().toISOString(),
        },
      ],
      isLoading: false,
      refetch: jest.fn(),
    })

    render(<StockOpnamePage />)

    // Open active opname drawer
    fireEvent.click(screen.getByRole("button", { name: /Buka Penghitungan/i }))

    // The submit button for warehouse role should say "Ajukan untuk Persetujuan (Pending Approval)"
    expect(screen.getByRole("button", { name: /Ajukan untuk Persetujuan \(Pending Approval\)/i })).toBeInTheDocument()
  })
})
