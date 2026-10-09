"use client"

import { useState } from "react"
import { ColumnDef } from "@tanstack/react-table"
import { z } from "zod"
import { DataTable } from "@/components/ui/data-table"
import {
  Drawer,
  DrawerProvider,
  DrawerHeader,
  DrawerContent,
  DrawerFooter,
} from "@/components/ui/drawer"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Icon } from "@/components/ui/icon"
import { formatCurrency } from "@/lib/currency"
import type { Product } from "@/lib/api"
import {
  useAdjustInventory,
  useCreateProduct,
  useProductInventory,
  useProducts,
} from "@/hooks/useProducts"

const productSchema = z.object({
  name: z.string().min(2, "Nama minimal 2 karakter"),
  sku: z.string().min(1, "SKU wajib diisi"),
  description: z.string().optional(),
  price: z.coerce.number().gt(0, "Harga harus lebih dari 0"),
})

const columns: ColumnDef<Product>[] = [
  {
    accessorKey: "sku",
    header: "SKU",
    cell: ({ row }) => (
      <span className="font-mono text-xs font-medium">{row.original.sku}</span>
    ),
  },
  {
    accessorKey: "name",
    header: "Name",
    cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
  },
  {
    accessorKey: "description",
    header: "Description",
    cell: ({ row }) => (
      <span className="text-muted-foreground line-clamp-1 max-w-[280px]">
        {row.original.description || "—"}
      </span>
    ),
  },
  {
    accessorKey: "price",
    header: "Price",
    cell: ({ row }) => formatCurrency(row.original.price),
  },
  {
    accessorKey: "created_at",
    header: "Created",
    cell: ({ row }) => new Date(row.original.created_at).toLocaleDateString(),
  },
]

export default function ProductsPage() {
  const { data: products, isLoading } = useProducts()
  const [creating, setCreating] = useState(false)
  const [detail, setDetail] = useState<Product | null>(null)
  const { data: inventory, isLoading: inventoryLoading } = useProductInventory(
    detail?.id ?? null
  )
  const createProduct = useCreateProduct()
  const adjustInventory = useAdjustInventory(detail?.id ?? null)

  const [form, setForm] = useState({
    name: "",
    sku: "",
    description: "",
    price: "",
  })
  const [formError, setFormError] = useState<string | null>(null)

  function resetForm() {
    setForm({ name: "", sku: "", description: "", price: "" })
    setFormError(null)
  }

  function submitCreate() {
    const parsed = productSchema.safeParse(form)
    if (!parsed.success) {
      setFormError(parsed.error.issues[0]?.message ?? "Form tidak valid")
      return
    }
    setFormError(null)
    createProduct.mutate(
      {
        name: parsed.data.name,
        sku: parsed.data.sku,
        description: parsed.data.description || undefined,
        price: parsed.data.price,
      },
      {
        onSuccess: () => {
          setCreating(false)
          resetForm()
        },
        onError: (err) => setFormError(err.message),
      }
    )
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold flex items-center gap-2">
            <Icon name="Boxes" size="lg" />
            Products & Inventory
          </h1>
          <p className="text-sm text-muted-foreground mt-1">
            Kelola katalog produk dan stok gudang per tenant.
          </p>
        </div>
        <Button onClick={() => setCreating(true)}>
          <Icon name="Plus" size="sm" className="mr-2" />
          New Product
        </Button>
      </div>

      <DataTable
        columns={columns}
        data={products ?? []}
        isLoading={isLoading}
        onRowClick={(row) => setDetail(row)}
      />

      {/* ── Create drawer ── */}
      <DrawerProvider
        open={creating}
        onOpenChange={(open) => {
          if (!open) {
            setCreating(false)
            resetForm()
          }
        }}
      >
        <Drawer>
          <DrawerHeader title="New Product" icon={<Icon name="Boxes" />} />
          <DrawerContent>
            <form
              className="space-y-4"
              onSubmit={(e) => {
                e.preventDefault()
                submitCreate()
              }}
            >
              {formError && (
                <p className="text-sm text-red-600 bg-red-50 border border-red-200 rounded-md px-3 py-2">
                  {formError}
                </p>
              )}
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Name *</label>
                <Input
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="Contoh: Kertas A4 80gsm"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">SKU *</label>
                <Input
                  value={form.sku}
                  onChange={(e) => setForm({ ...form, sku: e.target.value })}
                  placeholder="Contoh: A4-80-WH"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Price (IDR)</label>
                <Input
                  type="number"
                  min={0}
                  step="0.01"
                  value={form.price}
                  onChange={(e) => setForm({ ...form, price: e.target.value })}
                  placeholder="0"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Description</label>
                <Input
                  value={form.description}
                  onChange={(e) => setForm({ ...form, description: e.target.value })}
                  placeholder="Deskripsi singkat"
                />
              </div>
              <DrawerFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setCreating(false)
                    resetForm()
                  }}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={createProduct.isPending}>
                  {createProduct.isPending ? "Menyimpan…" : "Create Product"}
                </Button>
              </DrawerFooter>
            </form>
          </DrawerContent>
        </Drawer>
      </DrawerProvider>

      {/* ── Detail drawer ── */}
      <DrawerProvider
        open={!!detail}
        onOpenChange={(open) => !open && setDetail(null)}
      >
        <Drawer>
          <DrawerHeader
            title={detail ? detail.name : "Product"}
            icon={<Icon name="Boxes" />}
          />
          <DrawerContent>
            {detail && (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-muted-foreground">SKU</p>
                    <p className="font-mono text-sm font-medium">{detail.sku}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">Price</p>
                    <p className="font-medium">{formatCurrency(detail.price)}</p>
                  </div>
                  <div className="col-span-2">
                    <p className="text-sm text-muted-foreground">Description</p>
                    <p className="text-sm">{detail.description || "—"}</p>
                  </div>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-2">
                    <p className="text-sm font-semibold">Inventory</p>
                    {inventoryLoading && (
                      <span className="text-xs text-muted-foreground">
                        Loading…
                      </span>
                    )}
                  </div>
                  {!inventoryLoading &&
                  (!inventory || inventory.length === 0) ? (
                    <p className="text-sm text-muted-foreground border rounded-md p-4 text-center">
                      Belum ada catatan stok. Stok awal dibuat otomatis saat
                      produk dibuat.
                    </p>
                  ) : (
                    <div className="space-y-2">
                      {inventory?.map((item) => (
                        <div
                          key={item.id}
                          className="flex items-center justify-between border rounded-md px-4 py-3"
                        >
                          <div>
                            <p className="text-sm font-medium">
                              {item.warehouse_location || "Main"}
                            </p>
                            <p className="text-xs text-muted-foreground">
                              Qty: {item.quantity}
                            </p>
                          </div>
                          <div className="flex items-center gap-2">
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={adjustInventory.isPending}
                              onClick={() =>
                                adjustInventory.mutate({
                                  inventoryId: item.id,
                                  delta: -1,
                                })
                              }
                            >
                              −1
                            </Button>
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={adjustInventory.isPending}
                              onClick={() =>
                                adjustInventory.mutate({
                                  inventoryId: item.id,
                                  delta: 1,
                                })
                              }
                            >
                              +1
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
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
