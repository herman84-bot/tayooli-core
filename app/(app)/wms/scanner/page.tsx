"use client"

import React, { useState, useEffect, useRef, useCallback, useMemo } from "react"
import Link from "next/link"
import {
  ScanLine,
  Camera,
  CameraOff,
  Zap,
  ZapOff,
  Search,
  CheckCircle2,
  AlertCircle,
  Package,
  Layers,
  Tag,
  ArrowRight,
  RefreshCw,
  ArrowLeft,
  Volume2,
  VolumeX,
  Keyboard,
  Barcode as BarcodeIcon,
  Store,
  Box,
  Truck,
  FileCheck,
  Check,
  XCircle,
} from "lucide-react"
import { useBarcodeScanner, playScannerTone } from "@/hooks/useBarcodeScanner"
import { useCameraBarcodeDecoder } from "@/hooks/useCameraBarcodeDecoder"
import {
  useResolveBarcode,
  useWarehouseLocations,
  useConfirmPutaway,
  usePutawayPending,
  useDeliveryOrders,
} from "@/hooks/useWMS"
import {
  useShippingManifests,
  useShippingManifestDetail,
  useScanLoadingDO,
} from "@/hooks/useWMSManifests"
import {
  useStockLPNs,
  useStockLPNDetail,
  useMoveLPN,
} from "@/hooks/useWMSDocksAndLPNs"
import { api, ResolvedProduct } from "@/lib/api"

type ScannerMode = "PUTAWAY" | "OUTBOUND" | "PRICE_CHECK" | "LOADING_TRUCK" | "PALLET_LPN"

export default function BarcodeScannerPage() {
  const [mode, setMode] = useState<ScannerMode>("PUTAWAY")

  // Camera stream state
  const videoRef = useRef<HTMLVideoElement | null>(null)
  const [cameraActive, setCameraActive] = useState(false)
  const [cameraError, setCameraError] = useState<string | null>(null)
  const [hasTorch, setHasTorch] = useState(false)
  const [torchOn, setTorchOn] = useState(false)
  const [soundEnabled, setSoundEnabled] = useState(true)

  // Scanning input & buffer
  const [manualCode, setManualCode] = useState("")
  const [activeCode, setActiveCode] = useState<string>("")
  const [lastScannedCode, setLastScannedCode] = useState<string | null>(null)

  // Putaway 2-step state: Scan Rak -> Scan Barang
  const [putawayRack, setPutawayRack] = useState<string | null>(null)
  const [putawaySuccessMessage, setPutawaySuccessMessage] = useState<string | null>(null)

  const [putawayRackId, setPutawayRackId] = useState<string | null>(null)
  const [putawayWarehouseId, setPutawayWarehouseId] = useState<string | null>(null)
  const [putawayError, setPutawayError] = useState<string | null>(null)
  const [putawaySubmitting, setPutawaySubmitting] = useState(false)
  const confirmPutaway = useConfirmPutaway()
  const { data: putawayPending = [] } = usePutawayPending(putawayWarehouseId)

  // Outbound verification state (real Surat Jalan pack-scan)
  const [outboundDoId, setOutboundDoId] = useState<string | null>(null)
  const [outboundResult, setOutboundResult] = useState<{
    success: boolean
    message: string
    progress?: string
  } | null>(null)
  const { data: allDeliveryOrders = [] } = useDeliveryOrders()
  const outboundOrders = useMemo(
    () => allDeliveryOrders.filter((o) => o.status === "CONFIRMED" || o.status === "PICKED"),
    [allDeliveryOrders]
  )

  // State for LOADING_TRUCK mode
  const [selectedManifestId, setSelectedManifestId] = useState<string | null>(null)
  const [loadingScanResult, setLoadingScanResult] = useState<{
    success: boolean
    message: string
    scannedDoNumber?: string
  } | null>(null)

  // Manifests queries and mutations for LOADING_TRUCK mode
  const { data: allManifests = [] } = useShippingManifests()
  const activeManifests = useMemo(
    () => allManifests.filter((m) => m.status === "STAGED" || m.status === "LOADED"),
    [allManifests]
  )
  const { data: manifestDetail } = useShippingManifestDetail(selectedManifestId || undefined)
  const scanLoadingMutation = useScanLoadingDO()

  // State for PALLET_LPN mode
  const [scannedLPNCode, setScannedLPNCode] = useState<string | null>(null)
  const [matchedLPNId, setMatchedLPNId] = useState<string | null>(null)
  const [targetRackLocation, setTargetRackLocation] = useState<string | null>(null)
  const [targetRackLocationId, setTargetRackLocationId] = useState<string | null>(null)
  const [lpnPutawaySuccessMessage, setLpnPutawaySuccessMessage] = useState<string | null>(null)
  const [lpnPutawayErrorMessage, setLpnPutawayErrorMessage] = useState<string | null>(null)

  // Query product resolution
  const { data: resolvedProduct, isFetching: resolving, isError: resolveError } = useResolveBarcode(
    activeCode || null
  )

  // Available locations for suggestion and LPN queries
  const { data: locations = [] } = useWarehouseLocations()
  const activeWarehouseId = locations[0]?.warehouse_id || null

  // Queries and mutations for PALLET_LPN mode
  const { data: stockLPNs = [] } = useStockLPNs(activeWarehouseId)
  const { data: lpnDetail, isLoading: lpnDetailLoading } = useStockLPNDetail(matchedLPNId)
  const moveLPNMutation = useMoveLPN()

  // Putaway step 2: resolve product, find its pending staging line, confirm putaway.
  const submitPutaway = useCallback(
    async (code: string) => {
      if (!putawayRackId || !putawayWarehouseId) return
      setPutawaySubmitting(true)
      setPutawayError(null)
      try {
        let product: ResolvedProduct
        try {
          product = await api.wms.barcodes.resolve(code)
        } catch {
          throw new Error(`Barcode [${code}] tidak terdaftar di master produk.`)
        }
        const line = putawayPending.find((p) => p.product_id === product.product_id)
        if (!line) {
          throw new Error(
            `${product.name} tidak memiliki stok di area staging inbound gudang ini (tidak ada yang perlu di-putaway).`
          )
        }
        if (line.suggested_location_id && line.suggested_location_id !== putawayRackId) {
          throw new Error(
            `Rak ${putawayRack} bukan rak tujuan untuk ${product.name}. Rak yang disarankan: ${
              line.suggested_location_code ?? line.suggested_location_id
            }. Gunakan menu Putaway untuk override dengan alasan.`
          )
        }
        await confirmPutaway.mutateAsync({
          warehouse_id: putawayWarehouseId,
          product_id: line.product_id,
          batch_id: line.batch_id,
          quantity: Number(line.quantity),
          dest_location_id: putawayRackId,
        })
        playScannerTone("success")
        setPutawaySuccessMessage(
          `BERHASIL: ${line.quantity} ${product.name} (batch ${line.batch_number}) dipindahkan ke ${putawayRack}.`
        )
      } catch (err) {
        playScannerTone("error")
        setPutawaySuccessMessage(null)
        setPutawayError(err instanceof Error ? err.message : "Putaway gagal diproses server.")
      } finally {
        setPutawaySubmitting(false)
      }
    },
    [putawayRackId, putawayWarehouseId, putawayRack, putawayPending, confirmPutaway]
  )

  // Outbound: server-side validation of scanned item against the selected Surat Jalan.
  const submitOutboundScan = useCallback(
    async (code: string) => {
      if (!outboundDoId) return
      try {
        const res = await api.wms.deliveryOrders.scanPackItem(outboundDoId, code, 1)
        const d = res.data
        playScannerTone("success")
        setOutboundResult({
          success: true,
          message: `${d.product_name} (${d.product_sku}) cocok dengan Surat Jalan.`,
          progress: `${d.packed_qty}/${d.requested_qty} unit item ini · ${d.packed_items}/${d.total_items} item lengkap${
            d.order_completed ? " · SURAT JALAN LENGKAP" : ""
          }`,
        })
      } catch (err) {
        playScannerTone("error")
        setOutboundResult({
          success: false,
          message:
            err instanceof Error && err.message
              ? `Barang [${code}] ditolak: ${err.message}`
              : `Barang [${code}] tidak sesuai dengan Surat Jalan ini.`,
        })
      }
    },
    [outboundDoId]
  )

  // Hardware Scanner Integration
  const handleBarcodeDetected = useCallback(
    (code: string) => {
      setLastScannedCode(code)
      setActiveCode(code)

      // Pallet LPN Putaway Mode Logic
      if (mode === "PALLET_LPN") {
        // Step 1: Scan Palet LPN
        if (!scannedLPNCode) {
          const trimmedCode = code.trim()
          const foundLPN = stockLPNs.find(
            (l) =>
              l.lpn_code.toUpperCase() === trimmedCode.toUpperCase() ||
              l.id.toUpperCase() === trimmedCode.toUpperCase()
          )

          if (foundLPN) {
            setScannedLPNCode(foundLPN.lpn_code)
            setMatchedLPNId(foundLPN.id)
            setLpnPutawayErrorMessage(null)
            setLpnPutawaySuccessMessage(`Palet [${foundLPN.lpn_code}] terpilih! Silakan scan barcode Rak Tujuan.`)
            playTone("success")
          } else {
            playTone("error")
            setLpnPutawayErrorMessage(
              trimmedCode.toUpperCase().startsWith("LPN-")
                ? `Palet [${trimmedCode.toUpperCase()}] tidak ditemukan di gudang ini atau belum terdaftar!`
                : `Barcode [${trimmedCode}] bukan palet LPN valid! (Gunakan format LPN-XXXX)`
            )
            setLpnPutawaySuccessMessage(null)
          }
          return
        }

        // Step 2: Scan Rak Tujuan & Pindahkan Seluruh Isi Palet
        const trimmedLocCode = code.trim().toUpperCase()
        const loc = locations.find(
          (l) =>
            l.code.toUpperCase() === trimmedLocCode ||
            l.id.toUpperCase() === trimmedLocCode ||
            (l.barcode && l.barcode.toUpperCase() === trimmedLocCode)
        )

        if (!loc) {
          playTone("error")
          setLpnPutawayErrorMessage(`Lokasi rak [${code.trim()}] tidak valid atau tidak terdaftar!`)
          return
        }

        setTargetRackLocation(loc.code)
        setTargetRackLocationId(loc.id)

        moveLPNMutation.mutate(
          {
            id: matchedLPNId!,
            data: { target_location_id: loc.id },
          },
          {
            onSuccess: () => {
              playTone("success")
              setLpnPutawaySuccessMessage(
                `BERHASIL: Seluruh isi palet [${scannedLPNCode}] berhasil dipindahkan ke rak [${loc.code}]!`
              )
              setLpnPutawayErrorMessage(null)
              setScannedLPNCode(null)
              setMatchedLPNId(null)
              setTargetRackLocation(null)
              setTargetRackLocationId(null)
            },
            onError: (err: unknown) => {
              playTone("error")
              const errMsg =
                err instanceof Error && err.message
                  ? err.message
                  : `Gagal memindahkan palet [${scannedLPNCode}] ke rak [${loc.code}]`
              setLpnPutawayErrorMessage(errMsg)
            },
          }
        )
        return
      }

      // Truck Loading Mode Logic
      if (mode === "LOADING_TRUCK") {
        if (!selectedManifestId) {
          playTone("error")
          setLoadingScanResult({
            success: false,
            message: "Pilih manifest tujuan terlebih dahulu",
          })
          return
        }

        scanLoadingMutation.mutate(
          { manifestId: selectedManifestId, barcode: code },
          {
            onSuccess: () => {
              playTone("success")
              setLoadingScanResult({
                success: true,
                message: `Surat Jalan [${code}] berhasil dimuat ke armada!`,
                scannedDoNumber: code,
              })
            },
            onError: (err: unknown) => {
              playTone("error")
              const errMsg =
                err instanceof Error && err.message
                  ? err.message
                  : `MISLOAD: Barcode tidak terdaftar dalam manifest ini!`
              setLoadingScanResult({
                success: false,
                message: errMsg,
                scannedDoNumber: code,
              })
            },
          }
        )
        return
      }

      // Inbound Putaway Mode: Step 1 scan a registered rack -> Step 2 scan product.
      // Step 2 posts POST /wms/putaway/confirm (staging -> rack) — real stock movement.
      if (mode === "PUTAWAY") {
        const trimmed = code.trim().toUpperCase()
        if (!putawayRack) {
          const loc = locations.find(
            (l) =>
              l.code.toUpperCase() === trimmed ||
              l.id.toUpperCase() === trimmed ||
              (l.barcode && l.barcode.toUpperCase() === trimmed)
          )
          if (!loc) {
            playTone("error")
            setPutawayError(`Lokasi rak [${code.trim()}] tidak terdaftar di master lokasi gudang.`)
            setPutawaySuccessMessage(null)
            return
          }
          setPutawayRack(loc.code)
          setPutawayRackId(loc.id)
          setPutawayWarehouseId(loc.warehouse_id ?? null)
          setPutawayError(null)
          setPutawaySuccessMessage(`Rak ${loc.code} terpilih. Silakan scan barang.`)
          return
        }
        if (!putawayRackId || !putawayWarehouseId || putawaySubmitting) return
        void submitPutaway(code.trim())
        return
      }

      // Outbound verification: POST /wms/delivery-orders/{id}/pack/scan validates the
      // scanned barcode against the selected Surat Jalan items on the server.
      if (mode === "OUTBOUND") {
        if (!outboundDoId) {
          playTone("error")
          setOutboundResult({ success: false, message: "Pilih Surat Jalan terlebih dahulu." })
          return
        }
        void submitOutboundScan(code.trim())
      }
    },
    [
      mode,
      scannedLPNCode,
      matchedLPNId,
      stockLPNs,
      locations,
      moveLPNMutation,
      selectedManifestId,
      scanLoadingMutation,
      putawayRack,
      putawayRackId,
      putawayWarehouseId,
      putawaySubmitting,
      submitPutaway,
      outboundDoId,
      submitOutboundScan,
    ]
  )

  const { triggerScan, playTone } = useBarcodeScanner({
    onScan: handleBarcodeDetected,
    soundFeedback: soundEnabled,
    hapticFeedback: true,
  })

  // Camera frames -> native BarcodeDetector -> same handler as USB/manual scans
  const { supported: cameraDecodeSupported } = useCameraBarcodeDecoder(videoRef, cameraActive, (code) =>
    triggerScan(code)
  )

  // Start Camera
  const startCamera = async () => {
    setCameraError(null)
    try {
      if (!navigator.mediaDevices || !navigator.mediaDevices.getUserMedia) {
        throw new Error("Kamera WebRTC tidak didukung pada browser ini.")
      }

      const stream = await navigator.mediaDevices.getUserMedia({
        video: {
          facingMode: { ideal: "environment" },
          width: { ideal: 1280 },
          height: { ideal: 720 },
        },
      })

      if (videoRef.current) {
        videoRef.current.srcObject = stream
        await videoRef.current.play()
      }

      setCameraActive(true)

      // Check torch capability
      const track = stream.getVideoTracks()[0]
      const capabilities = track?.getCapabilities?.() as { torch?: boolean } | undefined
      if (capabilities && "torch" in capabilities) {
        setHasTorch(true)
      }
    } catch (err: unknown) {
      setCameraActive(false)
      setCameraError(
        err instanceof Error
          ? err.message
          : "Izin kamera ditolak atau kamera tidak ditemukan."
      )
    }
  }

  // Stop Camera
  const stopCamera = () => {
    if (videoRef.current && videoRef.current.srcObject) {
      const stream = videoRef.current.srcObject as MediaStream
      stream.getTracks().forEach((track) => track.stop())
      videoRef.current.srcObject = null
    }
    setCameraActive(false)
    setTorchOn(false)
  }

  // Toggle Torch
  const toggleTorch = async () => {
    if (!videoRef.current || !videoRef.current.srcObject) return
    const stream = videoRef.current.srcObject as MediaStream
    const track = stream.getVideoTracks()[0]
    if (track) {
      try {
        const newTorchState = !torchOn
        await (track as unknown as { applyConstraints: (c: unknown) => Promise<void> }).applyConstraints({
          advanced: [{ torch: newTorchState }],
        })
        setTorchOn(newTorchState)
      } catch {
        // Torch not supported
      }
    }
  }

  // Cleanup camera on unmount
  useEffect(() => {
    return () => {
      stopCamera()
    }
  }, [])

  // Manual code submit
  const handleManualSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!manualCode.trim()) return
    triggerScan(manualCode.trim())
    setManualCode("")
  }

  // Only real server data is shown; unknown barcodes surface as "not found".
  const displayProduct: ResolvedProduct | null = resolvedProduct ?? null

  return (
    <div className="min-h-screen bg-[#F8FAFC] pb-16">
      {/* ── Top Header ── */}
      <div className="bg-white border-b border-[#E2E8F0] px-4 sm:px-6 py-4 sticky top-0 z-30 shadow-xs">
        <div className="max-w-4xl mx-auto flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <Link
              href="/wms"
              className="p-2.5 rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 min-h-[48px] min-w-[48px] flex items-center justify-center transition-colors"
              aria-label="Kembali ke WMS"
            >
              <ArrowLeft className="w-5 h-5" />
            </Link>
            <div>
              <div className="flex items-center gap-2">
                <ScanLine className="w-5 h-5 text-[#2563EB]" />
                <h1 className="text-lg sm:text-xl font-bold tracking-tight text-slate-900">
                  Barcode Scanner Mobile
                </h1>
              </div>
              <p className="text-xs text-slate-500">
                WebRTC Camera & USB Gun Auto-Detection (&le;30ms)
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setSoundEnabled((prev) => !prev)}
              className={`p-2.5 rounded-lg border min-h-[48px] min-w-[48px] flex items-center justify-center transition-colors ${
                soundEnabled
                  ? "border-blue-200 bg-blue-50 text-[#2563EB]"
                  : "border-slate-200 bg-slate-50 text-slate-400"
              }`}
              title={soundEnabled ? "Audio beep aktif" : "Audio mute"}
              aria-label="Toggle Sound"
            >
              {soundEnabled ? <Volume2 className="w-5 h-5" /> : <VolumeX className="w-5 h-5" />}
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-4xl mx-auto px-4 sm:px-6 pt-5 space-y-5">
        {/* ── Mode Switcher Pills (min 48px touch target) ── */}
        <div className="bg-white p-1.5 rounded-xl border border-[#E2E8F0] grid grid-cols-2 sm:grid-cols-3 md:grid-cols-5 gap-1 shadow-xs">
          <button
            type="button"
            onClick={() => {
              setMode("PUTAWAY")
              setPutawayRack(null)
              setPutawayRackId(null)
              setPutawayError(null)
              setPutawaySuccessMessage(null)
            }}
            className={`flex flex-col sm:flex-row items-center justify-center gap-1.5 py-3 px-2 rounded-lg font-semibold text-xs sm:text-sm min-h-[48px] transition-all ${
              mode === "PUTAWAY"
                ? "bg-[#2563EB] text-white shadow-xs"
                : "text-slate-600 hover:bg-slate-100"
            }`}
          >
            <Layers className="w-4 h-4 shrink-0" />
            <span className="truncate">Inbound Putaway</span>
          </button>

          <button
            type="button"
            onClick={() => {
              setMode("OUTBOUND")
              setOutboundResult(null)
            }}
            className={`flex flex-col sm:flex-row items-center justify-center gap-1.5 py-3 px-2 rounded-lg font-semibold text-xs sm:text-sm min-h-[48px] transition-all ${
              mode === "OUTBOUND"
                ? "bg-[#2563EB] text-white shadow-xs"
                : "text-slate-600 hover:bg-slate-100"
            }`}
          >
            <Package className="w-4 h-4 shrink-0" />
            <span className="truncate">Outbound Picking</span>
          </button>

          <button
            type="button"
            onClick={() => {
              setMode("LOADING_TRUCK")
              setLoadingScanResult(null)
            }}
            className={`flex flex-col sm:flex-row items-center justify-center gap-1.5 py-3 px-2 rounded-lg font-semibold text-xs sm:text-sm min-h-[48px] transition-all ${
              mode === "LOADING_TRUCK"
                ? "bg-[#2563EB] text-white shadow-xs"
                : "text-slate-600 hover:bg-slate-100"
            }`}
          >
            <Truck className="w-4 h-4 shrink-0" />
            <span className="truncate">Truck (Loading Truk)</span>
          </button>

          <button
            type="button"
            onClick={() => {
              setMode("PALLET_LPN")
              setLpnPutawaySuccessMessage(null)
              setLpnPutawayErrorMessage(null)
            }}
            className={`flex flex-col sm:flex-row items-center justify-center gap-1.5 py-3 px-2 rounded-lg font-semibold text-xs sm:text-sm min-h-[48px] transition-all ${
              mode === "PALLET_LPN"
                ? "bg-[#2563EB] text-white shadow-xs"
                : "text-slate-600 hover:bg-slate-100"
            }`}
          >
            <Box className="w-4 h-4 shrink-0" />
            <span className="truncate">Palet (LPN Putaway)</span>
          </button>

          <button
            type="button"
            onClick={() => setMode("PRICE_CHECK")}
            className={`flex flex-col sm:flex-row items-center justify-center gap-1.5 py-3 px-2 rounded-lg font-semibold text-xs sm:text-sm min-h-[48px] transition-all ${
              mode === "PRICE_CHECK"
                ? "bg-[#2563EB] text-white shadow-xs"
                : "text-slate-600 hover:bg-slate-100"
            }`}
          >
            <Tag className="w-4 h-4 shrink-0" />
            <span className="truncate">Cek SKU</span>
          </button>
        </div>

        {/* ── Mode Instruction Banner ── */}
        <div className="p-3.5 rounded-xl bg-[#EFF6FF] border border-[#BFDBFE] text-slate-800 text-xs sm:text-sm flex items-start sm:items-center justify-between gap-3">
          <div className="flex items-center gap-2.5">
            <span className="flex h-2.5 w-2.5 rounded-full bg-[#2563EB] animate-ping" />
            <div>
              {mode === "PUTAWAY" && (
                <span>
                  <strong>Alur Putaway:</strong> Langkah 1 Scan Barcode Rak Lokasi &rarr; Langkah 2 Scan
                  Barcode Produk.
                </span>
              )}
              {mode === "OUTBOUND" && (
                <span>
                  <strong>Verifikasi Keluar:</strong> Scan item untuk mencocokkan dengan Surat Jalan / Picking
                  Order.
                </span>
              )}
              {mode === "LOADING_TRUCK" && (
                <span>
                  <strong>Pemuatan Armada:</strong> Pilih nomor manifest pengiriman, kemudian scan barcode
                  Surat Jalan (DO) saat dimuat ke truk.
                </span>
              )}
              {mode === "PALLET_LPN" && (
                <span>
                  <strong>Putaway Palet LPN:</strong> Langkah 1 Scan Barcode Palet LPN &rarr; Langkah 2 Scan
                  Barcode Rak Lokasi Tujuan.
                </span>
              )}
              {mode === "PRICE_CHECK" && (
                <span>
                  <strong>Pemeriksaan Cepat:</strong> Scan barcode fisik atau ketik kode untuk melihat master
                  SKU dan packaging.
                </span>
              )}
            </div>
          </div>
          {mode === "PUTAWAY" && putawayRack && (
            <button
              onClick={() => {
                setPutawayRack(null)
                setPutawayRackId(null)
                setPutawayError(null)
                setPutawaySuccessMessage(null)
              }}
              className="text-xs font-bold text-rose-600 hover:underline min-h-[36px] flex items-center"
            >
              Reset Rak
            </button>
          )}
        </div>

        {/* ── Putaway 2-Step Progress Indicator ── */}
        {mode === "PUTAWAY" && (
          <div className="bg-white p-4 rounded-xl border border-[#E2E8F0] shadow-xs">
            <div className="grid grid-cols-2 gap-3 text-center">
              <div
                className={`p-3 rounded-lg border transition-all ${
                  putawayRack
                    ? "bg-emerald-50 border-emerald-300 text-emerald-800 font-bold"
                    : "bg-blue-50 border-[#2563EB] text-[#2563EB] font-bold ring-2 ring-blue-100"
                }`}
              >
                <div className="text-[11px] uppercase tracking-wider">Langkah 1: Rak Lokasi</div>
                <div className="text-sm font-mono mt-1">
                  {putawayRack ? `✓ ${putawayRack}` : "Menunggu Scan Rak..."}
                </div>
              </div>

              <div
                className={`p-3 rounded-lg border transition-all ${
                  !putawayRack
                    ? "bg-slate-50 border-slate-200 text-slate-400"
                    : putawaySuccessMessage?.includes("BERHASIL")
                    ? "bg-emerald-50 border-emerald-300 text-emerald-800 font-bold"
                    : "bg-blue-50 border-[#2563EB] text-[#2563EB] font-bold ring-2 ring-blue-100"
                }`}
              >
                <div className="text-[11px] uppercase tracking-wider">Langkah 2: Produk / Barang</div>
                <div className="text-sm font-mono mt-1">
                  {putawaySubmitting
                    ? "Memproses..."
                    : putawaySuccessMessage?.includes("BERHASIL")
                    ? `✓ ${lastScannedCode}`
                    : putawayRack
                    ? "Siap Scan Barang..."
                    : "Terkunci"}
                </div>
              </div>
            </div>

            {putawayError && (
              <div className="mt-3 p-3 rounded-lg bg-rose-50 border border-rose-200 text-rose-900 text-xs font-semibold flex items-center gap-2">
                <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
                <span>{putawayError}</span>
              </div>
            )}

            {putawaySuccessMessage && (
              <div className="mt-3 p-3 rounded-lg bg-emerald-50 border border-emerald-200 text-emerald-900 text-xs font-semibold flex items-center gap-2">
                <CheckCircle2 className="w-4 h-4 text-emerald-600 shrink-0" />
                <span>{putawaySuccessMessage}</span>
              </div>
            )}
          </div>
        )}

        {/* ── Outbound Verification Status Banner (Match / Mismatch) ── */}
        {mode === "OUTBOUND" && (
          <div className="bg-white p-4 rounded-xl border border-[#E2E8F0] shadow-xs space-y-3">
            <div>
              <label
                htmlFor="outbound-do-select"
                className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5"
              >
                Pilih Surat Jalan (CONFIRMED / PICKED)
              </label>
              <select
                id="outbound-do-select"
                value={outboundDoId || ""}
                onChange={(e) => {
                  setOutboundDoId(e.target.value || null)
                  setOutboundResult(null)
                }}
                className="w-full min-h-[48px] px-3 py-2 text-sm bg-slate-50 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-[#2563EB] font-medium"
              >
                <option value="">-- Pilih Surat Jalan --</option>
                {outboundOrders.map((o) => (
                  <option key={o.id} value={o.id}>
                    {o.do_number}
                    {o.customer_name ? ` — ${o.customer_name}` : ""} [{o.status}]
                  </option>
                ))}
              </select>
            </div>

            {outboundResult && (
              <div
                className={`p-4 rounded-xl border-2 flex items-center gap-3 ${
                  outboundResult.success
                    ? "bg-emerald-50 border-emerald-500 text-emerald-900"
                    : "bg-rose-50 border-rose-500 text-rose-900"
                }`}
              >
                {outboundResult.success ? (
                  <CheckCircle2 className="w-6 h-6 text-emerald-700 shrink-0" />
                ) : (
                  <AlertCircle className="w-6 h-6 text-rose-700 shrink-0" />
                )}
                <div>
                  <div className="text-base font-extrabold">
                    {outboundResult.success ? "COCOK" : "TIDAK COCOK"}
                  </div>
                  <div className="text-xs">{outboundResult.message}</div>
                  {outboundResult.progress && (
                    <div className="text-xs font-mono mt-0.5">{outboundResult.progress}</div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {/* ── Truck Loading Mode: Manifest Selector & Status Banner ── */}
        {mode === "LOADING_TRUCK" && (
          <div className="bg-white p-4 rounded-xl border border-[#E2E8F0] shadow-xs space-y-4">
            <div>
              <label
                htmlFor="manifest-select"
                className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-1.5"
              >
                Pilih Manifest Pengiriman (Armada / Truk)
              </label>
              <select
                id="manifest-select"
                value={selectedManifestId || ""}
                onChange={(e) => {
                  setSelectedManifestId(e.target.value || null)
                  setLoadingScanResult(null)
                }}
                className="w-full min-h-[48px] px-3 py-2 text-sm bg-slate-50 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-[#2563EB] font-medium"
              >
                <option value="">-- Pilih Manifest Aktif (STAGED / LOADED) --</option>
                {activeManifests.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.manifest_number} — {m.expedition_name} ({m.vehicle_plate}) — [{m.status}]
                  </option>
                ))}
              </select>
            </div>

            {/* Recent scan feedback banner */}
            {loadingScanResult && (
              <div
                className={`p-4 rounded-xl border-2 flex items-center gap-3 transition-all ${
                  loadingScanResult.success
                    ? "bg-emerald-50 border-emerald-500 text-emerald-900"
                    : "bg-rose-50 border-rose-500 text-rose-900"
                }`}
              >
                <div
                  className={`p-2 rounded-full ${
                    loadingScanResult.success
                      ? "bg-emerald-200 text-emerald-800"
                      : "bg-rose-200 text-rose-800"
                  }`}
                >
                  {loadingScanResult.success ? (
                    <CheckCircle2 className="w-6 h-6" />
                  ) : (
                    <AlertCircle className="w-6 h-6" />
                  )}
                </div>
                <div>
                  <div className="text-sm font-bold">
                    {loadingScanResult.success ? "BERHASIL DIMUAT" : "PERINGATAN MISLOAD"}
                  </div>
                  <div className="text-xs mt-0.5">{loadingScanResult.message}</div>
                </div>
              </div>
            )}

            {/* Selected Manifest Detail Cards & DO Checklist */}
            {manifestDetail && (
              <div className="space-y-4 pt-2 border-t border-slate-100">
                {/* Manifest Metadata */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-slate-50 p-3.5 rounded-lg border border-slate-200 text-xs">
                  <div>
                    <div className="text-slate-500">No. Manifest:</div>
                    <div className="font-mono font-bold text-slate-900 mt-0.5">
                      {manifestDetail.manifest.manifest_number}
                    </div>
                  </div>
                  <div>
                    <div className="text-slate-500">Ekspedisi:</div>
                    <div className="font-semibold text-slate-900 mt-0.5">
                      {manifestDetail.manifest.expedition_name}
                    </div>
                  </div>
                  <div>
                    <div className="text-slate-500">Sopir:</div>
                    <div className="font-semibold text-slate-900 mt-0.5">
                      {manifestDetail.manifest.driver_name}
                    </div>
                  </div>
                  <div>
                    <div className="text-slate-500">Armada:</div>
                    <div className="font-mono font-bold text-slate-900 mt-0.5">
                      {manifestDetail.manifest.vehicle_plate}
                    </div>
                  </div>
                </div>

                {/* Progress counter & Visual Progress Bar */}
                {(() => {
                  const items = manifestDetail.items || []
                  const total = items.length
                  const scannedCount = items.filter((it) => it.scanned).length
                  const pct = total > 0 ? Math.round((scannedCount / total) * 100) : 0

                  return (
                    <div className="space-y-1.5 bg-blue-50/50 p-3.5 rounded-lg border border-blue-100">
                      <div className="flex items-center justify-between text-xs font-semibold text-slate-700">
                        <span>Progress Pemuatan:</span>
                        <span className="font-mono text-sm font-bold text-blue-700">
                          {scannedCount} dari {total} Koli Termuat ({pct}%)
                        </span>
                      </div>
                      <div className="w-full bg-slate-200 rounded-full h-3 overflow-hidden">
                        <div
                          className="bg-blue-600 h-3 rounded-full transition-all duration-300 ease-out"
                          style={{ width: `${pct}%` }}
                        />
                      </div>
                    </div>
                  )
                })()}

                {/* List of DOs in manifest with badges */}
                <div className="space-y-2">
                  <div className="text-xs font-bold text-slate-700 uppercase tracking-wider">
                    Daftar Surat Jalan (DO) Dalam Manifest:
                  </div>
                  <div className="space-y-1.5 max-h-60 overflow-y-auto pr-1">
                    {(manifestDetail.items || []).map((item) => (
                      <div
                        key={item.delivery_order_id}
                        className={`flex items-center justify-between p-2.5 rounded-lg border text-xs transition-colors ${
                          item.scanned
                            ? "bg-emerald-50/60 border-emerald-200"
                            : "bg-white border-slate-200"
                        }`}
                      >
                        <div className="flex items-center gap-2">
                          {item.scanned ? (
                            <Check className="w-4 h-4 text-emerald-600 shrink-0" />
                          ) : (
                            <Box className="w-4 h-4 text-slate-400 shrink-0" />
                          )}
                          <div>
                            <span className="font-mono font-bold text-slate-800">
                              {item.do_number}
                            </span>
                            <span className="text-slate-500 ml-2">({item.customer_name})</span>
                          </div>
                        </div>

                        <div>
                          {item.scanned ? (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-bold bg-emerald-100 text-emerald-800">
                              Dimuat
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2 py-0.5 rounded text-[11px] font-medium bg-slate-100 text-slate-600">
                              Belum Dimuat
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            )}
          </div>
        )}

        {/* ── Pallet LPN Putaway Mode Section ── */}
        {mode === "PALLET_LPN" && (
          <div className="bg-white p-4 rounded-xl border border-[#E2E8F0] shadow-xs space-y-4">
            {/* Step Indicators */}
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              {/* Step 1 Indicator */}
              <div
                className={`p-3.5 rounded-xl border-2 transition-all ${
                  scannedLPNCode
                    ? "bg-emerald-50 border-emerald-500 text-emerald-900"
                    : "bg-blue-50 border-blue-500 text-blue-900"
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span
                      className={`flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold ${
                        scannedLPNCode
                          ? "bg-emerald-600 text-white"
                          : "bg-blue-600 text-white"
                      }`}
                    >
                      1
                    </span>
                    <span className="font-bold text-xs uppercase tracking-wider">
                      Langkah 1: Scan Barcode Palet LPN
                    </span>
                  </div>
                  <span
                    className={`inline-flex items-center gap-1 text-[11px] font-bold px-2 py-0.5 rounded ${
                      scannedLPNCode
                        ? "text-emerald-700 bg-emerald-100"
                        : "text-amber-800 bg-amber-100"
                    }`}
                  >
                    {scannedLPNCode ? (
                      <>
                        <Check className="w-3 h-3" />
                        Sudah Terpilih
                      </>
                    ) : (
                      "Belum Terpilih"
                    )}
                  </span>
                </div>
                <div className="text-xs mt-2">
                  {scannedLPNCode ? (
                    <div>
                      Palet terpilih:{" "}
                      <span className="font-mono font-bold text-emerald-900">
                        {scannedLPNCode}
                      </span>
                    </div>
                  ) : (
                    <div className="text-blue-700">
                      Scan barcode palet LPN (format LPN-XXXX) untuk memulai.
                    </div>
                  )}
                </div>
              </div>

              {/* Step 2 Indicator */}
              <div
                className={`p-3.5 rounded-xl border-2 transition-all ${
                  !scannedLPNCode
                    ? "bg-slate-50 border-slate-200 text-slate-400 opacity-70"
                    : targetRackLocation
                    ? "bg-emerald-50 border-emerald-500 text-emerald-900"
                    : "bg-amber-50 border-amber-500 text-amber-900"
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span
                      className={`flex items-center justify-center w-6 h-6 rounded-full text-xs font-bold ${
                        !scannedLPNCode
                          ? "bg-slate-300 text-slate-600"
                          : targetRackLocation
                          ? "bg-emerald-600 text-white"
                          : "bg-amber-600 text-white"
                      }`}
                    >
                      2
                    </span>
                    <span className="font-bold text-xs uppercase tracking-wider">
                      Langkah 2: Scan Barcode Rak Lokasi Tujuan
                    </span>
                  </div>
                  {targetRackLocation && (
                    <span className="inline-flex items-center gap-1 text-[11px] font-bold text-emerald-700 bg-emerald-100 px-2 py-0.5 rounded">
                      <Check className="w-3 h-3" />
                      Rak Diset
                    </span>
                  )}
                </div>
                <div className="text-xs mt-2">
                  {!scannedLPNCode ? (
                    <div>Selesaikan langkah 1 terlebih dahulu.</div>
                  ) : targetRackLocation ? (
                    <div>
                      Rak tujuan:{" "}
                      <span className="font-mono font-bold">
                        {targetRackLocation}
                      </span>
                    </div>
                  ) : (
                    <div className="text-amber-800 font-medium">
                      Arahkan forklift &amp; scan barcode rak lokasi tujuan penyimpanan.
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Banners for feedback */}
            {lpnPutawaySuccessMessage && (
              <div className="p-4 rounded-xl border-2 bg-emerald-50 border-emerald-500 text-emerald-900 flex items-center gap-3 transition-all">
                <div className="p-2 rounded-full bg-emerald-200 text-emerald-800 shrink-0">
                  <CheckCircle2 className="w-6 h-6" />
                </div>
                <div>
                  <div className="text-sm font-bold">PUTAWAY PALET BERHASIL</div>
                  <div className="text-xs mt-0.5">{lpnPutawaySuccessMessage}</div>
                </div>
              </div>
            )}

            {lpnPutawayErrorMessage && (
              <div className="p-4 rounded-xl border-2 bg-rose-50 border-rose-500 text-rose-900 flex items-center gap-3 transition-all">
                <div className="p-2 rounded-full bg-rose-200 text-rose-800 shrink-0">
                  <AlertCircle className="w-6 h-6" />
                </div>
                <div>
                  <div className="text-sm font-bold">PERINGATAN SCAN PALET</div>
                  <div className="text-xs mt-0.5">{lpnPutawayErrorMessage}</div>
                </div>
              </div>
            )}

            {/* Selected Pallet Details & Items List */}
            {matchedLPNId && (
              <div className="bg-slate-50 p-4 rounded-xl border border-slate-200 space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Box className="w-5 h-5 text-blue-600" />
                    <span className="font-bold text-slate-800 text-sm">
                      Informasi Palet Terpilih
                    </span>
                  </div>
                  <button
                    type="button"
                    onClick={() => {
                      setScannedLPNCode(null)
                      setMatchedLPNId(null)
                      setTargetRackLocation(null)
                      setTargetRackLocationId(null)
                      setLpnPutawaySuccessMessage(null)
                      setLpnPutawayErrorMessage(null)
                    }}
                    className="px-3 py-1.5 text-xs font-semibold rounded-lg border border-slate-300 text-slate-700 bg-white hover:bg-slate-100 transition-colors shadow-2xs"
                  >
                    Ganti Palet
                  </button>
                </div>

                {/* Pallet Badge: Kode LPN, Tipe Palet, Lokasi Saat Ini, Total Berat kg */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-white p-3.5 rounded-lg border border-slate-200 text-xs">
                  <div>
                    <div className="text-slate-500">Kode LPN:</div>
                    <div className="font-mono font-bold text-slate-900 mt-0.5">
                      {lpnDetail?.lpn?.lpn_code || scannedLPNCode}
                    </div>
                  </div>
                  <div>
                    <div className="text-slate-500">Tipe Palet:</div>
                    <div className="font-semibold text-slate-900 mt-0.5">
                      {lpnDetail?.lpn?.pallet_type || "STANDARD"}
                    </div>
                  </div>
                  <div>
                    <div className="text-slate-500">Lokasi Saat Ini:</div>
                    <div className="font-mono font-bold text-slate-900 mt-0.5">
                      {lpnDetail?.lpn?.location_code || lpnDetail?.lpn?.location_name || "-"}
                    </div>
                  </div>
                  <div>
                    <div className="text-slate-500">Total Berat:</div>
                    <div className="font-bold text-slate-900 mt-0.5">
                      {lpnDetail?.lpn?.total_weight_kg ?? 0} kg
                    </div>
                  </div>
                </div>

                {/* Table/List ringkas isi koli di dalam palet */}
                <div>
                  <div className="text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                    Daftar Koli Di Dalam Palet:
                  </div>
                  {lpnDetailLoading ? (
                    <div className="text-xs text-slate-500 py-3 text-center">
                      Memuat detail koli palet...
                    </div>
                  ) : !lpnDetail?.items || lpnDetail.items.length === 0 ? (
                    <div className="text-xs text-slate-400 py-3 text-center italic bg-white rounded-lg border border-slate-200">
                      Palet ini kosong (belum ada item yang dikemas).
                    </div>
                  ) : (
                    <div className="overflow-x-auto rounded-lg border border-slate-200 bg-white">
                      <table className="w-full text-xs text-left">
                        <thead className="bg-slate-50 border-b border-slate-200 text-slate-600 font-semibold">
                          <tr>
                            <th className="py-2.5 px-3 w-10">No</th>
                            <th className="py-2.5 px-3">SKU</th>
                            <th className="py-2.5 px-3">Nama Barang</th>
                            <th className="py-2.5 px-3">Batch</th>
                            <th className="py-2.5 px-3 text-right">Jumlah</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                          {lpnDetail.items.map((item, idx) => (
                            <tr key={item.id || idx} className="hover:bg-slate-50">
                              <td className="py-2.5 px-3 text-slate-500">{idx + 1}</td>
                              <td className="py-2.5 px-3 font-mono font-bold text-slate-800">
                                {item.product_sku || "-"}
                              </td>
                              <td className="py-2.5 px-3 text-slate-800">
                                {item.product_name || "-"}
                              </td>
                              <td className="py-2.5 px-3 font-mono text-slate-600">
                                {item.batch_number || "-"}
                              </td>
                              <td className="py-2.5 px-3 text-right font-bold text-slate-900">
                                {item.quantity}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {/* ── Camera Viewfinder Card with Animated Reticle ── */}
        <div className="bg-slate-900 rounded-2xl overflow-hidden relative shadow-lg border border-slate-800">
          <div className="aspect-4/3 sm:aspect-16/9 w-full relative flex items-center justify-center bg-black">
            <video
              ref={videoRef}
              playsInline
              muted
              className={`w-full h-full object-cover ${cameraActive ? "block" : "hidden"}`}
            />

            {!cameraActive && (
              <div className="text-center p-6 space-y-3">
                <div className="w-16 h-16 rounded-full bg-slate-800 flex items-center justify-center mx-auto text-slate-400">
                  <CameraOff className="w-8 h-8" />
                </div>
                <div className="text-white text-sm font-semibold">Kamera Belum Aktif</div>
                <p className="text-xs text-slate-400 max-w-xs mx-auto">
                  Gunakan kamera perangkat Anda untuk memindai barcode, atau gunakan USB Scanner Gun fisik.
                </p>
                {cameraError && (
                  <div className="text-xs text-rose-400 bg-rose-950/50 p-2.5 rounded-lg border border-rose-800 max-w-sm mx-auto">
                    {cameraError}
                  </div>
                )}
                <button
                  type="button"
                  onClick={startCamera}
                  className="inline-flex items-center gap-2 px-5 min-h-[48px] rounded-lg font-bold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] transition-all shadow-md"
                >
                  <Camera className="w-4 h-4" />
                  <span>Nyalakan Kamera Web</span>
                </button>
              </div>
            )}

            {/* Viewfinder Reticle Overlay when active */}
            {cameraActive && (
              <div className="absolute inset-0 pointer-events-none flex items-center justify-center">
                {/* Dark Mask Around Scan Window */}
                <div className="relative w-64 sm:w-80 h-44 sm:h-52 border-2 border-white/70 rounded-xl overflow-hidden shadow-[0_0_0_9999px_rgba(0,0,0,0.45)]">
                  {/* Targeting Corners */}
                  <div className="absolute top-0 left-0 w-6 h-6 border-t-4 border-l-4 border-[#2563EB] rounded-tl-sm" />
                  <div className="absolute top-0 right-0 w-6 h-6 border-t-4 border-r-4 border-[#2563EB] rounded-tr-sm" />
                  <div className="absolute bottom-0 left-0 w-6 h-6 border-b-4 border-l-4 border-[#2563EB] rounded-bl-sm" />
                  <div className="absolute bottom-0 right-0 w-6 h-6 border-b-4 border-r-4 border-[#2563EB] rounded-br-sm" />

                  {/* Red Laser Sweeping Line Animation */}
                  <div className="absolute left-2 right-2 h-0.5 bg-gradient-to-r from-transparent via-[#EA580C] to-transparent shadow-[0_0_8px_#EA580C] animate-pulse top-1/2 -translate-y-1/2" />
                </div>
              </div>
            )}

            {cameraActive && cameraDecodeSupported === false && (
              <div className="absolute bottom-3 left-3 right-3 text-xs text-amber-100 bg-amber-900/80 p-2.5 rounded-lg">
                Browser ini tidak mendukung pembacaan barcode dari kamera (BarcodeDetector). Gunakan Chrome/Edge di
                Android, USB scanner, atau ketik kode manual.
              </div>
            )}

            {/* Camera Floating Controls */}
            {cameraActive && (
              <div className="absolute top-3 right-3 flex items-center gap-2">
                {hasTorch && (
                  <button
                    type="button"
                    onClick={toggleTorch}
                    className={`p-3 rounded-full min-h-[48px] min-w-[48px] flex items-center justify-center text-white backdrop-blur-md transition-all ${
                      torchOn ? "bg-amber-500 shadow-lg" : "bg-black/40 hover:bg-black/60"
                    }`}
                    aria-label="Flashlight"
                  >
                    {torchOn ? <Zap className="w-5 h-5" /> : <ZapOff className="w-5 h-5" />}
                  </button>
                )}
                <button
                  type="button"
                  onClick={stopCamera}
                  className="p-3 rounded-full min-h-[48px] min-w-[48px] flex items-center justify-center bg-black/40 hover:bg-black/60 text-white backdrop-blur-md transition-all"
                  aria-label="Matikan Kamera"
                >
                  <CameraOff className="w-5 h-5" />
                </button>
              </div>
            )}
          </div>
        </div>

        {/* ── Manual Barcode Entry & Simulation Triggers ── */}
        <div className="bg-white p-4 rounded-xl border border-[#E2E8F0] shadow-xs">
          <form onSubmit={handleManualSubmit} className="flex gap-2">
            <div className="relative flex-1">
              <Keyboard className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-slate-400" />
              <input
                type="text"
                value={manualCode}
                onChange={(e) => setManualCode(e.target.value)}
                placeholder="Ketik barcode / SKU lalu Enter..."
                className="w-full pl-10 pr-4 min-h-[48px] text-sm bg-slate-50 border border-slate-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-[#2563EB] font-mono"
              />
            </div>
            <button
              type="submit"
              className="px-5 min-h-[48px] rounded-lg font-bold text-sm bg-[#2563EB] text-white hover:bg-[#1D4ED8] active:scale-95 transition-all shadow-xs"
            >
              Scan
            </button>
          </form>

        </div>

        {activeCode && !resolving && resolveError && !displayProduct && mode === "PRICE_CHECK" && (
          <div className="p-4 rounded-xl bg-rose-50 border border-rose-200 text-rose-900 text-sm flex items-center gap-2">
            <AlertCircle className="w-4 h-4 text-rose-600 shrink-0" />
            <span>
              Barcode <span className="font-mono font-bold">{activeCode}</span> tidak terdaftar di master produk.
            </span>
          </div>
        )}

        {/* ── Immediate Product Resolution Card ── */}
        {displayProduct && (
          <div className="bg-white rounded-xl border-2 border-[#BFDBFE] p-5 shadow-sm space-y-4 animate-in fade-in duration-150">
            <div className="flex items-start justify-between gap-3">
              <div>
                <span className="inline-flex items-center gap-1 px-2.5 py-0.5 rounded text-xs font-bold bg-blue-100 text-[#2563EB]">
                  <BarcodeIcon className="w-3.5 h-3.5" />
                  Resolved Master SKU
                </span>
                <h3 className="text-lg font-extrabold text-slate-900 mt-1">
                  {displayProduct.name}
                </h3>
              </div>
              <span className="font-mono text-sm font-bold text-slate-700 bg-slate-100 px-3 py-1 rounded-md">
                {displayProduct.sku}
              </span>
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 bg-slate-50 p-3.5 rounded-lg border border-slate-100 text-xs">
              <div>
                <div className="text-slate-500">Barcode Fisik:</div>
                <div className="font-mono font-bold text-slate-900 mt-0.5">
                  {displayProduct.barcode}
                </div>
              </div>
              <div>
                <div className="text-slate-500">Packaging Multiplier:</div>
                <div className="font-bold text-slate-900 mt-0.5">
                  1 Box = {displayProduct.multiplier} Pcs
                </div>
              </div>
              <div>
                <div className="text-slate-500">Sumber Pemetaan:</div>
                <div className="font-semibold text-[#2563EB] mt-0.5">
                  {displayProduct.source}
                </div>
              </div>
              <div>
                <div className="text-slate-500">Omnichannel SKU:</div>
                <div className="font-semibold text-slate-800 mt-0.5">
                  {displayProduct.external_sku || "Tokopedia / Shopee Ready"}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
