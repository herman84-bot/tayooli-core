import type { ExportColumn } from './types'

/** Build an .xlsx Blob. The library is loaded lazily so it never enters the main bundle. */
export async function generateXLSX<T>(rows: T[], columns: ExportColumn<T>[], sheet = 'Data'): Promise<Blob> {
  const { default: writeXlsxFile } = await import('write-excel-file')

  const header = columns.map((c) => ({
    value: c.header,
    fontWeight: 'bold' as const,
    backgroundColor: '#E2E8F0',
    align: c.align,
  }))

  const body = rows.map((row) =>
    columns.map((c) => {
      const v = c.value(row)
      if (v === null || v === undefined || v === '') return null
      if (typeof v === 'number') {
        return Number.isFinite(v) ? { type: Number, value: v, align: c.align ?? 'right' } : null
      }
      // XLSX cells are typed as String, so formula injection is not possible here.
      return { type: String, value: String(v), align: c.align }
    })
  )

  return writeXlsxFile([header, ...body] as never, {
    columns: columns.map((c) => ({ width: c.width ?? 18 })),
    sheet: sheet.slice(0, 31),
    stickyRowsCount: 1,
  } as never) as unknown as Promise<Blob>
}
