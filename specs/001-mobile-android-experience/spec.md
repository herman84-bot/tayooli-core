# Feature Specification: Mobile Android Experience & Responsive Hallmarks

**Feature**: Mobile Android Experience for Tayooli ERP Core  
**Directory**: `specs/001-mobile-android-experience`  
**Status**: In Progress  
**Target**: Mobile Browser & Android Device Touch Experience  

## 1. Executive Summary

Tayooli ERP Core is frequently accessed by warehouse operators, retail store cashiers, and managers directly from Android smartphones and mobile browsers (Chrome Mobile, Samsung Internet, Firefox Mobile). 
The goal of this feature is to ensure the entire application provides a first-class, ergonomic, and touch-friendly mobile experience ("ramah di aplikasi android"):
- Touch targets strictly meet Android Material Guidelines (min 44px - 48px).
- Zero accidental horizontal viewport scrolling (`overflow-x: clip`).
- Native-like navigation: robust top app bar, persistent or accessible mobile drawer, and quick bottom navigation for high-frequency actions.
- Touch-friendly forms, preventing unwanted browser auto-zoom (min 16px font on inputs).
- Responsive table containers and card fallbacks.
- Flawless mobile ergonomics across high-impact core modules: POS Kasir, Pemindai Barcode (Scanner), Gudang & Stok, Arus Barang, and Dasbor.

---

## 2. User Scenarios & Acceptance Criteria

### Scenario 1: Kasir Menggunakan HP Android di Toko Ritel (POS Mobile)
- **Actor**: Kasir / Toko Owner
- **Context**: Membuka `https://tayooli.my.id/pos` melalui browser Chrome di smartphone Android (layar 360px - 412px).
- **Flow**:
  1. Kasir melihat katalog produk dalam tampilan grid/card yang nyaman (tidak terhimpit).
  2. Input pencarian dan pemindai barcode mudah disentuh, tidak memicu zoom layar otomatis.
  3. Indikator keranjang melayang (floating cart bar) di bagian bawah menampilkan jumlah item & total harga secara jelas.
  4. Ketika ditekan, modal keranjang belanja terbuka penuh (full-height sheet) dengan tombol penyesuaian kuantitas (+/-) minimal 44px.
  5. Tombol "Bayar Sekarang" berukuran besar dan mudah ditekan dengan jempol satu tangan.
  6. Dialog pembayaran (Tunai / QRIS) dan input nominal uang tertata rapi tanpa kepotong keyboard virtual.

### Scenario 2: Operator Gudang Melakukan Pemindaian Barcode (WMS Scanner)
- **Actor**: Operator Gudang
- **Context**: Membuka `/wms/scanner` di HP Android dengan kamera belakang aktif.
- **Flow**:
  1. Tampilan kamera memenuhi area pandang dengan scrim overlay pemindai yang jelas.
  2. Tombol beralih kamera, input manual SKU/LPN, dan tombol konfirmasi berukuran minimal 48px sehingga dapat dioperasikan saat mengenakan sarung tangan kerja ringan.
  3. Hasil scan kartu produk atau LPN langsung muncul di bawah layar tanpa mengharuskan scroll ke bawah berulang-ulang.

### Scenario 3: Manajer Mengakses Navigasi & Dasbor di Ponsel
- **Actor**: Manajer Gudang / Bisnis
- **Context**: Membuka `/dashboard` dan berpindah modul di smartphone.
- **Flow**:
  1. Header atas memiliki tombol hamburger menu yang responsif dan selalu dapat diakses.
  2. Drawer samping terbuka mulus dari kiri dengan backdrop transparan, menutup otomatis saat rute ditekan atau backdrop diklik.
  3. Kartu KPI Dasbor tertata 1 kolom di layar sempit (<640px) dan 2 kolom di tablet, teks angka tidak terpotong.
  4. Grafik metrik otomatis menyesuaikan lebar layar tanpa memicu horizontal scrollbar.

### Scenario 4: Pengoperasian Tabel Data & Form di Layar Ponsel (Arus Barang & Produk)
- **Actor**: Staf Administrasi / Logistik
- **Context**: Melihat daftar barang masuk/keluar atau mengedit produk di `/wms/arus-barang` atau `/products`.
- **Flow**:
  1. Tabel data dibungkus dalam container horizontal-scroll yang mulus dengan bayangan indikator overflow.
  2. Tombol aksi (Edit, Hapus, Cetak, Detail) memiliki jarak yang cukup agar tidak salah sentuh.
  3. Drawer atau Modal form formulir tampil sebagai bottom sheet atau modal responsif yang dapat di-scroll vertikal dan memiliki safe-area inset (tidak tertutup bilah navigasi Android).

---

## 3. Functional Requirements

### FR-1: Viewport & Layout Ergonomics
- [x] FR-1.1: `html` dan `body` menerapkan `overflow-x: clip` untuk meniadakan pergeseran horizontal tak sengaja.
- [x] FR-1.2: Shell aplikasi di `app/(app)/layout.tsx` menggunakan layout CSS responsif (`md:hidden` / `hidden md:flex`) yang stabil tanpa flicker hydration.
- [x] FR-1.3: Mobile header memiliki tinggi minimal 56px dengan tombol hamburger menu berukuran touch target minimal 48x48px dan label yang jelas.
- [x] FR-1.4: Drawer mobile (`Sidebar.tsx`) dapat ditutup dengan swipe atau tombol silang (min 44px) serta backdrop klik.
- [x] FR-1.5: Dukungan safe-area inset (`env(safe-area-inset-bottom)`, `safe-area-inset-top`) untuk ponsel modern berponi/gesture bar.

### FR-2: Touch Targets & Input Accessibility
- [x] FR-2.1: Seluruh tombol primer, sekunder, ikon aksi, dan tombol paginasi memiliki ukuran sentuh minimal 44px x 44px (rekomendasi 48px untuk tombol aksi utama).
- [x] FR-2.2: Input teks, number, select, dan combobox pada viewport mobile memiliki ukuran font minimal 16px (`text-base sm:text-sm`) untuk mencegah auto-zoom browser ponsel.
- [x] FR-2.3: Tombol aksi numerik (+ / - kuantitas keranjang) memiliki touch padding memadai agar jempol pengguna nyaman.

### FR-3: Modul POS Kasir Mobile
- [x] FR-3.1: Pada layar < 1024px, katalog produk POS tersusun dalam 1-2 kolom yang proporsional.
- [x] FR-3.2: Floating bottom bar keranjang menampilkan total item dan nominal secara tegas dengan tombol checkout yang ramah jempol satu tangan.
- [x] FR-3.3: Dialog pembayaran kasir (Metode Cash/QRIS, input uang diterima, tombol nominal cepat) responsif dan tidak terpotong di layar 360px.

### FR-4: Modul Pemindai Barcode (WMS Scanner) Mobile
- [x] FR-4.1: Viewfinder kamera responsif terhadap rasio layar smartphone (portrait).
- [x] FR-4.2: Mode toggle (Inbound, Outbound, QC, Audit) dan tombol aksi manual mudah dijangkau di layar sentuh.

### FR-5: Tabel & Form Responsif
- [x] FR-5.1: Semua tabel data memiliki container `overflow-x-auto` dengan styling scrollbar yang rapi dan tidak merusak layout halaman.
- [x] FR-5.2: Modal formulir (tambah produk, buat surat jalan, filter) memiliki batas tinggi maksimal `max-h-[90vh]` dengan header/footer sticky dan isi yang dapat di-scroll.

---

## 4. Success Criteria

1. **Touch Target Compliance**: 100% tombol navigasi dan tombol aksi utama memenuhi batas minimal 44px.
2. **Zero Horizontal Layout Breakage**: Tidak ada pergeseran layar mendatar liar (horizontal wobble) saat membuka halaman utama di viewport 360px, 375px, 390px, dan 412px.
3. **Zero Input Auto-Zoom**: Input teks di perangkat ponsel tidak memicu lonjakan zoom otomatis browser saat fokus.
4. **Build & Test Guard**: Semua 63 test suite Jest tetap 100% lulus (692+ tests) dan TypeScript typecheck 0 errors.
5. **Hallmark Design Floor Compliance**: Memenuhi prinsip desain Hallmark (warna token, visual hierarchy bersih, tanpa slop).
