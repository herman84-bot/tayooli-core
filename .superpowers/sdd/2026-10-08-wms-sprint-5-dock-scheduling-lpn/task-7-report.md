# Task 7 Report: Frontend PDA Scanner Mode Pallet Putaway (`/wms/scanner`)

**Status:** DONE  
**Commit:** `753673c`  
**Date:** 2026-10-08  

## Summary of Changes

1. **Extended `ScannerMode` in `app/(app)/wms/scanner/page.tsx`**:
   - Added `"PALLET_LPN"` to `ScannerMode`:
     ```ts
     type ScannerMode = "PUTAWAY" | "OUTBOUND" | "PRICE_CHECK" | "LOADING_TRUCK" | "PALLET_LPN"
     ```
   - Imported `useStockLPNs`, `useStockLPNDetail`, `useMoveLPN` from `@/hooks/useWMSDocksAndLPNs`.

2. **Added State & Hook Integrations for `PALLET_LPN` Mode**:
   - `scannedLPNCode: string | null`
   - `matchedLPNId: string | null`
   - `targetRackLocation: string | null`
   - `targetRackLocationId: string | null`
   - `lpnPutawaySuccessMessage: string | null`
   - `lpnPutawayErrorMessage: string | null`
   - Hooks: `useStockLPNs(activeWarehouseId)`, `useStockLPNDetail(matchedLPNId)`, and `useMoveLPN()`.

3. **Implemented 2-Step Scanning Logic in `handleBarcodeDetected`**:
   - **Step 1 (Scan Palet LPN)**:
     - When `!scannedLPNCode`:
       - Scanned barcode is matched against `stockLPNs` list or verified as starting with `"LPN-"`.
       - Sets `scannedLPNCode` and `matchedLPNId`.
       - Plays success tone via `playTone("success")`.
       - Displays notification banner: `"Palet [LPN-XXXX] terpilih! Silakan scan barcode Rak Tujuan."`
       - If invalid, plays error tone and shows error banner.
   - **Step 2 (Scan Rak Tujuan & Pindahkan Seluruh Isi Palet)**:
     - When `scannedLPNCode` is set:
       - Validates scanned code against warehouse rack locations list (`locations`).
       - Dispatches `moveLPNMutation.mutate({ id: matchedLPNId, data: { target_location_id: loc.id } })`.
       - On success: Plays success tone, displays banner `"BERHASIL: Seluruh isi palet [LPN-XXX] berhasil dipindahkan ke rak [RAK-XXX]!"`, and resets Step 1 & 2 for next pallet.
       - On error: Plays error tone and displays error message banner.

4. **Added UI Components in PDA Scanner View**:
   - Added 5th tab button in the mode selector: `Palet (LPN Putaway)` with `Box` icon.
   - Added mode instruction banner explaining the 2-step putaway workflow for forklift operators.
   - Step progress indicator cards:
     - Langkah 1: Scan Barcode Palet LPN (status: `Belum Terpilih` / `Sudah Terpilih`).
     - Langkah 2: Scan Barcode Rak Lokasi Tujuan (status: `Menunggu Langkah 1` / `Siap Scan Rak Tujuan` / `Rak Diset`).
   - Selected Pallet Info & Line Items Table:
     - Pallet Info Badge displaying: Kode LPN, Tipe Palet, Lokasi Saat Ini, Total Berat kg.
     - Summary table of packed batch line items: No, SKU, Nama Barang, Batch, Jumlah.
     - Action button "Ganti Palet" allowing operator to reset pallet selection immediately.
   - Added quick test simulation button for sample pallet `[Palet] LPN-0001`.

5. **Created `__tests__/scanner-lpn.test.tsx`**:
   - 8 comprehensive unit tests covering:
     - Mode switch button rendering for `Palet (LPN Putaway)`.
     - Switching to `PALLET_LPN` mode and displaying step indicators.
     - Step 1 error feedback when scanning an invalid non-LPN code.
     - Step 1 pallet scan matching, pallet badge rendering, and line items display.
     - "Ganti Palet" action button resetting pallet selection.
     - Step 2 error handling when scanning an invalid rack location.
     - Step 2 successful rack scan dispatching `moveLPN` mutation, playing success tone, and resetting for next pallet.
     - Step 2 mutation error handling displaying error banner and playing error tone.

## Verification Evidence

- `npx tsc --noEmit`: 0 errors (PASS)
- `npm test -- __tests__/scanner-lpn.test.tsx`: 8 passed, 0 failed (PASS)
- `npm test -- __tests__/scanner-loading.test.tsx`: 6 passed, 0 failed (PASS)
- `npm test -- __tests__/scanner`: 14 passed, 0 failed (PASS)
