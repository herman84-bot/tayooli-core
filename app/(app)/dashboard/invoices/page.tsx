// LEGACY: This page uses deprecated types. Use app/(shell)/dashboard/invoices instead.
"use client"

import { useState } from "react"
import Link from "next/link"
import { ColumnDef } from "@tanstack/react-table"
import { DataTable } from "@/components/ui/data-table"
import { Drawer, DrawerProvider, DrawerHeader, DrawerContent } from "@/components/ui/drawer"
import { Icon } from "@/components/ui/icon"
import { AnomalyBadge } from "@/components/ui/anomaly-badge"
import { Pagination } from "@/components/ui/pagination"
import { useInvoices } from "@/hooks/useInvoices"
import { useInferenceStatus, useTriggerInference } from "@/hooks/useInference"
import { formatCurrency } from "@/lib/currency"
import type { Invoice } from "@/lib/api"

const statusConfig: Record<string, { label: string; color: string; bg: string }> = {
  pending:        { label: "Pending",        color: "text-yellow-700", bg: "bg-yellow-100" },
  processing:     { label: "Processing",     color: "text-blue-700",   bg: "bg-blue-100"   },
  pending_review: { label: "Review AI",      color: "text-blue-700",   bg: "bg-blue-100"   },
  ai_processed:   { label: "AI Processed",   color: "text-blue-700",   bg: "bg-blue-100"   },
  ai_failed:      { label: "AI Failed",      color: "text-red-700",    bg: "bg-red-100"    },
  approved:       { label: "Approved",       color: "text-green-700",  bg: "bg-green-100"  },
  rejected:       { label: "Rejected",       color: "text-red-700",    bg: "bg-red-100"    },
}

const columns: ColumnDef<Invoice>[] = [
  {
    accessorKey: "invoice_number",
    header: "Invoice #",
    cell: ({ row }) => (
      <span className="font-medium">{row.original.invoice_number}</span>
    ),
  },
  {
    accessorKey: "vendor_name",
    header: "Vendor",
  },
  {
    accessorKey: "amount",
    header: "Amount",
    cell: ({ row }) =>
      formatCurrency(row.original.amount),
  },
  {
    accessorKey: "status",
    header: "Status",
    cell: ({ row }) => {
      const s = statusConfig[row.original.status] ?? { label: row.original.status, color: "text-zinc-700", bg: "bg-zinc-100" }
      return (
        <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${s.bg} ${s.color}`}>
          {s.label}
        </span>
      )
    },
  },
  {
    accessorKey: "anomaly_score",
    header: "AI Analysis",
    cell: ({ row }) => (
      <AnomalyBadge
        score={row.original.anomaly_score}
        detected={row.original.anomaly_detected}
      />
    ),
  },
  {
    accessorKey: "due_date",
    header: "Due Date",
    cell: ({ row }) => new Date(row.original.due_date).toLocaleDateString(),
  },
  {
    accessorKey: "created_at",
    header: "Created",
    cell: ({ row }) => new Date(row.original.created_at).toLocaleDateString(),
  },
]

export default function InvoicesPage() {
  const [page, setPage] = useState(1)
  const [perPage, setPerPage] = useState(20)
  const { data: invoiceList, isLoading } = useInvoices({ page, perPage })
  const invoices = invoiceList?.data ?? []
  const total = invoiceList?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / perPage))
  const startIndex = total > 0 ? (page - 1) * perPage + 1 : 0
  const endIndex = Math.min(page * perPage, total)
  const [selected, setSelected] = useState<Invoice | null>(null)
  const [ingestedId, setIngestedId] = useState<string | null>(null)
  const runMutation = useTriggerInference()
  const inferred = useInferenceStatus(ingestedId)

  return (
    <div className="p-6 space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Icon name="FileText" size="lg" />
            Purchase Invoices (Vendor Bills)
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Daftar tagihan pembelian dan faktur masuk dari vendor/pemasok.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">
            {total} total
          </span>
          <Link
            href="/dashboard/invoices/new"
            className="bg-indigo-600 hover:bg-indigo-700 text-white text-sm font-semibold rounded-xl px-4 py-2 transition-colors flex items-center gap-1.5 shadow-sm"
          >
            + Buat Purchase Invoice Baru
          </Link>
        </div>
      </div>

      <DataTable
        columns={columns}
        data={invoices}
        isLoading={isLoading}
        onRowClick={(row) => setSelected(row)}
      />

      {!isLoading && total > 0 && (
        <div className="flex items-center justify-between">
          <p className="text-xs text-zinc-500">
            Menampilkan {startIndex}-{endIndex} dari {total}
          </p>
          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={setPage}
            perPage={perPage}
            onPerPageChange={(v) => {
              setPerPage(v)
              setPage(1)
            }}
          />
        </div>
      )}

      <DrawerProvider
        open={!!selected}
        onOpenChange={(open) => {
          if (!open) {
            setSelected(null)
            setIngestedId(null)
          }
        }}
      >
        <Drawer>
          <DrawerHeader title="Purchase Invoice Detail" icon={<Icon name="FileText" />} />
          <DrawerContent>
            {selected && (
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Purchase Invoice #</p>
                    <p className="font-medium">{selected.invoice_number}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Vendor</p>
                    <p className="font-medium">{selected.vendor_name}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Amount</p>
                    <p className="font-medium">
                      {formatCurrency(selected.amount)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Status</p>
                    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${(statusConfig[selected.status] ?? { label: selected.status, color: "text-zinc-700", bg: "bg-zinc-100" }).bg} ${(statusConfig[selected.status] ?? { label: selected.status, color: "text-zinc-700", bg: "bg-zinc-100" }).color}`}>
                      {(statusConfig[selected.status] ?? { label: selected.status }).label}
                    </span>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">AI Analysis</p>
                    <AnomalyBadge
                      score={selected.anomaly_score}
                      detected={selected.anomaly_detected}
                    />
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Due Date</p>
                    <p className="font-medium">{new Date(selected.due_date).toLocaleDateString()}</p>
                  </div>
                </div>

                <div className="border rounded-lg p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-medium flex items-center gap-2">
                      <Icon name="Sparkles" size="sm" />
                      Live AI analysis
                    </p>
                    <button
                      onClick={() =>
                        runMutation.mutate(
                          {
                            invoice_id: selected.id,
                            amount: String(selected.amount),
                            vendor_id: selected.vendor_id ?? undefined,
                          },
                          {
                            onSuccess: (res) => setIngestedId(res.job_id),
                          }
                        )
                      }
                      disabled={runMutation.isPending || inferred?.data?.status === "completed"}
                      className="text-xs font-medium px-3 py-1.5 rounded-md bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50 transition-colors"
                    >
                      {runMutation.isPending
                        ? "Analyzing…"
                        : inferred?.data?.status === "completed"
                          ? "Analyzed"
                          : "Run AI analysis"}
                    </button>
                  </div>

                  {inferred?.data?.status === "completed" && (
                    <div className="space-y-2">
                      <AnomalyBadge
                        score={inferred.data.anomaly_score ?? 0}
                        detected={(inferred.data.anomaly_score ?? 0) >= 50}
                      />
                      <p className="text-sm text-muted-foreground">
                        Suggested GL:{" "}
                        <span className="font-medium text-zinc-900">
                          {inferred.data.suggested_gl_account ?? "—"}
                        </span>
                      </p>
                    </div>
                  )}
                  {inferred?.data?.status === "failed" && (
                    <p className="text-sm text-red-600">
                      {inferred.data.error ?? "Analysis failed."}
                    </p>
                  )}
                </div>
              </div>
            )}
          </DrawerContent>
        </Drawer>
      </DrawerProvider>
    </div>
  )
}
