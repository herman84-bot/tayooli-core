import { z } from 'zod'

export const VendorStatusSchema = z.enum(['active', 'inactive', 'suspended'])
export type VendorStatus = z.infer<typeof VendorStatusSchema>

export const VendorSchema = z.object({
  id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  name: z.string(),
  email: z.string().nullable().optional(),
  phone: z.string().nullable().optional(),
  address: z.string().nullable().optional(),
  bank_account: z.string().nullable().optional(),
  bank_name: z.string().nullable().optional(),
  tax_id: z.string().nullable().optional(),
  status: VendorStatusSchema,
  avg_rating: z.string(),
  rating_count: z.number().int().nonnegative(),
  created_at: z.string(),
  updated_at: z.string(),
})
export type Vendor = z.infer<typeof VendorSchema>

export const VendorListSchema = z.object({
  data: z.array(VendorSchema),
  total: z.number().int().nonnegative(),
  page: z.number().int().positive().optional(),
  per_page: z.number().int().positive().optional(),
})
export type VendorList = z.infer<typeof VendorListSchema>

export const CreateVendorInputSchema = z.object({
  name: z.string().min(1, 'Nama vendor wajib diisi').max(255, 'Nama maksimal 255 karakter'),
  email: z.string().email('Format email tidak valid').max(255).optional(),
  phone: z.string().max(50).optional(),
  address: z.string().optional(),
  bank_account: z.string().max(100).optional(),
  bank_name: z.string().max(100).optional(),
  tax_id: z.string().max(50).optional(),
})
export type CreateVendorInput = z.infer<typeof CreateVendorInputSchema>

export const UpdateVendorInputSchema = z.object({
  name: z.string().min(1, 'Nama vendor wajib diisi').max(255, 'Nama maksimal 255 karakter'),
  email: z.string().email('Format email tidak valid').max(255).optional(),
  phone: z.string().max(50).optional(),
  address: z.string().optional(),
  bank_account: z.string().max(100).optional(),
  bank_name: z.string().max(100).optional(),
  tax_id: z.string().max(50).optional(),
})
export type UpdateVendorInput = z.infer<typeof UpdateVendorInputSchema>

export const RateVendorInputSchema = z.object({
  rating: z
    .number()
    .int('Rating harus bilangan bulat')
    .min(1, 'Rating minimal 1')
    .max(5, 'Rating maksimal 5'),
  comment: z.string().max(1000).optional(),
})
export type RateVendorInput = z.infer<typeof RateVendorInputSchema>

export const VendorRatingSchema = z.object({
  id: z.string().uuid(),
  tenant_id: z.string().uuid(),
  vendor_id: z.string().uuid(),
  rated_by: z.string().uuid().nullable().optional(),
  rating: z.number().int(),
  comment: z.string().nullable().optional(),
  created_at: z.string(),
})
export type VendorRating = z.infer<typeof VendorRatingSchema>

export const statusLabels: Record<VendorStatus, string> = {
  active: 'Aktif',
  inactive: 'Nonaktif',
  suspended: 'Ditangguhkan',
}
