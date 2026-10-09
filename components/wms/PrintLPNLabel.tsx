"use client"

// FE-09: Printable 100x150 mm / A6 Thermal Pallet LPN Container Sticker Label
// Compliant with ISO-28219 container label specifications (Sentry-WMS §1.3, OCA-WMS §1.3).

import React from "react"
import { Printer, X, Box, Layers, Weight, MapPin, Calendar, CheckCircle2 } from "lucide-react"
import type { StockLPNDetail, PalletType } from "@/lib/api"

export interface PrintLPNLabelProps {
  lpnDetail: StockLPNDetail
  onClose: () => void
}

// ---------------------------------------------------------------------------
// ISO/IEC 15417 Code 128 Pattern Lookup Table (Subset B, 107 symbols)
// ---------------------------------------------------------------------------
const CODE128_PATTERNS: readonly string[] = [
  "212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312", "132212", "221213",
  "221312", "231212", "112232", "122132", "122231", "113222", "123122", "123221", "223211", "221132",
  "221231", "213212", "223112", "312131", "311222", "321122", "321221", "312212", "322112", "322211",
  "212123", "212321", "232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313",
  "231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121", "313121", "211331",
  "231131", "213113", "213311", "213131", "311123", "311321", "331121", "312113", "312311", "332111",
  "314111", "221411", "431111", "111224", "111422", "121124", "121421", "141122", "141221", "112214",
  "112412", "122114", "122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111",
  "111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112", "421211", "212141",
  "214121", "412121", "111143", "111341", "131141", "114113", "114311", "411113", "411311", "113141",
  "114131", "311141", "411131", "211412", "211214", "211232", "2331112",
]

/**
 * Generate SVG rectangles for a given text using Code 128 (Subset B).
 */
export function generateCode128Bars(text: string): { bars: { x: number; width: number }[]; totalWidth: number } {
  const clean = text.trim() || "LPN"
  const startCode = 104 // Start Code B
  const codeIndices: number[] = [startCode]

  let checksum = startCode
  for (let i = 0; i < clean.length; i++) {
    const codePoint = clean.charCodeAt(i)
    // Code 128-B covers ASCII 32 to 126
    const val = codePoint >= 32 && codePoint <= 126 ? codePoint - 32 : 0
    codeIndices.push(val)
    checksum += val * (i + 1)
  }
  const checkDigit = checksum % 103
  codeIndices.push(checkDigit)
  codeIndices.push(106) // Stop code

  const quietZone = 10
  let currentX = quietZone
  const bars: { x: number; width: number }[] = []

  for (const idx of codeIndices) {
    const pattern = CODE128_PATTERNS[idx] ?? "111111"
    for (let p = 0; p < pattern.length; p++) {
      const width = parseInt(pattern[p], 10) || 1
      const isBar = p % 2 === 0
      if (isBar) {
        bars.push({ x: currentX, width })
      }
      currentX += width
    }
  }

  const totalWidth = currentX + quietZone
  return { bars, totalWidth }
}

/**
 * High-contrast Vector Barcode SVG Component.
 */
export function LPNBarcodeSVG({ code, className = "h-16 w-full" }: { code: string; className?: string }) {
  const { bars, totalWidth } = generateCode128Bars(code)

  return (
    <svg
      role="img"
      aria-label={`Barcode untuk ${code}`}
      data-testid="lpn-barcode-svg"
      viewBox={`0 0 ${totalWidth} 60`}
      preserveAspectRatio="none"
      className={className}
    >
      <rect x="0" y="0" width={totalWidth} height="60" fill="white" />
      {bars.map((bar, i) => (
        <rect key={i} x={bar.x} y="0" width={bar.width} height="60" fill="black" />
      ))}
    </svg>
  )
}

const PALLET_LABELS: Record<PalletType, { label: string; desc: string }> = {
  WOODEN: { label: "KAYU (WOODEN)", desc: "Palet kayu standar ISPM-15" },
  PLASTIC: { label: "PLASTIK (PLASTIC)", desc: "Palet higienis HDPE food grade" },
  METAL: { label: "LOGAM / BESI (METAL)", desc: "Palet baja heavy duty beban tinggi" },
  CAGE: { label: "KERANJANG / CAGE", desc: "Palet jaring / collapsible stillage" },
}

function parseNum(val: unknown): number {
  if (typeof val === "number") return val
  if (typeof val === "string") {
    const parsed = parseFloat(val)
    return isNaN(parsed) ? 0 : parsed
  }
  return 0
}

export function PrintLPNLabel({ lpnDetail, onClose }: PrintLPNLabelProps) {
  const { lpn, items = [] } = lpnDetail

  const handlePrint = () => {
    window.print()
  }

  const totalWeight = parseNum(lpn.total_weight_kg)
  const maxWeight = parseNum(lpn.max_weight_kg) || 1000
  const weightPercent = maxWeight > 0 ? Math.min(100, Math.round((totalWeight / maxWeight) * 100)) : 0

  const totalUnits = items.reduce((sum, it) => sum + parseNum(it.quantity), 0)
  const palletInfo = PALLET_LABELS[lpn.pallet_type] ?? {
    label: lpn.pallet_type || "STANDAR",
    desc: "Wadah palet",
  }

  const formattedDate = new Date().toLocaleDateString("id-ID", {
    day: "2-digit",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  })

  return (
    <div className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 overflow-y-auto print:p-0 print:bg-white print:static">
      <style>{`
        @media print {
          @page {
            size: 100mm 150mm;
            margin: 0;
          }
          body {
            -webkit-print-color-adjust: exact !important;
            print-color-adjust: exact !important;
          }
        }
      `}</style>

      <div className="bg-white text-zinc-900 rounded-xl shadow-2xl max-w-lg w-full p-6 print:p-0 print:shadow-none print:max-w-none print:w-[100mm] print:h-[150mm]">
        {/* Screen Controls Header (Hidden on Print) */}
        <div className="flex items-center justify-between border-b pb-4 mb-4 print:hidden">
          <div className="flex items-center gap-2">
            <Box className="w-5 h-5 text-emerald-600" />
            <div>
              <h2 className="text-base font-bold text-zinc-800">Label Palet LPN (100x150 mm)</h2>
              <p className="text-xs text-zinc-500">Stiker Thermal Container Barcode A6</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={handlePrint}
              className="flex items-center gap-1.5 px-3.5 py-1.5 bg-emerald-600 hover:bg-emerald-700 text-white rounded-lg font-medium text-xs transition shadow-sm"
            >
              <Printer className="w-4 h-4" />
              Cetak Label
            </button>
            <button
              onClick={onClose}
              className="p-1.5 text-zinc-400 hover:text-zinc-600 rounded-lg hover:bg-zinc-100 transition"
              aria-label="Tutup"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* 100mm x 150mm Thermal Sticker Canvas */}
        <div className="border-2 border-zinc-900 rounded p-3 text-zinc-900 bg-white font-sans text-xs flex flex-col justify-between min-h-[580px] print:min-h-0 print:h-full print:border-2 print:border-black print:rounded-none">
          {/* Top Section */}
          <div className="space-y-2">
            {/* Header: Company & Warehouse */}
            <div className="flex items-center justify-between border-b-2 border-zinc-900 pb-2">
              <div>
                <span className="text-[11px] font-black tracking-widest text-zinc-900 block uppercase">
                  TAYOOLI ERP - LOGISTICS
                </span>
                <span className="text-[10px] font-bold text-zinc-600">
                  {lpn.warehouse_name || "GUDANG UTAMA LOGISTIK"}
                </span>
              </div>
              <div className="text-right">
                <span className="text-[9px] text-zinc-500 block">Tgl Cetak</span>
                <span className="text-[10px] font-mono font-bold text-zinc-800">{formattedDate}</span>
              </div>
            </div>

            {/* Giant High-Contrast Barcode */}
            <div className="py-2 text-center border-b-2 border-zinc-900 bg-white">
              <div className="px-2 mb-1 flex justify-center">
                <LPNBarcodeSVG code={lpn.lpn_code} className="h-16 w-full max-w-[95%]" />
              </div>
              <div className="font-mono text-xl md:text-2xl font-black tracking-widest text-zinc-900">
                {lpn.lpn_code}
              </div>
            </div>

            {/* Pallet Specs & Location Badges */}
            <div className="grid grid-cols-2 gap-2 border-b-2 border-zinc-900 pb-2 text-[10px]">
              {/* Left: Pallet Type & Status */}
              <div className="space-y-1 border-r border-zinc-300 pr-2">
                <div>
                  <span className="text-[9px] font-bold text-zinc-500 uppercase block">Tipe Palet</span>
                  <span className="text-xs font-black text-zinc-900 block">{palletInfo.label}</span>
                </div>
                <div>
                  <span className="text-[9px] font-bold text-zinc-500 uppercase block">Status LPN</span>
                  <span className="inline-block px-1.5 py-0.5 border border-zinc-900 font-bold text-[9px] uppercase bg-zinc-100">
                    {lpn.status || "STAGED"}
                  </span>
                </div>
              </div>

              {/* Right: Storage Location & Creator */}
              <div className="space-y-1 pl-1">
                <div>
                  <span className="text-[9px] font-bold text-zinc-500 uppercase block">Lokasi Penyimpanan</span>
                  <span className="text-xs font-black font-mono text-zinc-900 block">
                    {lpn.location_code || lpn.location_name || "STAGING INBOUND"}
                  </span>
                  {lpn.location_name && lpn.location_code && (
                    <span className="text-[9px] text-zinc-600 truncate block">{lpn.location_name}</span>
                  )}
                </div>
              </div>
            </div>

            {/* Weight Specifications */}
            <div className="border-b-2 border-zinc-900 pb-2 text-[10px]">
              <div className="flex justify-between items-baseline mb-1">
                <span className="font-bold text-zinc-600 uppercase text-[9px]">Spesifikasi Berat (Muatan)</span>
                <span className="font-mono font-bold text-zinc-800">
                  {totalWeight} kg / {maxWeight} kg ({weightPercent}%)
                </span>
              </div>
              {/* Visual weight bar */}
              <div className="w-full bg-zinc-200 h-2 rounded-sm overflow-hidden border border-zinc-400">
                <div
                  className="bg-zinc-900 h-full"
                  style={{ width: `${Math.min(100, weightPercent)}%` }}
                />
              </div>
              <div className="flex justify-between text-[8px] text-zinc-500 mt-0.5">
                <span>Total Muatan: {totalWeight} kg</span>
                <span>Kapasitas Maks: {maxWeight} kg</span>
              </div>
            </div>

            {/* Table of Contents: Line items */}
            <div className="text-[10px]">
              <div className="flex justify-between items-center font-bold border-b border-zinc-400 pb-1 mb-1">
                <span className="uppercase text-[9px] text-zinc-700">Daftar Isi Kontainer ({items.length} SKU)</span>
                <span className="font-mono">Total: {totalUnits} unit</span>
              </div>

              {items.length === 0 ? (
                <div className="py-4 text-center text-zinc-400 italic text-[10px]">
                  Palet kosong (belum ada item batch yang dimuat)
                </div>
              ) : (
                <div className="max-h-40 overflow-hidden divide-y divide-zinc-200">
                  <table className="w-full text-left text-[9px] border-collapse">
                    <thead>
                      <tr className="border-b border-zinc-300 text-zinc-500 font-bold uppercase text-[8px]">
                        <th className="py-0.5 pr-1">No. Batch / Lot</th>
                        <th className="py-0.5 px-1">SKU & Produk</th>
                        <th className="py-0.5 pl-1 text-right">Qty</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-zinc-100 font-mono">
                      {items.slice(0, 5).map((it, idx) => (
                        <tr key={it.id || idx}>
                          <td className="py-1 pr-1 font-bold text-zinc-900 whitespace-nowrap">
                            {it.batch_number || it.batch_id || "-"}
                            {it.expiry_date && (
                              <span className="block text-[7px] text-zinc-500 font-sans">
                                Exp: {it.expiry_date.slice(0, 10)}
                              </span>
                            )}
                          </td>
                          <td className="py-1 px-1 font-sans">
                            <span className="font-bold font-mono text-zinc-900 block truncate max-w-[150px]">
                              {it.product_sku || "-"}
                            </span>
                            <span className="text-zinc-600 truncate block max-w-[150px] text-[8px]">
                              {it.product_name || "Produk"}
                            </span>
                          </td>
                          <td className="py-1 pl-1 text-right font-black text-zinc-900 whitespace-nowrap">
                            {it.quantity}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {items.length > 5 && (
                    <p className="text-center text-[8px] text-zinc-500 pt-1 italic font-sans">
                      +{items.length - 5} item batch lainnya termuat dalam palet
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Bottom Footer */}
          <div className="pt-2 border-t-2 border-zinc-900 flex justify-between items-center text-[8px] text-zinc-500 font-mono">
            <span>ISO-28219 PALLET STANDARD</span>
            <span className="font-bold text-zinc-900">TAYOOLI WMS CONTAINER</span>
          </div>
        </div>
      </div>
    </div>
  )
}
