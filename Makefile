.PHONY: dev infra-up infra-down infra-logs migrate migrate-ocr migrate-amount-check migrate-match migrate-fresh test build sqlc fmt lint ai-worker-build ai-worker-logs

BACKEND_DIR := backend/go-core
PG_URL      := postgres://tayooli:tayooli@localhost:5432/tayooli?sslmode=disable

## Infra
infra-up:
	docker compose up -d

infra-down:
	docker compose down

infra-logs:
	docker compose logs -f

infra-status:
	docker compose ps

## Dev
dev: infra-up
	cd $(BACKEND_DIR) && go run ./cmd/api

## Migrations
migrate:
	@echo "--- Applying migrations ---"
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/001_init_schema.sql
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/002_seed_data.sql
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/003_add_ocr_fields.sql
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/004_amount_numeric.sql
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/005_invoice_amount_check.sql
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/006_purchase_orders_goods_receipts.sql
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/007_invoice_match_columns.sql

migrate-match:
	@echo "--- Applying 3-way match migrations ---"
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/006_purchase_orders_goods_receipts.sql
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/007_invoice_match_columns.sql

migrate-ocr:
	@echo "--- Applying OCR migration ---"
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/003_add_ocr_fields.sql

migrate-amount-check:
	@echo "--- Applying amount check constraint migration ---"
	PGPASSWORD=tayooli psql -h localhost -U tayooli -d tayooli -f $(BACKEND_DIR)/migrations/005_invoice_amount_check.sql

migrate-fresh:
	@echo "--- Resetting database ---"
	PGPASSWORD=tayooli psql -h localhost -U tayooli -c "DROP DATABASE IF EXISTS tayooli; CREATE DATABASE tayooli;"
	$(MAKE) migrate

## Build & Test
build:
	cd $(BACKEND_DIR) && go build -o bin/api ./cmd/api

test:
	cd $(BACKEND_DIR) && go test ./... -v -race -coverprofile=coverage.out

test-short:
	cd $(BACKEND_DIR) && go test ./... -short -race

## Code Quality
fmt:
	cd $(BACKEND_DIR) && gofmt -w .

lint:
	cd $(BACKEND_DIR) && go vet ./...

## sqlc code generation
sqlc:
	cd $(BACKEND_DIR) && sqlc generate

## AI Worker
ai-worker-build:
	docker build -t tayooli-ai-worker ./ai-worker

ai-worker-logs:
	docker compose logs -f ai-worker

.DEFAULT_GOAL := dev
