/**
 * Regression test for the "[object Object]" bug reported in WMS Stock
 * Transfer creation (and any other endpoint using handler.RespondError's
 * structured envelope: { error: { code, message, details } }).
 *
 * Root cause: lib/api.ts's request() did `body?.error || body?.message`
 * without checking whether `body.error` was an object. When it was an
 * object, `new Error(objectValue)` coerced it to the literal string
 * "[object Object]" via Object.prototype.toString(), which is what users
 * saw in the transfer creation modal.
 */
import { api } from '@/lib/api'

const originalFetch = global.fetch

afterEach(() => {
  global.fetch = originalFetch
  jest.resetAllMocks()
})

function mockJsonErrorResponse(status: number, body: unknown) {
  global.fetch = jest.fn().mockResolvedValue({
    ok: false,
    status,
    headers: {
      get: (name: string) => (name.toLowerCase() === 'content-type' ? 'application/json' : null),
    },
    json: async () => body,
    text: async () => JSON.stringify(body),
  } as unknown as Response)
}

describe('lib/api.ts request() error parsing', () => {
  it('extracts the human-readable message from a structured { error: { message } } envelope (handler.RespondError shape)', async () => {
    mockJsonErrorResponse(422, {
      error: { code: 'Unprocessable Entity', message: 'insufficient stock at location' },
    })

    await expect(
      api.wms.transfers.create({
        from_warehouse_id: 'wh-1',
        to_warehouse_id: 'wh-2',
        items: [{ product_id: 'p-1', requested_qty: 10 }],
      })
    ).rejects.toThrow('insufficient stock at location')
  })

  it('never throws the literal "[object Object]" string when error is an object', async () => {
    mockJsonErrorResponse(400, {
      error: { code: 'Bad Request', message: 'invalid transfer status transition' },
    })

    try {
      await api.wms.transfers.create({
        from_warehouse_id: 'wh-1',
        to_warehouse_id: 'wh-2',
        items: [{ product_id: 'p-1', requested_qty: 10 }],
      })
      fail('expected request to throw')
    } catch (err) {
      expect(err).toBeInstanceOf(Error)
      expect((err as Error).message).not.toContain('[object Object]')
      expect((err as Error).message).toBe('invalid transfer status transition')
    }
  })

  it('falls back to error.code when error.message is absent', async () => {
    mockJsonErrorResponse(403, {
      error: { code: 'Forbidden' },
    })

    await expect(
      api.wms.transfers.create({
        from_warehouse_id: 'wh-1',
        to_warehouse_id: 'wh-2',
        items: [{ product_id: 'p-1', requested_qty: 10 }],
      })
    ).rejects.toThrow('Forbidden')
  })

  it('still supports the legacy flat { error: "message" } shape', async () => {
    mockJsonErrorResponse(400, { error: 'conflict: duplicate transfer number' })

    await expect(
      api.wms.transfers.create({
        from_warehouse_id: 'wh-1',
        to_warehouse_id: 'wh-2',
        items: [{ product_id: 'p-1', requested_qty: 10 }],
      })
    ).rejects.toThrow('duplicate transfer number')
  })
})
