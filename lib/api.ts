import { extractErrorMessage } from "@/lib/api/errors"

const BASE = "/api/v1"

/** Default timeout for API requests (ms). */
const DEFAULT_TIMEOUT = 15_000

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), DEFAULT_TIMEOUT)

  try {
    const isFormData = typeof FormData !== "undefined" && init?.body instanceof FormData
    const headers: Record<string, string> = isFormData
      ? { ...(init?.headers as Record<string, string>) }
      : { "Content-Type": "application/json", ...(init?.headers as Record<string, string>) }

    const res = await fetch(`${BASE}${path}`, {
      credentials: "include",
      signal: controller.signal,
      ...init,
      headers,
    })

    if (!res.ok) {
      const contentType = res.headers.get("content-type") || ""
      let message = ""

      if (contentType.includes("application/json")) {
        const body = await res.json().catch(() => null)
        // Handles { error: { code, message } }, { error: "..." }, { message },
        // { errors } — never lets an object stringify to "[object Object]".
        message = extractErrorMessage(body, "")
      } else {
        // Backend handlers (net/http.Error) send plain text bodies, e.g.
        // "conflict: SKU sudah digunakan atau produk memiliki riwayat mutasi/stok"
        message = (await res.text().catch(() => "")).trim()
      }

      // Strip technical "<category>: " prefixes (e.g. "conflict: ", "not found: ")
      // so the raw backend error code doesn't leak into the UI.
      message = message.replace(/^[a-z_]+:\s*/i, "").trim()

      throw new Error(message || `Request failed: ${res.status}`)
    }

    // No-content responses (e.g. DELETE -> 204) have no JSON body to parse.
    if (res.status === 204 || res.headers.get("content-length") === "0") {
      return undefined as T
    }

    const json = await res.json()
    // Backend paginated envelope: { data: [...], total, page, per_page }
    // Unwrap .data when the response is a paginated envelope.
    if (
      json && typeof json === "object" && !Array.isArray(json) &&
      "data" in json && Array.isArray(json.data) &&
      "total" in json && "page" in json
    ) {
      return json as T
    }
    return json as T
  } finally {
    clearTimeout(timeoutId)
  }
}

/** Paginated response envelope from the backend. */
export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  per_page: number
}

export interface PaginationParams {
  page?: number
  per_page?: number
}

export interface Invoice {
  id: string
  tenant_id: string
  vendor_id: string
  vendor_name: string
  amount: number
  status: "pending" | "approved" | "rejected" | "processing" | "pending_review" | "ai_processed" | "ai_failed"
  ai_confidence_score: number
  anomaly_score: number
  anomaly_detected: boolean
  invoice_number: string
  due_date: string
  created_at: string
  updated_at: string
}

export interface Account {
  id: string
  tenant_id: string
  code: string
  name: string
  type: "Asset" | "Liability" | "Equity" | "Revenue" | "Expense"
  parent_id: string | null
  balance: number
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateAccountInput {
  code: string
  name: string
  type: "Asset" | "Liability" | "Equity" | "Revenue" | "Expense"
  parent_id?: string
}

export interface Approval {
  id: string
  tenant_id: string
  workflow_id: string
  target_type: string
  target_id: string
  status: "pending" | "approved" | "rejected"
  current_step_index: number
  requested_by: string
  approved_by?: string
  approved_at?: string
  rejected_by?: string
  rejected_at?: string
  rejection_reason?: string
  created_at: string
  updated_at: string
}

export interface JournalEntryLine {
  id?: string
  account_id: string
  account_name?: string
  debit: number
  credit: number
}

export interface JournalEntry {
  id: string
  tenant_id: string
  date: string
  reference_id: string
  description: string
  total_debit: number
  total_credit: number
  status: "draft" | "posted" | "voided"
  lines: JournalEntryLine[]
  created_at: string
  updated_at: string
}

export interface CreateJournalEntryPayload {
  date: string
  reference_id: string
  description: string
  lines: { account_id: string; debit: number; credit: number }[]
}

export interface Product {
  id: string
  tenant_id: string
  name: string
  description?: string
  sku: string
  price: number
  created_at: string
  updated_at: string
}

export interface CreateProductInput {
  name: string
  sku: string
  description?: string
  price: number
}

export interface InventoryItem {
  id: string
  tenant_id: string
  product_id: string
  quantity: number
  warehouse_location?: string
  created_at: string
  updated_at: string
}

export interface InferenceResult {
  invoice_id: string
  status: string
  anomaly_score?: number
  suggested_gl_account?: string
  error?: string
}

export interface Subscription {
  id: string
  tenant_id: string
  plan: "trial" | "starter" | "bisnis" | "enterprise"
  status: "trialing" | "active" | "past_due" | "cancelled" | "expired"
  trial_started_at: string
  trial_ends_at: string
  billing_period: "monthly" | "annual"
  current_period_start?: string
  current_period_end?: string
  payment_provider?: string
  cancel_at?: string
  cancelled_at?: string
  created_at: string
  updated_at: string
}

export interface SubscriptionInvoice {
  id: string
  tenant_id: string
  subscription_id: string
  invoice_number: string
  amount: number
  currency: string
  status: "draft" | "open" | "paid" | "void"
  period_start: string
  period_end: string
  payment_link?: string
  payment_provider: string
  payment_reference?: string
  due_date: string
  paid_at?: string
  created_at: string
  updated_at: string
}

export interface Plan {
  name: string
  monthly: number | null
  annual: number | null
  featured?: boolean
}

export interface PlanLimits {
  id: string
  plan: string
  max_users: number
  max_vendors: number
  max_invoices_per_month: number
  max_ocr_per_month: number
  multi_entity: boolean
  api_access: boolean
  created_at: string
}

export interface UsageResponse {
  users: number
  vendors: number
  invoices: number
  ocr: number
}

export interface IngestPayload {
  invoice_id: string
  amount: string
  vendor_id?: string
  extracted_text?: string
}

// -----------------------------------------------------------------------------
// WMS & Warehouse Types
// -----------------------------------------------------------------------------

export interface Warehouse {
  id: string
  tenant_id: string
  regional_id?: string | null
  code: string
  name: string
  address?: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateWarehouseInput {
  code: string
  name: string
  regional_id?: string
  address?: string
  is_active?: boolean
}

export type LocationType = "INTERNAL" | "VENDOR" | "CUSTOMER" | "TRANSIT" | "LOSS" | "SCRAP"

export interface WarehouseLocation {
  id: string
  tenant_id: string
  warehouse_id?: string | null
  parent_id?: string | null
  code: string
  barcode?: string | null
  name: string
  type: LocationType
  is_pallet: boolean
  pallet_number?: string | null
  max_capacity?: string | number | null
  created_at: string
  updated_at: string
}

export interface CreateLocationInput {
  warehouse_id?: string
  parent_id?: string
  code: string
  barcode?: string
  name: string
  type?: LocationType
  is_pallet?: boolean
  pallet_number?: string
  max_capacity?: string | number
}

export interface ResolvedProduct {
  product_id: string
  sku: string
  name: string
  barcode: string
  external_sku?: string
  multiplier: string | number
  source: "SKU" | "BARCODE" | "MAPPING"
}

export type TransferStatus =
  | "DRAFT"
  | "PENDING_APPROVAL"
  | "APPROVED"
  | "DISPATCHED"
  | "IN_TRANSIT"
  | "RECEIVED"
  | "REJECTED"
  | "CANCELLED"

export interface StockTransfer {
  id: string
  tenant_id: string
  transfer_number: string
  from_warehouse_id: string
  to_warehouse_id: string
  status: TransferStatus
  requested_by: string
  approved_by?: string | null
  vehicle_plate?: string | null
  driver_name?: string | null
  dispatched_at?: string | null
  received_at?: string | null
  notes?: string | null
  rejection_reason?: string | null
  created_at: string
  updated_at: string
}

export interface StockTransferItem {
  id: string
  tenant_id: string
  transfer_id: string
  product_id: string
  requested_qty: string | number
  sent_qty: string | number
  received_qty: string | number
  source_location_id?: string | null
  dest_location_id?: string | null
  created_at: string
}

export interface CreateTransferItemInput {
  product_id: string
  requested_qty: number | string
  source_location_id?: string
  dest_location_id?: string
}

export interface CreateTransferInput {
  from_warehouse_id: string
  to_warehouse_id: string
  transfer_number?: string
  vehicle_plate?: string
  driver_name?: string
  notes?: string
  items: CreateTransferItemInput[]
}

export interface TransferDetailResponse {
  transfer: StockTransfer
  items: StockTransferItem[]
}

export type DeliveryOrderStatus =
  | "DRAFT"
  | "CONFIRMED"
  | "PICKED"
  | "PACKED"
  | "SHIPPED"
  | "DELIVERED"
  | "RETURNED"
  | "CANCELLED"

export interface DeliveryOrder {
  id: string
  tenant_id: string
  /** null = direct Surat Jalan without a Sales Order */
  sales_order_id: string | null
  customer_id?: string | null
  customer_name?: string | null
  created_by?: string | null
  created_by_name?: string | null
  confirmed_by?: string | null
  confirmed_by_name?: string | null
  packed_by?: string | null
  packed_by_name?: string | null
  dispatched_by?: string | null
  dispatched_by_name?: string | null
  package_weight_kg?: string | number | null
  package_length_cm?: string | number | null
  package_width_cm?: string | number | null
  package_height_cm?: string | number | null
  packaging_type?: string | null
  order_type?: string
  warehouse_id: string
  do_number: string
  status: DeliveryOrderStatus
  expedition_name?: string | null
  tracking_number?: string | null
  driver_name?: string | null
  vehicle_plate?: string | null
  recipient_name?: string | null
  received_date?: string | null
  created_at: string
  updated_at: string
}

export interface DeliveryOrderItem {
  id: string
  tenant_id: string
  delivery_order_id: string
  product_id: string
  quantity: string | number
  location_id: string
  batch_id?: string | null
  batch_number?: string | null
  expiry_date?: string | null
  is_free_item?: boolean
  packed_qty?: string | number
  created_at: string
  product_name?: string
  product_sku?: string
  location_code?: string
}

export interface CreateDeliveryOrderItemInput {
  product_id: string
  quantity: number | string
  location_id?: string
  batch_id?: string
  is_free_item?: boolean
}

export interface CreateDeliveryOrderInput {
  /** Optional Sales Order UUID; omit for a direct Surat Jalan */
  sales_order_id?: string
  customer_id?: string
  order_type?: string
  warehouse_id: string
  do_number: string
  status?: DeliveryOrderStatus
  expedition_name?: string
  tracking_number?: string
  driver_name?: string
  vehicle_plate?: string
  recipient_name?: string
  items: CreateDeliveryOrderItemInput[]
}

export interface PickingTaskItem {
  id: string
  task_id: string
  product_id: string
  batch_id: string
  source_location_id: string
  requested_qty: string | number
  picked_qty: string | number
  damaged_qty: string | number
  status: "PENDING" | "PICKED" | "SHORTAGE" | "DAMAGED"
  shelf_order: number
  is_free_item?: boolean
  product_name?: string
  product_sku?: string
  location_code?: string
  batch_number?: string
  expiry_date?: string
}

export interface PickingTask {
  id: string
  delivery_order_id: string
  task_number: string
  status: "PENDING" | "IN_PROGRESS" | "COMPLETED" | "SHORTAGE" | "CANCELLED"
  picker_id?: string | null
  picker_name?: string | null
  started_at?: string | null
  completed_at?: string | null
  notes?: string | null
}

export interface PickingTaskDetail {
  task: PickingTask
  items: PickingTaskItem[]
  delivery_order: DeliveryOrder
}

export interface Customer {
  id: string
  tenant_id?: string
  name: string
  email?: string | null
  phone?: string | null
  address?: string | null
}

export interface CreateCustomerInput {
  name: string
  email?: string
  phone?: string
  address?: string
}

export interface PackScanResult {
  item_id: string
  product_id: string
  product_name: string
  product_sku: string
  scanned_qty: string | number
  packed_qty: string | number
  requested_qty: string | number
  item_completed: boolean
  order_completed: boolean
  total_items: number
  packed_items: number
}

export interface DeliveryOrderDetailResponse {
  delivery_order: DeliveryOrder
  items: DeliveryOrderItem[]
}

export interface StockMovement {
  id: string
  tenant_id: string
  movement_number: string
  product_id: string
  source_location_id: string
  dest_location_id: string
  quantity: string | number
  unit_cost: string | number
  status: "PENDING" | "DONE" | "CANCELLED"
  reference_type: string
  reference_id: string
  executed_by?: string | null
  created_at: string
}

export type StockOpnameStatus = "DRAFT" | "IN_PROGRESS" | "COMPLETED" | "CANCELLED"

export interface StockOpname {
  id: string
  tenant_id: string
  warehouse_id: string
  opname_number: string
  status: StockOpnameStatus
  conducted_by: string
  approved_by?: string | null
  notes?: string | null
  created_at: string
  updated_at: string
}

export interface StockOpnameItem {
  id: string
  opname_id: string
  tenant_id: string
  product_id: string
  location_id: string
  system_qty: string | number
  physical_qty: string | number
  discrepancy_qty: string | number
  notes?: string | null
  created_at: string
}

export interface CreateStockOpnameInput {
  warehouse_id: string
  opname_number?: string
  notes?: string
}

export interface AddOpnameItemInput {
  product_id: string
  location_id: string
  physical_qty: number | string
  notes?: string
}

export interface OpnameDetailResponse {
  opname: StockOpname
  items: StockOpnameItem[]
}

export interface StockScrap {
  id: string
  tenant_id: string
  scrap_number: string
  warehouse_id: string
  product_id: string
  source_location_id: string
  scrap_location_id: string
  quantity: string | number
  reason: string
  reported_by: string
  created_at: string
}

export interface StockMovement {
  id: string
  tenant_id: string
  movement_number: string
  product_id: string
  product_name?: string
  sku?: string
  source_location_id: string
  source_location_code?: string
  dest_location_id: string
  dest_location_code?: string
  quantity: number | string
  unit_cost: number | string
  status: "PENDING" | "DONE" | "CANCELLED"
  reference_type: string
  reference_id: string
  executed_by?: string | null
  executed_by_name?: string
  created_at: string
}

export interface StockSummary {
  product_id: string
  sku: string
  product_name: string
  warehouse_id?: string | null
  warehouse_name?: string
  location_id?: string | null
  location_code?: string
  quantity: number | string
}

export interface POSOrderItem {
  id: string
  product_id: string
  product_name: string
  sku: string
  quantity: number
  price: number
  discount: number
  subtotal: number
  created_at: string
}

export interface POSOrder {
  id: string
  order_number: string
  customer_id?: string | null
  customer_name?: string
  warehouse_id?: string | null
  warehouse_name?: string
  subtotal: number
  tax_amount: number
  discount_amount: number
  total_amount: number
  payment_method: string
  payment_amount: number
  change_amount: number
  sale_mode: string
  status: string
  sales_order_id?: string | null
  sales_invoice_id?: string | null
  items?: POSOrderItem[]
  created_at: string
}

export interface POSPaymentCharge {
  order_id: string
  provider: string
  method: string
  amount: number
  status: string
  /** True while the tenant has no gateway credentials (ADR-008 demo mode). */
  demo: boolean
  qr_string?: string
  payment_link?: string
  expires_at: string
}

export interface POSPaymentStatus {
  order_id: string
  status: "pending" | "completed" | "consumed" | "failed" | "expired"
  amount: number
  method: string
  demo: boolean
  completed_at?: string
}

export interface TenantPaymentConfigResponse {
  provider: string
  slug?: string
  client_key?: string
  hasCredentials: boolean
  is_production?: boolean
  is_active: boolean
  settlement_bank_name?: string
  settlement_bank_account?: string
  settlement_holder_name?: string
  gateway_fee_percent?: number
}

export interface UpsertPaymentConfigPayload {
  provider: string
  slug?: string
  api_key?: string
  server_key?: string
  client_key?: string
  is_production: boolean
  is_active: boolean
  settlement_bank_name?: string
  settlement_bank_account?: string
  settlement_holder_name?: string
  gateway_fee_percent?: number
}

export interface CreateStockScrapInput {
  warehouse_id: string
  product_id: string
  source_location_id: string
  scrap_location_id?: string
  quantity: number | string
  reason: string
  scrap_number?: string
}

export type StockReceiptStatus = "DRAFT" | "POSTED" | "CANCELLED"
export type StockReceiptType = "PRODUCTION" | "TRANSFER" | "VENDOR"

export interface StockReceipt {
  id: string
  tenant_id: string
  receipt_number: string
  receipt_type: StockReceiptType
  warehouse_id: string
  dest_location_id: string
  from_name: string
  from_warehouse_id?: string
  from_warehouse_name?: string
  source_ref?: string
  transfer_id?: string
  supplier_name?: string
  supplier_ref?: string
  notes?: string
  status: StockReceiptStatus
  created_by: string
  created_at: string
  updated_at: string
  posted_by?: string
  posted_at?: string
  cancelled_by?: string
  cancelled_at?: string
  cancel_reason?: string
  released_by?: string
  released_at?: string
  created_by_name?: string
  posted_by_name?: string
  cancelled_by_name?: string
  released_by_name?: string
  on_hold_batch_count?: number
  item_count: number
  total_accepted_qty: string | number
  total_rejected_qty: string | number
}

export interface StockReceiptItem {
  id: string
  tenant_id: string
  receipt_id: string
  product_id: string
  product_name: string
  product_sku: string
  batch_id?: string
  batch_number?: string
  expiry_date?: string
  expected_qty?: string | number
  accepted_qty: string | number
  rejected_qty: string | number
  reject_reason?: string
  created_at: string
}

export interface StockReceiptItemInput {
  product_id: string
  batch_number?: string
  expiry_date?: string
  expected_qty?: number
  accepted_qty: number
  rejected_qty: number
  reject_reason?: string
}

export interface StockReceiptInput {
  receipt_type?: StockReceiptType
  warehouse_id: string
  dest_location_id: string
  from_name?: string
  from_warehouse_id?: string
  source_ref?: string
  transfer_id?: string
  supplier_name?: string
  supplier_ref?: string
  notes?: string
  items: StockReceiptItemInput[]
}

export interface StockReceiptDetailResponse {
  receipt: StockReceipt
  items: StockReceiptItem[]
}

// --- Sprint 2: QC & Karantina ---------------------------------------------
export type QCInspectionMode = "FULL" | "SAMPLING"
export type QCInspectionStatus = "QC_PASSED" | "QUARANTINED" | "QC_REJECTED"

export interface QCInspection {
  id: string
  receipt_id: string
  receipt_number: string
  warehouse_id: string
  inspection_mode: QCInspectionMode
  sample_qty?: string | null
  gross_cartons: number
  status: QCInspectionStatus
  total_checked_qty: string
  total_passed_qty: string
  total_damaged_qty: string
  shortage_qty: string
  overage_qty: string
  bak_number?: string | null
  bak_notes?: string | null
  driver_name?: string | null
  driver_signed: boolean
  notes?: string | null
  inspector_id: string
  inspector_name: string
  supplier_name: string
  created_at: string
}

export interface QCInspectionItem {
  id: string
  product_id: string
  product_name: string
  product_sku: string
  batch_id: string
  batch_number: string
  expiry_date?: string | null
  staged_qty: string
  checked_qty: string
  passed_qty: string
  damaged_qty: string
  shortage_qty: string
  overage_qty: string
  damage_reason?: string | null
}

export interface QCInspectionDetail {
  inspection: QCInspection
  items: QCInspectionItem[]
}

export interface QCInspectionInput {
  inspection_mode: QCInspectionMode
  sample_qty?: number
  gross_cartons: number
  driver_name?: string
  driver_signed: boolean
  bak_notes?: string
  notes?: string
  items: { batch_id: string; checked_qty: number; damaged_qty: number; damage_reason?: string }[]
}

export interface QuarantineLine {
  batch_id: string
  batch_number: string
  expiry_date?: string | null
  status: string
  location_id: string
  location_code: string
  product_id: string
  product_name?: string
  product_sku?: string
  quantity: string
}

export interface QuarantineActionInput {
  warehouse_id: string
  product_id: string
  batch_id: string
  quantity: number
  notes?: string
}

export interface PutawayPendingLine {
  product_id: string
  product_name?: string
  product_sku?: string
  batch_id: string
  batch_number: string
  expiry_date?: string
  batch_status: string
  staging_location_id: string
  quantity: string | number
  source_receipt_id?: string
  source_receipt_number?: string
  suggested_location_id?: string
  suggested_location_code?: string
  suggestion_source?: string
  default_location_id?: string
}

export interface PutawayInput {
  warehouse_id: string
  product_id: string
  batch_id: string
  quantity: number
  dest_location_id: string
  reason?: string
}

export interface WMSSettings {
  tenant_id: string
  require_release_approval: boolean
  updated_at: string
}

export interface ProductDefaultLocation {
  tenant_id: string
  product_id: string
  product_name?: string
  product_sku?: string
  warehouse_id: string
  warehouse_name?: string
  location_id: string
  location_code: string
}

export interface BatchTraceMovement {
  movement_id: string
  movement_number: string
  product_id: string
  product_name: string
  product_sku: string
  batch_id: string
  batch_number: string
  expiry_date?: string
  source_location_code: string
  dest_location_code: string
  quantity: string | number
  reference_type: string
  reference_id: string
  counterparty?: string
  executed_by_name?: string
  created_at: string
}

export interface BatchTrace {
  batch: {
    id: string
    batch_number: string
    product_id: string
    status: string
    expiry_date?: string
    created_at: string
  }
  movements: BatchTraceMovement[]
  balances: Array<{
    batch_id: string
    batch_number: string
    location_id: string
    location_code: string
    quantity: string | number
    product_id: string
  }>
  total_in: string | number
  total_out: string | number
  on_hand: string | number
}

export interface AuditTrailEntry {
  id: string
  entity_type: string
  entity_id: string
  action: string
  user_name: string
  details?: Record<string, unknown>
  created_at: string
}

export type MarketplaceChannel =
  | "SHOPEE"
  | "TOKOPEDIA"
  | "TIKTOK"
  | "LAZADA"
  | "BLIBLI"
  | "OTHER"

export type MarketplaceBatchStatus = "PENDING" | "PROCESSING" | "COMPLETED" | "FAILED"

export type MarketplaceOrderStatus =
  | "PENDING"
  | "PROCESSING"
  | "COMPLETED"
  | "FAILED"
  | "UNMAPPED_SKU"
  | "STOCK_INSUFFICIENT"

export interface MarketplaceImportBatch {
  id: string
  tenant_id: string
  batch_number: string
  channel: MarketplaceChannel
  warehouse_id: string
  warehouse_name?: string
  file_name: string
  total_orders: number
  processed_orders: number
  failed_orders: number
  unmapped_skus: number
  status: MarketplaceBatchStatus
  uploaded_by: string
  created_at: string
}

export interface MarketplaceOrderItem {
  id: string
  tenant_id: string
  order_id: string
  external_sku: string
  product_id?: string | null
  item_name: string
  quantity: string | number
  unit_price: string | number
  subtotal: string | number
  is_mapped: boolean
  product_name?: string
  product_sku?: string
  multiplier?: string | number
}

export interface MarketplaceOrder {
  id: string
  tenant_id: string
  batch_id?: string | null
  warehouse_id: string
  warehouse_name?: string
  channel: MarketplaceChannel
  external_order_id: string
  order_date: string
  customer_name?: string | null
  customer_phone?: string | null
  shipping_address?: string | null
  courier?: string | null
  tracking_number?: string | null
  total_amount: string | number
  shipping_fee?: string | number
  marketplace_fee?: string | number
  net_amount: string | number
  status: MarketplaceOrderStatus
  sales_order_id?: string | null
  created_at: string
  items?: MarketplaceOrderItem[]
}

export interface ImportMarketplaceOrderItemPayload {
  external_sku: string
  item_name: string
  quantity: number | string
  unit_price: number | string
  subtotal: number | string
}

export interface ImportMarketplaceOrderInput {
  external_order_id: string
  order_date?: string
  customer_name?: string
  customer_phone?: string
  shipping_address?: string
  courier?: string
  tracking_number?: string
  total_amount: number | string
  shipping_fee?: number | string
  marketplace_fee?: number | string
  net_amount?: number | string
  items: ImportMarketplaceOrderItemPayload[]
}

export interface ImportMarketplaceOrdersPayload {
  warehouse_id: string
  channel: MarketplaceChannel
  file_name?: string
  orders?: ImportMarketplaceOrderInput[]
  csv_data?: string
}

export interface CreateSKUMappingPayload {
  product_id: string
  mapping_type?: "MARKETPLACE" | "CUSTOMER" | "VENDOR"
  channel_name: MarketplaceChannel | string
  external_sku: string
  external_name?: string
  multiplier?: number | string
}

export interface MarketplaceSKUMapping {
  id: string
  tenant_id: string
  product_id: string
  mapping_type: "MARKETPLACE" | "CUSTOMER" | "VENDOR"
  channel_name: MarketplaceChannel | string
  external_sku: string
  external_name?: string | null
  multiplier: number | string
  created_at: string
  updated_at?: string
  product_name?: string
  product_sku?: string
}

export interface ImportMarketplaceResponse {
  data?: {
    batch: MarketplaceImportBatch
    orders: MarketplaceOrder[]
  }
  batch?: MarketplaceImportBatch
  orders?: MarketplaceOrder[]
}

function qs(params: PaginationParams): string {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined)
  return entries.length ? `?${new URLSearchParams(entries.map(([k, v]) => [k, String(v)]))}` : ""
}

export const api = {
  invoices: {
    list: (p?: PaginationParams) => request<PaginatedResponse<Invoice>>(`/invoices${p ? qs(p) : ""}`),
    get: (id: string) => request<Invoice>(`/invoices/${id}`),
  },
  accounts: {
    list: (p?: PaginationParams) => request<PaginatedResponse<Account>>(`/accounts${p ? qs(p) : ""}`),
    get: (id: string) => request<Account>(`/accounts/${id}`),
    create: (input: CreateAccountInput) =>
      request<Account>("/accounts", {
        method: "POST",
        body: JSON.stringify(input),
      }),
  },
  approvals: {
    list: (p?: PaginationParams) => request<PaginatedResponse<Approval>>(`/approvals${p ? qs(p) : ""}`),
    get: (id: string) => request<Approval>(`/approvals/${id}`),
    approve: (id: string) =>
      request<Approval>(`/approvals/${id}/approve`, { method: "POST" }),
    reject: (id: string, reason: string) =>
      request<Approval>(`/approvals/${id}/reject`, {
        method: "POST",
        body: JSON.stringify({ reason }),
      }),
  },
  journalEntries: {
    list: (p?: PaginationParams) => request<PaginatedResponse<JournalEntry>>(`/journal-entries${p ? qs(p) : ""}`),
    get: (id: string) => request<JournalEntry>(`/journal-entries/${id}`),
    create: (input: CreateJournalEntryPayload) =>
      request<JournalEntry>("/journal-entries", {
        method: "POST",
        body: JSON.stringify(input),
      }),
  },
  products: {
    list: (p?: PaginationParams) => request<PaginatedResponse<Product>>(`/products${p ? qs(p) : ""}`),
    get: (id: string) => request<Product>(`/products/${id}`),
    create: (input: CreateProductInput) =>
      request<Product>("/products", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    update: (id: string, input: Partial<CreateProductInput>) =>
      request<Product>(`/products/${id}`, {
        method: "PUT",
        body: JSON.stringify(input),
      }),
    delete: (id: string) =>
      request<void>(`/products/${id}`, {
        method: "DELETE",
      }),
    inventory: (productId: string) =>
      request<InventoryItem[]>(`/products/${productId}/inventory`),
    adjustInventory: (productId: string, inventoryId: string, delta: number) =>
      request<InventoryItem[]>(`/products/${productId}/inventory`, {
        method: "PATCH",
        body: JSON.stringify({ inventory_id: inventoryId, delta }),
      }),
  },
  inference: {
    ingest: (input: IngestPayload) =>
      request<{ job_id: string; status: string }>("/inference/ingest", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    status: (invoiceId: string) =>
      request<InferenceResult>(`/inference/status/${invoiceId}`),
  },

  subscription: {
    get: () => request<Subscription>("/subscription"),
    create: (plan: string, period: string) =>
      request<Subscription>("/subscription", {
        method: "POST",
        body: JSON.stringify({ plan, period }),
      }),
    update: (plan: string, period: string) =>
      request<{ status: string }>("/subscription", {
        method: "PATCH",
        body: JSON.stringify({ plan, period }),
      }),
    cancel: () =>
      request<{ status: string }>("/subscription", {
        method: "DELETE",
      }),
    invoices: () => request<SubscriptionInvoice[]>("/subscription/invoices"),
    pay: (invoiceId: string) =>
      request<{ payment_url: string }>("/subscription/pay", {
        method: "POST",
        body: JSON.stringify({ invoice_id: invoiceId }),
      }),
  },

  plans: {
    list: () => request<Plan[]>("/plans"),
    limits: (plan: string) => request<PlanLimits>(`/plans/${plan}/limits`),
  },

  usage: {
    get: () => request<UsageResponse>("/usage"),
  },

  wms: {
    warehouses: {
      list: () => request<{ data: Warehouse[] }>("/wms/warehouses"),
      get: (id: string) => request<Warehouse>(`/wms/warehouses/${id}`),
      create: (data: CreateWarehouseInput) =>
        request<Warehouse>("/wms/warehouses", {
          method: "POST",
          body: JSON.stringify(data),
        }),
    },
    locations: {
      list: (warehouseId?: string) =>
        request<{ data: WarehouseLocation[] }>(
          `/wms/locations${warehouseId ? `?warehouse_id=${encodeURIComponent(warehouseId)}` : ""}`
        ),
      create: (data: CreateLocationInput) =>
        request<WarehouseLocation>("/wms/locations", {
          method: "POST",
          body: JSON.stringify(data),
        }),
    },
    barcodes: {
      resolve: (code: string) =>
        request<ResolvedProduct>(`/wms/barcodes/resolve?code=${encodeURIComponent(code)}`),
    },
    transfers: {
      list: (warehouseId?: string) =>
        request<{ data: StockTransfer[] }>(
          `/wms/transfers${warehouseId ? `?warehouse_id=${encodeURIComponent(warehouseId)}` : ""}`
        ),
      get: (id: string) => request<TransferDetailResponse>(`/wms/transfers/${id}`),
      create: (data: CreateTransferInput) =>
        request<StockTransfer>("/wms/transfers", {
          method: "POST",
          body: JSON.stringify(data),
        }),
      submit: (id: string) =>
        request<StockTransfer>(`/wms/transfers/${id}/submit`, {
          method: "POST",
        }),
      approve: (id: string) =>
        request<StockTransfer>(`/wms/transfers/${id}/approve`, {
          method: "POST",
        }),
      reject: (id: string, reason: string) =>
        request<StockTransfer>(`/wms/transfers/${id}/reject`, {
          method: "POST",
          body: JSON.stringify({ reason }),
        }),
      dispatch: (id: string) =>
        request<StockTransfer>(`/wms/transfers/${id}/dispatch`, {
          method: "POST",
        }),
      receive: (id: string) =>
        request<StockTransfer>(`/wms/transfers/${id}/receive`, {
          method: "POST",
        }),
      // Cancel a DRAFT transfer. No stock moved, so nothing is reversed.
      cancel: (id: string) =>
        request<StockTransfer>(`/wms/transfers/${id}`, {
          method: "DELETE",
        }),
    },
    deliveryOrders: {
      list: (warehouseId?: string) =>
        request<{ data: DeliveryOrder[] }>(
          `/wms/delivery-orders${warehouseId ? `?warehouse_id=${encodeURIComponent(warehouseId)}` : ""}`
        ),
      get: (id: string) => request<DeliveryOrderDetailResponse>(`/wms/delivery-orders/${id}`),
      create: (data: CreateDeliveryOrderInput) =>
        request<DeliveryOrder>("/wms/delivery-orders", {
          method: "POST",
          body: JSON.stringify(data),
        }),
      dispatch: (id: string) =>
        request<DeliveryOrder>(`/wms/delivery-orders/${id}/dispatch`, {
          method: "POST",
        }),
      getPickingTask: (id: string) =>
        request<{ data: PickingTaskDetail }>(`/wms/delivery-orders/${id}/picking`),
      startPickingTask: (id: string) =>
        request<{ data: PickingTaskDetail }>(`/wms/delivery-orders/${id}/picking/start`, {
          method: "POST",
        }),
      recordPickingItem: (id: string, itemId: string, pickedQty: number | string) =>
        request<{ data: PickingTaskDetail }>(`/wms/delivery-orders/${id}/picking/items/${itemId}`, {
          method: "POST",
          body: JSON.stringify({ picked_qty: pickedQty }),
        }),
      reportPickingDamaged: (
        id: string,
        payload: {
          task_item_id: string
          product_id: string
          batch_id: string
          source_location_id: string
          damaged_qty: number | string
          reason: string
        }
      ) =>
        request<{ data: any }>(`/wms/delivery-orders/${id}/picking/damaged`, {
          method: "POST",
          body: JSON.stringify(payload),
        }),
      scanPackItem: (id: string, barcode: string, quantity: number = 1) =>
        request<{ data: PackScanResult }>(`/wms/delivery-orders/${id}/pack/scan`, {
          method: "POST",
          body: JSON.stringify({ barcode, quantity }),
        }),
      completePack: (
        id: string,
        payload: {
          package_weight_kg?: number | string
          package_length_cm?: number | string
          package_width_cm?: number | string
          package_height_cm?: number | string
          packaging_type?: string
        }
      ) =>
        request<{ data: DeliveryOrder }>(`/wms/delivery-orders/${id}/pack/complete`, {
          method: "POST",
          body: JSON.stringify(payload),
        }),
    },
    opnames: {
      list: (warehouseId?: string) =>
        request<{ data: StockOpname[] }>(
          `/wms/opnames${warehouseId ? `?warehouse_id=${encodeURIComponent(warehouseId)}` : ""}`
        ),
      get: (id: string) => request<OpnameDetailResponse>(`/wms/opnames/${id}`),
      create: (data: CreateStockOpnameInput) =>
        request<StockOpname>("/wms/opnames", {
          method: "POST",
          body: JSON.stringify(data),
        }),
      addItem: (id: string, data: AddOpnameItemInput) =>
        request<StockOpnameItem>(`/wms/opnames/${id}/items`, {
          method: "POST",
          body: JSON.stringify(data),
        }),
      complete: (id: string) =>
        request<StockOpname>(`/wms/opnames/${id}/complete`, {
          method: "POST",
        }),
    },
    scraps: {
      list: (warehouseId?: string) =>
        request<{ data: StockScrap[] }>(
          `/wms/scraps${warehouseId ? `?warehouse_id=${encodeURIComponent(warehouseId)}` : ""}`
        ),
      create: (data: CreateStockScrapInput) =>
        request<StockScrap>("/wms/scraps", {
          method: "POST",
          body: JSON.stringify(data),
        }),
    },
    receipts: {
      list: (params?: { warehouseId?: string; status?: StockReceiptStatus; receipt_type?: StockReceiptType }) => {
        const q = new URLSearchParams()
        if (params?.warehouseId) q.set("warehouse_id", params.warehouseId)
        if (params?.status) q.set("status", params.status)
        if (params?.receipt_type) q.set("receipt_type", params.receipt_type)
        const s = q.toString()
        return request<{ data: StockReceipt[] }>(`/wms/receipts${s ? `?${s}` : ""}`)
      },
      get: (id: string) => request<StockReceiptDetailResponse>(`/wms/receipts/${id}`),
      create: (data: StockReceiptInput) =>
        request<StockReceiptDetailResponse>("/wms/receipts", {
          method: "POST",
          body: JSON.stringify(data),
        }),
      update: (id: string, data: StockReceiptInput) =>
        request<StockReceiptDetailResponse>(`/wms/receipts/${id}`, {
          method: "PUT",
          body: JSON.stringify(data),
        }),
      post: (id: string) =>
        request<StockReceipt>(`/wms/receipts/${id}/post`, { method: "POST" }),
      cancel: (id: string, reason: string) =>
        request<StockReceipt>(`/wms/receipts/${id}/cancel`, {
          method: "POST",
          body: JSON.stringify({ reason }),
        }),
      release: (id: string) =>
        request<{ data: StockReceipt }>(`/wms/receipts/${id}/release`, { method: "POST" }),
    },
    putaway: {
      getPending: (warehouseId: string) =>
        request<{ data: PutawayPendingLine[] }>(`/wms/putaway/pending?warehouse_id=${encodeURIComponent(warehouseId)}`),
      confirm: (data: PutawayInput) =>
        request<{ data: unknown }>("/wms/putaway/confirm", {
          method: "POST",
          body: JSON.stringify(data),
        }),
    },
    qc: {
      getByReceipt: (receiptId: string) =>
        request<{ data: QCInspectionDetail | null }>(`/wms/receipts/${receiptId}/qc`),
      submit: (receiptId: string, input: QCInspectionInput) =>
        request<{ data: QCInspectionDetail }>(`/wms/receipts/${receiptId}/qc`, {
          method: "POST",
          body: JSON.stringify(input),
        }),
      list: (warehouseId: string) =>
        request<{ data: QCInspection[] }>(`/wms/qc-inspections?warehouse_id=${encodeURIComponent(warehouseId)}`),
      quarantine: (warehouseId: string) =>
        request<{ data: QuarantineLine[] }>(`/wms/quarantine?warehouse_id=${encodeURIComponent(warehouseId)}`),
      release: (input: QuarantineActionInput) =>
        request<{ data: StockMovement }>("/wms/quarantine/release", {
          method: "POST",
          body: JSON.stringify(input),
        }),
      scrap: (input: QuarantineActionInput) =>
        request<{ data: StockMovement }>("/wms/quarantine/scrap", {
          method: "POST",
          body: JSON.stringify(input),
        }),
    },
    settings: {
      get: () => request<{ data: WMSSettings }>("/wms/settings"),
      update: (requireReleaseApproval: boolean) =>
        request<{ data: WMSSettings }>("/wms/settings", {
          method: "PUT",
          body: JSON.stringify({ require_release_approval: requireReleaseApproval }),
        }),
    },
    defaultLocations: {
      list: (productId?: string) =>
        request<{ data: ProductDefaultLocation[] }>(
          `/wms/default-locations${productId ? `?product_id=${encodeURIComponent(productId)}` : ""}`
        ),
      set: (data: { product_id: string; warehouse_id: string; location_id: string }) =>
        request<{ message: string }>("/wms/default-locations", {
          method: "POST",
          body: JSON.stringify(data),
        }),
      delete: (productId: string, warehouseId: string) =>
        request<{ message: string }>(
          `/wms/default-locations?product_id=${encodeURIComponent(productId)}&warehouse_id=${encodeURIComponent(warehouseId)}`,
          { method: "DELETE" }
        ),
    },
    trace: {
      batch: (id: string) => request<{ data: BatchTrace }>(`/wms/trace/batch/${id}`),
      document: (type: string, id: string) =>
        request<{ data: unknown }>(`/wms/trace/document?type=${encodeURIComponent(type)}&id=${encodeURIComponent(id)}`),
      auditTrail: (entityType: string, entityId: string) =>
        request<{ data: AuditTrailEntry[] }>(
          `/wms/audit-trail?entity_type=${encodeURIComponent(entityType)}&entity_id=${encodeURIComponent(entityId)}`
        ),
    },
    marketplace: {
      import: (data: FormData | ImportMarketplaceOrdersPayload) => {
        const isFormData = typeof FormData !== "undefined" && data instanceof FormData
        return request<ImportMarketplaceResponse>("/wms/marketplace/import", {
          method: "POST",
          body: isFormData ? data : JSON.stringify(data),
        })
      },
      listBatches: (warehouseId?: string) =>
        request<{ data: MarketplaceImportBatch[] }>(
          `/wms/marketplace/batches${warehouseId ? `?warehouse_id=${encodeURIComponent(warehouseId)}` : ""}`
        ),
      listOrders: (params?: { warehouse_id?: string; batch_id?: string; status?: string }) => {
        const searchParams = new URLSearchParams()
        if (params?.warehouse_id) searchParams.set("warehouse_id", params.warehouse_id)
        if (params?.batch_id) searchParams.set("batch_id", params.batch_id)
        if (params?.status) searchParams.set("status", params.status)
        const q = searchParams.toString()
        return request<{ data: MarketplaceOrder[] }>(
          `/wms/marketplace/orders${q ? `?${q}` : ""}`
        )
      },
      getOrder: (id: string) =>
        request<{ data: MarketplaceOrder } | MarketplaceOrder>(`/wms/marketplace/orders/${id}`),
      listSKUMappings: (channel?: string) =>
        request<{ data: MarketplaceSKUMapping[] }>(
          `/wms/marketplace/sku-mappings${channel ? `?channel_name=${encodeURIComponent(channel)}` : ""}`
        ),
      createSKUMapping: (data: CreateSKUMappingPayload) =>
        request<{ data: MarketplaceSKUMapping }>("/wms/marketplace/sku-mappings", {
          method: "POST",
          body: JSON.stringify(data),
        }),
    },
    movements: {
      list: (params?: { product_id?: string; location_id?: string; limit?: number }) => {
        const sp = new URLSearchParams()
        if (params?.product_id) sp.set("product_id", params.product_id)
        if (params?.location_id) sp.set("location_id", params.location_id)
        if (params?.limit) sp.set("limit", String(params.limit))
        const q = sp.toString()
        return request<{ data: StockMovement[] }>(`/wms/movements${q ? `?${q}` : ""}`)
      },
    },
    stock: {
      list: (warehouseId?: string) =>
        request<{ data: StockSummary[] }>(
          `/wms/stock${warehouseId ? `?warehouse_id=${encodeURIComponent(warehouseId)}` : ""}`
        ),
    },
  },
  pos: {
    checkout: (input: {
      items: { product_id: string; qty: number; price: number; discount?: number }[]
      payments: { method: string; amount: number }[]
      tax?: number
      discount?: number
      customer_id?: string
      warehouse_id?: string
      sale_mode?: string
      payment_order_id?: string
    }) =>
      request<{
        order_number: string
        total: number
        subtotal: number
        tax: number
        discount: number
        payment_method: string
        paid_amount: number
        change: number
        sales_order_id?: string
        sales_invoice_id?: string
        created_at: string
        items: POSOrderItem[]
      }>("/pos/checkout", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    orders: (limit?: number) =>
      request<{ data: POSOrder[] }>(`/pos/orders${limit ? `?limit=${limit}` : ""}`),
    order: (id: string) => request<POSOrder>(`/pos/orders/${id}`),
    createPayment: (input: { amount: number; method: string }) =>
      request<POSPaymentCharge>("/pos/payments", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    paymentStatus: (orderId: string) =>
      request<POSPaymentStatus>(`/pos/payments/${encodeURIComponent(orderId)}/status`),
    simulatePayment: (orderId: string) =>
      request<{ status: string }>(`/pos/payments/${encodeURIComponent(orderId)}/simulate`, {
        method: "POST",
      }),
  },
  customers: {
    list: async () => {
      const res = await request<any>("/customers")
      if (Array.isArray(res)) return { data: res as Customer[] }
      if (res && Array.isArray(res.data)) return { data: res.data as Customer[] }
      return { data: [] }
    },
    create: (data: CreateCustomerInput) =>
      request<Customer>("/customers", {
        method: "POST",
        body: JSON.stringify(data),
      }),
  },
  payments: {
    getConfig: (provider: string = "midtrans") =>
      request<TenantPaymentConfigResponse>(`/payments/configs?provider=${encodeURIComponent(provider)}`),
    upsertConfig: (payload: UpsertPaymentConfigPayload) =>
      request<{ message: string; provider: string }>("/payments/configs", {
        method: "POST",
        body: JSON.stringify(payload),
      }),
  },
}
