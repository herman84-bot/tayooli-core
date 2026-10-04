import { z } from 'zod'

export const InvoiceStatusSchema = z.enum(['pending', 'ai_processed', 'ai_failed', 'pending_review', 'approved', 'rejected'])
export type InvoiceStatus = z.infer<typeof InvoiceStatusSchema>

export const InvoiceSchema = z.object({
  id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  vendor_id: z.string(),
  invoice_number: z.string().min(1),
  // Backend returns NUMERIC(20,4) serialised as a decimal string (e.g. "1234.5600").
  // Never parse as z.number() — precision would be silently lost via float64.
  amount: z.string(),
  currency: z.string().default('IDR'),
  status: InvoiceStatusSchema,
  due_date: z.string().nullable().optional(),
  ai_confidence_score: z.number().min(0).max(1).nullable().optional(),
  match_result: z.string().nullable().optional(),
  po_id: z.string().uuid().nullable().optional(),
  created_at: z.string(),
  updated_at: z.string(),
})
export type Invoice = z.infer<typeof InvoiceSchema>

export const InvoiceListSchema = z.object({
  data: z.array(InvoiceSchema),
  total: z.number().int().nonnegative(),
  page: z.number().int().positive().optional(),
  per_page: z.number().int().positive().optional(),
})
export type InvoiceList = z.infer<typeof InvoiceListSchema>

export const CreateInvoiceInputSchema = z.object({
  vendor_id: z.string().min(1, 'Vendor ID diperlukan'),
  invoice_number: z.string().min(1, 'Nomor invoice diperlukan').max(64),
  // Amount is sent as a string to preserve full decimal precision end-to-end.
  // Validated as a positive decimal; the backend further enforces NUMERIC(20,4) bounds.
  amount: z
    .string()
    .min(1, 'Jumlah diperlukan')
    .regex(/^\d+(\.\d+)?$/, 'Jumlah harus berupa angka positif')
    .refine((v) => parseFloat(v) > 0, 'Jumlah harus lebih dari 0'),
  currency: z.string().length(3, 'Kode mata uang harus 3 karakter').default('IDR'),
  due_date: z.string().min(1, 'Tanggal jatuh tempo diperlukan').optional(),
})
export type CreateInvoiceInput = z.infer<typeof CreateInvoiceInputSchema>
