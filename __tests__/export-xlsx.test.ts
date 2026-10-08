/**
 * @jest-environment node
 */
import JSZip from 'jszip'
import { buildXLSXBytes, generateXLSX } from '@/lib/export/excel'
import type { ExportColumn } from '@/lib/export/types'

type Row = { sku: string; price: number | null; note: string | null | undefined; name: string }

const columns: ExportColumn<Row>[] = [
  { header: 'SKU', value: (r) => r.sku },
  { header: 'Harga', value: (r) => r.price, align: 'right' },
  { header: 'Catatan', value: (r) => r.note },
  { header: 'Nama', value: (r) => r.name },
]

async function sheetXml(bytes: Uint8Array): Promise<string> {
  const zip = await JSZip.loadAsync(bytes)
  const f = zip.file('xl/worksheets/sheet1.xml')
  if (!f) throw new Error('sheet1.xml missing')
  return f.async('string')
}

describe('xlsx export (dependency-free writer)', () => {
  it('produces a valid zip with all OOXML parts', async () => {
    const bytes = buildXLSXBytes<Row>([{ sku: 'A-1', price: 15000, note: null, name: 'Beras' }], columns, 'Produk')
    expect(String.fromCharCode(bytes[0], bytes[1])).toBe('PK')
    const zip = await JSZip.loadAsync(bytes)
    for (const part of ['[Content_Types].xml', '_rels/.rels', 'xl/workbook.xml', 'xl/_rels/workbook.xml.rels', 'xl/styles.xml', 'xl/worksheets/sheet1.xml']) {
      expect(zip.file(part)).not.toBeNull()
    }
    expect(await zip.file('xl/workbook.xml')!.async('string')).toContain('name="Produk"')
  })

  it('writes numbers as numeric cells and text as inline strings (no formula injection)', async () => {
    const xml = await sheetXml(buildXLSXBytes<Row>([{ sku: '=cmd|calc', price: 15000, note: 'ok', name: 'X' }], columns))
    expect(xml).toContain('<c r="B2" s="2"><v>15000</v></c>')
    expect(xml).toContain('t="inlineStr"><is><t xml:space="preserve">=cmd|calc</t>')
    expect(xml).not.toContain('<f>')
  })

  it('skips null, undefined, NaN, Infinity and escapes XML special chars', async () => {
    const xml = await sheetXml(
      buildXLSXBytes<Row>(
        [
          { sku: 'S1', price: null, note: undefined, name: 'Tom & <Jerry> "x"' },
          { sku: 'S2', price: Number.NaN, note: null, name: 'bad\u0001char' },
          { sku: 'S3', price: Number.POSITIVE_INFINITY, note: '', name: 'ok' },
        ],
        columns
      )
    )
    expect(xml).not.toContain('r="B2"')
    expect(xml).not.toContain('r="C2"')
    expect(xml).not.toContain('r="B3"')
    expect(xml).not.toContain('r="B4"')
    expect(xml).toContain('Tom &amp; &lt;Jerry&gt; &quot;x&quot;')
    expect(xml).toContain('badchar')
  })

  it('does not abort when a column accessor throws', async () => {
    const cols: ExportColumn<Row>[] = [...columns, { header: 'Boom', value: () => { throw new Error('x') } }]
    const xml = await sheetXml(buildXLSXBytes<Row>([{ sku: 'S', price: 1, note: null, name: 'n' }], cols))
    expect(xml).toContain('>Boom<')
    expect(xml).not.toContain('r="E2"')
  })

  it('handles empty rows and 5000 rows quickly', async () => {
    expect((await sheetXml(buildXLSXBytes<Row>([], columns))).match(/<row /g)).toHaveLength(1)
    const many: Row[] = Array.from({ length: 5000 }, (_, i) => ({ sku: `SKU-${i}`, price: i, note: null, name: `Produk ${i}` }))
    const t0 = Date.now()
    const blob = await generateXLSX(many, columns, 'Produk')
    expect(Date.now() - t0).toBeLessThan(3000)
    expect(blob.type).toBe('application/vnd.openxmlformats-officedocument.spreadsheetml.sheet')
    expect(blob.size).toBeGreaterThan(1000)
  })
})
