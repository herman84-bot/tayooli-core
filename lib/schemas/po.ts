import { z } from 'zod'

export const POStatusSchema = z.enum(['open', 'partially_received', 'received', 'closed', 'cancelled'])
export type POStatus = z.infer<typeof POStatusSchema>

export const POSchema = z.object({
  id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  vendor_id: z.string(),
  po_number: z.string(),
  amount: z.string(),       // decimal string
  qty: z.number().int(),
  currency: z.string(),
  status: POStatusSchema,
  created_at: z.string(),
  updated_at: z.string(),
})
export type PO = z.infer<typeof POSchema>

export const CreatePOInputSchema = z.object({
  vendor_id: z.string().min(1, 'Vendor ID diperlukan'),
  po_number: z.string().min(1, 'Nomor PO diperlukan').max(64),
  amount: z
    .string()
    .min(1, 'Jumlah diperlukan')
    .regex(/^\d+(\.\d+)?$/, 'Harus angka positif')
    .refine((v) => parseFloat(v) > 0, 'Harus lebih dari 0'),
  qty: z.coerce.number().int().min(1, 'Qty minimal 1'),
  currency: z.string().length(3, 'Kode mata uang 3 karakter').default('IDR'),
})
export type CreatePOInput = z.infer<typeof CreatePOInputSchema>
