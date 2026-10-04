"use client"

import { useState } from "react"
import { Icon } from "@/components/ui/icon"
import { Button } from "@/components/ui/button"
import { Drawer, DrawerProvider, DrawerContent } from "@/components/ui/drawer"
import { CoATable } from "@/components/accounting/CoATable"
import { CoADrawer } from "@/components/accounting/CoADrawer"
import { useAccounts, useCreateAccount } from "@/hooks/useAccounts"
import { formatCurrency } from "@/lib/currency"
import type { Account } from "@/lib/api"

const typeCounts = (accounts: Account[]) => {
  const counts: Record<string, number> = {
    Asset: 0, Liability: 0, Equity: 0, Revenue: 0, Expense: 0,
  }
  accounts.forEach((a) => { counts[a.type] = (counts[a.type] ?? 0) + 1 })
  return counts
}

export default function ChartOfAccountsPage() {
  const { data: accounts, isLoading } = useAccounts()
  const createAccount = useCreateAccount()
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [selected, setSelected] = useState<Account | null>(null)

  const counts = typeCounts(accounts ?? [])

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Icon name="Calculator" size="lg" />
          <div>
            <h1 className="text-2xl font-bold">Chart of Accounts</h1>
            <p className="text-sm text-muted-foreground">
              Manage your chart of accounts structure
            </p>
          </div>
        </div>
        <Button onClick={() => setDrawerOpen(true)}>
          <Icon name="Plus" size="sm" className="mr-2" />
          New Account
        </Button>
      </div>

      {/* Type summary cards */}
      <div className="grid grid-cols-5 gap-3">
        {(["Asset", "Liability", "Equity", "Revenue", "Expense"] as const).map((t) => {
          const colors: Record<string, { bg: string; border: string; text: string }> = {
            Asset:     { bg: "bg-blue-50",    border: "border-blue-200",    text: "text-blue-800"   },
            Liability: { bg: "bg-red-50",     border: "border-red-200",     text: "text-red-800"    },
            Equity:    { bg: "bg-purple-50",  border: "border-purple-200",  text: "text-purple-800" },
            Revenue:   { bg: "bg-green-50",   border: "border-green-200",   text: "text-green-800"  },
            Expense:   { bg: "bg-orange-50",  border: "border-orange-200",  text: "text-orange-800" },
          }
          const c = colors[t]
          return (
            <div key={t} className={`rounded-lg border ${c.border} ${c.bg} p-3`}>
              <p className={`text-xs font-medium ${c.text}`}>{t}</p>
              <p className="text-2xl font-bold mt-1">{counts[t] ?? 0}</p>
            </div>
          )
        })}
      </div>

      {/* Table */}
      <CoATable
        data={accounts ?? []}
        isLoading={isLoading}
        onRowClick={(row) => setSelected(row)}
      />

      {/* Create Drawer */}
      <DrawerProvider open={drawerOpen} onOpenChange={setDrawerOpen}>
        <Drawer>
          <DrawerContent>
            <CoADrawer
              isSubmitting={createAccount.isPending}
              onSubmit={(data) => {
                createAccount.mutate(data, {
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
                  <Icon name="Calculator" size="lg" />
                  <h2 className="text-lg font-semibold">Account Detail</h2>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">Code</p>
                    <p className="font-mono font-medium">{selected.code}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Name</p>
                    <p className="font-medium">{selected.name}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Type</p>
                    <p className="font-medium">{selected.type}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Balance</p>
                    <p className="font-medium tabular-nums">
                      {formatCurrency(selected.balance)}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Parent ID</p>
                    <p className="font-mono text-sm">{selected.parent_id ?? "None (top-level)"}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Status</p>
                    <p className={selected.is_active ? "text-green-600 font-medium" : "text-muted-foreground"}>
                      {selected.is_active ? "Active" : "Inactive"}
                    </p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Created</p>
                    <p className="text-sm">{new Date(selected.created_at).toLocaleString()}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Updated</p>
                    <p className="text-sm">{new Date(selected.updated_at).toLocaleString()}</p>
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
