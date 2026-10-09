# Task 5 Report: Frontend Shipping Manifests Management Panel, Signature Canvas, and Print View

## Status
DONE

## Commit
`bcd2407`
`feat(wms): add shipping manifests management panel, signature canvas, and print view`

## Changed
- `components/wms/SignatureCanvas.tsx`:
  - Interactive HTML5 canvas capturing digital signatures with mouse & touch support.
  - Implements `SignatureCanvasProps`: `onSave`, `disabled`, `height`, `className`.
  - Pen stroke styling (lineWidth 2.5, round lineCap & lineJoin, `#1E293B`).
  - Stroke array capture exporting to trimmed scalable SVG string or PNG dataURL fallback.
  - Clear/reset button, non-empty validation blocking empty signature submissions.
- `components/wms/PrintShippingManifest.tsx`:
  - Formatted for standard A4 printing (`print:block`, `@media print` scoped isolation).
  - Displays company branding: "MANIFEST SERAH TERIMA PENGIRIMAN", manifest number, date, expedition, driver, vehicle plate, driver phone.
  - Detailed table of delivery orders: No, No. DO, Pelanggan, Kota Tujuan, Berat (kg), Tipe Kemasan, Status Scan Loading.
  - Live summary: Total Koli, Total Berat (kg).
  - Dual signature box section: "Diserahkan Oleh (Staf Gudang)" and "Diterima Oleh (Sopir Ekspedisi)" with `driver_signature_svg` rendering.
  - Trigger buttons: `window.print()` and `onClose()`.
- `components/wms/ShippingManifestsPanel.tsx`:
  - Outbound manifest management panel using TanStack Query hooks (`useShippingManifests`, `useCreateShippingManifest`, `useShippingManifestDetail`, `useScanLoadingDO`, `useDispatchShippingManifest`, `useDeliveryOrders`, `useWarehouses`).
  - Metrics cards: Total Manifest, Staged (Siap Muat), Loaded (Terpindai Penuh), Dispatched (Berangkat).
  - Status filters (`ALL`, `STAGED`, `LOADED`, `DISPATCHED`) and live search filter.
  - "Buat Manifest Baru" modal:
    - Warehouse selector, expedition selector (presets & custom), driver name, vehicle plate, phone, notes.
    - Delivery order multi-selection checkbox table (filtering `PACKED` / `CONFIRMED` DOs).
    - Real-time sum of packages and weight.
  - Manifests data table:
    - Columns: No. Manifest, Ekspedisi, Sopir / Plat, Koli, Berat, Status, Tanggal Dibuat, Aksi.
    - Status badges: `STAGED`, `LOADED`, `DISPATCHED`, `CANCELLED`.
  - Actions:
    - "Scan / Detail" loading drawer: real-time loading progress bar (X / Y koli, percentage), barcode input scanner, single-click "Tandai Termuat".
    - "Dispatch" modal: integrates `SignatureCanvas` for courier digital signature capture and mutation submission.
    - "Cetak Manifest": opens A4 printable modal.
- `components/wms/DeliveryOrdersPanel.tsx`:
  - Added tab switcher at top: `[Surat Jalan (DO) | Manifest Ekspedisi]`.
  - Seamlessly renders `ShippingManifestsPanel` with selected warehouse context when `MANIFEST` tab is active.
  - Preserved all existing DO creation, filtering, wave release, and dispatch workflows intact.
- `__tests__/wms-manifest-components.test.tsx`:
  - 11 unit tests covering canvas drawing & export, print view formatting, manifest panel filtering/modals/loading-scan/dispatch, and panel tab toggle.

## Verified
- `npx tsc --noEmit`: 0 errors (PASS).
- `npm test -- __tests__/wms-manifest-components.test.tsx`: 11/11 tests PASS.
- `npm test -- __tests__/wms-outbound-components.test.tsx __tests__/wms-manifest-components.test.tsx`: 21/21 tests PASS.
