"use client"

import React, { useId } from "react"
import { Printer, X, Truck, FileText, CheckCircle2, Clock } from "lucide-react"
import { useCompanyProfile } from "@/hooks/useCompanyProfile"
import type { ShippingManifestDetail, CompanyProfile } from "@/lib/api"

export interface PrintShippingManifestProps {
  manifestDetail: ShippingManifestDetail
  companyProfile?: Partial<CompanyProfile>
  onClose: () => void
}

export function PrintShippingManifest({
  manifestDetail,
  companyProfile,
  onClose,
}: PrintShippingManifestProps) {
  const { manifest, items = [] } = manifestDetail
  const printAreaId = useId().replace(/:/g, "")
  const elementId = `printable-manifest-${printAreaId}`

  const { profile: hookProfile } = useCompanyProfile()
  const activeProfile = companyProfile || hookProfile
  const companyName = (activeProfile?.name || activeProfile?.company_name || "TAYOOLI LOGISTICS & WMS").trim()
  const division = (activeProfile?.division || "Sistem Manajemen Pergudangan & Distribusi Terpadu").trim()

  const handlePrint = () => {
    if (typeof window !== "undefined") {
      window.print()
    }
  }

  const formatDate = (isoString?: string | null) => {
    if (!isoString) return "-"
    try {
      const d = new Date(isoString)
      return d.toLocaleDateString("id-ID", {
        day: "2-digit",
        month: "long",
        year: "numeric",
      })
    } catch {
      return isoString
    }
  }

  const formatTime = (isoString?: string | null) => {
    if (!isoString) return ""
    try {
      const d = new Date(isoString)
      return d.toLocaleTimeString("id-ID", {
        hour: "2-digit",
        minute: "2-digit",
      })
    } catch {
      return ""
    }
  }

  // Calculate totals
  const totalKoli = items.length > 0 ? items.length : manifest.total_packages || 0
  const totalWeight =
    items.length > 0
      ? items.reduce((sum, it) => sum + Number(it.package_weight_kg || 0), 0)
      : Number(manifest.total_weight_kg || 0)

  const isSvg =
    manifest.driver_signature_svg?.trim().startsWith("<svg") ?? false
  const isDataUrl =
    manifest.driver_signature_svg?.trim().startsWith("data:image") ?? false

  return (
    <>
      {/* Global Print Stylesheet for Isolated A4 Output */}
      <style jsx global>{`
        @media print {
          body * {
            visibility: hidden !important;
          }
          #${elementId},
          #${elementId} * {
            visibility: visible !important;
          }
          #${elementId} {
            position: absolute !important;
            left: 0 !important;
            top: 0 !important;
            width: 100% !important;
            max-width: 100% !important;
            margin: 0 !important;
            padding: 8mm 10mm !important;
            background: #ffffff !important;
            color: #09090b !important;
            box-shadow: none !important;
            border: none !important;
          }
          .no-print {
            display: none !important;
          }
          @page {
            size: A4 portrait;
            margin: 8mm;
          }
        }
      `}</style>

      {/* Modal Backdrop Container */}
      <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-zinc-950/80 p-4 backdrop-blur-sm sm:p-6 lg:p-8 no-print">
        <div className="flex w-full max-w-4xl flex-col items-center gap-4">
          {/* Action Control Bar */}
          <div className="flex w-full items-center justify-between rounded-xl border border-zinc-800 bg-zinc-900/95 px-4 py-3 text-white shadow-xl backdrop-blur-md">
            <div className="flex items-center gap-2">
              <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/10 text-blue-400">
                <Truck className="h-4 w-4" />
              </span>
              <div>
                <h3 className="text-sm font-semibold">Cetak Manifest Serah Terima Pengiriman (A4)</h3>
                <p className="text-xs text-zinc-400">
                  {manifest.manifest_number} &bull; {manifest.expedition_name} &bull; {manifest.vehicle_plate}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handlePrint}
                className="inline-flex items-center gap-1.5 rounded-lg bg-blue-600 px-3.5 py-1.5 text-xs font-semibold text-white shadow hover:bg-blue-500 active:scale-95 transition"
              >
                <Printer className="h-3.5 w-3.5" />
                <span>Cetak / Cetak PDF</span>
              </button>
              <button
                type="button"
                onClick={onClose}
                className="inline-flex items-center gap-1.5 rounded-lg border border-zinc-700 bg-zinc-800 px-3 py-1.5 text-xs font-medium text-zinc-300 hover:bg-zinc-700 hover:text-white transition"
              >
                <X className="h-3.5 w-3.5" />
                <span>Tutup</span>
              </button>
            </div>
          </div>

          {/* Printable Manifest A4 Sheet */}
          <div
            id={elementId}
            className="w-full bg-white text-zinc-900 shadow-2xl rounded-sm p-8 sm:p-10 border border-zinc-200"
            style={{ minHeight: "297mm", maxWidth: "210mm" }}
          >
            {/* Header: Company & Title */}
            <div className="border-b-2 border-zinc-900 pb-4 mb-5">
              <div className="flex items-start justify-between">
                <div>
                  <h1 className="text-xl font-black tracking-tight text-zinc-950 uppercase font-sans">
                    {companyName}
                  </h1>
                  <p className="text-xs text-zinc-600">
                    {division}
                  </p>
                  <p className="text-[11px] text-zinc-500 mt-0.5">
                    Gudang: {manifest.warehouse_name || "Pusat Distribusi"}
                  </p>
                </div>
                <div className="text-right">
                  <div className="inline-block bg-zinc-900 text-white font-mono font-bold text-xs px-2.5 py-1 rounded">
                    MANIFEST RESMI
                  </div>
                  <h2 className="text-lg font-bold text-zinc-900 mt-1 font-mono">
                    {manifest.manifest_number}
                  </h2>
                  <p className="text-[11px] text-zinc-500">
                    Status: <strong className="uppercase">{manifest.status}</strong>
                  </p>
                </div>
              </div>

              <div className="mt-4 pt-2 border-t border-zinc-200 text-center">
                <h2 className="text-base font-extrabold tracking-wide uppercase text-zinc-900">
                  MANIFEST SERAH TERIMA PENGIRIMAN
                </h2>
                <p className="text-[11px] text-zinc-500">
                  Bukti sah serah terima koli dan Surat Jalan (Delivery Order) dari Gudang ke Ekspedisi Pengangkut
                </p>
              </div>
            </div>

            {/* Meta Details: Ekspedisi, Driver, Waktu */}
            <div className="grid grid-cols-2 gap-4 bg-zinc-50 p-3 rounded-lg border border-zinc-200 text-xs mb-5">
              <div className="space-y-1">
                <div className="flex">
                  <span className="w-28 text-zinc-500">Jasa Ekspedisi</span>
                  <span className="font-semibold text-zinc-900">: {manifest.expedition_name}</span>
                </div>
                <div className="flex">
                  <span className="w-28 text-zinc-500">Nama Pengemudi</span>
                  <span className="font-semibold text-zinc-900">: {manifest.driver_name}</span>
                </div>
                <div className="flex">
                  <span className="w-28 text-zinc-500">No. Polisi Armada</span>
                  <span className="font-mono font-bold text-zinc-900">: {manifest.vehicle_plate}</span>
                </div>
                <div className="flex">
                  <span className="w-28 text-zinc-500">No. Telepon Sopir</span>
                  <span className="text-zinc-900">: {manifest.driver_phone || "-"}</span>
                </div>
              </div>

              <div className="space-y-1">
                <div className="flex">
                  <span className="w-28 text-zinc-500">Tanggal Dibuat</span>
                  <span className="text-zinc-900">: {formatDate(manifest.created_at)} {formatTime(manifest.created_at)}</span>
                </div>
                <div className="flex">
                  <span className="w-28 text-zinc-500">Tanggal Berangkat</span>
                  <span className="text-zinc-900">
                    : {manifest.dispatched_at ? `${formatDate(manifest.dispatched_at)} ${formatTime(manifest.dispatched_at)}` : "-"}
                  </span>
                </div>
                <div className="flex">
                  <span className="w-28 text-zinc-500">Dibuat Oleh</span>
                  <span className="text-zinc-900">: {manifest.created_by_name || "Staf Gudang"}</span>
                </div>
                {manifest.notes && (
                  <div className="flex">
                    <span className="w-28 text-zinc-500">Catatan</span>
                    <span className="text-zinc-800 italic">: {manifest.notes}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Table of Delivery Orders */}
            <div className="mb-5 overflow-hidden border border-zinc-300 rounded">
              <table className="w-full text-[11px] text-left border-collapse">
                <thead className="bg-zinc-100 border-b border-zinc-300 text-zinc-700 uppercase font-semibold">
                  <tr>
                    <th className="py-2 px-2 text-center w-8 border-r border-zinc-300">No</th>
                    <th className="py-2 px-2.5 border-r border-zinc-300">No. DO</th>
                    <th className="py-2 px-2.5 border-r border-zinc-300">Pelanggan</th>
                    <th className="py-2 px-2.5 border-r border-zinc-300">Kota Tujuan</th>
                    <th className="py-2 px-2 text-right border-r border-zinc-300 w-20">Berat (kg)</th>
                    <th className="py-2 px-2 text-center border-r border-zinc-300 w-24">Tipe Kemasan</th>
                    <th className="py-2 px-2 text-center w-28">Status Scan</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-zinc-200">
                  {items.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="py-4 text-center text-zinc-500">
                        Tidak ada Surat Jalan terdaftar di manifest ini.
                      </td>
                    </tr>
                  ) : (
                    items.map((it, idx) => (
                      <tr key={it.delivery_order_id || idx} className="hover:bg-zinc-50">
                        <td className="py-1.5 px-2 text-center border-r border-zinc-200 text-zinc-500">
                          {idx + 1}
                        </td>
                        <td className="py-1.5 px-2.5 border-r border-zinc-200 font-mono font-semibold text-zinc-900">
                          {it.do_number}
                        </td>
                        <td className="py-1.5 px-2.5 border-r border-zinc-200 text-zinc-800">
                          {it.customer_name || "-"}
                        </td>
                        <td className="py-1.5 px-2.5 border-r border-zinc-200 text-zinc-700">
                          {it.destination_city || "-"}
                        </td>
                        <td className="py-1.5 px-2 text-right border-r border-zinc-200 font-mono">
                          {Number(it.package_weight_kg || 0).toFixed(2)}
                        </td>
                        <td className="py-1.5 px-2 text-center border-r border-zinc-200 text-zinc-600">
                          {it.packaging_type || "Koli"}
                        </td>
                        <td className="py-1.5 px-2 text-center">
                          {it.scanned ? (
                            <span className="inline-flex items-center gap-1 text-[10px] text-emerald-700 font-semibold">
                              <CheckCircle2 className="w-3 h-3 text-emerald-600" />
                              Dimuat
                            </span>
                          ) : (
                            <span className="inline-flex items-center gap-1 text-[10px] text-amber-700 font-semibold">
                              <Clock className="w-3 h-3 text-amber-600" />
                              Menunggu
                            </span>
                          )}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
                <tfoot className="bg-zinc-100 border-t-2 border-zinc-300 font-bold text-zinc-900">
                  <tr>
                    <td colSpan={4} className="py-2 px-3 text-right border-r border-zinc-300">
                      TOTAL SUMMARY:
                    </td>
                    <td className="py-2 px-2 text-right border-r border-zinc-300 font-mono">
                      {totalWeight.toFixed(2)} kg
                    </td>
                    <td colSpan={2} className="py-2 px-3 text-left">
                      <strong>{totalKoli}</strong> Koli
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>

            {/* Signature Section */}
            <div className="mt-8 pt-4 border-t border-zinc-300">
              <div className="grid grid-cols-2 gap-8 text-center text-xs">
                {/* Warehouse Signature */}
                <div className="border border-zinc-300 rounded p-3 bg-zinc-50/50 flex flex-col justify-between h-44">
                  <div>
                    <p className="font-semibold text-zinc-800">Diserahkan Oleh (Staf Gudang)</p>
                    <p className="text-[10px] text-zinc-500">Tayooli Logistics Hub</p>
                  </div>

                  <div className="h-20 flex items-center justify-center">
                    <span className="text-[11px] text-zinc-400 italic">
                      [Tanda Tangan &amp; Cap Gudang]
                    </span>
                  </div>

                  <div className="border-t border-zinc-300 pt-1">
                    <p className="font-medium text-zinc-900">
                      ( {manifest.dispatched_by_name || manifest.created_by_name || "......................................."} )
                    </p>
                    <p className="text-[10px] text-zinc-500">Petugas Dispatch</p>
                  </div>
                </div>

                {/* Driver Signature */}
                <div className="border border-zinc-300 rounded p-3 bg-zinc-50/50 flex flex-col justify-between h-44">
                  <div>
                    <p className="font-semibold text-zinc-800">Diterima Oleh (Sopir Ekspedisi)</p>
                    <p className="text-[10px] text-zinc-500">{manifest.expedition_name}</p>
                  </div>

                  <div className="h-20 flex items-center justify-center overflow-hidden">
                    {isSvg && manifest.driver_signature_svg ? (
                      <div
                        className="max-h-20 max-w-full flex items-center justify-center"
                        dangerouslySetInnerHTML={{ __html: manifest.driver_signature_svg }}
                      />
                    ) : isDataUrl && manifest.driver_signature_svg ? (
                      <img
                        src={manifest.driver_signature_svg}
                        alt="Tanda Tangan Driver"
                        className="max-h-16 max-w-full object-contain"
                      />
                    ) : (
                      <span className="text-[11px] text-zinc-400 italic">
                        [Tanda Tangan Pengemudi]
                      </span>
                    )}
                  </div>

                  <div className="border-t border-zinc-300 pt-1">
                    <p className="font-medium text-zinc-900">
                      ( {manifest.driver_name || "......................................."} )
                    </p>
                    <p className="text-[10px] text-zinc-500">No. Plat: {manifest.vehicle_plate}</p>
                  </div>
                </div>
              </div>

              <div className="mt-4 text-[10px] text-zinc-400 flex items-center justify-between">
                <span>Dokumen digenerate otomatis oleh Tayooli ERP WMS Outbound Engine</span>
                <span>Waktu Cetak: {new Date().toLocaleString("id-ID")}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}
