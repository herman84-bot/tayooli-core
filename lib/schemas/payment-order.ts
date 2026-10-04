import { z } from 'zod'

export const PaymentOrderStatusSchema = z.enum([
  'draft',
  'pending_approval',
  'approved',
  'paid',
  'rejected',
  'cancelled',
])
export type PaymentOrderStatus = z.infer<typeof PaymentOrderStatusSchema>

export const PaymentOrderSchema = z.object({
  id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  invoice_id: z.string().uuid(),
  // Backend returns NUMERIC(15,2) as a decimal string.
  amount: z.string(),
  currency: z.string().default('IDR'),
  payment_method: z.string(),
  reference_number: z.string().nullable().optional(),
  status: PaymentOrderStatusSchema,
  notes: z.string().nullable().optional(),
  created_by: z.string().uuid().nullable().optional(),
  approved_by: z.string().uuid().nullable().optional(),
  paid_at: z.string().nullable().optional(),
  created_at: z.string(),
  updated_at: z.string(),
})
export type PaymentOrder = z.infer<typeof PaymentOrderSchema>

export const PaymentOrderListSchema = z.object({
  data: z.array(PaymentOrderSchema),
  total: z.number().int().nonnegative(),
  page: z.number().int().positive().optional(),
  per_page: z.number().int().positive().optional(),
})
export type PaymentOrderList = z.infer<typeof PaymentOrderListSchema>

export const CreatePaymentOrderInputSchema = z.object({
  invoice_id: z.string().uuid('Invoice ID tidak valid'),
  amount: z
    .string()
    .min(1, 'Jumlah diperlukan')
    .regex(/^\d+(\.\d+)?$/, 'Jumlah harus berupa angka positif')
    .refine((v) => parseFloat(v) > 0, 'Jumlah harus lebih dari 0'),
  currency: z.string().length(3, 'Kode mata uang harus 3 karakter').default('IDR'),
  payment_method: z
    .enum(['bank_transfer', 'virtual_account', 'credit_card', 'e_wallet', 'check', 'cash'])
    .default('bank_transfer'),
  reference_number: z.string().max(100).optional(),
  notes: z.string().optional(),
})
export type CreatePaymentOrderInput = z.infer<typeof CreatePaymentOrderInputSchema>

export const paymentMethodLabels: Record<string, string> = {
  bank_transfer: 'Transfer Bank',
  virtual_account: 'Virtual Account',
  credit_card: 'Kartu Kredit',
  e_wallet: 'E-Wallet',
  check: 'Cek',
  cash: 'Tunai',
}
