import { createQueryClient } from '@/lib/query-client'

// Regression: mutations (POST/PUT/DELETE) must never auto-retry. A retry after a
// proxy timeout re-sends a write the backend may already have committed,
// creating duplicate documents / double stock movements.
// Ref: TanStack Query default is "will not retry a mutation on error".
describe('createQueryClient defaults', () => {
  it('does not retry mutations', () => {
    const qc = createQueryClient()
    expect(qc.getDefaultOptions().mutations?.retry).toBe(0)
  })

  it('still retries queries (reads are safe to repeat)', () => {
    const qc = createQueryClient()
    expect(qc.getDefaultOptions().queries?.retry).toBe(2)
  })
})
