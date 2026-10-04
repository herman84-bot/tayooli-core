"use client"

import { ColumnDef } from "@tanstack/react-table"
import { DataTable } from "@/components/ui/data-table"
import { formatCurrency } from "@/lib/currency"
import type { Account } from "@/lib/api"

const typeColors: Record<string, { bg: string; text: string }> = {
  Asset:     { bg: "bg-blue-100",   text: "text-blue-800"   },
  Liability: { bg: "bg-red-100",    text: "text-red-800"    },
  Equity:    { bg: "bg-purple-100", text: "text-purple-800" },
  Revenue:   { bg: "bg-green-100",  text: "text-green-800"  },
  Expense:   { bg: "bg-orange-100", text: "text-orange-800" },
}

const columns: ColumnDef<Account>[] = [
  {
    accessorKey: "code",
    header: "Code",
    cell: ({ row }) => (
      <span className="font-mono text-sm font-medium">{row.original.code}</span>
    ),
  },
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => (
      <span className="font-medium">{row.original.name}</span>
    ),
  },
  {
    accessorKey: "type",
    header: "Type",
    cell: ({ row }) => {
      const t = row.original.type
      const c = typeColors[t] ?? { bg: "bg-gray-100", text: "text-gray-800" }
      return (
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${c.bg} ${c.text}`}>
          {t}
        </span>
      )
    },
  },
  {
    accessorKey: "balance",
    header: "Balance",
    cell: ({ row }) => (
      <span className="tabular-nums">
        {formatCurrency(row.original.balance)}
      </span>
    ),
  },
  {
    accessorKey: "is_active",
    header: "Status",
    cell: ({ row }) => (
      <span className={`text-xs ${row.original.is_active ? "text-green-600" : "text-muted-foreground"}`}>
        {row.original.is_active ? "Active" : "Inactive"}
      </span>
    ),
  },
  {
    accessorKey: "created_at",
    header: "Created",
    cell: ({ row }) => new Date(row.original.created_at).toLocaleDateString(),
  },
]

interface CoATableProps {
  data: Account[]
  isLoading?: boolean
  onRowClick?: (account: Account) => void
}

export function CoATable({ data, isLoading, onRowClick }: CoATableProps) {
  return (
    <DataTable
      columns={columns}
      data={data}
      isLoading={isLoading}
      onRowClick={onRowClick}
    />
  )
}
