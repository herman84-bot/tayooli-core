# Tayooli ERP Core (Standalone)

> 📖 **PANDUAN PENTING UNTUK DEVELOPER & AI AGENT**:  
> Repositori ini adalah **hasil pecahan dan penyederhanaan mandiri (*standalone decoupling*)** dari repositori enterprise `Erp-Like-PAPER-ID`.  
> Untuk panduan lengkap orientasi, batasan arsitektur, dan aturan pengerjaan agar tidak bingung, silakan baca:  
> 👉 **[`AI_ONBOARDING_GUIDE.md`](./AI_ONBOARDING_GUIDE.md)** dan **[`ARCHITECTURE.md`](./ARCHITECTURE.md)**.

---

## 📌 Latar Belakang & Asal-Usul Proyek

Sistem ERP ini sebelumnya tergabung dalam repositori monolitik `Erp-Like-PAPER-ID` yang di-deploy pada server Google Cloud Platform (GCP). Karena kebutuhan pasar UMKM dan bisnis ritel/gudang di Indonesia memerlukan sistem yang **ringan, cepat, mandiri, dan dapat di-host dengan biaya terjangkau**, maka fitur inti diekstraksi ke repositori terpisah ini: **`tayooli-core`**.

### Perbedaan Utama:

| Komponen | Repositori Induk (`Erp-Like-PAPER-ID`) | Repositori Ini (`tayooli-core`) |
| :--- | :--- | :--- |
| **Fokus Bisnis** | Korporat besar (P2P, O2C, Multi-ledger, Approval matriks) | Ritel, Pergudangan WMS, Kasir POS mandiri |
| **Server & Hosting** | Google Cloud Platform (GCP) VM (`104.197.178.237`) | **Zeabur PaaS** (`https://tayooli.my.id`) |
| **Broker Pesan** | Apache Kafka (KRaft mode) | Dipangkas / Tidak diperlukan (*Zero bloat*) |
| **AI Worker** | Worker Python gRPC terpisah | Ringan / Terintegrasi langsung via Next.js API |
| **Modul UI** | Lusinan modul enterprise | **Tepat 13 Modul Operasional Inti** |

---

## 🚀 13 Modul Operasional Inti (The 13 Core Modules)

Sistem antarmuka `tayooli-core` dirancang terstruktur dalam 4 kelompok hierarki yang bersih:

### 1. OVERVIEW
- **Dashboard (`/dashboard`)**: Ringkasan performa omzet POS harian, jumlah transaksi kasir, total fisik stok barang di seluruh gudang, serta pusat peringatan barang menipis (*Low Stock Alert*).

### 2. INVENTORY
- **Products (`/products`)**: Master data barang, pengelolaan SKU unik, harga modal (HPP), harga jual ritel, batas stok aman, dan barcode kemasan.

### 3. WAREHOUSE & POS
- **Barang Masuk (Inbound) (`/wms/inbound`)**: Pencatatan barang masuk dari pemasok, hasil produksi, dan transfer antar gudang, langsung menambah stok di rak tujuan. Mendukung tiga tipe penerimaan: Hasil Produksi (dapur/pabrik), Transfer Cabang (antar-gudang), dan Pemasok Luar (vendor).
- **Warehouse & Stock (`/wms`)**: Manajemen multi-gudang, pemantauan saldo unit fisik per rak/lokasi, dan buku besar mutasi keluar-masuk barang.
- **Surat Jalan DO (`/wms/delivery-orders`)**: Penerbitan, alokasi armada/ekspedisi, dan cetak dokumen resmi Delivery Order (DO) berstandar bisnis Indonesia.
- **Marketplace Omnichannel (`/wms/marketplace`)**: Sinkronisasi stok terpusat lintas channel (Tokopedia, Shopee, TikTok Shop, Lazada) guna mencegah *overselling*.
- **Stock Transfers (`/wms/transfers`)**: Alur perpindahan barang antar gudang (*Draft $\rightarrow$ Pending Approval $\rightarrow$ In Transit $\rightarrow$ Received*).
- **Stock Opname (`/wms/opname`)**: Audit fisik berkala dan penyesuaian selisih (*variance reconciliation*) otomatis.
- **Barang Rusak / Scrap (`/wms/scrap`)**: Pencatatan barang cacat, afkir, kadaluwarsa, dan eksekusi *write-off* pengurangan stok dari gudang aktif.
- **Barcode Scanner (`/wms/scanner`)**: Pemindaian barcode produk dengan kamera laptop/ponsel maupun alat scanner gun eksternal.
- **Point of Sale (`/pos`)**: Kasir kasir cepat toko fisik, barcode input, perhitungan uang kembalian, cetak struk thermal 58mm/80mm, serta pemotongan stok gudang secara otomatis.

### 4. ACCOUNT
- **Settings (`/settings`)**: Konfigurasi profil perusahaan, preferensi mata uang (IDR), pengaturan tarif PPN, dan keamanan akun.
- **Help & Support (`/help`)**: Asisten AI panduan resmi Tayooli Support dan pelaporan tiket kendala teknis.

---

## 🛠️ Tech Stack

- **Frontend**: Next.js 15 (App Router, React 19, TypeScript strict mode, Tailwind CSS, TanStack Query v5, Zustand).
- **Backend**: Go 1.24 (Chi router, Clean/Hexagonal Architecture, pure `database/sql` & `pgx`, zero ORM).
- **Database**: PostgreSQL 15 (Row-Level Security multi-tenant).
- **Support Engine**: Native AI Assistant with built-in 13-module Indonesian ERP knowledge engine and clean markdown parser.

---

## 💻 Panduan Menjalankan Secara Lokal (Local Development)

### 1. Menjalankan Frontend
```bash
# Masuk ke direktori tayooli-core
cd tayooli-core

# Install dependensi
npm install --legacy-peer-deps

# Jalankan dev server (Otomatis berjalan di Port 3000)
npm run dev
```
Akses di browser: **`http://localhost:3000`**  
*(Catatan: Jangan gunakan Port 3080 karena port tersebut adalah port DeepSeek Harness GUI).*

### 2. Menjalankan Backend Go (Opsional)
```bash
cd backend/go-core
go run cmd/api/main.go
```
Backend berjalan di port `8081`. Frontend secara bawaan telah dikonfigurasi untuk meneruskan API proxy ke backend Zeabur (`https://tayooli-backend.zeabur.app`) atau backend lokal jika aktif.

---

## 🌐 Alur Deployment Produksi (Zeabur PaaS)

1. Repository terhubung ke **Zeabur PaaS**:
   - Frontend: `https://tayooli.my.id` (A record: `43.157.210.155`).
   - Backend: `https://tayooli-backend.zeabur.app`.
2. Setiap kali perubahan di-push ke branch `main`:
   ```bash
   git add .
   git commit -m "feat/fix: deskripsi perubahan"
   git push origin main
   ```
   Workflow GitHub Actions `.github/workflows/ci.yml` (job **Deploy to Zeabur**) lalu otomatis me-redeploy `tayooli-backend` dan `tayooli-frontend` **setelah** job `test-go` dan `test-frontend` lulus. Bila tes gagal, tidak ada deploy.

   **Mengapa tidak lewat webhook Zeabur:** kedua service memakai sumber **Arbitrary Git** (clone anonim `https://github.com/herman84-bot/tayooli-core.git`, branch `main`), bukan sumber "GitHub Repository". Sumber Arbitrary Git tidak memasang webhook GitHub, sehingga push tidak pernah memicu build. Hal ini terverifikasi di Settings → Source pada Oktober 2026.

   **Syarat satu kali:** buat access token di Zeabur → Account → API Keys, lalu simpan sebagai secret repo `ZEABUR_API_TOKEN` (GitHub → Settings → Secrets and variables → Actions). Tanpa secret ini, job deploy gagal dengan pesan jelas.

   **Cadangan manual:** klik **Redeploy** di dashboard Zeabur, atau jalankan `node scripts/redeploy-service.js backend|frontend` (lewat Chrome yang sudah login, CDP `:9100`), atau `ZEABUR_API_TOKEN=... ZEABUR_ENV_ID=6ac2683a6a873116ad572b40 bash scripts/zeabur-redeploy.sh <serviceID>`. Selalu verifikasi URL live setelah deploy.
