import type { ExportCell, ExportColumn } from './types'

const FORMULA_TRIGGERS = ['=', '+', '-', '@', '\t', '\r']

/** Neutralise spreadsheet formula injection (CWE-1236). Numbers are left untouched. */
export function sanitizeCell(value: ExportCell): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'number') return Number.isFinite(value) ? String(value) : ''
  const s = String(value)
  return FORMULA_TRIGGERS.includes(s.charAt(0)) ? `'${s}` : s
}

/** RFC-4180 quoting. */
export function escapeCSVCell(value: string): string {
  return /[",\r\n;]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value
}

/** Build CSV text with UTF-8 BOM so Excel on Windows reads it correctly. */
export function generateCSV<T>(rows: T[], columns: ExportColumn<T>[]): string {
  const lines = [
    columns.map((c) => escapeCSVCell(c.header)).join(','),
    ...rows.map((row) => columns.map((c) => escapeCSVCell(sanitizeCell(c.value(row)))).join(',')),
  ]
  return '\uFEFF' + lines.join('\r\n')
}
