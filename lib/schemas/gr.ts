import { z } from 'zod'

export const GRStatusSchema = z.enum(['pending', 'accepted', 'rejected'])
export type GRStatus = z.infer<typeof GRStatusSchema>

export const GRSchema = z.object({
  id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  po_id: z.string().uuid(),
  vendor_id: z.string(),
  received_qty: z.number().int(),
  received_amount: z.string(),  // decimal string
  currency: z.string(),
  status: GRStatusSchema,
  received_at: z.string(),
  created_at: z.string(),
})
export type GR = z.infer<typeof GRSchema>

export const CreateGRInputSchema = z.object({
  po_id: z.string().uuid('PO ID harus UUID valid'),
  vendor_id: z.string().min(1, 'Vendor ID diperlukan'),
  received_qty: z.coerce.number().int().min(1, 'Qty minimal 1'),
  received_amount: z
    .string()
    .min(1, 'Jumlah diperlukan')
    .regex(/^\d+(\.\d+)?$/, 'Harus angka positif')
    .refine((v) => parseFloat(v) > 0, 'Harus lebih dari 0'),
  currency: z.string().length(3, 'Kode mata uang 3 karakter').default('IDR'),
})
export type CreateGRInput = z.infer<typeof CreateGRInputSchema>
