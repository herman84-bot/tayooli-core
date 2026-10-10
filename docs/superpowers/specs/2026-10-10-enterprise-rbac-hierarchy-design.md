# Desain Spesifikasi: Hierarki Role & Kontrol Akses Perusahaan (RBAC)

**Dokumen:** `docs/superpowers/specs/2026-10-10-enterprise-rbac-hierarchy-design.md`  
**Tanggal:** 10 Oktober 2026  
**Status:** Draf Tinjauan Pengguna  
**Referensi Pola:** Mengadopsi pola pertahanan berlapis (*Defense-in-Depth*) & gerbang spesifikasi formal (*Specification Gate* - Devin 2.0 / Codex).

---

## 1. Ringkasan Eksekutif & Latar Belakang

Aplikasi **Tayooli ERP Core** saat ini memiliki aturan kontrol akses pergudangan yang kuat di backend Go (`wms_usecase.go` dan `user_warehouses`), namun terdapat 3 celah besar pada lapisan antarmuka dan manajemen pengguna:
1. Pilihan role di menu `Settings > Tim` hanya menyediakan **Admin** dan **Member**. Role riil seperti staf gudang, kepala gudang, dan kasir belum bisa dipilih dari antarmuka web.
2. Navigasi samping (Sidebar) menampilkan seluruh 12 menu kepada semua orang tanpa memandang jabatannya.
3. Belum ada role resmi untuk **Kasir (Cashier)**, sehingga staf kasir toko masih bisa mengakses menu pergudangan jika mengetahui URL-nya.

Spesifikasi ini mendefinisikan hierarki 5 level role standar operasional perusahaan dagang/distribusi di Indonesia dengan prinsip **Pemisahan Wewenang (*Segregation of Duties*)**.

---

## 2. Struktur Hierarki 5 Role Resmi

| Level | Nama Jabatan | Kode Sistem (`role`) | Cakupan Akses | Deskripsi & Batas Wewenang |
|:---:|---|---|---|---|
| **1** | **Pemilik Perusahaan** | `owner` | Seluruh Perusahaan | Akses 100% mutlak. Kelola langganan, profil PT, keuangan, undang/hapus tim, ubah role. Tidak dapat dihapus/di-downgrade. |
| **2** | **Admin Operasional** | `admin` | Seluruh Gudang | Kelola master data produk & harga, persetujuan transfer barang & scrap, impor marketplace, kelola daftar gudang. Tidak bisa hapus Owner. |
| **3** | **Kepala Gudang / SPV** | `warehouse_manager` | Gudang yang Ditugaskan | Menyetujui transfer masuk/keluar di gudangnya, validasi hasil stock opname fisik, monitor aktivitas tim gudang. Tidak bisa ubah harga/settings. |
| **4** | **Staf Gudang Lapangan** | `warehouse` | Gudang Tertentu (`user_warehouses`) | Eksekusi fisik: scan barang masuk, putaway rak, picking, packing DO, hitung fisik opname, barcode scanner. Dilarang approve. |
| **5** | **Kasir Toko** | `cashier` | Toko / POS | Hanya modul Point of Sale (POS): transaksi kasir toko harian, cetak struk. Terkunci mutlak dari seluruh menu pergudangan & kantor pusat. |
| *Ref* | *Auditor Internal* | `auditor` | Seluruh Gudang | Khusus pemeriksa: akses hanya-baca (*read-only*) ke laporan dan mutasi. Dilarang melakukan mutasi/simpan data apapun (`403`). |

---

## 3. Matriks Hak Akses Modul & Navigasi Sidebar

Setiap role hanya melihat menu yang relevan pada navigasi Sidebar. Halaman lain dijaga ketat di tingkat rute web dan API:

| Modul / URL | `owner` | `admin` | `warehouse_manager` | `warehouse` | `cashier` | `auditor` |
|---|:---:|:---:|:---:|:---:|:---:|:---:|
| **Dashboard** (`/dashboard`) | ✅ Ya | ✅ Ya | ✅ Ya (Gudangnya) | ❌ Sembunyi | ❌ Sembunyi | ✅ Lihat Saja |
| **Products** (`/products`) | ✅ Penuh | ✅ Penuh | 👁️ Lihat Saja | 👁️ Lihat Saja | ❌ Sembunyi | 👁️ Lihat Saja |
| **Arus Barang** (`/wms/arus-barang`) | ✅ Penuh | ✅ Penuh | ✅ Penuh (Gudangnya) | 📝 Input Fisik | ❌ Sembunyi | 👁️ Lihat Saja |
| **Warehouse & Stock** (`/wms`) | ✅ Penuh | ✅ Penuh | ✅ Penuh (Gudangnya) | 📝 Gudangnya | ❌ Sembunyi | 👁️ Lihat Saja |
| **Marketplace** (`/wms/marketplace`) | ✅ Penuh | ✅ Penuh | ❌ Sembunyi | ❌ Sembunyi | ❌ Sembunyi | 👁️ Lihat Saja |
| **Stock Transfers** (`/wms/transfers`) | ✅ Setujui | ✅ Setujui | ✅ Setujui (Gudangnya) | 📝 Buat Draf | ❌ Sembunyi | 👁️ Lihat Saja |
| **Stock Opname** (`/wms/opname`) | ✅ Posting | ✅ Posting | ✅ Validasi | 📝 Hitung Fisik | ❌ Sembunyi | 👁️ Lihat Saja |
| **Barang Rusak / Scrap** (`/wms/scrap`) | ✅ Penuh | ✅ Penuh | ✅ Setujui (Gudangnya)| 📝 Lapor Fisik | ❌ Sembunyi | 👁️ Lihat Saja |
| **Barcode Scanner** (`/wms/scanner`) | ✅ Ya | ✅ Ya | ✅ Ya | ✅ Ya (Utama) | ❌ Sembunyi | ❌ Sembunyi |
| **Point of Sale** (`/pos`) | ✅ Ya | ✅ Ya | ❌ Sembunyi | ❌ Sembunyi | ✅ **Default** | ❌ Sembunyi |
| **Pengaturan** (`/settings`) | ✅ Penuh | ✅ Terbatas | ❌ Sembunyi | ❌ Sembunyi | ❌ Sembunyi | ❌ Sembunyi |
| **Bantuan & Dukungan** (`/help`)| ✅ Ya | ✅ Ya | ✅ Ya | ✅ Ya | ✅ Ya | ✅ Ya |

---

## 4. Alur Khusus Pengalaman Kasir (`cashier`)

1. **Login & Redirection Otomatis:**
   - Saat pengguna ber-role `cashier` berhasil login, sistem langsung mengarahkan URL ke `/pos` (bukan `/dashboard`).
2. **Sidebar Minimal:**
   - Navigasi samping kasir disederhanakan: hanya menampilkan logo toko, tombol aktif **Kasir POS**, tautan Bantuan, dan tombol Keluar (*Logout*).
3. **Route Guarding:**
   - Jika kasir sengaja mengetik URL `/products` atau `/wms` di bilah alamat browser, komponen `AppLayout` langsung menolak dan mengembalikan pengguna ke rute `/pos` dengan toast pemberitahuan: *"Akses terbatas untuk akun kasir."*

---

## 5. Arsitektur Backend Go & Database

### A. Validasi Role di Usecase Tim (`internal/usecase/team/team_usecase.go`)
Memperbarui daftar role yang diizinkan untuk diundang dan diubah:
```go
var ValidRoles = map[string]bool{
    "admin":             true,
    "warehouse_manager": true,
    "warehouse":         true,
    "cashier":           true,
    "auditor":           true,
    "member":            true,
}
```

### B. Penyimpanan Penugasan Gudang (`user_warehouses`)
Tabel PostgreSQL yang digunakan:
```sql
CREATE TABLE IF NOT EXISTS user_warehouses (
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    warehouse_id  UUID NOT NULL,
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    assigned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, warehouse_id),
    FOREIGN KEY (warehouse_id, tenant_id) REFERENCES warehouses(id, tenant_id) ON DELETE CASCADE
);
```

### C. Pembaruan Endpoint REST API Manajemen Tim
1. **`POST /api/v1/settings/team` (Undang Anggota):**
   - Menerima JSON:
     ```json
     {
       "email": "staf.cakung@perusahaan.com",
       "role": "warehouse",
       "warehouse_ids": ["wh-uuid-cakung-01"]
     }
     ```
   - Aturan Bisnis: Jika role adalah `warehouse` atau `warehouse_manager`, field `warehouse_ids` wajib diisi minimal 1 gudang.
   - Database Transaction: Menyimpan record `users` dan meng-insert penugasan ke `user_warehouses`.
2. **`PATCH /api/v1/settings/team/{id}/role` (Ubah Role & Penugasan Gudang):**
   - Menerima payload role baru dan array `warehouse_ids`.
   - Menghapus penugasan lama di `user_warehouses` dan memasukkan penugasan baru dalam 1 transaksi atomik.
3. **`GET /api/v1/settings/team` (Daftar Tim):**
   - Mengembalikan daftar anggota lengkap dengan array `assigned_warehouses: [{ id, name, code }]`.

### D. Penjagaan Rute API Backend (`cmd/api/main.go` & `wms_usecase.go`)
- Endpoint POS (`/api/v1/pos/*`): diizinkan untuk `owner`, `admin`, `cashier`.
- Endpoint WMS Mutasi (`/api/v1/wms/*`): diizinkan untuk `owner`, `admin`, `warehouse_manager`, `warehouse`. Staf kasir diblokir dengan `403 Forbidden`.
- Penjagaan `ValidateWarehouseWriteAccess`: Staf gudang dan manajer gudang hanya boleh menulis transaksi di gudang yang terdaftar di `user_warehouses`.

---

## 6. Arsitektur Antarmuka Pengguna (Frontend Next.js)

### A. Sidebar Dinamis (`components/layout/Sidebar.tsx`)
Fungsi navigasi memfilter `navGroups` berdasarkan `user?.role`:
- Membaca `user.role` dari hook `useAuth()`.
- Menyesuaikan daftar menu sesuai Matriks Bagian 3.
- Menjamin tidak ada kebocoran visual menu administrasi ke staf operasional bawah.

### B. Formulir Pengaturan Tim (`app/(app)/settings/page.tsx`)
1. **Modal Undang Anggota (*Invite Member*):**
   - Dropdown pilihan role kini menampilkan opsi lengkap:
     - *Admin Operasional*
     - *Kepala Gudang / Supervisor*
     - *Staf Gudang*
     - *Kasir (POS)*
     - *Auditor (Hanya Baca)*
   - Jika pengguna memilih role *Kepala Gudang* atau *Staf Gudang*, muncul otomatis blok centang pilihan gudang aktif (*Multi-Select / Checkbox*) yang diambil dari master gudang perusahaan.
2. **Modal Ubah Role (*Change Role*):**
   - Menampilkan role saat ini dan daftar gudang yang sedang ditugaskan.
   - Admin/Owner dapat menambah atau mencabut penugasan gudang staf.
3. **Tabel Anggota Tim:**
   - Kolom Role menampilkan lencana (*badge*) berwarna informatif:
     - `Owner`: Emas / Ungu Primer
     - `Admin`: Biru
     - `Kepala Gudang`: Indigo
     - `Staf Gudang`: Oranye
     - `Kasir`: Hijau Zamrud
     - `Auditor`: Abu-abu Netral
   - Menampilkan nama-nama gudang yang ditugaskan di bawah email/nama pengguna.

---

## 7. Aturan Keamanan & Invarian Bisnis (Wajib Lolos Uji)

1. **Invarian 1 (Owner Tak Tersentuh):** Akun dengan role `owner` tidak boleh diubah role-nya, tidak boleh dihapus, dan tidak boleh diturunkan statusnya oleh siapapun termasuk admin lain.
2. **Invarian 2 (Isolasi Kasir):** Akun `cashier` tidak boleh memiliki izin baca/tulis ke endpoint WMS (`/wms/*`), Marketplace, maupun Pengaturan Tim.
3. **Invarian 3 (Isolasi Fisik Gudang):** Akun `warehouse` yang ditugaskan di Gudang A tidak boleh melakukan mutasi (terima, transfer, scrap, opname) di Gudang B. Percobaan spoofing harus menghasilkan `403 Unauthorized Warehouse Access`.
4. **Invarian 4 (Pencegahan Self-Demotion):** Pengguna tidak boleh mengubah role dirinya sendiri.

---

## 8. Rencana Pengujian (Test Strategy)

1. **Backend Go Unit & Adversarial Tests:**
   - `team_usecase_test.go`: Pengujian invite & change role untuk semua role baru, validasi penugasan gudang, dan penolakan manipulasi owner.
   - `wms_usecase_test.go`: Pengujian pemblokiran staf kasir dari API gudang, dan pemblokiran staf gudang lintas warehouse.
2. **Frontend Component & Role Tests:**
   - Pengujian rendering menu sidebar untuk masing-masing role (kasir hanya melihat POS; staf gudang hanya melihat scanner & arus barang).
   - Pengujian modal penugasan gudang saat invite anggota di Settings.
3. **End-to-End Typecheck & Build:**
   - `npm run build` dan `go test ./...` bebas dari error.
