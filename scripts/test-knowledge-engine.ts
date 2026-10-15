import { getCustomerServiceReply, findBestTopic } from '../lib/chat/knowledge-engine'

interface TestCase {
  query: string
  expectedTopicId?: string
  mustContainText?: string
  mustContainAll?: string[]
  mustNotContainText?: string
  description: string
}

const testCases: TestCase[] = [
  // ── 1. USER CORE MANDATE: CAPABILITY & INTERROGATIVE QUERIES ──
  {
    query: 'apakah aplikasi ini bisa transfer gudang?',
    expectedTopicId: 'stock_transfers',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Mandat Pengguna: "apakah aplikasi ini bisa transfer gudang?" harus langsung jawab tegas "Ya, tentu saja bisa!" dan panduan transfer stok',
  },
  {
    query: 'bisa transfer barang antar cabang?',
    expectedTopicId: 'stock_transfers',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Variasi kapabilitas: "bisa transfer barang antar cabang?"',
  },
  {
    query: 'apakah bisa multi gudang?',
    expectedTopicId: 'warehouse_wms',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas multi-gudang: "apakah bisa multi gudang?"',
  },
  {
    query: 'apakah aplikasi ini bisa kasir?',
    expectedTopicId: 'pos_cashier',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas POS: "apakah aplikasi ini bisa kasir?"',
  },
  {
    query: 'bisa cetak struk kasir?',
    expectedTopicId: 'pos_receipt_print',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas cetak struk: "bisa cetak struk kasir?"',
  },
  {
    query: 'apakah bisa cetak surat jalan?',
    expectedTopicId: 'delivery_order',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas Surat Jalan DO: "apakah bisa cetak surat jalan?"',
  },
  {
    query: 'bisa catat barang masuk dari supplier?',
    expectedTopicId: 'inbound_workflow',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas Inbound: "bisa catat barang masuk dari supplier?"',
  },
  {
    query: 'apakah mendukung barcode scanner hp?',
    expectedTopicId: 'barcode_scanner',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas Barcode Scanner: "apakah mendukung barcode scanner hp?"',
  },
  {
    query: 'apakah ada fitur barang rusak?',
    expectedTopicId: 'scrap_damaged',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas Scrap / Afkir: "apakah ada fitur barang rusak?"',
  },
  {
    query: 'bisa integrasi shopee dan tokopedia?',
    expectedTopicId: 'marketplace_sync',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas Marketplace: "bisa integrasi shopee dan tokopedia?"',
  },
  {
    query: 'bisa stok opname?',
    expectedTopicId: 'stock_opname',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas Stock Opname: "bisa stok opname?"',
  },
  {
    query: 'bisa lihat omzet toko?',
    expectedTopicId: 'dashboard_analytics',
    mustContainText: 'Ya, tentu saja bisa!',
    description: 'Kapabilitas Dasbor: "bisa lihat omzet toko?"',
  },

  // ── 2. SALAM, SAPAAN & POLITE RESPONSE ──
  {
    query: 'Halo min selamat pagi',
    expectedTopicId: 'greetings',
    mustContainText: 'Halo kak! Selamat datang di layanan Customer Service Tayooli ERP',
    description: 'Salam pembuka ramah CS',
  },
  {
    query: 'terima kasih banyak atas bantuannya',
    expectedTopicId: 'gratitude',
    mustContainText: 'Sama-sama kak!',
    description: 'Ucapan terima kasih ramah CS',
  },
  {
    query: 'siapa kamu sebenarnya?',
    expectedTopicId: 'identity_about',
    mustContainText: 'Tayooli ERP',
    description: 'Identitas bot asisten resmi Tayooli',
  },

  // ── 3. DISAMBIGUASI PENCETAKAN & KATA AMBIGU ──
  {
    query: 'Bagaimana cara cetak surat jalan?',
    expectedTopicId: 'delivery_order',
    mustContainText: 'Surat Jalan',
    mustNotContainText: 'Cetak Struk Belanja Kasir',
    description: 'Disambiguasi: Cetak surat jalan harus ke Surat Jalan (DO), BUKAN ke Struk Kasir POS',
  },
  {
    query: 'Bagaimana cara cetak struk belanja di kasir?',
    expectedTopicId: 'pos_receipt_print',
    mustContainText: 'Cetak Struk Belanja Kasir',
    mustNotContainText: 'Surat Jalan (DO)',
    description: 'Disambiguasi: Cetak struk kasir harus ke Kasir POS, BUKAN Surat Jalan',
  },
  {
    query: 'gimana cara cetak?',
    mustContainAll: ['Struk Kasir (POS)', 'Surat Jalan (DO)'],
    description: 'Query ambigu "cara cetak" harus memunculkan klarifikasi ramah',
  },
  {
    query: 'gimana cara ngeprint?',
    mustContainText: 'Struk Kasir (POS)',
    description: 'Colloquial affix "ngeprint" harus terdeteksi disambiguasi struk vs surat jalan',
  },
  {
    query: 'alur barang di tayooli',
    mustContainText: 'Barang Masuk & Keluar',
    description: 'Query ambigu "alur barang" harus memunculkan pilihan alur',
  },

  // ── 4. PROSEDUR & TUTORIAL ──
  {
    query: 'Bagaimana alur transfer barang antar gudang cabang?',
    expectedTopicId: 'stock_transfers',
    mustContainText: 'Transfer Stok Antar Gudang',
    description: 'Panduan transfer stok antar cabang',
  },
  {
    query: 'bagaimana cara tambah produk baru dan isi SKU?',
    expectedTopicId: 'products_master',
    mustContainText: 'Master Produk & Katalog SKU',
    description: 'Master data produk dan SKU',
  },
  {
    query: 'bagaimana alur penerimaan barang masuk dari supplier?',
    expectedTopicId: 'inbound_workflow',
    mustContainText: 'Penerimaan Barang Masuk',
    description: 'Alur penerimaan barang masuk supplier (Inbound)',
  },
  {
    query: 'bagaimana cara ganti password dan atur tarif pajak ppn?',
    expectedTopicId: 'settings_account',
    mustContainText: 'Pengaturan Usaha, Pajak & Akun',
    description: 'Pengaturan akun, profil usaha, dan pajak PPN',
  },

  // ── 5. TROUBLESHOOTING & KENDALA TEKNIS ──
  {
    query: 'kamera scanner barcode saya tidak bisa dibuka di hp',
    expectedTopicId: 'troubleshooting_camera',
    mustContainText: 'Kamera Barcode Scanner Tidak Bisa Dibuka',
    description: 'Troubleshooting kamera scanner barcode',
  },
  {
    query: 'printer thermal bluetooth saya macet tidak merespons',
    expectedTopicId: 'troubleshooting_printer',
    mustContainText: 'Printer Thermal Struk Tidak Merespons',
    description: 'Troubleshooting printer thermal kasir',
  },
  {
    query: 'kenapa stok di kasir tidak berkurang setelah transaksi?',
    expectedTopicId: 'troubleshooting_stock',
    mustContainText: 'Stok Selisih atau Tidak Terpotong',
    description: 'Troubleshooting stok selisih/tidak terpotong',
  },
  {
    query: 'ada barang pecah dan kadaluwarsa mau dicatat afkir',
    expectedTopicId: 'scrap_damaged',
    mustContainText: 'Barang Rusak / Afkir / Scrap',
    description: 'Pencatatan barang rusak / scrap / write-off',
  },

  // ── 6. ENTERPRISE DECOUPLED MODULES (KO-3) ──
  {
    query: 'bagaimana cara buat chart of accounts dan entri jurnal debit kredit?',
    expectedTopicId: 'legacy_accounting_faq',
    mustContainText: '12 Modul Operasional Inti',
    description: 'Permintaan modul akuntansi lama (CoA/Jurnal) ditolak dengan edukasi 12 modul inti',
  },

  // ── 7. FALLBACK / DI LUAR DOMAIN ──
  {
    query: 'siapa pemenang piala dunia sepak bola 2022?',
    mustContainText: 'Mohon maaf, kami belum dapat memahami pertanyaan Anda secara spesifik',
    description: 'Pertanyaan di luar domain ERP menghasilkan smart fallback yang ramah',
  },
]

let passedCount = 0
let failedCount = 0

console.log('=== MENJALANKAN PENGUJIAN TAYOOLI KNOWLEDGE ENGINE ===\n')

for (const tc of testCases) {
  const reply = getCustomerServiceReply(tc.query)
  const matchResult = findBestTopic(tc.query)

  let passed = true
  const errors: string[] = []

  if (tc.expectedTopicId) {
    if (!matchResult || matchResult.topic.id !== tc.expectedTopicId) {
      passed = false
      errors.push(
        `Topic ID salah: diharapkan '${tc.expectedTopicId}', didapat '${matchResult ? matchResult.topic.id : 'null'}'`
      )
    }
  }

  if (tc.mustContainText && !reply.includes(tc.mustContainText)) {
    passed = false
    errors.push(`Balasan tidak mengandung teks wajib: '${tc.mustContainText}'`)
  }

  if (tc.mustContainAll) {
    for (const requiredText of tc.mustContainAll) {
      if (!reply.includes(requiredText)) {
        passed = false
        errors.push(`Balasan tidak mengandung teks wajib: '${requiredText}'`)
      }
    }
  }

  if (tc.mustNotContainText && reply.includes(tc.mustNotContainText)) {
    passed = false
    errors.push(`Balasan mengandung teks terlarang: '${tc.mustNotContainText}'`)
  }

  if (passed) {
    passedCount++
    console.log(`[PASS] ${tc.description}`)
    if (matchResult) {
      console.log(`       Topic: ${matchResult.topic.id} (score: ${matchResult.score})`)
    }
  } else {
    failedCount++
    console.error(`[FAIL] ${tc.description}`)
    for (const err of errors) {
      console.error(`       -> ${err}`)
    }
    console.log(`       Reply snippet: ${reply.slice(0, 120)}...`)
  }
}

console.log(`\nHasil: ${passedCount} passed, ${failedCount} failed dari total ${testCases.length} uji kasus.`)

if (failedCount > 0) {
  process.exit(1)
}
