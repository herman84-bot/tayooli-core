# Task 6 Report: Frontend Pallet LPN & Cetak Stiker Barcode (`PrintLPNLabel.tsx` & LPN Modal)

**Status:** DONE  
**Commit:** `f966397`  
**Date:** 2026-10-08  

## Summary of Changes

1. **Created `components/wms/PrintLPNLabel.tsx`**:
   - Thermal sticker label print component tailored for 100x150 mm (4x6 inch) / A6 thermal label printers with native `@media print` layout and page styling (`@page { size: 100mm 150mm; margin: 0; }`).
   - Pure vector Code 128 (Subset B) ISO/IEC 15417 SVG barcode rendering (`LPNBarcodeSVG`), producing high-contrast bars for LPN codes (e.g. `LPN-20261008-0001`).
   - Header with company branding "TAYOOLI ERP - LOGISTICS", warehouse name, and generation date.
   - Large legible LPN code text alongside Pallet Type badges (`WOODEN`, `PLASTIC`, `METAL`, `CAGE`).
   - Storage location badge (`location_code` / `location_name`).
   - Total weight vs max weight capacity specs with visual progress indicator.
   - Manifest table of contents detailing lot batch numbers, SKU codes, product names, expiry dates, and quantities on the pallet.
   - Action controls: "Cetak Label" triggering `window.print()` and "Tutup" button.

2. **Created `components/wms/LPNManagementModal.tsx`**:
   - Comprehensive containerization modal for managing LPN pallets in the warehouse.
   - List view with status filtering (`ALL`, `STAGED`, `STORED`, `PICKED`), quick search, and active pallet cards.
   - "Lihat Isi" view inspecting batch line items via `useStockLPNDetail`.
   - "Tambah Item" view adding product batches and quantities to pallets via `useAddLPNItem` with quick-pick from staging inbound goods (`usePutawayPending`).
   - "Cetak Label Palet" action launching thermal label print dialog (`PrintLPNLabel`).
   - "Buat Palet LPN Baru" form creating pallets with location selection (`useWarehouseLocations`), pallet types, max capacity (default 1000 kg), and notes via `useCreateLPN`.

3. **Integrated into `app/(app)/wms/arus-barang/page.tsx`**:
   - Added "Kelola Palet (LPN)" action button to Inbound Receiving (`InboundReceivingSubView`) toolbar.
   - Added "Kelola Palet (LPN)" action button to Inbound Receipt Detail (`InboundReceiptDetail`) header.
   - Rendered `<LPNManagementModal>` connected to the currently selected warehouse.

4. **Created `__tests__/lpn-print-label.test.tsx`**:
   - 13 comprehensive unit tests covering:
     - Code 128 bar pattern generation and module validity.
     - SVG barcode rendering and accessibility attributes.
     - Header, LPN code, pallet specs, location, weight, and batch items table rendering.
     - Empty pallet fallback notice.
     - `window.print()` trigger and modal close callbacks.
     - `LPNManagementModal` visibility and LPN list rendering.
     - "Buat Palet LPN Baru" creation submission via `useCreateLPN`.
     - "Lihat Isi" detail expansion.
     - "Tambah Item" batch allocation submission via `useAddLPNItem`.
     - Thermal label print invocation from modal.

## Verification Evidence

- `npx tsc --noEmit`: 0 errors (PASS)
- `npm test -- __tests__/lpn-print-label.test.tsx`: 13 passed, 0 failed (PASS)
- `npm test -- __tests__/dock-board.test.tsx`: 11 passed, 0 failed (PASS)
- `npm test -- __tests__/wms-validation.test.ts`: 40 passed, 0 failed (PASS)
