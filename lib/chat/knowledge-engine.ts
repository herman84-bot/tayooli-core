/**
 * Tayooli ERP Customer Service Knowledge & Intent Engine
 * Provides intelligent, high-precision, natural customer support responses
 * grounded in Tayooli's 12 core modules, business rules (ADR-014, KO-1, KO-2),
 * and common user questions (POS, WMS, Inbound, Outbound, Troubleshooting, Pricing).
 */

export interface KnowledgeTopic {
  id: string
  title: string
  moduleName?: string
  moduleUrl?: string
  exactPhrases?: string[]
  primaryKeywords: RegExp[]
  secondaryKeywords: RegExp[]
  negativeKeywords?: RegExp[]
  priority?: number // Higher priority evaluated first
  getReply: (query?: string) => string
}

/**
 * Helper to wrap topic explanations in a warm, contextual CS persona response.
 * Detects whether the user is asking capability ("apakah bisa..."), tutorial ("bagaimana cara..."),
 * or troubleshooting ("kenapa error..."), ensuring immediate clarity.
 */
export function formatCsResponse(
  query: string | undefined,
  topicTitle: string,
  moduleName: string,
  moduleUrl: string,
  bodyContent: string
): string {
  const q = (query || '').toLowerCase()
  const isCapability = /\b(apakah|bisa|bisakah|bisa kah|bisa gak|bisa tidak|ada fitur|mendukung|support|tersedia|apakah ada)\b/i.test(q)
  const isTutorial = /\b(bagaimana|cara|panduan|langkah|tutorial|gimana|alur)\b/i.test(q)
  const isTrouble = /\b(kenapa|mengapa|error|gagal|tidak bisa|masalah|kendala|macet|rusak|mati|blank|bermasalah)\b/i.test(q)

  if (isCapability) {
    return `Halo kak! Ya, tentu saja bisa! Aplikasi Tayooli ERP sudah mendukung penuh fitur **${topicTitle}** melalui modul **${moduleName} (${moduleUrl})**. 😊\n\nBerikut panduan langkah-langkahnya:\n\n${bodyContent}\n\nApakah ada bagian dari alur ini yang ingin kami bantu jelaskan lebih lanjut, kak?`
  }

  if (isTutorial) {
    return `Halo kak! Berikut adalah panduan langkah demi langkah **${topicTitle}** di modul **${moduleName} (${moduleUrl})**:\n\n${bodyContent}\n\nJika ada langkah yang terkendala atau butuh panduan langsung, jangan ragu untuk tanyakan ya kak!`
  }

  if (isTrouble) {
    return `Halo kak! Jangan khawatir, berikut adalah panduan solusi untuk kendala **${topicTitle}**:\n\n${bodyContent}\n\nSemoga membantu kak, kabari kami jika masih terkendala ya!`
  }

  return `Halo kak! Terkait **${topicTitle}**, Anda dapat mengaksesnya di modul **${moduleName} (${moduleUrl})**:\n\n${bodyContent}\n\nAda hal lain seputar modul ini yang ingin Anda ketahui, kak?`
}

export const TAYOOLI_KNOWLEDGE_TOPICS: KnowledgeTopic[] = [
  // ── 1. GREETINGS & CASUAL INTERACTION ──
  {
    id: 'greetings',
    title: 'Salam & Sapaan',
    exactPhrases: [
      'halo',
      'hai',
      'hi',
      'pagi',
      'selamat pagi',
      'selamat siang',
      'selamat sore',
      'selamat malam',
      'assalamualaikum',
      'halo cs',
      'hi bot',
      'permisi',
      'p',
      'ping',
    ],
    primaryKeywords: [/\b(halo|hai|hey|hei)\b/i, /\bselamat (pagi|siang|sore|malam)\b/i, /\bassalamu'?alaikum\b/i],
    secondaryKeywords: [/\b(bisa bantu|ada orang|siap)\b/i],
    negativeKeywords: [/\b(cetak|transaksi|kasir|stok|gudang|transfer|surat jalan|error|rusak|bisa|bagaimana|cara|apa)\b/i],
    priority: 8,
    getReply: () =>
      `Halo kak! Selamat datang di layanan Customer Service Tayooli ERP. 😊

Ada yang bisa kami bantu seputar operasional bisnis Anda hari ini? Anda dapat menanyakan tentang:
• **Kasir POS**: Cara transaksi, pembayaran QRIS/Tunai, dan cetak struk thermal
• **Barang Masuk & Keluar**: Penerimaan barang (Inbound) & Surat Jalan (Delivery Order)
• **Gudang & Stok (WMS)**: Manajemen multi-gudang, penataan rak/bin, dan sistem nomor batch
• **Transfer Stok**: Prosedur mutasi barang antar cabang dari draf hingga selesai
• **Stok Opname**: Audit stok fisik dan penyesuaian selisih otomatis
• **Integrasi Marketplace**: Sinkronisasi stok otomatis dengan Tokopedia, Shopee, dan TikTok Shop

Silakan ketik pertanyaan Anda, kami siap memandu langkah demi langkah!`,
  },
  {
    id: 'gratitude',
    title: 'Ucapan Terima Kasih',
    exactPhrases: [
      'terima kasih',
      'makasih',
      'makasi',
      'thanks',
      'thank you',
      'tengkyu',
      'ok makasih',
      'oke terima kasih',
      'siap makasih',
      'mantap makasih',
      'matur nuwun',
    ],
    primaryKeywords: [/\b(terima\s*kasih|makasih|makasi|thanks|thank\s*you|tengkyu|matur\s*nuwun)\b/i],
    secondaryKeywords: [/\b(sangat membantu|oke|siap|bagus|keren|mantap)\b/i],
    negativeKeywords: [/\b(tapi|namun|belum|gagal|bisa|bagaimana)\b/i],
    priority: 8,
    getReply: () =>
      `Sama-sama kak! Senang sekali bisa membantu kelancaran operasional bisnis Anda. 🙏

Jika nanti ada kendala lain atau butuh panduan fitur Tayooli berikutnya, jangan ragu untuk chat ke sini lagi ya. Sukses selalu untuk bisnis Anda!`,
  },
  {
    id: 'identity_about',
    title: 'Profil & Identitas Tayooli',
    exactPhrases: [
      'siapa kamu',
      'siapa kamu?',
      'tayooli itu apa',
      'apa itu tayooli',
      'tentang tayooli',
      'apa fungsi tayooli',
      'aplikasi apa ini',
    ],
    primaryKeywords: [/\b(siapa (kamu|anda)|kamu siapa|tentang tayooli|apa itu tayooli|aplikasi apa ini)\b/i],
    secondaryKeywords: [/\b(fungsi tayooli|keunggulan tayooli|profil|tentang sistem)\b/i],
    priority: 7,
    getReply: () =>
      `Saya adalah asisten Customer Service resmi **Tayooli ERP**.

**Tayooli ERP Core** adalah ekosistem aplikasi bisnis terpadu yang dirancang khusus untuk operasional ritel, pergudangan, dan distribusi modern di Indonesia dengan 12 modul inti:
1. **Dasbor (/dashboard)**: Ringkasan pendapatan, omzet kasir, dan peringatan stok kritis
2. **Master Produk (/products)**: Katalog SKU, harga modal (HPP), harga jual, dan barcode
3. **Barang Masuk & Keluar (/wms/arus-barang)**: Penerimaan inbound dari supplier dan penerbitan Surat Jalan DO
4. **Gudang & Stok (/wms)**: Manajemen multi-gudang dan alokasi rak/bin
5. **Marketplace Omnichannel (/wms/marketplace)**: Sinkronisasi stok Tokopedia, Shopee, TikTok Shop
6. **Transfer Stok (/wms/transfers)**: Perpindahan stok antar gudang dengan otorisasi bertingkat
7. **Stok Opname (/wms/opname)**: Audit fisik dan rekonsiliasi varians stok
8. **Barang Rusak / Scrap (/wms/scrap)**: Pencatatan afkir, kadaluwarsa, dan write-off stok
9. **Barcode Scanner (/wms/scanner)**: Pemindaian barcode kamera ponsel atau USB scanner
10. **Kasir POS (/pos)**: Mesin kasir ritel kilat, QRIS dinamis, dan cetak struk thermal
11. **Pengaturan (/settings)**: Profil toko, pajak PPN, dan opsi cetak
12. **Bantuan (/help)**: Pusat panduan operasional dan customer service

Ada modul tertentu yang ingin kami jelaskan lebih dalam, kak?`,
  },
  {
    id: 'pricing_plans',
    title: 'Harga & Paket Langganan',
    exactPhrases: [
      'berapa harganya',
      'harga paket',
      'biaya langganan',
      'paket tayooli',
      'biaya tayooli',
      'apakah ada trial',
      'free trial',
      'harga aplikasi',
      'biaya aplikasi',
    ],
    primaryKeywords: [/\b(harga|biaya|tarif|biaya langganan|paket|pricing|bayar berapa|berlangganan)\b/i],
    secondaryKeywords: [/\b(starter|bisnis|enterprise|trial|gratis|promo|uji coba)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Paket Langganan & Biaya Tayooli ERP',
        'Pengaturan',
        '/settings',
        `Tayooli menyediakan paket fleksibel sesuai skala bisnis Anda:
• **Free Trial 14 Hari**: Gratis 100% tanpa komitmen, nikmati seluruh modul lengkap (POS, WMS, Marketplace, Multi-Gudang) untuk menguji kecocokan dengan bisnis Anda.
• **Paket Starter (Rp199.000 / bulan)**:
  - Cocok untuk UMKM & toko tunggal
  - Hingga 5 pengguna
  - Fitur lengkap Kasir POS, Produk, dan Pergudangan dasar
• **Paket Bisnis (Rp449.000 / bulan)**:
  - Cocok untuk bisnis berkembang & cabang ritel
  - Hingga 25 pengguna aktif
  - Multi-gudang, Stock Opname, Marketplace Omnichannel, dan otorisasi transfer stok
• **Paket Enterprise (Custom)**:
  - Pengguna tanpa batas, integrasi custom API, dan dukungan tim support teknis prioritas

Untuk mulai mencoba atau upgrade paket, Anda dapat membuka menu **Settings (/settings)** atau hubungi tim sales kami.`
      ),
  },

  // ── 2. POINT OF SALE (POS) & KASIR ──
  {
    id: 'pos_cashier',
    title: 'Kasir Point of Sale (POS)',
    exactPhrases: [
      'apakah ada fitur kasir',
      'apakah bisa kasir',
      'bisa kasir',
      'cara pakai kasir',
      'cara transaksi kasir',
      'bagaimana cara pos',
      'cara menggunakan pos',
      'fitur kasir',
      'jual di toko',
    ],
    primaryKeywords: [
      /\b(kasir|pos|point of sale|transaksi kasir|belanja|keranjang belanja|mesin kasir|aplikasi kasir)\b/i,
      /\b(bisa kasir|fitur kasir|buat jualan|penjualan ritel)\b/i,
    ],
    secondaryKeywords: [/\b(jual putus|konsinyasi|diskon|kembalian|kasir pos|uang pas|tunai|cash|qris)\b/i],
    negativeKeywords: [/\b(cetak struk|print struk|printer thermal|surat jalan|do)\b/i],
    priority: 6,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Kasir Point of Sale (POS)',
        'Kasir POS',
        '/pos',
        `1. **Buka Menu POS**: Masuk ke menu **Point of Sale (/pos)** pada navigasi samping.
2. **Pilih atau Scan Barang**:
   • Cari nama/SKU produk pada katalog.
   • Atau arahkan barcode scanner (USB / kamera) ke kemasan produk; item akan masuk ke keranjang belanja seketika.
3. **Atur Transaksi**:
   • Sesuaikan jumlah item (tambah/kurang kuantitas).
   • Masukkan diskon jika ada promosi.
   • Pilih tipe penjualan: **Jual Putus** (penjualan standar) atau **Konsinyasi**.
4. **Metode Pembayaran**:
   • **Tunai (Cash)**: Masukkan nominal uang yang diterima pelanggan, sistem otomatis menghitung uang kembalian secara presisi.
   • **QRIS / Non-Tunai**: Tampilkan QRIS dinamis untuk di-scan pelanggan, atau pilih kartu debit/transfer.
5. **Selesaikan Transaksi**:
   • Klik **Selesaikan Transaksi & Cetak**.
   • Saldo stok fisik di gudang toko otomatis terpotong secara real-time tanpa penundaan.`
      ),
  },
  {
    id: 'pos_receipt_print',
    title: 'Cetak Struk Belanja Kasir',
    exactPhrases: [
      'apakah bisa cetak struk',
      'bisa cetak struk',
      'cara cetak struk',
      'print struk',
      'printer thermal',
      'cetak nota kasir',
      'cetak struk pos',
      'cetak nota',
      'print nota',
      'ngeprint struk',
    ],
    primaryKeywords: [
      /\b(cetak struk|print struk|ngeprint struk|cetak nota|print nota|ngeprint nota|nota kasir|struk belanja|printer thermal|ukuran 58mm|ukuran 80mm)\b/i,
      /\b(bisa cetak struk|bisa print nota|struk pos)\b/i,
    ],
    secondaryKeywords: [/\b(printer|struk|nota|bluetooth printer|esc\/pos|kertas thermal|kertas struk)\b/i],
    negativeKeywords: [/\b(surat jalan|delivery order|\bdo\b|pengiriman|ekspedisi|macet|rusak|mati|error|tidak jalan|tidak respon|tidak merespons|bermasalah)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Cetak Struk Belanja Kasir',
        'Kasir POS',
        '/pos',
        `1. **Koneksi Printer Thermal**:
   • Tayooli mendukung printer thermal standar industri berprotokol **ESC/POS**, baik koneksi **USB** maupun **Bluetooth**.
   • Format kertas yang didukung: lebar **58mm** dan **80mm**.
2. **Saat Transaksi Berlangsung**:
   • Setelah memilih metode bayar di menu **/pos**, klik tombol **Selesaikan & Cetak Struk**.
   • Jendela dialog printer browser akan muncul secara otomatis.
3. **Kustomisasi Tampilan Struk**:
   • Anda dapat menambahkan logo toko, nama gerai, alamat, nomor telepon, dan pesan terima kasih pada bagian bawah struk.
   • Pengaturan ini berada di menu **Settings (/settings)** pada tab konfigurasi struk.
4. **Tips Pengoperasian**:
   • Pastikan driver printer thermal sudah terpasang di sistem operasi komputer/tablet.
   • Pastikan opsi *Headers and Footers* bawaan browser di-nonaktifkan agar struk rapi dan hemat kertas.`
      ),
  },

  // ── 3. SURAT JALAN & PENGIRIMAN OUTBOUND ──
  {
    id: 'delivery_order',
    title: 'Surat Jalan (Delivery Order / DO)',
    exactPhrases: [
      'apakah bisa cetak surat jalan',
      'bisa cetak surat jalan',
      'cara buat surat jalan',
      'cara buat do',
      'cetak surat jalan',
      'panduan delivery order',
      'bisa buat surat jalan',
      'surat jalan pengiriman',
    ],
    primaryKeywords: [
      /\b(surat jalan|delivery order|\bdo\b|kirim barang|pengiriman barang)\b/i,
      /\b(cetak surat jalan|print surat jalan|ngeprint surat jalan|buat surat jalan|terbitkan surat jalan)\b/i,
    ],
    secondaryKeywords: [/\b(sopir|driver|kurir|ekspedisi|plat nomor|resi|nomor do|tanda tangan|surat kirim)\b/i],
    negativeKeywords: [/\b(pos|struk|thermal|kasir)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Surat Jalan (Delivery Order / DO)',
        'Barang Masuk & Keluar',
        '/wms/arus-barang',
        `1. **Akses Menu**: Buka menu **Barang Masuk & Keluar (/wms/arus-barang)** dengan toggle **KELUAR**, atau langsung ke **Surat Jalan (/wms/delivery-orders)**.
2. **Buat Surat Jalan Baru**: Klik tombol **+ Terbitkan Surat Jalan**.
3. **Lengkapi Data Pengiriman**:
   • **Penerima / Pelanggan**: Pilih nama pemesan dan alamat tujuan pengiriman.
   • **Tanggal & Armada**: Masukkan tanggal kirim, nama sopir/kurir, serta nomor polisi kendaraan atau nomor resi ekspedisi.
   • **Rincian Barang**: Pilih SKU barang, jumlah unit, dan **nomor batch (batch_id)** yang dikirim sesuai prinsip ketertelusuran produk.
4. **Alur Status Pengiriman**:
   • **Draft**: Dokumen draf awal disiapkan.
   • **Picked & Packed**: Barang diambil dari rak gudang dan dikemas rapi.
   • **In Transit**: Barang diberangkatkan bersama armada kurir.
   • **Delivered**: Barang tiba dan diterima oleh pelanggan.
5. **Cetak Dokumen Resmi**:
   • Klik **Cetak Surat Jalan** untuk mencetak format standar bisnis Indonesia lengkap dengan Barcode/QR pelacak serta kolom tanda tangan Pengirim, Sopir, dan Penerima.`
      ),
  },

  // ── 4. ARUS BARANG MASUK (INBOUND) & BATCH MANAGEMENT ──
  {
    id: 'inbound_workflow',
    title: 'Penerimaan Barang Masuk (Inbound)',
    exactPhrases: [
      'apakah bisa terima barang',
      'bisa catat barang masuk',
      'cara terima barang',
      'cara barang masuk',
      'alur inbound',
      'barang masuk dari supplier',
      'terima barang supplier',
      'kulakan barang',
    ],
    primaryKeywords: [
      /\b(barang masuk|inbound|penerimaan barang|terima barang|pemasok|supplier|hasil produksi|kulakan)\b/i,
      /\b(catat barang masuk|alur barang masuk|bongkar muatan)\b/i,
    ],
    secondaryKeywords: [/\b(staging|putaway|bongkar|rak tujuan|dermaga|dock|surat jalan supplier)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Penerimaan Barang Masuk (Inbound)',
        'Barang Masuk & Keluar',
        '/wms/arus-barang',
        `1. **Akses Menu**: Buka **Barang Masuk & Keluar (/wms/arus-barang)** pada tab **MASUK**, atau akses **/wms/inbound**.
2. **Buat Penerimaan Baru**: Klik tombol **+ Penerimaan Barang Masuk**.
3. **Pilih Sumber & Barang**:
   • Tentukan asal penerimaan: apakah dari Pemasok Luar (Vendor), Hasil Produksi Internal, atau Retur Konsumen.
   • Pilih SKU produk yang diterima dan jumlah kuantitas riil yang dibongkar dari muatan.
4. **Wajib Nomor Batch & Tanggal Expired**:
   • Sesuai aturan akurasi pergudangan (KO-1), setiap barang masuk wajib diberikan **Nomor Batch (Batch ID)** dan tanggal kedaluwarsa (Expired Date) jika ada.
5. **Alur 2-Step Inbound (Staging -> Putaway)**:
   • **Tahap Staging**: Barang diletakkan sementara di area dermaga penerimaan untuk diperiksa mutu fisiknya.
   • **Tahap Putaway**: Setelah lolos verifikasi, staf gudang memindahkan barang ke rak/bin penyimpanan permanen, dan stok langsung bertambah pada inventori aktif.`
      ),
  },
  {
    id: 'batch_expiry_fefo',
    title: 'Manajemen Batch, Expired Date & FEFO',
    exactPhrases: [
      'apakah bisa sistem batch',
      'bisa lacak batch',
      'bagaimana sistem batch',
      'apa itu batch id',
      'rotasi fefo',
      'kadaluwarsa barang',
      'tanggal expired',
    ],
    primaryKeywords: [
      /\b(batch|batch id|nomor batch|expired|kadaluwarsa|kedaluwarsa|fefo|fifo|tanggal expired)\b/i,
      /\b(lacak batch|sistem batch|rotasi stok)\b/i,
    ],
    secondaryKeywords: [/\b(lacak batch|tracking batch|mutasi batch|alur batch|first expired)\b/i],
    negativeKeywords: [/\b(scrap|afkir|barang cacat|pecah|bocor|write-?off|buang)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Manajemen Batch & Rotasi Stok FEFO',
        'Gudang & Stok',
        '/wms',
        `1. **Prinsip Invarian Batch (KO-1)**:
   • Di Tayooli, setiap mutasi stok (barang masuk, pengeluaran, transfer, maupun penjualan) **wajib terhubung dengan Batch ID**.
   • Hal ini menjamin ketertelusuran asal-usul barang jika terjadi komplain kualitas atau penarikan produk (product recall).
2. **Rotasi FEFO (First-Expired, First-Out)**:
   • Sistem memprioritaskan barang dengan tanggal kedaluwarsa paling dekat untuk diambil lebih dulu saat pengemasan pesanan atau transaksi kasir.
   • Mengurangi risiko kerugian akibat barang basi atau kadaluwarsa di dalam rak gudang.
3. **Pengecekan Batch**:
   • Anda dapat melihat riwayat pergerakan batch tertentu di modul **Gudang & Stok (/wms)** pada tab data batch inventori.`
      ),
  },

  // ── 5. MANAJEMEN GUDANG & MULTI-GUDANG WMS ──
  {
    id: 'warehouse_wms',
    title: 'Gudang & Multi-Gudang (WMS)',
    exactPhrases: [
      'apakah bisa multi gudang',
      'apakah aplikasi ini bisa multi gudang',
      'bisa multi gudang',
      'cara tambah gudang',
      'manajemen multi gudang',
      'tambah gudang baru',
      'kelola gudang',
      'atur rak gudang',
      'atur bin gudang',
      'banyak gudang',
    ],
    primaryKeywords: [
      /\b(multi\s*gudang|banyak\s*gudang|tambah\s*gudang|buat\s*gudang|master\s*gudang|kelola\s*gudang|daftar\s*gudang|lokasi\s*gudang|manajemen\s*gudang)\b/i,
      /\b(rak\s*gudang|bin\s*gudang|zona\s*gudang|gudang\s*wms)\b/i,
      /\b(bisa\s*multi\s*gudang|apakah\s*bisa\s*banyak\s*gudang|cabang\s*gudang)\b/i,
    ],
    secondaryKeywords: [/\b(staging|pickable|kapasitas|alamat\s*gudang|kepala\s*gudang|saldo\s*gudang)\b/i],
    negativeKeywords: [/\b(transfer|pindah|arus barang|masuk|keluar|inbound|outbound|surat jalan|opname|scrap)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Manajemen Multi-Gudang & Stok WMS',
        'Gudang & Stok',
        '/wms',
        `1. **Akses Menu**: Buka menu **Gudang & Stok (/wms)** pada navigasi sidebar.
2. **Tambah Gudang / Cabang Baru**:
   • Klik tombol **+ Tambah Gudang Baru**.
   • Masukkan nama gudang (misal: *Gudang Utama Jakarta, Gudang Transit Surabaya, Toko Ritel Bandung*).
   • Lengkapi alamat fisik, penanggung jawab (PIC), dan kapasitas penyimpanan.
3. **Penataan Struktur Rak & Bin**:
   • Anda dapat membagi gudang ke dalam **Zona**, **Rak**, dan **Bin/Slot** penyimpanan spesifik.
   • Memudahkan staf pergudangan menemukan posisi barang saat proses *Picking* dan *Putaway*.
4. **Pantau Saldo Stok Real-Time**:
   • Setiap gudang memiliki pencatatan saldo inventori mandiri yang terisolasi rapi.
   • Anda dapat memfilter stok berdasarkan gudang tertentu, melihat kuantitas stok on-hand, stok reserved (pesanan online), maupun stok yang sedang dalam perjalanan transfer.`
      ),
  },

  // ── 6. TRANSFER STOK ANTAR GUDANG ──
  {
    id: 'stock_transfers',
    title: 'Transfer Stok Antar Gudang',
    exactPhrases: [
      'apakah bisa transfer gudang',
      'apakah aplikasi ini bisa transfer gudang',
      'bisa transfer gudang',
      'bisa transfer barang',
      'bisa transfer stok',
      'cara transfer stok',
      'pindah barang antar gudang',
      'mutasi gudang',
      'alur stock transfer',
      'transfer gudang',
      'transfer antar gudang',
      'mutasi barang antar gudang',
      'kirim barang ke cabang lain',
      'kirim barang antar gudang',
      'transfer barang',
      'mutasi barang',
      'mutasi stok',
      'pindah gudang',
      'pindah stok',
    ],
    primaryKeywords: [
      /\b(transfer|pindah|mutasi|kirim)\s+(stok|gudang|barang|produk|cabang|lokasi|antar\s*gudang|antar\s*cabang)\b/i,
      /\b(antar\s*gudang|antar\s*cabang|stock\s*transfer)\b/i,
      /\b(transfer\s*stok|transfer\s*gudang|transfer\s*barang|mutasi\s*gudang|mutasi\s*stok|mutasi\s*barang|pindah\s*gudang|pindah\s*barang|pindah\s*stok)\b/i,
      /\b(bisa|apakah\s*bisa|bisakah)\s+(transfer|pindah\s*barang|mutasi)\b/i,
      /\btransfer\b/i,
    ],
    secondaryKeywords: [
      /\b(approval|in\s*transit|gudang\s*asal|gudang\s*tujuan|otorisasi|persetujuan|terima\s*transfer|konfirmasi\s*terima)\b/i,
      /\b(cabang\s*asal|cabang\s*tujuan|armada|kurir\s*antar\s*gudang)\b/i,
    ],
    negativeKeywords: [/\b(transfer\s*bank|rekening|bca|mandiri|bri|bni|qris|metode\s*bayar|kartu\s*debit)\b/i],
    priority: 8,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Transfer Stok Antar Gudang',
        'Transfer Stok',
        '/wms/transfers',
        `1. **Buka Menu Transfer**: Masuk ke menu **Transfer Stok (/wms/transfers)** pada sidebar navigasi.
2. **Buat Pengajuan**: Klik tombol **+ Buat Transfer**.
3. **Pilih Lokasi & Barang**:
   • Tentukan **Gudang Asal** (pengirim) dan **Gudang Tujuan** (penerima).
   • Masukkan daftar SKU produk, nomor batch (**batch_id**), dan jumlah unit yang dipindahkan.
4. **Siklus Status Transfer**:
   • **Draft**: Draf perpindahan barang sedang disiapkan.
   • **Pending Approval**: Menunggu verifikasi dan persetujuan kepala gudang / supervisor.
   • **In Transit**: Barang sudah diberangkatkan dari gudang asal dalam perjalanan ekspedisi/armada.
   • **Received**: Barang telah tiba di lokasi tujuan dan dihitung fisik oleh staf gudang penerima.
5. **Konfirmasi & Pembaruan Saldo Otomatis**:
   • Begitu gudang tujuan menekan tombol konfirmasi penerimaan, saldo gudang asal berkurang dan saldo gudang tujuan bertambah otomatis secara real-time tanpa selisih.`
      ),
  },

  // ── 7. STOK OPNAME & AUDIT FISIK ──
  {
    id: 'stock_opname',
    title: 'Stock Opname & Rekonsiliasi Varians',
    exactPhrases: [
      'apakah bisa stok opname',
      'bisa stok opname',
      'bisa opname',
      'cara stock opname',
      'bagaimana opname',
      'audit stok fisik',
      'selisih stok fisik',
      'hitung stok fisik',
      'penyesuaian stok',
      'adjustment stok',
    ],
    primaryKeywords: [
      /\b(stock opname|stok opname|opname|hitung fisik|audit stok|selisih stok|varians stok)\b/i,
      /\b(bisa opname|bisa stok opname|penyesuaian stok|adjust stok)\b/i,
    ],
    secondaryKeywords: [/\b(rekonsiliasi|adjustment|penyesuaian stok|stok tidak cocok|stok minus|audit berkala)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Stock Opname & Rekonsiliasi Varians',
        'Stok Opname',
        '/wms/opname',
        `1. **Akses Menu**: Buka **Stok Opname (/wms/opname)**.
2. **Jadwalkan Sesi Opname**: Klik tombol **+ Buat Jadwal Opname Baru** dan tentukan gudang mana yang akan diaudit.
3. **Hitung Fisik Aktual**:
   • Tim lapangan menghitung jumlah fisik barang yang ada di rak/bin penyimpanan (dapat menggunakan barcode scanner).
   • Input angka fisik yang dihitung ke kolom penghitungan sistem.
4. **Kalkulasi Selisih (Varians)**:
   • Sistem secara otomatis memperlihatkan perbandingan: Stok Tercatat di Komputer vs Stok Nyata di Rak.
   • Muncul nilai selisih kuantitas (+ surplus / - defisit) beserta nilai rupiahnya.
5. **Terapkan Penyesuaian (Adjustment)**:
   • Supervisor meninjau alasan selisih, lalu mengklik **Terapkan Penyesuaian Stok (Adjust)**.
   • Saldo sistem akan otomatis diperbarui agar cocok 100% dengan fisik gudang dan tercatat di buku besar riwayat audit.`
      ),
  },

  // ── 8. BARANG RUSAK & SCRAP ──
  {
    id: 'scrap_damaged',
    title: 'Barang Rusak / Afkir / Scrap',
    exactPhrases: [
      'apakah bisa catat barang rusak',
      'bisa catat barang rusak',
      'cara catat barang rusak',
      'barang cacat',
      'cara scrap',
      'write off stok',
      'afkir barang',
      'buang barang rusak',
    ],
    primaryKeywords: [
      /\b(barang rusak|scrap|afkir|barang cacat|rusak|pecah|bocor|write-?off|expired barang)\b/i,
      /\b(pecah|afkir|barang basi|barang retur rusak)\b/i,
      /\b(bisa catat barang rusak|cara scrap)\b/i,
    ],
    secondaryKeywords: [/\b(buang barang|alasan rusak|potong inventori|penghapusan stok)\b/i],
    priority: 8,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Pencatatan Barang Rusak / Afkir / Scrap',
        'Barang Rusak / Scrap',
        '/wms/scrap',
        `1. **Buka Menu Scrap**: Masuk ke **Barang Rusak / Scrap (/wms/scrap)**.
2. **Buat Laporan Scrap**: Klik **+ Catat Scrap Baru**.
3. **Pilih Barang & Keterangan**:
   • Pilih gudang asal dan SKU produk yang mengalami kerusakan.
   • Masukkan nomor batch dan kuantitas unit yang diafkir/rusak.
   • Pilih kategori alasan (misalnya: *Cacat Pabrik, Rusak Saat Bongkar Muat, Kadaluwarsa / Expired, atau Rusak Air/Hama*).
4. **Eksekusi Pemotongan**:
   • Setelah data disimpan, sistem langsung mengeluarkan barang tersebut dari inventori aktif sehingga tidak akan tertampil di kasir POS atau marketplace.
   • Kerugian barang rusak tercatat pada laporan performa gudang secara akurat.`
      ),
  },

  // ── 9. BARCODE SCANNER & HARDWARE ──
  {
    id: 'barcode_scanner',
    title: 'Pemindai Barcode (Kamera & Hardware Gun)',
    exactPhrases: [
      'apakah bisa scan barcode',
      'bisa scan barcode',
      'bisa scan barcode lewat hp',
      'scan barcode pakai hp',
      'cara pakai scanner',
      'kamera barcode',
      'barcode scanner gun',
      'cara scan barcode',
      'apakah ada scanner',
    ],
    primaryKeywords: [
      /\b(barcode scanner|pemindai barcode|kamera scanner|scanner gun|scan barcode|alat scan)\b/i,
      /\b(bisa scan|bisa barcode|scan pakai kamera|scan lewat hp|scan usb|scanner bluetooth|scan qr)\b/i,
    ],
    secondaryKeywords: [/\b(kamera hp|webcam|usb barcode|bluetooth scanner|ean-13|code 128|qr code)\b/i],
    negativeKeywords: [/\b(tidak bisa|gagal|error|macet|ditolak|rusak|mati|blank|izin|permission)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Pemindai Barcode (Kamera & Hardware Gun)',
        'Pemindai Barcode',
        '/wms/scanner',
        `Tayooli mendukung dua metode pemindaian barcode yang fleksibel:
1. **Pemindai Kamera Bawaan (Webcam / Smartphone)**:
   • Buka menu **Barcode Scanner (/wms/scanner)** atau klik ikon kamera di halaman Kasir POS dan WMS.
   • Izinkan akses kamera pada peramban (browser).
   • Arahkan kamera ke barcode EAN-13, Code 128, atau QR Code produk. Sistem dilengkapi *visual pulse feedback* hijau saat barcode berhasil terbaca.
2. **Hardware Barcode Scanner Gun (USB / Bluetooth)**:
   • Cukup hubungkan barcode scanner fisik Anda ke komputer/laptop.
   • Scanner akan berfungsi sebagai keyboard input otomatis (**HID Mode**) tanpa perlu instalasi aplikasi tambahan.
   • Di kasir POS, Anda cukup menembakkan scanner ke produk, item akan otomatis masuk ke keranjang dalam sekejap.`
      ),
  },

  // ── 10. MARKETPLACE OMNICHANNEL ──
  {
    id: 'marketplace_sync',
    title: 'Marketplace Omnichannel & Sinkronisasi Stok',
    exactPhrases: [
      'apakah bisa integrasi shopee',
      'bisa integrasi shopee',
      'cara sinkron marketplace',
      'integrasi shopee',
      'integrasi tokopedia',
      'integrasi tiktok',
      'cegah overselling',
      'bisa sinkron stok',
    ],
    primaryKeywords: [
      /\b(marketplace|omnichannel|tokopedia|shopee|tiktok shop|tiktok|lazada|sinkronisasi stok|sync marketplace)\b/i,
      /\b(bisa integrasi|bisa sinkron|jual online|toko online)\b/i,
    ],
    secondaryKeywords: [/\b(alokasi stok|cegah overselling|pesanan online|multi channel)\b/i],
    priority: 7,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Marketplace Omnichannel & Sinkronisasi Stok',
        'Marketplace',
        '/wms/marketplace',
        `1. **Akses Modul**: Masuk ke menu **Marketplace (/wms/marketplace)**.
2. **Hubungkan Saluran Toko**:
   • Hubungkan akun toko online Anda (Tokopedia, Shopee, TikTok Shop, Lazada).
3. **Mekanisme Sinkronisasi Stok 2-Arah**:
   • **Penjualan Kasir POS Fisik**: Saat ada transaksi di kasir outlet, saldo stok di seluruh marketplace otomatis berkurang seketika.
   • **Pesanan Online Masuk**: Begitu pesanan baru terbit dari Shopee/Tokopedia, stok gudang Tayooli langsung dialokasikan (*reserved*) agar tidak terjual kembali ke orang lain.
4. **Pencegahan Overselling**:
   • Sistem mengunci stok pesanan aktif sehingga toko online Anda terbebas dari penalti pembatalan pesanan akibat stok habis (*out of stock*).`
      ),
  },

  // ── 11. MASTER PRODUK & KATALOG ──
  {
    id: 'products_master',
    title: 'Master Produk & Katalog SKU',
    exactPhrases: [
      'apakah bisa tambah produk',
      'cara tambah produk',
      'cara buat produk',
      'tambah barang baru',
      'daftar produk baru',
      'cara input barang baru',
      'input produk baru',
      'tambah master barang',
    ],
    primaryKeywords: [
      /\b(tambah produk|master produk|katalog produk|daftar barang|buat produk baru|sku produk|input produk|input barang|tambah barang|daftar produk)\b/i,
      /\b(kelola sku|atur harga|harga jual|harga beli|setting hpp)\b/i,
    ],
    secondaryKeywords: [/\b(harga beli|harga jual|hpp|stok minimal|minimum stok|barcode produk|satuan barang|pcs|box|lusin)\b/i],
    priority: 6,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Master Produk & Katalog SKU',
        'Produk',
        '/products',
        `1. **Buka Menu**: Masuk ke menu **Products (/products)** pada navigasi.
2. **Klik Tombol**: Tekan tombol **+ Tambah Produk**.
3. **Lengkapi Informasi Barang**:
   • **Kode SKU**: Kode unik pengenal barang (contoh: \`SKU-KAOS-001\`).
   • **Nama Produk**: Nama lengkap produk komersial.
   • **Harga Beli (HPP)** & **Harga Jual** (harga ritel kasir).
   • **Stok Minimum (Alert)**: Batas ambang stok agar muncul peringatan di dashboard saat stok menipis.
   • **Kode Barcode**: Masukkan angka barcode dari kemasan pabrik (atau scan langsung).
4. **Simpan**: Klik **Simpan**. Produk akan langsung siap digunakan di Kasir POS, Gudang WMS, dan etalase penjualan.`
      ),
  },

  // ── 12. DASBOR & ANALISIS BISNIS ──
  {
    id: 'dashboard_analytics',
    title: 'Dasbor & Ringkasan Performa Bisnis',
    exactPhrases: [
      'cara lihat omzet',
      'laporan penjualan',
      'pantau stok kritis',
      'grafik omzet',
      'dasbor bisnis',
      'analisis penjualan',
      'lihat pendapatan',
      'cara lihat omzet di dasbor',
      'bisa lihat omzet',
    ],
    primaryKeywords: [
      /\b(dashboard|dasbor|omzet|pendapatan|laporan penjualan|grafik penjualan|stok kritis|ringkasan bisnis|analisis penjualan)\b/i,
      /\b(bisa lihat omzet|pantau penjualan|performa toko|monitoring stok)\b/i,
    ],
    secondaryKeywords: [/\b(penjualan harian|ringkasan omzet|omzet kasir|grafik omset)\b/i],
    priority: 6,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Dasbor & Ringkasan Performa Bisnis',
        'Dasbor',
        '/dashboard',
        `1. **Akses Menu**: Buka menu **Dasbor (/dashboard)** pada navigasi paling atas.
2. **Ringkasan Omzet & Transaksi**:
   • Menampilkan total pendapatan harian, mingguan, dan bulanan secara real-time dari seluruh transaksi Kasir POS dan pesanan Marketplace.
   • Dilengkapi grafik pergerakan omzet interaktif untuk membaca tren pertumbuhan bisnis Anda.
3. **Peringatan Stok Kritis (Low Stock Alert)**:
   • Menampilkan daftar SKU produk yang telah mencapai batas minimum stok.
   • Membantu tim pengadaan segera melakukan pemesanan ulang (Re-order) ke pemasok sebelum barang habis.
4. **Performa Produk Terlaris**:
   • Melihat peringkat produk paling laku (*Top Best Seller*) untuk menentukan strategi promosi dan alokasi inventori.`
      ),
  },

  // ── 13. PENGATURAN USAHA, PAJAK & AKUN ──
  {
    id: 'settings_account',
    title: 'Pengaturan Usaha, Pajak & Akun',
    exactPhrases: [
      'cara ganti password',
      'atur pajak ppn',
      'ganti logo toko',
      'pengaturan usaha',
      'ubah kata sandi',
      'setting ppn',
    ],
    primaryKeywords: [
      /\b(pengaturan|settings|ubah password|ganti kata sandi|profil toko|logo toko|pajak ppn)\b/i,
      /\b(atur pajak|setting ppn|profil usaha)\b/i,
    ],
    secondaryKeywords: [/\b(akun saya|alamat toko|informasi bisnis)\b/i],
    priority: 6,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Pengaturan Usaha, Pajak & Akun',
        'Pengaturan',
        '/settings',
        `Buka menu **Settings (/settings)** pada sidebar untuk menyesuaikan:
• **Profil Usaha**: Nama bisnis, nomor telepon resmi, email, alamat fisik toko/kantor, serta upload logo perusahaan untuk dokumen resmi.
• **Preferensi Pajak & Mata Uang**: Aktifkan tarif PPN (misal: 11%) agar otomatis terhitung di struk kasir dan faktur penjualan.
• **Format Struk Kasir**: Masukkan teks sambutan di bagian atas (*Header*) dan ucapan terima kasih di bagian bawah (*Footer*) struk belanja.
• **Keamanan Akun**: Perbarui kata sandi secara berkala untuk menjaga keamanan data usaha Anda.`
      ),
  },

  // ── 14. PUSAT BANTUAN & CS SUPPORT ──
  {
    id: 'help_center',
    title: 'Pusat Bantuan & Layanan Support',
    exactPhrases: [
      'pusat bantuan',
      'help desk',
      'hubungi customer service',
      'bantuan operasional',
      'kontak support',
      'bantuan tayooli',
    ],
    primaryKeywords: [
      /\b(pusat bantuan|help center|help desk|kontak cs|kontak support|layanan pelanggan|live chat support)\b/i,
    ],
    secondaryKeywords: [/\b(faq|panduan pengguna|tiket kendala|bantuan teknis)\b/i],
    priority: 6,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Pusat Bantuan & Layanan Support',
        'Bantuan',
        '/help',
        `1. **Akses Modul**: Masuk ke menu **Bantuan (/help)** pada navigasi samping.
2. **Panduan Lengkap (FAQ)**:
   • Memuat dokumentasi operasional 12 modul inti Tayooli ERP mulai dari Kasir POS, Gudang WMS, Inbound, Outbound, hingga integrasi Marketplace.
3. **Live Chat Customer Service**:
   • Anda dapat langsung bertanya kapan saja melalui jendela chat support ini untuk mendapatkan panduan praktis dan troubleshooting kendala secara instan.
4. **Dukungan Tiket Teknis**:
   • Untuk kendala teknis mendalam atau integrasi custom, tim support kami siap mendampingi operasional usaha Anda agar berjalan lancar 24/7.`
      ),
  },

  // ── 15. TROUBLESHOOTING & KENDALA TEKNIS ──
  {
    id: 'troubleshooting_camera',
    title: 'Troubleshooting: Izin Kamera Scanner',
    exactPhrases: [
      'kamera tidak bisa dibuka',
      'scanner tidak bisa buka kamera',
      'kamera error',
      'izin kamera ditolak',
    ],
    primaryKeywords: [
      /\b(kamera.*(tidak|buka|dibuka|gagal|error|macet|ditolak|blank)|izin kamera|permission denied|camera permission)\b/i,
      /\b(scanner.*(tidak bisa|gagal|error|macet|blank))\b/i,
    ],
    secondaryKeywords: [/\b(browser|https|chrome|safari)\b/i],
    priority: 10,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Solusi Kamera Barcode Scanner Tidak Bisa Dibuka',
        'Pemindai Barcode',
        '/wms/scanner',
        `1. **Pastikan Menggunakan Protokol HTTPS**:
   • Peramban modern (Chrome, Safari, Edge) mewajibkan koneksi aman (HTTPS) untuk mengizinkan akses kamera web.
2. **Cek Izin Kamera di Browser**:
   • Klik ikon gembok/setelan di samping bilah alamat URL (address bar) browser Anda.
   • Pastikan setelan **Camera** berada pada status **Allow (Izinkan)**.
3. **Tutup Aplikasi Lain**:
   • Pastikan kamera tidak sedang dipakai oleh aplikasi lain (seperti Zoom, Google Meet, atau aplikasi kamera bawaan OS).
4. **Alternatif Terbaik**:
   • Jika digunakan di PC kasir, kami sangat menyarankan memakai **Hardware Barcode Gun (USB)** yang dapat langsung dicolok tanpa butuh izin kamera.`
      ),
  },
  {
    id: 'troubleshooting_printer',
    title: 'Troubleshooting: Printer Thermal Tidak Merespons',
    exactPhrases: [
      'printer tidak bisa cetak',
      'printer tidak merespon',
      'printer macet',
      'struk tidak keluar',
      'printer tidak merespons',
    ],
    primaryKeywords: [
      /\b(printer|struk).*(tidak|gagal|error|macet|rusak|mati|merespons|respon)\b/i,
      /\b(printer (mati|error|rusak|macet|tidak keluar|tidak terdeteksi))\b/i,
      /\b(printer.*macet|printer.*tidak merespons|printer.*tidak respon)\b/i,
    ],
    secondaryKeywords: [/\b(thermal|kabel usb|bluetooth|paper cut)\b/i],
    priority: 10,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Solusi Printer Thermal Struk Tidak Merespons',
        'Kasir POS',
        '/pos',
        `1. **Periksa Kabel & Daya**:
   • Pastikan lampu indikator daya printer menyala (warna hijau/biru) dan tidak berkedip merah (indikator kehabisan kertas).
   • Pastikan gulungan kertas thermal terpasang pada arah yang benar (sisi berselaput kimia menghadap pemanas cetak).
2. **Uji Print Test dari Sistem Operasi**:
   • Masuk ke setelan Printer OS (Windows / macOS) dan lakukan *Print Test Page*.
   • Jika test page berhasil keluar, berarti koneksi fisik sudah aman.
3. **Pengaturan Dialog Browser**:
   • Saat dialog cetak muncul di browser, pastikan tujuan (*Destination*) mengarah ke nama printer thermal Anda, bukan *Save as PDF*.
   • Set ukuran kertas sesuai tipe printer Anda: **58mm (Roll Paper)** atau **80mm (Roll Paper)**.`
      ),
  },
  {
    id: 'troubleshooting_stock',
    title: 'Troubleshooting: Stok Selisih / Tidak Terpotong',
    exactPhrases: [
      'kenapa stok tidak berkurang',
      'stok tidak sinkron',
      'stok minus',
      'kenapa stok berbeda',
      'kenapa stok selisih',
    ],
    primaryKeywords: [/\b(stok.*(tidak berkurang|selisih|beda|minus|salah|tidak cocok)|kenapa stok)\b/i],
    secondaryKeywords: [/\b(kasir|opname|sync)\b/i],
    priority: 8,
    getReply: (query?: string) =>
      formatCsResponse(
        query,
        'Solusi Kendala Stok Selisih atau Tidak Terpotong',
        'Gudang & Stok',
        '/wms',
        `1. **Periksa Outlet / Gudang yang Dipilih**:
   • Di Kasir POS, pastikan Anda memilih gudang/outlet yang tepat saat memulai sesi kasir. Transaksi kasir hanya memotong stok di gudang yang aktif dipilih.
2. **Cek Riwayat Mutasi Buku Besar**:
   • Buka menu **Gudang & Stok (/wms)** untuk meninjau riwayat mutasi SKU tersebut.
   • Periksa apakah ada pesanan marketplace yang berstatus *Reserved* (terkunci) atau transfer stok yang masih berstatus *In Transit*.
3. **Lakukan Rekonsiliasi via Stok Opname**:
   • Jika ada barang fisik hilang atau rusak yang belum tercatat, buka menu **Stok Opname (/wms/opname)**, hitung ulang fisik, dan lakukan *Adjustment* agar angka sistem dan rak kembali serasi 100%.`
      ),
  },

  // ── 16. MODUL AKUNTANSI LAMA (DI-DECOUPLE) ──
  {
    id: 'legacy_accounting_faq',
    title: 'Informasi Modul Akuntansi & Fokus Sistem',
    exactPhrases: [
      'jurnal umum',
      'chart of accounts',
      'buku kas masuk keluar',
      'laporan laba rugi akuntansi',
      'coa',
      'debit kredit',
    ],
    primaryKeywords: [/\b(jurnal umum|jurnal akuntansi|chart of accounts|\bcoa\b|debit kredit|neraca saldo|p2p|o2c)\b/i],
    secondaryKeywords: [/\b(akuntansi manual|jurnal manual)\b/i],
    priority: 8,
    getReply: () =>
      `Halo kak! **Tayooli ERP Core** dirancang khusus berfokus pada **12 Modul Operasional Inti**:
1. **Dasbor** (/dashboard)
2. **Produk & SKU** (/products)
3. **Barang Masuk & Keluar** (/wms/arus-barang)
4. **Gudang & Stok** (/wms)
5. **Marketplace Omnichannel** (/wms/marketplace)
6. **Transfer Stok** (/wms/transfers)
7. **Stok Opname** (/wms/opname)
8. **Barang Rusak / Scrap** (/wms/scrap)
9. **Pemindai Barcode** (/wms/scanner)
10. **Kasir POS** (/pos)
11. **Pengaturan Usaha** (/settings)
12. **Pusat Bantuan** (/help)

Sistem sengaja mengeliminasi kerumitan akuntansi manual (seperti Chart of Accounts atau entri jurnal debit-kredit manual) agar operasional pergudangan, kasir, dan sinkronisasi stok toko Anda berjalan ultra cepat, akurat, dan mudah digunakan oleh staf tanpa latar belakang akuntansi.

Apakah ada alur operasional barang atau kasir yang bisa kami bantu jelaskan lebih lanjut, kak?`,
  },
]

export interface MatchResult {
  topic: KnowledgeTopic
  score: number
  matchedKeywords: string[]
}

/**
 * Normalizes input text for accurate intent recognition
 */
export function normalizeQuery(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

/**
 * Strips conversational filler and question words to isolate the core operational intent
 */
export function extractCoreIntent(text: string): string {
  return normalizeQuery(text)
    .replace(
      /\b(apakah|aplikasi|ini|bisa|bisakah|bisa kah|bisa gak|bisa tidak|ada|fitur|tolong|bagaimana|gimana|cara|alur|dong|ya|kak|min|admin|sistem|di tayooli|tayooli|minta|butuh|ingin|mau|untuk)\b/g,
      ' '
    )
    .replace(/\s+/g, ' ')
    .trim()
}

/**
 * Finds the best matching knowledge topic using multi-tier weighted scoring
 */
export function findBestTopic(query: string): MatchResult | null {
  const normalized = normalizeQuery(query)
  if (!normalized) return null

  const coreIntent = extractCoreIntent(query)

  let bestResult: MatchResult | null = null

  for (const topic of TAYOOLI_KNOWLEDGE_TOPICS) {
    let score = 0
    const matched: string[] = []

    // 1. Check Exact Phrases (weight: 25) against normalized OR coreIntent
    if (topic.exactPhrases) {
      for (const phrase of topic.exactPhrases) {
        const normPhrase = normalizeQuery(phrase)
        if (!normPhrase) continue

        // Single or 2-letter phrases (like 'p' or 'hi') MUST be strictly exact matches, never substrings
        const isStrictExact = normalized === normPhrase || (coreIntent ? coreIntent === normPhrase : false)
        let isContainedWord = false
        if (!isStrictExact && normPhrase.length >= 3) {
          const escaped = normPhrase.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
          const wordRx = new RegExp(`(?:^|\\s)${escaped}(?:\\s|$)`, 'i')
          isContainedWord = wordRx.test(normalized) || (coreIntent ? wordRx.test(coreIntent) : false)
        }

        if (isStrictExact || isContainedWord) {
          score += 25
          matched.push(`exact:${phrase}`)
        }
      }
    }

    // 2. Check Negative Keywords (penalty: -25)
    if (topic.negativeKeywords) {
      for (const neg of topic.negativeKeywords) {
        if (neg.test(normalized)) {
          score -= 25
        }
      }
    }

    // 3. Check Primary Keywords (weight: 12)
    for (const rx of topic.primaryKeywords) {
      if (rx.test(normalized) || (coreIntent && rx.test(coreIntent))) {
        score += 12
        matched.push(rx.source)
      }
    }

    // 4. Check Secondary Keywords (weight: 4)
    for (const rx of topic.secondaryKeywords) {
      if (rx.test(normalized) || (coreIntent && rx.test(coreIntent))) {
        score += 4
        matched.push(rx.source)
      }
    }

    // 5. Cross-term correlation boosts for natural queries
    if (topic.id === 'stock_transfers') {
      if (/\btransfer\b/i.test(normalized) && /\b(gudang|stok|barang|produk|cabang|lokasi|asal|tujuan)\b/i.test(normalized)) {
        score += 20
        matched.push('cross:transfer+location')
      }
    } else if (topic.id === 'barcode_scanner') {
      if (/\b(scan|scanner)\b/i.test(normalized) && /\b(barcode|kamera|hp|alat|gun|qr)\b/i.test(normalized)) {
        score += 20
        matched.push('cross:scan+hardware')
      }
    } else if (topic.id === 'pos_receipt_print') {
      if (/\b(cetak|print|ngeprint)\b/i.test(normalized) && /\b(struk|nota|thermal|kasir)\b/i.test(normalized)) {
        score += 20
        matched.push('cross:print+receipt')
      }
    } else if (topic.id === 'warehouse_wms') {
      if (/\b(multi|banyak|tambah|kelola|daftar)\b/i.test(normalized) && /\b(gudang|rak|bin)\b/i.test(normalized)) {
        score += 20
        matched.push('cross:multi+warehouse')
      }
    } else if (topic.id === 'marketplace_sync') {
      if (/\b(integrasi|sinkron|sync|hubungkan)\b/i.test(normalized) && /\b(shopee|tokopedia|tiktok|lazada|marketplace)\b/i.test(normalized)) {
        score += 20
        matched.push('cross:marketplace+sync')
      }
    } else if (topic.id === 'dashboard_analytics') {
      if (/\b(lihat|pantau|cek|laporan)\b/i.test(normalized) && /\b(omzet|dasbor|dashboard|penjualan|stok kritis)\b/i.test(normalized)) {
        score += 20
        matched.push('cross:dashboard+analytics')
      }
    } else if (topic.id === 'troubleshooting_camera') {
      if (/\bkamera\b/i.test(normalized) && /\b(tidak|gagal|error|macet|ditolak|blank|izin|permission)\b/i.test(normalized)) {
        score += 30
        matched.push('cross:camera+error')
      }
    } else if (topic.id === 'scrap_damaged') {
      if (/\b(pecah|rusak|cacat|afkir|bocor|kadaluwarsa|kedaluwarsa|expired)\b/i.test(normalized) && /\b(barang|stok|produk|catat|write-?off)\b/i.test(normalized)) {
        score += 25
        matched.push('cross:scrap+damage')
      }
    }

    // Apply topic priority bonus only if at least one keyword/phrase actually matched
    if (matched.length > 0 && topic.priority) {
      score += topic.priority
    }

    if (score > 0 && (!bestResult || score > bestResult.score)) {
      bestResult = { topic, score, matchedKeywords: matched }
    }
  }

  return bestResult
}

/**
 * Handles ambiguous queries (e.g. user just asks "cara cetak")
 */
export function handleAmbiguousQuery(query: string): string | null {
  const norm = normalizeQuery(query)

  // 1. Ambiguous print query (mentions cetak/print/ngeprint but doesn't specify receipt vs delivery order)
  const isPrintWord = /\b(cetak|mencetak|print|ngeprint)\b/i.test(norm)
  const hasReceiptContext = /\b(struk|nota|kasir|thermal|58mm|80mm)\b/i.test(norm)
  const hasDoContext = /\b(surat jalan|delivery order|\bdo\b|pengiriman|ekspedisi)\b/i.test(norm)

  if (isPrintWord && !hasReceiptContext && !hasDoContext) {
    return `Halo kak! Terkait panduan pencetakan di Tayooli ERP, terdapat 2 jenis cetakan yang sering digunakan:

1. **Struk Kasir (POS)**: Dicetak pada printer thermal (58mm/80mm) saat transaksi belanja pelanggan selesai di menu **Point of Sale (/pos)**.
2. **Surat Jalan (DO)**: Dicetak sebagai bukti resmi pengiriman muatan barang ke pelanggan/ekspedisi di menu **Barang Masuk & Keluar (/wms/arus-barang)**.

Boleh diperjelas kak, jenis cetakan mana yang sedang Anda butuhkan saat ini? Kami siap memandu langkahnya!`
  }

  // 2. Ambiguous item flow / mutasi query
  const isGeneralFlow = /\b(alur barang|proses barang|kelola barang|manajemen barang)\b/i.test(norm)
  const hasSpecificFlow = /\b(masuk|keluar|inbound|outbound|transfer|opname|scrap|rusak|afkir|supplier|gudang|cabang)\b/i.test(norm)

  if ((isGeneralFlow || norm === 'barang') && !hasSpecificFlow) {
    return `Halo kak! Terkait pengelolaan barang di Tayooli ERP, terdapat beberapa modul yang bisa digunakan:
• **Barang Masuk & Keluar (/wms/arus-barang)**: Penerimaan barang supplier (Inbound) dan Surat Jalan (Outbound).
• **Transfer Stok (/wms/transfers)**: Perpindahan barang antar cabang gudang.
• **Stok Opname (/wms/opname)**: Pengecekan fisik dan audit selisih stok.
• **Barang Rusak / Scrap (/wms/scrap)**: Pencatatan afkir barang cacat/expired.

Alur mana yang sedang ingin Anda lakukan saat ini, kak?`
  }

  return null
}

/**
 * Generates an empathetic, professional fallback reply when no topic matches strongly
 */
export function generateSmartFallback(query: string): string {
  const trimmed = query.trim()
  return `Halo kak! Mohon maaf, kami belum dapat memahami pertanyaan Anda secara spesifik mengenai *"**${trimmed}**"*. 😊

Sebagai asisten operasional resmi **Tayooli ERP**, kami siap memandu Anda pada 12 modul inti berikut:
• **Kasir POS (/pos)**: Mesin kasir kilat, barcode scanner, pembayaran QRIS/Tunai, dan cetak struk thermal (58mm/80mm)
• **Barang Masuk & Keluar (/wms/arus-barang)**: Penerimaan inbound dari supplier dan penerbitan Surat Jalan pengiriman (DO)
• **Gudang & Stok (/wms)**: Saldo inventori multi-gudang, penataan rak/bin, dan pelacakan nomor batch (KO-1)
• **Transfer Stok (/wms/transfers)**: Mutasi dan perpindahan barang antar cabang gudang dengan persetujuan (approval)
• **Stok Opname (/wms/opname)**: Audit stok fisik berkala dan penyesuaian selisih kuantitas/rupiah otomatis
• **Barang Rusak / Scrap (/wms/scrap)**: Pencatatan afkir barang cacat, bocor, atau kedaluwarsa
• **Pemindai Barcode (/wms/scanner)**: Scan barcode via kamera HP/laptop atau hardware scanner USB/Bluetooth
• **Marketplace (/wms/marketplace)**: Sinkronisasi stok otomatis toko online (Tokopedia, Shopee, TikTok Shop)
• **Master Produk (/products)**: Katalog SKU, harga modal (HPP), harga jual, dan stok minimum alert
• **Dasbor (/dashboard)**: Ringkasan pendapatan, omzet kasir, dan grafik penjualan real-time
• **Pengaturan (/settings)**: Profil usaha, format struk belanja, dan tarif pajak PPN

Bisa ceritakan alur atau modul apa yang sedang ingin Anda gunakan saat ini, kak? Kami siap membantu!`
}

/**
 * Main dispatcher to get the best customer service response
 */
export function getCustomerServiceReply(userQuery: string): string {
  const trimmed = userQuery.trim()
  if (!trimmed) {
    return 'Halo kak! Ada yang bisa kami bantu seputar penggunaan Tayooli ERP hari ini?'
  }

  // Check ambiguous queries first
  const ambiguous = handleAmbiguousQuery(trimmed)
  if (ambiguous) {
    return ambiguous
  }

  // Find best topic
  const match = findBestTopic(trimmed)
  if (match && match.score >= 10) {
    return match.topic.getReply(trimmed)
  }

  // Fallback
  return generateSmartFallback(trimmed)
}
