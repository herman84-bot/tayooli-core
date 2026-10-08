"use client"

// FE-07: Pack Station Verification UI with 100% barcode scan matching & Web Audio beeper.
// Follows Sentry-WMS §1.3 & OCA §1.3 packaging protocols.

import React, { useState, useEffect, useRef } from "react"
import {
  Package,
  Barcode,
  CheckCircle2,
  AlertCircle,
  X,
  Volume2,
  VolumeX,
  Scale,
  Box,
  Truck,
  Sparkles,
  RefreshCw,
} from "lucide-react"
import { api, DeliveryOrder, DeliveryOrderItem } from "@/lib/api"

interface PackStationModalProps {
  order: DeliveryOrder
  initialItems: DeliveryOrderItem[]
  onClose: () => void
  onSuccess: (updatedOrder: DeliveryOrder) => void
}

export function PackStationModal({
  order,
  initialItems,
  onClose,
  onSuccess,
}: PackStationModalProps) {
  const [items, setItems] = useState<DeliveryOrderItem[]>(initialItems)
  const [barcodeInput, setBarcodeInput] = useState("")
  const [loadingScan, setLoadingScan] = useState(false)
  const [completing, setCompleting] = useState(false)
  const [audioEnabled, setAudioEnabled] = useState(true)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [lastScannedName, setLastScannedName] = useState<string | null>(null)

  // Package measurements state
  const [weightKg, setWeightKg] = useState<string>(
    order.package_weight_kg ? String(order.package_weight_kg) : "1.0"
  )
  const [lengthCm, setLengthCm] = useState<string>(
    order.package_length_cm ? String(order.package_length_cm) : "30"
  )
  const [widthCm, setWidthCm] = useState<string>(
    order.package_width_cm ? String(order.package_width_cm) : "20"
  )
  const [heightCm, setHeightCm] = useState<string>(
    order.package_height_cm ? String(order.package_height_cm) : "15"
  )
  const [packagingType, setPackagingType] = useState<string>(
    order.packaging_type || "KARTON"
  )

  const barcodeInputRef = useRef<HTMLInputElement>(null)
  const audioCtxRef = useRef<AudioContext | null>(null)

  // Web Audio Beeper
  const playSound = (isSuccess: boolean) => {
    if (!audioEnabled || typeof window === "undefined") return
    try {
      if (!audioCtxRef.current) {
        const AudioCtx = window.AudioContext || (window as any).webkitAudioContext
        if (AudioCtx) {
          audioCtxRef.current = new AudioCtx()
        }
      }
      const ctx = audioCtxRef.current
      if (!ctx) return
      if (ctx.state === "suspended") {
        ctx.resume()
      }

      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.connect(gain)
      gain.connect(ctx.destination)

      const now = ctx.currentTime
      if (isSuccess) {
        // High pitch pleasant sine beep (880 Hz -> 1320 Hz)
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

  // Auto focus input on mount & reset error timer
  useEffect(() => {
    barcodeInputRef.current?.focus()
  }, [])

  // Calculate scan progress
  const totalQty = items.reduce((s, it) => s + Number(it.quantity || 0), 0)
  const packedQty = items.reduce((s, it) => s + Number(it.packed_qty || 0), 0)
  const progressPercent = totalQty > 0 ? Math.min(100, Math.round((packedQty / totalQty) * 100)) : 0
  const isAllPacked = totalQty > 0 && packedQty >= totalQty

  const handleScanSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const code = barcodeInput.trim()
    if (!code || loadingScan) return

    setLoadingScan(true)
    setErrorMessage(null)

    try {
      const res = await api.wms.deliveryOrders.scanPackItem(order.id, code, 1)
      const data = res.data

      // Update local state items
      setItems((prev) =>
        prev.map((it) =>
          it.id === data.item_id
            ? { ...it, packed_qty: Number(data.packed_qty) }
            : it
        )
      )

      setLastScannedName(`${data.product_name} (${data.product_sku})`)
      playSound(true)
      setBarcodeInput("")
    } catch (err: any) {
      playSound(false)
      setErrorMessage(
        err.message ||
          "Barcode tidak cocok dengan pesanan ini atau item sudah lengkap dipindai!"
      )
    } finally {
      setLoadingScan(false)
      setTimeout(() => {
        barcodeInputRef.current?.focus()
      }, 50)
    }
  }

  const handleCompletePacking = async () => {
    if (!isAllPacked || completing) return
    setCompleting(true)
    setErrorMessage(null)

    try {
      const res = await api.wms.deliveryOrders.completePack(order.id, {
        package_weight_kg: Number(weightKg) || undefined,
        package_length_cm: Number(lengthCm) || undefined,
        package_width_cm: Number(widthCm) || undefined,
        package_height_cm: Number(heightCm) || undefined,
        packaging_type: packagingType,
      })
      onSuccess(res.data)
    } catch (err: any) {
      setErrorMessage(err.message || "Gagal menyelesaikan pengemasan pesanan")
      setCompleting(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto">
      <div className="bg-white text-zinc-900 rounded-2xl shadow-2xl max-w-4xl w-full p-6 space-y-6">
        {/* Header */}
        <div className="flex items-center justify-between border-b pb-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-xl bg-indigo-50 border border-indigo-200 flex items-center justify-center text-indigo-600">
              <Package className="w-5 h-5" />
            </div>
            <div>
              <h2 className="text-xl font-bold text-zinc-900 flex items-center gap-2">
                Stasiun Meja Kemas (Pack Station)
                <span className="text-xs px-2 py-0.5 rounded-full bg-indigo-100 text-indigo-700 font-mono">
                  {order.do_number}
                </span>
              </h2>
              <p className="text-xs text-zinc-500">
                Pindai 100% barcode barang sebelum paket disegel dan diserahkan ke ekspedisi.
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setAudioEnabled(!audioEnabled)}
              className={`p-2 rounded-lg border transition ${
                audioEnabled
                  ? "bg-emerald-50 border-emerald-300 text-emerald-700"
                  : "bg-zinc-100 border-zinc-200 text-zinc-400"
              }`}
              title={audioEnabled ? "Suara bip aktif" : "Suara bip nonaktif"}
            >
              {audioEnabled ? <Volume2 className="w-4 h-4" /> : <VolumeX className="w-4 h-4" />}
            </button>
            <button
              onClick={onClose}
              className="p-2 text-zinc-400 hover:text-zinc-600 rounded-lg hover:bg-zinc-100 transition"
              aria-label="Tutup"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Scan Progress Bar */}
        <div className="space-y-2 bg-zinc-50 p-4 rounded-xl border border-zinc-200">
          <div className="flex justify-between items-center text-xs font-semibold">
            <span className="text-zinc-700 flex items-center gap-1.5">
              <Sparkles className="w-4 h-4 text-amber-500" />
              Kemajuan Pemindaian Barang Keluar
            </span>
            <span
              className={`font-mono text-sm font-bold ${
                isAllPacked ? "text-emerald-600" : "text-indigo-600"
              }`}
            >
              {packedQty} / {totalQty} Unit ({progressPercent}%)
            </span>
          </div>
          <div className="w-full bg-zinc-200 rounded-full h-3 overflow-hidden">
            <div
              className={`h-full transition-all duration-300 rounded-full ${
                isAllPacked
                  ? "bg-emerald-500"
                  : progressPercent > 50
                  ? "bg-indigo-600"
                  : "bg-amber-500"
              }`}
              style={{ width: `${progressPercent}%` }}
            />
          </div>
        </div>

        {/* Barcode Scanner Input Form */}
        <form onSubmit={handleScanSubmit} className="space-y-2">
          <label className="block text-xs font-bold text-zinc-700 uppercase tracking-wide">
            Pindai Barcode / SKU / Batch
          </label>
          <div className="flex gap-2">
            <div className="relative flex-1">
              <Barcode className="w-5 h-5 text-zinc-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
              <input
                ref={barcodeInputRef}
                type="text"
                value={barcodeInput}
                onChange={(e) => setBarcodeInput(e.target.value)}
                placeholder="Arahkan scanner ke barcode atau ketik SKU/Batch..."
                className="w-full pl-11 pr-4 py-3 rounded-xl border-2 border-indigo-300 focus:border-indigo-600 focus:outline-none font-mono text-sm bg-white"
                autoFocus
              />
            </div>
            <button
              type="submit"
              disabled={loadingScan || !barcodeInput.trim()}
              className="px-6 py-3 bg-indigo-600 hover:bg-indigo-700 disabled:opacity-50 text-white rounded-xl font-bold text-sm transition shadow-sm"
            >
              {loadingScan ? "Memproses..." : "Pindai"}
            </button>
          </div>

          {/* Feedback messages */}
          {errorMessage && (
            <div className="flex items-center gap-2 p-3 bg-rose-50 border border-rose-200 text-rose-800 rounded-xl text-xs font-medium animate-shake">
              <AlertCircle className="w-4 h-4 shrink-0 text-rose-600" />
              <span>{errorMessage}</span>
            </div>
          )}
          {lastScannedName && !errorMessage && (
            <div className="flex items-center gap-2 p-2.5 bg-emerald-50 border border-emerald-200 text-emerald-800 rounded-xl text-xs font-medium">
              <CheckCircle2 className="w-4 h-4 shrink-0 text-emerald-600" />
              <span>
                Berhasil diverifikasi: <strong>{lastScannedName}</strong>
              </span>
            </div>
          )}
        </form>

        {/* Items Verification Checklist */}
        <div className="border border-zinc-200 rounded-xl overflow-hidden">
          <div className="bg-zinc-100 px-4 py-2.5 text-xs font-bold text-zinc-700 uppercase tracking-wider flex justify-between">
            <span>Daftar Barang Pesanan ({items.length} Item)</span>
            <span>Status Verifikasi</span>
          </div>
          <div className="divide-y divide-zinc-200 max-h-56 overflow-y-auto">
            {items.map((it) => {
              const packed = Number(it.packed_qty || 0)
              const reqQty = Number(it.quantity || 0)
              const complete = packed >= reqQty

              return (
                <div
                  key={it.id}
                  className={`p-3.5 flex items-center justify-between transition ${
                    complete ? "bg-emerald-50/50" : "bg-white"
                  }`}
                >
                  <div className="space-y-0.5">
                    <div className="flex items-center gap-2">
                      <span className="font-bold text-sm text-zinc-900">{it.product_name}</span>
                      {it.is_free_item && (
                        <span className="px-2 py-0.5 rounded text-[10px] font-black bg-emerald-600 text-white tracking-wider">
                          BONUS
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-3 text-xs text-zinc-500 font-mono">
                      <span>SKU: {it.product_sku}</span>
                      {it.batch_number && <span>Batch: {it.batch_number}</span>}
                      {it.location_code && <span>Rak: {it.location_code}</span>}
                    </div>
                  </div>

                  <div className="flex items-center gap-4 text-right">
                    <div>
                      <span
                        className={`text-sm font-black font-mono ${
                          complete ? "text-emerald-700" : "text-amber-600"
                        }`}
                      >
                        {packed} / {reqQty}
                      </span>
                      <span className="text-xs text-zinc-400 block">terpindai</span>
                    </div>

                    <div
                      className={`w-8 h-8 rounded-full flex items-center justify-center ${
                        complete
                          ? "bg-emerald-100 text-emerald-700"
                          : "bg-amber-100 text-amber-700"
                      }`}
                    >
                      {complete ? (
                        <CheckCircle2 className="w-5 h-5" />
                      ) : (
                        <Box className="w-4 h-4" />
                      )}
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        </div>

        {/* Package Specifications */}
        <div className="bg-zinc-50 p-4 rounded-xl border border-zinc-200 space-y-3">
          <div className="flex items-center gap-2 text-xs font-bold text-zinc-800 uppercase tracking-wide">
            <Scale className="w-4 h-4 text-indigo-600" />
            Spesifikasi & Dimensi Paket Ekspedisi
          </div>
          <div className="grid grid-cols-2 sm:grid-cols-5 gap-3 text-xs">
            <div>
              <label className="block text-zinc-500 font-medium mb-1">Berat (kg)</label>
              <input
                type="number"
                step="0.1"
                min="0"
                value={weightKg}
                onChange={(e) => setWeightKg(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-zinc-300 font-mono text-zinc-800"
              />
            </div>
            <div>
              <label className="block text-zinc-500 font-medium mb-1">Panjang (cm)</label>
              <input
                type="number"
                min="0"
                value={lengthCm}
                onChange={(e) => setLengthCm(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-zinc-300 font-mono text-zinc-800"
              />
            </div>
            <div>
              <label className="block text-zinc-500 font-medium mb-1">Lebar (cm)</label>
              <input
                type="number"
                min="0"
                value={widthCm}
                onChange={(e) => setWidthCm(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-zinc-300 font-mono text-zinc-800"
              />
            </div>
            <div>
              <label className="block text-zinc-500 font-medium mb-1">Tinggi (cm)</label>
              <input
                type="number"
                min="0"
                value={heightCm}
                onChange={(e) => setHeightCm(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-zinc-300 font-mono text-zinc-800"
              />
            </div>
            <div>
              <label className="block text-zinc-500 font-medium mb-1">Tipe Kemasan</label>
              <select
                value={packagingType}
                onChange={(e) => setPackagingType(e.target.value)}
                className="w-full px-3 py-2 rounded-lg border border-zinc-300 text-zinc-800 font-medium"
              >
                <option value="KARTON">Karton / Kardus</option>
                <option value="PLASTIK_POLYMAILER">Plastik Polymailer</option>
                <option value="BUBBLE_WRAP">Bubble Wrap</option>
                <option value="KARUNG">Karung / Goni</option>
                <option value="KAYU">Peti Kayu</option>
              </select>
            </div>
          </div>
        </div>

        {/* Footer Actions */}
        <div className="flex items-center justify-between border-t pt-4">
          <div className="text-xs text-zinc-500">
            {!isAllPacked ? (
              <span className="text-amber-600 font-medium flex items-center gap-1.5">
                <AlertCircle className="w-4 h-4" />
                Tombol segel terkunci: selesaikan pemindaian 100% item terlebih dahulu.
              </span>
            ) : (
              <span className="text-emerald-600 font-bold flex items-center gap-1.5">
                <CheckCircle2 className="w-4 h-4" />
                Semua item lengkap! Paket siap disegel dan dilabeli.
              </span>
            )}
          </div>

          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 border border-zinc-300 hover:bg-zinc-100 rounded-xl text-zinc-700 text-sm font-semibold transition"
            >
              Tutup Sementara
            </button>
            <button
              type="button"
              onClick={handleCompletePacking}
              disabled={!isAllPacked || completing}
              className="flex items-center gap-2 px-6 py-2.5 bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 disabled:cursor-not-allowed text-white rounded-xl text-sm font-bold transition shadow-sm"
            >
              {completing ? (
                <>
                  <RefreshCw className="w-4 h-4 animate-spin" />
                  Menyegel Paket...
                </>
              ) : (
                <>
                  <CheckCircle2 className="w-4 h-4" />
                  Selesaikan Pengemasan (PACKED)
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
