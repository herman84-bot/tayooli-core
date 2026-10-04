# PRD: Official Document Generation & Print Engine (Surat Jalan & Faktur Penjualan)

**Status:** Approved / Architecture Specification  
**Version:** 1.0.0  
**Author:** CTO Office  
**Target:** Tayooli ERP Engine (WMS & O2C Requirement #14)  
**Date:** 2026-09-09  

---

## 1. Problem Statement
Dalam operasional logistik gudang dan distribusi B2B/B2C, dokumen fisik cetak merupakan syarat legal wajib serah-terima muatan:
1. **Surat Jalan (Delivery Order / DO Ekspedisi):** Pengemudi truk atau kurir pihak ketiga (ekspedisi) tidak diperkenankan keluar dari pintu gerbang gudang tanpa membawa dokumen Surat Jalan fisik bertanda tangan legal (Pengirim, Pengemudi, dan Penerima), lengkap dengan rincian nomor rak/bin pengambilan untuk pemeriksaan muatan.
2. **Faktur Penjualan (Sales Invoice):** Tim *Finance & Accounts Receivable* membutuhkan dokumen penagihan resmi berlogo perusahaan, rincian PPN 11%, nilai terbilang Rupiah, serta detail nomor rekening bank untuk penagihan resmi ke pelanggan.

**Yang harus dibangun:**
1. Modul Antarmuka Pengelolaan Surat Jalan (`app/(app)/wms/delivery-orders/page.tsx`):
   - Tabel Surat Jalan dengan filter gudang dan status (`DRAFT`, `CONFIRMED`, `PICKED`, `PACKED`, `SHIPPED`, `DELIVERED`, `CANCELLED`).
   - Modal penerbitan Surat Jalan baru (terkait Sales Order, gudang pemenuhan, armada pengiriman, dan rincian item rak).
   - Tombol eksekusi `Dispatch` yang memicu pemotongan stok otomatis ke `@CUSTOMER` via WMS ledger.
2. Mesin Cetak Dokumen Resmi Standar A4 (`components/wms/PrintDeliveryOrder.tsx` & `components/sales/PrintSalesInvoice.tsx`):
   - Standar cetak A4 Indonesia dengan CSS `@media print` terisolasi (menyembunyikan sidebar, navbar, tombol kontrol).
   - Surat Jalan mencakup: KOP perusahaan, No. DO, No. Ref Sales Order, Ekspedisi, No. Polisi, Sopir, Tabel Barang (SKU, Nama Barang, Rak/Bin/Pallet Pengambilan, Qty, Satuan), Catatan, serta **3 Kolom Tanda Tangan Resmi**: *Yang Menyerahkan (Gudang)*, *Yang Membawa (Sopir/Kurir)*, *Yang Menerima (Customer)*.
   - Faktur Penjualan mencakup: KOP perusahaan, No. Invoice, Tanggal Jatuh Tempo, Rujukan No. Surat Jalan, Rincian Pembeli (NPWP), Tabel Nilai Barang, Subtotal, PPN 11%, Biaya Kirim, Terbilang Rupiah, Rekening Bank, dan Watermark Status Pelunasan (*PAID* / *UNPAID*).
3. Integrasi Tombol Cetak Langsung:
   - Tombol cetak Surat Jalan di `/wms/delivery-orders`.
   - Tombol cetak Faktur Penjualan di `/sales-invoices`.

---

## 2. Benchmark Repositori Acuan (Odoo, ERPNext, Dolibarr)

| Repositori | Pola Desain (*Pattern*) yang Diadopsi | Implementasi di Tayooli ERP |
|---|---|---|
| **Odoo (Stock Picking Slip / Delivery Slip)** | *Printable Picking & Delivery Slip with 3-party Signature* | Dokumen cetak A4 memadukan data logistik (plat nomor kendaraan, nama sopir) dan operasional gudang (nomor rak/lokasi pengambilan barang) disertai 3 blok tanda tangan legal serah-terima. |
| **ERPNext (Print Format Engine)** | *Isolated Print Media Queries & High-Fidelity Typography* | Menggunakan styling `@media print` murni dengan ukuran baku A4 ($210\text{mm} \times 297\text{mm}$), penghilangan elemen navigasi browser, dan kontras tajam hitam-putih yang ramah printer laser/dot-matrix. |
| **Dolibarr (Commercial Invoice & Delivery Receipts)** | *Bilingual Invoice with Tax Breakdown & Bank Coordinates* | Dokumen faktur komersial yang menyajikan rujukan silang ke nomor Surat Jalan, rincian perpajakan (PPN 11%), teks terbilang resmi, serta koordinat perbankan perusahaan. |

---

## 3. Scope (In-Scope & Out-of-Scope)

### In-Scope:
- Dashboard dan manajemen Surat Jalan (`/wms/delivery-orders`).
- Pemutakhiran backend `GetDeliveryOrderByID` untuk menyertakan `product_name`, `product_sku`, dan `location_code` agar data cetak lengkap seketika.
- Komponen cetak resolusi tinggi A4 untuk Surat Jalan dan Faktur Penjualan dengan fitur `window.print()` dan pratinjau modal interaktif.
- QR Code verifikasi integritas dokumen (nomor referensi, tanggal, dan tanda tangan digital hash).
- Pemutakhiran navigasi sidebar grup "Warehouse & POS".

### Out-of-Scope:
- Server-side headless Chrome (Puppeteer) PDF generation (menggunakan browser native print rendering yang jauh lebih cepat, hemat memori server, dan bebas dependensi eksternal).

---

## 4. Acceptance Criteria (Testable & Numbered)

- **AC-1 (Delivery Order Management):** Staf gudang dapat melihat daftar Surat Jalan, memfilter berdasarkan gudang dan status, serta menerbitkan Surat Jalan baru berstatus `DRAFT`.
- **AC-2 (Dispatch & Stock Deduction):** Tombol `Dispatch` pada Surat Jalan berhasil memicu transisi status ke `SHIPPED` dan memotong stok gudang dari rak fisik ke lokasi virtual `@CUSTOMER` dalam satu transaksi database.
- **AC-3 (Surat Jalan Print Format):** Format cetak Surat Jalan A4 memuat KOP perusahaan, No. DO, Plat Nomor, Nama Supir, Ekspedisi, daftar barang beserta nomor rak pengambilan, catatan serah terima, dan 3 kolom tanda tangan resmi (*Pengirim*, *Pengemudi*, *Penerima*).
- **AC-4 (Sales Invoice Print Format):** Format cetak Faktur Penjualan memuat No. Invoice, tanggal jatuh tempo, rujukan No. DO / Sales Order, rincian PPN 11%, teks terbilang bahasa Indonesia yang tepat, dan instruksi transfer rekening bank.
- **AC-5 (Print Media Isolation):** Saat dialog cetak browser (`Ctrl+P` / `window.print()`) aktif, seluruh elemen antarmuka web (sidebar, navbar, tombol aksi, modal backdrop) otomatis disembunyikan sehingga hanya kertas A4 yang tercetak sempurna.
