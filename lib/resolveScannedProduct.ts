import { api, type ApiHttpError, type Product } from "@/lib/api"

export type ScanResolution =
  | { kind: "found"; product: Product }
  | { kind: "not_found" }
  | { kind: "error"; message: string }

/**
 * resolveScannedProduct maps a scanned/typed code to a product without loading
 * the full catalog (which the list endpoint caps at 1000 rows).
 *
 * 1. Server resolve: SKU, barcode, or marketplace SKU mapping.
 * 2. On 404 only: server search by name/SKU, accepted when exactly one product
 *    matches (keeps the old "type a name in the scan box" habit without
 *    guessing between several candidates).
 *
 * Network/server failures are reported as "error", not "not_found", so an
 * outage is not mistaken for an unknown barcode.
 */
export async function resolveScannedProduct(code: string): Promise<ScanResolution> {
  const clean = code.trim()
  if (!clean) return { kind: "not_found" }

  try {
    const resolved = await api.wms.barcodes.resolve(clean)
    return { kind: "found", product: await api.products.get(resolved.product_id) }
  } catch (err) {
    if ((err as ApiHttpError)?.status !== 404) {
      return { kind: "error", message: err instanceof Error ? err.message : "Gagal menghubungi server." }
    }
  }

  try {
    const res = await api.products.list({ q: clean, page: 1, per_page: 2 })
    const hits = Array.isArray(res?.data) ? res.data : []
    return hits.length === 1 ? { kind: "found", product: hits[0] } : { kind: "not_found" }
  } catch (err) {
    return { kind: "error", message: err instanceof Error ? err.message : "Gagal menghubungi server." }
  }
}
