# Tayooli ERP Core (Standalone)

Standalone, high-performance B2B Enterprise Resource Planning (ERP) platform designed for self-hosting on affordable VPS infrastructure (1–2 vCPU, 1–2 GB RAM).

Decoupled from heavy message brokers (Kafka) and cloud AI bloat, this core edition is laser-focused on daily retail, warehouse, and financial operations.

---

## 🚀 Fitur Utama

1. **Dashboard Eksekutif**:
   - Konsolidasi real-time omzet POS harian/total, struk kasir, pelanggan aktif.
   - Metrik pergudangan WMS (total unit fisik, gudang aktif, mutasi harian).
   - Widget peringatan stok menipis (Low Stock Alert).
   - Trend invoice pembelian 6 bulan & Top 5 vendor.

2. **POS (Point of Sale / Kasir)**:
   - Antarmuka kasir cepat, barcode scanner support, mode direct sale.
   - Kalkulasi diskon, pajak, nominal bayar, dan uang kembalian.
   - **Pemotongan stok gudang otomatis** secara real-time saat transaksi selesai.
   - Cetak struk kasir (printer thermal ESC/POS 58mm/80mm & cetak standar).
   - Riwayat struk & cetak ulang.

3. **Gudang & Inventori (WMS)**:
   - Master Produk & SKU unik per tenant.
   - Manajemen Multi-Gudang & Hierarki Lokasi Rak/Palet.
   - Siklus Transfer Antar-Gudang (Draft $\rightarrow$ Pending Approval $\rightarrow$ Approved/Rejected $\rightarrow$ In Transit $\rightarrow$ Received).
   - Stock Opname fisik & penyesuaian selisih stok.
   - Penerbitan Surat Jalan resmi (Delivery Order).

4. **Keuangan & Akuntansi (Finance & Accounting)**:
   - **Procure-to-Pay (AP)**: Vendor Invoices, Purchase Orders (PO), Goods Receipts (GR), 3-Way Matching, Payment Orders, dan Approval Requests.
   - **Order-to-Cash (AR)**: Pelanggan (Customers), Pesanan Penjualan (Sales Orders), dan Faktur Penjualan (Sales Invoices).
   - **Buku Besar**: Bagan Akun (Chart of Accounts) & Jurnal Umum (Journal Entries).

5. **Akun & Multi-Tenancy (Auth & Tenancy)**:
   - Row-Level Security (RLS) PostgreSQL tingkat lanjut (isolasi data antar tenant).
   - Registrasi, Login sesi berbasis HttpOnly JWT Cookie.
   - Onboarding Workspace mandiri.
   - Integrasi Brevo HTTP API v3 (dan fallback SMTP) untuk pengiriman email verifikasi dan reset password.
   - Manajemen Anggota Tim & Hak Akses (Role-Based Access Control).

---

## 🛠️ Tech Stack

- **Backend**: Go 1.23 (Chi router, Clean/Hexagonal Architecture, pure `database/sql` without ORM).
- **Frontend**: Next.js 15 (App Router, TypeScript, Tailwind CSS, TanStack Query, Zustand, ZenSpace Design System).
- **Database**: PostgreSQL 15 (dengan Row-Level Security).
- **Gateway**: Nginx Reverse Proxy (Single-Domain architecture).

---

## 📦 Panduan Deploy ke VPS (Docker Compose)

### Prasyarat di VPS:
- Ubuntu 22.04 LTS atau 24.04 LTS (RAM minimal 1 GB atau 2 GB).
- Docker & Docker Compose plugin terpasang:
  ```bash
  sudo apt update && sudo apt install -y docker.io docker-compose-plugin
  sudo systemctl enable --now docker
  ```

### Langkah 1: Clone Repository
```bash
git clone https://github.com/herman84-bot/tayooli-core.git /opt/tayooli
cd /opt/tayooli
```

### Langkah 2: Konfigurasi Environment
Salin file template `.env.example`:
```bash
cp .env.example .env
nano .env
```
Sesuaikan variabel berikut:
- `DB_PASSWORD`: Password database PostgreSQL baru yang kuat.
- `JWT_SECRET`: Token acak minimal 32 karakter (misal hasil dari `openssl rand -hex 32`).
- `APP_URL`: Alamat domain Anda (contoh: `https://erp.domainanda.com` atau `http://IP_VPS`).
- `FRONTEND_ORIGIN`: Sama dengan `APP_URL`.
- `BREVO_API_KEY`: Kunci API Brevo jika ingin mengaktifkan pengiriman email (opsional).

### Langkah 3: Jalankan Aplikasi
```bash
docker compose up -d --build
```
Sistem akan otomatis:
1. Menjalankan container PostgreSQL 15 dan mengeksekusi seluruh migrasi database (`001` s/d `027`).
2. Meng-compile backend Go ke static binary yang sangat ringan.
3. Membangun Next.js ke mode standalone.
4. Menyalakan Nginx reverse proxy di port 80.

Cek status layanan:
```bash
docker compose ps
```

---

## 🌐 Menghubungkan Domain & SSL (HTTPS)

1. Arahkan **DNS A Record** domain Anda (misal `erp.domainanda.com`) ke IP publik VPS Anda.
2. Edit `nginx/nginx.conf` di baris `server_name`:
   ```nginx
   server_name erp.domainanda.com;
   ```
3. Pasang sertifikat SSL gratis dengan Certbot:
   ```bash
   sudo apt install -y certbot python3-certbot-nginx
   sudo certbot --nginx -d erp.domainanda.com
   ```
4. Restart Nginx:
   ```bash
   docker compose restart nginx
   ```

---

## 💻 Menjalankan Frontend Secara Lokal
Untuk menjalankan frontend secara lokal:
```bash
npm run dev
# Aplikasi siap diakses di http://localhost:3000
```
> **PENTING UNTUK PENGEMBANG & AI AGENT:**
> - Port aplikasi frontend Tayooli di lokal adalah **`http://localhost:3000`**.
> - Jangan gunakan port **`3080`** (port 3080 adalah port antarmuka DeepSeek Harness GUI, bukan aplikasi Tayooli).

---

## 🧪 Pengujian & Verifikasi Lokal

Untuk menjalankan pengujian unit secara lokal:

**Backend Go:**
```bash
cd backend/go-core
go test ./...
```

**Frontend Next.js:**
```bash
npm test
npx tsc --noEmit
```
