# Tayooli ERP Core Architecture (Standalone)

> **Catatan Penting untuk AI Agent**:  
> Repositori ini adalah **pecahan/ekstraksi mandiri (*standalone decoupling*)** dari repositori enterprise `Erp-Like-PAPER-ID`.  
> Untuk panduan lengkap orientasi, batas repositori GCP vs Zeabur, dan aturan port, baca **`AI_ONBOARDING_GUIDE.md`**.

---

## 1. Visi & DNA Produk
Tayooli Core adalah sistem ERP Standalone berkinerja tinggi, ramping, dan mandiri yang dirancang khusus untuk bisnis ritel, pergudangan (WMS), dan kasir POS di Indonesia. 
Sistem ini dipisahkan dari beban arsitektur enterprise yang kompleks (*Apache Kafka message broker, Python AI worker*) agar dapat berjalan sangat cepat dan hemat sumber daya pada lingkungan PaaS (Zeabur) atau VPS hemat (1–2 vCPU, 1–2 GB RAM).

---

## 2. Struktur Modul: 13 Fitur Inti (The 13 Core Modules)
Navigasi dan antarmuka pengguna `tayooli-core` dibatasi secara ketat hanya pada **13 modul operasional utama**:

```text
1. OVERVIEW
   - Dashboard (/dashboard) : Ringkasan omzet kasir harian, total unit stok gudang, dan notifikasi stok menipis.

2. INVENTORY
   - Products (/products) : Master data katalog barang, SKU unik, harga modal (HPP), harga jual, dan kode barcode.

3. WAREHOUSE & POS
   - Barang Masuk / Inbound (/wms/inbound) : Pencatatan barang masuk dari pemasok, hasil produksi, dan transfer antar gudang, langsung menambah stok di rak tujuan.
   - Warehouse & Stock (/wms) : Manajemen multi-gudang, monitoring level stok fisik, dan riwayat mutasi barang.
   - Surat Jalan DO (/wms/delivery-orders) : Penerbitan dan cetak dokumen resmi Delivery Order untuk ekspedisi/kurir.
   - Marketplace Omnichannel (/wms/marketplace) : Sinkronisasi stok terpusat lintas channel (Tokopedia, Shopee, TikTok Shop, Lazada).
   - Stock Transfers (/wms/transfers) : Alur pemindahan barang antar gudang (Draft -> Pending Approval -> In Transit -> Received).
   - Stock Opname (/wms/opname) : Audit fisik barang periodik dan kalkulasi selisih stok (varians) otomatis.
   - Barang Rusak / Scrap (/wms/scrap) : Pencatatan barang cacat/expired dan write-off pemotongan inventori aktif.
   - Barcode Scanner (/wms/scanner) : Pemindai cepat menggunakan kamera perangkat maupun alat barcode gun eksternal.
   - Point of Sale (/pos) : Kasir toko fisik, kalkulasi kembalian, cetak struk thermal 58mm/80mm, dan pemotongan stok otomatis.

4. ACCOUNT
   - Settings (/settings) : Profil bisnis, format mata uang Rupiah, tarif PPN, dan preferensi akun.
   - Help & Support (/help) : Asisten AI panduan resmi (Tayooli Support) dan form pelaporan kendala teknis.
```

> **Aturan untuk AI**: Modul lama enterprise dari repo induk seperti *Procure-to-Pay (Vendor Invoices, Purchase Orders, 3-Way Match), Order-to-Cash (Sales Orders, Sales Invoices), dan Akuntansi (Chart of Accounts, Jurnal Umum)* **TIDAK DIAKTIFKAN** di antarmuka pengguna `tayooli-core`. Dilarang memunculkannya kembali ke sidebar.

---

## 3. Tech Stack Terkini

### Frontend:
- **Framework**: Next.js 15 (App Router, React 19)
- **Styling**: Tailwind CSS & Lucide Icons
- **State & Data Fetching**: TanStack Query v5 & Zustand
- **Local Dev Port**: **3000** (`http://localhost:3000`)
- **API Proxy**: `app/api/v1/[...path]/route.ts` (meneruskan request ke backend Zeabur)
- **Built-in Support AI**: `app/api/v1/chat/message/route.ts` (penjawab panduan 13 modul dengan format rapi tanpa karakter markdown mentah)

### Backend:
- **Framework**: Go 1.24 (Chi Router, Clean/Hexagonal Architecture)
- **Database Driver**: Native `database/sql` & `pgx` (tanpa ORM seperti GORM)
- **Autentikasi**: JWT (Json Web Token) dalam HttpOnly Cookie (`tayooli_token`)
- **Multi-Tenancy**: PostgreSQL 15 Row-Level Security (`tenant_id` context enforcement)

---

## 4. Topologi Deployment & Hosting

```text
[ Pengguna / Browser ]
          │
          ▼
   tayooli.my.id (Domain Utama)
   43.157.210.155 (DNS A Record ke Zeabur Edge)
          │
          ├─────────────────────────────────────────┐
          ▼                                         ▼
[ Zeabur Frontend Service ]             [ Zeabur Backend Service ]
 Next.js 15 Standalone Node              Go 1.24 Chi Binary API
 (tayooli-frontend.zeabur.app)           (tayooli-backend.zeabur.app)
          │                                         │
          └─────────── PROXY /api/v1/* ─────────────┘
                                                    │
                                                    ▼
                                        [ PostgreSQL 15 Database ]
                                         Zeabur Managed PG Database
```

---

## 5. Hubungan dengan Repository Induk (`Erp-Like-PAPER-ID`)
- `Erp-Like-PAPER-ID` tetap berada di Google Cloud Platform (GCP) VM `104.197.178.237`.
- `tayooli-core` adalah entitas terpisah yang berada di Zeabur PaaS dan GitHub `github.com/herman84-bot/tayooli-core.git`.
- **Kedua repositori ini independen**: perubahan pada `tayooli-core` tidak boleh menyentuh atau merusak file di `Erp-Like-PAPER-ID`.
