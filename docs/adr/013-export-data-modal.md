# ADR-013: Reusable Export Data Modal (CSV, Excel, PDF)

**Status**: Decided  

**Context**: Tayooli Core must support exporting tabular data in three formats (CSV, Excel, PDF) with client-side filtering, respecting existing browser-print architecture (ADR-012).

---

## Decision

Implemented a unified export engine with:

1. **Zero-Dependency CSV** (`lib/export/csv.ts`):
   - UTF-8 BOM for Excel Windows compatibility
   - RFC-4180 escaping (quotes, commas, newlines)
   - CWE-1236 formula injection sanitization
   - ~200 bytes minified

2. **Lightweight Excel** (`lib/export/excel.ts`):
   - `write-excel-file` (~45 KB gzipped, zero polyfills)
   - Typed columns, auto-width, header styling
   - Async lazy-loaded (not in main bundle)

3. **Browser Print PDF** (`lib/export/pdf.ts`):
   - No server-side generator (adheres to ADR-012)
   - Scoped HTML template (A4 Landscape, filter summary, row count)
   - Opens print dialog via `window.open()`, user saves as PDF
   - 0 KB server impact

4. **Reusable Export Modal** (`components/ui/ExportModal.tsx`):
   - Date-range filters (RFC-3339)
   - Select filters (custom predicates)
   - Column toggle (omit unnecessary fields)
   - Scope radio: "Visible rows" (page-filtered) or "Custom" (modal-filtered)
   - Accessibility: ARIA roles, focus trap (Escape key), keyboard nav

---

## Implementation Details

### Data Flow

```
Page (e.g., /products)
  ├─ allRows (database records, unfiltered)
  ├─ visibleRows (after page search/filters)
  └─ <ExportModal
      columns={productExportColumns}
      filters={productExportFilters}
      allRows={products}
      visibleRows={filteredProducts}
      visibleSummary={["Pencarian: …"]}
    />
```

**Scope Logic**:
- **"Visible"** (default): exports `visibleRows` as-is (what user sees on page)
- **"Custom"**: applies modal's date & select filters to `allRows`, generates filtered subset

### File Structure

```
lib/export/
  types.ts          – ExportFormat, ExportColumn, ExportRequest, ExportFilter
  csv.ts            – generateCSV, sanitizeCell (CWE-1236), escapeCSVCell
  excel.ts          – generateXLSX (lazy-loaded write-excel-file)
  pdf.ts            – buildReportHTML, openPrintReport (no server PDF)
  index.ts          – runExport orchestrator, downloadBlob, buildFilename

components/ui/
  ExportModal.tsx   – <ExportModal> (state) → <ExportDialog> (UI), <ExportButton>
```

### Example Usage (Products)

```tsx
const [exporting, setExporting] = useState(false)

const productExportColumns: ExportColumn<Product>[] = [
  { header: "SKU", value: (p) => p.sku },
  { header: "Harga (Rp)", value: (p) => p.price, align: "right" },
  // ...
]

const productExportFilters: ExportFilter<Product>[] = [
  { type: "dateRange", id: "created", label: "Tanggal dibuat", getDate: (p) => p.created_at },
  {
    type: "select",
    id: "priceRange",
    label: "Rentang harga",
    options: [{ value: "lt50", label: "< Rp50.000" }, …],
    match: (p, v) => /* custom predicate */,
  },
]

<ExportModal
  open={exporting}
  onClose={() => setExporting(false)}
  title="Katalog Produk"
  filename="katalog-produk"
  columns={productExportColumns}
  allRows={products}
  visibleRows={filteredProducts}
  visibleSummary={[…]}
  filters={productExportFilters}
/>
```

---

## Trade-offs

| Choice | Why | Alt | Trade-off |
|--------|-----|-----|-----------|
| **Client-side CSV/XLSX** | Instant (no server latency), zero backend load | Server generation | Max ~10K rows limited by browser memory; paginated export not yet supported |
| **write-excel-file** | 45 KB, zero polyfills, React 19 safe | xlsx/exceljs | Less feature-rich (no charts, images, multiple sheets in single call) |
| **Browser print PDF** | Adheres to ADR-012, native, user controls quality | Puppeteer/wkhtmltopdf | Limited to ~A4 page size; user must click "Save as PDF" (not fully automatic) |
| **Modal-first UX** | Prevents accidental exports; gives user control over scope & filters | Export button → direct CSV | One more click; more intent required |

---

## Security & Compliance

- **CWE-1236** (Spreadsheet Formula Injection): All cell values sanitized before CSV export
- **XSS in PDF**: HTML report escapes all user data (`&lt;`, `&quot;`, `&amp;`)
- **No data sent to server**: All export happens in-browser
- **Row-level filtering respects tenant_id**: Export modal uses data already loaded & filtered by tenant middleware

---

## Future Extensions

1. **Paginated export** (>10K rows): Fetch in batches, stream to file
2. **Backend-driven CSV** (for large result sets): `/api/v1/export?resource=products&format=csv&filters=…`
3. **Excel charts & pivot tables**: Upgrade `write-excel-file` or add lightweight alternative
4. **Multi-sheet XLSX**: Re-export same template with different `sheet` name
5. **Advanced filtering**: Date picker, multi-select, full-text search in modal

---

## Verification

- **Unit tests** (`__tests__/export.test.ts`): CSV escaping, formula injection, HTML XSS, filename slug
- **Integration**: Wired into `/products` page (typecheck + lint pass). Manual browser test against a live backend still pending.
- **Accessibility**: ARIA dialog/radio roles, Escape to close, visible focus rings (no full focus trap yet)
- **Bundle**: write-excel-file is loaded via dynamic `import()` only when Excel is chosen

---

## References

- ADR-012: WMS Official Document Generation & Print Engine (browser print, no server PDF)
- RFC-4180: Common Format and MIME Type for Comma-Separated Values (CSV)
- [write-excel-file](https://github.com/catamphetamine/write-excel-file) – Docs & API
