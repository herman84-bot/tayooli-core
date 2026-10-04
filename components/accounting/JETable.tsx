"use client"

import { ColumnDef } from "@tanstack/react-table"
import { DataTable } from "@/components/ui/data-table"
import type { JournalEntry } from "@/lib/api"

const formatIDR = (value: number) =>
  new Intl.NumberFormat("id-ID", { style: "currency", currency: "IDR", maximumFractionDigits: 0 }).format(value)

const statusConfig: Record<string, { label: string; color: string; bg: string }> = {
  draft:  { label: "Draft",  color: "text-gray-700",   bg: "bg-gray-100"   },
  posted: { label: "Posted", color: "text-green-700",  bg: "bg-green-100"  },
  voided: { label: "Voided", color: "text-red-700",    bg: "bg-red-100"    },
}

const columns: ColumnDef<JournalEntry>[] = [
  {
    accessorKey: "date",
    header: "Date",
    cell: ({ row }) => new Date(row.original.date).toLocaleDateString(),
  },
  {
    accessorKey: "reference_id",
    header: "Reference",
    cell: ({ row }) => (
      <span className="font-mono text-sm">{row.original.reference_id}</span>
    ),
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => (
      <span className="max-w-[300px] truncate block">{row.original.description}</span>
    ),
  },
  {
    accessorKey: "total_debit",
    header: "Total Debit",
    cell: ({ row }) => (
      <span className="tabular-nums">{formatIDR(row.original.total_debit)}</span>
    ),
  },
  {
    accessorKey: "total_credit",
    header: "Total Credit",
    cell: ({ row }) => (
      <span className="tabular-nums">{formatIDR(row.original.total_credit)}</span>
    ),
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => {
      const s = statusConfig[row.original.status] ?? statusConfig.draft
      return (
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${s.bg} ${s.color}`}>
          {s.label}
        </span>
      )
    },
  },
  {
    accessorKey: "created_at",
    header: "Created",
    cell: ({ row }) => new Date(row.original.created_at).toLocaleDateString(),
  },
]

interface JETableProps {
  data: JournalEntry[]
  isLoading?: boolean
  onRowClick?: (entry: JournalEntry) => void
}

export function JETable({ data, isLoading, onRowClick }: JETableProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      isLoading={isLoading}
      onRowClick={onRowClick}
    />
  )
}
