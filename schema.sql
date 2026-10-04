-- PostgreSQL Database DDL
-- Core Tables: tenants, users, invoices, immutable_audit_logs
-- Multi-tenancy isolation using Row-Level Security (RLS) and custom session variables

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- -------------------------------------------------------------
-- 1. Tenants Table
-- -------------------------------------------------------------
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    domain VARCHAR(100) UNIQUE,
    subscription_tier VARCHAR(50) DEFAULT 'STANDARD' CHECK (subscription_tier IN ('STANDARD', 'ENTERPRISE', 'COMPLIANCE_PRO')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enable Row-Level Security
ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;

-- -------------------------------------------------------------
-- 2. Users Table
-- -------------------------------------------------------------
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'Staff' CHECK (role IN ('System Admin', 'CFO', 'Treasury Manager', 'Accountant', 'Auditor')),
    full_name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_email UNIQUE(tenant_id, email)
);

-- Enable Row-Level Security
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- -------------------------------------------------------------
-- 3. Invoices Table
-- -------------------------------------------------------------
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    invoice_number VARCHAR(100) NOT NULL,
    client_name VARCHAR(255) NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'APPROVED', 'RECONCILED', 'FLAGGED', 'REJECTED')),
    due_date TIMESTAMPTZ NOT NULL,
    ocr_confidence NUMERIC(5, 2), -- Percentage e.g. 98.50
    gl_account VARCHAR(100),
    anomaly_score NUMERIC(5, 2),  -- Sensitivity score
    payload_metadata JSONB NOT NULL DEFAULT '{}'::jsonb, -- Flexible schema-less storage
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_invoice UNIQUE(tenant_id, invoice_number)
);

-- Enable Row-Level Security
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;

-- -------------------------------------------------------------
-- 4. Immutable Audit Logs Table
-- -------------------------------------------------------------
-- Designed as a hash-chained audit ledger. Each log's current_hash
-- is a SHA-256 hash computed over: (prev_hash, user_id, event_type, action_details, created_at).
CREATE TABLE immutable_audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(100) NOT NULL,
    action_details JSONB NOT NULL,
    prev_hash VARCHAR(64) NOT NULL,  -- SHA-256 hash of the prior block
    current_hash VARCHAR(64) NOT NULL, -- SHA-256 hash of this entire block
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enable Row-Level Security
ALTER TABLE immutable_audit_logs ENABLE ROW LEVEL SECURITY;

-- -------------------------------------------------------------
-- High-Performance Indexes
-- -------------------------------------------------------------
-- General performance lookup indexes for tenant filtering
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_invoices_tenant_id ON invoices(tenant_id);
CREATE INDEX idx_audit_logs_tenant_id ON immutable_audit_logs(tenant_id);

-- Performance lookups for invoice status filtering and sorting
CREATE INDEX idx_invoices_status_due ON invoices(tenant_id, status, due_date DESC);
CREATE INDEX idx_invoices_search_trgm ON invoices(tenant_id, client_name, invoice_number);

-- Cryptographic chain lookup optimization
CREATE UNIQUE INDEX idx_audit_logs_current_hash ON immutable_audit_logs(current_hash);
CREATE INDEX idx_audit_logs_prev_hash ON immutable_audit_logs(prev_hash);

-- -------------------------------------------------------------
-- Row-Level Security (RLS) Policies
-- -------------------------------------------------------------
-- All policies utilize a session-level configuration 'app.current_tenant_id'.
-- In production, the backend server sets this variable inside a transaction block
-- prior to running any query:
-- SET LOCAL app.current_tenant_id = 'c7a2b53b-e0f3-42e5-8f6a-f8888b1f22e5';

-- Tenant table policy (Users can only see their own tenant record)
CREATE POLICY tenant_isolation_policy ON tenants
    FOR ALL
    USING (id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- Users table policy (Users can only see users inside their own tenant space)
CREATE POLICY user_isolation_policy ON users
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- Invoices table policy (Tenant-isolated CRUD actions)
CREATE POLICY invoice_isolation_policy ON invoices
    FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- Audit log table policy (Only SELECT operations are permitted; rows are append-only)
CREATE POLICY audit_log_isolation_policy ON immutable_audit_logs
    FOR SELECT
    USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- Absolute lock to prevent mutations or deletions on immutable logs
CREATE POLICY audit_log_no_update_policy ON immutable_audit_logs
    FOR UPDATE
    USING (FALSE);

CREATE POLICY audit_log_no_delete_policy ON immutable_audit_logs
    FOR DELETE
    USING (FALSE);

-- Append-only trigger for audit logs (Only current_tenant_id can insert)
CREATE POLICY audit_log_insert_policy ON immutable_audit_logs
    FOR INSERT
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- -------------------------------------------------------------
-- 5. Customers Table
-- -------------------------------------------------------------
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE customers ENABLE ROW LEVEL SECURITY;

-- -------------------------------------------------------------
-- 6. Sales Orders Table
-- -------------------------------------------------------------
CREATE TABLE sales_orders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    order_number VARCHAR(100) NOT NULL,
    total_amount NUMERIC(15, 2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING' CHECK (status IN ('PENDING', 'CONFIRMED', 'SHIPPED', 'CANCELLED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_sales_order UNIQUE(tenant_id, order_number)
);

ALTER TABLE sales_orders ENABLE ROW LEVEL SECURITY;

-- -------------------------------------------------------------
-- 7. Sales Invoices Table
-- -------------------------------------------------------------
CREATE TABLE sales_invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sales_order_id UUID REFERENCES sales_orders(id) ON DELETE SET NULL,
    invoice_number VARCHAR(100) NOT NULL,
    amount NUMERIC(15, 2) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'UNPAID' CHECK (status IN ('UNPAID', 'PARTIAL', 'PAID', 'CANCELLED')),
    due_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_sales_invoice UNIQUE(tenant_id, invoice_number)
);

ALTER TABLE sales_invoices ENABLE ROW LEVEL SECURITY;

-- High-Performance Indexes for O2C
CREATE INDEX idx_customers_tenant_id ON customers(tenant_id);
CREATE INDEX idx_sales_orders_tenant_id ON sales_orders(tenant_id);
CREATE INDEX idx_sales_orders_customer_id ON sales_orders(customer_id);
CREATE INDEX idx_sales_invoices_tenant_id ON sales_invoices(tenant_id);
CREATE INDEX idx_sales_invoices_sales_order_id ON sales_invoices(sales_order_id);

-- RLS Policies for O2C
CREATE POLICY customer_isolation_policy ON customers
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY sales_order_isolation_policy ON sales_orders
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE POLICY sales_invoice_isolation_policy ON sales_invoices
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

-- -------------------------------------------------------------
-- 8. Accounts Table (Accounting Foundation)
-- -------------------------------------------------------------
CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- Asset, Liability, Equity, Revenue, Expense
    balance NUMERIC(15, 4) DEFAULT 0.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, code)
);

ALTER TABLE accounts ENABLE ROW LEVEL SECURITY;
CREATE POLICY accounts_tenant_policy ON accounts
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE TABLE IF NOT EXISTS journal_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    account_id UUID NOT NULL REFERENCES accounts(id),
    transaction_date DATE NOT NULL,
    description TEXT,
    debit_amount NUMERIC(15, 4) DEFAULT 0.0000,
    credit_amount NUMERIC(15, 4) DEFAULT 0.0000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE journal_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY journal_entries_tenant_policy ON journal_entries
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- -------------------------------------------------------------
-- 9. Approvals Workflow
-- -------------------------------------------------------------
CREATE TABLE approval_workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    steps JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE approval_workflows ENABLE ROW LEVEL SECURITY;
CREATE POLICY approval_workflows_tenant_isolation_policy ON approval_workflows
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

CREATE TABLE approval_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_id UUID NOT NULL REFERENCES approval_workflows(id),
    target_type VARCHAR(255) NOT NULL,
    target_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    current_step_index INT NOT NULL DEFAULT 0,
    requested_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE approval_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY approval_requests_tenant_isolation_policy ON approval_requests
    FOR ALL USING (tenant_id = current_setting('app.current_tenant_id')::UUID);

-- -------------------------------------------------------------
-- 10. Products & Inventory
-- -------------------------------------------------------------
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sku VARCHAR(100) NOT NULL,
    price NUMERIC(15, 2) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_tenant_sku UNIQUE(tenant_id, sku)
);

ALTER TABLE products ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_products_tenant_id ON products(tenant_id);

CREATE POLICY product_isolation_policy ON products
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    quantity NUMERIC(15, 2) NOT NULL DEFAULT 0,
    warehouse_location VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE inventory ENABLE ROW LEVEL SECURITY;

CREATE INDEX idx_inventory_tenant_id ON inventory(tenant_id);
CREATE INDEX idx_inventory_product_id ON inventory(product_id);

CREATE POLICY inventory_isolation_policy ON inventory
    FOR ALL USING (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant_id', true), '')::uuid);

