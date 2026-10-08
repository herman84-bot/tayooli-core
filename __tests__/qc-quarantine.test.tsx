import React from 'react'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import '@testing-library/jest-dom'
import { QCQuarantineView } from '@/components/wms/QCQuarantineView'

const WH = 'wh-1'
const mockSubmit = jest.fn()
const mockRelease = jest.fn()
const mockScrap = jest.fn()

const receipt = {
  id: 'r-1',
  tenant_id: 't',
  receipt_number: 'GR-001',
  warehouse_id: WH,
  dest_location_id: 'loc-stg',
  from_name: 'PT Mitra',
  supplier_name: 'PT Mitra',
  status: 'POSTED',
  created_by: 'u',
  created_at: '2026-10-07T03:00:00Z',
  updated_at: '2026-10-07T03:00:00Z',
  item_count: 2,
  total_accepted_qty: '15',
  total_rejected_qty: '0',
}

jest.mock('@/hooks/useWMS', () => ({
  useStockReceipts: () => ({
    data: [receipt, { ...receipt, id: 'r-2', receipt_number: 'GR-DRAFT', status: 'DRAFT' }],
    isLoading: false,
    isError: false,
  }),
  useStockReceipt: () => ({
    data: {
      receipt,
      items: [
        { id: 'i-1', product_id: 'p-1', product_name: 'Kopi', product_sku: 'KOP-1', batch_id: 'b-1', batch_number: 'B1', accepted_qty: '10', rejected_qty: '0' },
        { id: 'i-2', product_id: 'p-2', product_name: 'Teh', product_sku: 'TEH-1', batch_id: 'b-2', batch_number: 'B2', accepted_qty: '5', rejected_qty: '0' },
        { id: 'i-3', product_id: 'p-3', product_name: 'NoBatch', product_sku: 'NB', accepted_qty: '1', rejected_qty: '0' },
      ],
    },
    isLoading: false,
    isError: false,
  }),
  useQCInspections: () => ({ data: [], isLoading: false }),
  useReceiptQC: () => ({ data: null, isLoading: false }),
  useSubmitQC: () => ({ mutateAsync: mockSubmit, isPending: false }),
  useQuarantineStock: () => ({
    data: [
      { batch_id: 'b-9', batch_number: 'B9', status: 'QUARANTINED', location_id: 'loc-q', location_code: 'QUARANTINE', product_id: 'p-9', product_name: 'Gula', product_sku: 'GUL-1', quantity: '4' },
    ],
    isLoading: false,
    isError: false,
  }),
  useReleaseQuarantine: () => ({ mutateAsync: mockRelease, isPending: false }),
  useScrapQuarantine: () => ({ mutateAsync: mockScrap, isPending: false }),
}))

function openInspection() {
  render(<QCQuarantineView warehouseId={WH} />)
  fireEvent.click(screen.getByRole('button', { name: /Mulai Inspeksi/i }))
  return screen.getByRole('dialog')
}

describe('QCQuarantineView', () => {
  beforeEach(() => jest.clearAllMocks())

  it('shows disabled state without warehouse', () => {
    render(<QCQuarantineView warehouseId="" />)
    expect(screen.getByText(/Pilih Gudang Terlebih Dahulu/i)).toBeInTheDocument()
  })

  it('lists only POSTED receipts with Menunggu QC badge', () => {
    render(<QCQuarantineView warehouseId={WH} />)
    expect(screen.getByText('GR-001')).toBeInTheDocument()
    expect(screen.queryByText('GR-DRAFT')).not.toBeInTheDocument()
    expect(screen.getByText('Menunggu QC')).toBeInTheDocument()
  })

  it('shows BAK fields only when damaged > 0', () => {
    const dlg = openInspection()
    expect(within(dlg).queryByLabelText(/Nama Sopir/i)).not.toBeInTheDocument()
    fireEvent.change(within(dlg).getByLabelText('Rusak KOP-1'), { target: { value: '2' } })
    expect(within(dlg).getByLabelText(/Nama Sopir/i)).toBeInTheDocument()
    expect(within(dlg).getByLabelText(/Sopir telah menandatangani BAK/i)).toBeInTheDocument()
  })

  it('SAMPLING + damaged blocks submit with error', () => {
    const dlg = openInspection()
    fireEvent.change(within(dlg).getByLabelText(/Mode Inspeksi/i), { target: { value: 'SAMPLING' } })
    fireEvent.change(within(dlg).getByLabelText('Rusak KOP-1'), { target: { value: '1' } })
    expect(within(dlg).getByText('Sampel gagal: wajib ulangi dengan inspeksi FULL')).toBeInTheDocument()
    const btn = within(dlg).getByRole('button', { name: /Simpan Hasil QC/i })
    expect(btn).toBeDisabled()
    fireEvent.click(btn)
    expect(mockSubmit).not.toHaveBeenCalled()
  })

  it('submits correct payload', async () => {
    mockSubmit.mockResolvedValue({ data: {} })
    const dlg = openInspection()
    fireEvent.change(within(dlg).getByLabelText(/Gross Count/i), { target: { value: '3' } })
    fireEvent.change(within(dlg).getByLabelText('Jumlah Dihitung KOP-1'), { target: { value: '9' } })
    fireEvent.change(within(dlg).getByLabelText('Rusak KOP-1'), { target: { value: '2' } })
    fireEvent.change(within(dlg).getByLabelText('Alasan Rusak KOP-1'), { target: { value: 'Penyok' } })
    fireEvent.change(within(dlg).getByLabelText('Jumlah Dihitung TEH-1'), { target: { value: '5' } })
    fireEvent.change(within(dlg).getByLabelText(/Nama Sopir/i), { target: { value: 'Budi' } })
    fireEvent.click(within(dlg).getByLabelText(/Sopir telah menandatangani BAK/i))
    expect(within(dlg).getByText('Kurang 1')).toBeInTheDocument()
    fireEvent.click(within(dlg).getByRole('button', { name: /Simpan Hasil QC/i }))
    await waitFor(() => expect(mockSubmit).toHaveBeenCalledTimes(1))
    expect(mockSubmit).toHaveBeenCalledWith({
      receiptId: 'r-1',
      input: {
        inspection_mode: 'FULL',
        gross_cartons: 3,
        driver_signed: true,
        driver_name: 'Budi',
        items: [
          { batch_id: 'b-1', checked_qty: 9, damaged_qty: 2, damage_reason: 'Penyok' },
          { batch_id: 'b-2', checked_qty: 5, damaged_qty: 0 },
        ],
      },
    })
  })

  it('shows server error message on submit failure', async () => {
    mockSubmit.mockRejectedValue(new Error('inspeksi sudah ada'))
    const dlg = openInspection()
    fireEvent.change(within(dlg).getByLabelText(/Gross Count/i), { target: { value: '1' } })
    fireEvent.change(within(dlg).getByLabelText('Jumlah Dihitung KOP-1'), { target: { value: '10' } })
    fireEvent.change(within(dlg).getByLabelText('Jumlah Dihitung TEH-1'), { target: { value: '5' } })
    fireEvent.click(within(dlg).getByRole('button', { name: /Simpan Hasil QC/i }))
    expect(await within(dlg).findByText('inspeksi sudah ada')).toBeInTheDocument()
  })

  it('quarantine table renders and scrap requires notes', async () => {
    mockScrap.mockResolvedValue({ data: {} })
    render(<QCQuarantineView warehouseId={WH} />)
    fireEvent.click(screen.getByRole('tab', { name: /Stok Karantina/i }))
    expect(screen.getByText('Gula')).toBeInTheDocument()
    expect(screen.getByText('B9')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: /Musnahkan \(Scrap\)/i }))
    const dlg = screen.getByRole('dialog')
    fireEvent.click(within(dlg).getByRole('button', { name: /Konfirmasi Scrap/i }))
    expect(within(dlg).getByText(/Catatan wajib diisi/i)).toBeInTheDocument()
    expect(mockScrap).not.toHaveBeenCalled()
    fireEvent.change(within(dlg).getByLabelText(/Catatan/i), { target: { value: 'Rusak berat' } })
    fireEvent.click(within(dlg).getByRole('button', { name: /Konfirmasi Scrap/i }))
    await waitFor(() =>
      expect(mockScrap).toHaveBeenCalledWith({ warehouse_id: WH, product_id: 'p-9', batch_id: 'b-9', quantity: 4, notes: 'Rusak berat' })
    )
  })
})
