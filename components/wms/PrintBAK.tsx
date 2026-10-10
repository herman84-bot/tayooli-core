"use client"

import React, { useId } from "react"
import { Printer, X, FileWarning } from "lucide-react"
import { useCompanyProfile } from "@/hooks/useCompanyProfile"
import type { QCInspection, QCInspectionItem, CompanyProfile } from "@/lib/api"

export interface PrintBAKProps {
  inspection: QCInspection
  items: QCInspectionItem[]
  warehouseName?: string
  companyProfile?: Partial<CompanyProfile>
  onClose: () => void
}

function fmtDate(iso?: string | null): string {
  if (!iso) return "—"
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleDateString("id-ID", { day: "2-digit", month: "long", year: "numeric" })
}

/**
 * Berita Acara Kerusakan Barang (BAK) — A4 printable, same isolation mechanism
 * as PrintDeliveryOrder (window.print + @media print visibility isolation).
 */
export function PrintBAK({ inspection, items, warehouseName, companyProfile, onClose }: PrintBAKProps) {
  const { profile: hookProfile } = useCompanyProfile()
  const activeProfile = companyProfile || hookProfile

  const companyName = (activeProfile?.name || activeProfile?.company_name || "PT TAYOOLI DISTRIBUSI UTAMA").trim()
  const division = (activeProfile?.division || "Divisi Logistik & Pergudangan (WMS Inbound QC)").trim()

  const elementId = `printable-bak-${useId().replace(/:/g, "")}`
  const damaged = items.filter((it) => Number(it.damaged_qty) > 0)
  const totalDamaged = damaged.reduce((s, it) => s + Number(it.damaged_qty || 0), 0)

  return (
    <>
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
            padding: 10mm 12mm !important;
            background: #ffffff !important;
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

      <div
        role="dialog"
        aria-modal="true"
        aria-label="Pratinjau Berita Acara Kerusakan Barang"
        className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto bg-zinc-950/80 p-4 sm:p-6"
      >
        <div className="flex w-full max-w-4xl flex-col items-center gap-4">
          <div className="no-print flex w-full items-center justify-between rounded-xl border border-zinc-800 bg-zinc-900/90 px-4 py-3 text-white shadow-xl">
            <div className="flex items-center gap-2">
              <FileWarning className="h-5 w-5 text-amber-400" />
              <div>
                <h3 className="text-sm font-semibold">Pratinjau BAK (A4)</h3>
                <p className="text-xs text-zinc-400">{inspection.bak_number || inspection.receipt_number}</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => window.print()}
                className="inline-flex min-h-[44px] items-center gap-1.5 rounded-lg bg-emerald-600 px-4 text-xs font-medium text-white hover:bg-emerald-500"
              >
                <Printer className="h-4 w-4" />
                Cetak / Print PDF
              </button>
              <button
                type="button"
                onClick={onClose}
                aria-label="Tutup pratinjau BAK"
                className="inline-flex min-h-[44px] min-w-[44px] items-center justify-center rounded-lg border border-zinc-700 bg-zinc-800 text-zinc-300 hover:bg-zinc-700"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>

          <div
            id={elementId}
            className="w-full rounded-sm border border-zinc-200 bg-white p-8 text-xs leading-relaxed text-zinc-900 shadow-2xl sm:p-10"
            style={{ minHeight: "297mm", maxWidth: "210mm" }}
          >
            <div className="mb-4 border-b-2 border-zinc-900 pb-3">
              <h1 className="text-base font-extrabold uppercase">{companyName}</h1>
              <p className="text-[11px] text-zinc-600">{division}</p>
            </div>

            <div className="my-4 text-center">
              <h2 className="inline-block border-b border-zinc-900 pb-0.5 text-base font-black uppercase tracking-wider">
                BERITA ACARA KERUSAKAN BARANG
              </h2>
              <p className="mt-1 font-mono text-[11px] font-bold">NOMOR: {inspection.bak_number || "—"}</p>
            </div>

            <div className="mb-4 grid grid-cols-2 gap-x-6 gap-y-1 rounded border border-zinc-300 p-3 text-[11px]">
              <span className="text-zinc-500">Tanggal</span>
              <span className="font-medium">: {fmtDate(inspection.created_at)}</span>
              <span className="text-zinc-500">No. Penerimaan (GR)</span>
              <span className="font-mono font-bold">: {inspection.receipt_number}</span>
              <span className="text-zinc-500">Pemasok</span>
              <span className="font-medium">: {inspection.supplier_name || "—"}</span>
              <span className="text-zinc-500">Gudang</span>
              <span className="font-medium">: {warehouseName || inspection.warehouse_id}</span>
              <span className="text-zinc-500">Petugas QC</span>
              <span className="font-medium">: {inspection.inspector_name || "—"}</span>
              <span className="text-zinc-500">Nama Sopir</span>
              <span className="font-medium">
                : {inspection.driver_name || "—"} {inspection.driver_signed ? "(telah menandatangani)" : ""}
              </span>
              <span className="text-zinc-500">Jumlah Karton (Gross)</span>
              <span className="font-medium">: {inspection.gross_cartons}</span>
            </div>

            <table className="mb-4 w-full border-collapse border border-zinc-900 text-[11px]">
              <thead>
                <tr className="border-b border-zinc-900 bg-zinc-100 font-bold">
                  <th className="border-r border-zinc-900 px-2 py-1.5 text-center">No</th>
                  <th className="border-r border-zinc-900 px-2 py-1.5 text-left">SKU</th>
                  <th className="border-r border-zinc-900 px-2 py-1.5 text-left">Produk</th>
                  <th className="border-r border-zinc-900 px-2 py-1.5 text-left">Batch</th>
                  <th className="border-r border-zinc-900 px-2 py-1.5 text-left">Exp</th>
                  <th className="border-r border-zinc-900 px-2 py-1.5 text-right">Qty Rusak</th>
                  <th className="px-2 py-1.5 text-left">Alasan</th>
                </tr>
              </thead>
              <tbody>
                {damaged.length === 0 ? (
                  <tr>
                    <td colSpan={7} className="py-6 text-center italic text-zinc-500">
                      Tidak ada barang rusak.
                    </td>
                  </tr>
                ) : (
                  damaged.map((it, i) => (
                    <tr key={it.id || it.batch_id} className="border-b border-zinc-300">
                      <td className="border-r border-zinc-300 px-2 py-1.5 text-center">{i + 1}</td>
                      <td className="border-r border-zinc-300 px-2 py-1.5 font-mono">{it.product_sku}</td>
                      <td className="border-r border-zinc-300 px-2 py-1.5">{it.product_name}</td>
                      <td className="border-r border-zinc-300 px-2 py-1.5 font-mono">{it.batch_number}</td>
                      <td className="border-r border-zinc-300 px-2 py-1.5">{fmtDate(it.expiry_date)}</td>
                      <td className="border-r border-zinc-300 px-2 py-1.5 text-right font-bold">
                        {Number(it.damaged_qty).toLocaleString("id-ID")}
                      </td>
                      <td className="px-2 py-1.5">{it.damage_reason || "—"}</td>
                    </tr>
                  ))
                )}
              </tbody>
              <tfoot>
                <tr className="border-t border-zinc-900 bg-zinc-100 font-bold">
                  <td colSpan={5} className="border-r border-zinc-900 px-2 py-1.5 text-right uppercase">
                    Total Rusak
                  </td>
                  <td className="border-r border-zinc-900 px-2 py-1.5 text-right">
                    {totalDamaged.toLocaleString("id-ID")}
                  </td>
                  <td />
                </tr>
              </tfoot>
            </table>

            <div className="mb-6 rounded border border-zinc-300 p-2.5 text-[11px]">
              <span className="block font-bold uppercase">Catatan:</span>
              <p className="whitespace-pre-wrap">{inspection.bak_notes || inspection.notes || "—"}</p>
            </div>

            <div className="mt-6 grid grid-cols-3 gap-4 text-center">
              {[
                { title: "Petugas QC", name: inspection.inspector_name },
                { title: "Sopir", name: inspection.driver_name },
                { title: "Supervisor", name: "" },
              ].map((s) => (
                <div key={s.title} className="flex h-36 flex-col justify-between rounded border border-zinc-300 p-3">
                  <p className="text-[10px] font-bold uppercase tracking-wider">{s.title}</p>
                  <div>
                    <p className="mb-1 font-mono text-[10px] text-zinc-400">( ........................................ )</p>
                    <p className="text-[10px] font-semibold">{s.name || "Nama Terang"}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

export default PrintBAK
