import React from 'react'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import '@testing-library/jest-dom'
import InboundPage from '@/app/(app)/wms/inbound/page'

const createMutateAsync = jest.fn()
const postMutateAsync = jest.fn()

const WH = 'wh-1'
const LOC = 'loc-1'

const receiptDraft = {
  id: 'r-1',
  tenant_id: 't',
  receipt_number: 'GR-20261005-ABCDEF12',
  warehouse_id: WH,
  dest_location_id: LOC,
  supplier_name: 'CV Sumber Rejeki',
  status: 'DRAFT',
  created_by: 'u',
  created_at: '2026-10-05T03:00:00Z',
  updated_at: '2026-10-05T03:00:00Z',
  item_count: 1,
  total_accepted_qty: '10',
  total_rejected_qty: '2',
}

jest.mock('@/hooks/useWMS', () => ({
  useWarehouses: () => ({ data: [{ id: 'wh-1', name: 'Gudang Utama' }] }),
  useWarehouseLocations: () => ({
    data: [
      { id: 'loc-1', warehouse_id: 'wh-1', code: 'A-01', name: 'Rak A1', type: 'INTERNAL' },
      { id: 'loc-v', warehouse_id: null, code: '@VENDOR', name: 'Vendor', type: 'VENDOR' },
    ],
  }),
  useStockReceipts: () => ({ data: [receiptDraft], isLoading: false, isError: false }),
  useStockReceipt: () => ({
    data: {
      receipt: receiptDraft,
      items: [
        {
          id: 'i-1', tenant_id: 't', receipt_id: 'r-1', product_id: 'p-1',
          product_name: 'Kopi Bubuk 200g', product_sku: 'KOPI-200',
          accepted_qty: '10', rejected_qty: '2', reject_reason: 'Kemasan sobek', created_at: '',
        },
      ],
    },
    isLoading: false,
    isError: false,
  }),
  useCreateStockReceipt: () => ({ mutateAsync: createMutateAsync, isPending: false }),
  useUpdateStockReceipt: () => ({ mutateAsync: jest.fn(), isPending: false }),
  usePostStockReceipt: () => ({ mutateAsync: postMutateAsync, isPending: false }),
  useCancelStockReceipt: () => ({ mutateAsync: jest.fn(), isPending: false }),
}))

jest.mock('@/hooks/useProducts', () => ({
  useProducts: () => ({ data: [{ id: 'p-1', name: 'Kopi Bubuk 200g', sku: 'KOPI-200', price: 1 }] }),
}))

jest.mock('@/hooks/useBarcodeScanner', () => ({
  useBarcodeScanner: () => ({}),
}))

beforeEach(() => {
  createMutateAsync.mockReset()
  postMutateAsync.mockReset()
})

describe('Barang Masuk page', () => {
  it('lists receipts with plain status labels', () => {
    render(<InboundPage />)
    expect(screen.getByRole('heading', { name: /Barang Masuk/i })).toBeInTheDocument()
    expect(screen.getByText('GR-20261005-ABCDEF12')).toBeInTheDocument()
    expect(screen.getByText('Draf')).toBeInTheDocument()
  })

  it('blocks saving without supplier and items, with a clear message', () => {
    render(<InboundPage />)
    fireEvent.click(screen.getByRole('button', { name: /Terima Barang/i }))
    fireEvent.click(screen.getByRole('button', { name: /Simpan Draf/i }))
    expect(screen.getByRole('alert')).toHaveTextContent('Pilih rak/lokasi tempat barang disimpan.')
    expect(createMutateAsync).not.toHaveBeenCalled()
  })

  it('only offers INTERNAL locations, requires a reject reason, and sends the right payload', async () => {
    createMutateAsync.mockResolvedValue({ receipt: receiptDraft, items: [] })
    render(<InboundPage />)
    fireEvent.click(screen.getByRole('button', { name: /Terima Barang/i }))

    const loc = screen.getByLabelText(/Simpan di rak/i) as HTMLSelectElement
    expect(Array.from(loc.options).map((o) => o.textContent)).not.toContain('@VENDOR — Vendor')
    fireEvent.change(loc, { target: { value: LOC } })
    fireEvent.change(screen.getByLabelText(/Nama pemasok/i), { target: { value: 'CV Sumber Rejeki' } })

    fireEvent.change(screen.getByLabelText('Cari produk'), { target: { value: 'kopi' } })
    fireEvent.click(screen.getByRole('button', { name: /Kopi Bubuk 200g/i }))

    fireEvent.change(screen.getByLabelText(/Jumlah baik Kopi/i), { target: { value: '10' } })
    fireEvent.change(screen.getByLabelText(/Jumlah rusak Kopi/i), { target: { value: '2' } })

    fireEvent.click(screen.getByRole('button', { name: /Simpan Draf/i }))
    expect(screen.getByRole('alert')).toHaveTextContent('Baris 1: isi alasan barang ditolak.')
    expect(createMutateAsync).not.toHaveBeenCalled()

    fireEvent.change(screen.getByLabelText(/Alasan rusak Kopi/i), { target: { value: 'Kemasan sobek' } })
    fireEvent.click(screen.getByRole('button', { name: /Simpan Draf/i }))

    await waitFor(() => expect(createMutateAsync).toHaveBeenCalledTimes(1))
    expect(createMutateAsync).toHaveBeenCalledWith({
      warehouse_id: WH,
      dest_location_id: LOC,
      supplier_name: 'CV Sumber Rejeki',
      supplier_ref: undefined,
      notes: undefined,
      items: [{ product_id: 'p-1', accepted_qty: 10, rejected_qty: 2, reject_reason: 'Kemasan sobek' }],
    })
  })

  it('asks for confirmation before posting a draft', async () => {
    postMutateAsync.mockResolvedValue({ ...receiptDraft, status: 'POSTED' })
    render(<InboundPage />)
    fireEvent.click(screen.getByText('GR-20261005-ABCDEF12'))
    fireEvent.click(screen.getByRole('button', { name: /^Konfirmasi$/ }))
    expect(screen.getByRole('dialog', { name: /Konfirmasi penerimaan/i })).toHaveTextContent('dicatat ke Scrap')
    expect(postMutateAsync).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: /Ya, konfirmasi/i }))
    await waitFor(() => expect(postMutateAsync).toHaveBeenCalledWith('r-1'))
    expect(await screen.findByRole('status')).toHaveTextContent('Stok sudah bertambah')
  })
})
