import React from "react"
import { render, screen, fireEvent } from "@testing-library/react"
import "@testing-library/jest-dom"
import MarketplacePage from "@/app/(app)/wms/marketplace/page"

const mockBatches = [
  {
    id: "batch-1",
    batch_number: "BATCH-MP-001",
    channel: "SHOPEE",
    processed_orders: 5,
    file_name: "shopee_orders.csv",
    status: "PENDING_APPROVAL",
    uploaded_by: "user-admin-1",
    created_at: new Date().toISOString(),
  },
]

let mockCurrentUser = {
  id: "user-admin-1",
  name: "Admin User",
  role: "admin",
}

const mockDecideBatchMutate = jest.fn()

jest.mock("@/hooks/useAuth", () => ({
  useAuthStore: (selector: any) => selector({ user: mockCurrentUser }),
}))

jest.mock("@/hooks/useWMS", () => ({
  useWarehouses: () => ({ data: [{ id: "wh-1", name: "Gudang Utama" }], isLoading: false }),
  useWarehouseLocations: () => ({ data: [{ id: "loc-1", name: "Rak A1", code: "RAK-A1", type: "INTERNAL" }] }),
  useMarketplaceBatches: () => ({ data: mockBatches, isLoading: false, refetch: jest.fn() }),
  useDecideMarketplaceBatch: () => ({ mutate: mockDecideBatchMutate, isPending: false }),
  useMarketplaceOrders: () => ({ data: [], isLoading: false, refetch: jest.fn(), dataUpdatedAt: Date.now() }),
  useMarketplaceOrder: () => ({ data: null, isLoading: false }),
  useMarketplaceSKUMappings: () => ({ data: [], isLoading: false, refetch: jest.fn() }),
  useImportMarketplaceOrders: () => ({ mutateAsync: jest.fn(), isPending: false }),
  useCreateSKUMapping: () => ({ mutateAsync: jest.fn(), isPending: false }),
}))

jest.mock("@/hooks/useManualRefresh", () => ({
  useManualRefresh: () => ({ refresh: jest.fn(), status: "idle", refreshError: null }),
  AUTO_REFRESH_MS: 30000,
}))

describe("Marketplace Batch Approval UI", () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  test("allows Admin or Owner to see and click Setujui & Potong Stok button", () => {
    mockCurrentUser = { id: "user-admin-1", name: "Admin User", role: "admin" }

    render(<MarketplacePage />)

    // Switch to tab 'orders' (Daftar Pesanan)
    const ordersTabBtn = screen.getByRole("button", { name: /Daftar Pesanan/i })
    fireEvent.click(ordersTabBtn)

    expect(screen.getByText("BATCH-MP-001")).toBeInTheDocument()
    const approveBtn = screen.getByRole("button", { name: /Setujui & Potong Stok/i })
    expect(approveBtn).toBeInTheDocument()

    fireEvent.click(approveBtn)
    expect(mockDecideBatchMutate).toHaveBeenCalledWith(
      { id: "batch-1", action: "approve" },
      expect.any(Object)
    )
  })

  test("hides approval buttons and shows waiting notice for warehouse staff", () => {
    mockCurrentUser = { id: "user-staff-1", name: "Staff Gudang", role: "warehouse" }

    render(<MarketplacePage />)

    // Switch to tab 'orders' (Daftar Pesanan)
    const ordersTabBtn = screen.getByRole("button", { name: /Daftar Pesanan/i })
    fireEvent.click(ordersTabBtn)

    expect(screen.getByText("BATCH-MP-001")).toBeInTheDocument()
    expect(screen.queryByRole("button", { name: /Setujui & Potong Stok/i })).not.toBeInTheDocument()
    expect(screen.getByText(/Menunggu persetujuan Admin\/Owner/i)).toBeInTheDocument()
  })
})
