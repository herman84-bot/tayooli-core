"use client"

import React, { useState, useMemo } from "react"
import { z } from "zod"
import {
  Boxes,
  Plus,
  Search,
  RefreshCw,
  AlertCircle,
  Package,
  Layers,
  ArrowRight,
  TrendingUp,
  Pencil,
  Trash2,
} from "lucide-react"
import {
  useProducts,
  useCreateProduct,
  useUpdateProduct,
  useDeleteProduct,
  useProductInventory,
  useAdjustInventory,
} from "@/hooks/useProducts"
import type { Product } from "@/lib/api"
import { formatCurrency } from "@/lib/currency"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { ExportModal, ExportButton, type ExportFilter } from "@/components/ui/ExportModal"
import type { ExportColumn } from "@/lib/export"
import {
  Drawer,
  DrawerProvider,
  DrawerHeader,
  DrawerContent,
  DrawerFooter,
} from "@/components/ui/drawer"

function fmtExportDate(iso?: string): string {
  if (!iso) return ""
  const d = new Date(iso)
  return Number.isNaN(d.getTime())
    ? ""
    : d.toLocaleString("id-ID", { day: "2-digit", month: "2-digit", year: "numeric", hour: "2-digit", minute: "2-digit" })
}

const productSchema = z.object({
  name: z.string().trim().min(2, "Nama produk minimal 2 karakter").max(255, "Nama produk maksimal 255 karakter"),
  sku: z.string().trim().min(1, "SKU wajib diisi").max(100, "SKU maksimal 100 karakter"),
  description: z.string().trim().max(2000, "Deskripsi maksimal 2000 karakter").optional(),
  price: z.coerce.number().min(0, "Harga tidak boleh negatif"),
})

export default function ProductsPage() {
  const { data: products = [], isLoading, isError, error, refetch, isFetching } = useProducts()
  const createProduct = useCreateProduct()
  const updateProduct = useUpdateProduct()
  const deleteProduct = useDeleteProduct()

  const [searchQuery, setSearchQuery] = useState("")
  const [creating, setCreating] = useState(false)
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(null)

  // Edit product state
  const [editingProduct, setEditingProduct] = useState<Product | null>(null)
  const [editFormData, setEditFormData] = useState({
    name: "",
    sku: "",
    description: "",
    price: "",
  })
  const [editFormError, setEditFormError] = useState<string | null>(null)

  // Delete product state
  const [deletingProduct, setDeletingProduct] = useState<Product | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  // Product form state
  const [formData, setFormData] = useState({
    name: "",
    sku: "",
    description: "",
    price: "",
  })
  const [formError, setFormError] = useState<string | null>(null)

  // Selected product inventory query and adjust
  const { data: inventory, isLoading: inventoryLoading } = useProductInventory(
    selectedProduct?.id ?? null
  )
  const adjustInventory = useAdjustInventory(selectedProduct?.id ?? null)

  const resetForm = () => {
    setFormData({ name: "", sku: "", description: "", price: "" })
    setFormError(null)
  }

  const handleOpenEdit = (p: Product, e: React.MouseEvent) => {
    e.stopPropagation()
    setEditingProduct(p)
    setEditFormData({
      name: p.name,
      sku: p.sku,
      description: p.description || "",
      price: String(p.price),
    })
    setEditFormError(null)
  }

  const handleEditSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!editingProduct) return
    setEditFormError(null)

    const parsed = productSchema.safeParse(editFormData)
    if (!parsed.success) {
      setEditFormError(parsed.error.issues[0]?.message ?? "Form tidak valid")
      return
    }

    updateProduct.mutate(
      {
        id: editingProduct.id,
        input: {
          name: parsed.data.name,
          sku: parsed.data.sku,
          description: parsed.data.description || undefined,
          price: parsed.data.price,
        },
      },
      {
        onSuccess: () => {
          setEditingProduct(null)
          if (selectedProduct?.id === editingProduct.id) {
            setSelectedProduct({
              ...selectedProduct,
              name: parsed.data.name,
              sku: parsed.data.sku,
              description: parsed.data.description || "",
              price: parsed.data.price,
            })
          }
        },
        onError: (err) => {
          setEditFormError(err instanceof Error ? err.message : "Gagal memperbarui produk")
        },
      }
    )
  }

  const handleOpenDelete = (p: Product, e: React.MouseEvent) => {
    e.stopPropagation()
    setDeletingProduct(p)
    setDeleteError(null)
  }

  const handleDeleteConfirm = () => {
    if (!deletingProduct) return
    setDeleteError(null)

    deleteProduct.mutate(deletingProduct.id, {
      onSuccess: () => {
        if (selectedProduct?.id === deletingProduct.id) {
          setSelectedProduct(null)
        }
        setDeletingProduct(null)
      },
      onError: (err) => {
        const raw = err instanceof Error ? err.message : ""
        const isConflict = /sku|mutasi|stok/i.test(raw)
        setDeleteError(
          isConflict
            ? "Produk tidak dapat dihapus karena SKU sudah digunakan atau memiliki riwayat mutasi stok di gudang/transaksi."
            : raw || "Terjadi kendala saat menghapus produk. Silakan coba lagi."
        )
      },
    })
  }

  // Filter products by search query
  const filteredProducts = useMemo(() => {
    if (!searchQuery.trim()) return products
    const q = searchQuery.toLowerCase()
    return products.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.sku.toLowerCase().includes(q) ||
        (p.description && p.description.toLowerCase().includes(q))
    )
  }, [products, searchQuery])

  const [exporting, setExporting] = useState(false)
  const productExportColumns = useMemo<ExportColumn<Product>[]>(
    () => [
      { header: "SKU", value: (p) => p.sku, width: 16 },
      { header: "Nama Produk", value: (p) => p.name, width: 32 },
      { header: "Deskripsi", value: (p) => p.description ?? "", width: 40 },
      { header: "Harga (Rp)", value: (p) => Number(p.price) || 0, width: 16, align: "right" },
      { header: "Dibuat", value: (p) => fmtExportDate(p.created_at), width: 18 },
      { header: "Diperbarui", value: (p) => fmtExportDate(p.updated_at), width: 18 },
    ],
    []
  )
  const productExportFilters = useMemo<ExportFilter<Product>[]>(
    () => [
      { type: "dateRange", id: "created", label: "Tanggal dibuat", getDate: (p) => p.created_at },
      {
        type: "select",
        id: "price",
        label: "Rentang harga",
        options: [
          { value: "lt50", label: "< Rp50.000" },
          { value: "50to500", label: "Rp50.000 – Rp500.000" },
          { value: "gt500", label: "> Rp500.000" },
        ],
        match: (p, v) => {
          const n = Number(p.price) || 0
          return v === "lt50" ? n < 50_000 : v === "50to500" ? n >= 50_000 && n <= 500_000 : n > 500_000
        },
      },
    ],
    []
  )

  // Metrics
  const metrics = useMemo(() => {
    const totalCount = products.length
    const totalPrice = products.reduce((acc, p) => acc + (Number(p.price) || 0), 0)
    const avgPrice = totalCount > 0 ? totalPrice / totalCount : 0
    return { totalCount, totalPrice, avgPrice }
  }, [products])

  // Submit product creation
  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    setFormError(null)

    const parsed = productSchema.safeParse(formData)
    if (!parsed.success) {
      setFormError(parsed.error.issues[0]?.message ?? "Form tidak valid")
      return
    }

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
        onError: (err) => {
          setFormError(err instanceof Error ? err.message : "Gagal menambahkan produk")
        },
      }
    )
  }

  return (
    <div className="min-h-screen bg-[#F8FAFC] pb-16">
      {/* ── Top Header ── */}
      <div className="bg-white border-b border-[#E2E8F0] shadow-xs">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 py-6 flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <div className="p-3 bg-blue-50 text-[#2563EB] rounded-xl border border-blue-100 shadow-xs">
              <Boxes className="w-6 h-6" />
            </div>
            <div>
              <h1 className="text-xl sm:text-2xl font-bold text-slate-900 flex items-center gap-2">
                Katalog Produk & Master Item
              </h1>
              <p className="text-xs sm:text-sm text-slate-500 mt-0.5">
                Kelola data master barang, harga jual satuan, SKU, dan pantau stok gudang.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={() => refetch()}
              disabled={isFetching}
              className="p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[44px] min-w-[44px] flex items-center justify-center transition-colors disabled:opacity-50"
              title="Segarkan data"
            >
              <RefreshCw className={`w-4 h-4 ${isFetching ? "animate-spin" : ""}`} />
            </button>

            <ExportButton onClick={() => setExporting(true)} disabled={products.length === 0} />

            <button
              type="button"
              onClick={() => setCreating(true)}
              className="inline-flex items-center justify-center gap-2 px-4 py-2.5 min-h-[44px] rounded-lg font-semibold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] active:scale-95 transition-all shadow-sm"
            >
              <Plus className="w-4 h-4" />
              <span>Tambah Produk</span>
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 pt-6 space-y-6">
        {/* ── Metric Cards ── */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 shadow-xs">
            <div className="flex items-center justify-between text-slate-500 text-xs mb-1">
              <span>Total Produk Aktif</span>
              <Layers className="w-4 h-4 text-slate-400" />
            </div>
            <div className="text-2xl font-bold text-slate-900">{metrics.totalCount}</div>
            <div className="text-[11px] text-slate-400 mt-1">Item terdaftar di katalog</div>
          </div>

          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 shadow-xs">
            <div className="flex items-center justify-between text-slate-500 text-xs mb-1">
              <span>Total Nilai Katalog</span>
              <TrendingUp className="w-4 h-4 text-emerald-500" />
            </div>
            <div className="text-2xl font-bold text-slate-900">
              {formatCurrency(metrics.totalPrice)}
            </div>
            <div className="text-[11px] text-slate-400 mt-1">Akumulasi harga katalog</div>
          </div>

          <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 shadow-xs">
            <div className="flex items-center justify-between text-slate-500 text-xs mb-1">
              <span>Rata-rata Harga Satuan</span>
              <Boxes className="w-4 h-4 text-blue-500" />
            </div>
            <div className="text-2xl font-bold text-slate-900">
              {formatCurrency(metrics.avgPrice)}
            </div>
            <div className="text-[11px] text-slate-400 mt-1">Estimasi rata-rata per item</div>
          </div>
        </div>

        {/* ── Search & Filter Bar ── */}
        <div className="bg-white rounded-xl border border-[#E2E8F0] p-4 flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 shadow-xs">
          <div className="relative flex-1">
            <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Cari berdasarkan nama produk, SKU, atau deskripsi..."
              className="w-full pl-10 pr-4 min-h-[44px] text-sm bg-slate-50 border border-slate-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-[#2563EB] focus:bg-white transition-all"
            />
          </div>
          {searchQuery && (
            <button
              onClick={() => setSearchQuery("")}
              className="text-xs text-slate-500 hover:text-slate-800 px-3 py-2 border border-slate-200 rounded-lg hover:bg-slate-50 transition"
            >
              Reset filter
            </button>
          )}
        </div>

        {/* ── Main Content Area ── */}
        {isLoading ? (
          <div className="bg-white rounded-xl border border-[#E2E8F0] p-12 text-center shadow-xs">
            <div className="w-8 h-8 border-3 border-[#2563EB] border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            <p className="text-sm font-medium text-slate-600">Memuat katalog produk...</p>
          </div>
        ) : isError ? (
          <div className="bg-white rounded-xl border border-red-200 p-8 text-center shadow-xs">
            <AlertCircle className="w-10 h-10 text-red-500 mx-auto mb-3" />
            <h3 className="text-base font-bold text-slate-900">Gagal memuat produk</h3>
            <p className="text-sm text-slate-500 max-w-md mx-auto mt-1 mb-4">
              {error instanceof Error ? error.message : "Terjadi kendala saat menghubungkan ke server."}
            </p>
            <Button variant="outline" onClick={() => refetch()} className="gap-2">
              <RefreshCw className="w-4 h-4" />
              Coba Lagi
            </Button>
          </div>
        ) : filteredProducts.length === 0 ? (
          <div className="bg-white rounded-xl border border-[#E2E8F0] p-12 text-center shadow-xs">
            <Package className="w-12 h-12 text-slate-300 mx-auto mb-3" />
            <h3 className="text-base font-bold text-slate-900">
              {searchQuery ? "Tidak ada produk yang cocok" : "Belum ada produk terdaftar"}
            </h3>
            <p className="text-sm text-slate-500 max-w-md mx-auto mt-1 mb-5">
              {searchQuery
                ? `Tidak ditemukan produk yang cocok dengan pencarian "${searchQuery}".`
                : "Mulai bangun katalog inventaris Anda dengan menambahkan produk pertama."}
            </p>
            {searchQuery ? (
              <Button variant="outline" onClick={() => setSearchQuery("")}>
                Hapus Pencarian
              </Button>
            ) : (
              <button
                onClick={() => setCreating(true)}
                className="inline-flex items-center gap-2 px-4 py-2.5 rounded-lg font-semibold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] transition"
              >
                <Plus className="w-4 h-4" />
                Tambah Produk Pertama
              </button>
            )}
          </div>
        ) : (
          <div className="bg-white rounded-xl border border-[#E2E8F0] shadow-xs overflow-hidden">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-slate-200">
                <thead className="bg-slate-50/70">
                  <tr>
                    <th className="px-6 py-3.5 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">
                      SKU
                    </th>
                    <th className="px-6 py-3.5 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">
                      Nama Produk
                    </th>
                    <th className="px-6 py-3.5 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">
                      Deskripsi
                    </th>
                    <th className="px-6 py-3.5 text-right text-xs font-semibold text-slate-500 uppercase tracking-wider">
                      Harga Satuan
                    </th>
                    <th className="px-6 py-3.5 text-left text-xs font-semibold text-slate-500 uppercase tracking-wider">
                      Tanggal Dibuat
                    </th>
                    <th className="px-6 py-3.5 text-right text-xs font-semibold text-slate-500 uppercase tracking-wider">
                      Aksi
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100 bg-white">
                  {filteredProducts.map((p) => (
                    <tr
                      key={p.id}
                      onClick={() => setSelectedProduct(p)}
                      className="hover:bg-slate-50/80 transition-colors cursor-pointer group"
                    >
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-mono text-xs font-semibold px-2 py-1 bg-slate-100 text-slate-700 rounded-md border border-slate-200">
                          {p.sku}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="font-semibold text-sm text-slate-900 group-hover:text-[#2563EB] transition-colors">
                          {p.name}
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        <span className="text-xs text-slate-500 line-clamp-1 max-w-[280px]">
                          {p.description || "—"}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right">
                        <span className="font-mono font-semibold text-sm text-slate-900">
                          {formatCurrency(p.price)}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-xs text-slate-500">
                        {p.created_at ? new Date(p.created_at).toLocaleDateString("id-ID") : "—"}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-xs">
                        <div className="inline-flex items-center gap-1.5" onClick={(e) => e.stopPropagation()}>
                          <button
                            type="button"
                            onClick={() => setSelectedProduct(p)}
                            className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg border border-slate-200 text-slate-700 bg-white hover:bg-slate-50 font-medium transition-colors shadow-2xs"
                            title="Detail & Stok"
                          >
                            <span>Detail</span>
                            <ArrowRight className="w-3.5 h-3.5" />
                          </button>
                          <button
                            type="button"
                            onClick={(e) => handleOpenEdit(p, e)}
                            className="p-1.5 rounded-lg border border-amber-200 text-amber-700 bg-amber-50 hover:bg-amber-100 transition-colors shadow-2xs"
                            title="Edit Produk"
                          >
                            <Pencil className="w-3.5 h-3.5" />
                          </button>
                          <button
                            type="button"
                            onClick={(e) => handleOpenDelete(p, e)}
                            className="p-1.5 rounded-lg border border-red-200 text-red-700 bg-red-50 hover:bg-red-100 transition-colors shadow-2xs"
                            title="Hapus Produk"
                          >
                            <Trash2 className="w-3.5 h-3.5" />
                          </button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <div className="px-6 py-3 border-t border-slate-100 bg-slate-50/50 flex items-center justify-between text-xs text-slate-500">
              <span>Menampilkan {filteredProducts.length} dari {products.length} produk</span>
              <span>Klik baris produk untuk mengelola stok & detail</span>
            </div>
          </div>
        )}
      </div>

      {/* ── Create Product Drawer ── */}
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
          <DrawerHeader title="Tambah Produk Baru" icon={<Boxes className="w-5 h-5 text-[#2563EB]" />} />
          <DrawerContent>
            <form onSubmit={handleCreateSubmit} className="space-y-4" noValidate>
              {formError && (
                <div className="flex items-start gap-2.5 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs">
                  <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
                  <div>{formError}</div>
                </div>
              )}

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Nama Produk *</label>
                <Input
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="Contoh: Kertas HVS A4 80gsm"
                  className="min-h-[42px]"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Kode SKU *</label>
                <Input
                  value={formData.sku}
                  onChange={(e) => setFormData({ ...formData, sku: e.target.value })}
                  placeholder="Contoh: HVS-A4-80"
                  className="min-h-[42px] font-mono"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Harga Satuan (IDR) *</label>
                <Input
                  type="number"
                  min={0}
                  step="1"
                  value={formData.price}
                  onChange={(e) => setFormData({ ...formData, price: e.target.value })}
                  placeholder="Contoh: 55000"
                  className="min-h-[42px] font-mono"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Deskripsi Produk (Opsional)</label>
                <Input
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Deskripsi singkat item barang..."
                  className="min-h-[42px]"
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
                  Batal
                </Button>
                <Button
                  type="submit"
                  disabled={createProduct.isPending}
                  className="bg-[#2563EB] hover:bg-[#1D4ED8] text-white"
                >
                  {createProduct.isPending ? "Menyimpan…" : "Simpan Produk"}
                </Button>
              </DrawerFooter>
            </form>
          </DrawerContent>
        </Drawer>
      </DrawerProvider>

      {/* ── Edit Product Drawer ── */}
      <DrawerProvider
        open={!!editingProduct}
        onOpenChange={(open) => {
          if (!open) {
            setEditingProduct(null)
            setEditFormError(null)
          }
        }}
      >
        <Drawer>
          <DrawerHeader title="Edit Master Produk" icon={<Pencil className="w-5 h-5 text-amber-600" />} />
          <DrawerContent>
            <form onSubmit={handleEditSubmit} className="space-y-4" noValidate>
              {editFormError && (
                <div className="flex items-start gap-2.5 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs">
                  <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
                  <div>{editFormError}</div>
                </div>
              )}

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Nama Produk *</label>
                <Input
                  value={editFormData.name}
                  onChange={(e) => setEditFormData({ ...editFormData, name: e.target.value })}
                  placeholder="Contoh: Kopi Susu Aren 250ml"
                  className="min-h-[42px]"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Kode SKU * (Unik)</label>
                <Input
                  value={editFormData.sku}
                  onChange={(e) => setEditFormData({ ...editFormData, sku: e.target.value.toUpperCase() })}
                  placeholder="Contoh: SKU-KOPI-250"
                  className="min-h-[42px] font-mono uppercase"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Harga Satuan (Rp) *</label>
                <Input
                  type="number"
                  min="0"
                  value={editFormData.price}
                  onChange={(e) => setEditFormData({ ...editFormData, price: e.target.value })}
                  placeholder="0"
                  className="min-h-[42px] font-mono"
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-xs font-semibold text-slate-700">Deskripsi Produk (Opsional)</label>
                <Input
                  value={editFormData.description}
                  onChange={(e) => setEditFormData({ ...editFormData, description: e.target.value })}
                  placeholder="Deskripsi singkat item barang..."
                  className="min-h-[42px]"
                />
              </div>

              <DrawerFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => {
                    setEditingProduct(null)
                    setEditFormError(null)
                  }}
                >
                  Batal
                </Button>
                <Button
                  type="submit"
                  disabled={updateProduct.isPending}
                  className="bg-amber-600 hover:bg-amber-700 text-white"
                >
                  {updateProduct.isPending ? "Menyimpan…" : "Perbarui Produk"}
                </Button>
              </DrawerFooter>
            </form>
          </DrawerContent>
        </Drawer>
      </DrawerProvider>

      {/* ── Delete Confirmation Dialog ── */}
      {deletingProduct && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-xs">
          <div className="bg-white rounded-2xl max-w-md w-full p-6 shadow-xl border border-slate-200 animate-in fade-in zoom-in-95 duration-150">
            <div className="flex items-center gap-3 text-red-600 mb-4">
              <div className="p-3 bg-red-50 rounded-xl border border-red-100">
                <Trash2 className="w-6 h-6" />
              </div>
              <div>
                <h3 className="text-base font-bold text-slate-900">Hapus Produk?</h3>
                <p className="text-xs text-slate-500 font-mono mt-0.5">{deletingProduct.sku}</p>
              </div>
            </div>

            <p className="text-sm text-slate-600 mb-4">
              Apakah Anda yakin ingin menghapus produk <strong className="text-slate-900">{deletingProduct.name}</strong>?
              Tindakan ini permanen dan tidak dapat dibatalkan.
            </p>

            {deleteError && (
              <div className="flex items-start gap-2.5 p-3 rounded-lg bg-red-50 border border-red-200 text-red-700 text-xs mb-4">
                <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
                <div>{deleteError}</div>
              </div>
            )}

            <div className="flex items-center justify-end gap-2.5 pt-2 border-t border-slate-100">
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setDeletingProduct(null)
                  setDeleteError(null)
                }}
              >
                Batal
              </Button>
              <Button
                type="button"
                disabled={deleteProduct.isPending}
                onClick={handleDeleteConfirm}
                className="bg-red-600 hover:bg-red-700 text-white"
              >
                {deleteProduct.isPending ? "Menghapus…" : "Hapus Produk"}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* ── Detail & Inventory Drawer ── */}
      <DrawerProvider
        open={!!selectedProduct}
        onOpenChange={(open) => !open && setSelectedProduct(null)}
      >
        <Drawer>
          <DrawerHeader
            title={selectedProduct?.name ?? "Detail Produk"}
            icon={<Boxes className="w-5 h-5 text-[#2563EB]" />}
          />
          <DrawerContent>
            {selectedProduct && (
              <div className="space-y-6">
                <div className="grid grid-cols-2 gap-4 bg-slate-50 p-4 rounded-xl border border-slate-200">
                  <div>
                    <p className="text-xs font-medium text-slate-500">Kode SKU</p>
                    <p className="font-mono text-sm font-semibold text-slate-900 mt-0.5">
                      {selectedProduct.sku}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs font-medium text-slate-500">Harga Satuan</p>
                    <p className="font-mono text-sm font-semibold text-slate-900 mt-0.5">
                      {formatCurrency(selectedProduct.price)}
                    </p>
                  </div>
                  <div className="col-span-2 pt-2 border-t border-slate-200">
                    <p className="text-xs font-medium text-slate-500">Deskripsi</p>
                    <p className="text-xs text-slate-700 mt-0.5">
                      {selectedProduct.description || "Tidak ada deskripsi."}
                    </p>
                  </div>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-3">
                    <h4 className="text-xs font-bold text-slate-900 uppercase tracking-wider">
                      Stok Per Lokasi Gudang
                    </h4>
                    {inventoryLoading && (
                      <span className="text-xs text-slate-500 flex items-center gap-1">
                        <RefreshCw className="w-3 h-3 animate-spin" /> Memuat...
                      </span>
                    )}
                  </div>

                  {!inventoryLoading && (!inventory || inventory.length === 0) ? (
                    <div className="text-center py-6 px-4 bg-slate-50 rounded-xl border border-slate-200 text-xs text-slate-500">
                      Belum ada catatan lokasi stok untuk produk ini.
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {inventory?.map((item) => (
                        <div
                          key={item.id}
                          className="flex items-center justify-between bg-white border border-slate-200 rounded-xl p-3 shadow-xs"
                        >
                          <div>
                            <p className="text-xs font-bold text-slate-800">
                              {item.warehouse_location || "Gudang Utama"}
                            </p>
                            <p className="text-xs font-semibold text-slate-500 mt-0.5">
                              Stok saat ini: <span className="font-mono text-slate-900">{item.quantity}</span> unit
                            </p>
                          </div>
                          <div className="flex items-center gap-1.5">
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
                              className="h-8 px-2.5 font-bold"
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
                              className="h-8 px-2.5 font-bold"
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

      <ExportModal<Product>
        open={exporting}
        onClose={() => setExporting(false)}
        title="Katalog Produk"
        filename="katalog-produk"
        columns={productExportColumns}
        allRows={products}
        visibleRows={filteredProducts}
        visibleSummary={searchQuery.trim() ? [`Pencarian: "${searchQuery.trim()}"`] : []}
        filters={productExportFilters}
      />
    </div>
  )
}
