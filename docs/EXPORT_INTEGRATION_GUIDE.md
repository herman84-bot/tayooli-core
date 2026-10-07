# Export Feature Integration Guide

This guide shows how to add export functionality to any Tayooli Core module.

---

## Quick Start: 4 Steps

### 1. Import Components & Types

```tsx
import { ExportModal, ExportButton, type ExportFilter } from "@/components/ui/ExportModal"
import type { ExportColumn } from "@/lib/export"
```

### 2. Define Export Columns

Each column maps a data field to an export cell:

```tsx
const productExportColumns: ExportColumn<Product>[] = [
  { header: "SKU", value: (p) => p.sku, width: 16 },
  { header: "Nama", value: (p) => p.name, width: 32 },
  { header: "Harga (Rp)", value: (p) => p.price, width: 16, align: "right" },
]
```

**Column properties**:
- `header`: Display name (Excel, CSV, PDF)
- `value(row)`: Extract cell content from row
- `width?`: Character width (Excel column width, PDF hint)
- `align?`: 'left' | 'right' | 'center' (default: 'left')

### 3. Define Filters (Optional)

Filters allow users to refine exported data in the modal:

**Date Range Filter**:
```tsx
{
  type: "dateRange",
  id: "created",
  label: "Tanggal dibuat",
  getDate: (row) => row.created_at,  // Must return ISO string or null/undefined
}
```

**Select Dropdown Filter**:
```tsx
{
  type: "select",
  id: "status",
  label: "Status",
  options: [
    { value: "DRAFT", label: "Draf" },
    { value: "POSTED", label: "Sudah Posted" },
  ],
  match: (row, selectedValue) => row.status === selectedValue,  // Predicate
}
```

### 4. Add Modal & Button

```tsx
const [exporting, setExporting] = useState(false)

return (
  <>
    {/* Export button in toolbar */}
    <ExportButton onClick={() => setExporting(true)} disabled={data.length === 0} />

    {/* Export modal */}
    <ExportModal<Product>
      open={exporting}
      onClose={() => setExporting(false)}
      title="Katalog Produk"           // Page title
      filename="katalog-produk"        // Filename base (auto-dated)
      columns={productExportColumns}
      allRows={products}               // Unfiltered data
      visibleRows={filteredProducts}   // Optional: page-filtered data
      visibleSummary={[                // Optional: active filter labels
        `Pencarian: "${searchQuery}"`,
      ]}
      filters={productExportFilters}    // Optional: advanced filters
    />
  </>
)
```

---

## Example 1: WMS Stock Summary Export

**File**: `app/(app)/wms/page.tsx`

```tsx
import { ExportModal, ExportButton } from "@/components/ui/ExportModal"

export default function WMSDashboardPage() {
  const { data: stockSummary = [] } = useWMSStock()
  const [exporting, setExporting] = useState(false)

  const stockExportColumns: ExportColumn<StockSummary>[] = [
    { header: "SKU", value: (s) => s.sku, width: 16 },
    { header: "Nama Produk", value: (s) => s.product_name, width: 32 },
    { header: "Gudang", value: (s) => s.warehouse_name || "-", width: 22 },
    { header: "Lokasi", value: (s) => s.location_code || "-", width: 18 },
    { header: "Qty", value: (s) => Number(s.quantity), width: 12, align: "right" },
  ]

  const stockExportFilters: ExportFilter<StockSummary>[] = [
    {
      type: "select",
      id: "warehouse",
      label: "Gudang",
      options: warehouses.map((w) => ({ value: w.id, label: w.name })),
      match: (s, v) => s.warehouse_id === v,
    },
  ]

  return (
    <>
      <ExportButton onClick={() => setExporting(true)} disabled={stockSummary.length === 0} />

      <ExportModal<StockSummary>
        open={exporting}
        onClose={() => setExporting(false)}
        title="Stok & Lokasi Gudang"
        filename="stok-gudang"
        allRows={stockSummary}
        visibleRows={stockSummary}  // No page-level filters in this view
        columns={stockExportColumns}
        filters={stockExportFilters}
      />
    </>
  )
}
```

---

## Example 2: POS Order History Export

**File**: `app/(app)/pos/page.tsx`

```tsx
import { ExportModal, ExportButton } from "@/components/ui/ExportModal"

export default function POSPage() {
  const { data: posOrders = [] } = usePOSOrders(30)
  const [exporting, setExporting] = useState(false)

  const fmtTime = (iso: string): string => {
    const d = new Date(iso)
    return d.toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit", second: "2-digit" })
  }

  const posExportColumns: ExportColumn<POSOrder>[] = [
    { header: "No. Transaksi", value: (o) => o.order_number, width: 20 },
    { header: "Waktu", value: (o) => fmtTime(o.created_at), width: 16 },
    { header: "Pelanggan", value: (o) => o.customer_name || "Walk-in", width: 22 },
    { header: "Metode Bayar", value: (o) => o.payment_method, width: 18 },
    { header: "Subtotal (Rp)", value: (o) => o.subtotal, width: 16, align: "right" },
    { header: "Pajak (Rp)", value: (o) => o.tax_amount, width: 14, align: "right" },
    { header: "Diskon (Rp)", value: (o) => o.discount_amount, width: 14, align: "right" },
    { header: "Total (Rp)", value: (o) => o.total_amount, width: 16, align: "right" },
    { header: "Status", value: (o) => o.status, width: 14 },
  ]

  const posExportFilters: ExportFilter<POSOrder>[] = [
    { type: "dateRange", id: "date", label: "Tanggal transaksi", getDate: (o) => o.created_at },
    {
      type: "select",
      id: "method",
      label: "Metode bayar",
      options: [
        { value: "CASH", label: "Tunai" },
        { value: "CARD", label: "Kartu" },
        { value: "QRIS", label: "QRIS" },
        { value: "OTHER", label: "Lainnya" },
      ],
      match: (o, v) => o.payment_method === v,
    },
  ]

  return (
    <>
      <ExportButton onClick={() => setExporting(true)} disabled={posOrders.length === 0} />

      <ExportModal<POSOrder>
        open={exporting}
        onClose={() => setExporting(false)}
        title="Riwayat Transaksi POS"
        filename="riwayat-pos"
        allRows={posOrders}
        visibleRows={posOrders}
        columns={posExportColumns}
        filters={posExportFilters}
      />
    </>
  )
}
```

---

## Formatting Helpers

### Date Formatting

```tsx
const fmtDate = (iso?: string | null): string => {
  if (!iso) return ""
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? "" : d.toLocaleDateString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  })
}
```

### Number Formatting

```tsx
const fmtNumber = (n: unknown): number => {
  const val = typeof n === "number" ? n : Number(n) || 0
  return Number.isFinite(val) ? val : 0
}

// In column:
{ header: "Qty", value: (r) => fmtNumber(r.quantity), align: "right" }
```

### Currency

Return a plain number, not a formatted string ("Rp 15.000"), so Excel keeps the
cell numeric for SUM/pivot. Put the unit in the header instead:

```tsx
{ header: "Total (Rp)", value: (r) => Math.round(Number(r.total_amount) || 0), align: "right" }
```

---

## Data Flow

```
User clicks "Ekspor"
  ↓
ExportModal opens
  ↓
Choose filters (date range, select) [optional]
Choose scope: "Visible" (page data) or "Custom" (filtered subset)
Choose columns to include
  ↓
Click "Unduh" (CSV/Excel) or "Cetak / PDF"
  ↓
runExport() orchestrates format-specific generator:
  • CSV: generateCSV() → sanitize + escape → BOM-prefixed UTF-8 → download
  • XLSX: generateXLSX() → lazy-load write-excel-file → Blob → download
  • PDF: buildReportHTML() → open print window → user "Save as PDF"
```

---

## Scope: Visible vs. Custom

**"Sesuai tampilan"** (Visible):
- Exports exactly what user sees on the page
- Respects page-level search filters
- No additional modal filtering
- Quick & matches user expectation

**"Filter khusus"** (Custom):
- Applies modal-defined filters (date range, select dropdowns)
- Ignores page-level search
- Exports all matching rows from `allRows` (unfiltered source)
- For reports & advanced filtering

---

## Security & Best Practices

1. **Always sanitize CSV cells** (CWE-1236 formula injection):
   ```tsx
   // Built into generateCSV (lib/export/csv.ts); no manual escaping needed
   ```

2. **HTML-escape PDF content**:
   ```tsx
   // Built into buildReportHTML; safe for user-supplied data
   ```

3. **Respect row-level security**:
   - Data passed to modal is already filtered by tenant_id (middleware)
   - No additional server-side access control needed in export

4. **Test edge cases**:
   - Empty strings, `null`, `undefined`, very large numbers
   - Special characters (quotes, commas, newlines)
   - Non-ASCII text (Indonesian diacritics)

---

## Checklist for Adding Export to a Module

- [ ] Import `ExportModal` and `ExportButton`
- [ ] Define `ExportColumn[]` array with accurate field mappings
- [ ] Define `ExportFilter[]` (optional) for advanced filtering
- [ ] Create `exporting` state and setter
- [ ] Place `<ExportButton />` in toolbar
- [ ] Place `<ExportModal />` at module root (outside form/drawer)
- [ ] Test typecheck: `npx tsc --noEmit`
- [ ] Test lint: `npx eslint <file>`
- [ ] Manual browser test:
  - Open module
  - Click "Ekspor" button
  - Choose scope, filters, columns
  - Export to CSV, Excel, PDF
  - Verify file content

---

## References

- **Core Lib**: `lib/export/` (CSV, XLSX, PDF generators)
- **Modal Component**: `components/ui/ExportModal.tsx`
- **Architecture**: `docs/adr/013-export-data-modal.md`
- **Tests**: `__tests__/export.test.ts`
- **Implemented Examples**: `/products`, `/wms`, `/wms/inbound`, `/wms/delivery-orders`, `/pos`
- **E2E (Playwright)**: `frontend/e2e/export.spec.ts`
