# Handoff — Perbaikan Hasil Uji Preview (status: SELESAI SELURUHNYA)

> Dokumen ini hasil pengujian nyata aplikasi di preview (`localhost:3000`, dev server, **demo mode** — tanpa backend Go).
> **Status: SEMUA ISSUE SELESAI & TERVERIFIKASI (Baseline: tsc 0 error, jest 14 suite/129 test PASS, go build+vet+test PASS). Verifikasi runtime 2 Okt 2026: semua flow lolos.**
> - P0-1: ✅ LULUS (list & create produk via `/api/v1/products`, muncul tanpa reload, formatCurrency).
> - P1-1: ✅ LULUS (`handleResponse` memakai `body.error`; tidak ada JSON mentah di UI).
> - P1-2: ✅ LULUS (DO tema terang, 0 elemen `bg-zinc`, konsisten dengan halaman lain).
> - P1-3: ✅ LULUS (seluruh 14+ endpoint fallback demo tersedia, 0 halaman 502).
> - P1-3b: ✅ LULUS (seluruh UUID demo diperbarui ke format RFC-4122 v4 valid, VendorListSchema/POSchema/GRSchema/PaymentOrderSchema/InvoiceListSchema terparsing sempurna).
> - P2-1: ✅ LULUS (0 `confirm(`/`alert(`; Dispatch & Receive memakai modal in-app, POST 200, <5 detik, tanpa hang).
> - P2-2: ✅ LULUS (gap harga↔stok terukur **12px** di xl & 2xl, tidak terpotong).
> - P2-3: ✅ LULUS (clear log → `/dashboard` + `/wms` load = **0×502**; `subscription` & `ai/permissions` = 200).
> - P3-1: ✅ LULUS (banner impor marketplace Total == Sukses + Gagal, dihitung dari payload).
> - P4-1: ✅ LULUS (0 literal `@CUSTOMER` tampil di teks user-facing; diganti "pelanggan tujuan" / "pelanggan").
> - P4-2: ✅ LULUS (mutasi demo in-memory: create produk, dispatch DO→SHIPPED, dispatch transfer→IN_TRANSIT, receive→RECEIVED, impor marketplace persisten).
> - P5-1: ✅ LULUS (refaktorisasi terminologi navigasi: Procure-to-Pay "Purchase Invoices (Vendor Bills)" vs Order-to-Cash "Sales Invoices", regression test sidebar pass).

## Konteks lingkungan uji

- Frontend Next.js 15 di `localhost:3000` (dev). **Tidak ada `NEXT_PUBLIC_API_URL`/`BACKEND_URL`** → proxy
  `app/api/v1/[...path]/route.ts` gagal fetch ke `http://localhost:8081` → jatuh ke demo fallback
  `lib/api/demo-data.ts`, dan mengembalikan **502 `{"error":"backend unreachable"}`** untuk path yang tidak ada demo-nya.
- Login demo: `admin@test.com` / `password123`.
- Endpoint **200 (tes ulang 1 Okt 2026)** — sudah termasuk 14 fallback baru: `auth/me`, `dashboard/summary`,
  `invoices`, `vendors`, `products`, `accounts`, `journal-entries`, `approvals`, `customers`, `sales-orders`,
  `sales-invoices`, `purchase-orders`, `goods-receipts`, `payment-orders`, `team-members`, `subscription`,
  `subscription/invoices`, `plans`, `usage`, `ai/permissions`,
  `wms/{warehouses,locations,transfers,delivery-orders,opnames,scraps,barcodes/resolve,marketplace/*}`, `payments/configs`.
- Endpoint yang **masih 502**: `inventory` (+ `stock`, `warehouses` non-`wms/`, `pos/orders`, `reports/sales`,
  `notifications`, `audit-logs`) — **tidak dipanggil UI mana pun** (grep pemanggilan kosong), jadi bukan noise.
  Kecuali `inventory` bila kelak dipakai, aman diabaikan.

Baseline hijau (terverifikasi 2 Okt 2026):
`npx tsc --noEmit` = **0 error** · `npx jest` = **14 suite / 129 test PASS** (naik dari 13/118, ada suite baru `demo-endpoints` + `print-engine` + `terbilang`) ·
`cd backend/go-core && go build ./... && go vet ./... && go test ./internal/...` = **PASS** — termasuk paket `wms`, `sales_order`.

---

## P0-1 — Halaman Products mati total (rusak juga di produksi)

**Gejala:** `/products` selalu menampilkan "No products found." dan tombol **Save Product** selalu gagal.

**Bukti:**
- `GET /api/products` → **404** (`content-type: text/html`, HTML 404 Next.js).
- `ls app/api/products` → **tidak ada route-nya**.
- Padahal `GET /api/v1/products` → **200** dengan demo data (5 produk), dan hook `useProducts` sudah ada.

**Akar masalah:** halaman legacy [`app/(app)/products/page.tsx`](../app/(app)/products/page.tsx) fetch ke endpoint
yang tidak pernah ada:
- baris 49: `const res = await fetch('/api/products'); // Requires actual API proxy setup`
- baris 93: `const res = await fetch('/api/products', { method: 'POST', ... })`

Karena `res.ok === false`, blok `if (res.ok)` dilewati → list tetap kosong, create selalu gagal.

**Arahan perbaikan:** pindahkan halaman ini ke jalur API yang benar (`/api/v1/products` via `lib/api.ts` /
`hooks/useProducts.ts`). Gunakan `components/products/ProductDrawer.tsx` + pola shell/desain yang dipakai halaman lain,
dan format rupiah lewat helper `lib/currency.ts` (halaman ini masih memakai `${p.price.toFixed(2)}`).

**Kriteria terima:** daftar produk termuat dari `GET /api/v1/products`; menambah produk mengirim `POST /api/v1/products`
dan produk baru muncul di daftar tanpa reload penuh; `npx tsc --noEmit` bersih.

**Verifikasi:** buka `/products` di preview → minimal 5 baris produk tampil; submit form → produk baru tampil.

---

## P1-1 — Raw JSON bocor ke UI di Sales Invoices

**Gejala:** `/sales-invoices` menampilkan teks mentah `{"error":"backend unreachable"}` di badan halaman.

**Bukti:** console `Failed to fetch sales invoices Error: {"error":"backend unreachable"}` dari
`hooks/useSalesInvoices.ts`.

**Akar masalah:** [`hooks/useSalesInvoices.ts`](../hooks/useSalesInvoices.ts) baris 49–54 memakai body mentah sebagai pesan error:

```ts
const body = await response.text().catch(() => '');   // baris 52
if (body) message = body;                             // baris 53  <— raw text
```

Bandingkan dengan [`hooks/useSalesOrders.ts:29-31`](../hooks/useSalesOrders.ts) dan
[`hooks/useCustomers.ts:33-35`](../hooks/useCustomers.ts) yang sudah benar: `body.error || 'Request failed: ${status}'`.

**Arahan perbaikan:** samakan dengan pola dua hook lain (`response.json()` → `body.error`), lalu render pesan yang ramah
di [`app/(app)/sales-invoices/page.tsx:204-205`] (`<div className={styles.emptyState}>{error}</div>`), jangan cetak objek.

**Kriteria terima:** tidak ada karakter `{`/`"error"` yang tampil di UI; pesan berbentuk kalimat manusiawi.

**Verifikasi:** buka `/sales-invoices` di demo mode → muncul pesan ramah + opsi retry, bukan JSON.

---

## P1-2 — Halaman Surat Jalan (DO) bertema gelap sendirian

**Gejala:** `/wms/delivery-orders` berlatar hitam (`zinc-950`) sementara Sidebar dan seluruh halaman lain terang.

**Bukti:** [`app/(app)/wms/delivery-orders/page.tsx:259`](../app/(app)/wms/delivery-orders/page.tsx#L259)
`className="min-h-screen bg-zinc-950 text-zinc-100 p-4 sm:p-6 lg:p-8"`.
Semua halaman WMS lain memakai `bg-[#F8FAFC]` (wms, opname, transfers, scrap, scanner) atau `bg-slate-50/60` (marketplace).

**Arahan perbaikan:** konversikan ke tema terang agar konsisten (`bg-[#F8FAFC]`, teks slate, border `#E2E8F0`),
termasuk kartu KPI, tabel, modal, dan panel detail. Modal/tabel gelap di baris 259–750 perlu disesuaikan.

**Kriteria terima:** halaman DO terlihat konsisten dengan `/wms/transfers` dan halaman lain; tidak ada `bg-zinc-9xx` tersisa.

**Verifikasi:** bandingkan screenshot `/wms/delivery-orders` vs `/wms/transfers`.

---

## P1-3 — Tanpa demo data, modul O2C & Purchase rusak di preview

**Gejala:** `/customers` → "Failed to load customers: backend unreachable"; `/sales-orders` → teks "backend unreachable";
`/dashboard/purchase-orders`, `/dashboard/goods-receipts`, `/dashboard/payment-orders` → error state kosong;
`/accounting/journal-entries` + Chart of Accounts + `/approvals` → "No results found"; `/billing` → section
**"Pilih Paket" kosong** (`/plans` 502); `/settings` tab Tim → "Gagal memuat data tim"; `/ai/permissions` (izin Copilot) → 502.

**Akar masalah:** [`lib/api/demo-data.ts`](../lib/api/demo-data.ts) `getDemoResponse()` hanya meng-cover
`/dashboard/summary`, `/wms/*`, dan paginated `/invoices`, `/vendors`, `/products`. Path lain jatuh ke 502.

**Arahan perbaikan (pilih salah satu, jangan campur):**
1. Tambahkan demo data untuk path O2C + finance (mengikuti pola `DEMO_*` + cabang `if` yang sudah ada), atau
2. Ganti fallback agar mengembalikan **array kosong yang valid** untuk list endpoint (200) sehingga halaman menampilkan
   empty state yang benar, bukan error.

**Kriteria terima:** setiap halaman daftar di atas minimal menampilkan empty state yang valid (bukan pesan
"backend unreachable"/"Tidak dapat memuat data") saat backend tidak ada.

**Verifikasi:** probe dari console halaman:
```js
for (const p of ['/api/v1/customers','/api/v1/sales-orders','/api/v1/sales-invoices','/api/v1/accounts',
  '/api/v1/journal-entries','/api/v1/approvals','/api/v1/purchase-orders','/api/v1/goods-receipts',
  '/api/v1/payment-orders','/api/v1/plans','/api/v1/team-members'])
  console.log((await fetch(p)).status, p)
```
Tidak boleh ada `502`.

**HASIL TES ULANG (1 Okt 2026):** probe 16 endpoint → hanya `/api/v1/inventory` yang masih 502 (tidak dipanggil UI mana pun, grep aman). Halaman beres & menampilkan data: customers, sales-orders, sales-invoices, accounts/CoA, journal-entries, approvals, billing (3 paket), Settings Tim (3 anggota).

### P1-3b — 4 halaman gagal karena id demo bukan UUID valid (✅ SELESAI & TERVERIFIKASI)

**Gejala:** `/dashboard/purchase-orders`, `/dashboard/goods-receipts`, `/dashboard/payment-orders`, `/dashboard/vendors` sebelumnya sempat menampilkan **"Tidak dapat memuat data ..."** + **"Terjadi kesalahan yang tidak diketahui."** — bukan empty state valid, melanggar kriteria P1-3.

**Bukti Temuan Awal:**
- `GET /api/v1/purchase-orders?page=1&per_page=20` → **200** dengan 3 item yang tampak valid.
- Reproduksi Node: `zod@4.4.3` → `z.string().uuid().safeParse('00000000-0000-0000-0004-000000000001')` = **false** (regex UUID hanya menerima nibble varian `89ab`; id demo sebelumnya memakai `0000`); id tenant `550e8400-e29b-41d4-a716-446655440000` = true; id vendor `v-001` = false.
- `getClientErrorMessage` menghasilkan "Terjadi kesalahan yang tidak diketahui." karena error **bukan** AxiosError → error-nya ZodError dari `z.array(POSchema).parse(arr)` di queryFn (`lib/queries/po.ts`).

**Akar masalah:** id demo `DEMO_PURCHASE_ORDERS`/`DEMO_GOODS_RECEIPTS`/`DEMO_PAYMENT_ORDERS` (sebelumnya berpola `00000000-0000-0000-0004-...`, `lib/api/demo-data.ts`) dan `DEMO_VENDORS` (`v-001`) gagal validasi `.uuid()` di `lib/schemas/{po,gr,payment-order,vendor}.ts`. Selain itu, `DEMO_VENDORS` kehilangan field wajib skema seperti `status`, `avg_rating`, `rating_count`, `tax_id`, dan `address`.

**Solusi & Resolusi (Opsi 1 Terpilih):**
1. Seluruh ID demo di `lib/api/demo-data.ts` dimigrasikan menjadi UUID standar RFC-4122 versi 4 valid dengan nibble varian `8`, `9`, `a`, atau `b` (misalnya vendor `11111111-1111-4111-8111-111111111101`, PO `44444444-4444-4444-8444-444444440001`, GR `55555555-5555-4555-8555-555555550001`, Payment Order `66666666-6666-4666-8666-666666660001`, Invoice `a0000000-0000-4000-8000-000000000001`).
2. Menambahkan field wajib skema `VendorListSchema` pada `DEMO_VENDORS`:
   - `status: "active"`
   - `avg_rating: "4.5"`
   - `rating_count: 10`
   - `tax_id: "01.234.567.8-012.000"`
   - `address: "Jl. Jenderal Sudirman No. 45, Jakarta"`
3. Ditambahkan suite pengujian otomatis `__tests__/demo-endpoints.test.ts` yang memverifikasi bahwa parser skema Zod (`VendorListSchema`, `POSchema`, `GRSchema`, `PaymentOrderListSchema`, `InvoiceListSchema`) mem-parse data demo tanpa error runtime dan format regex UUID valid.

**Kriteria terima:** ke-4 halaman menampilkan baris data (bukan "Tidak dapat memuat data") saat endpoint 200.

**Verifikasi (Terverifikasi 2 Okt 2026):**
- Runtime preview: Membuka 4 URL di atas berhasil menampilkan tabel data secara penuh dan interaktif tanpa ZodError.
- Automated tests: `npx jest __tests__/demo-endpoints.test.ts` PASS (5 test cases skema Zod valid 100%).

---

## P2-1 — `window.confirm()` / `alert()` native memblokir renderer

**Gejala (teramati langsung):** klik tombol **Dispatch** pada Surat Jalan membuka dialog `confirm()` native →
klik otomatis timeout 20 detik, `Escape`/`Enter` tidak menutup dialog, main thread terblokir
(`preview_evaluate` timeout), dan tab harus di-reload. Ini juga memblokir e2e Playwright tanpa dialog handler.

**Bukti (baris kode):**
- [`delivery-orders/page.tsx:130`](../app/(app)/wms/delivery-orders/page.tsx#L130) `if (!confirm("Konfirmasi pengiriman Surat Jalan ini? ..."))`
- [`delivery-orders/page.tsx:138`](../app/(app)/wms/delivery-orders/page.tsx#L138) `alert(msg)`
- [`transfers/page.tsx:102`](../app/(app)/wms/transfers/page.tsx#L102), `:115` `confirm(...)`
- [`transfers/page.tsx:109`](../app/(app)/wms/transfers/page.tsx#L109), `:122` `alert(...)`
- [`pos/page.tsx:263`](../app/(app)/pos/page.tsx#L263) `if (cart.length > 0 && confirm("Kosongkan keranjang belanja?"))`

**Arahan perbaikan:** ganti dengan dialog/toast internal aplikasi (pola ZenSpace yang dipakai modul lain),
bukan `window.confirm`/`alert`.

**Kriteria terima:** tidak ada `confirm(`/`alert(` tersisa di `app/`; aksi Dispatch/Receive/Kosongkan Keranjang
bisa diuji otomatis tanpa dialog handler dan tidak memblokir thread.

**Verifikasi:** `grep -rn "confirm(\|alert(" app/` → kosong; klik Dispatch di preview → selesai < 5 detik tanpa hang.

---

## P2-2 — Kartu produk POS berhimpit (harga menempel stok)

**Gejala:** pada layout `xl` (4 kolom) teks terlihat sebagai `Rp 25.000Stok: 50` — tanpa jarak.

**Bukti pengukuran DOM:** kartu lebar **179px**; blok harga `x=279,right=363`; blok stok `x=363,right=403` → **gap = 0**.
Label stok terhimpit pada lebar 40px.

**Akar masalah:** [`app/(app)/pos/page.tsx`](../app/(app)/pos/page.tsx#L519-L536) — dua blok bersaudara di dalam
`flex items-end justify-between` tanpa `gap`.

**Arahan perbaikan:** tambahkan `gap-2`/`gap-3` dan `min-w-0` + `shrink-0`, atau tata vertikal agar harga & stok
tidak bertabrakan di kolom sempit.

**Kriteria terima:** jarak harga↔stok ≥ 8px pada breakpoint `xl` dan `2xl`; teks `Stok: 50` tidak terpotong/wrap aneh.

**Verifikasi:** ukur ulang bounding box seperti di atas, `gap >= 8`.

---

## P2-3 — Noise 502 di hampir semua halaman (`/ai/permissions` + `/subscription`)

**Gejala:** setiap navigasi memunculkan ±2 error console 502 dan badge "N Issues" pada Next.js dev overlay,
menutupi error lain saat debugging.

**Bukti:** network log tiap halaman berisi `? /api/v1/subscription → 502` dan `? /api/v1/ai/permissions → 502`;
sumber panggilan: [`hooks/useAIPermissions.ts:26,51`](../hooks/useAIPermissions.ts) dan `hooks/useSubscription.ts`.

**Arahan perbaikan:** beri demo fallback untuk kedua endpoint (masuk ke P1-3), dan/atau jangan panggil keduanya bila
belum diperlukan (lazy/`enabled`).

**Kriteria terima:** membuka `/dashboard`, `/pos`, `/wms` tidak lagi menghasilkan 502 di console.

**Verifikasi:** clear log → reload 3 halaman → `preview_logs` tidak memuat 502.

---

## P3-1 — Ringkasan impor marketplace tidak konsisten (demo mode) (✅ SELESAI & TERVERIFIKASI)

**Gejala:** setelah impor sampel 3 pesanan, banner sempat menampilkan
"Sesi Impor Selesai: **Total 3 pesanan** diproses. **Sukses: 118**, Gagal/Duplikat: 0, SKU Belum Terpetakan: 2."

**Akar masalah:** sebelumnya angka "Sukses" berasal dari batch demo statis (`DEMO_MARKETPLACE_BATCHES[0]` di
[`lib/api/demo-data.ts`](../lib/api/demo-data.ts)) sedangkan angka "Total" berasal dari state lokal frontend. Sebelumnya, handler juga sempat memotong data statis dengan `slice(0, 2)` sehingga terjadi inkonsistensi `Total 3 ≠ Sukses 2 + Gagal 0`.

**Kriteria terima:** "Total" == "Sukses + Gagal" pada respons demo dan tampilan banner.

**HASIL TES ULANG (2 Okt 2026): ✅ LULUS & SELESAI.**
- **Resolusi:** Handler demo fallback `POST /wms/marketplace/import` di `lib/api/demo-data.ts` sekarang menginspeksi request payload body (`bodyRecord?.orders`).
  - Bila body memuat daftar pesanan (`rawOrders.length > 0`), handler memetakan item secara dinamis dan menghitung metrik batch secara real: `total_orders: rawOrders.length`, `processed_orders: rawOrders.length`, `failed_orders: 0`, dan `unmapped_skus: 0`.
  - Bila fallback dipanggil tanpa body, handler menggunakan `DEMO_MARKETPLACE_ORDERS.slice(0, 3)` dan menetapkan `total_orders: 3`, `processed_orders: 3`, `failed_orders: 0`.
  - Dengan demikian, relasi `Total == Sukses + Gagal` (`total_orders == processed_orders + failed_orders`) selalu konsisten secara matematis.
- **Verifikasi:**
  - Banner setelah impor sampel Shopee (3 pesanan): **"Total 3 pesanan diproses. Sukses: 3, Gagal / Duplikat: 0, SKU Belum Terpetakan: 0"** (3 == 3 + 0).
  - Terverifikasi otomatis lewat unit test `__tests__/demo-endpoints.test.ts` (test case `P3-1: marketplace import summary matches actual orders returned (fallback without body)` dan `P3-1: marketplace import with 3 orders in body computes exact counts`).

---

## P4-1 — Placeholder @CUSTOMER tampil mentah di UI (✅ SELESAI & TERVERIFIKASI)

**Gejala:** teks literal `@CUSTOMER` sempat terlihat pengguna di dua tempat antarmuka:
- Modal konfirmasi Dispatch Surat Jalan: "…stok gudang otomatis dipotong ke **@CUSTOMER**."
- Subtitle halaman Marketplace Omnichannel: "…packaging multiplier & potong stok otomatis ke **@CUSTOMER**."

**Akar masalah:** hardcoded placeholder token internal WMS `@CUSTOMER` terbawa langsung ke string user-facing tanpa formatting/lokalisasi.

**Solusi & Resolusi:**
Seluruh kemunculan literal `@CUSTOMER` yang tampil ke pengguna telah digantikan dengan terminologi humanis yang kontekstual:
- [`app/(app)/wms/delivery-orders/page.tsx:563`](../app/(app)/wms/delivery-orders/page.tsx#L563): Diganti menjadi `"stok gudang otomatis dipotong untuk pelanggan tujuan."`
- [`app/(app)/wms/marketplace/page.tsx`](../app/(app)/wms/marketplace/page.tsx) (baris 400, 629, 915, 1565, 2143): Diganti menjadi `"potong stok otomatis untuk pelanggan"`, `"mutasi stok ke pelanggan"`, dan `"pesanan pelanggan"`.
- [`app/(app)/wms/page.tsx:255-261`](../app/(app)/wms/page.tsx#L255-L261): Deteksi lokasi `startsWith("@CUSTOMER") || locName === "Pelanggan"` memetakan ke badge UI yang rapi dengan label **"Pelanggan Tujuan"** disertai icon `ArrowUpRight`.

**Kriteria terima:** tidak ada literal `@CUSTOMER` (atau placeholder `@…` lain) yang tampil di teks antarmuka UI.

**Verifikasi (Terverifikasi 2 Okt 2026):**
- Grep `app/` dan `components/` untuk string teks UI bebas dari literal `@CUSTOMER`.
- UI preview terverifikasi: Modal Dispatch Surat Jalan, halaman WMS overview, dan halaman Marketplace menampilkan teks bahasa Indonesia yang ramah dan profesional.

---

## P4-2 — Mutasi demo persisten in-memory (✅ SELESAI & TERVERIFIKASI)

**Gejala & Konteks Awal:**
- Create produk: sukses & baris ke-6 muncul, tapi reload halaman → kembali ke 5 karena `DEMO_PRODUCTS` sebelumnya merupakan konstanta statis tanpa persistensi mutasi.
- Dispatch Surat Jalan (DO): `POST …/dispatch` → respons 200 `SHIPPED`, tetapi query GET berikutnya masih berstatus `DRAFT` (`demo-data.ts` tidak mengubah status array).
- Receive transfer gudang: respons 200, tetapi status transfer tidak berubah persisten dalam sesi demo.

**Solusi & Resolusi (In-Memory State Mutability):**
`lib/api/demo-data.ts` dikonversi agar menggunakan mutable module-level state (`let`) dengan handler mutasi aktif:
1. Array data demo diubah dari konstanta beku menjadi mutable module-level `let`:
   - `let DEMO_PRODUCTS`
   - `let DEMO_DELIVERY_ORDERS`
   - `let DEMO_DELIVERY_ORDER_ITEMS`
   - `let DEMO_TRANSFERS`
   - `let DEMO_MARKETPLACE_BATCHES`
   - `let DEMO_MARKETPLACE_ORDERS`
   - `let DEMO_MARKETPLACE_SKU_MAPPINGS`
2. Handler POST meng-update array module-level tersebut secara real-time saat dipanggil:
   - `POST /products`: mem-prepend produk baru (`DEMO_PRODUCTS.unshift(...)`) dengan ID baru dan timestamp terkini.
   - `POST /wms/delivery-orders/:id/dispatch`: memutasi status DO menjadi `SHIPPED` pada elemen di `DEMO_DELIVERY_ORDERS`.
   - `POST /wms/transfers/:id/dispatch`: memutasi status transfer menjadi `IN_TRANSIT` pada `DEMO_TRANSFERS`.
   - `POST /wms/transfers/:id/receive`: memutasi status transfer menjadi `RECEIVED` pada `DEMO_TRANSFERS`.
   - `POST /wms/marketplace/import`: mem-prepend batch baru ke `DEMO_MARKETPLACE_BATCHES` dan pesanan ke `DEMO_MARKETPLACE_ORDERS`.

**Kriteria terima:** mutasi aksi pengguna dalam sesi demo tersimpan in-memory sehingga navigasi bolak-balik antar halaman tetap konsisten mencerminkan perubahan status terbaru.

**Verifikasi (Terverifikasi 2 Okt 2026):**
- Automated tests: `__tests__/demo-endpoints.test.ts` (test case `P0-1: POST /products creates and prepends product`, `P4-2: POST /wms/delivery-orders/:id/dispatch mutates DO status to SHIPPED`, dan `P4-2: POST /wms/transfers/:id/dispatch and receive mutates transfer status`) lolos 100%.
- Runtime preview: Pengujian alur pembuatan produk, pengiriman DO, transfer gudang, dan impor marketplace berjalan mulus dengan persistensi in-memory yang stabil.

---

## P5-1 — Refaktorisasi Naming & Terminologi Sidebar ERP ("Purchase Invoices" vs "Sales Invoices") (✅ SELESAI & TERVERIFIKASI)

**Latar belakang & Masalah:**
Di sidebar navigasi sebelumnya, menu di bawah grup **Procure-to-Pay** hanya berlabel `Invoices`, yang menyebabkan ambiguitas bagi pengguna dan sering tertukar dengan `Sales Invoices` di modul Order-to-Cash. Dalam standar akuntansi ERP modern (seperti SAP, Odoo, Paper.id):
- Tagihan dari vendor/supplier dalam alur Procure-to-Pay adalah **Purchase Invoices (Vendor Bills / Faktur Pembelian)**.
- Penagihan ke pelanggan dalam alur Order-to-Cash adalah **Sales Invoices (Customer Invoices / Faktur Penjualan)**.

**Solusi & Implementasi:**
1. **Sidebar Navigation ([`components/layout/Sidebar.tsx`](../components/layout/Sidebar.tsx)):**
   Menu Procure-to-Pay diubah dari `Invoices` menjadi `Purchase Invoices` (`href: '/dashboard/invoices'`), sementara Order-to-Cash tetap mempertahankan `Sales Invoices` (`href: '/sales-invoices'`). Menghapus label ambigu `Invoices` yang berdiri sendiri.
2. **Halaman Daftar Invoice ([`app/(app)/dashboard/invoices/page.tsx`](../app/(app)/dashboard/invoices/page.tsx)):**
   - Header utama diperbarui menjadi `Purchase Invoices (Vendor Bills)` dengan subjudul deskriptif: *"Daftar tagihan pembelian dan faktur masuk dari vendor/pemasok."*
   - Tombol aksi diperbarui menjadi `+ Buat Purchase Invoice Baru` mengarah ke `/dashboard/invoices/new`.
   - Header drawer detail disesuaikan menjadi `Purchase Invoice Detail` dan label `Purchase Invoice #`.
3. **Halaman Buat Invoice Baru ([`app/(app)/dashboard/invoices/new/page.tsx`](../app/(app)/dashboard/invoices/new/page.tsx)):**
   - Judul diubah menjadi `Buat Purchase Invoice Baru` dengan subjudul *"Catat tagihan vendor dan isi detail invoice pembelian untuk dikirim ke sistem."*
   - Label nomor invoice disesuaikan menjadi `Nomor Purchase Invoice (Vendor Bill #)`.
   - Tombol aksi menjadi `Simpan Purchase Invoice` dan navigasi kembali menjadi `Kembali ke Daftar Purchase Invoice`.
4. **Halaman Detail Invoice ([`app/(app)/dashboard/invoices/[id]/page.tsx`](../app/(app)/dashboard/invoices/[id]/page.tsx)):**
   - Menyelaraskan seluruh copy teks menjadi `Purchase invoice`, termasuk navigasi kembali `Kembali ke Daftar Purchase Invoice`, status error, feedback banner, peringatan verifikasi dokumen OCR, dan tombol aksi `Setujui Purchase Invoice` / `Tolak Purchase Invoice`.

**Verifikasi & Regresi (Commit `fc888a3`):**
- Ditambahkan regression test suite [`__tests__/sidebar.test.tsx`](../__tests__/sidebar.test.tsx) dengan 3 skenario uji:
  1. Memastikan link `Purchase Invoices` dirender dan mengarah ke `/dashboard/invoices`.
  2. Memastikan link `Sales Invoices` dirender dan mengarah ke `/sales-invoices`.
  3. Memastikan tidak ada label generik ambigu `Invoices` yang berdiri sendiri di sidebar (`screen.queryByRole('link', { name: /^Invoices$/i }) === null`).
- `npx jest __tests__/sidebar.test.tsx` → **PASS** (3 passed).
- `npx tsc --noEmit` → **0 error**.

---

## Status Akhir & Riwayat Eksekusi (Terverifikasi 2 Okt 2026)

Seluruh 12 item pekerjaan (P0-1 hingga P4-2, ditambah P5-1) telah diselesaikan, diuji, dan diverifikasi secara komprehensif pada baseline runtime preview:

| Kode | Kategori | Ringkasan Perbaikan | Status | Verifikasi |
|---|---|---|---|---|
| **P0-1** | P0 Blocker | Migrasi `/products` ke endpoint `/api/v1/products`, drawer drawer, currency formatting | ✅ LULUS | Interaktif preview & Jest pass |
| **P1-1** | P1 Defect | Eliminasi kebocoran raw JSON error di `/sales-invoices` via `body.error` | ✅ LULUS | Error handling ramah di UI |
| **P1-2** | P1 Defect | Penyeragaman tema Surat Jalan (DO) ke mode terang (0 `bg-zinc-950`) | ✅ LULUS | CSS & styling konsisten |
| **P1-3** | P1 Defect | 14+ endpoint fallback demo O2C & Procure-to-Pay (0 HTTP 502) | ✅ LULUS | Probe 16 endpoint 200 OK |
| **P1-3b**| P1 Defect | Standardisasi RFC-4122 v4 UUID & field wajib vendor di data demo | ✅ LULUS | `demo-endpoints.test.ts` PASS |
| **P2-1** | P2 UX | Penggantian `window.confirm`/`alert` native dengan modal internal | ✅ LULUS | Thread aman & dialog non-blocking |
| **P2-2** | P2 Layout | Penambahan gap 12px antara harga dan stok pada kartu produk POS | ✅ LULUS | Pengukuran bounding box DOM OK |
| **P2-3** | P2 Quality| Eliminasi noise 502 di console (`/subscription` & `/ai/permissions`) | ✅ LULUS | Console log bersih saat navigasi |
| **P3-1** | P3 Logic | Kalkulasi dinamis ringkasan impor marketplace (`Total == Sukses + Gagal`) | ✅ LULUS | Banner math konsisten & Jest pass |
| **P4-1** | P4 UI | Pembersihan placeholder `@CUSTOMER` menjadi teks humanis | ✅ LULUS | Grep UI 0 hit, modal/page bersih |
| **P4-2** | P4 Quality| Persistensi mutasi data demo in-memory (`let` mutable state) | ✅ LULUS | Unit tests & interaksi real-time |
| **P5-1** | P5 Domain | Refaktorisasi terminologi navigasi "Purchase Invoices" vs "Sales Invoices" | ✅ LULUS | `sidebar.test.tsx` PASS & typecheck OK |

**Ringkasan Status Baseline:**
- `npx tsc --noEmit`: **0 error**
- `npx jest`: **14 test suites / 129 tests PASS**
- Go Backend (`backend/go-core`): `go build ./...`, `go vet ./...`, `go test ./internal/...` **PASS**
- Runtime Preview: Seluruh alur kerja (P2P, O2C, WMS, POS, Accounting, Settings) berfungsi dengan mulus.
