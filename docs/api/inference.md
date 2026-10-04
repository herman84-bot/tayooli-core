# AI Serving — API Reference

Tayooli's AI Serving module provides **anomaly detection** and
**suggested GL account** mapping for invoices, exposed to the Go backend
through a small FastAPI HTTP service.

```
┌─────────────┐    HTTP/JSON     ┌──────────────────┐
│  Go Backend │ ───────────────► │ FastAPI Service  │
│  (Client)   │  /inference/*    │  (ai-worker)     │
└─────────────┘                  └────────┬─────────┘
                                          │ background thread
                                          ▼
                                   OCR consumer
                                   (invoice.created → OCR → invoice.ocr_completed)
```

The Go client lives in `backend/go-core/internal/infra/ai/ai_client.go`
(consumed by `internal/usecase/ai` and exposed via the Go handlers
`internal/handler/inference_handler.go`).

## Service endpoints (Python — `ai-worker/service.py`)

### `POST /api/v1/inference/ingest`

Submits an invoice for AI analysis. Returns **202 Accepted** with a `job_id`
(the invoice id) and a queued status. Inference runs asynchronously on a
background thread.

**Request body**

| Field | Type | Required | Description |
|---|---|---|---|
| `invoice_id` | string (UUID) | Yes | Invoice being analyzed |
| `tenant_id` | string (UUID) | Yes | Tenant that owns the invoice (tenant isolation) |
| `amount` | string | No | Decimal string, up to 16 integer digits + 4 decimals |
| `vendor_id` | string (UUID) | No | Vendor associated with the invoice |
| `extracted_text` | string | No | OCR-extracted document text |

**Response `202`**

```json
{ "job_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6", "status": "queued" }
```

**Errors:** `400` on invalid UUIDs or an invalid `amount` format.

### `GET /api/v1/inference/status/{invoice_id}?tenant_id=<uuid>`

Polls the inference result. Returns **404** while the job is unknown or when
the `tenant_id` does not match the job owner (no information leak).

**Response `200`**

```json
{
  "invoice_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "status": "completed",
  "anomaly_score": 62.5,
  "suggested_gl_account": "5102 — Utilities",
  "error": null
}
```

`status` is one of `queued` | `processing` | `completed` | `failed`.
`anomaly_score` is a percentage in `[0, 100]`; values ≥ 50 are flagged as
anomalies. `error` is present only when `status` is `failed`.

### `GET /health`

Liveness probe used by Docker/K8s: `{ "status": "ok", "service": "ai-serving" }`.

## Go proxy endpoints (`backend/go-core`)

| Method | Path | Description |
|---|---|---|
| POST | `/api/v1/inference/ingest` | Rate-limited; forwards to the Python service, returns 202 |
| GET | `/api/v1/inference/status/{id}` | Rate-limited; forwards to the Python service |

Both require the tenant cookie (`tayooli_auth`) via `TenantMiddleware`.

## Environment variables

| Variable | Service | Description |
|---|---|---|
| `AI_SERVING_URL` | Go backend | Base URL of the FastAPI service (default `http://localhost:8000`) |
| `AI_BASE_URL` | Go backend | Optional; when set, invoice creation also triggers AI analysis automatically |
| `KAFKA_BROKERS` | ai-worker | Comma-separated broker list (default `localhost:9092`) |

## Inference heuristics (v1)

Pure-python logic in `ai-worker/inference.py`:

- **Anomaly score** — combination of amount magnitude (Rp ≥ 1e9 → +60,
  ≥ 1e8 → +35, ≥ 1e7 → +15), unparseable amount (+15), missing OCR text
  (+20), missing vendor (+10), and discrepancy keywords in the text (+40).
  Clamped to `[0, 100]`, flagged when ≥ 50.
- **GL suggestion** — keyword → account mapping against the extracted text
  (e.g. `listrik`/`internet` → `5102 — Utilities`, `gaji` → `5101 — Salaries
  & Wages`), falling back to `5199 — Miscellaneous Expense`.

These functions are the single swap point for a real ML model later.

## Running the tests

```bash
python3 -m py_compile ai-worker/*.py
python3 ai-worker/test_ai_serving.py   # self-running; also pytest-compatible
```
