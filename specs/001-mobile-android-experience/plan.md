# Implementation Plan: Mobile Android Experience

**Feature**: Mobile Android Experience (`specs/001-mobile-android-experience`)  
**Architecture Guidelines**: Next.js 15 App Router, Tailwind CSS v4, Lucide Icons, Hallmark Design Rules  

## Phase 1: CSS & Shell Foundation
- Update `app/globals.css`:
  - Enforce `overflow-x: clip` on `html` and `body` (Hallmark rule 4).
  - Add tap-highlight cleanup `-webkit-tap-highlight-color: transparent`.
  - Add Android touch-target utility classes and input zoom prevention styling.
- Update `app/(app)/layout.tsx`:
  - Make mobile header render smoothly via CSS media queries (`md:hidden`) rather than delaying until JS `isMobile` hydrates.
  - Add mobile header bar with clean Tayooli branding, touch-friendly hamburger button (min 48px target), and current page title or context badge.
  - Ensure main container respects safe-area insets.
- Update `components/layout/Sidebar.tsx`:
  - Separate desktop sidebar (`hidden md:flex`) and mobile drawer (`md:hidden` portal/fixed) using CSS classes to eliminate SSR hydration layout jumps.
  - Optimize mobile drawer menu items for touch: min-h-[44px], clear active state, easy close action.

## Phase 2: Common Components & Touch Ergonomics
- `components/ui/data-table.tsx`:
  - Ensure smooth horizontal scrolling container (`overflow-x-auto` with `-webkit-overflow-scrolling: touch`) and subtle shadow/scrollbar cues so users know content scrolls horizontally.
  - Stack pagination controls on small screens (page info above buttons or wrapped neatly without breaking).
- Form inputs & Comboboxes (`ProductCombobox.tsx`, `CategoryCombobox.tsx`, `components/ui/input.tsx`):
  - Ensure mobile text size is at least 16px (`text-base sm:text-sm`) to prevent browser auto-zoom.
  - Ensure dropdown list items have minimum 44px height for finger tapping.
- Modals & Drawers (`components/ui/drawer.tsx`, `components/ui/ExportModal.tsx`):
  - Ensure full width on mobile screens (`w-full sm:max-w-md`), scrollable body, sticky action footer, and safe area padding.

## Phase 3: Core Modules Optimization for Mobile / Android
1. **POS Kasir (`app/(app)/pos/page.tsx`)**:
   - Header: Responsive layout, stack or horizontal scroll for quick filters.
   - Product Grid: 2 columns on mobile (`grid-cols-2 sm:grid-cols-3 md:grid-cols-4`), clear product card tap targets.
   - Floating Cart Bar: Enhanced visibility, clear contrast, easy thumb tap.
   - Cart Sheet / Modal: Full screen or bottom sheet with large (+/-) buttons and clear checkout CTA.
   - Payment Modal: Responsive grid for Quick Cash buttons, full width payment method selectors, comfortable input fields.
2. **Pemindai Barcode (`app/(app)/wms/scanner/page.tsx`)**:
   - Camera viewport: Responsive aspect ratio on portrait Android devices.
   - Mode toggles: Touch-friendly segmented control or pill buttons.
   - Scan results card: Prominent display without pushing controls off-screen.
3. **Gudang & Stok / Arus Barang (`app/(app)/wms/arus-barang/page.tsx`, `app/(app)/wms/page.tsx`)**:
   - MASUK | KELUAR toggle: Large touch target, clear active indicator.
   - Action buttons (+ Buat Surat Jalan, + Catat Inbound): Stack or wrap nicely on narrow screens.
   - Filter bars: Responsive flex-wrap with min-h-[44px] controls.
4. **Dasbor (`app/(app)/dashboard/page.tsx`)**:
   - KPI cards: 1 column on mobile, 2 columns on tablet, 4 on desktop.
   - Metric charts: Responsive container with touch tooltips.

## Phase 4: Verification & Code Review
- Run Jest test suite (`npm test`).
- Run TypeScript typecheck (`npx tsc --noEmit`).
- Verify production build (`npm run build`).
- Execute two-axis Code Review (Standards & Spec) using parallel sub-agents or thorough checks before final commit.
