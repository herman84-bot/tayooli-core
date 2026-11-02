# SOP & Rencana Kerja: Pembersihan AI Slop & Pengamanan Logika Bisnis (Tayooli Core)

> **Dokumen Status:** Aktif & Mengikat (Binding)  
> **Target:** 12 Modul Inti Tayooli ERP Core  
> **Prinsip Utama:** Perbaikan kualitas visual dan performa UI **TANPA PERNAH** menyentuh atau merusak logika bisnis (*Zero Business Logic Regression*).

---

## 1. Latar Belakang & Hasil Audit Hallmark

Berdasarkan audit otomatis menggunakan engine standar Hallmark (`sloplint`), ditemukan total **1.092 titik AI Slop** pada kode antarmuka (`app/` dan `components/`):
* **Modul Inti Aktif (12 Modul):** 766 titik
* **Modul Warisan/Sampah (Sisa Decouple GCP):** 238 titik
* **Halaman Publik / Autentikasi:** 88 titik

Penyakit utama yang ditemukan:
1. **Gate 48 (Hex Color Hardcoded):** 266 titik (penulisan kode warna hex acak seperti `[#2563EB]`, `[#E2E8F0]`).
2. **Gate 67 (All-Caps Screaming Labels):** 187 titik (refleks AI memberi `uppercase tracking-wider text-xs font-semibold` di semua form & tabel).
3. **Gate 10a (Lazy transition-all):** 85 titik (membebani CPU/GPU perangkat mobile gudang & POS karena memicu reflow layout).
4. **Gate 64 (Monospace as Costume):** 52 titik (membungkus teks normal seperti `"hari terakhir"` dan `"· Berlaku s/d"` dengan `font-mono`).
5. **Gate 29 (Backdrop Blur Berat):** 43 titik (`backdrop-blur-xs` pada modal dan sheet).
6. **Gate 30a & 63 (Emoji & Simbol Tempel):** Emoji nyasar pada dropdown (`✨ Deteksi Otomatis`, `🌐`, `📦`) dan panah tempel (`→`).

---

## 2. Lima Aturan Pengaman Logika Bisnis (Non-Negotiable)

Setiap pengerjaan refactoring wajib tunduk pada 5 aturan baku berikut:

### Aturan 1: Isolasi Total Logika Bisnis (Zero Business Logic Touch)
* **DILARANG** mengubah atau memindahkan fungsi kalkulasi: perhitungan diskon, subtotal, PPN, sisa stok, status batch FEFO, konversi desimal (`decimal.Decimal`), dan parser angka (`parseQty`, `parseNum`).
* **DILARANG** menyentuh state management & API hooks: pemanggilan TanStack Query (`useQuery`, `useMutation`), hook form, dan store Zustand.
* **DILARANG** mengubah URL endpoint API, payload request JSON, maupun mapping response field dari backend Go.

### Aturan 2: Kekebalan Atribut DOM (DOM Attribute Invariance)
Refactoring hanya boleh menyentuh **nama class CSS/Tailwind** dan **format teks display**.  
Dilarang keras mengubah, mengganti nama, atau menghapus atribut berikut:
* `name`, `id`, `type`, `value`, `defaultValue`
* `onChange`, `onSubmit`, `onClick`, `onBlur`, `onKeyDown`
* `data-testid`, `key`, `ref`, `disabled`, `readOnly`

### Aturan 3: Verifikasi Pengujian Otomatis Red-Green (Automated Test Guardrail)
Sebelum dan sesudah menyentuh satu modul:
1. Jalankan unit test Jest frontend terkait (`npm test -- <path-test>`). Wajib **PASS**.
2. Lakukan perapihan visual sesuai scope modul tersebut.
3. Jalankan kembali test Jest dan typecheck (`npx tsc --noEmit`).
4. **Jika ada 1 saja test yang gagal/merah, perubahan detik itu juga di-revert (dibatalkan).** Tidak ada kompromi toleransi.

### Aturan 4: Isolasi Cakupan Per Modul (One Module, One Scope)
* **DILARANG** melakukan perombakan massal lintas file sekaligus (*no bulk sweeping*).
* Pengerjaan dipecah per modul inti secara terisolasi: selesaikan satu modul, uji hingga hijau, baru pindah ke modul berikutnya.

### Aturan 5: Preservasi Teks Teknis & Akronim
Saat merapikan teks "screaming caps" (`uppercase`):
* **HANYA** ubah label umum (misal: `<label>KODE GUDANG</label>` $\to$ `Kode Gudang`).
* **DILARANG** mengubah akronim teknis, singkatan industri, atau format kode sistem:
  * Singkatan: `SKU`, `POS`, `WMS`, `QC`, `LPN`, `PO`, `DO`, `SLA`, `PPN`, `IDR`.
  * Nilai data dinamis/enum: `WH-01`, `BATCH-202611`, `PENDING`, `APPROVED`.

---

## 3. Rencana Kerja Bertahap (Step-by-Step Plan)

```
[FASE 0: Baseline & Safety Audit]
        │
        ▼
[FASE 1: Quick Wins (Risiko 0%)] ──► Hapus emoji, panah tempel, monospace nyasar
        │
        ▼
[FASE 2: Normalisasi Label (Per Modul)] ──► Sentence case form & table header
        │
        ▼
[FASE 3: Optimasi Interaksi & GPU] ──► transition-all -> transition-colors, hapus backdrop-blur
        │
        ▼
[FASE 4: Standardisasi Token Warna] ──► Ganti hex liar [#2563EB] ke utility tema
        │
        ▼
[FASE 5: Karantina Modul Warisan] ──► Isolasi/hapus 238 titik sisa modul GCP
```

---

### FASE 0: Baseline & Verifikasi Awal
1. Pastikan working tree git bersih atau perubahan sebelumnya tercatat rapi.
2. Jalankan test baseline:
   ```bash
   npm test
   npx tsc --noEmit
   ```
3. Pastikan test suite 100% hijau sebelum memulai Fase 1.

---

### FASE 1: Quick Wins — Kosmetik Risiko 0%
Fokus pada elemen terisolasi tanpa risiko merusak layout:
1. **Hapus emoji nyasar:**
   * Dropdown marketplace: `<option>✨ Deteksi Otomatis</option>` $\to$ `<option>Deteksi Otomatis</option>`.
   * Dropdown gudang: `<option>🌐 Semua Gudang</option>` $\to$ `<option>Semua Gudang</option>`.
   * Dropdown opsi: `<option>📦 [ ]</option>` $\to$ bersihkan emoji.
   * Ganti emoji peringatan `⚠️` dengan komponen Lucide Icon (`<AlertTriangle />`).
2. **Hapus panah tempel (`→`):**
   * `"Master Produk →"` $\to$ `"Master Produk"`
   * `"Buka Arus Barang →"` $\to$ `"Buka Arus Barang"`
   * `"Petakan Sekarang →"` $\to$ `"Petakan Sekarang"`
3. **Rapikan Monospace Nyasar:**
   * Kembalikan kata biasa (`"hari terakhir"`, `"unit"`, `"· Berlaku s/d"`) ke font sans reguler.
   * Pertahankan monospace **hanya** untuk angka nominal mata uang dan kode barcode/SKU/LPN.
* **Verifikasi:** Jalankan unit test Jest terkait.

---

### FASE 2: Normalisasi Label & Header Tabel (Sentence Case)
Dikerjakan bertahap satu per satu modul:
* **Batch 2A — Modul Dasbor (`app/(app)/dashboard/`):** 18 titik.
* **Batch 2B — Modul Produk (`app/(app)/products/`):** 34 titik.
* **Batch 2C — Modul Kasir POS (`app/(app)/pos/`):** 77 titik.
* **Batch 2D — Modul WMS & Pergudangan (`app/(app)/wms/*`):**
  * Sub-batch 2D.1: Gudang & Stok (`/wms/page.tsx`)
  * Sub-batch 2D.2: Arus Barang Masuk & Keluar (`/wms/arus-barang/`)
  * Sub-batch 2D.3: Pemindai Barcode (`/wms/scanner/`)
  * Sub-batch 2D.4: Transfer & Opname (`/wms/transfers/`, `/wms/opname/`)
  * Sub-batch 2D.5: Barang Rusak / Scrap (`/wms/scrap/`)
* **Verifikasi Tiap Batch:** Jalankan test Jest modul bersangkutan dan typecheck TypeScript.

---

### FASE 3: Optimasi Animasi & Efisiensi Perangkat
Fokus pada performa di HP scanner gudang dan tablet kasir berdaya rendah:
1. **Eliminasi `transition-all`:** Ganti `transition-all duration-200` pada tombol, input, dan select menjadi `transition-colors duration-150` atau `transition-opacity`. Mencegah reflow layout CPU/GPU.
2. **Eliminasi Backdrop Blur Berat:** Ganti `backdrop-blur-xs over bg-slate-900/60` dengan solid overlay netral (`bg-slate-900/60`).
* **Verifikasi:** Uji interaksi form, modal pop-up, dan pergantian tab.

---

### FASE 4: Standardisasi Token Warna
1. Ganti penulisan arbitrary hex `[#2563EB]` dengan class tema `blue-600` / token primer.
2. Ganti arbitrary hex `[#E2E8F0]` dengan `border-border` atau `border-slate-200`.
3. Ganti arbitrary hex `[#EA580C]` dengan `orange-600`.
4. Pastikan kontras teks terhadap background memenuhi standar WCAG (minimal 4.5:1).
* **Verifikasi:** Cek visual seluruh halaman untuk memastikan tidak ada warna teks yang bertabrakan dengan background.

---

### FASE 5: Karantina Modul Warisan (238 Titik Dead Code)
1. Periksa dependensi file-file sisa decouple enterprise:
   * `app/o2c.module.css`
   * `app/(app)/o2c.module.css`
   * `app/(app)/accounting/`
   * `app/(app)/sales-orders/`
   * `app/(app)/sales-invoices/`
   * `app/(app)/dashboard/purchase-orders/`, `invoices/`, `vendors/`, `payment-orders/`
2. Pastikan tidak ada satupun dari 12 modul aktif yang mengimpor komponen dari modul warisan tersebut.
3. Karantina atau hapus aman file-file tersebut dari repository.
* **Verifikasi:** Jalankan `npm run build` penuh (`next build`). Pastikan build sukses (`exit=0`) dengan ukuran output bundle yang lebih efisien.

---

## 4. Protokol Rollback

Jika pada tahap mana pun terjadi kegagalan test atau komplain fungsional:
```bash
# Rollback per file/modul yang bermasalah secara spesifik:
git checkout HEAD -- <path-file-yang-bermasalah>
npm test -- <path-test>
```
Dilarang memaksakan commit jika ada tes yang belum hijau.
