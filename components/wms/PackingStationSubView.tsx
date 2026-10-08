"use client"

import React, { useState, useEffect, useRef } from "react"
import {
  Package,
  Barcode,
  CheckCircle2,
  AlertCircle,
  Volume2,
  VolumeX,
  Scale,
  Box,
  Truck,
  Printer,
  Search,
  RefreshCw,
  Building,
  CheckSquare,
  ArrowRight,
  ClipboardList
} from "lucide-react"
import { api, DeliveryOrder, DeliveryOrderItem } from "@/lib/api"
import { useWarehouses } from "@/hooks/useWMS"
import { PrintThermalAWB } from "@/components/wms/PrintThermalAWB"

interface PackingStationSubViewProps {
  warehouseId?: string
}

export function PackingStationSubView({ warehouseId }: PackingStationSubViewProps) {
  const { data: warehouses = [] } = useWarehouses()
  const [selectedWarehouseId, setSelectedWarehouseId] = useState<string>(
    warehouseId || ""
  )
  const [searchQuery, setSearchQuery] = useState("")
  const [orders, setOrders] = useState<DeliveryOrder[]>([])
  const [loadingOrders, setLoadingOrders] = useState(false)

  // Selected Order for Packing
  const [selectedOrder, setSelectedOrder] = useState<DeliveryOrder | null>(null)
  const [orderItems, setOrderItems] = useState<DeliveryOrderItem[]>([])
  const [loadingItems, setLoadingItems] = useState(false)

  // Scanner & Packaging State
  const [barcodeInput, setBarcodeInput] = useState("")
  const [loadingScan, setLoadingScan] = useState(false)
  const [completing, setCompleting] = useState(false)
  const [audioEnabled, setAudioEnabled] = useState(true)
  const [scanMessage, setScanMessage] = useState<{ type: "success" | "error"; text: string } | null>(null)
  const [lastScannedProduct, setLastScannedProduct] = useState<string | null>(null)

  // Dimensions & Weight
  const [weightKg, setWeightKg] = useState<string>("1.0")
  const [lengthCm, setLengthCm] = useState<string>("30")
  const [widthCm, setWidthCm] = useState<string>("20")
  const [heightCm, setHeightCm] = useState<string>("15")
  const [packagingType, setPackagingType] = useState<string>("KARTON")

  // Modal Print Thermal
  const [showThermalModal, setShowThermalModal] = useState(false)

  const barcodeInputRef = useRef<HTMLInputElement>(null)
  const audioCtxRef = useRef<AudioContext | null>(null)

  useEffect(() => {
    if (warehouseId) {
      setSelectedWarehouseId(warehouseId)
    }
  }, [warehouseId])

  useEffect(() => {
    loadOrders()
  }, [selectedWarehouseId])

  const loadOrders = async () => {
    try {
      setLoadingOrders(true)
      const res = await api.wms.deliveryOrders.list(selectedWarehouseId || undefined)
      // Filter orders ready for packing: CONFIRMED, PICKED, or PACKED
      const packable = (res.data || []).filter(
        (o) => o.status === "CONFIRMED" || o.status === "PICKED" || o.status === "PACKED"
      )
      setOrders(packable)
    } catch {
      // silently fail list
    } finally {
      setLoadingOrders(false)
    }
  }

  const handleSelectOrder = async (order: DeliveryOrder) => {
    try {
      setLoadingItems(true)
      setScanMessage(null)
      setSelectedOrder(order)
      setWeightKg(order.package_weight_kg ? String(order.package_weight_kg) : "1.0")
      setLengthCm(order.package_length_cm ? String(order.package_length_cm) : "30")
      setWidthCm(order.package_width_cm ? String(order.package_width_cm) : "20")
      setHeightCm(order.package_height_cm ? String(order.package_height_cm) : "15")
      setPackagingType(order.packaging_type || "KARTON")

      const res = await api.wms.deliveryOrders.get(order.id)
      setOrderItems(res.items || [])
    } catch (err: any) {
      setScanMessage({ type: "error", text: err?.message || "Gagal memuat rincian item pesanan" })
    } finally {
      setLoadingItems(false)
      setTimeout(() => {
        barcodeInputRef.current?.focus()
      }, 50)
    }
  }

  // Web Audio Beeper (FE-07)
  const playSound = (isSuccess: boolean) => {
    if (!audioEnabled || typeof window === "undefined") return
    try {
      if (!audioCtxRef.current) {
        const AudioCtx = window.AudioContext || (window as any).webkitAudioContext
        if (AudioCtx) audioCtxRef.current = new AudioCtx()
      }
      const ctx = audioCtxRef.current
      if (!ctx) return
      if (ctx.state === "suspended") ctx.resume()

      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.connect(gain)
      gain.connect(ctx.destination)

      const now = ctx.currentTime
      if (isSuccess) {
        // High pitch pleasant rising chime (880 Hz -> 1320 Hz)
        osc.type = "sine"
        osc.frequency.setValueAtTime(880, now)
        osc.frequency.exponentialRampToValueAtTime(1320, now + 0.12)
        gain.gain.setValueAtTime(0.25, now)
        gain.gain.exponentialRampToValueAtTime(0.01, now + 0.15)
        osc.start(now)
        osc.stop(now + 0.15)
      } else {
        // Low pitch buzzing square beep (180 Hz)
        osc.type = "square"
        osc.frequency.setValueAtTime(180, now)
        gain.gain.setValueAtTime(0.3, now)
        gain.gain.exponentialRampToValueAtTime(0.01, now + 0.28)
        osc.start(now)
        osc.stop(now + 0.28)
      }
    } catch {
      // AudioContext failure silently ignored
    }
  }

  // Handle SKU Barcode Scan
  const handleScanSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedOrder) return
    const code = barcodeInput.trim()
    if (!code || loadingScan) return

    setLoadingScan(true)
    setScanMessage(null)

    try {
      const res = await api.wms.deliveryOrders.scanPackItem(selectedOrder.id, code, 1)
      const data = res.data

      // Update scanned quantity in local items
      setOrderItems((prev) =>
        prev.map((it) =>
          it.id === data.item_id
            ? { ...it, packed_qty: Number(data.packed_qty) }
            : it
        )
      )

      setLastScannedProduct(`${data.product_name} (${data.product_sku})`)
      setScanMessage({
        type: "success",
        text: `Berhasil memindai: ${data.product_name} (${data.packed_qty} unit terkemas)`,
      })
      playSound(true)
      setBarcodeInput("")
    } catch (err: any) {
      playSound(false)
      setScanMessage({
        type: "error",
        text: err?.message || "Barcode tidak cocok dengan pesanan ini atau kuantitas telah terpenuhi!",
      })
    } finally {
      setLoadingScan(false)
      setTimeout(() => {
        barcodeInputRef.current?.focus()
      }, 50)
    }
  }

  // Complete Packing (Transitions to PACKED)
  const handleCompletePack = async () => {
    if (!selectedOrder || completing) return
    setCompleting(true)
    setScanMessage(null)

    try {
      const res = await api.wms.deliveryOrders.completePack(selectedOrder.id, {
        package_weight_kg: Number(weightKg) || undefined,
        package_length_cm: Number(lengthCm) || undefined,
        package_width_cm: Number(widthCm) || undefined,
        package_height_cm: Number(heightCm) || undefined,
        packaging_type: packagingType || undefined,
      })
      setSelectedOrder(res.data)
      setScanMessage({
        type: "success",
        text: `Pesanan ${res.data.do_number} telah selesai dikemas 100% (PACKED)!`,
      })
      playSound(true)
      await loadOrders()
    } catch (err: any) {
      playSound(false)
      setScanMessage({
        type: "error",
        text: err?.message || "Gagal menyelesaikan pengemasan. Pastikan seluruh barang telah discan 100%.",
      })
    } finally {
      setCompleting(false)
    }
  }

  // Calculate Progress
  const totalQty = orderItems.reduce((s, it) => s + Number(it.quantity || 0), 0)
  const packedQty = orderItems.reduce((s, it) => s + Number(it.packed_qty || 0), 0)
  const progressPercent = totalQty > 0 ? Math.min(100, Math.round((packedQty / totalQty) * 100)) : 0
  const isAllPacked = totalQty > 0 && packedQty >= totalQty

  const filteredOrders = orders.filter((o) => {
    if (!searchQuery) return true
    const q = searchQuery.toLowerCase()
    return (
      o.do_number.toLowerCase().includes(q) ||
      (o.recipient_name && o.recipient_name.toLowerCase().includes(q)) ||
      (o.customer_name && o.customer_name.toLowerCase().includes(q))
    )
  })

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        {/* ── Left Column: Daftar Pesanan Siap Kemas ── */}
        <div className="bg-white rounded-xl border border-slate-200 overflow-hidden shadow-2xs">
          <div className="p-4 border-b border-slate-200 bg-slate-50/70">
            <h3 className="text-sm font-bold text-slate-900 flex items-center justify-between">
              <span>Pesanan Siap Kemas ({orders.length})</span>
              <button
                type="button"
                onClick={loadOrders}
                disabled={loadingOrders}
                className="p-1 rounded text-slate-400 hover:text-slate-700 transition"
                title="Segarkan daftar pesanan"
              >
                <RefreshCw className={`h-3.5 w-3.5 ${loadingOrders ? "animate-spin" : ""}`} />
              </button>
            </h3>
            <div className="mt-2.5 relative">
              <Search className="h-3.5 w-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                placeholder="Cari No. DO, penerima..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-8 pr-2.5 py-1.5 bg-white border border-slate-200 rounded-lg text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:ring-1 focus:ring-indigo-500"
              />
            </div>
          </div>

          <div className="divide-y divide-slate-100 max-h-[600px] overflow-y-auto">
            {loadingOrders ? (
              <div className="p-8 text-center text-xs text-slate-400">Memuat pesanan...</div>
            ) : filteredOrders.length === 0 ? (
              <div className="p-8 text-center text-xs text-slate-400 space-y-1">
                <Box className="h-8 w-8 mx-auto text-slate-300 mb-1" />
                <p className="font-semibold text-slate-700">Tidak ada antrean kemas</p>
                <p>Seluruh Surat Jalan telah dikemas atau belum dikonfirmasi.</p>
              </div>
            ) : (
              filteredOrders.map((o) => {
                const isSelected = selectedOrder?.id === o.id
                return (
                  <button
                    key={o.id}
                    type="button"
                    onClick={() => handleSelectOrder(o)}
                    className={`w-full text-left p-3.5 transition flex items-center justify-between gap-3 ${
                      isSelected
                        ? "bg-indigo-50/70 border-l-4 border-indigo-600"
                        : "hover:bg-slate-50 border-l-4 border-transparent"
                    }`}
                  >
                    <div className="space-y-0.5 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-bold text-xs text-slate-900 truncate">
                          {o.do_number}
                        </span>
                        <span
                          className={`text-[9px] font-bold px-1.5 py-0.5 rounded uppercase ${
                            o.status === "PACKED"
                              ? "bg-emerald-100 text-emerald-800"
                              : "bg-amber-100 text-amber-800"
                          }`}
                        >
                          {o.status}
                        </span>
                      </div>
                      <div className="text-[11px] text-slate-600 truncate">
                        {o.customer_name || o.recipient_name || "Tanpa Nama"}
                      </div>
                      <div className="text-[10px] text-slate-400">
                        {o.expedition_name || "Internal"} &bull; {new Date(o.created_at).toLocaleDateString("id-ID")}
                      </div>
                    </div>
                    <ArrowRight className={`h-4 w-4 shrink-0 ${isSelected ? "text-indigo-600" : "text-slate-300"}`} />
                  </button>
                )
              })
            )}
          </div>
        </div>

        {/* ── Right Column: Meja Kemas Barcode Workbench ── */}
        <div className="lg:col-span-2 bg-white rounded-xl border border-slate-200 overflow-hidden shadow-2xs p-6 space-y-6">
          {!selectedOrder ? (
            <div className="py-24 text-center text-slate-400 space-y-2">
              <ClipboardList className="h-12 w-12 mx-auto text-slate-300" />
              <h4 className="text-sm font-bold text-slate-700">Pilih Surat Jalan untuk Mulai Memeriksa</h4>
              <p className="text-xs max-w-sm mx-auto text-slate-400">
                Pilih pesanan dari daftar di sisi kiri untuk memverifikasi kecocokan barcode SKU barang sebelum segel
                paket dan penerbitan resi ekspedisi.
              </p>
            </div>
          ) : (
            <>
              {/* Header Info Pesanan */}
              <div className="flex flex-col sm:flex-row sm:items-center justify-between pb-4 border-b border-slate-200 gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-mono font-black text-lg text-slate-900">{selectedOrder.do_number}</span>
                    <span
                      className={`text-[10px] font-bold px-2 py-0.5 rounded-full uppercase border ${
                        selectedOrder.status === "PACKED"
                          ? "bg-emerald-50 text-emerald-700 border-emerald-200"
                          : "bg-indigo-50 text-indigo-700 border-indigo-200"
                      }`}
                    >
                      {selectedOrder.status}
                    </span>
                  </div>
                  <p className="text-xs text-slate-500 mt-0.5">
                    Penerima: <strong className="text-slate-800">{selectedOrder.customer_name || selectedOrder.recipient_name}</strong>
                    {" &bull; "}
                    Kurir: <strong className="text-slate-800">{selectedOrder.expedition_name || "Internal"}</strong>
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => setAudioEnabled(!audioEnabled)}
                    className={`p-2 rounded-lg border text-xs flex items-center gap-1.5 transition ${
                      audioEnabled
                        ? "bg-indigo-50 border-indigo-200 text-indigo-700"
                        : "bg-slate-50 border-slate-200 text-slate-400"
                    }`}
                    title={audioEnabled ? "Matikan Audio Beeper" : "Aktifkan Audio Beeper"}
                  >
                    {audioEnabled ? <Volume2 className="h-4 w-4" /> : <VolumeX className="h-4 w-4" />}
                    <span className="text-[11px] font-medium">{audioEnabled ? "Audio Aktif" : "Bisu"}</span>
                  </button>

                  {selectedOrder.status === "PACKED" && (
                    <button
                      type="button"
                      onClick={() => setShowThermalModal(true)}
                      className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-slate-900 hover:bg-slate-800 text-white rounded-lg text-xs font-semibold shadow-2xs transition"
                    >
                      <Printer className="h-3.5 w-3.5" />
                      <span>Cetak Resi AWB (FE-08)</span>
                    </button>
                  )}
                </div>
              </div>

              {/* Status Alert Notification */}
              {scanMessage && (
                <div
                  className={`p-3 rounded-xl border text-xs flex items-center gap-2 ${
                    scanMessage.type === "success"
                      ? "bg-emerald-50 text-emerald-800 border-emerald-200"
                      : "bg-rose-50 text-rose-800 border-rose-200 animate-shake"
                  }`}
                >
                  {scanMessage.type === "success" ? (
                    <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600" />
                  ) : (
                    <AlertCircle className="h-4 w-4 shrink-0 text-rose-600" />
                  )}
                  <span className="font-medium">{scanMessage.text}</span>
                </div>
              )}

              {/* ── Scan Barcode Input Prominent (FE-07) ── */}
              <form onSubmit={handleScanSubmit} className="space-y-2">
                <label className="block text-xs font-bold text-slate-800 uppercase tracking-wider">
                  Pindai Barcode / SKU Barang (Scan to Verify)
                </label>
                <div className="relative">
                  <Barcode className="h-5 w-5 absolute left-3.5 top-1/2 -translate-y-1/2 text-indigo-600" />
                  <input
                    ref={barcodeInputRef}
                    type="text"
                    value={barcodeInput}
                    onChange={(e) => setBarcodeInput(e.target.value)}
                    placeholder="Arahkan barcode scanner fisik atau ketik SKU lalu Enter..."
                    disabled={loadingScan || isAllPacked}
                    className="w-full pl-11 pr-24 py-3 bg-slate-50 border-2 border-indigo-200 rounded-xl text-sm font-mono text-slate-900 focus:outline-none focus:border-indigo-600 focus:bg-white transition disabled:opacity-50"
                    autoFocus
                  />
                  <button
                    type="submit"
                    disabled={loadingScan || !barcodeInput.trim() || isAllPacked}
                    className="absolute right-2 top-1/2 -translate-y-1/2 px-3 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg text-xs font-semibold disabled:opacity-50 transition"
                  >
                    {loadingScan ? "..." : "Pindai"}
                  </button>
                </div>
              </form>

              {/* ── Progress Bar ── */}
              <div className="space-y-1.5">
                <div className="flex justify-between items-center text-xs font-bold text-slate-700">
                  <span>Kemajuan Pemeriksaan Kemasan:</span>
                  <span className={isAllPacked ? "text-emerald-600" : "text-indigo-600"}>
                    {packedQty} / {totalQty} Unit ({progressPercent}%)
                  </span>
                </div>
                <div className="h-3 w-full bg-slate-100 rounded-full overflow-hidden p-0.5 border border-slate-200">
                  <div
                    className={`h-full rounded-full transition-all duration-300 ${
                      isAllPacked
                        ? "bg-emerald-500"
                        : "bg-gradient-to-r from-indigo-500 to-indigo-600"
                    }`}
                    style={{ width: `${progressPercent}%` }}
                  />
                </div>
              </div>

              {/* ── Items Checklist Table ── */}
              <div className="border border-slate-200 rounded-xl overflow-hidden">
                <table className="w-full text-left text-xs border-collapse">
                  <thead>
                    <tr className="bg-slate-50 text-slate-500 font-semibold uppercase tracking-wider border-b border-slate-200">
                      <th className="py-2.5 px-3 w-10 text-center">Status</th>
                      <th className="py-2.5 px-3">SKU & Nama Barang</th>
                      <th className="py-2.5 px-3">Batch / Kedaluwarsa</th>
                      <th className="py-2.5 px-3 text-right">Target</th>
                      <th className="py-2.5 px-3 text-right">Terkemas</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {orderItems.map((it) => {
                      const isComplete = Number(it.packed_qty || 0) >= Number(it.quantity || 0)
                      return (
                        <tr
                          key={it.id}
                          className={`transition ${
                            isComplete ? "bg-emerald-50/40" : "hover:bg-slate-50"
                          }`}
                        >
                          <td className="py-2.5 px-3 text-center">
                            {isComplete ? (
                              <CheckCircle2 className="h-4 w-4 text-emerald-600 mx-auto" />
                            ) : (
                              <div className="h-4 w-4 border-2 border-slate-300 rounded-sm mx-auto" />
                            )}
                          </td>
                          <td className="py-2.5 px-3">
                            <div className="font-bold text-slate-800 flex items-center gap-1.5 flex-wrap">
                              <span>{it.product_name}</span>
                              {it.is_free_item && (
                                <span className="text-[9px] font-bold px-1.5 py-0.2 rounded bg-emerald-100 text-emerald-800 border border-emerald-300 uppercase">
                                  BONUS
                                </span>
                              )}
                            </div>
                            <div className="font-mono text-[11px] text-slate-500">{it.product_sku}</div>
                          </td>
                          <td className="py-2.5 px-3">
                            <div className="font-mono text-slate-700">{it.batch_number || "-"}</div>
                            {it.expiry_date && (
                              <div className="text-[10px] text-slate-400">
                                Exp: {new Date(it.expiry_date).toLocaleDateString("id-ID")}
                              </div>
                            )}
                          </td>
                          <td className="py-2.5 px-3 text-right font-bold text-slate-800">{it.quantity}</td>
                          <td
                            className={`py-2.5 px-3 text-right font-mono font-bold ${
                              isComplete ? "text-emerald-700" : "text-indigo-600"
                            }`}
                          >
                            {it.packed_qty || 0}
                          </td>
                        </tr>
                      )
                    })}
                  </tbody>
                </table>
              </div>

              {/* ── Packaging Dimensions & Selesai Kemas ── */}
              <div className="bg-slate-50/80 p-4 rounded-xl border border-slate-200 space-y-4">
                <h5 className="text-xs font-bold text-slate-800 flex items-center gap-1.5">
                  <Scale className="h-4 w-4 text-slate-500" />
                  <span>Dimensi Paket & Kemasan Fisik</span>
                </h5>

                <div className="grid grid-cols-2 sm:grid-cols-5 gap-3 text-xs">
                  <div>
                    <label className="block text-slate-500 mb-1">Berat (kg)</label>
                    <input
                      type="number"
                      step="0.1"
                      value={weightKg}
                      onChange={(e) => setWeightKg(e.target.value)}
                      className="w-full p-2 bg-white border border-slate-200 rounded-lg text-slate-900 font-bold focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="block text-slate-500 mb-1">Panjang (cm)</label>
                    <input
                      type="number"
                      value={lengthCm}
                      onChange={(e) => setLengthCm(e.target.value)}
                      className="w-full p-2 bg-white border border-slate-200 rounded-lg text-slate-900 font-bold focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="block text-slate-500 mb-1">Lebar (cm)</label>
                    <input
                      type="number"
                      value={widthCm}
                      onChange={(e) => setWidthCm(e.target.value)}
                      className="w-full p-2 bg-white border border-slate-200 rounded-lg text-slate-900 font-bold focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="block text-slate-500 mb-1">Tinggi (cm)</label>
                    <input
                      type="number"
                      value={heightCm}
                      onChange={(e) => setHeightCm(e.target.value)}
                      className="w-full p-2 bg-white border border-slate-200 rounded-lg text-slate-900 font-bold focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                  <div>
                    <label className="block text-slate-500 mb-1">Jenis Kemasan</label>
                    <select
                      value={packagingType}
                      onChange={(e) => setPackagingType(e.target.value)}
                      className="w-full p-2 bg-white border border-slate-200 rounded-lg text-slate-900 font-bold focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    >
                      <option value="KARTON">Karton Box</option>
                      <option value="PLASTIK">Polymailer</option>
                      <option value="KAYU">Peti Kayu</option>
                      <option value="BUBBLE">Bubble Wrap</option>
                    </select>
                  </div>
                </div>

                <div className="pt-2 flex justify-end gap-2">
                  <button
                    type="button"
                    onClick={handleCompletePack}
                    disabled={!isAllPacked || completing}
                    className="inline-flex items-center gap-2 px-5 py-2.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg text-xs font-bold transition shadow-2xs disabled:opacity-40 disabled:cursor-not-allowed"
                  >
                    <CheckCircle2 className="h-4 w-4" />
                    <span>{completing ? "Menyelesaikan..." : "Selesaikan Kemasan (100% PACKED)"}</span>
                  </button>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* ── Modal Cetak Label Thermal Resi AWB (FE-08) ── */}
      {showThermalModal && selectedOrder && (
        <PrintThermalAWB
          order={selectedOrder}
          items={orderItems}
          onClose={() => setShowThermalModal(false)}
        />
      )}
    </div>
  )
}
