"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  api,
  AddOpnameItemInput,
  CreateDeliveryOrderInput,
  CreateLocationInput,
  CreateStockOpnameInput,
  CreateStockScrapInput,
  CreateTransferInput,
  CreateWarehouseInput,
  DeliveryOrder,
  DeliveryOrderDetailResponse,
  OpnameDetailResponse,
  ResolvedProduct,
  StockOpname,
  StockOpnameItem,
  StockScrap,
  StockReceipt,
  StockReceiptType,
  StockReceiptDetailResponse,
  StockReceiptInput,
  StockTransfer,
  TransferDetailResponse,
  Warehouse,
  WarehouseLocation,
  CreateSKUMappingPayload,
  ImportMarketplaceOrdersPayload,
  ImportMarketplaceResponse,
  MarketplaceImportBatch,
  MarketplaceOrder,
  MarketplaceSKUMapping,
} from "@/lib/api"

/**
 * Fetch all warehouses accessible to the authenticated user.
 */
export function useWarehouses() {
  return useQuery<Warehouse[]>({
    queryKey: ["wms", "warehouses"],
    queryFn: async () => {
      const res = await api.wms.warehouses.list()
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Fetch a single warehouse by ID.
 */
export function useWarehouse(id: string | null) {
  return useQuery<Warehouse>({
    queryKey: ["wms", "warehouses", id],
    queryFn: () => api.wms.warehouses.get(id as string),
    enabled: !!id,
    retry: 1,
  })
}

/**
 * Mutation to create a new warehouse.
 */
export function useCreateWarehouse() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateWarehouseInput) => api.wms.warehouses.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wms", "warehouses"] })
    },
  })
}

/**
 * Fetch warehouse locations, optionally filtered by warehouse ID.
 */
export function useWarehouseLocations(warehouseId?: string | null) {
  return useQuery<WarehouseLocation[]>({
    queryKey: ["wms", "locations", warehouseId ?? "all"],
    queryFn: async () => {
      const res = await api.wms.locations.list(warehouseId || undefined)
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Mutation to create a warehouse location (Zone, Rack, Bin, Pallet).
 */
export function useCreateLocation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateLocationInput) => api.wms.locations.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wms", "locations"] })
    },
  })
}

/**
 * Resolve barcode/SKU to master product with packaging multiplier.
 */
export function useResolveBarcode(code: string | null) {
  return useQuery<ResolvedProduct>({
    queryKey: ["wms", "barcodes", code],
    queryFn: () => api.wms.barcodes.resolve(code as string),
    enabled: !!code && code.trim().length > 0,
    retry: false,
    staleTime: 1000 * 60 * 5, // 5 mins cache for resolved barcodes
  })
}

/**
 * Fetch stock transfers, optionally filtered by warehouse ID.
 */
export function useStockTransfers(warehouseId?: string | null) {
  return useQuery<StockTransfer[]>({
    queryKey: ["wms", "transfers", warehouseId ?? "all"],
    queryFn: async () => {
      const res = await api.wms.transfers.list(warehouseId || undefined)
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Fetch detailed stock transfer including line items.
 */
export function useStockTransfer(id: string | null) {
  return useQuery<TransferDetailResponse>({
    queryKey: ["wms", "transfers", id],
    queryFn: () => api.wms.transfers.get(id as string),
    enabled: !!id,
    retry: 1,
  })
}

/**
 * Mutation to create a new inter-warehouse stock transfer.
 */
export function useCreateTransfer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateTransferInput) => api.wms.transfers.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wms", "transfers"] })
    },
  })
}

/**
 * Submit a DRAFT transfer for approval (moves status to PENDING_APPROVAL).
 */
export function useSubmitTransfer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.wms.transfers.submit(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["wms", "transfers"] })
      qc.invalidateQueries({ queryKey: ["wms", "transfers", id] })
    },
  })
}

/**
 * Approve a PENDING_APPROVAL transfer (moves status to APPROVED, unlocks dispatch).
 * Only admin/owner/regional_manager roles may call this, and never the requester.
 */
export function useApproveTransfer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.wms.transfers.approve(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["wms", "transfers"] })
      qc.invalidateQueries({ queryKey: ["wms", "transfers", id] })
    },
  })
}

/**
 * Reject a PENDING_APPROVAL transfer with a mandatory reason (moves status to REJECTED).
 */
export function useRejectTransfer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.wms.transfers.reject(id, reason),
    onSuccess: (_, { id }) => {
      qc.invalidateQueries({ queryKey: ["wms", "transfers"] })
      qc.invalidateQueries({ queryKey: ["wms", "transfers", id] })
    },
  })
}

/**
 * Cancel a DRAFT stock transfer (moves status to CANCELLED). No stock ever
 * moved for a draft, so nothing is reversed in the ledger — the record is kept
 * for audit instead of being hard-deleted.
 */
export function useCancelTransfer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.wms.transfers.cancel(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["wms", "transfers"] })
      qc.invalidateQueries({ queryKey: ["wms", "transfers", id] })
    },
  })
}

/**
 * Dispatch an approved transfer (moves status to IN_TRANSIT).
 */
export function useDispatchTransfer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.wms.transfers.dispatch(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["wms", "transfers"] })
      qc.invalidateQueries({ queryKey: ["wms", "transfers", id] })
    },
  })
}

/**
 * Receive an in-transit transfer at target warehouse (moves status to RECEIVED).
 */
export function useReceiveTransfer() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.wms.transfers.receive(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["wms", "transfers"] })
      qc.invalidateQueries({ queryKey: ["wms", "transfers", id] })
    },
  })
}

/**
 * Fetch delivery orders (Surat Jalan), optionally filtered by warehouse ID.
 */
export function useDeliveryOrders(warehouseId?: string | null) {
  return useQuery<DeliveryOrder[]>({
    queryKey: ["wms", "delivery-orders", warehouseId ?? "all"],
    queryFn: async () => {
      const res = await api.wms.deliveryOrders.list(warehouseId || undefined)
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Fetch detailed delivery order including line items.
 */
export function useDeliveryOrder(id: string | null) {
  return useQuery<DeliveryOrderDetailResponse>({
    queryKey: ["wms", "delivery-orders", id],
    queryFn: () => api.wms.deliveryOrders.get(id as string),
    enabled: !!id,
    retry: 1,
  })
}

/**
 * Mutation to create a new delivery order.
 */
export function useCreateDeliveryOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateDeliveryOrderInput) => api.wms.deliveryOrders.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wms", "delivery-orders"] })
    },
  })
}

/**
 * Mutation to dispatch a delivery order.
 */
export function useDispatchDeliveryOrder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.wms.deliveryOrders.dispatch(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["wms", "delivery-orders"] })
      qc.invalidateQueries({ queryKey: ["wms", "delivery-orders", id] })
    },
  })
}

/**
 * Fetch stock opnames, optionally filtered by warehouse ID.
 */
export function useStockOpnames(warehouseId?: string | null) {
  return useQuery<StockOpname[]>({
    queryKey: ["wms", "opnames", warehouseId ?? "all"],
    queryFn: async () => {
      const res = await api.wms.opnames.list(warehouseId || undefined)
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Fetch a single stock opname and its line items.
 */
export function useStockOpname(id: string | null) {
  return useQuery<OpnameDetailResponse>({
    queryKey: ["wms", "opnames", id],
    queryFn: () => api.wms.opnames.get(id as string),
    enabled: !!id,
    retry: 1,
  })
}

/**
 * Mutation to create a new stock opname session.
 */
export function useCreateStockOpname() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateStockOpnameInput) => api.wms.opnames.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wms", "opnames"] })
    },
  })
}

/**
 * Mutation to add or update a counted item in a stock opname session.
 */
export function useAddOpnameItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: AddOpnameItemInput }) =>
      api.wms.opnames.addItem(id, data),
    onSuccess: (_, variables) => {
      qc.invalidateQueries({ queryKey: ["wms", "opnames"] })
      qc.invalidateQueries({ queryKey: ["wms", "opnames", variables.id] })
    },
  })
}

/**
 * Mutation to complete and reconcile a stock opname session (@LOSS ledger posting).
 */
export function useCompleteStockOpname() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.wms.opnames.complete(id),
    onSuccess: (_, id) => {
      qc.invalidateQueries({ queryKey: ["wms", "opnames"] })
      qc.invalidateQueries({ queryKey: ["wms", "opnames", id] })
      qc.invalidateQueries({ queryKey: ["wms", "locations"] })
    },
  })
}

/**
 * Fetch stock scrap ledger records, optionally filtered by warehouse ID.
 */
export function useStockScraps(warehouseId?: string | null) {
  return useQuery<StockScrap[]>({
    queryKey: ["wms", "scraps", warehouseId ?? "all"],
    queryFn: async () => {
      const res = await api.wms.scraps.list(warehouseId || undefined)
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Mutation to report damaged goods and quarantine stock.
 */
export function useCreateStockScrap() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateStockScrapInput) => api.wms.scraps.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wms", "scraps"] })
      qc.invalidateQueries({ queryKey: ["wms", "locations"] })
    },
  })
}

/**
 * Fetch inbound goods receipts (Barang Masuk), optionally filtered by warehouse.
 */
export function useStockReceipts(warehouseId?: string | null, receiptType?: StockReceiptType | null) {
  return useQuery<StockReceipt[]>({
    queryKey: ["wms", "receipts", warehouseId ?? "all", receiptType ?? "all"],
    queryFn: async () => {
      const res = await api.wms.receipts.list({
        warehouseId: warehouseId || undefined,
        receipt_type: receiptType || undefined,
      })
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Fetch one goods receipt with its items.
 */
export function useStockReceipt(id: string | null) {
  return useQuery<StockReceiptDetailResponse>({
    queryKey: ["wms", "receipts", "detail", id],
    queryFn: () => api.wms.receipts.get(id as string),
    enabled: !!id,
    retry: 1,
  })
}

function invalidateReceipts(qc: ReturnType<typeof useQueryClient>) {
  qc.invalidateQueries({ queryKey: ["wms", "receipts"] })
}

/** Create a DRAFT receipt (stock not changed yet). */
export function useCreateStockReceipt() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: StockReceiptInput) => api.wms.receipts.create(data),
    onSuccess: () => invalidateReceipts(qc),
  })
}

/** Replace a DRAFT receipt's header and items. */
export function useUpdateStockReceipt() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: StockReceiptInput }) =>
      api.wms.receipts.update(id, data),
    onSuccess: () => invalidateReceipts(qc),
  })
}

/** Confirm a receipt: writes stock movements and adds stock. */
export function usePostStockReceipt() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.wms.receipts.post(id),
    onSuccess: () => {
      invalidateReceipts(qc)
      qc.invalidateQueries({ queryKey: ["wms", "scraps"] })
      qc.invalidateQueries({ queryKey: ["wms", "locations"] })
    },
  })
}

/** Cancel a receipt; a confirmed one is reversed with counter movements. */
export function useCancelStockReceipt() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason: string }) =>
      api.wms.receipts.cancel(id, reason),
    onSuccess: () => {
      invalidateReceipts(qc)
      qc.invalidateQueries({ queryKey: ["wms", "scraps"] })
      qc.invalidateQueries({ queryKey: ["wms", "locations"] })
    },
  })
}

/**
 * Fetch marketplace import batches, optionally filtered by warehouse ID.
 */
export function useMarketplaceBatches(warehouseId?: string | null) {
  return useQuery<MarketplaceImportBatch[]>({
    queryKey: ["wms", "marketplace", "batches", warehouseId ?? "all"],
    queryFn: async () => {
      const res = await api.wms.marketplace.listBatches(warehouseId || undefined)
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Fetch canonical marketplace orders with filtering by warehouse, batch, or status.
 */
export function useMarketplaceOrders(params?: {
  warehouse_id?: string
  batch_id?: string
  status?: string
}) {
  return useQuery<MarketplaceOrder[]>({
    queryKey: ["wms", "marketplace", "orders", params ?? {}],
    queryFn: async () => {
      const res = await api.wms.marketplace.listOrders(params)
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Fetch single marketplace order details.
 */
export function useMarketplaceOrder(id: string | null) {
  return useQuery<MarketplaceOrder>({
    queryKey: ["wms", "marketplace", "orders", id],
    queryFn: async () => {
      const res = await api.wms.marketplace.getOrder(id as string)
      if (res && "data" in res && res.data) {
        return res.data as MarketplaceOrder
      }
      return res as MarketplaceOrder
    },
    enabled: !!id,
    retry: 1,
  })
}

/**
 * Fetch external channel SKU mappings.
 */
export function useMarketplaceSKUMappings(channel?: string | null) {
  return useQuery<MarketplaceSKUMapping[]>({
    queryKey: ["wms", "marketplace", "sku-mappings", channel ?? "all"],
    queryFn: async () => {
      const res = await api.wms.marketplace.listSKUMappings(channel || undefined)
      return res.data ?? []
    },
    retry: 1,
  })
}

/**
 * Mutation to import sales orders from marketplace channels via CSV or JSON payload.
 */
export function useImportMarketplaceOrders() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: FormData | ImportMarketplaceOrdersPayload) =>
      api.wms.marketplace.import(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wms", "marketplace", "batches"] })
      qc.invalidateQueries({ queryKey: ["wms", "marketplace", "orders"] })
      qc.invalidateQueries({ queryKey: ["wms", "locations"] })
    },
  })
}

/**
 * Mutation to create or update channel SKU mapping and reprocess pending orders.
 */
export function useCreateSKUMapping() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateSKUMappingPayload) =>
      api.wms.marketplace.createSKUMapping(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["wms", "marketplace", "sku-mappings"] })
      qc.invalidateQueries({ queryKey: ["wms", "marketplace", "orders"] })
      qc.invalidateQueries({ queryKey: ["wms", "marketplace", "batches"] })
      qc.invalidateQueries({ queryKey: ["wms", "locations"] })
    },
  })
}
