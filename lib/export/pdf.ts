import type { ExportRequest } from './types'

function esc(value: unknown): string {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

/** Build a standalone, print-ready HTML report (A4 landscape). */
export function buildReportHTML<T>(req: ExportRequest<T>, printedAt = new Date()): string {
  const { title, columns, rows, filterSummary = [] } = req
  const stamp = printedAt.toLocaleString('id-ID', { dateStyle: 'long', timeStyle: 'short' })

  const head = columns
    .map((c) => `<th style="text-align:${c.align ?? 'left'}">${esc(c.header)}</th>`)
    .join('')
  const body = rows.length
    ? rows
        .map(
          (r, i) =>
            `<tr><td class="n">${i + 1}</td>${columns
              .map((c) => {
                const v = c.value(r)
                const shown = typeof v === 'number' ? v.toLocaleString('id-ID') : v
                return `<td style="text-align:${c.align ?? (typeof v === 'number' ? 'right' : 'left')}">${esc(shown)}</td>`
              })
              .join('')}</tr>`
        )
        .join('')
    : `<tr><td colspan="${columns.length + 1}" class="empty">Tidak ada data sesuai filter.</td></tr>`

  const filters = filterSummary.length
    ? `<ul class="filters">${filterSummary.map((f) => `<li>${esc(f)}</li>`).join('')}</ul>`
    : ''

  return `<!doctype html><html lang="id"><head><meta charset="utf-8"><title>${esc(title)}</title>
<style>
@page{size:A4 landscape;margin:12mm}
*{box-sizing:border-box}
body{font-family:Inter,Segoe UI,Arial,sans-serif;color:#0f172a;margin:0;font-size:10px}
header{display:flex;justify-content:space-between;align-items:flex-end;border-bottom:2px solid #2563EB;padding-bottom:8px;margin-bottom:10px}
.brand{font-size:11px;font-weight:700;color:#2563EB;letter-spacing:.04em}
h1{font-size:16px;margin:2px 0 0}
.meta{text-align:right;color:#475569}
.filters{list-style:none;padding:0;margin:0 0 10px;display:flex;flex-wrap:wrap;gap:4px}
.filters li{background:#f1f5f9;border:1px solid #e2e8f0;border-radius:4px;padding:2px 6px}
table{width:100%;border-collapse:collapse}
thead{display:table-header-group}
tr{page-break-inside:avoid}
th{background:#f1f5f9;font-weight:600;border:1px solid #cbd5e1;padding:5px 6px}
td{border:1px solid #e2e8f0;padding:4px 6px;vertical-align:top}
tbody tr:nth-child(even) td{background:#f8fafc}
td.n{color:#64748b;text-align:right;width:28px}
td.empty{text-align:center;color:#64748b;padding:16px}
footer{margin-top:8px;color:#64748b;font-size:9px}
</style></head><body>
<header><div><div class="brand">TAYOOLI ERP</div><h1>${esc(title)}</h1></div>
<div class="meta">Dicetak: ${esc(stamp)}<br>Total: ${rows.length.toLocaleString('id-ID')} baris</div></header>
${filters}
<table><thead><tr><th>#</th>${head}</tr></thead><tbody>${body}</tbody></table>
<footer>Dokumen dibuat otomatis oleh Tayooli ERP.</footer>
<script>window.addEventListener('load',function(){setTimeout(function(){window.focus();window.print()},150)});window.addEventListener('afterprint',function(){window.close()});</script>
</body></html>`
}

/**
 * Open the report in a new tab and trigger the browser print dialog ("Simpan sebagai PDF").
 * Must be called synchronously inside a user click handler to avoid popup blockers.
 * Browser-native print per ADR-012 (no server-side PDF generator).
 */
export function openPrintReport<T>(req: ExportRequest<T>): boolean {
  const win = window.open('', '_blank')
  if (!win) return false
  win.document.open()
  win.document.write(buildReportHTML(req))
  win.document.close()
  return true
}
