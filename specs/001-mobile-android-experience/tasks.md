# Tasks: Mobile Android Experience

**Feature**: Mobile Android Experience (`specs/001-mobile-android-experience`)

- [x] Task 1: Viewport & Base CSS Mobile Optimization
  - [x] 1.1 Add `overflow-x: clip` to `html` and `body` in `app/globals.css`.
  - [x] 1.2 Add touch tap-highlight removal and mobile touch utility classes.
  - [x] 1.3 Add form input zoom prevention styling (font-size >= 16px on mobile).

- [x] Task 2: Responsive Shell & Navigation Stabilization
  - [x] 2.1 Refactor `app/(app)/layout.tsx` to render mobile topbar via CSS classes (`md:hidden`) with proper touch target (48px) and safe-area insets.
  - [x] 2.2 Refactor `components/layout/Sidebar.tsx` to use CSS visibility for desktop vs mobile drawer (`hidden md:flex` vs `fixed md:hidden`) eliminating hydration layout shifts.
  - [x] 2.3 Ensure drawer close button and nav links have min 44px touch targets and clear active states.

- [x] Task 3: Common UI Components Touch Ergonomics
  - [x] 3.1 Review and enhance `components/ui/data-table.tsx` for touch horizontal scrolling and responsive pagination.
  - [x] 3.2 Review and enhance `components/ui/drawer.tsx` and modal components for mobile viewports.
  - [x] 3.3 Ensure comboboxes and inputs have touch-friendly hit areas and no auto-zoom.

- [x] Task 4: High-Impact Modules Mobile Refinement
  - [x] 4.1 POS Kasir (`app/(app)/pos/page.tsx`): Enhance mobile layout, product grid (2 cols on mobile), floating cart bar, mobile cart drawer/modal, payment dialog.
  - [x] 4.2 Barcode Scanner (`app/(app)/wms/scanner/page.tsx`): Enhance camera viewfinder and control ergonomics on portrait Android devices.
  - [x] 4.3 Arus Barang & WMS (`app/(app)/wms/arus-barang/page.tsx`, etc.): Ensure MASUK | KELUAR toggle and action buttons wrap cleanly on mobile.
  - [x] 4.4 Dasbor (`app/(app)/dashboard/page.tsx`): Verify KPI cards and chart responsiveness.

- [x] Task 5: Testing, Build & Two-Axis Code Review
  - [x] 5.1 Run full Jest test suite (`npm test`).
  - [x] 5.2 Run TypeScript compiler check (`npx tsc --noEmit`).
  - [x] 5.3 Run production build check (`npm run build`).
  - [x] 5.4 Perform Standards & Spec Code Review before committing.
