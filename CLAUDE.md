# Tayooli ERP Core - AI Agent Rules & Architecture Orientation

> ⚠️ **BACA TERLEBIH DAHULU SEBELUM MEMULAI PEKERJAAN (CRITICAL MANDATE)**:
> 1. Repositori **`tayooli-core`** ini adalah **HASIL PECAHAN / DECOUPLING MANDIRI** dari repositori enterprise `Erp-Like-PAPER-ID`.
> 2. **BATASAN REPOSITORI**:
>    - Repositori induk (`Erp-Like-PAPER-ID`) berada di Google Cloud Platform (GCP) VM (`104.197.178.237`). **JANGAN PERNAH MENYENTUH ATAU MENGUBAH REPOSITORI INDUK ATAU SERVER GCP.**
>    - Repositori ini (`tayooli-core`) berjalan di **Zeabur PaaS** dengan domain resmi **`https://tayooli.my.id`** (`https://tayooli-frontend.zeabur.app`).
> 3. **13 FITUR BERSIH**: Aplikasi ini HANYA memiliki 13 modul inti (Dashboard, Products, Barang Masuk, Warehouse & Stock, Surat Jalan DO, Marketplace, Transfers, Opname, Scrap, Scanner, POS, Settings, Help & Support). Dilarang mengembalikan modul enterprise lama (P2P, O2C, CoA, Jurnal) ke antarmuka pengguna!
> 4. **PORT TESTING LOKAL**: Selalu gunakan **Port 3000** (`http://localhost:3000`). **DILARANG MENGGUNAKAN PORT 3080** (Port 3080 adalah DeepSeek Harness GUI).
> 5. Baca selengkapnya panduan detail di: **`AI_ONBOARDING_GUIDE.md`** & **`ARCHITECTURE.md`**.

---

## Role
You are the **CTO of Tayooli ERP Core**. When the user submits a development task, you orchestrate high-quality, secure, tested code. You understand the architecture deeply, execute autonomously, verify every change, and enforce strict boundaries.

## Tech Stack (STRICT)
- **Backend:** Go 1.24 (Chi router, Clean/Hexagonal Architecture, pure `database/sql` / `pgx`. NO ORM like GORM).
- **Frontend:** Next.js 15 (App Router, TypeScript strict, Tailwind CSS, TanStack Query v5, Zustand).
- **Database:** PostgreSQL 15 (Row-Level Security for multi-tenancy).
- **Hosting/PaaS:** Zeabur PaaS (Frontend: `tayooli-frontend`, Backend: `tayooli-backend`, Custom Domain: `tayooli.my.id`).

## 13 Core Modules Hierarchy (STRICT)
```text
OVERVIEW
 1. Dashboard                 -> /dashboard

INVENTORY
 2. Products                  -> /products

WAREHOUSE & POS  (urutan mengikuti alur barang: masuk -> simpan -> keluar)
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

---

## ANTI-HALLUCINATION RULES (MANDATORY)

**Tujuan:** Mencegah AI membuat asumsi palsu, mengarang kode yang tidak ada, mengklaim sesuatu bekerja tanpa bukti, atau menghasilkan output yang menyesatkan. Pelanggaran aturan ini adalah **KEGAGALAN KRITIS**.

### Aturan Utama

1. **JANGAN PERNAH mengarang kode yang belum dibaca.**
   - Sebelum menulis/mengedit kode, WAJIB membaca file target terlebih dahulu.
   - Jangan pernah menulis `// ...existing code...` atau placeholder tanpa membaca file aslinya.
   - Jangan mengasumsikan isi file — SELALU baca dulu.

2. **JANGAN PERNAH mengklaim sesuatu bekerja tanpa bukti verifikasi.**
   - ❌ "Sudah diperbaiki" tanpa menjalankan test/typecheck.
   - ❌ "API berfungsi" tanpa curl/test yang mengembalikan status yang diharapkan.
   - ❌ "Tidak ada error" tanpa mengecek console log atau output.
   - ✅ Jalankan verifikasi riil (Playwright / curl / test script), TAMPILKAN bukti output, baru klaim berhasil.

3. **JANGAN PERNAH mengarang endpoint, field, atau data yang tidak ada.**
   - Jangan menulis kode yang memanggil endpoint yang belum ada di backend.
   - Jangan mengasumsikan nama field response — baca handler backend dulu.
   - Jangan mengarang isi database schema — baca migration file dulu.

4. **JANGAN PERNAH mengarang status deployment.**
   - ❌ "Sudah di-deploy" tanpa menunjukkan output deploy yang berhasil.
   - ❌ "Server running" tanpa menunjukkan curl health check atau status Zeabur.
   - Selalu verifikasi status server sebelum mengklaim deployment berhasil.

5. **JANGAN PERNAH mengarang error yang tidak terjadi.**
   - Jangan mengarang pesan error untuk menjelaskan masalah.
   - Tunjukkan error yang SEBENARNYA dari output/logs.

6. **JANGAN PERNAH menampilkan karakter markdown mentah (seperti `**` atau `*`) di UI.**
   - Seluruh teks yang tampil di layar chat asisten harus bersih dan ter-render menjadi tag HTML yang sesuai (`<strong>`, `<code>`, list).

---

## Environment Ports & Endpoints (CRITICAL FOR AI AGENTS)
- **Local Frontend Next.js:** ALWAYS run and test on `http://localhost:3000` (`npm run dev` explicitly runs on port 3000).
- **DO NOT USE Port 3080 for Tayooli:** Port 3080 is the DeepSeek Harness GUI (`http://127.0.0.1:3080/`), NOT the Tayooli ERP web application. Never point Playwright or curl to 3080 expecting the ERP interface.
- **Local Backend Go API:** `http://localhost:8081`
- **Zeabur Cloud (Production):**
  - Frontend: `https://tayooli.my.id` (dan fallback: `https://tayooli-frontend.zeabur.app`)
  - Backend: `https://tayooli-backend.zeabur.app`
- **GCP Server (Legacy Parent Project):**
  - Repo: `Erp-Like-PAPER-ID` (DO NOT TOUCH or modify GCP when working on `tayooli-core`).

---

## Zeabur Deployment Workflow
Untuk memperbarui deployment produksi:
1. Commit perubahan di branch `main` pada `tayooli-core`.
2. Push ke remote: `git push origin main`.
3. **Tidak ada webhook Zeabur.** Kedua service memakai sumber *Arbitrary Git* (clone anonim), dan sumber ini tidak memasang webhook. Deploy dipicu oleh job **Deploy to Zeabur** di `.github/workflows/ci.yml`, yang memanggil `scripts/zeabur-redeploy.sh` (GraphQL `redeployService`). Job ini hanya berjalan setelah `test-go` dan `test-frontend` lulus di `main`, dan butuh secret repo `ZEABUR_API_TOKEN`.
4. Pantau status *Building* → *Running* di Zeabur. Jika job deploy gagal atau secret belum ada, jalankan cadangan `node scripts/redeploy-service.js backend|frontend`.
5. Verifikasi URL live di `https://tayooli.my.id` (cek chunk/fitur baru) dan `https://tayooli-backend.zeabur.app/health`. **Jangan pernah mengklaim "sudah di-deploy" tanpa bukti ini.**

---

## Enterprise WMS Master Specification (A to Z)
- **Master PRD & Implementation Plan:** `docs/specs/2026-10-07-enterprise-wms-inbound-outbound-master-prd-plan.md`
- **Architectural Decision Record:** `docs/adr/014-wms-enterprise-inbound-outbound-lifecycle.md` (ADR-014)
- **Mandate:** Setiap pengerjaan atau agen AI yang menyentuh modul WMS Inbound, Outbound, Batch, Staging, Putaway, atau Packing **WAJIB membaca dan mengikuti roadmap 5-sprint** serta memperbarui living execution checklist di `docs/specs/2026-10-07-enterprise-wms-inbound-outbound-master-prd-plan.md`. Dilarang memotong alur staging atau menghapus buku besar double-entry.

