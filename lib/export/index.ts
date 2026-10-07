import { generateCSV } from './csv'
import { generateXLSX } from './excel'
import { openPrintReport } from './pdf'
import type { ExportFormat, ExportRequest } from './types'

export * from './types'
export { generateCSV } from './csv'
export { buildReportHTML } from './pdf'

export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.style.display = 'none'
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/** Safe filename: `produk-2025-03-31.xlsx`. */
export function buildFilename(base: string, ext: string, date = new Date()): string {
  const slug = base.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') || 'export'
  const d = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  return `${slug}-${d}.${ext}`
}

/**
 * Run an export. PDF opens a print window, so the caller must invoke this
 * directly from a click handler (no await before it).
 */
export async function runExport<T>(format: ExportFormat, req: ExportRequest<T>): Promise<void> {
  if (format === 'pdf') {
    if (!openPrintReport(req)) {
      throw new Error('Pop-up diblokir browser. Izinkan pop-up untuk situs ini lalu coba lagi.')
    }
    return
  }
  if (format === 'csv') {
    const csv = generateCSV(req.rows, req.columns)
    downloadBlob(new Blob([csv], { type: 'text/csv;charset=utf-8' }), buildFilename(req.filename, 'csv'))
    return
  }
  const blob = await generateXLSX(req.rows, req.columns, req.title)
  downloadBlob(blob, buildFilename(req.filename, 'xlsx'))
}
