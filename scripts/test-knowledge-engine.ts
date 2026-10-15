import { getCustomerServiceReply, findBestTopic, TAYOOLI_KNOWLEDGE_TOPICS } from '../lib/chat/knowledge-engine'

interface TestCase {
  query: string
  expectedTopicId?: string
  mustContainText?: string
  mustNotContainText?: string
  description: string
}

const testCases: TestCase[] = [
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
    query: 'Bagaimana alur transfer barang antar gudang cabang?',
    expectedTopicId: 'stock_transfers',
    mustContainText: 'Transfer Stok Antar Gudang',
    description: 'Panduan transfer stok antar cabang',
  },
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
    query: 'ada barang pecah dan kadaluwarsa mau dicatat afkir',
    expectedTopicId: 'scrap_damaged',
    mustContainText: 'Barang Rusak / Scrap',
    description: 'Pencatatan barang rusak / scrap / write-off',
  },
  {
    query: 'bagaimana cara tambah produk baru dan isi SKU?',
    expectedTopicId: 'products_master',
    mustContainText: 'Menambahkan Master Produk',
    description: 'Master data produk dan SKU',
  },
  {
    query: 'bagaimana cara sinkronisasi stok toko shopee tokopedia tiktok?',
    expectedTopicId: 'marketplace_sync',
    mustContainText: 'Marketplace Omnichannel',
    description: 'Integrasi omnichannel marketplace',
  },
  {
    query: 'gimana cara cetak?',
    mustContainText: '**Struk Kasir** atau **Surat Jalan**',
    description: 'Query ambigu "cetak" harus memunculkan klarifikasi ramah',
  },
  {
    query: 'alur barang di tayooli',
    mustContainText: 'Barang Masuk',
    description: 'Query ambigu "alur barang" harus memunculkan pilihan alur',
  },
  {
    query: 'bagaimana cara buat chart of accounts dan entri jurnal debit kredit?',
    expectedTopicId: 'legacy_accounting_faq',
    mustContainText: '12 Modul Operasional Inti',
    description: 'Permintaan modul akuntansi lama (CoA/Jurnal) ditolak dengan edukasi 12 modul inti',
  },
  {
    query: 'siapa pemenang piala dunia sepak bola 2022?',
    mustContainText: 'Terima kasih atas pertanyaannya',
    description: 'Pertanyaan di luar domain ERP menghasilkan smart fallback yang ramah',
  },
  {
    query: 'bagaimana cara penerimaan barang masuk dari supplier?',
    expectedTopicId: 'inbound_workflow',
    mustContainText: 'Barang Masuk',
    description: 'Alur penerimaan barang masuk supplier (Inbound)',
  },
  {
    query: 'bagaimana cara ganti password dan atur tarif pajak ppn?',
    expectedTopicId: 'settings_account',
    mustContainText: 'Pengaturan',
    description: 'Pengaturan akun, profil usaha, dan pajak PPN',
  },
  {
    query: 'gimana cara ngeprint?',
    mustContainText: 'Struk Belanja Kasir',
    description: 'Colloquial affix "ngeprint" harus terdeteksi disambiguasi struk vs surat jalan',
  },
  {
    query: 'bagaimana mencetak dokumen?',
    mustContainText: 'Surat Jalan',
    description: 'Colloquial affix "mencetak" harus terdeteksi disambiguasi struk vs surat jalan',
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
