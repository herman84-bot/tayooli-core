import { matchRuleBasedAction } from "@/lib/copilot/fallback-matcher"
import { TAYOOLI_ERP_KNOWLEDGE_MAP } from "@/lib/copilot/knowledge-map"
import { COPILOT_TOOLS } from "@/lib/copilot/tools"

describe("Copilot WMS & POS Knowledge & Navigation", () => {
  it("includes WMS and POS in TAYOOLI_ERP_KNOWLEDGE_MAP", () => {
    expect(TAYOOLI_ERP_KNOWLEDGE_MAP).toContain("Warehouse & Logistics (WMS)")
    expect(TAYOOLI_ERP_KNOWLEDGE_MAP).toContain("Point of Sale (POS / Kasir Ritel)")
    expect(TAYOOLI_ERP_KNOWLEDGE_MAP).toContain("Stock Transfers")
    expect(TAYOOLI_ERP_KNOWLEDGE_MAP).toContain("Stock Opname")
    expect(TAYOOLI_ERP_KNOWLEDGE_MAP).toContain("Surat Jalan (Delivery Order / DO)")
    expect(TAYOOLI_ERP_KNOWLEDGE_MAP).toContain("Jual Putus vs Konsinyasi")
  })

  it("includes WMS and POS routes in navigate_to_module tool schema", () => {
    const navTool = COPILOT_TOOLS.find((t) => t.name === "navigate_to_module")
    expect(navTool).toBeDefined()
    const pathEnum = navTool?.parameters?.properties?.path?.enum
    expect(pathEnum).toContain("/pos")
    expect(pathEnum).toContain("/wms")
    expect(pathEnum).toContain("/wms/delivery-orders")
    expect(pathEnum).toContain("/wms/marketplace")
    expect(pathEnum).toContain("/wms/transfers")
    expect(pathEnum).toContain("/wms/opname")
    expect(pathEnum).toContain("/wms/scrap")
    expect(pathEnum).toContain("/wms/scanner")
  })

  const navTestCases = [
    { message: "buka kasir pos", expectedPath: "/pos", expectedMod: "Point of Sale" },
    { message: "ke menu point of sale", expectedPath: "/pos", expectedMod: "Point of Sale" },
    { message: "buka gudang", expectedPath: "/wms", expectedMod: "Warehouse & Stock" },
    { message: "buka menu transfer stok", expectedPath: "/wms/transfers", expectedMod: "Stock Transfers" },
    { message: "buka stock opname", expectedPath: "/wms/opname", expectedMod: "Stock Opname" },
    { message: "lihat barang rusak scrap", expectedPath: "/wms/scrap", expectedMod: "Barang Rusak / Scrap" },
    { message: "buka barcode scanner", expectedPath: "/wms/scanner", expectedMod: "Barcode Scanner" },
    { message: "buka surat jalan delivery order", expectedPath: "/wms/delivery-orders", expectedMod: "Surat Jalan (DO)" },
    { message: "buka marketplace omnichannel", expectedPath: "/wms/marketplace", expectedMod: "Marketplace Omnichannel" },
  ]

  navTestCases.forEach(({ message, expectedPath, expectedMod }) => {
    it(`navigates correctly for '${message}'`, () => {
      const res = matchRuleBasedAction(message)
      expect(res.actions.length).toBe(1)
      expect(res.actions[0].toolName).toBe("navigate_to_module")
      expect(res.actions[0].payload.path).toBe(expectedPath)
      expect(res.actions[0].payload.module_name).toBe(expectedMod)
    })
  })
})
