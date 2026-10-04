# ADR-002: Python AI Worker — Kafka Event-Driven OCR Pipeline

Date: 2026-06-27
Status: Accepted

---

## Context

Phase 3 requires automatic text extraction from uploaded invoice documents
(PDF and images) so the platform can pre-fill fields, flag anomalies, and
drive autonomous approval decisions.

Several integration strategies were considered.  The constraints that shaped
this decision:

1. **Language boundary**: OCR in the Tayooli stack requires Tesseract, Pillow,
   and pdf2image — a Python-native ecosystem.  Invoking Tesseract from Go means
   CGo bindings (complex, non-portable) or shelling out to a subprocess (no worker
   pooling, brittle error handling).  A separate Python process eliminates both
   problems.

2. **Blocking vs. async**: OCR on a multi-page PDF at 150 dpi takes 2–15 seconds
   per document depending on page count.  Running OCR synchronously inside the
   Go HTTP handler would hold the connection open for that duration, degrade p99
   latency for all invoice operations, and couple Go availability to OCR
   availability.

3. **Existing Kafka infrastructure**: The Go backend already publishes
   `invoice.created` events to Kafka (ADR-001).  Consuming that topic from
   a separate Python service requires no new protocol or additional broker
   configuration.

4. **gRPC as an alternative**: A synchronous gRPC call from Go to a Python
   FastAPI/gRPC service would give immediate OCR results in the create-invoice
   response.  This was rejected because:
   - It couples invoice-creation availability to AI worker availability.  If the
     AI worker is down or slow, `POST /api/v1/invoices` degrades.
   - It requires managing a gRPC connection pool and service-discovery between
     Go and Python.
   - OCR output is not needed in the create-invoice HTTP response; the frontend
     polls or subscribes to status updates.

---

## Decision

Run OCR in a dedicated Python service (`ai-worker`) that consumes the existing
`invoice.created` Kafka topic, performs Tesseract OCR, and publishes results to
a new `invoice.ocr_completed` topic consumed by the Go backend.

The integration is event-driven (not synchronous gRPC), Python-based
(Tesseract + pdf2image + Pillow), and deployed as an independent process
(`ai-worker/main.py`) with its own Kafka consumer group (`ai-worker-ocr`).

---

## Key Security Decisions

### Status allowlist on the Go consumer side

When the Go backend consumes `invoice.ocr_completed`, it must only accept
`ai_processed` or `ai_failed` as valid `status` values.  Any other string must
be rejected and logged.  This prevents a compromised or misbehaving ai-worker
from setting arbitrary invoice states (e.g. `approved`) through the OCR result
topic.

### UUID validation before processing

The ai-worker validates both `invoice_id` and `tenant_id` against a strict UUID
regex (`[0-9a-f]{8}-...-[0-9a-f]{12}`) before running OCR or publishing results.
Messages that fail this check are logged and the offset committed without
processing.  This prevents malformed or injected identifiers from reaching the
OCR pipeline or the result topic.

### Tombstone (null value) handling

Kafka compaction can produce tombstone records (messages with a null value).
The consumer explicitly checks `msg.value is None` and raises `ValueError("tombstone")`
before attempting JSON decode.  The tombstone is logged and the offset committed
without processing, preventing a `NullPointerException`-equivalent boot loop.

### 10 MB payload cap + 20-page PDF limit

`MAX_OCR_BYTES = 10 * 1024 * 1024` rejects documents whose base64-encoded size
implies more than 10 MB of raw bytes.  `MAX_PDF_PAGES = 20` caps the number of
pages converted by pdf2image.  Both limits prevent decompression-bomb and
memory-exhaustion DoS attacks via crafted documents embedded in Kafka messages.

### MIME type allowlist

Only `image/png`, `image/jpeg`, `image/tiff`, `image/bmp`, and
`application/pdf` are accepted.  All other MIME types are rejected with a
warning log and a no-op OCR result (empty string, confidence 0.0), rather than
attempting to process an unknown format.

### Manual Kafka offset commit

The consumer sets `enable_auto_commit=False` and calls `consumer.commit()` only
after `producer.flush()` succeeds.  This means:

- A process crash after OCR but before flush causes Kafka to redeliver the
  message; OCR re-runs (idempotent outcome — same document, same result).
- A process crash after flush but before commit also causes redelivery and
  re-processing; the Go consumer must be idempotent on `invoice.ocr_completed`
  (last-write-wins on the OCR result columns).
- Malformed messages (bad UTF-8, non-JSON, tombstones) commit the offset
  immediately, so a single bad message never blocks the consumer group.

---

## Consequences

### Positive

- Invoice creation (`POST /api/v1/invoices`) is not blocked by OCR latency or
  AI worker availability.  The endpoint returns in <100 ms regardless of document
  size.
- The Python service can be scaled horizontally by adding consumer instances to
  the `ai-worker-ocr` group; Kafka partitioning handles load distribution.
- Tesseract requires no GPU and no paid API key, keeping the OCR dependency
  entirely self-hosted and cost-free at low volume.
- The ai-worker is independently deployable and restartable without affecting
  the Go backend.

### Negative / Trade-offs

- **Eventual consistency**: after `POST /api/v1/invoices` returns, the invoice
  is in `pending` status with null OCR fields.  Clients must poll
  `GET /api/v1/invoices/{id}` or subscribe to a WebSocket/SSE channel to detect
  when status transitions to `ai_processed` or `ai_failed`.
- **Tesseract accuracy ceiling**: Tesseract 5 achieves ~85–92% character
  accuracy on clean invoices; structured-document models (LayoutLMv3, Donut)
  achieve higher accuracy, especially on scanned or skewed documents.  Upgrading
  to a model-based extractor is the recommended next step for production accuracy
  targets above 95%.
- **Kafka as a dependency**: the OCR pipeline is unavailable if Kafka is down.
  Invoices will stay in `pending` status until Kafka recovers and the consumer
  group catches up.  The backlog is bounded by the Kafka topic retention setting.
- **No strict at-least-once guarantee on the producer side**: the Go backend
  publishes `invoice.created` best-effort (ADR-001).  If Kafka is unavailable
  at creation time the event is lost; the invoice will never receive OCR
  processing unless a re-trigger mechanism (e.g. admin re-queue endpoint) is
  added.  The transactional outbox pattern is the recommended upgrade path.
- **ai-worker is single-threaded per process**: the consumer loop processes one
  message at a time.  Horizontal scaling requires multiple process replicas
  (and sufficient Kafka partitions) rather than threading within one process.

---

## References

- `ai-worker/main.py` — consumer loop, OCR pipeline, security guards
- `backend/go-core/internal/domain/invoice.go` — `InvoiceCreatedEvent` struct,
  `UpdateOCRResult` repository method
- `docs/adr/001-go-backend-hexagonal-jwt-rls.md` — Kafka publishing strategy
  (best-effort, invoice UUID as message key)
- `docs/kafka-topics.md` — topic schemas and message contracts
