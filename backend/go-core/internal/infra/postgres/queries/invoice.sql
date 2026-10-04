-- name: ListInvoicesByTenant :many
SELECT id, tenant_id, vendor_id, invoice_number, amount, currency, status,
       ai_confidence_score, due_date, created_at, updated_at
FROM invoices
WHERE tenant_id = $1
ORDER BY created_at DESC;

-- name: GetInvoiceByID :one
SELECT id, tenant_id, vendor_id, invoice_number, amount, currency, status,
       ai_confidence_score, due_date, created_at, updated_at
FROM invoices
WHERE id = $1 AND tenant_id = $2;

-- name: CreateInvoice :one
INSERT INTO invoices (id, tenant_id, vendor_id, invoice_number, amount, currency, status, due_date)
VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)
RETURNING *;

-- name: ApproveInvoice :one
UPDATE invoices
SET status = 'approved', updated_at = NOW()
WHERE id = $1 AND tenant_id = $2
RETURNING *;
