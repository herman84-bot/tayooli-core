import type { ExportColumn } from './types'

/**
 * Build an .xlsx Blob synchronously, with no third-party dependency.
 *
 * The previous implementation used `write-excel-file` (JSZip `generateAsync`),
 * which could stall in the browser and trip the 20-second export guard. This
 * writer produces a minimal, valid OOXML workbook packed in an uncompressed
 * (STORE) ZIP, so it finishes in one pass and can never hang on an async chunk.
 */

const XLSX_MIME = 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'

// ── XML helpers ────────────────────────────────────────────────────────────

// Strip control characters that are illegal in XML 1.0, then escape entities.
function xmlEscape(input: string): string {
  // eslint-disable-next-line no-control-regex
  const clean = input.replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F\uFFFE\uFFFF]/g, '')
  return clean
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

function colLetter(index: number): string {
  let n = index + 1
  let s = ''
  while (n > 0) {
    const m = (n - 1) % 26
    s = String.fromCharCode(65 + m) + s
    n = Math.floor((n - 1) / 26)
  }
  return s
}

// Style ids defined in styles.xml below.
const STYLE_DEFAULT = 0
const STYLE_HEADER = 1
const STYLE_RIGHT = 2

type Cell = { kind: 'str'; value: string; style: number } | { kind: 'num'; value: number; style: number } | null

function cellXml(cell: Cell, ref: string): string {
  if (!cell) return ''
  const s = cell.style ? ` s="${cell.style}"` : ''
  if (cell.kind === 'num') return `<c r="${ref}"${s}><v>${cell.value}</v></c>`
  // Inline strings are always typed as text, so formula injection is not possible.
  return `<c r="${ref}"${s} t="inlineStr"><is><t xml:space="preserve">${xmlEscape(cell.value)}</t></is></c>`
}

function toCell<T>(row: T, col: ExportColumn<T>): Cell {
  let v: unknown
  try {
    v = col.value(row)
  } catch {
    // A faulty accessor on one row must not abort the whole export.
    return null
  }
  if (v === null || v === undefined || v === '') return null
  if (typeof v === 'number') {
    if (!Number.isFinite(v)) return null
    return { kind: 'num', value: v, style: col.align === 'left' ? STYLE_DEFAULT : STYLE_RIGHT }
  }
  if (typeof v === 'boolean') return { kind: 'str', value: v ? 'Ya' : 'Tidak', style: STYLE_DEFAULT }
  if (v instanceof Date) {
    return Number.isNaN(v.getTime()) ? null : { kind: 'str', value: v.toISOString(), style: STYLE_DEFAULT }
  }
  return { kind: 'str', value: String(v), style: col.align === 'right' ? STYLE_RIGHT : STYLE_DEFAULT }
}

export function buildSheetXml<T>(rows: T[], columns: ExportColumn<T>[]): string {
  const cols = columns
    .map((c, i) => {
      const w = Number.isFinite(c.width) && (c.width as number) > 0 ? c.width : 18
      return `<col min="${i + 1}" max="${i + 1}" width="${w}" customWidth="1"/>`
    })
    .join('')

  const parts: string[] = []
  const header = columns
    .map((c, i) => cellXml({ kind: 'str', value: c.header ?? '', style: STYLE_HEADER }, `${colLetter(i)}1`))
    .join('')
  parts.push(`<row r="1">${header}</row>`)

  for (let r = 0; r < rows.length; r++) {
    const rowNum = r + 2
    let cells = ''
    for (let c = 0; c < columns.length; c++) {
      cells += cellXml(toCell(rows[r], columns[c]), `${colLetter(c)}${rowNum}`)
    }
    parts.push(`<row r="${rowNum}">${cells}</row>`)
  }

  return (
    '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
    '<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">' +
    '<sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>' +
    (cols ? `<cols>${cols}</cols>` : '') +
    `<sheetData>${parts.join('')}</sheetData>` +
    '</worksheet>'
  )
}

function workbookFiles(sheetName: string, sheetXml: string): Array<[string, string]> {
  const safeName = xmlEscape(sheetName.replace(/[\\/?*[\]:]/g, ' ').slice(0, 31).trim() || 'Data')
  return [
    [
      '[Content_Types].xml',
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">' +
        '<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>' +
        '<Default Extension="xml" ContentType="application/xml"/>' +
        '<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>' +
        '<Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>' +
        '<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>' +
        '</Types>',
    ],
    [
      '_rels/.rels',
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">' +
        '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>' +
        '</Relationships>',
    ],
    [
      'xl/workbook.xml',
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">' +
        `<sheets><sheet name="${safeName}" sheetId="1" r:id="rId1"/></sheets>` +
        '</workbook>',
    ],
    [
      'xl/_rels/workbook.xml.rels',
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">' +
        '<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>' +
        '<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>' +
        '</Relationships>',
    ],
    [
      'xl/styles.xml',
      '<?xml version="1.0" encoding="UTF-8" standalone="yes"?>' +
        '<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">' +
        '<fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts>' +
        '<fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill>' +
        '<fill><patternFill patternType="solid"><fgColor rgb="FFE2E8F0"/><bgColor indexed="64"/></patternFill></fill></fills>' +
        '<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>' +
        '<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>' +
        '<cellXfs count="3">' +
        '<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>' +
        '<xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"/>' +
        '<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="1"><alignment horizontal="right"/></xf>' +
        '</cellXfs>' +
        '</styleSheet>',
    ],
    ['xl/worksheets/sheet1.xml', sheetXml],
  ]
}

// ── Minimal ZIP (STORE, no compression) ─────────────────────────────────────

let CRC_TABLE: Uint32Array | null = null
function crc32(data: Uint8Array): number {
  if (!CRC_TABLE) {
    CRC_TABLE = new Uint32Array(256)
    for (let n = 0; n < 256; n++) {
      let c = n
      for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
      CRC_TABLE[n] = c >>> 0
    }
  }
  let crc = 0xffffffff
  for (let i = 0; i < data.length; i++) crc = CRC_TABLE[(crc ^ data[i]) & 0xff] ^ (crc >>> 8)
  return (crc ^ 0xffffffff) >>> 0
}

export function buildZip(files: Array<[string, string]>): Uint8Array {
  const enc = new TextEncoder()
  const entries = files.map(([name, content]) => {
    const nameBytes = enc.encode(name)
    const data = enc.encode(content)
    return { nameBytes, data, crc: crc32(data) }
  })

  let localSize = 0
  for (const e of entries) localSize += 30 + e.nameBytes.length + e.data.length
  let centralSize = 0
  for (const e of entries) centralSize += 46 + e.nameBytes.length
  const out = new Uint8Array(localSize + centralSize + 22)
  const view = new DataView(out.buffer)

  let p = 0
  const offsets: number[] = []
  for (const e of entries) {
    offsets.push(p)
    view.setUint32(p, 0x04034b50, true)
    view.setUint16(p + 4, 20, true) // version needed
    view.setUint16(p + 6, 0x0800, true) // UTF-8 names
    view.setUint16(p + 8, 0, true) // STORE
    view.setUint16(p + 10, 0, true)
    view.setUint16(p + 12, 0x21, true) // 1980-01-01
    view.setUint32(p + 14, e.crc, true)
    view.setUint32(p + 18, e.data.length, true)
    view.setUint32(p + 22, e.data.length, true)
    view.setUint16(p + 26, e.nameBytes.length, true)
    view.setUint16(p + 28, 0, true)
    out.set(e.nameBytes, p + 30)
    out.set(e.data, p + 30 + e.nameBytes.length)
    p += 30 + e.nameBytes.length + e.data.length
  }

  const centralStart = p
  entries.forEach((e, i) => {
    view.setUint32(p, 0x02014b50, true)
    view.setUint16(p + 4, 20, true)
    view.setUint16(p + 6, 20, true)
    view.setUint16(p + 8, 0x0800, true)
    view.setUint16(p + 10, 0, true)
    view.setUint16(p + 12, 0, true)
    view.setUint16(p + 14, 0x21, true)
    view.setUint32(p + 16, e.crc, true)
    view.setUint32(p + 20, e.data.length, true)
    view.setUint32(p + 24, e.data.length, true)
    view.setUint16(p + 28, e.nameBytes.length, true)
    view.setUint16(p + 30, 0, true)
    view.setUint16(p + 32, 0, true)
    view.setUint16(p + 34, 0, true)
    view.setUint16(p + 36, 0, true)
    view.setUint32(p + 38, 0, true)
    view.setUint32(p + 42, offsets[i], true)
    out.set(e.nameBytes, p + 46)
    p += 46 + e.nameBytes.length
  })

  view.setUint32(p, 0x06054b50, true)
  view.setUint16(p + 4, 0, true)
  view.setUint16(p + 6, 0, true)
  view.setUint16(p + 8, entries.length, true)
  view.setUint16(p + 10, entries.length, true)
  view.setUint32(p + 12, p - centralStart, true)
  view.setUint32(p + 16, centralStart, true)
  view.setUint16(p + 20, 0, true)
  return out
}

export function buildXLSXBytes<T>(rows: T[], columns: ExportColumn<T>[], sheet = 'Data'): Uint8Array {
  return buildZip(workbookFiles(sheet, buildSheetXml(rows ?? [], columns ?? [])))
}

export async function generateXLSX<T>(rows: T[], columns: ExportColumn<T>[], sheet = 'Data'): Promise<Blob> {
  const bytes = buildXLSXBytes(rows, columns, sheet)
  return new Blob([bytes as BlobPart], { type: XLSX_MIME })
}
