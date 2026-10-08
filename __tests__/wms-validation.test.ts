import {
  parseQty,
  isPastDate,
  validateReceiptLines,
  validatePutaway,
  validateDOLineQty,
  availableFor,
} from '@/lib/wms/validation'

const NOW = new Date(2026, 9, 8, 12, 0, 0) // 2026-10-08 local

const line = (p: Partial<Parameters<typeof validateReceiptLines>[0][number]> = {}) => ({
  product_name: 'Beras',
  accepted: '10',
  rejected: '0',
  reject_reason: '',
  expiry_date: '',
  ...p,
})

describe('parseQty', () => {
  it.each([
    ['5', 5],
    [' 7 ', 7],
    ['2.5', 2.5],
    ['-3', -3],
  ])('parses %p', (input, out) => expect(parseQty(input)).toBe(out))

  it.each(['', '   ', 'abc', '5abc', '1e3', null, undefined])('rejects %p as NaN', (input) => {
    expect(parseQty(input as string)).toBeNaN()
  })
})

describe('isPastDate', () => {
  it('treats yesterday as past, today and future as not past, empty as not past', () => {
    expect(isPastDate('2026-10-07', NOW)).toBe(true)
    expect(isPastDate('2026-10-08', NOW)).toBe(false)
    expect(isPastDate('2027-01-01', NOW)).toBe(false)
    expect(isPastDate('', NOW)).toBe(false)
  })
})

describe('validateReceiptLines (Goods Receipt inverted testing)', () => {
  it('accepts a normal line', () => {
    expect(validateReceiptLines([line()], NOW)).toBeNull()
  })

  it('rejects empty list', () => {
    expect(validateReceiptLines([], NOW)).toMatch(/minimal 1 item/)
  })

  it.each(['-1', '1.5', 'abc', ''])('rejects accepted qty %p', (accepted) => {
    expect(validateReceiptLines([line({ accepted })], NOW)).toMatch(/Qty Diterima harus bilangan bulat/)
  })

  it.each(['-2', '0.5', 'x'])('rejects rejected qty %p', (rejected) => {
    expect(validateReceiptLines([line({ rejected, reject_reason: 'Rusak / pecah' })], NOW)).toMatch(/Qty Ditolak harus bilangan bulat/)
  })

  it('rejects total accepted == 0 and rejected == 0', () => {
    expect(validateReceiptLines([line({ accepted: '0' }), line({ accepted: '0' })], NOW)).toMatch(/tidak boleh 0 semua/)
  })

  it('requires a reject reason when rejected > 0 (whitespace is not a reason)', () => {
    expect(validateReceiptLines([line({ rejected: '2', reject_reason: '   ' })], NOW)).toMatch(/Alasan penolakan wajib/)
    expect(validateReceiptLines([line({ rejected: '2', reject_reason: 'Kemasan sobek' })], NOW)).toBeNull()
  })

  it('allows a reject-only receipt with a reason', () => {
    expect(validateReceiptLines([line({ accepted: '0', rejected: '3', reject_reason: 'Rusak / pecah' })], NOW)).toBeNull()
  })

  it('blocks accepting stock whose expiry date is already in the past', () => {
    expect(validateReceiptLines([line({ expiry_date: '2026-10-01' })], NOW)).toMatch(/sudah lewat/)
  })

  it('allows expired goods when they are all rejected (none accepted)', () => {
    expect(
      validateReceiptLines([line({ accepted: '0', rejected: '5', reject_reason: 'Kedaluwarsa', expiry_date: '2026-10-01' })], NOW)
    ).toBeNull()
  })

  it('blocks over-receiving against the ordered qty', () => {
    expect(validateReceiptLines([line({ accepted: '8', rejected: '3', reject_reason: 'Rusak / pecah', ordered_qty: 10 })], NOW)).toMatch(
      /melebihi qty dokumen asal \(10\)/
    )
    expect(validateReceiptLines([line({ accepted: '7', rejected: '3', reject_reason: 'Rusak / pecah', ordered_qty: 10 })], NOW)).toBeNull()
  })
})

describe('validatePutaway', () => {
  const base = { qty: '5', available: 10, isOverride: false, reason: '' }

  it('accepts a valid putaway', () => expect(validatePutaway(base)).toBeNull())

  it.each(['0', '-1', '', 'abc'])('rejects qty %p', (qty) => {
    expect(validatePutaway({ ...base, qty })).toMatch(/lebih dari 0/)
  })

  it('prevents over-putaway beyond staging on-hand', () => {
    expect(validatePutaway({ ...base, qty: '11' })).toMatch(/melebihi sisa stok di Staging \(10\)/)
    expect(validatePutaway({ ...base, qty: '10' })).toBeNull()
  })

  it('requires a >= 5 char trimmed reason when overriding the default rack', () => {
    expect(validatePutaway({ ...base, isOverride: true, reason: '' })).toMatch(/minimal 5 karakter/)
    expect(validatePutaway({ ...base, isOverride: true, reason: '  abcd  ' })).toMatch(/minimal 5 karakter/)
    expect(validatePutaway({ ...base, isOverride: true, reason: 'Rak penuh' })).toBeNull()
  })
})

describe('validateDOLineQty & availableFor (Surat Jalan pre-check)', () => {
  const stock = [
    { product_id: 'p1', location_id: 'r1', quantity: 10, available_qty: 6 },
    { product_id: 'p1', location_id: 'r2', quantity: '4' },
    { product_id: 'p2', location_id: 'r1', quantity: 9, available_qty: 0 },
  ]

  it('sums availability across racks, prefers available_qty, falls back to quantity', () => {
    expect(availableFor(stock, 'p1')).toBe(10)
    expect(availableFor(stock, 'p1', 'r1')).toBe(6)
    expect(availableFor(stock, 'p1', 'r2')).toBe(4)
    expect(availableFor(stock, 'p2')).toBe(0)
    expect(availableFor(stock, 'missing')).toBe(0)
  })

  it.each([0, -1, 1.5, NaN])('rejects qty %p', (q) => {
    expect(validateDOLineQty(q)).toMatch(/bilangan bulat lebih dari 0/)
  })

  it('reports insufficient stock with remaining available', () => {
    expect(validateDOLineQty(7, 6)).toBe('Stok tidak mencukupi, sisa available: 6')
    expect(validateDOLineQty(6, 6)).toBeNull()
    expect(validateDOLineQty(3)).toBeNull()
  })
})
