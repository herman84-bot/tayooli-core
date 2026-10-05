import { NextRequest, NextResponse } from 'next/server'

interface ChatRequestBody {
  message?: string
  session_id?: string
}

const TAYOOLI_KNOWLEDGE: Array<{
  keywords: RegExp
  reply: string
}> = [
  {
    keywords: /(kasir|pos|point of sale|struk|nota|printer|thermal|cetak)/i,
    reply: `**Panduan Kasir POS & Cetak Struk di Tayooli ERP:**

1. **Akses Menu POS**: Klik menu **Point of Sale** (\`/pos\`) pada sidebar.
2. **Pilih Produk**: Cari barang dari katalog, klik produk, atau gunakan barcode scanner untuk memasukkan item ke keranjang belanja.
3. **Atur Transaksi**: Anda dapat menyesuaikan jumlah kuantitas, menambahkan diskon, atau memilih pelanggan.
4. **Pembayaran**: Pilih metode pembayaran (Tunai / Cash, QRIS, Debit, atau Transfer). Masukkan jumlah uang tunai yang diterima untuk menghitung kembalian otomatis.
5. **Cetak Struk**: Klik **Selesaikan Transaksi & Cetak**. Sistem mendukung printer thermal ESC/POS (ukuran 58mm maupun 80mm) dan browser standard print.
6. **Pemotongan Stok**: Stok fisik barang di gudang otomatis terpotong secara real-time begitu transaksi dinyatakan berhasil.`,
  },
  {
    keywords: /(transfer|pindah|antar gudang|transit)/i,
    reply: `**Panduan Transfer Stok Antar Gudang di Tayooli WMS:**

1. **Buka Modul**: Pilih menu **Stock Transfers** (\`/wms/transfers\`).
2. **Buat Transfer Baru**: Klik tombol **+ Buat Transfer**.
3. **Pilih Lokasi & Barang**: Tentukan Gudang Asal, Gudang Tujuan, serta daftar SKU beserta jumlah unit yang dipindahkan.
4. **Siklus Status Transfer**:
   - **Draft**: Draf perpindahan barang sedang disusun.
   - **Pending Approval**: Menunggu otorisasi pimpinan/supervisor gudang.
   - **In Transit**: Barang telah keluar dari gudang asal dan dalam perjalanan pengiriman.
   - **Received**: Barang telah tiba dan diverifikasi di gudang tujuan.
5. **Otomasi Saldo**: Saat gudang tujuan menekan tombol konfirmasi penerimaan, saldo stok di gudang asal berkurang dan saldo gudang tujuan bertambah otomatis tanpa selisih.`,
  },
  {
    keywords: /(surat jalan|delivery order|\bdo\b|pengiriman|kirim barang|ekspedisi)/i,
    reply: `**Panduan Surat Jalan (Delivery Order / DO) di Tayooli ERP:**

1. **Buka Menu**: Pilih **Surat Jalan (DO)** (\`/wms/delivery-orders\`).
2. **Terbitkan Surat Jalan**: Klik **+ Terbitkan Surat Jalan**.
3. **Isi Informasi Pengiriman**:
   - Pilih nama penerima / pelanggan dan alamat tujuan pengiriman.
   - Tentukan tanggal pengiriman dan armada/kurir (nama driver & plat nomor / nomor resi).
   - Masukkan daftar SKU barang dan jumlah kuantitas fisik yang dikirimkan.
4. **Cetak Dokumen Resmi**: Klik **Cetak Surat Jalan**. Dokumen sudah diformat standar resmi bisnis Indonesia, lengkap dengan barcode/QR pelacak, nomor DO unik, rincian barang, serta kolom tanda tangan Pengirim, Sopir, dan Penerima.`,
  },
  {
    keywords: /(stock opname|opname|hitung fisik|selisih stok|varians|audit stok)/i,
    reply: `**Panduan Stock Opname (Penyesuaian Fisik) di Tayooli WMS:**

1. **Buka Menu**: Pilih **Stock Opname** (\`/wms/opname\`).
2. **Jadwalkan Opname**: Klik **+ Buat Jadwal Opname Baru** dan tentukan gudang mana yang akan diaudit.
3. **Input Penghitungan Fisik**: Tim gudang menginput jumlah fisik barang yang dihitung nyata di rak/bin penyimpanan.
4. **Kalkulasi Varians Otomatis**: Sistem secara otomatis membandingkan stok tercatat di komputer vs stok aktual fisik di gudang.
5. **Rekonsiliasi / Adjustment**: Supervisor dapat meninjau selisih dan mengklik **Terapkan Penyesuaian Stok (Adjust)** untuk memperbarui saldo sistem menjadi cocok 100% dengan fisik gudang.`,
  },
  {
    keywords: /(rusak|scrap|cacat|kadaluwarsa|expired|write.?off|pecah)/i,
    reply: `**Panduan Pencatatan Barang Rusak / Scrap di Tayooli WMS:**

1. **Buka Menu**: Pilih **Barang Rusak / Scrap** (\`/wms/scrap\`).
2. **Catat Barang Rusak**: Klik **+ Catat Scrap Baru**.
3. **Pilih Item & Alasan**:
   - Pilih SKU barang yang rusak/kadaluwarsa dan gudang asalnya.
   - Masukkan jumlah unit yang diafkir/dibuang.
   - Pilih kategori alasan (misalnya: *Cacat Pabrik, Kerusakan Saat Bongkar Muat, Kadaluwarsa / Expired, atau Rusak Air*).
4. **Eksekusi Pengurangan Stok**: Setelah disimpan, sistem otomatis memotong stok barang tersebut dari inventori aktif dan mencatat kerugian operasional pada laporan pergudangan.`,
  },
  {
    keywords: /(barcode|scanner|pindai|scan)/i,
    reply: `**Panduan Barcode Scanner di Tayooli ERP:**

1. **Buka Menu**: Pilih **Barcode Scanner** (\`/wms/scanner\`) pada sidebar, atau klik ikon scanner di menu POS dan WMS.
2. **Dukungan Perangkat**:
   - **Kamera Ponsel / Web Camera**: Arahkan kamera ke barcode EAN-13, Code 128, atau QR Code produk.
   - **Hardware Barcode Gun (USB / Bluetooth)**: Anda cukup colok barcode scanner ke komputer, sistem akan otomatis mendeteksi input barcode secara instan.
3. **Fungsi Integrasi**:
   - Di modul **Kasir POS**: Menambah item ke keranjang secara instan.
   - Di modul **WMS Gudang**: Mengecek ketersediaan stok, lokasi rak, dan riwayat mutasi SKU secara real-time.`,
  },
  {
    keywords: /(marketplace|omnichannel|tokopedia|shopee|tiktok|lazada|sinkron|sync)/i,
    reply: `**Panduan Marketplace Omnichannel di Tayooli ERP:**

1. **Buka Menu**: Pilih **Marketplace Omnichannel** (\`/wms/marketplace\`).
2. **Integrasi Toko Online**: Hubungkan channel penjualan online Anda (Tokopedia, Shopee, TikTok Shop, Lazada).
3. **Sinkronisasi Stok Otomatis**:
   - Ketika ada penjualan di Kasir POS fisik, stok di seluruh toko online marketplace akan otomatis berkurang.
   - Sebaliknya, jika ada pesanan masuk dari marketplace, stok di gudang Tayooli langsung teralokasi sehingga **mencegah terjadinya overselling (kehabisan stok)**.
4. **Pantau Status**: Anda dapat memonitor status sync stok dan pesanan lintas channel dalam satu layar monitor.`,
  },
  {
    keywords: /(produk|product|sku|katalog|tambah barang|harga|hpp)/i,
    reply: `**Panduan Master Produk di Tayooli ERP:**

1. **Buka Menu**: Pilih **Products** (\`/products\`).
2. **Tambah Produk Baru**: Klik tombol **+ Tambah Produk**.
3. **Isi Parameter Produk**:
   - **Kode SKU**: Kode unik identifikasi barang (misal: \`SKU-KOP-001\`).
   - **Nama Barang**: Nama lengkap produk komersial.
   - **Harga Beli (HPP)** & **Harga Jual (Retail/Grosir)**.
   - **Minimum Stok (Alert)**: Batas stok menipis agar muncul peringatan di Dashboard.
   - **Barcode**: Kode angka barcode jika kemasan produk memiliki barcode pabrik.
4. Klik **Simpan**. Produk akan langsung tersedia di Kasir POS, Gudang WMS, dan Katalog Penjualan.`,
  },
  {
    keywords: /(dashboard|ringkasan|omzet|metrik)/i,
    reply: `**Panduan Dashboard Operasional di Tayooli ERP:**

Dashboard (\`/dashboard\`) menyajikan metrik bisnis terpadu secara real-time:
1. **Omzet & Penjualan POS**: Total pendapatan kasir harian, jumlah transaksi struk, dan rata-rata belanja.
2. **Inventori & Pergudangan (WMS)**: Total SKU aktif, jumlah total fisik unit barang, dan mutasi barang harian.
3. **Peringatan Stok Menipis**: Notifikasi produk dengan sisa stok di bawah batas aman agar pengadaan dapat segera dilakukan.
4. **Aksi Cepat**: Tombol pintas untuk langsung membuka Kasir POS, menambah produk, atau memeriksa stok gudang.`,
  },
  {
    keywords: /(setting|pengaturan|profil|pajak|ppn|akun)/i,
    reply: `**Panduan Pengaturan & Konfigurasi Sistem:**

Buka menu **Settings** (\`/settings\`) pada sidebar untuk mengatur:
1. **Profil Perusahaan**: Nama usaha, alamat operasional, logo bisnis, nomor telepon, dan email resmi.
2. **Preferensi Keuangan**: Format mata uang (IDR - Rupiah) dan persentase tarif PPN.
3. **Pengaturan Struk Kasir**: Header nama toko, footer ucapan terima kasih, dan info nomor kontak pada struk belanja thermal.
4. **Keamanan Akun**: Ubah kata sandi dan manajemen hak akses pengguna.`,
  },
]

async function queryExternalAI(prompt: string): Promise<string | null> {
  const geminiKey = process.env.GEMINI_API_KEY
  if (geminiKey) {
    try {
      const endpoint = `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=${geminiKey}`
      const systemInstruction = `Anda adalah asisten AI resmi "Tayooli Support" untuk sistem Tayooli ERP Standalone di Indonesia.
Sistem ini memiliki 13 modul inti:
1. Dashboard (/dashboard)
2. Products (/products)
3. Barang Masuk / Inbound (/wms/inbound)
4. Warehouse & Stock (/wms)
5. Surat Jalan DO (/wms/delivery-orders)
6. Marketplace Omnichannel (/wms/marketplace)
7. Stock Transfers (/wms/transfers)
8. Stock Opname (/wms/opname)
9. Barang Rusak / Scrap (/wms/scrap)
10. Barcode Scanner (/wms/scanner)
11. Point of Sale (/pos)
12. Settings (/settings)
13. Help & Support (/help)
Jawablah pertanyaan pengguna dengan sopan, ramah, jelas, terstruktur (dengan poin nomor atau bullet jika alur langkah), menggunakan Bahasa Indonesia yang baik dan profesional.`

      const res = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          contents: [
            {
              role: 'user',
              parts: [{ text: `${systemInstruction}\n\nPertanyaan pengguna: ${prompt}` }],
            },
          ],
        }),
        signal: AbortSignal.timeout(8000),
      })

      if (res.ok) {
        const data = await res.json()
        const text = data?.candidates?.[0]?.content?.parts?.[0]?.text
        if (text && typeof text === 'string' && text.trim().length > 0) {
          return text.trim()
        }
      }
    } catch {
      // Fall through to knowledge engine
    }
  }

  return null
}

export async function POST(req: NextRequest) {
  try {
    const body: ChatRequestBody = await req.json().catch(() => ({}))
    const userMessage = (body.message ?? '').trim()
    const sessionId = body.session_id || `session_${Date.now()}`

    if (!userMessage) {
      return NextResponse.json({ error: 'Pesan tidak boleh kosong' }, { status: 400 })
    }

    // 1. Coba external AI jika API key tersedia
    const aiReply = await queryExternalAI(userMessage)
    if (aiReply) {
      return NextResponse.json({
        reply: aiReply,
        session_id: sessionId,
      })
    }

    // 2. Gunakan built-in knowledge engine Tayooli ERP
    for (const item of TAYOOLI_KNOWLEDGE) {
      if (item.keywords.test(userMessage)) {
        return NextResponse.json({
          reply: item.reply,
          session_id: sessionId,
        })
      }
    }

    // 3. Fallback respons ramah dan informatif jika query umum / belum spesifik
    const defaultReply = `Halo! Saya **Tayooli Support AI** siap membantu operasional bisnis Anda.

Anda dapat menanyakan panduan dan alur kerja untuk 13 fitur Tayooli ERP:
- **Point of Sale (POS)**: Cara transaksi kasir, potong stok otomatis, dan cetak struk thermal.
- **Barang Masuk (Inbound)**: Penerimaan barang dari hasil produksi, transfer cabang, atau pemasok luar ke rak tujuan.
- **Warehouse & Stok (WMS)**: Manajemen gudang, mutasi barang, dan lokasi rak.
- **Surat Jalan (DO)**: Penerbitan dan cetak surat jalan delivery order resmi.
- **Stock Transfers**: Prosedur transfer stok antar gudang dari draft hingga diterima.
- **Stock Opname**: Audit stok fisik dan penyesuaian selisih otomatis.
- **Barang Rusak / Scrap**: Pencatatan barang afkir, kadaluwarsa, dan write-off stok.
- **Barcode Scanner**: Pemindaian barcode produk via kamera maupun scanner gun.
- **Marketplace Omnichannel**: Sinkronisasi stok otomatis dengan Tokopedia, Shopee, TikTok Shop.
- **Master Produk**: Pendaftaran SKU, batas stok minimal, harga beli, dan jual.
- **Pengaturan Akun**: Konfigurasi profil toko, PPN, dan opsi cetak.

Ketik pertanyaan spesifik Anda (misalnya: *"Bagaimana cara transfer stok antar gudang?"*), dan saya akan memandu langkah demi langkah!`

    return NextResponse.json({
      reply: defaultReply,
      session_id: sessionId,
    })
  } catch (err) {
    console.error('Tayooli support chat error:', err)
    return NextResponse.json({
      reply: 'Layanan Tayooli Support AI siap melayani. Silakan ulangi pertanyaan Anda.',
      session_id: `session_${Date.now()}`,
    })
  }
}
