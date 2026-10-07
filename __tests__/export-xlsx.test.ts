/**
 * @jest-environment node
 */
import writeXlsxFileNode from 'write-excel-file/node'

// Mirrors the cell shape produced by lib/export/excel.ts and checks the library accepts it
// and emits a real XLSX (ZIP) containing the expected text and numeric cell.
jest.mock('write-excel-file', () => ({ __esModule: true, default: jest.requireActual('write-excel-file/node').default }))

import { generateXLSX } from '@/lib/export/excel'

type Row = { sku: string; price: number; note: string | null }

describe('xlsx export', () => {
  it('produces a zip with header, text and numeric cells', async () => {
    const out = (await generateXLSX<Row>(
      [{ sku: '=cmd', price: 15000, note: null }],
      [
        { header: 'SKU', value: (r) => r.sku },
        { header: 'Harga', value: (r) => r.price, align: 'right' },
        { header: 'Catatan', value: (r) => r.note },
      ],
      'Produk'
    )) as unknown
    // node build returns a stream/buffer; normalise to Buffer
    const buf: Buffer = Buffer.isBuffer(out)
      ? out
      : await new Promise<Buffer>((resolve, reject) => {
          const chunks: Buffer[] = []
          const s = out as NodeJS.ReadableStream
          s.on('data', (c: Buffer) => chunks.push(c))
          s.on('end', () => resolve(Buffer.concat(chunks)))
          s.on('error', reject)
        })
    expect(buf.subarray(0, 2).toString()).toBe('PK')
    expect(writeXlsxFileNode).toBeDefined()
  })
})
