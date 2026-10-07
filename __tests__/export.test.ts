import { generateCSV, buildReportHTML, buildFilename, type ExportColumn } from '@/lib/export'
import { sanitizeCell, escapeCSVCell } from '@/lib/export/csv'

type Row = { sku: string; name: string; price: number }
const cols: ExportColumn<Row>[] = [
  { header: 'SKU', value: (r) => r.sku },
  { header: 'Nama', value: (r) => r.name },
  { header: 'Harga', value: (r) => r.price },
]

describe('csv export', () => {
  it('adds UTF-8 BOM and header row', () => {
    const csv = generateCSV([{ sku: 'A1', name: 'Kopi', price: 15000 }], cols)
    expect(csv.charCodeAt(0)).toBe(0xfeff)
    expect(csv.slice(1).split('\r\n')).toEqual(['SKU,Nama,Harga', 'A1,Kopi,15000'])
  })

  it('escapes commas, quotes and newlines (RFC-4180)', () => {
    expect(escapeCSVCell('a,b')).toBe('"a,b"')
    expect(escapeCSVCell('say "hi"')).toBe('"say ""hi"""')
    expect(escapeCSVCell('x\ny')).toBe('"x\ny"')
  })

  it('neutralises formula injection but keeps numbers', () => {
    expect(sanitizeCell('=HYPERLINK("x")')).toBe(`'=HYPERLINK("x")`)
    expect(sanitizeCell('@SUM(A1)')).toBe("'@SUM(A1)")
    expect(sanitizeCell(-5)).toBe('-5')
    expect(sanitizeCell(null)).toBe('')
  })
})

describe('pdf report html', () => {
  it('escapes HTML and shows filter summary + row count', () => {
    const html = buildReportHTML({
      title: 'Produk',
      filename: 'produk',
      columns: cols,
      rows: [{ sku: '<b>', name: 'A & B', price: 1000 }],
      filterSummary: ['Harga: < Rp50.000'],
    })
    expect(html).toContain('&lt;b&gt;')
    expect(html).toContain('A &amp; B')
    expect(html).toContain('Harga: &lt; Rp50.000')
    expect(html).toContain('Total: 1 baris')
    expect(html).not.toContain('<b></td>')
  })

  it('renders an empty-state row', () => {
    const html = buildReportHTML({ title: 'X', filename: 'x', columns: cols, rows: [] })
    expect(html).toContain('Tidak ada data sesuai filter.')
  })
})

describe('filename', () => {
  it('slugifies and dates the filename', () => {
    expect(buildFilename('Katalog Produk!', 'xlsx', new Date(2025, 2, 5))).toBe('katalog-produk-2025-03-05.xlsx')
  })
})
