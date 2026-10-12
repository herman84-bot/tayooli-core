/**
 * @jest-environment jsdom
 */
import { api } from "@/lib/api"

function mockOkResponse(payload: any, status = 200) {
  return {
    ok: true,
    status,
    headers: {
      get: (header: string) => (header.toLowerCase() === "content-type" ? "application/json" : null),
    },
    json: async () => payload,
    text: async () => JSON.stringify(payload),
  }
}

describe("WMS API Idempotency-Key forwarding", () => {
  const originalFetch = global.fetch

  afterEach(() => {
    global.fetch = originalFetch
  })

  it("extracts idempotency_key from scraps.create and forwards as header", async () => {
    let capturedInit: RequestInit | undefined
    global.fetch = jest.fn().mockImplementation((_url, init) => {
      capturedInit = init
      return Promise.resolve(mockOkResponse({ id: "scr-1" }, 201))
    })

    await api.wms.scraps.create({
      warehouse_id: "wh-1",
      product_id: "prod-1",
      source_location_id: "loc-1",
      quantity: 5,
      reason: "Barang rusak kena air hujan",
      idempotency_key: "key-scrap-123",
    })

    expect(capturedInit).toBeDefined()
    const headers = capturedInit?.headers as Record<string, string>
    expect(headers["Idempotency-Key"]).toBe("key-scrap-123")
    const body = JSON.parse(capturedInit?.body as string)
    expect(body.idempotency_key).toBeUndefined()
    expect(body.product_id).toBe("prod-1")
    expect(body.quantity).toBe(5)
  })

  it("extracts idempotency_key from transfers.create and forwards as header", async () => {
    let capturedInit: RequestInit | undefined
    global.fetch = jest.fn().mockImplementation((_url, init) => {
      capturedInit = init
      return Promise.resolve(mockOkResponse({ id: "trf-1" }, 201))
    })

    await api.wms.transfers.create({
      from_warehouse_id: "wh-1",
      to_warehouse_id: "wh-2",
      items: [{ product_id: "p1", requested_qty: 10 }],
      idempotency_key: "key-trf-456",
    })

    expect(capturedInit).toBeDefined()
    const headers = capturedInit?.headers as Record<string, string>
    expect(headers["Idempotency-Key"]).toBe("key-trf-456")
    const body = JSON.parse(capturedInit?.body as string)
    expect(body.idempotency_key).toBeUndefined()
    expect(body.from_warehouse_id).toBe("wh-1")
  })

  it("sends Idempotency-Key header when completing opname", async () => {
    let capturedInit: RequestInit | undefined
    global.fetch = jest.fn().mockImplementation((_url, init) => {
      capturedInit = init
      return Promise.resolve(mockOkResponse({ id: "opn-1" }, 200))
    })

    await api.wms.opnames.complete("opn-1", "key-opn-complete-789")

    expect(capturedInit).toBeDefined()
    const headers = capturedInit?.headers as Record<string, string>
    expect(headers["Idempotency-Key"]).toBe("key-opn-complete-789")
  })
})
