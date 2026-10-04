"use client"

import { useState } from "react"
import { Icon } from "@/components/ui/icon"
import { Button } from "@/components/ui/button"
import { Drawer, DrawerProvider, DrawerContent } from "@/components/ui/drawer"
import { JETable } from "@/components/accounting/JETable"
import { JEDrawer } from "@/components/accounting/JEDrawer"
import { useJournalEntries, useCreateJournalEntry } from "@/hooks/useJournalEntries"
import { useAccounts } from "@/hooks/useAccounts"
import { formatCurrency } from "@/lib/currency"
import type { JournalEntry } from "@/lib/api"

export default function JournalEntriesPage() {
  const { data: entries, isLoading } = useJournalEntries()
  const createEntry = useCreateJournalEntry()
  const { data: accounts } = useAccounts()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [selected, setSelected] = useState<JournalEntry | null>(null)

  const accountsForDropdown = (accounts ?? []).map((a) => ({
    id: a.id,
    code: a.code,
    name: a.name,
  }))

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Icon name="BookOpen" size="lg" />
          <div>
            <h1 className="text-2xl font-bold">Journal Entries</h1>
            <p className="text-sm text-muted-foreground">
              Record and manage double-entry accounting transactions
            </p>
          </div>
        </div>
        <Button onClick={() => setDrawerOpen(true)}>
          <Icon name="Plus" size="sm" className="mr-2" />
          New Entry
        </Button>
      </div>

      {/* Table */}
      <JETable
        data={entries ?? []}
        isLoading={isLoading}
        onRowClick={(row) => setSelected(row)}
      />

      {/* Create Drawer */}
      <DrawerProvider open={drawerOpen} onOpenChange={setDrawerOpen}>
        <Drawer>
          <DrawerContent>
            <JEDrawer
              accounts={accountsForDropdown}
              isSubmitting={createEntry.isPending}
              onSubmit={(data) => {
                createEntry.mutate(data, {
                  onSuccess: () => setDrawerOpen(false),
                })
              }}
            />
          </DrawerContent>
        </Drawer>
      </DrawerProvider>

      {/* Detail Drawer */}
      <DrawerProvider open={!!selected} onOpenChange={(open) => !open && setSelected(null)}>
        <Drawer>
          <DrawerContent>
            {selected && (
              <div className="space-y-6">
                <div className="flex items-center gap-3 pb-4 border-b">
                  <Icon name="BookOpen" size="lg" />
                  <h2 className="text-lg font-semibold">Journal Entry Detail</h2>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Date</p>
                    <p className="font-medium">{new Date(selected.date).toLocaleDateString()}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Reference</p>
                    <p className="font-mono text-sm font-medium">{selected.reference_id}</p>
                  </div>
                  <div className="col-span-2">
                    <p className="text-sm text-muted-foreground">Description</p>
                    <p className="font-medium">{selected.description}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Total Debit</p>
                    <p className="font-medium tabular-nums">{formatCurrency(selected.total_debit)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Total Credit</p>
                    <p className="font-medium tabular-nums">{formatCurrency(selected.total_credit)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Status</p>
                    <p className="font-medium capitalize">{selected.status}</p>
                  </div>
                </div>

                {/* Line items table */}
                <div>
                  <h3 className="text-sm font-medium mb-2">Line Items</h3>
                  <div className="rounded-md border">
                    <table className="w-full text-sm">
                      <thead className="[&_tr]:border-b bg-muted/50">
                        <tr>
                          <th className="p-2 text-left font-medium text-muted-foreground">Account</th>
                          <th className="p-2 text-right font-medium text-muted-foreground">Debit</th>
                          <th className="p-2 text-right font-medium text-muted-foreground">Credit</th>
                        </tr>
                      </thead>
                      <tbody>
                        {selected.lines.map((line, i) => (
                          <tr key={line.id ?? i} className="border-b last:border-0">
                            <td className="p-2">{line.account_name ?? line.account_id}</td>
                            <td className="p-2 text-right tabular-nums">
                              {line.debit > 0 ? formatCurrency(line.debit) : "-"}
                            </td>
                            <td className="p-2 text-right tabular-nums">
                              {line.credit > 0 ? formatCurrency(line.credit) : "-"}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 text-sm text-muted-foreground">
                  <div>
                    <p>Created</p>
                    <p>{new Date(selected.created_at).toLocaleString()}</p>
                  </div>
                  <div>
                    <p>Updated</p>
                    <p>{new Date(selected.updated_at).toLocaleString()}</p>
                  </div>
                </div>
              </div>
            )}
          </DrawerContent>
        </Drawer>
      </DrawerProvider>
    </div>
  )
}
