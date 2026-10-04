package knowledge

// Document represents a knowledge base document.
type Document struct {
	ID      string
	Title   string
	Content string
	Tags    []string
}

// GetAllDocuments returns all knowledge base documents.
func GetAllDocuments() []Document {
	return []Document{
		// ── Produk ──────────────────────────────────────────────────────────
		{
			ID:    "produk-overview",
			Title: "Tayooli ERP Overview",
			Content: `Tayooli adalah sistem ERP B2B lengkap untuk bisnis Indonesia. 
Fitur utama: invoice management (OCR + AI), purchase order, goods receipt, payment order, approval workflow, vendor management, customer management, sales order, sales invoice, accounting (chart of accounts + journal entries), products & inventory, Warehouse Management System (WMS multi-gudang, stock transfer, stock opname, barang rusak/scrap, barcode scanner, surat jalan/delivery order), integrasi marketplace omnichannel (Shopee/Tokopedia/TikTok), dan Point of Sale (POS / kasir toko).
Tayooli dirancang untuk mengintegrasikan proses procure-to-pay, order-to-cash, pergudangan modern, dan operasional kasir dalam satu platform.`,
			Tags: []string{"produk", "overview", "tentang", "tayooli", "erp", "fitur", "wms", "gudang", "pos", "kasir"},
		},
		{
			ID:    "produk-fitur-wms",
			Title: "Warehouse & Logistics (WMS)",
			Content: `Fitur Warehouse Management System (WMS) Tayooli:
1. Multi-Gudang & Lokasi: kelola beberapa gudang fisik (pusat, cabang, transit) dan lokasi spesifik (rak, bin, baris). Tipe lokasi meliputi: Internal (stok tersedia), Transit (pengiriman antar-gudang), Loss (selisih hilang), Scrap (barang rusak/afkir).
2. Surat Jalan (Delivery Orders / DO): pembuatan surat jalan pengiriman dari Sales Order dengan alur status: DRAFT -> CONFIRMED -> PICKED -> PACKED -> IN_TRANSIT -> DELIVERED. Mengurangi stok gudang secara otomatis saat barang dikirim.
3. Stock Transfers (Transfer Antar Gudang): mutasi persediaan antar gudang atau lokasi bin dengan sistem approval berjenjang. Alur: DRAFT -> PENDING_APPROVAL -> APPROVED -> DISPATCHED -> IN_TRANSIT -> RECEIVED. Mencegah selisih pengiriman barang.
4. Stock Opname (Penyesuaian Stok Fisik): audit dan penghitungan fisik stok berkala dengan pencatatan selisih (discrepancy). Alur: DRAFT -> IN_PROGRESS -> COMPLETED -> CANCELLED. Sistem secara otomatis membuat penyesuaian stok dan jurnal koreksi inventaris.
5. Barang Rusak / Scrap: pencatatan barang rusak, kedaluwarsa, atau cacat dari gudang ke lokasi scrap dengan dokumentasi alasan (kerusakan handling, kedaluwarsa, cacat pabrik). Mengurangi stok aktif dan mencatat beban kerugian.
6. Barcode Scanner: pemindai barcode / QR menggunakan kamera ponsel/laptop maupun alat scanner fisik (USB/Bluetooth) untuk pencarian produk cepat, verifikasi nomor seri/SKU, dan validasi pergerakan barang.
Cara akses: Menu navigasi Warehouse & POS di sidebar (/wms, /wms/transfers, /wms/opname, /wms/scrap, /wms/scanner, /wms/delivery-orders).`,
			Tags: []string{"warehouse", "gudang", "wms", "lokasi", "bin", "surat jalan", "delivery order", "transfer stok", "mutasi stok", "stock opname", "opname", "scrap", "barang rusak", "barcode", "scanner", "logistik"},
		},
		{
			ID:    "produk-fitur-pos",
			Title: "Point of Sale (POS / Kasir)",
			Content: `Fitur Point of Sale (POS / Kasir Toko) Tayooli:
1. Kasir Modern & Cepat: antarmuka kasir responsif untuk transaksi ritel toko, minimarket, atau outlet langsung.
2. Barcode Scanning: input item belanja dengan scan barcode menggunakan scanner fisik atau kamera perangkat, otomatis masuk ke keranjang belanja.
3. Keranjang & Manajemen Item: atur kuantitas (tambah/kurang), hapus item, dan hitung subtotal serta pajak/diskon seketika.
4. Mode Penjualan Fleksibel:
   - Jual Putus: barang milik toko sendiri, keuntungan/margin langsung milik perusahaan.
   - Konsinyasi: barang titipan vendor/mitra, pencatatan otomatis vendor konsinyasi untuk kemudahan bagi hasil dan rekonsiliasi.
5. Multi-Metode Pembayaran:
   - Tunai (Cash): input uang bayar pelanggan, sistem menghitung uang kembalian secara otomatis.
   - QRIS Dinamis: generate QRIS instan di layar untuk dipindai pelanggan menggunakan e-wallet (GoPay, OVO, Dana) atau mobile banking.
   - Transfer Bank / Kartu Debit.
6. Cetak Struk: mendukung pencetakan nota/struk belanja instan ke printer thermal (58mm/80mm) atau unduh struk digital.
7. Real-Time Inventory Sync: setiap transaksi kasir yang selesai langsung memotong stok produk di gudang toko yang aktif, mencegah selisih stok fisik vs sistem.
Cara akses: Buka menu Point of Sale di sidebar (/pos).`,
			Tags: []string{"pos", "kasir", "point of sale", "barcode", "struk", "cetak struk", "qris", "tunai", "cash", "kembalian", "konsinyasi", "jual putus", "toko", "ritel"},
		},
		{
			ID:    "produk-fitur-wms-marketplace",
			Title: "Marketplace Omnichannel Integration",
			Content: `Fitur Marketplace Omnichannel Tayooli:
1. Multi-Platform: integrasi dan sinkronisasi pesanan dari e-commerce terkemuka di Indonesia (Shopee, Tokopedia, TikTok Shop, Lazada).
2. Import Batch Pesanan: unggah laporan pesanan marketplace (CSV/Excel) untuk pemrosesan otomatis ratusan pesanan sekaligus.
3. Pemetaan SKU (SKU Mapping): petakan SKU dari marketplace ke master SKU produk internal Tayooli sehingga stok terpotong akurat tanpa duplikasi produk.
4. Status Integrasi: deteksi pesanan duplikat, verifikasi total nominal, dan otomatisasi pembuatan sales order/invoice dari transaksi marketplace.
Cara akses: Buka menu Warehouse & Stock -> Marketplace Omnichannel (/wms/marketplace).`,
			Tags: []string{"marketplace", "omnichannel", "shopee", "tokopedia", "tiktok", "lazada", "impor pesanan", "sku mapping", "sync stok"},
		},
		{
			ID:    "produk-fitur-invoice",
			Title: "Invoice Management",
			Content: `Fitur Invoice Tayooli:
1. Buat Invoice: masukkan data vendor, jumlah, tanggal jatuh tempo
2. Upload Foto: unggah foto/foto invoice untuk OCR otomatis
3. OCR: sistem akan ekstrak teks dari gambar secara otomatis
4. Review: periksa data hasil OCR sebelum disimpan
5. Approve: approver bisa approve atau reject invoice
6. Bayar: setelah approve, buat payment order untuk pembayaran
Status invoice: pending, approved, rejected, processing, pending_review, ai_processed, ai_failed`,
			Tags: []string{"invoice", "ocr", "upload", "foto", "approve", "bayar", "status"},
		},
		{
			ID:    "produk-fitur-po",
			Title: "Purchase Order",
			Content: `Fitur Purchase Order (PO):
1. Buat PO: pilih vendor, tambahkan item produk, jumlah, harga
2. Kirim PO: kirim ke vendor
3. Vendor kirim barang
4. Goods Receipt: terima barang, catat quantities yang diterima
5. Payment Order: buat permintaan pembayaran ke vendor
Status PO: draft, sent, partially_received, received, cancelled`,
			Tags: []string{"purchase order", "po", "vendor", "barang", "kirim"},
		},
		{
			ID:    "produk-fitur-approval",
			Title: "Approval Workflow",
			Content: `Fitur Approval Workflow:
- Role-based approval: admin, accountant, approver, treasury, cfo, purchaser, warehouse
- Invoice approval: approver bisa approve atau reject invoice dengan alasan
- Payment approval: cfo atau treasury approve payment order
- Multi-step approval untuk transaksi besar
Cara approve: buka halaman Approvals → lihat pending approvals → klik Approve atau Reject
Cara reject: pilih invoice/PO → klik Reject → masukkan alasan rejection`,
			Tags: []string{"approval", "approve", "reject", "workflow", "role", "persetujuan"},
		},
		{
			ID:    "produk-fitur-payment",
			Title: "Payment Orders",
			Content: `Fitur Payment Orders:
1. Buat Payment Order: pilih invoice yang akan dibayar
2. Pilih metode pembayaran
3. Submit untuk approval
4. CFO/Treasury approve
5. Proses pembayaran
Status: pending_approval, approved, paid, rejected
Integrasi payment gateway: Midtrans untuk pembayaran customer, Pakasir untuk subscription billing`,
			Tags: []string{"payment", "pembayaran", "payment order", "bayar", "metode"},
		},
		{
			ID:    "produk-fitur-vendor",
			Title: "Vendor Management",
			Content: `Fitur Vendor Management:
- Tambah vendor: masukkan nama, kode vendor, kontak, alamat
- Edit vendor: update informasi vendor
- Hapus vendor: admin bisa hapus vendor
- Lihat vendor: daftar semua vendor dengan detail
- Invoice per vendor: lihat semua invoice dari vendor tertentu
Data vendor tersimpan di database dan bisa diakses untuk pembuatan PO dan invoice.`,
			Tags: []string{"vendor", "supplier", "data", "kelola", "tambah", "edit"},
		},
		{
			ID:    "produk-fitur-accounting",
			Title: "Accounting",
			Content: `Fitur Accounting:
- Chart of Accounts (CoA): buat dan kelola struktur akun
- Journal Entries: catat transaksi akuntansi
- Tipe akun: Asset, Liability, Equity, Revenue, Expense
- Setiap journal entry punya debit dan credit yang harus balance
- Status journal entry: draft, posted, voided
Cara buat journal entry: Accounting → Journal Entries → Create → isi tanggal, referensi, deskripsi, tambahkan lines (akun + debit/credit)`,
			Tags: []string{"accounting", "akuntansi", "journal", "akun", "coa", "debit", "credit"},
		},
		{
			ID:    "produk-fitur-product",
			Title: "Products & Inventory",
			Content: `Fitur Products & Inventory:
- Tambah produk: nama, SKU, deskripsi, harga
- Kelola inventory: jumlah stok, lokasi gudang
- Adjust inventory: tambah atau kurangi stok
- Produk digunakan di PO dan Sales Order
SKU harus unik per produk.`,
			Tags: []string{"product", "produk", "inventory", "stok", "sku", "gudang"},
		},
		{
			ID:    "produk-fitur-customer",
			Title: "Customer & Sales",
			Content: `Fitur Order-to-Cash:
- Customers: kelola data customer
- Sales Orders: buat pesanan dari customer
- Sales Invoices: buat tagihan untuk customer
- Integrasi payment gateway: customer bisa bayar via Midtrans
Proses: Customer order → Sales Order → Sales Invoice → Payment → Selesai`,
			Tags: []string{"customer", "sales", "order", "invoice", "penjualan", "pelanggan"},
		},
		{
			ID:    "produk-fitur-dashboard",
			Title: "Dashboard",
			Content: `Dashboard Tayooli:
- Ringkasan keuangan dan operasional
- KPI: Total Invoice, Disetujui, Menunggu Persetujuan, Vendor Aktif, Purchase Orders, Goods Receipts
- Status Invoice: donut chart menunggu/disetujui/ditolak/review AI
- Pipeline Pembayaran: total payment orders, sudah dibayar, pending
- Trend Invoice 6 bulan terakhir
- Top 5 Vendor berdasarkan nilai invoice
- Invoice terbaru`,
			Tags: []string{"dashboard", "ringkasan", "kpi", "chart", "trend"},
		},
		// ── Pricing ─────────────────────────────────────────────────────────
		{
			ID:    "pricing-overview",
			Title: "Pricing Plans",
			Content: `Paket Langganan Tayooli:
1. Trial: Gratis 14 hari, semua fitur Bisnis
2. Starter: Rp199.000/bulan atau Rp1.592.000/tahun (hemat 20%)
   - 5 pengguna
   - 100 vendor
   - 500 invoice/bulan
   - OCR 200/bulan
3. Bisnis: Rp449.000/bulan atau Rp3.592.000/tahun (hemat 20%)
   - 25 pengguna
   - 500 vendor
   - 2.000 invoice/bulan
   - OCR 1.000/bulan
   - Multi-entitas
   - API akses
4. Enterprise: Custom (hubungi sales)
   - Pengguna tanpa batas
   - Vendor tanpa batas
   - Invoice tanpa batas
   - OCR tanpa batas
   - Multi-entitas
   - API akses
   - Dedicated support`,
			Tags: []string{"pricing", "harga", "paket", "starter", "bisnis", "enterprise", "trial", "langganan", "subscription"},
		},
		{
			ID:    "pricing-billing",
			Title: "Billing & Payment",
			Content: `Cara kelola billing:
- Buka halaman Billing di sidebar (menu Account → Billing)
- Lihat paket saat ini dan status langganan
- Upgrade paket: pilih paket baru → bayar via Pakasir
- Lihat riwayat tagihan di halaman Billing
- Pembayaran via Pakasir (payment gateway Indonesia)
- Trial 14 hari otomatis aktif saat daftar
- Upgrade kapan saja selama trial`,
			Tags: []string{"billing", "bayar", "tagihan", "upgrade", "paketasir", "payment"},
		},
		// ── Troubleshooting ──────────────────────────────────────────────────
		{
			ID:    "troubleshooting-login",
			Title: "Login Issues",
			Content: `Masalah Login:
1. Email atau password salah: pastikan email dan password benar
2. Akun tidak ditemukan: hubungi admin untuk membuat akun
3. Session expired: login ulang dengan email dan password
4. Lupa password: hubungi admin untuk reset password
Demo account: admin@test.com / password123`,
			Tags: []string{"login", "gagal", "password", "email", "session", "akun"},
		},
		{
			ID:    "troubleshooting-invoice",
			Title: "Invoice Issues",
			Content: `Masalah Invoice:
1. Invoice tidak muncul: cek filter status di halaman Invoices
2. OCR tidak akurat: pastikan foto invoice jelas dan terbaca
3. Invoice stuck di pending: cek apakah ada approver yang belum approve
4. Invoice ditolak: periksa alasan rejection, perbaiki data, submit ulang
5. Duplicate invoice: cek nomor invoice sebelum membuat baru`,
			Tags: []string{"invoice", "tidak muncul", "ocr", "stuck", "pending", "ditolak", "duplikat"},
		},
		{
			ID:    "troubleshooting-approval",
			Title: "Approval Issues",
			Content: `Masalah Approval:
1. Approval tidak muncul: pastikan kamu punya role approver
2. Approval stuck: hubungi admin untuk cek workflow
3. Salah approve: hubungi admin untuk reversal
4. Tidak bisa approve: cek role dan permission kamu
Role yang bisa approve: admin, approver (untuk invoice), cfo/treasury (untuk payment)`,
			Tags: []string{"approval", "persetujuan", "stuck", "tidak muncul", "role", "permission"},
		},
		{
			ID:    "troubleshooting-payment",
			Title: "Payment Issues",
			Content: `Masalah Pembayaran:
1. Payment belum masuk: cek status di Payment Orders
2. Payment ditolak: periksa alasan, perbaiki data
3. Payment pending approval: tunggu cfo/treasury approve
4. Refund: hubungi admin
Integrasi: Midtrans untuk customer payment, Pakasir untuk subscription`,
			Tags: []string{"payment", "pembayaran", "belum masuk", "ditolak", "pending", "refund"},
		},
		{
			ID:    "troubleshooting-performance",
			Title: "Performance Issues",
			Content: `Masalah Performa:
1. Halaman lambat: cek koneksi internet
2. Upload gagal: pastikan ukuran file < 10MB
3. OCR lambat: server mungkin sedang load tinggi, tunggu sebentar
4. Chart tidak muncul: refresh halaman
5. Data tidak update: tunggu beberapa detik untuk auto-refresh`,
			Tags: []string{"performa", "lambat", "loading", "gagal", "upload", "chart"},
		},
		// ── Workflow ─────────────────────────────────────────────────────────
		{
			ID:    "workflow-procure-to-pay",
			Title: "Procure-to-Pay Workflow",
			Content: `Alur Procure-to-Pay:
1. Buat Purchase Order (PO) → pilih vendor, item, jumlah
2. Kirim PO ke vendor
3. Vendor kirim barang
4. Goods Receipt → terima barang, catat quantities
5. Invoice → vendor kirim invoice, upload ke sistem
6. OCR → sistem ekstrak data invoice otomatis
7. Review → periksa data hasil OCR
8. Approve → approver approve invoice
9. Payment Order → buat permintaan pembayaran
10. Approve Payment → cfo/treasury approve
11. Bayar → proses pembayaran ke vendor`,
			Tags: []string{"procure", "pay", "workflow", "alur", "po", "gr", "invoice", "payment"},
		},
		{
			ID:    "workflow-order-to-cash",
			Title: "Order-to-Cash Workflow",
			Content: `Alur Order-to-Cash:
1. Customer hubungi sales
2. Buat Sales Order → detail pesanan
3. Buat Sales Invoice → tagihan ke customer
4. Customer bayar → via Midtrans atau transfer
5. Payment terverifikasi
6. Selesai
Data customer tersimpan untuk order berikutnya.`,
			Tags: []string{"order", "cash", "workflow", "alur", "sales", "customer", "midtrans"},
		},
		// ── FAQ ──────────────────────────────────────────────────────────────
		{
			ID:    "faq-umum",
			Title: "FAQ Umum",
			Content: `Pertanyaan Umum:
Q: Apakah Tayooli bisa dipakai gratis?
A: Ya, trial 14 hari gratis tanpa kartu kredit.

Q: Bagaimana cara upgrade paket?
A: Buka halaman Billing → pilih paket → bayar via Pakasir.

Q: Apakah ada batasan pengguna?
A: Tergantung paket. Starter: 5, Bisnis: 25, Enterprise: unlimited.

Q: Bagaimana cara menghubungi support?
A: Gunakan chat support di pojok kanan bawah. AI support kami siap membantu 24/7.

Q: Apakah data saya aman?
A: Ya, Tayooli menggunakan enkripsi dan isolasi data per tenant.`,
			Tags: []string{"faq", "pertanyaan", "gratis", "upgrade", "support", "aman", "keamanan"},
		},
	}
}
