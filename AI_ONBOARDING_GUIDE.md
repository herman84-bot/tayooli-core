# PANDUAN ORIENTASI ARSITEKTUR & PENGEMBANGAN AI (TAYOOLI CORE)
> **PENTING UNTUK SEMUA AI AGENT (Claude, Copilot, DeepSeek, Cursor, ChatGPT, dll.)**:
> Dokumen ini adalah **sumber kebenaran utama (Single Source of Truth)** mengenai asal-usul, batasan arsitektur, dan aturan pengerjaan repository `tayooli-core`. **BACA DOKUMEN INI SEBELUM MENULIS ATAU MENGUBAH KODE APAPUN.**

---

## 1. Asal-Usul & Latar Belakang (Genesis & History)

Repository **`tayooli-core`** ini **BUKAN** proyek yang dibuat dari nol (*greenfield*), melainkan hasil **pecahan, ekstraksi, dan penyederhanaan mandiri (*standalone decoupling*)** dari repository induk:
- **Repository Induk**: `Erp-Like-PAPER-ID` (versi lengkap skala *Enterprise Multi-Tenant*).
- **Server Induk**: Google Cloud Platform (GCP) VM `104.197.178.237`.
- **Repository Pecahan (Proyek Ini)**: `tayooli-core` (versi ramping khusus Retail, Gudang WMS, dan Kasir POS untuk UMKM/Bisnis Indonesia).
- **Server Proyek Ini**: Zeabur PaaS & domain resmi **`https://tayooli.my.id`** (alias: `https://tayooli-frontend.zeabur.app`).

---

## 2. Batasan Keras Antara GCP vs Zeabur (STRICT BOUNDARIES)

Sebagai AI yang bekerja di repositori ini, Anda **DILARANG KERAS** melanggar batas pemisahan berikut:

| Parameter | Repository Induk (`Erp-Like-PAPER-ID`) | Repository Ini (`tayooli-core`) |
| :--- | :--- | :--- |
| **Lokasi Folder Lokal** | `C:\Users\rehan\Desktop\project\Erp-Like-PAPER-ID` | `C:\Users\rehan\Desktop\project\tayooli-core` |
| **Server Target** | Google Cloud Platform (GCP) VM | Zeabur PaaS Cloud |
| **Alamat Akses** | `http://104.197.178.237` (atau `http://gcp.tayooli.my.id`) | **`https://tayooli.my.id`** (Produksi) |
| **Dev Port Lokal** | Port `3005` (jika dijalankan) | **Port `3000`** (`http://localhost:3000`) |
| **Aturan Sentuh** | **DILARANG MENGUBAH / MENYENTUH APAPUN** | **Tempat Anda Bekerja & Berinovasi** |

> âš ï¸ **PERINGATAN UNTUK AI:**  
> Jika pengguna meminta perbaikan, fitur baru, atau debugging, **HANYA kerjakan di folder `tayooli-core`**. Jangan pernah membuka, mengedit, merevert, atau menjalankan perintah di folder `Erp-Like-PAPER-ID` kecuali ada perintah eksplisit tertulis dari pengguna.

---

## 3. Fitur yang Diminta: TEPAT 13 FITUR BERSIH (THE 13 CORE MODULES)

> **Catatan 2026-10-07 (KO-2):** Barang Masuk dan Surat Jalan akan digabung jadi 1 modul **Barang Masuk & Keluar** (target 12 modul). Lihat CLAUDE.md dan PRD master §3.5. Daftar di bawah ini masih menggambarkan kode saat ini sampai item KO-2a/KO-2c selesai.

Pada saat proses pemisahan dari repo induk, aplikasi lama memiliki banyak fitur enterprise yang sangat rumit (*Procure-to-Pay, Order-to-Cash, Chart of Accounts, Journal Entries, Billing/Subscriptions, Kafka broker, Python AI worker*).

Pengguna telah menetapkan mandat tegas: **Aplikasi Tayooli Core HANYA berisi 13 Modul Inti dengan hierarki navigasi yang bersih**:

```text
OVERVIEW
 1. Dashboard                 -> /dashboard

INVENTORY
 2. Products                  -> /products

WAREHOUSE & POS
 3. Barang Masuk (Inbound)    -> /wms/inbound
 4. Warehouse & Stock         -> /wms
 5. Surat Jalan (DO)          -> /wms/delivery-orders
 6. Marketplace Omnichannel   -> /wms/marketplace
 7. Stock Transfers           -> /wms/transfers
 8. Stock Opname              -> /wms/opname
 9. Barang Rusak / Scrap      -> /wms/scrap
 10. Barcode Scanner          -> /wms/scanner
 11. Point of Sale            -> /pos

ACCOUNT
 12. Settings                 -> /settings
 13. Help & Support           -> /help
```

### ðŸš« Yang DILARANG Dikembalikan ke Navigasi Sidebar / Tampilan Utama:
1. **Procure-to-Pay (P2P)**: *Purchase Invoices, Purchase Orders, Goods Receipts, Payment Orders, Vendors, Approvals*.
2. **Order-to-Cash (O2C)**: *Customers, Sales Orders, Sales Invoices*.
3. **Akuntansi (Accounting)**: *Chart of Accounts (CoA), Jurnal Umum*.
4. **Billing**: *Paket langganan dan tagihan SaaS*.

> ðŸ’¡ **Mengapa folder/file route lama masih ada di codebase?**  
> Di dalam folder `app/(app)/dashboard/invoices`, `app/(app)/sales-orders`, dll., beberapa file sengaja tidak dihapus dari filesystem untuk mencegah *broken import* atau error kompilasi TypeScript. **Namun modul-modul ini sengaja dihilangkan dari navigasi sidebar (`Sidebar.tsx`)**. AI dilarang memunculkan kembali modul-modul lama tersebut ke antarmuka pengguna!

---

## 4. Panduan Port Lokal & Aturan Lingkungan (CRITICAL PORTS)

Banyak AI sebelumnya mengalami kebingungan mengenai port pengujian. Ikuti pedoman wajib ini:

1. **Port `3000` (`http://localhost:3000`)**:
   - Ini adalah port resmi aplikasi **Tayooli Core Frontend (Next.js)**.
   - Menjalankan perintah `npm run dev` di repository ini sudah dikunci otomatis ke port 3000 via `package.json` (`next dev -p 3000`).
   - Setiap pengujian visual atau pengujian Playwright lokal **WAJIB diarahkan ke `http://localhost:3000`**.
2. **Port `3080` (`http://127.0.0.1:3080/`)**:
   - **BUKAN APLIKASI TAYOOLI!**
   - Port 3080 adalah port internal runner antarmuka DeepSeek Harness GUI tempat manusia berinteraksi dengan AI.
   - **JANGAN PERNAH** mengarahkan Playwright atau browser ke `3080` untuk mengecek fitur Tayooli!
3. **Port `8081` (`http://localhost:8081`)**:
   - Port backend Go lokal jika dijalankan. Di Zeabur, backend cloud aktif di `https://tayooli-backend.zeabur.app`.

---

## 5. Arsitektur Teknis `tayooli-core`

### A. Frontend (Next.js 15 App Router)
- **Framework**: Next.js 15 (React 19, TypeScript strict mode, Tailwind CSS).
- **Layout Inti**: `app/(app)/layout.tsx` (menyediakan konteks autentikasi dan wadah sidebar).
- **Sidebar Navigasi**: `components/layout/Sidebar.tsx` (berisi 13 menu dalam 4 kelompok hierarki).
- **Proxy Catch-all API**: `app/api/v1/[...path]/route.ts` meneruskan panggilan `/api/v1/*` ke backend Zeabur (`https://tayooli-backend.zeabur.app`).
- **Tayooli Support AI**: 
  - Route handler `app/api/v1/chat/message/route.ts` menyediakan asistensi panduan 13 fitur dengan respons Bahasa Indonesia tanpa ketergantungan API eksternal (dan auto-switch ke Gemini jika env key dipasang).
  - Komponen `components/chat/ChatMessage.tsx` memiliki parser markdown internal yang otomatis merender `**teks**` menjadi tag `<strong>`, `*teks*` menjadi `<em>`, list nomor/bullet rapi, dan **menjamin 0 tanda bintang mentah (*) di layar pengguna**.

### B. Backend (Go 1.24 Chi Router)
- **Lokasi Kode**: `backend/go-core/`
- **Arsitektur**: Clean / Hexagonal Architecture (`handler` $\rightarrow$ `usecase` $\rightarrow$ `repository`).
- **Database**: PostgreSQL 15 multi-tenant dengan Row-Level Security (`tenant_id`).
- **CORS Middleware**: `internal/middleware/cors.go` disetel menggunakan env var `FRONTEND_ORIGIN=https://tayooli.my.id`.

---

## 6. Checklist Verifikasi Mandiri untuk AI (Anti-Hallucination)

Sebelum Anda menyatakan pekerjaan Anda "selesai", jalankan checklist ini:
- [ ] Apakah saya bekerja murni di `tayooli-core` tanpa menyentuh `Erp-Like-PAPER-ID`?
- [ ] Apakah saya menjaga sidebar tetap berisi **13 fitur wajib** tanpa membangkitkan modul lama P2P/O2C/Akuntansi?
- [ ] Apakah pengujian Playwright diarahkan ke `http://localhost:3000` (bukan 3080 atau 3005)?
- [ ] Apakah verifikasi diuji langsung dengan output riil (bukan klaim atau asumsi)?
- [ ] Apakah teks yang ditampilkan ke pengguna bebas dari karakter markdown mentah (seperti `**` atau `*`)?

---

*Dokumen ini dibuat dan disahkan sebagai pedoman pengembang & AI agen untuk menjaga stabilitas dan keselarasan proyek Tayooli Core.*
