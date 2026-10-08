"use client"

import React, { useState } from "react"
import {
  X,
  Settings,
  ShieldCheck,
  MapPin,
  Plus,
  Trash2,
  CheckCircle2,
  AlertCircle,
  Warehouse,
} from "lucide-react"
import {
  useWMSSettings,
  useUpdateWMSSettings,
  useProductDefaultLocations,
  useSetDefaultLocation,
  useDeleteDefaultLocation,
  useWarehouses,
  useWarehouseLocations,
} from "@/hooks/useWMS"
import { useProducts } from "@/hooks/useProducts"

interface WMSSettingsModalProps {
  isOpen: boolean
  onClose: () => void
}

export function WMSSettingsModal({ isOpen, onClose }: WMSSettingsModalProps) {
  const [activeTab, setActiveTab] = useState<"policy" | "default_racks">("policy")
  const [toast, setToast] = useState<{ type: "success" | "error"; msg: string } | null>(null)

  // Settings
  const { data: settings, refetch: refetchSettings } = useWMSSettings()
  const { mutate: updateSettings, isPending: isUpdatingSettings } = useUpdateWMSSettings()

  // Default Locations
  const { data: defaultLocs = [], refetch: refetchDefaultLocs } = useProductDefaultLocations()
  const { mutate: setDefaultLoc, isPending: isSettingDefault } = useSetDefaultLocation()
  const { mutate: deleteDefaultLoc } = useDeleteDefaultLocation()

  // Masters
  const { data: products = [] } = useProducts()
  const { data: warehouses = [] } = useWarehouses()

  // New Default Rack Form
  const [selectedProductId, setSelectedProductId] = useState("")
  const [selectedWarehouseId, setSelectedWarehouseId] = useState("")
  const [selectedLocationId, setSelectedLocationId] = useState("")

  const { data: locations = [] } = useWarehouseLocations(selectedWarehouseId || null)
  const internalRacks = locations.filter((l) => l.type === "INTERNAL")

  if (!isOpen) return null

  const handleToggleReleaseApproval = (val: boolean) => {
    updateSettings(val, {
      onSuccess: () => {
        setToast({ type: "success", msg: "Pengaturan kebijakan rilis berhasil diperbarui." })
        refetchSettings()
        setTimeout(() => setToast(null), 3000)
      },
      onError: (err: unknown) => {
        setToast({ type: "error", msg: err instanceof Error ? err.message : "Gagal menyimpan pengaturan" })
      },
    })
  }

  const handleAddDefaultRack = (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedProductId || !selectedWarehouseId || !selectedLocationId) {
      setToast({ type: "error", msg: "Lengkapi semua data produk, gudang, dan rak internal." })
      return
    }

    setDefaultLoc(
      {
        product_id: selectedProductId,
        warehouse_id: selectedWarehouseId,
        location_id: selectedLocationId,
      },
      {
        onSuccess: () => {
          setToast({ type: "success", msg: "Rak default produk berhasil ditetapkan." })
          setSelectedProductId("")
          setSelectedLocationId("")
          refetchDefaultLocs()
          setTimeout(() => setToast(null), 3000)
        },
        onError: (err: unknown) => {
          setToast({ type: "error", msg: err instanceof Error ? err.message : "Gagal menetapkan rak default" })
        },
      }
    )
  }

  const handleDeleteDefaultRack = (productId: string, warehouseId: string) => {
    deleteDefaultLoc(
      { productId, warehouseId },
      {
        onSuccess: () => {
          setToast({ type: "success", msg: "Rak default berhasil dihapus." })
          refetchDefaultLocs()
          setTimeout(() => setToast(null), 3000)
        },
      }
    )
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in">
      <div className="w-full max-w-2xl rounded-xl bg-white shadow-xl border border-slate-200 overflow-hidden flex flex-col max-h-[90vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-200 px-6 py-4 bg-slate-50">
          <div className="flex items-center gap-2">
            <Settings className="h-5 w-5 text-primary" />
            <h2 className="text-base font-semibold text-slate-900">Pengaturan Alur WMS Inbound & Rak</h2>
          </div>
          <button onClick={onClose} className="rounded-lg p-1 text-slate-400 hover:bg-slate-200 text-slate-600 transition-colors">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Tab navigation */}
        <div className="flex border-b border-slate-200 bg-white px-6">
          <button
            onClick={() => setActiveTab("policy")}
            className={`py-3 text-xs font-semibold border-b-2 transition-colors mr-6 ${
              activeTab === "policy"
                ? "border-primary text-primary"
                : "border-transparent text-slate-500 hover:text-slate-800"
            }`}
          >
            Kebijakan Rilis Lot (Release Approval PDF-06)
          </button>
          <button
            onClick={() => setActiveTab("default_racks")}
            className={`py-3 text-xs font-semibold border-b-2 transition-colors ${
              activeTab === "default_racks"
                ? "border-primary text-primary"
                : "border-transparent text-slate-500 hover:text-slate-800"
            }`}
          >
            Rak Default Produk (CR-03)
          </button>
        </div>

        {/* Toast */}
        {toast && (
          <div
            className={`mx-6 mt-4 flex items-center gap-2 rounded-lg px-4 py-2.5 text-xs ${
              toast.type === "success"
                ? "border border-emerald-200 bg-emerald-50 text-emerald-800"
                : "border border-rose-200 bg-rose-50 text-rose-800"
            }`}
          >
            {toast.type === "success" ? <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600" /> : <AlertCircle className="h-4 w-4 shrink-0 text-rose-600" />}
            <span>{toast.msg}</span>
          </div>
        )}

        {/* Content */}
        <div className="p-6 overflow-y-auto flex-1 space-y-6">
          {activeTab === "policy" && (
            <div className="space-y-4">
              <div className="rounded-xl border border-slate-200 p-4 bg-slate-50/50 space-y-3">
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <h3 className="text-sm font-semibold text-slate-900">
                      Wajib Persetujuan Rilis Lot (Release Approval)
                    </h3>
                    <p className="text-xs text-slate-500 mt-1 leading-relaxed">
                      Sesuai SOP Pergudangan ISO & PDF-06: Jika diaktifkan, setiap penerimaan barang yang diposting akan
                      memasukkan lot ke status <strong>ON_HOLD (Tertahan)</strong>. Stok tidak dapat dialokasikan untuk
                      penjualan atau Surat Jalan sampai disetujui (Rilis) oleh Supervisor Warehouse.
                    </p>
                  </div>
                  <label className="relative inline-flex items-center cursor-pointer shrink-0 mt-1">
                    <input
                      type="checkbox"
                      checked={settings?.require_release_approval || false}
                      onChange={(e) => handleToggleReleaseApproval(e.target.checked)}
                      disabled={isUpdatingSettings}
                      className="sr-only peer"
                    />
                    <div className="w-11 h-6 bg-slate-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
                  </label>
                </div>
                <div className="text-[11px] text-slate-400 border-t border-slate-200 pt-2">
                  Status saat ini:{" "}
                  <span className="font-semibold text-slate-700">
                    {settings?.require_release_approval
                      ? "AKTIF (Wajib persetujuan rilis oleh Admin / Supervisor)"
                      : "NONAKTIF (Lot otomatis RELEASED siap jual saat penerimaan)"}
                  </span>
                </div>
              </div>
            </div>
          )}

          {activeTab === "default_racks" && (
            <div className="space-y-6">
              {/* Add form */}
              <form onSubmit={handleAddDefaultRack} className="rounded-xl border border-slate-200 p-4 bg-slate-50/60 space-y-3">
                <h4 className="text-xs font-semibold text-slate-800 uppercase tracking-wider flex items-center gap-1.5">
                  <Plus className="h-3.5 w-3.5 text-primary" />
                  Tetapkan Rak Default Baru
                </h4>
                <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                  <div>
                    <label className="block text-[11px] font-medium text-slate-600 mb-1">Pilih Produk</label>
                    <select
                      value={selectedProductId}
                      onChange={(e) => setSelectedProductId(e.target.value)}
                      className="w-full text-xs px-2.5 py-1.5 border border-slate-300 rounded bg-white"
                    >
                      <option value="">-- Pilih Produk --</option>
                      {products.map((p) => (
                        <option key={p.id} value={p.id}>
                          {p.name} ({p.sku})
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-[11px] font-medium text-slate-600 mb-1">Pilih Gudang</label>
                    <select
                      value={selectedWarehouseId}
                      onChange={(e) => setSelectedWarehouseId(e.target.value)}
                      className="w-full text-xs px-2.5 py-1.5 border border-slate-300 rounded bg-white"
                    >
                      <option value="">-- Pilih Gudang --</option>
                      {warehouses.map((w) => (
                        <option key={w.id} value={w.id}>
                          {w.name}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <label className="block text-[11px] font-medium text-slate-600 mb-1">Pilih Rak Internal</label>
                    <select
                      value={selectedLocationId}
                      onChange={(e) => setSelectedLocationId(e.target.value)}
                      disabled={!selectedWarehouseId}
                      className="w-full text-xs px-2.5 py-1.5 border border-slate-300 rounded bg-white disabled:bg-slate-100"
                    >
                      <option value="">-- Pilih Rak --</option>
                      {internalRacks.map((r) => (
                        <option key={r.id} value={r.id}>
                          {r.code} {r.name ? `(${r.name})` : ""}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
                <div className="flex justify-end pt-1">
                  <button
                    type="submit"
                    disabled={isSettingDefault}
                    className="px-3 py-1.5 bg-primary text-primary-foreground text-xs font-medium rounded hover:bg-primary/90 transition-colors shadow-sm"
                  >
                    {isSettingDefault ? "Menyimpan..." : "Simpan Rak Default"}
                  </button>
                </div>
              </form>

              {/* List */}
              <div className="rounded-xl border border-slate-200 overflow-hidden">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-50 text-slate-600 font-semibold border-b border-slate-200">
                    <tr>
                      <th className="px-4 py-2.5">Produk</th>
                      <th className="px-4 py-2.5">Gudang</th>
                      <th className="px-4 py-2.5">Rak Default</th>
                      <th className="px-4 py-2.5 text-center">Aksi</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {defaultLocs.length === 0 ? (
                      <tr>
                        <td colSpan={4} className="px-4 py-6 text-center text-slate-400">
                          Belum ada konfigurasi rak default produk.
                        </td>
                      </tr>
                    ) : (
                      defaultLocs.map((d) => (
                        <tr key={`${d.product_id}-${d.warehouse_id}`} className="hover:bg-slate-50/60">
                          <td className="px-4 py-2.5">
                            <span className="font-medium text-slate-900">{d.product_name}</span>
                            <span className="font-mono text-slate-500 text-[11px] block">{d.product_sku}</span>
                          </td>
                          <td className="px-4 py-2.5 text-slate-700">{d.warehouse_name}</td>
                          <td className="px-4 py-2.5 font-mono font-bold text-primary">{d.location_code}</td>
                          <td className="px-4 py-2.5 text-center">
                            <button
                              onClick={() => handleDeleteDefaultRack(d.product_id, d.warehouse_id)}
                              className="text-rose-600 hover:text-rose-800 p-1 rounded"
                              title="Hapus rak default"
                            >
                              <Trash2 className="h-3.5 w-3.5 inline" />
                            </button>
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-slate-200 px-6 py-3 bg-slate-50 flex justify-end">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-xs font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors"
          >
            Tutup
          </button>
        </div>
      </div>
    </div>
  )
}
