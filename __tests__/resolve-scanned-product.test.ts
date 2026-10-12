import { resolveScannedProduct } from "@/lib/resolveScannedProduct"
import { api } from "@/lib/api"

jest.mock("@/lib/api", () => ({
  api: {
    wms: { barcodes: { resolve: jest.fn() } },
    products: { get: jest.fn(), list: jest.fn() },
  },
}))

const resolve = api.wms.barcodes.resolve as jest.Mock
const getProduct = api.products.get as jest.Mock
const listProducts = api.products.list as jest.Mock

const httpError = (status: number, message = "x") => Object.assign(new Error(message), { status })
const beras = { id: "p-1", name: "Beras 5kg", sku: "BRS-5" }

beforeEach(() => jest.resetAllMocks())

describe("resolveScannedProduct", () => {
  it("returns not_found for blank input without calling the server", async () => {
    await expect(resolveScannedProduct("   ")).resolves.toEqual({ kind: "not_found" })
    expect(resolve).not.toHaveBeenCalled()
  })

  it("resolves SKU/barcode via the server and loads the product", async () => {
    resolve.mockResolvedValue({ product_id: "p-1" })
    getProduct.mockResolvedValue(beras)
    await expect(resolveScannedProduct(" BRS-5 ")).resolves.toEqual({ kind: "found", product: beras })
    expect(resolve).toHaveBeenCalledWith("BRS-5")
    expect(listProducts).not.toHaveBeenCalled()
  })

  it("falls back to name search on 404 and accepts exactly one match", async () => {
    resolve.mockRejectedValue(httpError(404))
    listProducts.mockResolvedValue({ data: [beras] })
    await expect(resolveScannedProduct("beras")).resolves.toEqual({ kind: "found", product: beras })
    expect(listProducts).toHaveBeenCalledWith({ q: "beras", page: 1, per_page: 2 })
  })

  it("does not guess when the name search is ambiguous", async () => {
    resolve.mockRejectedValue(httpError(404))
    listProducts.mockResolvedValue({ data: [beras, { ...beras, id: "p-2" }] })
    await expect(resolveScannedProduct("beras")).resolves.toEqual({ kind: "not_found" })
  })

  it("reports server/network failures as error, not as an unknown barcode", async () => {
    resolve.mockRejectedValue(httpError(503, "Layanan sibuk"))
    await expect(resolveScannedProduct("BRS-5")).resolves.toEqual({ kind: "error", message: "Layanan sibuk" })
    expect(listProducts).not.toHaveBeenCalled()
  })
})
