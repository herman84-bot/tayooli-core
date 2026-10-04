import {
  MarketplaceChannel,
  ImportMarketplaceOrderInput,
  ImportMarketplaceOrderItemPayload,
} from "./api"

export interface ParsedCSVRow {
  external_order_id: string
  external_sku: string
  item_name: string
  quantity: number
  unit_price: number
  subtotal: number
  total_amount: number
  shipping_fee: number
  marketplace_fee: number
  customer_name: string
  customer_phone: string
  shipping_address: string
  courier: string
  tracking_number: string
  order_date: string
}

export interface ParsedOrderPreview extends ImportMarketplaceOrderInput {
  isValid: boolean
  validationErrors: string[]
  detectedChannel: MarketplaceChannel
}

export interface ParseResult {
  detectedChannel: MarketplaceChannel
  headers: string[]
  orders: ParsedOrderPreview[]
  rawRowCount: number
  validOrderCount: number
  errorOrderCount: number
  totalRevenue: number
  unmappedSkuSet: string[]
  errors: string[]
}

/** Sanitize input string to prevent CSV/Spreadsheet formula injection (CWE-1236) */
export function sanitizeCellValue(val: string): string {
  if (!val) return ""
  const trimmed = val.trim()
  if (
    trimmed.startsWith("=") ||
    trimmed.startsWith("+") ||
    trimmed.startsWith("@") ||
    (trimmed.startsWith("-") && isNaN(Number(trimmed)))
  ) {
    return `'${trimmed}`
  }
  return trimmed
}

/** Clean currency and Indonesian number strings: "Rp 150.000,00" -> 150000 */
export function parseCurrencyOrNumber(val: unknown): number {
  if (typeof val === "number") return isNaN(val) ? 0 : val
  if (!val || typeof val !== "string") return 0

  let clean = val
    .replace(/[Rr][Pp]\.?\s*/g, "")
    .replace(/[Ii][Dd][Rr]\s*/g, "")
    .trim()

  // Indonesian format with thousands separator dot and decimal comma: e.g. "150.000,50"
  if (clean.includes(".") && clean.includes(",")) {
    clean = clean.replace(/\./g, "").replace(",", ".")
  } else if (clean.includes(".") && !clean.includes(",")) {
    // Check if dot is thousands separator like "150.000" (3 digits after dot)
    const parts = clean.split(".")
    if (parts.length > 1 && parts.every((p, idx) => idx === 0 || p.length === 3)) {
      clean = clean.replace(/\./g, "")
    }
  } else if (clean.includes(",")) {
    clean = clean.replace(",", ".")
  }

  const num = parseFloat(clean)
  return isNaN(num) ? 0 : num
}

/** Header matching rules */
const HEADER_SYNONYMS: Record<keyof ParsedCSVRow, string[]> = {
  external_order_id: [
    "nomor pesanan",
    "no. pesanan",
    "no pesanan",
    "order id",
    "order sn",
    "order_id",
    "external order id",
    "external_order_id",
    "ordernumber",
    "invoice",
  ],
  external_sku: [
    "nomor referensi sku",
    "no. referensi sku",
    "sku induk",
    "sku",
    "seller sku",
    "item sku",
    "product sku",
    "external sku",
    "external_sku",
    "sellersku",
    "kode sku",
  ],
  item_name: [
    "nama produk",
    "nama barang",
    "product name",
    "item name",
    "product",
    "itemname",
    "deskripsi barang",
  ],
  quantity: ["jumlah", "quantity", "qty", "jumlah produk", "kuantitas"],
  unit_price: [
    "harga awal",
    "harga satuan",
    "unit price",
    "deal price",
    "harga",
    "price",
    "unit_price",
    "unitprice",
  ],
  subtotal: [
    "total harga produk",
    "subtotal",
    "total price",
    "jumlah harga",
    "sub_total",
  ],
  total_amount: [
    "total pembayaran",
    "total amount",
    "grand total",
    "total pesanan",
    "total",
    "total belanja",
  ],
  shipping_fee: [
    "ongkos kirim dibayar pembeli",
    "ongkir",
    "shipping fee",
    "biaya pengiriman",
    "shipping_fee",
    "ongkos kirim",
  ],
  marketplace_fee: [
    "biaya layanan",
    "biaya transaksi",
    "marketplace fee",
    "service fee",
    "marketplace_fee",
    "potongan marketplace",
  ],
  customer_name: [
    "nama pembeli",
    "customer name",
    "username (pembeli)",
    "nama penerima",
    "customer",
    "pembeli",
    "recipient name",
  ],
  customer_phone: [
    "nomor telepon pembeli",
    "no. telepon",
    "no telepon",
    "phone number",
    "phone",
    "telepon",
    "kontak",
    "recipient phone",
  ],
  shipping_address: [
    "alamat pengiriman",
    "shipping address",
    "alamat penerima",
    "alamat",
    "address",
    "delivery address",
  ],
  courier: [
    "opsi pengiriman",
    "kurir",
    "shipping option",
    "jasa kirim",
    "courier",
    "ekspedisi",
    "delivery service",
  ],
  tracking_number: [
    "no. resi",
    "no resi",
    "tracking number",
    "nomor pelacakan",
    "tracking",
    "airway bill",
    "waybill",
    "resi",
  ],
  order_date: [
    "waktu pesanan dibuat",
    "tanggal pesanan",
    "order date",
    "waktu pembayaran",
    "created time",
    "created_at",
    "date",
  ],
}

/** Normalize header title for tolerant matching */
function normalizeHeader(h: string): string {
  return h
    .toLowerCase()
    .replace(/^\uFEFF/, "") // Strip BOM
    .trim()
}

/** Detect marketplace channel from header patterns */
export function detectChannelFromHeaders(headers: string[]): MarketplaceChannel {
  const normalized = headers.map(normalizeHeader)

  // Shopee hallmarks
  const isShopee = normalized.some(
    (h) =>
      h.includes("no. pesanan") ||
      h.includes("nomor referensi sku") ||
      h.includes("ongkos kirim dibayar pembeli") ||
      h.includes("sku induk")
  )
  if (isShopee) return "SHOPEE"

  // Tokopedia hallmarks
  const isTokopedia = normalized.some(
    (h) =>
      h.includes("nomor pesanan") ||
      h.includes("nama produk") ||
      h.includes("total pembayaran")
  ) && normalized.some((h) => h.includes("harga satuan") || h.includes("kurir"))
  if (isTokopedia) return "TOKOPEDIA"

  // TikTok hallmarks
  const isTikTok = normalized.some(
    (h) =>
      h.includes("seller sku") ||
      h.includes("order id") ||
      h.includes("product name") ||
      h.includes("tiktok")
  )
  if (isTikTok) return "TIKTOK"

  // Lazada hallmarks
  const isLazada = normalized.some(
    (h) =>
      h.includes("ordernumber") ||
      h.includes("sellersku") ||
      h.includes("itemname")
  )
  if (isLazada) return "LAZADA"

  return "OTHER"
}

/** Split a CSV row respecting double quotes */
function splitCSVLine(line: string, separator: string): string[] {
  const result: string[] = []
  let current = ""
  let inQuotes = false

  for (let i = 0; i < line.length; i++) {
    const char = line[i]
    if (char === '"') {
      if (inQuotes && line[i + 1] === '"') {
        current += '"'
        i++ // Skip escaped quote
      } else {
        inQuotes = !inQuotes
      }
    } else if (char === separator && !inQuotes) {
      result.push(current.trim())
      current = ""
    } else {
      current += char
    }
  }
  result.push(current.trim())
  return result
}

/** Parse CSV string to canonical Marketplace Orders */
export function parseMarketplaceCSV(
  csvText: string,
  preferredChannel?: MarketplaceChannel
): ParseResult {
  const errors: string[] = []
  if (!csvText || !csvText.trim()) {
    return {
      detectedChannel: preferredChannel || "OTHER",
      headers: [],
      orders: [],
      rawRowCount: 0,
      validOrderCount: 0,
      errorOrderCount: 0,
      totalRevenue: 0,
      unmappedSkuSet: [],
      errors: ["File kosong atau tidak mengandung data tabular."],
    }
  }

  // Remove Windows carriage returns and split into lines
  const lines = csvText.replace(/\r\n/g, "\n").replace(/\r/g, "\n").split("\n").filter((l) => l.trim().length > 0)
  if (lines.length < 2) {
    return {
      detectedChannel: preferredChannel || "OTHER",
      headers: [],
      orders: [],
      rawRowCount: 0,
      validOrderCount: 0,
      errorOrderCount: 0,
      totalRevenue: 0,
      unmappedSkuSet: [],
      errors: ["File harus memiliki baris header dan minimal satu baris data pesanan."],
    }
  }

  // Determine delimiter (, or ;)
  const firstLine = lines[0]
  const commaCount = (firstLine.match(/,/g) || []).length
  const semicolonCount = (firstLine.match(/;/g) || []).length
  const tabCount = (firstLine.match(/\t/g) || []).length
  let separator = ","
  if (tabCount > commaCount && tabCount > semicolonCount) separator = "\t"
  else if (semicolonCount > commaCount) separator = ";"

  const rawHeaders = splitCSVLine(lines[0], separator)
  const normHeaders = rawHeaders.map(normalizeHeader)

  // Map each standard field to its column index in the CSV
  const colIndexMap: Partial<Record<keyof ParsedCSVRow, number>> = {}

  for (const [key, synonyms] of Object.entries(HEADER_SYNONYMS)) {
    const fieldKey = key as keyof ParsedCSVRow
    const foundIndex = normHeaders.findIndex((h) =>
      synonyms.some((syn) => h === syn || h.includes(syn))
    )
    if (foundIndex !== -1) {
      colIndexMap[fieldKey] = foundIndex
    }
  }

  // If order id or product/sku not found, record errors
  if (colIndexMap.external_order_id === undefined) {
    errors.push("Kolom 'Nomor Pesanan / Order ID' tidak ditemukan pada header CSV.")
  }
  if (colIndexMap.external_sku === undefined && colIndexMap.item_name === undefined) {
    errors.push("Kolom SKU atau Nama Produk tidak ditemukan pada header CSV.")
  }

  const detectedChannel = preferredChannel || detectChannelFromHeaders(rawHeaders)

  // Group rows by external_order_id
  const orderMap = new Map<
    string,
    {
      orderData: Partial<ImportMarketplaceOrderInput>
      items: ImportMarketplaceOrderItemPayload[]
      validationErrors: string[]
    }
  >()

  const unmappedSkuSet = new Set<string>()
  let rawRowCount = 0

  for (let i = 1; i < lines.length; i++) {
    const rawLine = lines[i].trim()
    if (!rawLine) continue
    rawRowCount++

    const cols = splitCSVLine(rawLine, separator)

    const getCol = (key: keyof ParsedCSVRow): string => {
      const idx = colIndexMap[key]
      return idx !== undefined && idx < cols.length ? sanitizeCellValue(cols[idx]) : ""
    }

    const orderId = getCol("external_order_id") || `ROW-ORD-${i}`
    const rawSku = getCol("external_sku") || `SKU-AUTO-${i}`
    const rawItemName = getCol("item_name") || rawSku || `Barang Pesanan #${orderId}`
    const qty = Math.max(1, parseCurrencyOrNumber(getCol("quantity")) || 1)
    const unitPrice = parseCurrencyOrNumber(getCol("unit_price"))
    const subtotal = parseCurrencyOrNumber(getCol("subtotal")) || qty * unitPrice
    const totalAmount = parseCurrencyOrNumber(getCol("total_amount")) || subtotal
    const shippingFee = parseCurrencyOrNumber(getCol("shipping_fee"))
    const marketplaceFee = parseCurrencyOrNumber(getCol("marketplace_fee"))
    const customerName = getCol("customer_name") || "Pelanggan Marketplace"
    const customerPhone = getCol("customer_phone")
    const shippingAddress = getCol("shipping_address")
    const courier = getCol("courier") || "Standard Expedited"
    const trackingNumber = getCol("tracking_number")
    const orderDate = getCol("order_date") || new Date().toISOString()

    unmappedSkuSet.add(rawSku)

    const item: ImportMarketplaceOrderItemPayload = {
      external_sku: rawSku,
      item_name: rawItemName,
      quantity: qty,
      unit_price: unitPrice,
      subtotal: subtotal,
    }

    if (!orderMap.has(orderId)) {
      const valErrors: string[] = []
      if (!orderId || orderId.startsWith("ROW-ORD-")) {
        valErrors.push("Nomor pesanan kosong pada baris " + i)
      }
      if (qty <= 0) {
        valErrors.push("Kuantitas produk tidak valid")
      }

      orderMap.set(orderId, {
        orderData: {
          external_order_id: orderId,
          order_date: orderDate,
          customer_name: customerName,
          customer_phone: customerPhone,
          shipping_address: shippingAddress,
          courier: courier,
          tracking_number: trackingNumber,
          total_amount: totalAmount,
          shipping_fee: shippingFee,
          marketplace_fee: marketplaceFee,
          net_amount: totalAmount + shippingFee - marketplaceFee,
        },
        items: [item],
        validationErrors: valErrors,
      })
    } else {
      const existing = orderMap.get(orderId)!
      existing.items.push(item)
      // Recalculate totals if subtotal was additive
      const sumItems = existing.items.reduce((acc, it) => acc + (Number(it.subtotal) || 0), 0)
      if (sumItems > Number(existing.orderData.total_amount || 0)) {
        existing.orderData.total_amount = sumItems
        existing.orderData.net_amount =
          sumItems +
          Number(existing.orderData.shipping_fee || 0) -
          Number(existing.orderData.marketplace_fee || 0)
      }
    }
  }

  const orders: ParsedOrderPreview[] = []
  let totalRevenue = 0
  let validOrderCount = 0
  let errorOrderCount = 0

  orderMap.forEach(({ orderData, items, validationErrors }, orderId) => {
    const isValid = validationErrors.length === 0
    if (isValid) {
      validOrderCount++
      totalRevenue += Number(orderData.net_amount || orderData.total_amount || 0)
    } else {
      errorOrderCount++
    }

    orders.push({
      external_order_id: orderId,
      order_date: orderData.order_date || new Date().toISOString(),
      customer_name: orderData.customer_name,
      customer_phone: orderData.customer_phone,
      shipping_address: orderData.shipping_address,
      courier: orderData.courier,
      tracking_number: orderData.tracking_number,
      total_amount: orderData.total_amount || 0,
      shipping_fee: orderData.shipping_fee || 0,
      marketplace_fee: orderData.marketplace_fee || 0,
      net_amount: orderData.net_amount || orderData.total_amount || 0,
      items,
      isValid,
      validationErrors,
      detectedChannel,
    })
  })

  return {
    detectedChannel,
    headers: rawHeaders,
    orders,
    rawRowCount,
    validOrderCount,
    errorOrderCount,
    totalRevenue,
    unmappedSkuSet: Array.from(unmappedSkuSet),
    errors,
  }
}

/** Pre-packaged demo CSV templates for instant 1-click user testing */
export const SAMPLE_CSV_TEMPLATES = {
  SHOPEE: `No. Pesanan,Nomor Referensi SKU,Nama Produk,Jumlah,Harga Awal,Total Harga Produk,Total Pembayaran,Ongkos Kirim Dibayar Pembeli,Biaya Layanan,Nama Pembeli,No. Telepon,Alamat Pengiriman,Opsi Pengiriman,No. Resi
240909SHP-SAMPLE-01,KOPISUSU-BOTOL-250ML,Kopi Susu Gula Aren 250ml Botol,3,25000,75000,85000,10000,2500,Ahmad Fauzi,081234567891,Jl. Kemang Raya No. 14 Jakarta Selatan,J&T Express,JT9928172635
240909SHP-SAMPLE-02,MINYAK-GORENG-2L,Minyak Goreng Sawit Pouch 2L,2,30000,60000,69000,9000,1800,Rina Marlina,085712348899,Jl. Tebet Barat Dalam No. 8 Jakarta Selatan,Shopee Xpress,SPXID902188231
240909SHP-SAMPLE-03,KOPISUSU-BOTOL-250ML,Kopi Susu Gula Aren 250ml Botol,1,25000,25000,32000,7000,1000,Doni Prasetyo,081399887766,Jl. Senopati No. 22 Jakarta Selatan,SiCepat REG,004182938472`,

  TOKOPEDIA: `Nomor Pesanan,Nomor Referensi SKU,Nama Produk,Jumlah,Harga Satuan,Total Pembayaran,Kurir,No Resi,Nama Pembeli
TKP-DEMO-2026-8801,KOPISUSU-DUS-24,Kopi Susu Dus (Karton isi 24 pcs),2,120000,255000,SiCepat REG,004099887766,Dewi Kusuma
TKP-DEMO-2026-8802,BERAS-PREM-5KG,Beras Premium Pulen Karung 5kg,3,80000,260000,Anteraja Regular,100029384756,Hendra Wijaya
TKP-DEMO-2026-8803,NEW-UNMAPPED-GULA-AREN,Gula Aren Bubuk Organik 500gr,5,28000,140000,JNE REG,JNE8899001122,Lestari Putri`,

  TIKTOK: `Order ID,Seller SKU,Product Name,Quantity,Unit Price,Total Amount,Customer Name,Phone,Courier,Tracking Number
TT-LIVE-908123,TT-VIRAL-MATCHA-LATTE,Matcha Latte Creamy 250ml (Viral TikTok),4,42500,170000,Siska Amanda,081299881122,J&T Cargo,JTC8899112233
TT-LIVE-908124,GULA-PASIR-1KG,Gula Pasir Kristal Putih 1kg,5,17500,87500,Bambang Pamungkas,087811223344,Ninja Van,NVN9018273645
TT-LIVE-908125,NEW-TIKTOK-HONEY-LEMON,Minuman Madu Lemon Dingin 300ml,2,35000,70000,Indah Permata,085611223344,J&T Express,JT8819203948`,

  LAZADA: `orderNumber,sellerSku,itemName,quantity,unitPrice,total,customer,courier,tracking
LAZ-DEMO-991201,BERAS-PREM-5KG,Beras Putih Premium 5kg,2,80000,160000,Rudy Gunawan,Ninja Van,NVN7788990011
LAZ-DEMO-991202,MINYAK-GORENG-2L,Minyak Goreng Sawit 2L Refill,4,30000,120000,Surya Saputra,J&T Express,JT5566778899
LAZ-DEMO-991203,KOPISUSU-BOTOL-250ML,Kopi Susu Gula Aren 250ml,10,25000,250000,Dian Sastrowardoyo,Lazada Express,LEX8829103948`,
}
