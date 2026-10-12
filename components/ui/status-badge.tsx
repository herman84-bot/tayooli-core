import { cn } from '@/lib/utils'

const defaultColorMap: Record<string, string> = {
  pending: 'bg-amber-50 text-amber-700 border-amber-200',
  approved: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  accepted: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  matched: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  received: 'bg-emerald-50 text-emerald-700 border-emerald-200',
  rejected: 'bg-rose-50 text-rose-600 border-rose-200',
  cancelled: 'bg-rose-50 text-rose-600 border-rose-200',
  pending_review: 'bg-orange-50 text-orange-700 border-orange-200',
  partially_received: 'bg-blue-50 text-blue-700 border-blue-200',
  open: 'bg-blue-50 text-blue-700 border-blue-200',
  ai_processed: 'bg-indigo-50 text-indigo-700 border-indigo-200',
  ai_failed: 'bg-red-50 text-red-700 border-red-200',
}

const defaultLabelMap: Record<string, string> = {
  draft: 'Draf',
  DRAFT: 'Draf',
  pending: 'Menunggu Persetujuan',
  PENDING: 'Menunggu Persetujuan',
  pending_approval: 'Menunggu Persetujuan',
  PENDING_APPROVAL: 'Menunggu Persetujuan',
  approved: 'Disetujui',
  APPROVED: 'Disetujui',
  accepted: 'Diterima',
  ACCEPTED: 'Diterima',
  matched: 'Sesuai',
  MATCHED: 'Sesuai',
  received: 'Diterima',
  RECEIVED: 'Diterima',
  rejected: 'Ditolak',
  REJECTED: 'Ditolak',
  cancelled: 'Dibatalkan',
  CANCELLED: 'Dibatalkan',
  canceled: 'Dibatalkan',
  CANCELED: 'Dibatalkan',
  pending_review: 'Menunggu Peninjauan',
  PENDING_REVIEW: 'Menunggu Peninjauan',
  partially_received: 'Diterima Sebagian',
  PARTIALLY_RECEIVED: 'Diterima Sebagian',
  open: 'Terbuka',
  OPEN: 'Terbuka',
  completed: 'Selesai',
  COMPLETED: 'Selesai',
  done: 'Selesai',
  DONE: 'Selesai',
  in_progress: 'Sedang Berjalan',
  IN_PROGRESS: 'Sedang Berjalan',
  in_transit: 'Dalam Pengiriman',
  IN_TRANSIT: 'Dalam Pengiriman',
  transit: 'Dalam Pengiriman',
  TRANSIT: 'Dalam Pengiriman',
  picking: 'Pengambilan',
  PICKING: 'Pengambilan',
  packing: 'Pengemasan',
  PACKING: 'Pengemasan',
  shipping: 'Pengiriman',
  SHIPPING: 'Pengiriman',
  shipped: 'Dikirim',
  SHIPPED: 'Dikirim',
  putaway: 'Penataan Rak',
  PUTAWAY: 'Penataan Rak',
  ai_processed: 'Diproses AI',
  AI_PROCESSED: 'Diproses AI',
  ai_failed: 'Gagal AI',
  AI_FAILED: 'Gagal AI',
}

export interface StatusBadgeProps {
  status: string
  colorMap?: Record<string, string>
  label?: string
}

export function StatusBadge({ status, colorMap, label }: StatusBadgeProps) {
  const map = colorMap ?? defaultColorMap
  const classes = map[status] ?? 'bg-zinc-50 text-zinc-600 border-zinc-200'
  const displayLabel = label ?? defaultLabelMap[status] ?? status
  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold border',
        classes,
      )}
    >
      {displayLabel}
    </span>
  )
}
