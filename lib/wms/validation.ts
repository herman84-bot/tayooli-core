/**
 * Shared client-side validation for WMS inbound / putaway / outbound forms.
 * The backend enforces the same rules; these exist so the user gets immediate
 * feedback and cannot submit obviously invalid data.
 */

export function parseQty(v: string | number | null | undefined): number {
  if (v === null || v === undefined) return NaN
  if (typeof v === "number") return v
  const s = v.trim()
  if (s === "") return NaN
  // Reject partial parses like "5abc" or "1e3" that parseFloat would accept.
  if (!/^-?\d+(\.\d+)?$/.test(s)) return NaN
  return Number(s)
}

export function isNonNegativeInt(n: number): boolean {
  return Number.isInteger(n) && n >= 0
}

export function isPositiveInt(n: number): boolean {
  return Number.isInteger(n) && n > 0
}

/** Today as YYYY-MM-DD in local time (the format of <input type="date">). */
export function todayISODate(now = new Date()): string {
  const y = now.getFullYear()
  const m = String(now.getMonth() + 1).padStart(2, "0")
  const d = String(now.getDate()).padStart(2, "0")
  return `${y}-${m}-${d}`
}

export function isPastDate(dateStr: string | undefined | null, now = new Date()): boolean {
  if (!dateStr) return false
  return dateStr.slice(0, 10) < todayISODate(now)
}

export interface ReceiptLineInput {
  product_name?: string
  accepted: string
  rejected: string
  reject_reason: string
  expiry_date: string
  /** Optional ordered qty from a PO / supplier document, for over-receiving checks. */
  ordered_qty?: number | null
}

/**
 * Validate goods-receipt lines. Returns the first error message, or null.
 * Rules: integer qty >= 0, at least one real unit in total, reject reason
 * required when rejected > 0, no accepted stock with an expiry date in the
 * past, and no over-receiving against an ordered qty when one is known.
 */
export function validateReceiptLines(lines: ReceiptLineInput[], now = new Date()): string | null {
  if (lines.length === 0) return "Tambahkan minimal 1 item barang."
  let totalAccepted = 0
  let totalRejected = 0
  for (const l of lines) {
    const name = l.product_name || "Produk"
    const acc = parseQty(l.accepted)
    const rej = parseQty(l.rejected === "" ? "0" : l.rejected)
    if (!isNonNegativeInt(acc)) return `${name}: Qty Diterima harus bilangan bulat ≥ 0.`
    if (!isNonNegativeInt(rej)) return `${name}: Qty Ditolak harus bilangan bulat ≥ 0.`
    if (rej > 0 && !(l.reject_reason || "").trim()) return `${name}: Alasan penolakan wajib diisi karena Qty Ditolak > 0.`
    if (acc > 0 && isPastDate(l.expiry_date, now)) {
      return `${name}: Tanggal kedaluwarsa ${l.expiry_date} sudah lewat. Barang kedaluwarsa tidak boleh masuk stok normal — catat sebagai Qty Ditolak dengan alasan "Kedaluwarsa".`
    }
    if (l.ordered_qty !== undefined && l.ordered_qty !== null && Number.isFinite(l.ordered_qty) && acc + rej > l.ordered_qty) {
      return `${name}: Total diterima + ditolak (${acc + rej}) melebihi qty dokumen asal (${l.ordered_qty}).`
    }
    totalAccepted += acc
    totalRejected += rej
  }
  if (totalAccepted === 0 && totalRejected === 0) return "Total Qty Diterima dan Ditolak tidak boleh 0 semua. Tidak ada barang yang dicatat."
  return null
}

export const MIN_OVERRIDE_REASON_LEN = 5

/**
 * Validate a putaway confirmation. `available` is the batch qty still on hand
 * in staging. Returns the first error, or null.
 */
export function validatePutaway(opts: {
  qty: string | number
  available: number
  isOverride: boolean
  reason: string
}): string | null {
  const q = parseQty(opts.qty)
  if (!Number.isFinite(q) || q <= 0) return "Qty putaway harus lebih dari 0."
  if (Number.isFinite(opts.available) && q > opts.available) {
    return `Qty putaway (${q}) melebihi sisa stok di Staging (${opts.available}).`
  }
  if (opts.isOverride && opts.reason.trim().length < MIN_OVERRIDE_REASON_LEN) {
    return `Rak tujuan berbeda dari rak default. Alasan pemindahan wajib diisi (minimal ${MIN_OVERRIDE_REASON_LEN} karakter).`
  }
  return null
}

export interface StockRow {
  product_id: string
  location_id?: string | null
  quantity: number | string
  available_qty?: number | string
}

/**
 * Sellable qty for a product, at one location or across all locations of the
 * warehouse when `locationId` is empty (Auto FEFO). Uses `available_qty` when
 * the backend sends it, otherwise falls back to on-hand `quantity`.
 */
export function availableFor(stock: StockRow[], productId: string, locationId?: string | null): number {
  let total = 0
  for (const s of stock) {
    if (s.product_id !== productId) continue
    if (locationId && s.location_id !== locationId) continue
    const raw = s.available_qty !== undefined && s.available_qty !== null ? s.available_qty : s.quantity
    const n = typeof raw === "number" ? raw : Number(raw)
    if (Number.isFinite(n) && n > 0) total += n
  }
  return total
}

/**
 * Validate one delivery-order line. `available` is the sellable stock at the
 * chosen source (on hand − allocated − quarantine); pass undefined if unknown.
 */
export function validateDOLineQty(qty: string | number, available?: number): string | null {
  const q = parseQty(qty)
  if (!isPositiveInt(q)) return "Qty harus bilangan bulat lebih dari 0."
  if (available !== undefined && Number.isFinite(available) && q > available) {
    return `Stok tidak mencukupi, sisa available: ${Math.max(0, available)}`
  }
  return null
}
