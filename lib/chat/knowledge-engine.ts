/**
 * Tayooli ERP Customer Service Knowledge & Intent Engine
 * Provides intelligent, high-precision, natural customer support responses
 * grounded in Tayooli's 12 core modules, business rules (ADR-014, KO-1, KO-2),
 * and common user questions (POS, WMS, Inbound, Outbound, Troubleshooting, Pricing).
 */

export interface KnowledgeTopic {
  id: string
  title: string
  exactPhrases?: string[]
  primaryKeywords: RegExp[]
  secondaryKeywords: RegExp[]
  negativeKeywords?: RegExp[]
  priority?: number // Higher priority evaluated first
  getReply: (query: string) => string
}

export const TAYOOLI_KNOWLEDGE_TOPICS: KnowledgeTopic[] = [
  // ── 1. GREETINGS & CASUAL INTERACTION ──
  {
    id: 'greetings',
    title: 'Salam & Sapaan',
    exactPhrases: ['halo', 'hai', 'hi', 'pagi', 'selamat pagi', 'selamat siang', 'selamat sore', 'selamat malam', 'assalamualaikum', 'halo cs', 'hi bot', 'permisi'],
    primaryKeywords: [/\b(halo|hai|hey|hei)\b/i, /\bselamat (pagi|siang|sore|malam)\b/i, /\bassalamu'?alaikum\b/i],
    secondaryKeywords: [/\b(bisa bantu|ada orang|siap)\b/i],
    negativeKeywords: [/\b(cetak|transaksi|kasir|stok|gudang|transfer|surat jalan|error|rusak)\b/i],
    priority: 8,
    getReply: () =>
      `Halo kak! Selamat datang di layanan Customer Service Tayooli ERP. 😊

Ada yang bisa kami bantu seputar operasional bisnis Anda hari ini? Anda dapat menanyakan tentang:
• **Kasir POS**: Cara transaksi, pembayaran QRIS/Tunai, dan cetak struk thermal
• **Arus Barang Masuk & Keluar**: Penerimaan barang (Inbound) & Surat Jalan (Delivery Order)
• **Gudang & Stok (WMS)**: Manajemen multi-gudang, penataan rak/bin, dan sistem nomor batch
• **Transfer Stok**: Prosedur mutasi barang antar cabang dari draf hingga selesai
• **Stok Opname**: Audit stok fisik dan penyesuaian selisih otomatis
• **Integrasi Marketplace**: Sinkronisasi stok otomatis dengan Tokopedia, Shopee, dan TikTok Shop

Silakan ketik pertanyaan Anda, kami siap memandu langkah demi langkah!`,
  },
  {
    id: 'gratitude',
    title: 'Ucapan Terima Kasih',
    exactPhrases: ['terima kasih', 'makasih', 'makasi', 'thanks', 'thank you', 'tengkyu', 'ok makasih', 'oke terima kasih', 'siap makasih', 'mantap makasih'],
    primaryKeywords: [/\b(terima\s*kasih|makasih|makasi|thanks|thank\s*you|tengkyu)\b/i],
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
    exactPhrases: ['siapa kamu', 'siapa kamu?', 'tayooli itu apa', 'apa itu tayooli', 'tentang tayooli', 'apa fungsi tayooli'],
    primaryKeywords: [/\b(siapa (kamu|anda)|kamu siapa|tentang tayooli|apa itu tayooli)\b/i],
    secondaryKeywords: [/\b(fungsi tayooli|keunggulan tayooli|profil)\b/i],
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
    exactPhrases: ['berapa harganya', 'harga paket', 'biaya langganan', 'paket tayooli', 'biaya tayooli', 'apakah ada trial', 'free trial'],
    primaryKeywords: [/\b(harga|biaya|tarif|biaya langganan|paket|pricing|bayar berapa|berlangganan)\b/i],
    secondaryKeywords: [/\b(starter|bisnis|enterprise|trial|gratis|promo)\b/i],
    priority: 7,
    getReply: () =>
      `**Paket Langganan & Biaya Tayooli ERP:**

Tayooli menyediakan paket fleksibel sesuai skala bisnis Anda:
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

Untuk mulai mencoba atau upgrade paket, Anda dapat membuka menu **Settings (/settings)** atau hubungi tim sales kami.`,
  },

  // ── 2. POINT OF SALE (POS) & KASIR ──
  {
    id: 'pos_cashier',
    title: 'Kasir Point of Sale (POS)',
    exactPhrases: ['cara pakai kasir', 'cara transaksi kasir', 'bagaimana cara pos', 'cara menggunakan pos'],
    primaryKeywords: [/\b(kasir|pos|point of sale|transaksi kasir|belanja|keranjang belanja)\b/i],
    secondaryKeywords: [/\b(jual putus|konsinyasi|diskon|kembalian|kasir pos)\b/i],
    negativeKeywords: [/\b(cetak struk|print struk|printer thermal|surat jalan|do)\b/i],
    priority: 6,
    getReply: () =>
      `**Panduan Pengoperasian Kasir POS di Tayooli ERP:**

1. **Buka Menu POS**: Masuk ke menu **Point of Sale (/pos)** pada navigasi samping.
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
   • Saldo stok fisik di gudang toko otomatis terpotong secara real-time tanpa penundaan.`,
  },
  {
    id: 'pos_receipt_print',
    title: 'Cetak Struk Belanja Kasir',
    exactPhrases: ['cara cetak struk', 'print struk', 'printer thermal', 'cetak nota kasir', 'cetak struk pos'],
    primaryKeywords: [/\b(cetak struk|print struk|cetak nota|nota kasir|struk belanja|printer thermal|ukuran 58mm|ukuran 80mm)\b/i],
    secondaryKeywords: [/\b(printer|struk|nota|bluetooth printer|esc\/pos|kertas)\b/i],
    negativeKeywords: [/\b(surat jalan|delivery order|\bdo\b|pengiriman|ekspedisi|macet|rusak|mati|error|tidak jalan|tidak respon|tidak merespons|bermasalah)\b/i],
    priority: 7,
    getReply: () =>
      `**Panduan Cetak Struk Belanja Kasir (POS) di Tayooli ERP:**

1. **Koneksi Printer Thermal**:
   • Tayooli mendukung printer thermal standar industri berprotokol **ESC/POS**, baik koneksi **USB** maupun **Bluetooth**.
   • Format kertas yang didukung: lebar **58mm** dan **80mm**.
2. **Saat Transaksi Berlangsung**:
   • Setelah memilih metode bayar di menu **/pos**, klik tombol **Selesaikan & Cetak Struk**.
   • Jendela dialog printer browser akan muncul secara otomatis.
3. **Kustomisasi Tampilan Struk**:
   • Anda dapat menambahkan logo toko, nama gerai, alamat, nomor telepon, dan pesan terima kasih pada bagian bawah struk.
   • Pengaturan ini berada di menu **Settings (/settings)** pada tab konfigurasi struk.
4. **Tips Mengatasi Cetakan Macet**:
   • Pastikan driver printer thermal sudah terpasang di sistem operasi komputer/tablet.
   • Pastikan opsi *Headers and Footers* bawaan browser di-nonaktifkan agar struk rapi dan hemat kertas.`,
  },

  // ── 3. SURAT JALAN & PENGIRIMAN OUTBOUND ──
  {
    id: 'delivery_order',
    title: 'Surat Jalan (Delivery Order / DO)',
    exactPhrases: ['cara buat surat jalan', 'cara buat do', 'cetak surat jalan', 'panduan delivery order'],
    primaryKeywords: [/\b(surat jalan|delivery order|\bdo\b|kirim barang|pengiriman barang)\b/i],
    secondaryKeywords: [/\b(sopir|driver|kurir|ekspedisi|plat nomor|resi|nomor do|tanda tangan)\b/i],
    negativeKeywords: [/\b(pos|struk|thermal|kasir)\b/i],
    priority: 7,
    getReply: () =>
      `**Panduan Pembuatan & Cetak Surat Jalan (DO) di Tayooli ERP:**

1. **Akses Menu**: Buka menu **Barang Masuk & Keluar (/wms/arus-barang)** dengan toggle **KELUAR**, atau langsung ke **Surat Jalan (/wms/delivery-orders)**.
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
   • Klik **Cetak Surat Jalan** untuk mencetak format standar bisnis Indonesia lengkap dengan Barcode/QR pelacak serta kolom tanda tangan Pengirim, Sopir, dan Penerima.`,
  },

  // ── 4. ARUS BARANG MASUK (INBOUND) & BATCH MANAGEMENT ──
  {
    id: 'inbound_workflow',
    title: 'Penerimaan Barang Masuk (Inbound)',
    exactPhrases: ['cara terima barang', 'cara barang masuk', 'alur inbound', 'barang masuk dari supplier'],
    primaryKeywords: [/\b(barang masuk|inbound|penerimaan barang|terima barang|pemasok|supplier|hasil produksi)\b/i],
    secondaryKeywords: [/\b(staging|putaway|bongkar|rak tujuan|dermaga|dock)\b/i],
    priority: 7,
    getReply: () =>
      `**Panduan Penerimaan Barang Masuk (Inbound) di Tayooli WMS:**

1. **Akses Menu**: Buka **Barang Masuk & Keluar (/wms/arus-barang)** pada tab **MASUK**, atau akses **/wms/inbound**.
2. **Buat Penerimaan Baru**: Klik tombol **+ Penerimaan Barang Masuk**.
3. **Pilih Sumber & Barang**:
   • Tentukan asal penerimaan: apakah dari Pemasok Luar (Vendor), Hasil Produksi Internal, atau Retur Konsumen.
   • Pilih SKU produk yang diterima dan jumlah kuantitas riil yang dibongkar dari muatan.
4. **Wajib Nomor Batch & Tanggal Expired**:
   • Sesuai aturan akurasi pergudangan (KO-1), setiap barang masuk wajib diberikan **Nomor Batch (Batch ID)** dan tanggal kedaluwarsa (Expired Date) jika ada.
5. **Alur 2-Step Inbound (Staging -> Putaway)**:
   • **Tahap Staging**: Barang diletakkan sementara di area dermaga penerimaan untuk diperiksa mutu fisiknya.
   • **Tahap Putaway**: Setelah lolos verifikasi, staf gudang memindahkan barang ke rak/bin penyimpanan permanen, dan stok langsung bertambah pada inventori aktif.`,
  },
  {
    id: 'batch_expiry_fefo',
    title: 'Manajemen Batch, Expired Date & FEFO',
    exactPhrases: ['bagaimana sistem batch', 'apa itu batch id', 'rotasi fefo', 'kadaluwarsa barang'],
    primaryKeywords: [/\b(batch|batch id|nomor batch|expired|kadaluwarsa|fefo|fifo)\b/i],
    secondaryKeywords: [/\b(lacak batch|tracking batch|mutasi batch|alur batch)\b/i],
    negativeKeywords: [/\b(scrap|afkir|barang cacat|pecah|bocor|write-?off|buang)\b/i],
    priority: 7,
    getReply: () =>
      `**Manajemen Batch & Rotasi Stok FEFO di Tayooli WMS:**

1. **Prinsip Invarian Batch (KO-1)**:
   • Di Tayooli, setiap mutasi stok (barang masuk, pengeluaran, transfer, maupun penjualan) **wajib terhubung dengan Batch ID**.
   • Hal ini menjamin ketertelusuran asal-usul barang jika terjadi komplain kualitas atau penarikan produk (product recall).
2. **Rotasi FEFO (First-Expired, First-Out)**:
   • Sistem memprioritaskan barang dengan tanggal kedaluwarsa paling dekat untuk diambil lebih dulu saat pengemasan pesanan atau transaksi kasir.
   • Mengurangi risiko kerugian akibat barang basi atau kadaluwarsa di dalam rak gudang.
3. **Pengecekan Batch**:
   • Anda dapat melihat riwayat pergerakan batch tertentu di modul **Gudang & Stok (/wms)** pada tab data batch inventori.`,
  },

  // ── 5. TRANSFER STOK ANTAR GUDANG ──
  {
    id: 'stock_transfers',
    title: 'Transfer Stok Antar Gudang',
    exactPhrases: ['cara transfer stok', 'pindah barang antar gudang', 'mutasi gudang', 'alur stock transfer'],
    primaryKeywords: [/\b(transfer stok|stock transfer|pindah barang|pindah gudang|antar gudang|mutasi gudang)\b/i],
    secondaryKeywords: [/\b(approval|in transit|gudang asal|gudang tujuan|otorisasi)\b/i],
    priority: 7,
    getReply: () =>
      `**Panduan Transfer Stok Antar Gudang di Tayooli WMS:**

1. **Buka Menu Transfer**: Masuk ke **Transfer Stok (/wms/transfers)** pada sidebar.
2. **Buat Pengajuan**: Klik **+ Buat Transfer**.
3. **Pilih Lokasi & Barang**:
   • Tentukan **Gudang Asal** (pengirim) dan **Gudang Tujuan** (penerima).
   • Masukkan daftar SKU, nomor batch, dan jumlah unit yang dipindahkan.
4. **Siklus Status Transfer**:
   • **Draft**: Draf perpindahan barang sedang dibuat.
   • **Pending Approval**: Menunggu persetujuan kepala gudang / supervisor.
   • **In Transit**: Barang sudah keluar dari gudang asal dan sedang dalam perjalanan kurir/armada.
   • **Received**: Barang telah tiba dan dihitung fisik oleh gudang tujuan.
5. **Konfirmasi & Pembaruan Saldo**:
   • Begitu gudang tujuan menekan tombol konfirmasi penerimaan, saldo gudang asal berkurang dan saldo gudang tujuan bertambah otomatis tanpa ada selisih.`,
  },

  // ── 6. STOK OPNAME & AUDIT FISIK ──
  {
    id: 'stock_opname',
    title: 'Stock Opname & Rekonsiliasi Varians',
    exactPhrases: ['cara stock opname', 'bagaimana opname', 'audit stok fisik', 'selisih stok fisik'],
    primaryKeywords: [/\b(stock opname|stok opname|opname|hitung fisik|audit stok|selisih stok|varians stok)\b/i],
    secondaryKeywords: [/\b(rekonsiliasi|adjustment|penyesuaian stok|stok tidak cocok|stok minus)\b/i],
    priority: 7,
    getReply: () =>
      `**Panduan Stock Opname (Penyesuaian Fisik) di Tayooli WMS:**

1. **Akses Menu**: Buka **Stok Opname (/wms/opname)**.
2. **Jadwalkan Sesi Opname**: Klik tombol **+ Buat Jadwal Opname Baru** dan tentukan gudang mana yang akan diaudit.
3. **Hitung Fisik Aktual**:
   • Tim lapangan menghitung jumlah fisik barang yang ada di rak/bin penyimpanan (dapat menggunakan barcode scanner).
   • Input angka fisik yang dihitung ke kolom penghitungan sistem.
4. **Kalkulasi Selisih (Varians)**:
   • Sistem secara otomatis memperlihatkan perbandingan: Stok Tercatat di Komputer vs Stok Nyata di Rak.
   • Muncul nilai selisih kuantitas (+ surplus / - defisit) beserta nilai rupiahnya.
5. **Terapkan Penyesuaian (Adjustment)**:
   • Supervisor meninjau alasan selisih, lalu mengklik **Terapkan Penyesuaian Stok (Adjust)**.
   • Saldo sistem akan otomatis diperbarui agar cocok 100% dengan fisik gudang dan tercatat di buku besar riwayat audit.`,
  },

  // ── 7. BARANG RUSAK & SCRAP ──
  {
    id: 'scrap_damaged',
    title: 'Barang Rusak / Afkir / Scrap',
    exactPhrases: ['cara catat barang rusak', 'barang cacat', 'cara scrap', 'write off stok'],
    primaryKeywords: [
      /\b(barang rusak|scrap|afkir|barang cacat|rusak|pecah|bocor|write-?off|expired barang)\b/i,
      /\b(pecah|afkir)\b/i,
    ],
    secondaryKeywords: [/\b(buang barang|alasan rusak|potong inventori)\b/i],
    priority: 8,
    getReply: () =>
      `**Panduan Pencatatan Barang Rusak / Scrap di Tayooli WMS:**

1. **Buka Menu Scrap**: Masuk ke **Barang Rusak / Scrap (/wms/scrap)**.
2. **Buat Laporan Scrap**: Klik **+ Catat Scrap Baru**.
3. **Pilih Barang & Keterangan**:
   • Pilih gudang asal dan SKU produk yang mengalami kerusakan.
   • Masukkan nomor batch dan kuantitas unit yang diafkir/rusak.
   • Pilih kategori alasan (misalnya: *Cacat Pabrik, Rusak Saat Bongkar Muat, Kadaluwarsa / Expired, atau Rusak Air/Hama*).
4. **Eksekusi Pemotongan**:
   • Setelah data disimpan, sistem langsung mengeluarkan barang tersebut dari inventori aktif sehingga tidak akan tertampil di kasir POS atau marketplace.
   • Kerugian barang rusak tercatat pada laporan performa gudang secara akurat.`,
  },

  // ── 8. BARCODE SCANNER & HARDWARE ──
  {
    id: 'barcode_scanner',
    title: 'Pemindai Barcode (Kamera & Hardware Gun)',
    exactPhrases: ['cara pakai scanner', 'kamera barcode', 'barcode scanner gun', 'cara scan barcode'],
    primaryKeywords: [/\b(barcode scanner|pemindai barcode|kamera scanner|scanner gun|scan barcode|alat scan)\b/i],
    secondaryKeywords: [/\b(kamera hp|webcam|usb barcode|bluetooth scanner|ean-13|code 128|qr code)\b/i],
    priority: 7,
    getReply: () =>
      `**Panduan Barcode Scanner di Tayooli ERP:**

Tayooli mendukung dua metode pemindaian barcode yang fleksibel:
1. **Pemindai Kamera Bawaan (Webcam / Smartphone)**:
   • Buka menu **Barcode Scanner (/wms/scanner)** atau klik ikon kamera di halaman Kasir POS dan WMS.
   • Izinkan akses kamera pada peramban (browser).
   • Arahkan kamera ke barcode EAN-13, Code 128, atau QR Code produk. Sistem dilengkapi *visual pulse feedback* hijau saat barcode berhasil terbaca.
2. **Hardware Barcode Scanner Gun (USB / Bluetooth)**:
   • Cukup hubungkan barcode scanner fisik Anda ke komputer/laptop.
   • Scanner akan berfungsi sebagai keyboard input otomatis (**HID Mode**) tanpa perlu instalasi aplikasi tambahan.
   • Di kasir POS, Anda cukup menembakkan scanner ke produk, item akan otomatis masuk ke keranjang dalam sekejap.`,
  },

  // ── 9. MARKETPLACE OMNICHANNEL ──
  {
    id: 'marketplace_sync',
    title: 'Marketplace Omnichannel & Sinkronisasi Stok',
    exactPhrases: ['cara sinkron marketplace', 'integrasi shopee', 'integrasi tokopedia', 'integrasi tiktok', 'cegah overselling'],
    primaryKeywords: [/\b(marketplace|omnichannel|tokopedia|shopee|tiktok shop|tiktok|lazada|sinkronisasi stok|sync marketplace)\b/i],
    secondaryKeywords: [/\b(alokasi stok|cegah overselling|pesanan online|multi channel)\b/i],
    priority: 7,
    getReply: () =>
      `**Panduan Marketplace Omnichannel di Tayooli ERP:**

1. **Akses Modul**: Masuk ke menu **Marketplace (/wms/marketplace)**.
2. **Hubungkan Saluran Toko**:
   • Hubungkan akun toko online Anda (Tokopedia, Shopee, TikTok Shop, Lazada).
3. **Mekanisme Sinkronisasi Stok 2-Arah**:
   • **Penjualan Kasir POS Fisik**: Saat ada transaksi di kasir outlet, saldo stok di seluruh marketplace otomatis berkurang seketika.
   • **Pesanan Online Masuk**: Begitu pesanan baru terbit dari Shopee/Tokopedia, stok gudang Tayooli langsung dialokasikan (*reserved*) agar tidak terjual kembali ke orang lain.
4. **Pencegahan Overselling**:
   • Sistem mengunci stok pesanan aktif sehingga toko online Anda terbebas dari penalti pembatalan pesanan akibat stok habis (*out of stock*).`,
  },

  // ── 10. MASTER PRODUK & KATALOG ──
  {
    id: 'products_master',
    title: 'Master Produk & Katalog SKU',
    exactPhrases: ['cara tambah produk', 'cara buat produk', 'tambah barang baru', 'daftar produk baru'],
    primaryKeywords: [/\b(tambah produk|master produk|katalog produk|daftar barang|buat produk baru|sku produk)\b/i],
    secondaryKeywords: [/\b(harga beli|harga jual|hpp|stok minimal|minimum stok|barcode produk)\b/i],
    priority: 6,
    getReply: () =>
      `**Panduan Menambahkan Master Produk di Tayooli ERP:**

1. **Buka Menu**: Masuk ke menu **Products (/products)** pada navigasi.
2. **Klik Tombol**: Tekan tombol **+ Tambah Produk**.
3. **Lengkapi Informasi Barang**:
   • **Kode SKU**: Kode unik unik pengenal barang (contoh: \`SKU-KAOS-001\`).
   • **Nama Produk**: Nama lengkap produk komersial.
   • **Harga Beli (HPP)** & **Harga Jual** (harga ritel kasir).
   • **Stok Minimum (Alert)**: Batas ambang stok agar muncul peringatan di dashboard saat stok menipis.
   • **Kode Barcode**: Masukkan angka barcode dari kemasan pabrik (atau scan langsung).
4. **Simpan**: Klik **Simpan**. Produk akan langsung siap digunakan di Kasir POS, Gudang WMS, dan etalase penjualan.`,
  },

  // ── 11. TROUBLESHOOTING & KENDALA TEKNIS ──
  {
    id: 'troubleshooting_camera',
    title: 'Troubleshooting: Izin Kamera Scanner',
    exactPhrases: ['kamera tidak bisa dibuka', 'scanner tidak bisa buka kamera', 'kamera error', 'izin kamera ditolak'],
    primaryKeywords: [/\b(kamera.*(tidak|gagal|error|macet|ditolak)|izin kamera|permission denied)\b/i],
    secondaryKeywords: [/\b(browser|https|chrome|safari)\b/i],
    priority: 8,
    getReply: () =>
      `**Solusi Kamera Barcode Scanner Tidak Bisa Dibuka:**

1. **Pastikan Menggunakan Protokol HTTPS**:
   • Peramban modern (Chrome, Safari, Edge) mewajibkan koneksi aman (HTTPS) untuk mengizinkan akses kamera web.
2. **Cek Izin Kamera di Browser**:
   • Klik ikon gembok/setelan di samping bilah alamat URL (address bar) browser Anda.
   • Pastikan setelan **Camera** berada pada status **Allow (Izinkan)**.
3. **Tutup Aplikasi Lain**:
   • Pastikan kamera tidak sedang dipakai oleh aplikasi lain (seperti Zoom, Google Meet, atau aplikasi kamera bawaan OS).
4. **Alternatif Terbaik**:
   • Jika digunakan di PC kasir, kami sangat menyarankan memakai **Hardware Barcode Gun (USB)** yang dapat langsung dicolok tanpa butuh izin kamera.`,
  },
  {
    id: 'troubleshooting_printer',
    title: 'Troubleshooting: Printer Thermal Tidak Merespons',
    exactPhrases: ['printer tidak bisa cetak', 'printer tidak merespon', 'printer macet', 'struk tidak keluar', 'printer tidak merespons'],
    primaryKeywords: [
      /\b(printer|struk).*(tidak|gagal|error|macet|rusak|mati|merespons|respon)\b/i,
      /\b(printer (mati|error|rusak|macet|tidak keluar|tidak terdeteksi))\b/i,
      /\b(printer.*macet|printer.*tidak merespons|printer.*tidak respon)\b/i,
    ],
    secondaryKeywords: [/\b(thermal|kabel usb|bluetooth|paper cut)\b/i],
    priority: 10,
    getReply: () =>
      `**Solusi Printer Thermal Struk Tidak Merespons:**

1. **Periksa Kabel & Daya**:
   • Pastikan lampu indikator daya printer menyala (warna hijau/biru) dan tidak berkedip merah (indikator kehabisan kertas).
   • Pastikan gulungan kertas thermal terpasang pada arah yang benar (sisi berselaput kimia menghadap pemanas cetak).
2. **Uji Print Test dari Sistem Operasi**:
   • Masuk ke setelan Printer OS (Windows / macOS) dan lakukan *Print Test Page*.
   • Jika test page berhasil keluar, berarti koneksi fisik sudah aman.
3. **Pengaturan Dialog Browser**:
   • Saat dialog cetak muncul di browser, pastikan tujuan (*Destination*) mengarah ke nama printer thermal Anda, bukan *Save as PDF*.
   • Set ukuran kertas sesuai tipe printer Anda: **58mm (Roll Paper)** atau **80mm (Roll Paper)**.`,
  },
  {
    id: 'troubleshooting_stock',
    title: 'Troubleshooting: Stok Selisih / Tidak Terpotong',
    exactPhrases: ['kenapa stok tidak berkurang', 'stok tidak sinkron', 'stok minus', 'kenapa stok berbeda'],
    primaryKeywords: [/\b(stok.*(tidak berkurang|selisih|beda|minus|salah|tidak cocok)|kenapa stok)\b/i],
    secondaryKeywords: [/\b(kasir|opname|sync)\b/i],
    priority: 8,
    getReply: () =>
      `**Solusi Kendala Stok Selisih atau Tidak Terpotong di Tayooli:**

1. **Periksa Outlet / Gudang yang Dipilih**:
   • Di Kasir POS, pastikan Anda memilih gudang/outlet yang tepat saat memulai sesi kasir. Transaksi kasir hanya memotong stok di gudang yang aktif dipilih.
2. **Cek Riwayat Mutasi Buku Besar**:
   • Buka menu **Gudang & Stok (/wms)** untuk meninjau riwayat mutasi SKU tersebut.
   • Periksa apakah ada pesanan marketplace yang berstatus *Reserved* (terkunci) atau transfer stok yang masih berstatus *In Transit*.
3. **Lakukan Rekonsiliasi via Stok Opname**:
   • Jika ada barang fisik hilang atau rusak yang belum tercatat, buka menu **Stok Opname (/wms/opname)**, hitung ulang fisik, dan lakukan *Adjustment* agar angka sistem dan rak kembali serasi 100%.`,
  },
  {
    id: 'settings_account',
    title: 'Pengaturan Usaha, Pajak & Akun',
    exactPhrases: ['cara ganti password', 'atur pajak ppn', 'ganti logo toko', 'pengaturan usaha'],
    primaryKeywords: [/\b(pengaturan|settings|ubah password|ganti kata sandi|profil toko|logo toko|pajak ppn)\b/i],
    secondaryKeywords: [/\b(akun saya|alamat toko|informasi bisnis)\b/i],
    priority: 6,
    getReply: () =>
      `**Panduan Pengaturan Sistem & Profil Bisnis:**

Buka menu **Settings (/settings)** pada sidebar untuk menyesuaikan:
• **Profil Usaha**: Nama bisnis, nomor telepon resmi, email, alamat fisik toko/kantor, serta upload logo perusahaan untuk dokumen resmi.
• **Preferensi Pajak & Mata Uang**: Aktifkan tarif PPN (misal: 11%) agar otomatis terhitung di struk kasir dan faktur penjualan.
• **Format Struk Kasir**: Masukkan teks sambutan di bagian atas (*Header*) dan ucapan terima kasih di bagian bawah (*Footer*) struk belanja.
• **Keamanan Akun**: Perbarui kata sandi secara berkala untuk menjaga keamanan data usaha Anda.`,
  },
  {
    id: 'legacy_accounting_faq',
    title: 'Informasi Modul Akuntansi & Fokus Sistem',
    exactPhrases: ['jurnal umum', 'chart of accounts', 'buku kas masuk keluar', 'laporan laba rugi akuntansi', 'coa'],
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
 * Finds the best matching knowledge topic using weighted scoring
 */
export function findBestTopic(query: string): MatchResult | null {
  const normalized = normalizeQuery(query)
  if (!normalized) return null

  let bestResult: MatchResult | null = null

  for (const topic of TAYOOLI_KNOWLEDGE_TOPICS) {
    let score = 0
    const matched: string[] = []

    // 1. Check Exact Phrases (weight: 25)
    if (topic.exactPhrases) {
      for (const phrase of topic.exactPhrases) {
        if (normalized === phrase || normalized.includes(phrase)) {
          score += 25
          matched.push(`exact:${phrase}`)
        }
      }
    }

    // 2. Check Negative Keywords (penalty: -20)
    if (topic.negativeKeywords) {
      for (const neg of topic.negativeKeywords) {
        if (neg.test(normalized)) {
          score -= 20
        }
      }
    }

    // 3. Check Primary Keywords (weight: 12)
    for (const rx of topic.primaryKeywords) {
      if (rx.test(normalized)) {
        score += 12
        matched.push(rx.source)
      }
    }

    // 4. Check Secondary Keywords (weight: 4)
    for (const rx of topic.secondaryKeywords) {
      if (rx.test(normalized)) {
        score += 4
        matched.push(rx.source)
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
  const isPrintQuery = /\b(cetak|mencetak|print|ngeprint)\b/i.test(norm)
  const hasReceiptContext = /\b(struk|kasir|\bpos\b|nota|belanja|thermal|printer)\b/i.test(norm)
  const hasDoContext = /\b(surat jalan|delivery order|\bdo\b|pengiriman|ekspedisi|jalan|kirim)\b/i.test(norm)

  if (isPrintQuery && !hasReceiptContext && !hasDoContext) {
    return `Halo kak! Mengenai pencetakan dokumen di **Tayooli ERP**, sistem mendukung dua jenis dokumen utama:

1. **Struk Belanja Kasir (POS)**:
   • Dicetak langsung ke printer thermal (lebar 58mm / 80mm) di menu **Point of Sale (/pos)** saat transaksi selesai.
2. **Surat Jalan (Delivery Order / DO)**:
   • Dicetak berformat dokumen resmi lengkap dengan barcode tracking dan kolom tanda tangan di menu **Barang Masuk & Keluar (/wms/arus-barang)** atau **/wms/delivery-orders**.

Apakah yang ingin Anda cetak saat ini adalah **Struk Kasir** atau **Surat Jalan**? Silakan beri tahu kami agar kami bisa memberikan panduan yang tepat!`
  }

  // 2. Ambiguous item flow / mutasi query
  const isGeneralFlow = /\b(alur barang|proses barang|kelola barang|mutasi barang|manajemen barang)\b/i.test(norm)
  const hasSpecificFlow = /\b(masuk|keluar|inbound|outbound|transfer|opname|scrap|rusak|afkir|supplier)\b/i.test(norm)

  if ((isGeneralFlow || norm === 'barang' || norm === 'mutasi') && !hasSpecificFlow) {
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
  return `Halo kak! Terima kasih atas pertanyaannya.

Pertanyaan Anda mengenai *"**${query.trim()}**"* dapat kami bantu dengan lebih tepat jika Anda memilih topik berikut:
• **Kasir POS (/pos)**: Alur transaksi penjualan, barcode scanner, dan cetak struk thermal
• **Arus Barang (/wms/arus-barang)**: Penerimaan barang masuk (Inbound) atau Surat Jalan pengiriman (DO)
• **Transfer Stok (/wms/transfers)**: Mutasi barang antar lokasi gudang dengan approval
• **Stok Opname (/wms/opname)**: Pemeriksaan stok fisik dan penyesuaian selisih
• **Marketplace (/wms/marketplace)**: Sinkronisasi stok otomatis toko online
• **Master Produk (/products)**: Pendaftaran SKU barang dan harga jual/beli

Bisa ceritakan sedikit lebih detail kendala atau alur apa yang ingin Anda tanyakan, kak? Kami siap membantu!`
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
  if (match && match.score >= 12) {
    return match.topic.getReply(trimmed)
  }

  // Fallback
  return generateSmartFallback(trimmed)
}
