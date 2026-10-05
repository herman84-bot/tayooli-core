import { ProposedAction } from "@/hooks/useCopilot"

function parseAmount(message: string): number {
  const lower = message.toLowerCase()

  // 1. Multipliers: "5 juta", "2.5 jt", "500 ribu", "500rb", "10jt", "1 miliar"
  const multMatch = lower.match(/(?:rp\.?\s*)?(\d+(?:[.,]\d+)?)\s*(miliar|milyar|juta|jt|ribu|rb)\b/i)
  if (multMatch) {
    const numStr = multMatch[1].replace(",", ".")
    const val = parseFloat(numStr)
    if (!isNaN(val)) {
      const unit = multMatch[2].toLowerCase()
      if (unit === "miliar" || unit === "milyar") return val * 1_000_000_000
      if (unit === "juta" || unit === "jt") return val * 1_000_000
      if (unit === "ribu" || unit === "rb") return val * 1_000
    }
  }

  // 2. Formatted numbers: "5.000.000", "5,000,000", "5000000"
  let search = message
  const keywords = ["sebesar", "sejumlah", "rp", "nominal", "total", "nilai", "senilai", "harga"]
  for (const kw of keywords) {
    const idx = lower.indexOf(kw)
    if (idx >= 0) {
      search = message.slice(idx)
      break
    }
  }

  const currMatch = search.match(/(?:rp\.?\s*)?(\d{1,3}(?:\.\d{3})+(?:,\d+)?|\d{1,3}(?:,\d{3})+(?:\.\d+)?|\d{4,})/i)
  if (currMatch) {
    let numStr = currMatch[1]
    if (numStr.includes(".") && numStr.includes(",")) {
      if (numStr.lastIndexOf(",") > numStr.lastIndexOf(".")) {
        numStr = numStr.replace(/\./g, "").replace(",", ".")
      } else {
        numStr = numStr.replace(/,/g, "")
      }
    } else if (numStr.includes(".")) {
      numStr = numStr.replace(/\./g, "")
    } else if (numStr.includes(",")) {
      numStr = numStr.replace(/,/g, "")
    }
    const val = parseFloat(numStr)
    if (!isNaN(val) && val > 0) {
      return val
    }
  }

  return 0
}

function extractInvoiceVendor(message: string): string {
  const lower = message.toLowerCase()
  const prefixes = [
    "untuk vendor ", "dari vendor ", "kepada vendor ", "ke vendor ", "pada vendor ",
    "buatkan invoice untuk ", "buat invoice untuk ", "buatkan tagihan untuk ", "buat tagihan untuk ",
    "catat invoice untuk ", "catat tagihan untuk ",
    "invoice untuk vendor ", "tagihan untuk vendor ", "faktur untuk vendor ",
    "invoice untuk ", "tagihan untuk ", "faktur untuk ",
    "invoice vendor ", "tagihan vendor ", "faktur vendor ",
    "vendor ", "pemasok ", "supplier ",
    "untuk ", "dari ", "kepada ",
  ]

  let rest = ""
  for (const p of prefixes) {
    const idx = lower.indexOf(p)
    if (idx !== -1) {
      const candidate = message.slice(idx + p.length).trim()
      if (candidate) {
        rest = candidate
        break
      }
    }
  }

  if (!rest) return ""

  const stopWords = [
    " sebesar", " sejumlah", " senilai", " nominal", " harga", " total",
    " dengan", " di ", " nilai", " rp.", " rp", " seharga",
    " tanggal", " jatuh tempo", " email", " telepon", " telp", " phone", " hp",
    ",", ";",
  ]

  let best = rest.length
  const lowerRest = rest.toLowerCase()
  for (const sw of stopWords) {
    const i = lowerRest.indexOf(sw)
    if (i >= 0 && i < best) {
      best = i
    }
  }

  return rest.slice(0, best).replace(/^[\s,;:\-'"\t]+|[\s,;:\-'"\t]+$/g, "")
}

function extractMasterDataName(message: string, prefixes: string[]): string {
  const lower = message.toLowerCase()
  const sorted = [...prefixes].sort((a, b) => b.length - a.length)
  let rest = ""
  for (const p of sorted) {
    const idx = lower.indexOf(p.toLowerCase())
    if (idx !== -1) {
      const candidate = message.slice(idx + p.length).trim()
      if (candidate) {
        rest = candidate
        break
      }
    }
  }

  if (!rest) return ""

  const separators = [
    " sebesar", " sejumlah", " senilai", " nominal", " harga", " total",
    " dengan", " di ", " nilai", " rp.", " rp",
    " email", " telepon", " phone", " hp", " telp",
    " alamat", " beralamat", " lokasi",
    " npwp", " bank", " rekening", " sku", ",",
  ]

  let best = rest.length
  const lowerRest = rest.toLowerCase()
  for (const sep of separators) {
    const i = lowerRest.indexOf(sep)
    if (i >= 0 && i < best) {
      best = i
    }
  }

  return rest.slice(0, best).replace(/^[\s,;:\-'"\t]+|[\s,;:\-'"\t]+$/g, "")
}

function extractAddress(message: string): string {
  const m = message.match(/(?:\balamat\b|\bberalamat\b|\blokasi\b|\baddress\b)\s*:?\s*(.+)/i)
  if (!m || !m[1]) return ""
  let rest = m[1].trim()
  // Buang token pembatas lain jika ada
  const cut = rest.search(/\b(email|telepon|telp|phone|hp|npwp|bank|rekening)\s*:/i)
  if (cut >= 0) rest = rest.slice(0, cut)
  // Buang email & phone yang ikut terseret ke dalam alamat
  rest = rest.replace(/[\w.+-]+@[\w.-]+\.\w+/g, " ")
  rest = rest.replace(/(?:\+62|62|0)\d[\d\-\s]{7,14}\d/g, " ")
  rest = rest.replace(/\b(e-?mail|telepon|telp|phone|hp)\s*:?\s*/gi, " ")
  rest = rest.replace(/\s{2,}/g, " ").trim()
  rest = rest.replace(/^[\s,;:\-'"\t]+|[\s,;:\-'"\t]+$/g, "")
  return rest
}

const ENTITY_MARKER_RE = /\b(PT|CV|UD|TB|PD|FA|Toko|Koperasi|Yayasan)\b\.?/i

function hasNameIntroducer(message: string): boolean {
  return /\bbernama\b|\bdengan nama\b|\bnama\s*:|\bnama\s+\S|\byaitu\b/i.test(message)
}

function hasCompanyIntroducer(message: string): boolean {
  const lower = message.toLowerCase()
  return (
    /\bjadi\b/.test(lower) ||
    /\bmenjadi\b/.test(lower) ||
    /\byaitu\b/.test(lower) ||
    message.includes('"') ||
    /\bnama\s*:/i.test(message)
  )
}

function hasEmailOrPhone(message: string): boolean {
  return (
    /[\w.+-]+@[\w.-]+\.\w+/.test(message) ||
    /(?:\+62|62|0)\d[\d\-\s]{7,14}\d/.test(message)
  )
}

const REJECTED_NAME_SET = new Set(["baru", "perusahaan", "vendor", "customer", "produk"])

function cleanNameCandidate(name: string): string {
  return name.replace(/^[\s,;:\-'"\t]+|[\s,;:\-'"\t]+$/g, "").trim()
}

function isRejectedName(candidate: string): boolean {
  const t = cleanNameCandidate(candidate)
  if (!t || t.length < 2) return true
  const low = t.toLowerCase()
  if (REJECTED_NAME_SET.has(low)) return true
  if (/^(untuk|buat|buatkan|dengan|dari|ke|kepada|di)\b/i.test(t)) return true
  return false
}

function isGroundedName(candidate: string, message: string): boolean {
  if (isRejectedName(candidate)) return false
  if (hasNameIntroducer(message) || ENTITY_MARKER_RE.test(message) || hasEmailOrPhone(message)) {
    return true
  }
  const words = candidate.split(/\s+/).filter(Boolean)
  if (words.length >= 2) {
    const firstWord = words[0].toLowerCase()
    if (!REJECTED_NAME_SET.has(firstWord) && !/^(untuk|buat|buatkan|dengan|dari|ke|kepada|di)\b/i.test(firstWord)) {
      return true
    }
  } else if (words.length === 1 && words[0].length >= 3 && /^[A-Z]/.test(words[0])) {
    return !REJECTED_NAME_SET.has(words[0].toLowerCase())
  }
  return false
}

function isGroundedCompanyName(candidate: string, message: string): boolean {
  if (isRejectedName(candidate)) return false
  return hasCompanyIntroducer(message) || ENTITY_MARKER_RE.test(message)
}

const REJECTED_PRODUCT_SET = new Set([
  "baru", "produk", "barang", "item", "katalog", "sku",
  "vendor", "customer", "perusahaan", "pelanggan",
])

function isGroundedProductName(candidate: string, message: string, price: number): boolean {
  const t = cleanNameCandidate(candidate)
  if (!t || t.length < 2) return false
  if (REJECTED_PRODUCT_SET.has(t.toLowerCase())) return false
  if (/^(untuk|buat|buatkan|dengan|dari|ke|kepada|di)\b/i.test(t)) return false
  if (hasNameIntroducer(message)) return true
  if (/\b(sku|kode)\b/i.test(message)) return true
  if (price > 0) return true
  if (/[A-Z0-9]/.test(t) && t.length >= 4) return true
  return false
}

function stripAddressFromName(name: string, address: string): string {
  let out = name
  if (address) {
    const idx = out.toLowerCase().indexOf(address.toLowerCase())
    if (idx >= 0) out = (out.slice(0, idx) + out.slice(idx + address.length)).trim()
  }
  const cut = out.search(/\b(alamat|beralamat|lokasi|address)\b/i)
  if (cut >= 0) out = out.slice(0, cut).trim()
  return out.replace(/^[\s,;:\-'"\t]+|[\s,;:\-'"\t]+$/g, "")
}

function escapeRegExp(str: string): string {
  return str.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")
}

function cleanVendorName(name: string, email: string, phone: string): string {
  let cleaned = name
  if (email) cleaned = cleaned.replace(new RegExp(escapeRegExp(email), "gi"), "")
  if (phone) cleaned = cleaned.replace(new RegExp(escapeRegExp(phone), "gi"), "")

  const strips = [
    "email:", "email", "e-mail:", "e-mail",
    "telepon:", "telepon", "telp:", "telp", "phone:", "phone", "hp:", "hp",
    "no.", "nomor", "sebesar", "sejumlah", "rp.", "rp", "senilai", "nominal",
    "dengan", "dan",
  ]

  for (const s of strips) {
    const regex = new RegExp(`\\b${s}\\b`, "gi")
    cleaned = cleaned.replace(regex, "")
  }

  cleaned = cleaned.replace(/^[\s,;:\-'"\t]+|[\s,;:\-'"\t]+$/g, "")
  cleaned = cleaned.replace(/^baru\s+/i, "")
  return cleaned.replace(/^[\s,;:\-'"\t]+|[\s,;:\-'"\t]+$/g, "")
}

function formatPrice(val: number): string {
  return Math.round(val).toLocaleString("id-ID")
}

/**
 * Deterministic Zero-AI Rule-Based Fallback Engine
 * Runs regex and keyword matching when Gemini/LLM is rate-limited (429), offline, or unavailable.
 */
export function matchRuleBasedAction(message: string): {
  reply: string
  actions: ProposedAction[]
} {
  const normalized = message.toLowerCase().trim()
  const actions: ProposedAction[] = []

  // 0. Conversational & Empathetic Greetings (ZenSpace principles)
  if (
    /^(halo|hallo|hai|hi|hey|hei|assalamu)/i.test(normalized) ||
    normalized.includes("selamat pagi") ||
    normalized.includes("selamat siang") ||
    normalized.includes("selamat sore") ||
    normalized.includes("selamat malam")
  ) {
    return {
      reply: "Halo! Senang bisa menyapa Anda. Ada yang bisa saya bantu untuk operasional toko dan gudang Anda hari ini? Anda bisa meminta saya mengelola profil bisnis, memeriksa katalog produk, memantau stok, hingga membuka kasir POS.",
      actions: [],
    }
  }

  if (normalized.includes("apa kabar") || normalized.includes("gimana kabarnya") || normalized.includes("bagaimana kabarmu")) {
    return {
      reply: "Kabar baik dan sistem siap membantu Anda kapan saja! Ada tugas atau pengaturan workspace yang ingin kita selesaikan hari ini?",
      actions: [],
    }
  }

  if (
    normalized.includes("siapa kamu") ||
    normalized.includes("kamu siapa") ||
    normalized.includes("bisa apa") ||
    normalized.includes("bisa ngapain") ||
    normalized.includes("bantuan apa")
  ) {
    return {
      reply: "Saya Tayooli Copilot, asisten yang siap membantu Anda mengelola operasional di Tayooli ERP. Anda cukup memerintahkan saya dengan bahasa sehari-hari, misalnya: 'ubah nama toko', 'cek stok barang', 'buka kasir POS', atau 'buka stock opname'.",
      actions: [],
    }
  }

  if (
    normalized.includes("terima kasih") ||
    normalized.includes("makasih") ||
    normalized.includes("thanks") ||
    normalized.includes("tengkyu")
  ) {
    return {
      reply: "Sama-sama! Senang bisa membantu kelancaran kerja Anda. Beri tahu saya jika ada hal lain yang ingin diselesaikan ya.",
      actions: [],
    }
  }

  // 1. Navigation evaluated FIRST
  const isNavigation =
    normalized.includes("buka") ||
    normalized.includes("lihat") ||
    normalized.includes("menu") ||
    normalized.includes("ke halaman") ||
    normalized.includes("tampilkan") ||
    normalized.includes("navigasi")

  if (isNavigation) {
    // Summary navigation / action
    if (normalized.includes("ringkasan") || normalized.includes("summary")) {
      actions.push({
        id: `rule-summary-${Date.now()}`,
        toolName: "get_dashboard_summary",
        title: "Ringkasan Dashboard Finansial",
        description: "Mengambil dan menampilkan ringkasan metrik keuangan dan operasional workspace.",
        riskLevel: "low",
        diff: [{ field: "modul", oldValue: "-", newValue: "Dashboard Finansial" }],
        payload: { metric: "overview", period: "current_month" },
        status: "pending",
      })
      return {
        reply: "Saya telah menyiapkan aksi untuk mengambil ringkasan dashboard dan metrik keuangan workspace Anda. Silakan jalankan aksi di bawah.",
        actions,
      }
    }

    let targetPath = "/dashboard"
    let modName = "Dashboard"

    if (normalized.includes("vendor") || normalized.includes("pemasok") || normalized.includes("supplier")) {
      targetPath = "/dashboard/vendors"
      modName = "Vendors"
    } else if (normalized.includes("customer") || normalized.includes("pelanggan") || normalized.includes("klien")) {
      targetPath = "/dashboard/customers"
      modName = "Customers"
    } else if (normalized.includes("invoice") || normalized.includes("tagihan") || normalized.includes("faktur")) {
      targetPath = "/dashboard/invoices"
      modName = "Invoices"
    } else if (normalized.includes("purchase order") || normalized.includes("purchase") || /\bpo\b/i.test(normalized)) {
      targetPath = "/dashboard/purchase-orders"
      modName = "Purchase Orders"
    } else if (normalized.includes("gr") || normalized.includes("goods receipt") || normalized.includes("penerimaan")) {
      targetPath = "/dashboard/goods-receipts"
      modName = "Goods Receipts"
    } else if (normalized.includes("gateway") || normalized.includes("pakasir") || normalized.includes("midtrans")) {
      targetPath = "/dashboard/payment-gateways"
      modName = "Payment Gateways"
    } else if (normalized.includes("pembayaran") || normalized.includes("payment order") || normalized.includes("bayar")) {
      targetPath = "/dashboard/payment-orders"
      modName = "Payment Orders"
    } else if (normalized.includes("coa") || normalized.includes("akun") || normalized.includes("chart of account")) {
      targetPath = "/accounting/chart-of-accounts"
      modName = "Chart of Accounts"
    } else if (normalized.includes("jurnal") || normalized.includes("journal")) {
      targetPath = "/accounting/journal-entries"
      modName = "Journal Entries"
    } else if (normalized.includes("pos") || normalized.includes("kasir") || normalized.includes("point of sale")) {
      targetPath = "/pos"
      modName = "Point of Sale"
    } else if (normalized.includes("surat jalan") || normalized.includes("delivery order")) {
      targetPath = "/wms/delivery-orders"
      modName = "Surat Jalan (DO)"
    } else if (normalized.includes("marketplace") || normalized.includes("omnichannel")) {
      targetPath = "/wms/marketplace"
      modName = "Marketplace Omnichannel"
    } else if (normalized.includes("transfer") || normalized.includes("mutasi")) {
      targetPath = "/wms/transfers"
      modName = "Stock Transfers"
    } else if (normalized.includes("opname")) {
      targetPath = "/wms/opname"
      modName = "Stock Opname"
    } else if (normalized.includes("scrap") || normalized.includes("rusak") || normalized.includes("afkir")) {
      targetPath = "/wms/scrap"
      modName = "Barang Rusak / Scrap"
    } else if (normalized.includes("scanner") || normalized.includes("scan")) {
      targetPath = "/wms/scanner"
      modName = "Barcode Scanner"
    } else if (normalized.includes("gudang") || normalized.includes("warehouse") || normalized.includes("wms")) {
      targetPath = "/wms"
      modName = "Warehouse & Stock"
    } else if (normalized.includes("produk") || normalized.includes("barang") || normalized.includes("katalog") || normalized.includes("item")) {
      targetPath = "/products"
      modName = "Katalog Produk"
    } else if (normalized.includes("setting") || normalized.includes("pengaturan")) {
      targetPath = "/settings"
      modName = "Pengaturan"
    }

    actions.push({
      id: `rule-nav-${Date.now()}`,
      toolName: "navigate_to_module",
      title: `Navigasi ke ${modName}`,
      description: `Membuka dan mengarahkan tampilan langsung ke modul ${modName}.`,
      riskLevel: "low",
      diff: [{ field: "rute", oldValue: "Halaman Saat Ini", newValue: targetPath }],
      payload: { path: targetPath, module_name: modName },
      status: "pending",
    })
    return {
      reply: `Saya menyiapkan navigasi cepat ke ${modName}. Klik jalankan untuk membuka halaman.`,
      actions,
    }
  }

  // 2. Transactions evaluated BEFORE Master Data
  const isInvoiceCreation =
    (normalized.includes("invoice") || normalized.includes("tagihan") || normalized.includes("faktur")) &&
    (normalized.includes("buat") ||
      normalized.includes("buatkan") ||
      normalized.includes("tambah") ||
      normalized.includes("tambahkan") ||
      normalized.includes("catat") ||
      normalized.includes("terbitkan") ||
      normalized.includes("bikin") ||
      normalized.includes("input") ||
      normalized.includes("sebesar") ||
      normalized.includes("sejumlah") ||
      normalized.includes("senilai") ||
      normalized.includes("nominal") ||
      normalized.includes("rp"))

  if (isInvoiceCreation) {
    const vendorName = extractInvoiceVendor(message)
    const amount = parseAmount(message)

    if (!vendorName || isRejectedName(vendorName)) {
      return {
        reply: "Untuk membuat purchase invoice, saya membutuhkan nama vendor rekanan yang jelas. Contoh: 'buatkan invoice untuk vendor PT Maju Jaya sebesar 5 juta rupiah'. Vendor mana yang ingin dibuatkan invoice?",
        actions: [],
      }
    }

    const diff = [{ field: "vendor_name", oldValue: "-", newValue: vendorName }]
    let desc = `Membuat draf purchase invoice untuk vendor ${vendorName}.`
    if (amount > 0) {
      diff.push({ field: "amount", oldValue: "-", newValue: `Rp ${formatPrice(amount)}` })
      desc = `Membuat draf purchase invoice untuk vendor ${vendorName} senilai Rp ${formatPrice(amount)}.`
    }

    actions.push({
      id: `rule-invoice-${Date.now()}`,
      toolName: "create_purchase_invoice",
      title: "Pembuatan Purchase Invoice Baru",
      description: desc,
      riskLevel: "medium",
      diff,
      payload: {
        vendor_name: vendorName,
        amount,
        currency: "IDR",
      },
      status: "pending",
    })

    const reply =
      amount > 0
        ? `Proposal pembuatan draf purchase invoice untuk vendor '${vendorName}' senilai Rp ${formatPrice(amount)} telah disiapkan. Silakan tinjau dan jalankan aksi berikut.`
        : `Proposal pembuatan draf purchase invoice untuk vendor '${vendorName}' telah disiapkan. Silakan tinjau dan jalankan aksi berikut.`

    return { reply, actions }
  }

  const isSummaryRequest =
    normalized.includes("ringkasan") ||
    normalized.includes("summary") ||
    normalized.includes("total tagihan") ||
    normalized.includes("rekapitulasi")

  if (isSummaryRequest) {
    actions.push({
      id: `rule-summary-${Date.now()}`,
      toolName: "get_dashboard_summary",
      title: "Ringkasan Dashboard Finansial",
      description: "Mengambil dan menampilkan ringkasan metrik keuangan dan operasional workspace.",
      riskLevel: "low",
      diff: [{ field: "modul", oldValue: "-", newValue: "Dashboard Finansial" }],
      payload: { metric: "overview", period: "current_month" },
      status: "pending",
    })
    return {
      reply: "Saya telah menyiapkan aksi untuk mengambil ringkasan dashboard dan metrik keuangan workspace Anda. Silakan jalankan aksi di bawah.",
      actions,
    }
  }

  // 3. Settings: Company Profile, Team Member, Payment Gateway
  if (
    normalized.includes("ubah nama") ||
    normalized.includes("ganti nama pt") ||
    normalized.includes("nama perusahaan") ||
    normalized.includes("alamat pt") ||
    normalized.includes("npwp")
  ) {
    const rawCompanyName = extractMasterDataName(message, [
      "nama perusahaan jadi ", "nama perusahaan menjadi ", "nama perusahaan ",
      "ubah nama jadi ", "ubah nama menjadi ", "ubah nama ",
      "ganti nama pt ", "nama pt ",
    ])
    const companyName = cleanVendorName(rawCompanyName, "", "")

    if (!isGroundedCompanyName(companyName, message)) {
      return {
        reply: "Untuk memperbarui profil perusahaan, saya membutuhkan nama legal baru perusahaan yang jelas. Contoh: 'ubah nama perusahaan jadi PT Sukses Sejahtera'. Nama perusahaan baru apa yang ingin digunakan?",
        actions: [],
      }
    }

    actions.push({
      id: `rule-profile-${Date.now()}`,
      toolName: "update_company_profile",
      title: "Pembaruan Profil Perusahaan",
      description: `Memperbarui nama profil perusahaan menjadi '${companyName}'.`,
      riskLevel: "high",
      requiresConfirmationText: "KONFIRMASI",
      diff: [{ field: "company_name", oldValue: "(Profil Saat Ini)", newValue: companyName }],
      payload: { company_name: companyName },
      status: "pending",
    })
    return {
      reply: `Saya mendeteksi permintaan pengaturan profil perusahaan. Nama baru: '${companyName}'. Karena ini perubahan profil berisiko tinggi, ketik KONFIRMASI pada kartu aksi untuk melanjutkan.`,
      actions,
    }
  }

  if (
    normalized.includes("undang") ||
    normalized.includes("tambah anggota") ||
    normalized.includes("tambah tim") ||
    normalized.includes("tambah user") ||
    normalized.includes("invite")
  ) {
    const emailMatch = normalized.match(/[\w.-]+@[\w.-]+\.\w+/)
    const email = emailMatch ? emailMatch[0] : ""
    const role = normalized.includes("accountant")
      ? "accountant"
      : normalized.includes("approver")
      ? "approver"
      : normalized.includes("admin")
      ? "admin"
      : "member"

    if (!email) {
      return {
        reply: "Untuk mengundang anggota tim baru, mohon sertakan alamat email yang valid (contoh: 'undang anggota tim email budi@test.com sebagai finance').",
        actions: [],
      }
    }

    actions.push({
      id: `rule-team-${Date.now()}`,
      toolName: "invite_team_member",
      title: "Undang Anggota Tim",
      description: `Mengundang ${email} sebagai ${role}.`,
      riskLevel: "high",
      requiresConfirmationText: "KONFIRMASI",
      diff: [
        { field: "email", oldValue: "-", newValue: email },
        { field: "role", oldValue: "-", newValue: role },
      ],
      payload: { email, role },
      status: "pending",
    })
    return {
      reply: `Undangan untuk ${email} (peran: ${role}) telah disiapkan. Karena ini perubahan hak akses workspace, ketik KONFIRMASI untuk melanjutkan.`,
      actions,
    }
  }

  if (
    normalized.includes("gateway") ||
    normalized.includes("pakasir") ||
    normalized.includes("midtrans") ||
    normalized.includes("setup payment gateway")
  ) {
    if (!normalized.includes("midtrans") && !normalized.includes("pakasir")) {
      return {
        reply: "Mohon tentukan penyedia payment gateway yang ingin dikonfigurasi (pilihan: Pakasir atau Midtrans). Contoh: 'setup payment gateway midtrans'.",
        actions: [],
      }
    }
    const provider = normalized.includes("midtrans") ? "midtrans" : "pakasir"
    actions.push({
      id: `rule-gateway-${Date.now()}`,
      toolName: "configure_payment_gateway",
      title: `Konfigurasi Gateway: ${provider.toUpperCase()}`,
      description: `Mengatur penyedia ${provider} sebagai gateway pembayaran aktif.`,
      riskLevel: "high",
      requiresConfirmationText: "KONFIRMASI",
      diff: [
        { field: "provider", oldValue: "-", newValue: provider },
        { field: "is_active", oldValue: "false", newValue: "true" },
      ],
      payload: { provider, is_active: true },
      status: "pending",
    })
    return {
      reply: `Saya telah menyiapkan proposal konfigurasi ${provider.toUpperCase()}. Karena ini pengaturan finansial berisiko tinggi, konfirmasi teks wajib dilakukan.`,
      actions,
    }
  }

  // 4. Master Data: Vendor, Customer, Product
  // Never trigger master data creation if transactional words are present.
  const hasTransactionalWords =
    normalized.includes("invoice") ||
    normalized.includes("tagihan") ||
    normalized.includes("faktur") ||
    /\bpo\b/i.test(normalized) ||
    normalized.includes("purchase order") ||
    normalized.includes("bayar") ||
    normalized.includes("pembayaran")

  const hasInterrogativeWords = /^(vendor|customer|pelanggan|pemasok|supplier)\s+(apa|siapa|mana|kenapa|mengapa|bagaimana|berapa)\b/i.test(normalized)

  const isVendorCreation =
    !hasTransactionalWords &&
    !hasInterrogativeWords &&
    (normalized.includes("tambah vendor") ||
      normalized.includes("tambahkan vendor") ||
      normalized.includes("daftar vendor") ||
      normalized.includes("daftarkan vendor") ||
      normalized.includes("registrasi vendor") ||
      normalized.includes("vendor baru") ||
      normalized.includes("buat vendor") ||
      normalized.includes("buatkan vendor") ||
      normalized.includes("rekanan baru") ||
      normalized.includes("mitra baru") ||
      normalized.includes("tambah pemasok") ||
      normalized.includes("pemasok baru") ||
      normalized.includes("daftar pemasok") ||
      normalized.includes("daftarkan pemasok") ||
      normalized.includes("tambah supplier") ||
      normalized.includes("supplier baru") ||
      normalized.includes("daftar supplier") ||
      normalized.startsWith("vendor baru") ||
      normalized.startsWith("vendor "))

  if (isVendorCreation) {
    const emailMatch = normalized.match(/[\w.-]+@[\w.-]+\.\w+/)
    const email = emailMatch ? emailMatch[0] : ""
    const phoneMatch = normalized.match(/(?:\+62|62|0)\d[\d\-\s]{7,14}\d/)
    const phone = phoneMatch ? phoneMatch[0].replace(/[\s-]/g, "") : ""
    const address = extractAddress(message)

    let vendorName = extractMasterDataName(message, [
      "vendor baru bernama ", "vendor bernama ", "pemasok baru bernama ", "pemasok bernama ",
      "supplier baru bernama ", "supplier bernama ",
      "tambah vendor baru dengan nama ", "tambah vendor dengan nama ",
      "vendor baru dengan nama ", "vendor dengan nama ",
      "dengan nama ", "bernama ", "nama: ", "nama ",
      "tambah vendor baru bernama ", "tambah vendor baru ", "tambah pemasok baru ", "tambah supplier baru ",
      "daftarkan vendor baru ", "daftar vendor baru ", "buat vendor baru ", "buatkan vendor baru ",
      "vendor baru ", "pemasok baru ", "supplier baru ",
      "tambah vendor ", "daftarkan vendor ", "daftar vendor ", "buat vendor ", "buatkan vendor ",
      "vendor ", "pemasok ", "supplier ",
    ])
    vendorName = stripAddressFromName(vendorName, address)
    vendorName = cleanVendorName(vendorName, email, phone)
    vendorName = cleanNameCandidate(vendorName.replace(/^(dengan\s+nama|bernama|nama\s*:?|yaitu)\b\s*/i, ""))

    if (!isGroundedName(vendorName, message)) {
      return {
        reply: "Saya belum menemukan nama vendor pada pesan Anda. Mohon sebutkan nama vendor yang ingin didaftarkan (contoh: 'tambah vendor baru bernama PT Sinar Terang Sejati').",
        actions: [],
      }
    }

    const diff = [{ field: "name", oldValue: "-", newValue: vendorName }]
    const payload: Record<string, string> = { name: vendorName }
    if (email) {
      diff.push({ field: "email", oldValue: "-", newValue: email })
      payload.email = email
    }
    if (phone) {
      diff.push({ field: "phone", oldValue: "-", newValue: phone })
      payload.phone = phone
    }
    if (address) {
      diff.push({ field: "address", oldValue: "-", newValue: address })
      payload.address = address
    }

    actions.push({
      id: `rule-vendor-${Date.now()}`,
      toolName: "create_vendor",
      title: "Pendaftaran Rekanan Vendor Baru",
      description: `Mendaftarkan mitra vendor baru '${vendorName}' ke dalam direktori rekanan bisnis.`,
      riskLevel: "medium",
      diff,
      payload,
      status: "pending",
    })
    return {
      reply: `Proposal penambahan vendor '${vendorName}' disiapkan. Tinjau detail pada kartu aksi di bawah.`,
      actions,
    }
  }

  const isCustomerCreation =
    !hasTransactionalWords &&
    !hasInterrogativeWords &&
    (normalized.includes("tambah customer") ||
      normalized.includes("tambahkan customer") ||
      normalized.includes("customer baru") ||
      normalized.includes("daftar customer") ||
      normalized.includes("daftarkan customer") ||
      normalized.includes("buat customer") ||
      normalized.includes("buatkan customer") ||
      normalized.includes("tambah pelanggan") ||
      normalized.includes("tambahkan pelanggan") ||
      normalized.includes("pelanggan baru") ||
      normalized.includes("daftar pelanggan") ||
      normalized.includes("daftarkan pelanggan") ||
      normalized.includes("buat pelanggan") ||
      normalized.includes("tambah klien") ||
      normalized.includes("klien baru") ||
      normalized.includes("daftar klien") ||
      normalized.includes("buat klien") ||
      normalized.startsWith("customer baru") ||
      normalized.startsWith("customer "))

  if (isCustomerCreation) {
    const emailMatch = normalized.match(/[\w.-]+@[\w.-]+\.\w+/)
    const email = emailMatch ? emailMatch[0] : ""
    const phoneMatch = normalized.match(/(?:\+62|62|0)\d[\d\-\s]{7,14}\d/)
    const phone = phoneMatch ? phoneMatch[0].replace(/[\s-]/g, "") : ""
    const address = extractAddress(message)

    let custName = extractMasterDataName(message, [
      "customer baru bernama ", "customer bernama ", "pelanggan baru bernama ", "pelanggan bernama ",
      "klien baru bernama ", "klien bernama ",
      "tambah customer baru dengan nama ", "tambah customer dengan nama ",
      "customer baru dengan nama ", "customer dengan nama ",
      "dengan nama ", "bernama ", "nama: ", "nama ",
      "tambah customer baru bernama ", "tambah customer baru ", "tambah pelanggan baru ", "tambah klien baru ",
      "daftarkan customer baru ", "daftar customer baru ", "buat customer baru ", "buatkan customer baru ",
      "customer baru ", "pelanggan baru ", "klien baru ",
      "tambah customer ", "daftarkan customer ", "daftar customer ", "tambah pelanggan ", "daftar pelanggan ",
      "buat customer ", "buatkan customer ",
      "customer ", "pelanggan ", "klien ",
    ])
    custName = stripAddressFromName(custName, address)
    custName = cleanVendorName(custName, email, phone)
    custName = cleanNameCandidate(custName.replace(/^(dengan\s+nama|bernama|nama\s*:?|yaitu)\b\s*/i, ""))

    if (!isGroundedName(custName, message)) {
      return {
        reply: "Saya belum menemukan nama pelanggan pada pesan Anda. Mohon sebutkan nama pelanggan yang ingin didaftarkan (contoh: 'tambah customer baru bernama PT Mitra Sejahtera').",
        actions: [],
      }
    }

    const diff = [{ field: "name", oldValue: "-", newValue: custName }]
    const payload: Record<string, string> = { name: custName }
    if (email) {
      diff.push({ field: "email", oldValue: "-", newValue: email })
      payload.email = email
    }
    if (phone) {
      diff.push({ field: "phone", oldValue: "-", newValue: phone })
      payload.phone = phone
    }
    if (address) {
      diff.push({ field: "address", oldValue: "-", newValue: address })
      payload.address = address
    }

    actions.push({
      id: `rule-cust-${Date.now()}`,
      toolName: "create_customer",
      title: "Pendaftaran Pelanggan Baru",
      description: `Mendaftarkan pelanggan baru '${custName}' ke dalam sistem CRM dan penjualan.`,
      riskLevel: "medium",
      diff,
      payload,
      status: "pending",
    })
    return {
      reply: `Proposal penambahan pelanggan '${custName}' disiapkan. Tinjau detail pada kartu aksi di bawah.`,
      actions,
    }
  }

  const isProductCreation =
    normalized.includes("tambah produk") ||
    normalized.includes("tambahkan produk") ||
    normalized.includes("produk baru") ||
    normalized.includes("buat produk") ||
    normalized.includes("buatkan produk") ||
    normalized.includes("tambah barang") ||
    normalized.includes("tambahkan barang") ||
    normalized.includes("barang baru") ||
    normalized.includes("buat barang") ||
    normalized.includes("daftarkan barang") ||
    normalized.includes("daftar barang") ||
    normalized.includes("tambah item") ||
    normalized.includes("item baru") ||
    normalized.includes("buat item") ||
    normalized.includes("tambah sku") ||
    normalized.includes("tambahkan sku") ||
    normalized.includes("sku baru")

  if (isProductCreation) {
    const skuMatch = message.match(/(?:sku|kode)\s+([A-Za-z0-9\-_]+)/i)
    const sku = skuMatch ? skuMatch[1].toUpperCase() : ""
    const price = parseAmount(message)
    let prodName = extractMasterDataName(message, [
      "produk baru bernama ", "produk bernama ", "barang baru bernama ", "barang bernama ",
      "tambah produk baru bernama ", "tambah produk baru ", "tambah barang baru ", "tambah item baru ",
      "daftarkan produk baru ", "daftar produk baru ", "buat produk baru ", "buatkan produk baru ",
      "produk baru ", "barang baru ", "item baru ",
      "tambah produk ", "tambah barang ", "tambah item ", "tambah sku ", "tambahkan sku ",
      "produk ", "barang ", "item ",
    ])
    if (skuMatch) {
      prodName = prodName.replace(new RegExp(skuMatch[0], "gi"), "")
    }
    prodName = cleanNameCandidate(prodName)
    prodName = cleanNameCandidate(prodName.replace(/^(dengan\s+nama|bernama|nama\s*:?|yaitu)\b\s*/i, ""))

    if (!isGroundedProductName(prodName, message, price)) {
      return {
        reply: "Untuk menambahkan produk baru, mohon sertakan nama dan SKU atau harga produk. Contoh: 'buat produk baru bernama Laptop Asus kode ASUS-01 harga 15.000.000'.",
        actions: [],
      }
    }

    const finalSku = sku || `PRD-${Date.now() % 10000}`
    const diff = [
      { field: "sku", oldValue: "-", newValue: finalSku },
      { field: "name", oldValue: "-", newValue: prodName },
    ]
    if (price > 0) {
      diff.push({ field: "price", oldValue: "-", newValue: `Rp ${formatPrice(price)}` })
    }

    actions.push({
      id: `rule-prod-${Date.now()}`,
      toolName: "create_product",
      title: "Penambahan Produk Baru",
      description: `Membuat SKU produk baru '${prodName}' (${finalSku}) di katalog barang.`,
      riskLevel: "medium",
      diff,
      payload: { sku: finalSku, name: prodName, price },
      status: "pending",
    })
    return {
      reply: `Proposal pembuatan produk '${prodName}' (SKU: ${finalSku}) telah disiapkan. Tinjau detail pada kartu aksi di bawah.`,
      actions,
    }
  }

  // Default fallback guidance
  return {
    reply: "Saya siap membantu Anda di workspace ini. Anda dapat meminta saya untuk mengelola profil toko, mengundang rekan tim, memeriksa stok gudang, hingga membuka kasir POS. Apa yang ingin Anda kerjakan saat ini?",
    actions: [],
  }
}
