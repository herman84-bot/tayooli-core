export type ExportFormat = 'xlsx' | 'csv' | 'pdf'

export type ExportCell = string | number | null | undefined

/** One column definition shared by all three export formats. */
export interface ExportColumn<T> {
  header: string
  value: (row: T) => ExportCell
  /** Width in characters (Excel) – also used as a relative hint for PDF. */
  width?: number
  align?: 'left' | 'right' | 'center'
}

export interface ExportRequest<T> {
  title: string
  /** Filename without extension. */
  filename: string
  columns: ExportColumn<T>[]
  rows: T[]
  /** Human-readable filter summary, printed in the PDF header and the CSV/XLSX metadata. */
  filterSummary?: string[]
}
