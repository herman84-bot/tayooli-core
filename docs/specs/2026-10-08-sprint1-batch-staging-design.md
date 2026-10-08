# Sprint 1 — Desain Teknis: Batch Wajib, Staging → Putaway, Kategori, Rak Default, Pelaku

Turunan dari PRD master §3.5/§6 Sprint 1 dan ADR-014. Ditulis **sebelum** kode, sesuai GLOBAL-ENGINEERING-QA-POLICY (§2 flow, §3 invariants).

Pola acuan yang diadopsi:
- **sentry-wms §1.1:** bin `Staging` tidak bisa diambil untuk order.
- **sentry-wms §1.2:** inbound 2 langkah, Receive → Staging lalu Putaway → rak.
- **sentry-wms §3.1:** semua mutasi dalam 1 transaksi dengan tenant context dan lock.
- **sentry-wms §3.2:** audit log berantai hash.
- **OCA §1.3:** FEFO, yaitu stok diurutkan `expiry_date ASC`.

## 1. Fakta kode saat ini (recon 2026-10-08)

| Fakta | Bukti |
|---|---|
| Stok = SUM ledger `stock_movements`; tidak ada tabel saldo WMS | `wms_repo.go:921-974` |
| Hanya 1 SQL insert ledger, dipakai 3 fungsi (A `CreateStockMovement`, B `DeductLocationStock`, C `insertReceiptMovementTx`) | `wms_repo.go:882`, `wms_receipt_repo.go:350` |
| 11 jalur penulis stok: receipt post/cancel, transfer dispatch/receive, DO dispatch, opname complete, scrap, marketplace import, SKU-mapping backfill, POS checkout, legacy `/inventory` | laporan recon subagent |
| `stock_receipt_items` UNIQUE(receipt_id, product_id): 1 produk tidak bisa >1 batch per penerimaan | `\d stock_receipt_items` |
| **Bug:** `AuditLogRepo` insert kolom `details` (tidak ada) dan tidak mengisi `current_hash NOT NULL`, sehingga semua audit log gagal diam-diam | `audit_log_repo.go:13`, `\d audit_logs` |
| **Bug:** POS menulis mutasi source = dest (stok ledger tidak turun) dan menelan error | `pos.go:179-203` |

## 2. Model data (migrasi 033)

- `location_type` += `STAGING_INBOUND`, `STAGING_OUTBOUND`, `QUARANTINE`.
  - Staging inbound dibuat per gudang (kode `STG-IN`), dengan pembuatan yang aman dari race (`ON CONFLICT DO NOTHING`).
- `stock_batches(id, tenant_id, product_id, batch_number, expiry_date NULL, source_receipt_id NULL, status RELEASED|ON_HOLD, is_legacy, created_by, created_at)`.
  - UNIQUE(tenant_id, product_id, batch_number). RLS FORCE.
- `stock_movements.batch_id`:
  - Backfill batch `LEGACY` per (tenant, product) untuk mutasi lama, lalu `SET NOT NULL` + FK.
  - Index `(tenant_id, batch_id)` dan `(tenant_id, reference_type, reference_id)`.
- `stock_receipt_items`: tambah `batch_number`, `expiry_date`, `batch_id`. UNIQUE lama diganti UNIQUE(receipt_id, product_id, COALESCE(batch_number,'')).
- `stock_receipts`: tambah `released_by`, `released_at` (PDF-06).
- `product_categories(id, tenant_id, name)` dengan UNIQUE(tenant_id, lower(name)), plus `products.category_id` NULL (CR-01).
- `product_default_locations(tenant_id, product_id, warehouse_id, location_id)` dengan PK(tenant, product, warehouse) (CR-03).
- `wms_settings(tenant_id PK, require_release_approval bool default false, updated_by, updated_at)` (PDF-06).
- `audit_logs.details JSONB`. Repo menghitung `prev_hash`/`current_hash` (SHA-256 berantai per tenant, dengan advisory lock).

## 3. Alur

### 3.1 Barang Masuk (MASUK)
```
Draft (baris: produk, qty baik, qty rusak, no batch?, exp?)
  → POST /receipts/{id}/post   [1 tx, header FOR UPDATE]
      per baris:
        batch = get-or-create(produk, batch_number || "AUTO-<GR>-<baris>")
                status = ON_HOLD jika wms_settings.require_release_approval, selain itu RELEASED
                expiry bentrok dengan batch yang sudah ada → ditolak (validasi)
        qty baik  : @SUMBER → STG-IN gudang      (batch)
        qty rusak : @SUMBER → @SCRAP             (batch)   [Sprint 2 pindah ke QUARANTINE]
      audit_logs: receipt.posted (dalam tx yang sama)
  → GET  /putaway/pending?warehouse_id=   (saldo STG-IN per produk+batch + saran rak)
  → POST /putaway {product, batch, qty, dest_location, reason?}   [1 tx, lock, cek saldo batch]
        STG-IN → rak INTERNAL gudang yang sama (batch)
        bila produk punya rak default & dest ≠ default → reason wajib
        audit_logs: putaway.confirmed
  → (opsional) POST /receipts/{id}/release  [owner/admin/regional_manager]
        batch ON_HOLD milik receipt → RELEASED, released_by/at, audit_logs
```
- Saran rak, berurutan: rak default produk di gudang itu → `dest_location_id` receipt → kosong.
- Batal receipt POSTED: membalik **setiap** mutasi GR asli (lokasi + batch yang sama) dalam 1 tx dengan lock terurut. Bila stok sudah di-putaway atau terpakai, ditolak `ErrStockReceiptStockConsumed`.

### 3.2 Keluar / mutasi lain — alokasi batch
`DeductLocationStock` (B) menjadi alokator FEFO dalam 1 tx dengan advisory lock (tenant, lokasi, produk):
- Bila `mov.BatchID` diisi: cek saldo batch itu saja.
- Bila kosong: saldo per batch di lokasi diurutkan `expiry ASC NULLS LAST, created_at ASC`. Qty dipecah menjadi beberapa mutasi bersufiks `-B1..n`.
- Mode `sale` (DO, transfer, marketplace, POS) melewati batch `ON_HOLD`. Mode `adjust` (scrap, opname) menyertakannya.
- Total saldo kurang → `ErrInsufficientStock`, tanpa satu baris pun ditulis.

Per jalur:
| Jalur | Batch |
|---|---|
| DO dispatch | FEFO dari rak. Lokasi wajib tipe INTERNAL; staging ditolak |
| Transfer dispatch | FEFO. Receive memakai cermin per batch dari mutasi `TR-DISP` transfer itu |
| Receipt tipe TRANSFER dengan `transfer_id` | Ambil batch dari mutasi `TR-DISP` transfer itu, tidak membuat AUTO |
| Scrap | FEFO (mode adjust) |
| Marketplace / SKU mapping backfill | FEFO (lewat B) |
| Opname kurang | FEFO (mode adjust), sebelumnya tanpa cek stok |
| Opname lebih | Batch baru `ADJ-<no opname>` |
| POS | FEFO lintas rak INTERNAL gudang, error tidak lagi ditelan |
| Legacy `POST /inventory` | Tidak menulis ledger; di luar aturan batch (tabel legacy) |

`CreateStockMovement` (A) menolak `BatchID == nil` (`ErrBatchRequired`). DB `NOT NULL` adalah lapis kedua.

### 3.3 Lacak (KO-1c)
- `GET /trace/batch/{id}`: info batch, semua mutasi beserta nomor dokumen, dan saldo per lokasi.
- `GET /trace/receipt/{id}` (maju): baris → batch → mutasi keluar (DO/POS/transfer/marketplace/scrap) → pelanggan/tujuan.
- `GET /trace/delivery-order/{id}` (mundur): mutasi DO-SHIP → batch, exp, rak asal, GR sumber, pemasok.

## 4. Invariants (wajib jadi test otomatis)

| ID | Invariant | Test |
|---|---|---|
| I1 | Setiap baris `stock_movements` punya `batch_id` | DB NOT NULL + test A menolak nil + integration |
| I2 | Saldo batch di lokasi fisik tidak pernah negatif setelah operasi apa pun | property test FEFO + integration |
| I3 | Rekonsiliasi: Σ masuk batch = Σ keluar + Σ saldo semua lokasi | property test acak |
| I4 | FEFO: batch exp lebih awal habis lebih dulu; batch ON_HOLD tidak dialokasikan ke penjualan | unit + property |
| I5 | Post ganda / putaway melebihi saldo staging tidak menghasilkan efek ganda | integration (concurrent) |
| I6 | Stok STAGING tidak bisa dikirim lewat DO | unit |
| I7 | Setiap transisi receipt/putaway/release menulis audit_log dengan user_id ≠ nil, dalam tx yang sama | integration |
| I8 | Pelaku nil (uuid.Nil) ditolak | unit |
| I9 | Batal receipt setelah putaway ditolak dan tidak ada partial state | integration |
| I10 | Tenant A tidak bisa membaca/menulis batch, kategori, atau setting tenant B (RLS) | integration (role non-superuser) |

## 5. Di luar Sprint 1 (sengaja)
- QC sampling/full, Quarantine untuk barang rusak, dan BAK: Sprint 2.
- Wave, picking, packing: Sprint 3. KPI dan rekonsiliasi laporan: Sprint 4.
- Cetak label (FE-02): ditunda ke akhir Sprint 1 bila waktu cukup; akan dilaporkan jujur.
