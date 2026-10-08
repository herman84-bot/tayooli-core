"use client"

import React, { useMemo, useState } from "react"
import {
  AlertCircle,
  CheckCircle2,
  ClipboardCheck,
  Eye,
  FileWarning,
  PackageX,
  ShieldAlert,
  Undo2,
  Warehouse,
  X,
} from "lucide-react"
import {
  useQCInspections,
  useQuarantineStock,
  useReceiptQC,
  useReleaseQuarantine,
  useScrapQuarantine,
  useStockReceipt,
  useStockReceipts,
  useSubmitQC,
} from "@/hooks/useWMS"
import type {
  QCInspection,
  QCInspectionInput,
  QCInspectionMode,
  QCInspectionStatus,
  QuarantineLine,
  StockReceipt,
  StockReceiptItem,
} from "@/lib/api"
import { PrintBAK } from "@/components/wms/PrintBAK"

// Patterns: sentry-wms 2-step inbound (staging → QC → putaway) and OCA/wms
// quality-control isolation (quarantine bin) — see docs/references.

function errMsg(e: unknown): string {
  return e instanceof Error && e.message ? e.message : "Terjadi kesalahan. Coba lagi."
}

function fmtQty(v: string | number | null | undefined): string {
  return Number(v ?? 0).toLocaleString("id-ID")
}

function fmtDate(iso?: string | null): string {
  if (!iso) return "-"
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleDateString("id-ID")
}

const INPUT_CLS =
  "w-full min-h-[44px] px-3 py-2 border border-slate-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary/20"
const BTN_PRIMARY =
  "inline-flex min-h-[44px] items-center justify-center gap-1.5 px-4 text-xs font-medium bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 rounded-md shadow-sm"
const BTN_SECONDARY =
  "inline-flex min-h-[44px] items-center justify-center gap-1.5 px-4 text-xs font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-md"

type QCBadgeStatus = QCInspectionStatus | "PENDING_QC"

const QC_BADGE: Record<QCBadgeStatus, { text: string; cls: string }> = {
  PENDING_QC: { text: "Menunggu QC", cls: "bg-amber-50 text-amber-700 border-amber-200" },
  QC_PASSED: { text: "Lolos QC", cls: "bg-emerald-50 text-emerald-700 border-emerald-200" },
  QUARANTINED: { text: "Karantina", cls: "bg-orange-50 text-orange-700 border-orange-200" },
  QC_REJECTED: { text: "Ditolak", cls: "bg-rose-50 text-rose-700 border-rose-200" },
}

function QCBadge({ status }: { status: QCBadgeStatus }) {
  const b = QC_BADGE[status] ?? QC_BADGE.PENDING_QC
  return <span className={`inline-flex rounded-full border px-2.5 py-0.5 text-xs font-medium ${b.cls}`}>{b.text}</span>
}

function ErrorBanner({ text }: { text: string }) {
  return (
    <div role="alert" className="flex items-start gap-2 rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800">
      <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-rose-600" />
      <span>{text}</span>
    </div>
  )
}

function Modal({ title, onClose, children, wide }: { title: string; onClose: () => void; children: React.ReactNode; wide?: boolean }) {
  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto bg-black/50 p-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className={`my-8 w-full ${wide ? "max-w-4xl" : "max-w-lg"} space-y-4 rounded-xl border border-slate-100 bg-white p-6 shadow-xl`}
      >
        <div className="flex items-start justify-between gap-3">
          <h3 className="text-lg font-semibold text-slate-900">{title}</h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Tutup"
            className="inline-flex min-h-[44px] min-w-[44px] items-center justify-center rounded-md text-slate-500 hover:bg-slate-100"
          >
            <X className="h-4 w-4" />
          </button>
        </div>
        {children}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------

export function QCQuarantineView({ warehouseId }: { warehouseId: string }) {
  const [tab, setTab] = useState<"inspeksi" | "karantina">("inspeksi")

  if (!warehouseId) {
    return (
      <div className="rounded-xl border border-dashed border-slate-300 bg-slate-50 p-12 text-center">
        <Warehouse className="mx-auto h-12 w-12 text-slate-400" />
        <h3 className="mt-3 text-base font-medium text-slate-800">Pilih Gudang Terlebih Dahulu</h3>
        <p className="mt-1 text-sm text-slate-500">Pilih gudang pada filter di atas untuk memulai QC & Karantina.</p>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div role="tablist" aria-label="QC & Karantina" className="flex gap-2">
        {(
          [
            ["inspeksi", "Inspeksi Penerimaan", ClipboardCheck],
            ["karantina", "Stok Karantina", ShieldAlert],
          ] as const
        ).map(([key, label, Icon]) => (
          <button
            key={key}
            type="button"
            role="tab"
            aria-selected={tab === key}
            onClick={() => setTab(key)}
            className={`inline-flex min-h-[44px] items-center gap-1.5 rounded-lg border px-4 text-sm font-medium ${
              tab === key ? "border-primary bg-primary/10 text-primary" : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
            }`}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>
      {tab === "inspeksi" ? <InspectionSection warehouseId={warehouseId} /> : <QuarantineSection warehouseId={warehouseId} />}
    </div>
  )
}

// ---------------------------------------------------------------------------
// a) Inspeksi Penerimaan
// ---------------------------------------------------------------------------

function InspectionSection({ warehouseId }: { warehouseId: string }) {
  const { data: receipts = [], isLoading, isError, error } = useStockReceipts(warehouseId)
  const { data: inspections = [] } = useQCInspections(warehouseId)
  const [inspectId, setInspectId] = useState<StockReceipt | null>(null)
  const [resultId, setResultId] = useState<string | null>(null)
  const [toast, setToast] = useState<string | null>(null)

  const posted = receipts.filter((r) => r.status === "POSTED")
  const byReceipt = useMemo(() => {
    const m = new Map<string, QCInspection>()
    for (const i of inspections) m.set(i.receipt_id, i)
    return m
  }, [inspections])

  return (
    <div className="space-y-3">
      {toast && (
        <div role="status" className="flex items-center gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
          <CheckCircle2 className="h-4 w-4 text-emerald-600" />
          {toast}
        </div>
      )}
      {isError && <ErrorBanner text={`Gagal memuat penerimaan: ${errMsg(error)}`} />}
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-200 bg-slate-50/80 text-xs font-semibold uppercase tracking-wider text-slate-600">
              <tr>
                <th className="px-4 py-3">No. Penerimaan</th>
                <th className="px-4 py-3">Pemasok / Asal</th>
                <th className="px-4 py-3">Tanggal</th>
                <th className="px-4 py-3 text-right">Qty Diterima</th>
                <th className="px-4 py-3">Status QC</th>
                <th className="px-4 py-3 text-center">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {isLoading ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-slate-400">Memuat penerimaan...</td>
                </tr>
              ) : posted.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-12 text-center text-slate-400">
                    <ClipboardCheck className="mx-auto mb-2 h-8 w-8 text-slate-300" />
                    Belum ada penerimaan berstatus POSTED untuk diinspeksi.
                  </td>
                </tr>
              ) : (
                posted.map((r) => {
                  const insp = byReceipt.get(r.id)
                  return (
                    <tr key={r.id} className="hover:bg-slate-50/60">
                      <td className="px-4 py-3 font-mono text-xs font-medium text-slate-800">{r.receipt_number}</td>
                      <td className="px-4 py-3 text-slate-700">{r.supplier_name || r.from_name || "-"}</td>
                      <td className="px-4 py-3 text-xs text-slate-600">{fmtDate(r.posted_at || r.created_at)}</td>
                      <td className="px-4 py-3 text-right font-semibold">{fmtQty(r.total_accepted_qty)}</td>
                      <td className="px-4 py-3">
                        <QCBadge status={insp ? insp.status : "PENDING_QC"} />
                      </td>
                      <td className="px-4 py-3 text-center">
                        {insp ? (
                          <button type="button" onClick={() => setResultId(r.id)} className={BTN_SECONDARY}>
                            <Eye className="h-3.5 w-3.5" />
                            Lihat Hasil
                          </button>
                        ) : (
                          <button type="button" onClick={() => setInspectId(r)} className={BTN_PRIMARY}>
                            <ClipboardCheck className="h-3.5 w-3.5" />
                            Mulai Inspeksi
                          </button>
                        )}
                      </td>
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>
      </div>

      {inspectId && (
        <InspectionModal
          receipt={inspectId}
          onClose={() => setInspectId(null)}
          onDone={(msg) => {
            setInspectId(null)
            setToast(msg)
            setTimeout(() => setToast(null), 4000)
          }}
        />
      )}
      {resultId && <ResultModal receiptId={resultId} onClose={() => setResultId(null)} />}
    </div>
  )
}

interface LineState {
  checked: string
  damaged: string
  reason: string
}

export function validateInspection(
  mode: QCInspectionMode,
  sampleQty: string,
  grossCartons: string,
  lines: { item: StockReceiptItem; st: LineState }[],
  driverName: string,
  driverSigned: boolean
): string | null {
  const gc = Number(grossCartons)
  if (grossCartons.trim() === "" || !Number.isFinite(gc) || gc < 0 || !Number.isInteger(gc))
    return "Jumlah karton (Gross Count) wajib diisi dengan bilangan bulat ≥ 0"
  if (mode === "SAMPLING") {
    const s = Number(sampleQty)
    if (sampleQty.trim() === "" || !Number.isFinite(s) || s <= 0) return "Jumlah sampel wajib diisi (> 0) untuk mode SAMPLING"
  }
  if (lines.length === 0) return "Penerimaan tidak memiliki baris dengan batch untuk diinspeksi"
  let anyDamaged = false
  for (const { item, st } of lines) {
    const label = item.product_sku || item.product_name
    const c = Number(st.checked)
    const d = st.damaged.trim() === "" ? 0 : Number(st.damaged)
    if (st.checked.trim() === "" || !Number.isFinite(c) || c < 0) return `Jumlah Dihitung untuk ${label} wajib diisi (≥ 0)`
    if (!Number.isFinite(d) || d < 0) return `Jumlah Rusak untuk ${label} tidak boleh negatif`
    if (d > c) return `Jumlah Rusak untuk ${label} tidak boleh melebihi Jumlah Dihitung`
    if (d > 0) {
      anyDamaged = true
      if (!st.reason.trim()) return `Alasan kerusakan untuk ${label} wajib diisi`
    }
  }
  if (anyDamaged && mode === "SAMPLING") return "Sampel gagal: wajib ulangi dengan inspeksi FULL"
  if (anyDamaged) {
    if (!driverName.trim()) return "Nama sopir wajib diisi untuk BAK"
    if (!driverSigned) return "Sopir wajib menandatangani BAK"
  }
  return null
}

function InspectionModal({
  receipt,
  onClose,
  onDone,
}: {
  receipt: StockReceipt
  onClose: () => void
  onDone: (msg: string) => void
}) {
  const { data, isLoading, isError, error } = useStockReceipt(receipt.id)
  const submit = useSubmitQC()
  const [mode, setMode] = useState<QCInspectionMode>("FULL")
  const [sampleQty, setSampleQty] = useState("")
  const [grossCartons, setGrossCartons] = useState("")
  const [lineState, setLineState] = useState<Record<string, LineState>>({})
  const [driverName, setDriverName] = useState("")
  const [driverSigned, setDriverSigned] = useState(false)
  const [bakNotes, setBakNotes] = useState("")
  const [notes, setNotes] = useState("")
  const [errorText, setErrorText] = useState<string | null>(null)

  const items = (data?.items ?? []).filter((it) => !!it.batch_id)
  const lines = items.map((item) => ({
    item,
    st: lineState[item.id] ?? { checked: "", damaged: "", reason: "" },
  }))
  const setLine = (id: string, patch: Partial<LineState>) =>
    setLineState((prev) => ({ ...prev, [id]: { ...(prev[id] ?? { checked: "", damaged: "", reason: "" }), ...patch } }))

  const anyDamaged = lines.some((l) => Number(l.st.damaged) > 0)
  const totals = lines.reduce(
    (acc, { item, st }) => {
      const exp = Number(item.accepted_qty || 0)
      const c = Number(st.checked || 0)
      const d = Number(st.damaged || 0)
      acc.expected += exp
      acc.checked += c
      acc.damaged += d
      if (st.checked.trim() !== "") {
        if (c < exp) acc.short += exp - c
        if (c > exp) acc.over += c - exp
      }
      return acc
    },
    { expected: 0, checked: 0, damaged: 0, short: 0, over: 0 }
  )
  const liveError = anyDamaged && mode === "SAMPLING" ? "Sampel gagal: wajib ulangi dengan inspeksi FULL" : null

  const handleSubmit = async () => {
    const v = validateInspection(mode, sampleQty, grossCartons, lines, driverName, driverSigned)
    if (v) {
      setErrorText(v)
      return
    }
    setErrorText(null)
    const input: QCInspectionInput = {
      inspection_mode: mode,
      gross_cartons: Number(grossCartons),
      driver_signed: anyDamaged ? driverSigned : false,
      items: lines.map(({ item, st }) => {
        const d = st.damaged.trim() === "" ? 0 : Number(st.damaged)
        return {
          batch_id: item.batch_id as string,
          checked_qty: Number(st.checked),
          damaged_qty: d,
          ...(d > 0 ? { damage_reason: st.reason.trim() } : {}),
        }
      }),
    }
    if (mode === "SAMPLING") input.sample_qty = Number(sampleQty)
    if (anyDamaged) {
      input.driver_name = driverName.trim()
      if (bakNotes.trim()) input.bak_notes = bakNotes.trim()
    }
    if (notes.trim()) input.notes = notes.trim()
    try {
      await submit.mutateAsync({ receiptId: receipt.id, input })
      onDone(`Inspeksi QC ${receipt.receipt_number} tersimpan`)
    } catch (e) {
      setErrorText(errMsg(e))
    }
  }

  return (
    <Modal title={`Inspeksi QC — ${receipt.receipt_number}`} onClose={onClose} wide>
      {errorText && <ErrorBanner text={errorText} />}
      {!errorText && liveError && <ErrorBanner text={liveError} />}
      {isLoading ? (
        <p className="py-6 text-center text-sm text-slate-400">Memuat item penerimaan...</p>
      ) : isError ? (
        <ErrorBanner text={`Gagal memuat item: ${errMsg(error)}`} />
      ) : (
        <div className="space-y-4 text-sm">
          <div className="grid gap-3 sm:grid-cols-3">
            <div>
              <label htmlFor="qc-mode" className="mb-1 block text-xs font-medium text-slate-700">Mode Inspeksi</label>
              <select id="qc-mode" value={mode} onChange={(e) => setMode(e.target.value as QCInspectionMode)} className={INPUT_CLS}>
                <option value="FULL">FULL (100%)</option>
                <option value="SAMPLING">SAMPLING</option>
              </select>
            </div>
            {mode === "SAMPLING" && (
              <div>
                <label htmlFor="qc-sample" className="mb-1 block text-xs font-medium text-slate-700">
                  Jumlah Sampel <span className="text-rose-500">*</span>
                </label>
                <input id="qc-sample" type="number" min={0} step="any" value={sampleQty} onChange={(e) => setSampleQty(e.target.value)} className={INPUT_CLS} />
              </div>
            )}
            <div>
              <label htmlFor="qc-gross" className="mb-1 block text-xs font-medium text-slate-700">
                Jumlah Karton (Gross Count) <span className="text-rose-500">*</span>
              </label>
              <input id="qc-gross" type="number" min={0} step={1} value={grossCartons} onChange={(e) => setGrossCartons(e.target.value)} className={INPUT_CLS} />
            </div>
          </div>

          {items.length === 0 ? (
            <p className="rounded-lg border border-dashed border-slate-300 p-6 text-center text-slate-400">
              Tidak ada baris dengan batch untuk diinspeksi.
            </p>
          ) : (
            <div className="overflow-x-auto rounded-lg border border-slate-200">
              <table className="w-full text-left text-sm">
                <thead className="bg-slate-50 text-xs font-semibold uppercase text-slate-600">
                  <tr>
                    <th className="px-3 py-2">Produk / Batch</th>
                    <th className="px-3 py-2 text-right">Diharapkan</th>
                    <th className="px-3 py-2">Jumlah Dihitung</th>
                    <th className="px-3 py-2">Rusak</th>
                    <th className="px-3 py-2">Alasan Rusak</th>
                    <th className="px-3 py-2 text-right">Selisih</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {lines.map(({ item, st }) => {
                    const exp = Number(item.accepted_qty || 0)
                    const diff = st.checked.trim() === "" ? null : Number(st.checked) - exp
                    const dmg = Number(st.damaged) > 0
                    return (
                      <tr key={item.id}>
                        <td className="px-3 py-2">
                          <div className="font-medium text-slate-900">{item.product_name}</div>
                          <div className="font-mono text-xs text-slate-500">{item.product_sku} · {item.batch_number}</div>
                        </td>
                        <td className="px-3 py-2 text-right font-semibold">{fmtQty(exp)}</td>
                        <td className="px-3 py-2">
                          <input
                            type="number"
                            min={0}
                            step="any"
                            aria-label={`Jumlah Dihitung ${item.product_sku}`}
                            value={st.checked}
                            onChange={(e) => setLine(item.id, { checked: e.target.value })}
                            className={`${INPUT_CLS} w-28`}
                          />
                        </td>
                        <td className="px-3 py-2">
                          <input
                            type="number"
                            min={0}
                            step="any"
                            aria-label={`Rusak ${item.product_sku}`}
                            value={st.damaged}
                            onChange={(e) => setLine(item.id, { damaged: e.target.value })}
                            className={`${INPUT_CLS} w-24`}
                          />
                        </td>
                        <td className="px-3 py-2">
                          <input
                            type="text"
                            aria-label={`Alasan Rusak ${item.product_sku}`}
                            placeholder={dmg ? "Wajib diisi" : "-"}
                            disabled={!dmg}
                            value={st.reason}
                            onChange={(e) => setLine(item.id, { reason: e.target.value })}
                            className={`${INPUT_CLS} w-44 disabled:bg-slate-50`}
                          />
                        </td>
                        <td className="px-3 py-2 text-right text-xs font-semibold">
                          {diff === null || diff === 0 ? (
                            <span className="text-slate-400">{diff === 0 ? "Sesuai" : "-"}</span>
                          ) : diff < 0 ? (
                            <span className="text-rose-600">Kurang {fmtQty(-diff)}</span>
                          ) : (
                            <span className="text-amber-600">Lebih {fmtQty(diff)}</span>
                          )}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
                <tfoot className="bg-slate-50 text-xs font-semibold">
                  <tr>
                    <td className="px-3 py-2">Total</td>
                    <td className="px-3 py-2 text-right">{fmtQty(totals.expected)}</td>
                    <td className="px-3 py-2">{fmtQty(totals.checked)}</td>
                    <td className="px-3 py-2">{fmtQty(totals.damaged)}</td>
                    <td />
                    <td className="px-3 py-2 text-right" data-testid="qc-total-diff">
                      Kurang {fmtQty(totals.short)} / Lebih {fmtQty(totals.over)}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          )}

          {anyDamaged && (
            <fieldset className="space-y-3 rounded-lg border border-amber-300 bg-amber-50 p-3">
              <legend className="flex items-center gap-1.5 px-1 text-xs font-semibold text-amber-800">
                <FileWarning className="h-4 w-4" /> Berita Acara Kerusakan (BAK)
              </legend>
              <div>
                <label htmlFor="qc-driver" className="mb-1 block text-xs font-medium text-amber-900">
                  Nama Sopir <span className="text-rose-600">*</span>
                </label>
                <input id="qc-driver" type="text" value={driverName} onChange={(e) => setDriverName(e.target.value)} className={`${INPUT_CLS} bg-white`} />
              </div>
              <label className="flex min-h-[44px] items-center gap-2 text-xs font-medium text-amber-900">
                <input type="checkbox" checked={driverSigned} onChange={(e) => setDriverSigned(e.target.checked)} className="h-5 w-5" />
                Sopir telah menandatangani BAK <span className="text-rose-600">*</span>
              </label>
              <div>
                <label htmlFor="qc-bak-notes" className="mb-1 block text-xs font-medium text-amber-900">Catatan BAK</label>
                <textarea id="qc-bak-notes" rows={2} value={bakNotes} onChange={(e) => setBakNotes(e.target.value)} className={`${INPUT_CLS} bg-white`} />
              </div>
            </fieldset>
          )}

          <div>
            <label htmlFor="qc-notes" className="mb-1 block text-xs font-medium text-slate-700">Catatan Inspeksi</label>
            <textarea id="qc-notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} className={INPUT_CLS} />
          </div>
        </div>
      )}
      <div className="flex justify-end gap-2 border-t border-slate-100 pt-3">
        <button type="button" onClick={onClose} className={BTN_SECONDARY}>Batal</button>
        <button
          type="button"
          onClick={handleSubmit}
          disabled={submit.isPending || isLoading || !!liveError || items.length === 0}
          className={BTN_PRIMARY}
        >
          {submit.isPending ? "Menyimpan..." : "Simpan Hasil QC"}
        </button>
      </div>
    </Modal>
  )
}

function ResultModal({ receiptId, onClose }: { receiptId: string; onClose: () => void }) {
  const { data, isLoading, isError, error } = useReceiptQC(receiptId)
  const [showBAK, setShowBAK] = useState(false)
  const insp = data?.inspection
  const hasDamage = (data?.items ?? []).some((it) => Number(it.damaged_qty) > 0)

  return (
    <Modal title={`Hasil QC${insp ? ` — ${insp.receipt_number}` : ""}`} onClose={onClose} wide>
      {isLoading ? (
        <p className="py-6 text-center text-sm text-slate-400">Memuat hasil inspeksi...</p>
      ) : isError ? (
        <ErrorBanner text={`Gagal memuat hasil: ${errMsg(error)}`} />
      ) : !data || !insp ? (
        <p className="py-6 text-center text-sm text-slate-400">Belum ada hasil inspeksi.</p>
      ) : (
        <div className="space-y-4 text-sm">
          <div className="grid gap-2 rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs sm:grid-cols-3">
            <div>Status: <QCBadge status={insp.status} /></div>
            <div>Mode: <b>{insp.inspection_mode}</b>{insp.sample_qty ? ` (sampel ${fmtQty(insp.sample_qty)})` : ""}</div>
            <div>Karton: <b>{insp.gross_cartons}</b></div>
            <div>Petugas: <b>{insp.inspector_name}</b></div>
            <div>Dihitung: <b>{fmtQty(insp.total_checked_qty)}</b> · Lolos: <b>{fmtQty(insp.total_passed_qty)}</b></div>
            <div>Rusak: <b>{fmtQty(insp.total_damaged_qty)}</b> · Kurang {fmtQty(insp.shortage_qty)} / Lebih {fmtQty(insp.overage_qty)}</div>
            {insp.bak_number && <div className="sm:col-span-3">No. BAK: <b className="font-mono">{insp.bak_number}</b></div>}
          </div>
          <div className="overflow-x-auto rounded-lg border border-slate-200">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-50 font-semibold uppercase text-slate-600">
                <tr>
                  <th className="px-3 py-2">Produk / Batch</th>
                  <th className="px-3 py-2 text-right">Staging</th>
                  <th className="px-3 py-2 text-right">Dihitung</th>
                  <th className="px-3 py-2 text-right">Lolos</th>
                  <th className="px-3 py-2 text-right">Rusak</th>
                  <th className="px-3 py-2">Alasan</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {data.items.map((it) => (
                  <tr key={it.id}>
                    <td className="px-3 py-2">
                      <div className="font-medium">{it.product_name}</div>
                      <div className="font-mono text-slate-500">{it.product_sku} · {it.batch_number}</div>
                    </td>
                    <td className="px-3 py-2 text-right">{fmtQty(it.staged_qty)}</td>
                    <td className="px-3 py-2 text-right">{fmtQty(it.checked_qty)}</td>
                    <td className="px-3 py-2 text-right">{fmtQty(it.passed_qty)}</td>
                    <td className="px-3 py-2 text-right">{fmtQty(it.damaged_qty)}</td>
                    <td className="px-3 py-2">{it.damage_reason || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {hasDamage && (
            <div className="flex justify-end">
              <button type="button" onClick={() => setShowBAK(true)} className={BTN_PRIMARY}>
                <FileWarning className="h-4 w-4" />
                Lihat / Cetak BAK
              </button>
            </div>
          )}
          {showBAK && <PrintBAK inspection={insp} items={data.items} onClose={() => setShowBAK(false)} />}
        </div>
      )}
    </Modal>
  )
}

// ---------------------------------------------------------------------------
// b) Stok Karantina
// ---------------------------------------------------------------------------

type QAction = { kind: "release" | "scrap"; line: QuarantineLine }

function QuarantineSection({ warehouseId }: { warehouseId: string }) {
  const { data: lines = [], isLoading, isError, error } = useQuarantineStock(warehouseId)
  const [action, setAction] = useState<QAction | null>(null)
  const [toast, setToast] = useState<string | null>(null)

  return (
    <div className="space-y-3">
      {toast && (
        <div role="status" className="flex items-center gap-2 rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800">
          <CheckCircle2 className="h-4 w-4 text-emerald-600" />
          {toast}
        </div>
      )}
      {isError && <ErrorBanner text={`Gagal memuat stok karantina: ${errMsg(error)}`} />}
      <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-slate-200 bg-slate-50/80 text-xs font-semibold uppercase tracking-wider text-slate-600">
              <tr>
                <th className="px-4 py-3">Produk</th>
                <th className="px-4 py-3">SKU</th>
                <th className="px-4 py-3">Batch</th>
                <th className="px-4 py-3">Kedaluwarsa</th>
                <th className="px-4 py-3">Rak</th>
                <th className="px-4 py-3 text-right">Qty</th>
                <th className="px-4 py-3 text-center">Aksi</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100">
              {isLoading ? (
                <tr>
                  <td colSpan={7} className="px-4 py-8 text-center text-slate-400">Memuat stok karantina...</td>
                </tr>
              ) : lines.length === 0 ? (
                <tr>
                  <td colSpan={7} className="px-4 py-12 text-center text-slate-400">
                    <ShieldAlert className="mx-auto mb-2 h-8 w-8 text-slate-300" />
                    Tidak ada stok di area karantina.
                  </td>
                </tr>
              ) : (
                lines.map((l) => (
                  <tr key={`${l.batch_id}-${l.location_id}`} className="hover:bg-slate-50/60">
                    <td className="px-4 py-3 font-medium text-slate-900">{l.product_name || "-"}</td>
                    <td className="px-4 py-3 font-mono text-xs">{l.product_sku || "-"}</td>
                    <td className="px-4 py-3 font-mono text-xs">{l.batch_number}</td>
                    <td className="px-4 py-3 text-xs">{fmtDate(l.expiry_date)}</td>
                    <td className="px-4 py-3 font-mono text-xs">{l.location_code}</td>
                    <td className="px-4 py-3 text-right font-semibold">{fmtQty(l.quantity)}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap justify-center gap-2">
                        <button type="button" onClick={() => setAction({ kind: "release", line: l })} className={BTN_SECONDARY}>
                          <Undo2 className="h-3.5 w-3.5" />
                          Rilis ke Staging
                        </button>
                        <button
                          type="button"
                          onClick={() => setAction({ kind: "scrap", line: l })}
                          className="inline-flex min-h-[44px] items-center gap-1.5 rounded-md bg-rose-600 px-4 text-xs font-medium text-white hover:bg-rose-500"
                        >
                          <PackageX className="h-3.5 w-3.5" />
                          Musnahkan (Scrap)
                        </button>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
      {action && (
        <QuarantineActionModal
          warehouseId={warehouseId}
          action={action}
          onClose={() => setAction(null)}
          onDone={(msg) => {
            setAction(null)
            setToast(msg)
            setTimeout(() => setToast(null), 4000)
          }}
        />
      )}
    </div>
  )
}

function QuarantineActionModal({
  warehouseId,
  action,
  onClose,
  onDone,
}: {
  warehouseId: string
  action: QAction
  onClose: () => void
  onDone: (msg: string) => void
}) {
  const release = useReleaseQuarantine()
  const scrap = useScrapQuarantine()
  const { kind, line } = action
  const max = Number(line.quantity)
  const [qty, setQty] = useState(String(max))
  const [notes, setNotes] = useState("")
  const [errorText, setErrorText] = useState<string | null>(null)
  const isScrap = kind === "scrap"
  const pending = release.isPending || scrap.isPending

  const handleSubmit = async () => {
    const q = Number(qty)
    if (qty.trim() === "" || !Number.isFinite(q) || q <= 0) return setErrorText("Jumlah harus lebih besar dari 0")
    if (q > max) return setErrorText(`Jumlah tidak boleh melebihi stok karantina (${fmtQty(max)})`)
    if (isScrap && !notes.trim()) return setErrorText("Catatan wajib diisi untuk pemusnahan (scrap)")
    setErrorText(null)
    const input = {
      warehouse_id: warehouseId,
      product_id: line.product_id,
      batch_id: line.batch_id,
      quantity: q,
      ...(notes.trim() ? { notes: notes.trim() } : {}),
    }
    try {
      if (isScrap) {
        await scrap.mutateAsync(input)
        onDone(`${fmtQty(q)} unit batch ${line.batch_number} dimusnahkan`)
      } else {
        await release.mutateAsync(input)
        onDone(`${fmtQty(q)} unit batch ${line.batch_number} dirilis ke Staging (siap putaway)`)
      }
    } catch (e) {
      setErrorText(errMsg(e))
    }
  }

  return (
    <Modal title={isScrap ? "Musnahkan Stok Karantina" : "Rilis Karantina ke Staging"} onClose={onClose}>
      {errorText && <ErrorBanner text={errorText} />}
      <div className="rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs">
        <div className="font-semibold text-slate-800">{line.product_name} ({line.product_sku})</div>
        <div className="font-mono text-slate-600">Batch {line.batch_number} · Rak {line.location_code} · Stok {fmtQty(max)}</div>
      </div>
      <div className="space-y-3 text-sm">
        <div>
          <label htmlFor="qa-qty" className="mb-1 block text-xs font-medium text-slate-700">
            Jumlah <span className="text-rose-500">*</span>
          </label>
          <input id="qa-qty" type="number" min={0} max={max} step="any" value={qty} onChange={(e) => setQty(e.target.value)} className={INPUT_CLS} />
        </div>
        <div>
          <label htmlFor="qa-notes" className="mb-1 block text-xs font-medium text-slate-700">
            Catatan {isScrap && <span className="text-rose-500">*</span>}
          </label>
          <textarea id="qa-notes" rows={2} value={notes} onChange={(e) => setNotes(e.target.value)} className={INPUT_CLS} />
        </div>
      </div>
      <div className="flex justify-end gap-2 border-t border-slate-100 pt-3">
        <button type="button" onClick={onClose} className={BTN_SECONDARY}>Batal</button>
        <button type="button" onClick={handleSubmit} disabled={pending} className={BTN_PRIMARY}>
          {pending ? "Memproses..." : isScrap ? "Konfirmasi Scrap" : "Konfirmasi Rilis"}
        </button>
      </div>
    </Modal>
  )
}

export default QCQuarantineView
