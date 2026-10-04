# Tayooli ERP — Kafka Topics Reference

Broker: KRaft mode (no ZooKeeper).
All messages use UTF-8 JSON values. Message key is always the **invoice UUID**
(UTF-8 string) to guarantee partition ordering per invoice (see ADR-001).

---

## Topic Index

| Topic | Producer | Consumer | Purpose |
|---|---|---|---|
| `invoice.created` | Go `go-core` | Python `ai-worker` | Triggers OCR pipeline when an invoice is created |
| `invoice.ocr_completed` | Python `ai-worker` | Go `go-core` | Delivers OCR results back to the backend for persistence |
| `invoice.approved` | Go `go-core` | Downstream services | Signals a manual or auto-approval (see ADR-001) |

---

## `invoice.created`

**Producer**: `go-core` — `internal/usecase/invoice/create.go`
**Consumer**: `ai-worker` — consumer group `ai-worker-ocr`
**Kafka key**: `invoice_id` (UUID string)

Published immediately after the invoice row is committed to PostgreSQL.
Publishing is best-effort: a Kafka failure does not roll back the database write
(see ADR-001 for rationale and the transactional outbox upgrade path).

### Schema — `InvoiceCreatedEvent`

Go source: `internal/domain/invoice.go` → `InvoiceCreatedEvent`

| Field | Type | Required | Description |
|---|---|---|---|
| `invoice_id` | string (UUID) | Yes | Primary key of the invoice row |
| `tenant_id` | string (UUID) | Yes | Tenant that owns this invoice; used by the ai-worker for the OCR result payload |
| `amount` | number (float64) | Yes | Invoice amount in the currency specified at creation |
| `status` | string | Yes | Always `"pending"` at creation time |
| `raw_document_bytes` | string (base64) | No | Base64-encoded document content. Omitted when no file was attached at creation |
| `mime_type` | string | No | MIME type of the attached document. Omitted when `raw_document_bytes` is absent. Supported values: `application/pdf`, `image/png`, `image/jpeg`, `image/tiff`, `image/bmp` |

### Example message

```json
{
  "invoice_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "b1a2c3d4-e5f6-7890-abcd-ef1234567890",
  "amount": 1500000.00,
  "status": "pending",
  "raw_document_bytes": "JVBERi0xLjQKJcOkw...",
  "mime_type": "application/pdf"
}
```

### Consumer behavior (ai-worker)

1. Decode UTF-8 value; commit offset and skip on `UnicodeDecodeError`, `JSONDecodeError`, or null value (tombstone).
2. Validate `invoice_id` and `tenant_id` against UUID regex; skip if either is invalid.
3. If `raw_document_bytes` is absent or empty, OCR returns `("", 0.0)` and status is set to `ai_processed` (no document to process is a valid, non-error state).
4. Reject payloads where estimated decoded size exceeds 10 MB.
5. Cap PDF conversion at 20 pages.
6. Publish result to `invoice.ocr_completed`, flush, then commit consumer offset.

---

## `invoice.ocr_completed`

**Producer**: Python `ai-worker` — `ai-worker/main.py`
**Consumer**: Go `go-core` — `internal/infra/kafka/` (OCR result consumer)
**Kafka key**: `invoice_id` (UUID string)
**Producer acks**: `acks=all` (all in-sync replicas must acknowledge before flush returns)

### Schema — `OCRResult`

Python source: `ai-worker/main.py` → `result` dict in the main consumer loop.

| Field | Type | Required | Description |
|---|---|---|---|
| `invoice_id` | string (UUID) | Yes | Invoice being updated; must be validated as UUID by the Go consumer |
| `tenant_id` | string (UUID) | Yes | Tenant that owns the invoice; the Go consumer must use this for the `UpdateOCRResult` call to enforce tenant isolation |
| `confidence_score` | number (float, 0.0–1.0) | Yes | Mean Tesseract word-confidence across all processed pages. `0.0` when OCR failed or no document was provided |
| `extracted_text` | string | Yes | Space-joined OCR text from all pages. Empty string when OCR failed or no document was provided |
| `status` | string | Yes | `"ai_processed"` on success, `"ai_failed"` on any exception during OCR. **The Go consumer must enforce this allowlist and reject any other value.** |

### Example message — successful OCR

```json
{
  "invoice_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "b1a2c3d4-e5f6-7890-abcd-ef1234567890",
  "confidence_score": 0.94,
  "extracted_text": "PT Maju Bersama Invoice No: INV-2026-00042 Total: Rp 1.500.000",
  "status": "ai_processed"
}
```

### Example message — OCR failure

```json
{
  "invoice_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "tenant_id": "b1a2c3d4-e5f6-7890-abcd-ef1234567890",
  "confidence_score": 0.0,
  "extracted_text": "",
  "status": "ai_failed"
}
```

### Consumer behavior (go-core)

1. Validate `invoice_id` and `tenant_id` as UUIDs; discard message and log error if either is invalid.
2. Enforce status allowlist: accept only `"ai_processed"` or `"ai_failed"`. Discard and log any other value.
3. Call `InvoiceRepository.UpdateOCRResult(ctx, tenantID, invoiceID, score, extractedText, status)`.
   The repository sets `SET LOCAL app.current_tenant_id` before the UPDATE, enforcing RLS even on this internal write path.
4. Processing is idempotent: redelivery of the same `invoice_id` overwrites the OCR columns with the same values (last-write-wins).

---

## `invoice.approved`

**Producer**: Go `go-core` — `internal/usecase/invoice/approve.go`
**Consumer**: Downstream services (notification, audit pipeline)
**Kafka key**: `invoice_id` (UUID string)

Documented in ADR-001. Payload is the full `Invoice` JSON object.
Not consumed by the ai-worker.

---

## Operational Notes

### Partition count recommendation

Set `invoice.created` and `invoice.ocr_completed` to the same partition count
(minimum 6 for production).  Because both topics use `invoice_id` as the key,
events for the same invoice always land on the same partition number in both
topics, simplifying ordered processing.

### Consumer group isolation

The ai-worker uses consumer group `ai-worker-ocr` exclusively on `invoice.created`.
The Go backend's OCR result consumer must use a different group ID
(e.g. `go-core-ocr-result`) on `invoice.ocr_completed`.  Never share a consumer
group between the Go backend and the ai-worker.

### Retention policy

| Topic | Recommended retention | Reason |
|---|---|---|
| `invoice.created` | 7 days | Allows ai-worker catch-up after an outage without losing events |
| `invoice.ocr_completed` | 3 days | Short-lived; Go backend persists results to PostgreSQL on consumption |
| `invoice.approved` | 30 days | Audit trail for downstream services |

### Dead-letter handling

Neither the ai-worker nor the current Go consumer implements a dead-letter topic.
Messages that fail UUID validation or status allowlist checks are logged and
discarded (offset committed).  Add a `invoice.ocr_dlq` topic if stricter
observability over rejected messages is required.
