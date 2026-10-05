import { extractErrorMessage } from '@/lib/api/errors'

describe('extractErrorMessage (anti "[object Object]")', () => {
  const fb = 'fallback'

  it.each([
    ['structured envelope', { error: { code: 'Not Found', message: 'location not found' } }, 'location not found'],
    ['structured, code only', { error: { code: 'Forbidden' } }, 'Forbidden'],
    ['flat string error (proxy)', { error: 'backend unreachable' }, 'backend unreachable'],
    ['message field', { message: 'pesan' }, 'pesan'],
    ['errors array', { errors: ['a', { message: 'b' }] }, 'a, b'],
    ['errors map', { errors: { sku: 'wajib', name: ['kosong'] } }, 'wajib, kosong'],
    ['plain string payload', 'teks biasa', 'teks biasa'],
  ])('%s', (_label, payload, expected) => {
    expect(extractErrorMessage(payload, fb)).toBe(expected)
  })

  it.each([
    ['null', null],
    ['empty object', {}],
    ['error is array', { error: ['x'] }],
    ['message is object', { message: { nested: true } }],
    ['error.message is object', { error: { message: { a: 1 } } }],
    ['blank strings', { error: '   ', message: '' }],
  ])('falls back and never yields [object Object]: %s', (_label, payload) => {
    const msg = extractErrorMessage(payload, fb)
    expect(msg).toBe(fb)
    expect(msg).not.toContain('[object Object]')
  })

  it('uses a default Indonesian message when no fallback given', () => {
    expect(extractErrorMessage(undefined)).toBe('Terjadi kesalahan pada sistem.')
  })
})
