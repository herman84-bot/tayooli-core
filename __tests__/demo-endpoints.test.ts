import { getDemoResponse } from "@/lib/api/demo-data"
import { VendorListSchema } from "@/lib/schemas/vendor"
import { POSchema } from "@/lib/schemas/po"
import { GRSchema } from "@/lib/schemas/gr"
import { PaymentOrderListSchema } from "@/lib/schemas/payment-order"
import { InvoiceListSchema } from "@/lib/schemas/invoice"
import { z } from "zod"

describe("Demo Data Fallbacks (P1-3, P2-3, P3-1, P4-2)", () => {
  const endpoints = [
    "/customers",
    "/sales-orders",
    "/sales-invoices",
    "/accounts",
    "/journal-entries",
    "/approvals",
    "/purchase-orders",
    "/goods-receipts",
    "/payment-orders",
    "/plans",
    "/subscription",
    "/team-members",
    "/ai/permissions",
    "/usage",
  ]

  test.each(endpoints)("endpoint %s returns non-null demo data", (endpoint) => {
    const res = getDemoResponse(endpoint, "")
    expect(res).not.toBeNull()
    expect(res).toBeDefined()
  })

  test("P1-3b: Vendors demo data parses successfully with VendorListSchema", () => {
    const res = getDemoResponse("/vendors", "")
    expect(() => VendorListSchema.parse(res)).not.toThrow()
    const parsed = VendorListSchema.parse(res)
    expect(parsed.data.length).toBeGreaterThan(0)
    for (const v of parsed.data) {
      expect(v.id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
      expect(v.status).toBe("active")
      expect(typeof v.avg_rating).toBe("string")
      expect(Number.isInteger(v.rating_count)).toBe(true)
    }
  })

  test("P1-3b: Purchase Orders demo data parses successfully with POSchema", () => {
    const res = getDemoResponse("/purchase-orders", "") as { data: unknown[] }
    expect(() => z.array(POSchema).parse(res.data)).not.toThrow()
    const pos = z.array(POSchema).parse(res.data)
    expect(pos.length).toBeGreaterThan(0)
    for (const po of pos) {
      expect(po.id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
      expect(typeof po.amount).toBe("string")
      expect(Number.isInteger(po.qty)).toBe(true)
    }
  })

  test("P1-3b: Goods Receipts demo data parses successfully with GRSchema", () => {
    const res = getDemoResponse("/goods-receipts", "") as { data: unknown[] }
    expect(() => z.array(GRSchema).parse(res.data)).not.toThrow()
    const grs = z.array(GRSchema).parse(res.data)
    expect(grs.length).toBeGreaterThan(0)
    for (const gr of grs) {
      expect(gr.id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
      expect(gr.po_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
      expect(typeof gr.received_amount).toBe("string")
      expect(Number.isInteger(gr.received_qty)).toBe(true)
    }
  })

  test("P1-3b: Payment Orders demo data parses successfully with PaymentOrderListSchema", () => {
    const res = getDemoResponse("/payment-orders", "")
    expect(() => PaymentOrderListSchema.parse(res)).not.toThrow()
    const parsed = PaymentOrderListSchema.parse(res)
    expect(parsed.data.length).toBeGreaterThan(0)
    for (const po of parsed.data) {
      expect(po.id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
      expect(po.invoice_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
      expect(typeof po.amount).toBe("string")
    }
  })

  test("P1-3b: Invoices demo data parses successfully with InvoiceListSchema", () => {
    const res = getDemoResponse("/invoices", "")
    expect(() => InvoiceListSchema.parse(res)).not.toThrow()
    const parsed = InvoiceListSchema.parse(res)
    expect(parsed.data.length).toBeGreaterThan(0)
    for (const inv of parsed.data) {
      expect(inv.id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
      expect(typeof inv.amount).toBe("string")
    }
  })

  test("P3-1: marketplace import summary matches actual orders returned (fallback without body)", () => {
    const res = getDemoResponse("/wms/marketplace/import", "") as {
      data: {
        batch: { total_orders: number; processed_orders: number; failed_orders: number }
        orders: unknown[]
      }
    }
    expect(res).not.toBeNull()
    const { batch, orders } = res.data
    expect(orders.length).toBe(3)
    expect(batch.total_orders).toBe(3)
    expect(batch.processed_orders).toBe(3)
    expect(batch.failed_orders).toBe(0)
    expect(batch.total_orders).toBe(batch.processed_orders + batch.failed_orders)
  })

  test("P3-1: marketplace import with 3 orders in body computes exact counts", () => {
    const payload = {
      warehouse_id: "wh-001",
      channel: "SHOPEE",
      file_name: "shopee_orders.csv",
      orders: [
        { external_order_id: "ORD-001", customer_name: "Customer 1", total_amount: 50000 },
        { external_order_id: "ORD-002", customer_name: "Customer 2", total_amount: 75000 },
        { external_order_id: "ORD-003", customer_name: "Customer 3", total_amount: 100000 },
      ],
    }
    const res = getDemoResponse("/wms/marketplace/import", "", payload, "POST") as {
      data: {
        batch: { total_orders: number; processed_orders: number; failed_orders: number }
        orders: Array<{ id: string; external_order_id: string }>
      }
    }
    expect(res).not.toBeNull()
    const { batch, orders } = res.data
    expect(orders.length).toBe(3)
    expect(batch.total_orders).toBe(3)
    expect(batch.processed_orders).toBe(3)
    expect(batch.failed_orders).toBe(0)
    expect(orders[0].external_order_id).toBe("ORD-001")
  })

  test("P0-1: POST /products creates and prepends product", () => {
    const payload = {
      name: "Produk Test Jest",
      sku: "SKU-JEST-001",
      price: 99000,
      description: "Produk uji otomatis",
    }
    const created = getDemoResponse("/products", "", payload, "POST") as {
      id: string
      name: string
      sku: string
      price: number
    }
    expect(created).toBeDefined()
    expect(created.name).toBe("Produk Test Jest")
    expect(created.sku).toBe("SKU-JEST-001")
    expect(created.price).toBe(99000)

    const listRes = getDemoResponse("/products", "") as {
      data: Array<{ name: string; sku: string }>
      total: number
    }
    expect(listRes.data[0].name).toBe("Produk Test Jest")
  })

  test("P4-2: POST /wms/delivery-orders/:id/dispatch mutates DO status to SHIPPED", () => {
    const dispatchRes = getDemoResponse("/wms/delivery-orders/do-001/dispatch", "", undefined, "POST") as {
      id: string
      status: string
    }
    expect(dispatchRes.status).toBe("SHIPPED")

    const listRes = getDemoResponse("/wms/delivery-orders", "") as {
      data: Array<{ id: string; status: string }>
    }
    const updated = listRes.data.find((d) => d.id === "do-001")
    expect(updated?.status).toBe("SHIPPED")
  })

  test("P4-2: POST /wms/transfers/:id/dispatch and receive mutates transfer status", () => {
    const dispatchRes = getDemoResponse("/wms/transfers/tr-002/dispatch", "", undefined, "POST") as {
      id: string
      status: string
    }
    expect(dispatchRes.status).toBe("IN_TRANSIT")

    const receiveRes = getDemoResponse("/wms/transfers/tr-001/receive", "", undefined, "POST") as {
      id: string
      status: string
    }
    expect(receiveRes.status).toBe("RECEIVED")
  })
})
