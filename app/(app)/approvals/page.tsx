"use client"

import { useState } from "react"
import { ColumnDef } from "@tanstack/react-table"
import { DataTable } from "@/components/ui/data-table"
import { Pagination } from "@/components/ui/pagination"
import { Drawer, DrawerProvider, DrawerHeader, DrawerContent } from "@/components/ui/drawer"
import { ApprovalDrawer } from "@/components/approvals/ApprovalDrawer"
import { Icon } from "@/components/ui/icon"
import { useApprovals, useApprovalActions } from "@/hooks/useApprovals"
import type { Approval } from "@/lib/api"

const statusConfig: Record<string, { label: string; color: string; bg: string }> = {
  pending:  { label: "Pending",  color: "text-yellow-700",  bg: "bg-yellow-100" },
  approved: { label: "Approved", color: "text-green-700",   bg: "bg-green-100"  },
  rejected: { label: "Rejected", color: "text-red-700",     bg: "bg-red-100"    },
}

const columns: ColumnDef<Approval>[] = [
  {
    accessorKey: "target_type",
    header: "Type",
    cell: ({ row }) => (
      <span className="font-medium">{row.original.target_type}</span>
    ),
  },
  {
    accessorKey: "target_id",
    header: "Target",
    cell: ({ row }) => (
      <span className="text-muted-foreground font-mono text-xs">
        {row.original.target_id ? `${row.original.target_id.slice(0, 8)}...` : "-"}
      </span>
    ),
  },
  {
    accessorKey: "workflow_id",
    header: "Workflow",
    cell: ({ row }) => (
      <span className="text-muted-foreground font-mono text-xs">
        {row.original.workflow_id ? `${row.original.workflow_id.slice(0, 8)}...` : "-"}
      </span>
    ),
  },
  {
    accessorKey: "requested_by",
    header: "Requested By",
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => {
      const s = statusConfig[row.original.status]
      return (
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${s.bg} ${s.color}`}>
          {s.label}
        </span>
      )
    },
  },
  {
    accessorKey: "created_at",
    header: "Date",
    cell: ({ row }) => new Date(row.original.created_at).toLocaleDateString(),
  },
]

export default function ApprovalsPage() {
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(20)
  const { data: approvalsPage, isLoading } = useApprovals({ page, perPage })
  const approvals = Array.isArray(approvalsPage)
    ? approvalsPage
    : (approvalsPage?.data ?? [])
  const total = Array.isArray(approvalsPage)
    ? approvalsPage.length
    : (approvalsPage?.total ?? 0)
  const totalPages = Math.max(1, Math.ceil(total / perPage))
  const startIndex = total > 0 ? (page - 1) * perPage + 1 : 0
  const endIndex = Math.min(page * perPage, total)
  const { approve, reject } = useApprovalActions()
  const [selected, setSelected] = useState<Approval | null>(null)

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold flex items-center gap-2">
          <Icon name="FileText" size="lg" />
          Approvals
        </h1>
        <span className="text-sm text-muted-foreground">
          {(approvals ?? []).filter((a) => a.status === "pending").length} pending
        </span>
      </div>

      <DataTable
        columns={columns}
        data={approvals ?? []}
        isLoading={isLoading}
        onRowClick={(row) => setSelected(row)}
      />

      {!isLoading && total > 0 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            Menampilkan {startIndex}-{endIndex} dari {total}
          </p>
          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={setPage}
            perPage={perPage}
            onPerPageChange={(pp) => {
              setPerPage(pp)
              setPage(1)
            }}
          />
        </div>
      )}

      <DrawerProvider open={!!selected} onOpenChange={(open) => !open && setSelected(null)}>
        <Drawer>
          <DrawerHeader title="Approval Detail" icon={<Icon name="FileText" />} />
          <DrawerContent>
            {selected && (
              <ApprovalDrawer
                approval={selected}
                onApprove={(id) => {
                  approve.mutate(id)
                  setSelected(null)
                }}
                onReject={({ id, reason }) => {
                  reject.mutate({ id, reason })
                  setSelected(null)
                }}
              />
            )}
          </DrawerContent>
        </Drawer>
      </DrawerProvider>
    </div>
  )
}
