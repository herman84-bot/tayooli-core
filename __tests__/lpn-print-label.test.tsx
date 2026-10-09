import React from "react"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import "@testing-library/jest-dom"
import { PrintLPNLabel, LPNBarcodeSVG, generateCode128Bars } from "@/components/wms/PrintLPNLabel"
import { LPNManagementModal } from "@/components/wms/LPNManagementModal"
import type { StockLPN, StockLPNDetail, StockLPNItem, WarehouseLocation, Product } from "@/lib/api"

// ---------------------------------------------------------------------------
// Mock Data
// ---------------------------------------------------------------------------
const mockLPN: StockLPN = {
  id: "lpn-uuid-1",
  warehouse_id: "wh-1",
  warehouse_name: "Gudang Logistik Pusat",
  lpn_code: "LPN-20261008-0001",
  location_id: "loc-staging-1",
  location_code: "STG-IN-01",
  location_name: "Staging Area 1",
  pallet_type: "WOODEN",
  status: "STAGED",
  max_weight_kg: 1000,
  total_weight_kg: 350.5,
  notes: "Palet kayu muatan bahan baku kopi",
  created_at: "2026-10-08T08:00:00Z",
  updated_at: "2026-10-08T08:00:00Z",
}

const mockItems: StockLPNItem[] = [
  {
    id: "item-1",
    lpn_id: "lpn-uuid-1",
    product_id: "prod-1",
    product_name: "Biji Kopi Arabika 1kg",
    product_sku: "KOP-ARB-001",
    batch_id: "batch-1",
    batch_number: "LOT-ARB-202610",
    expiry_date: "2027-10-08T00:00:00Z",
    quantity: 150,
    created_at: "2026-10-08T08:10:00Z",
    updated_at: "2026-10-08T08:10:00Z",
  },
  {
    id: "item-2",
    lpn_id: "lpn-uuid-1",
    product_id: "prod-2",
    product_name: "Biji Kopi Robusta 1kg",
    product_sku: "KOP-ROB-002",
    batch_id: "batch-2",
    batch_number: "LOT-ROB-202610",
    expiry_date: "2027-10-08T00:00:00Z",
    quantity: 200,
    created_at: "2026-10-08T08:15:00Z",
    updated_at: "2026-10-08T08:15:00Z",
  },
]

const mockLPNDetail: StockLPNDetail = {
  lpn: mockLPN,
  items: mockItems,
}

const mockLocations: WarehouseLocation[] = [
  {
    id: "loc-staging-1",
    tenant_id: "tenant-1",
    warehouse_id: "wh-1",
    code: "STG-IN-01",
    name: "Staging Area Inbound 1",
    type: "INTERNAL",
    is_pallet: true,
    created_at: "2026-10-08T00:00:00Z",
    updated_at: "2026-10-08T00:00:00Z",
  },
  {
    id: "loc-rack-a1",
    tenant_id: "tenant-1",
    warehouse_id: "wh-1",
    code: "RCK-A-01",
    name: "Rak Utama A1",
    type: "INTERNAL",
    is_pallet: true,
    created_at: "2026-10-08T00:00:00Z",
    updated_at: "2026-10-08T00:00:00Z",
  },
]

const mockProducts: Product[] = [
  {
    id: "prod-1",
    tenant_id: "tenant-1",
    name: "Biji Kopi Arabika 1kg",
    sku: "KOP-ARB-001",
    price: 95000,
    created_at: "2026-10-08T00:00:00Z",
    updated_at: "2026-10-08T00:00:00Z",
  },
]

// ---------------------------------------------------------------------------
// Mocks for Hooks and APIs
// ---------------------------------------------------------------------------
const mockCreateLPNMutate = jest.fn().mockResolvedValue({
  data: {
    ...mockLPN,
    id: "lpn-new-123",
    lpn_code: "LPN-20261008-0002",
  },
})

const mockAddLPNItemMutate = jest.fn().mockResolvedValue({
  data: mockLPNDetail,
})

jest.mock("@/hooks/useWMSDocksAndLPNs", () => ({
  useStockLPNs: jest.fn(() => ({
    data: [mockLPN],
    isLoading: false,
    refetch: jest.fn(),
  })),
  useStockLPNDetail: jest.fn((id?: string | null) => ({
    data: id === mockLPN.id ? mockLPNDetail : null,
    isLoading: false,
    refetch: jest.fn(),
  })),
  useCreateLPN: jest.fn(() => ({
    mutateAsync: mockCreateLPNMutate,
    isPending: false,
  })),
  useAddLPNItem: jest.fn(() => ({
    mutateAsync: mockAddLPNItemMutate,
    isPending: false,
  })),
}))

jest.mock("@/hooks/useWMS", () => ({
  useWarehouseLocations: jest.fn(() => ({
    data: mockLocations,
    isLoading: false,
  })),
  usePutawayPending: jest.fn(() => ({
    data: [
      {
        product_id: "prod-1",
        product_name: "Biji Kopi Arabika 1kg",
        product_sku: "KOP-ARB-001",
        batch_id: "batch-1",
        batch_number: "LOT-ARB-202610",
        quantity: 150,
      },
    ],
    isLoading: false,
  })),
}))

jest.mock("@/hooks/useProducts", () => ({
  useProducts: jest.fn(() => ({
    data: mockProducts,
    isLoading: false,
  })),
}))

jest.mock("@/lib/api", () => {
  const actual = jest.requireActual("@/lib/api")
  return {
    ...actual,
    api: {
      ...actual.api,
      wms: {
        ...actual.api?.wms,
        lpns: {
          list: jest.fn(() => Promise.resolve({ data: [mockLPN] })),
          get: jest.fn(() => Promise.resolve({ data: mockLPNDetail })),
          create: jest.fn(() => Promise.resolve({ data: mockLPN })),
          addItem: jest.fn(() => Promise.resolve({ data: mockLPNDetail })),
          move: jest.fn(() => Promise.resolve({ data: mockLPNDetail })),
        },
      },
    },
  }
})

// ---------------------------------------------------------------------------
// Unit Tests
// ---------------------------------------------------------------------------

describe("PrintLPNLabel Component", () => {
  const originalPrint = window.print

  beforeAll(() => {
    window.print = jest.fn()
  })

  afterAll(() => {
    window.print = originalPrint
  })

  beforeEach(() => {
    jest.clearAllMocks()
  })

  test("generates Code 128 bars with valid module counts", () => {
    const result = generateCode128Bars("LPN-20261008-0001")
    expect(result.bars.length).toBeGreaterThan(0)
    expect(result.totalWidth).toBeGreaterThan(100)
  })

  test("renders LPNBarcodeSVG with svg element and accessible role", () => {
    render(<LPNBarcodeSVG code="LPN-TEST-123" />)
    const svg = screen.getByTestId("lpn-barcode-svg")
    expect(svg).toBeInTheDocument()
    expect(svg).toHaveAttribute("role", "img")
    expect(svg).toHaveAttribute("aria-label", "Barcode untuk LPN-TEST-123")
  })

  test("renders all header and LPN pallet specifications", () => {
    render(<PrintLPNLabel lpnDetail={mockLPNDetail} onClose={jest.fn()} />)

    // Header specifications
    expect(screen.getByText(/TAYOOLI ERP - LOGISTICS/i)).toBeInTheDocument()
    expect(screen.getByText("Gudang Logistik Pusat")).toBeInTheDocument()

    // Giant LPN code text
    expect(screen.getAllByText("LPN-20261008-0001").length).toBeGreaterThan(0)

    // Pallet Type & Location
    expect(screen.getByText(/KAYU \(WOODEN\)/i)).toBeInTheDocument()
    expect(screen.getByText("STG-IN-01")).toBeInTheDocument()

    // Weight specifications
    expect(screen.getByText(/350.5 kg \/ 1000 kg/i)).toBeInTheDocument()

    // Line items table
    expect(screen.getByText("LOT-ARB-202610")).toBeInTheDocument()
    expect(screen.getByText("KOP-ARB-001")).toBeInTheDocument()
    expect(screen.getByText("Biji Kopi Arabika 1kg")).toBeInTheDocument()
    expect(screen.getByText("150")).toBeInTheDocument()

    expect(screen.getByText("LOT-ROB-202610")).toBeInTheDocument()
    expect(screen.getByText("KOP-ROB-002")).toBeInTheDocument()
    expect(screen.getByText("200")).toBeInTheDocument()

    // Total units sum (150 + 200 = 350)
    expect(screen.getByText(/Total: 350 unit/i)).toBeInTheDocument()
  })

  test("renders empty pallet message when items array is empty", () => {
    const emptyDetail: StockLPNDetail = {
      lpn: mockLPN,
      items: [],
    }
    render(<PrintLPNLabel lpnDetail={emptyDetail} onClose={jest.fn()} />)
    expect(screen.getByText(/Palet kosong \(belum ada item batch yang dimuat\)/i)).toBeInTheDocument()
  })

  test("triggers window.print when Cetak Label button is clicked", () => {
    render(<PrintLPNLabel lpnDetail={mockLPNDetail} onClose={jest.fn()} />)
    const printBtn = screen.getByRole("button", { name: /Cetak Label/i })
    fireEvent.click(printBtn)
    expect(window.print).toHaveBeenCalledTimes(1)
  })

  test("triggers onClose callback when Tutup button is clicked", () => {
    const onCloseMock = jest.fn()
    render(<PrintLPNLabel lpnDetail={mockLPNDetail} onClose={onCloseMock} />)
    const closeBtn = screen.getByRole("button", { name: "Tutup" })
    fireEvent.click(closeBtn)
    expect(onCloseMock).toHaveBeenCalledTimes(1)
  })
})

describe("LPNManagementModal Component", () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test("does not render when isOpen is false", () => {
    const { container } = render(
      <LPNManagementModal isOpen={false} onClose={jest.fn()} warehouseId="wh-1" />
    )
    expect(container.firstChild).toBeNull()
  })

  test("renders modal and active LPN list when isOpen is true", () => {
    render(<LPNManagementModal isOpen={true} onClose={jest.fn()} warehouseId="wh-1" />)

    expect(screen.getByText(/Manajemen Palet & Kontainer LPN/i)).toBeInTheDocument()
    expect(screen.getByText("LPN-20261008-0001")).toBeInTheDocument()
    expect(screen.getByText(/Staging \(STAGED\)/i)).toBeInTheDocument()
    expect(screen.getByText("STG-IN-01")).toBeInTheDocument()
  })

  test("opens Buat Palet LPN form and submits new pallet container", async () => {
    render(<LPNManagementModal isOpen={true} onClose={jest.fn()} warehouseId="wh-1" />)

    const createBtn = screen.getByRole("button", { name: /Buat Palet LPN Baru/i })
    fireEvent.click(createBtn)

    // Form should appear
    expect(screen.getByText("Buat Palet LPN Baru")).toBeInTheDocument()

    // Location select is available
    const locationSelect = screen.getByLabelText(/Lokasi Staging/i)
    fireEvent.change(locationSelect, { target: { value: "loc-staging-1" } })

    // Submit form
    const submitBtn = screen.getByRole("button", { name: /Buat Palet LPN/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockCreateLPNMutate).toHaveBeenCalledWith(
        expect.objectContaining({
          warehouse_id: "wh-1",
          location_id: "loc-staging-1",
          pallet_type: "WOODEN",
          max_weight_kg: 1000,
        })
      )
    })
  })

  test("views LPN items detail when Lihat Isi is clicked", async () => {
    render(<LPNManagementModal isOpen={true} onClose={jest.fn()} warehouseId="wh-1" />)

    const detailBtn = screen.getByRole("button", { name: "Lihat Isi" })
    fireEvent.click(detailBtn)

    // Detail view should display
    expect(screen.getByText(/Kembali ke Daftar Palet/i)).toBeInTheDocument()
    expect(screen.getByText("LOT-ARB-202610")).toBeInTheDocument()
    expect(screen.getByText("LOT-ROB-202610")).toBeInTheDocument()
  })

  test("opens Tambah Item form and submits batch allocation", async () => {
    render(<LPNManagementModal isOpen={true} onClose={jest.fn()} warehouseId="wh-1" />)

    const addItemBtn = screen.getByRole("button", { name: "Tambah Item" })
    fireEvent.click(addItemBtn)

    // Form should appear
    expect(screen.getByText(/Tambah Item Batch ke Palet/i)).toBeInTheDocument()

    // Click quick pick from pending staging item
    const quickPickBtn = screen.getByText("Biji Kopi Arabika 1kg")
    fireEvent.click(quickPickBtn)

    // Submit button
    const submitBtn = screen.getByRole("button", { name: /Masukkan ke Palet/i })
    fireEvent.click(submitBtn)

    await waitFor(() => {
      expect(mockAddLPNItemMutate).toHaveBeenCalledWith(
        expect.objectContaining({
          lpnId: "lpn-uuid-1",
          data: {
            product_id: "prod-1",
            batch_id: "batch-1",
            quantity: 150,
          },
        })
      )
    })
  })

  test("opens thermal label print modal when Cetak Label Palet is clicked", async () => {
    render(<LPNManagementModal isOpen={true} onClose={jest.fn()} warehouseId="wh-1" />)

    const printBtn = screen.getByRole("button", { name: "Cetak Label Palet" })
    fireEvent.click(printBtn)

    await waitFor(() => {
      expect(screen.getByText(/Label Palet LPN \(100x150 mm\)/i)).toBeInTheDocument()
      expect(screen.getByText(/TAYOOLI ERP - LOGISTICS/i)).toBeInTheDocument()
    })
  })

  test("triggers onClose when header close button is clicked", () => {
    const onCloseMock = jest.fn()
    render(<LPNManagementModal isOpen={true} onClose={onCloseMock} warehouseId="wh-1" />)

    const closeBtn = screen.getByRole("button", { name: "Tutup" })
    fireEvent.click(closeBtn)
    expect(onCloseMock).toHaveBeenCalledTimes(1)
  })
})
