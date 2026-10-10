package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	accountingDomain "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/domain"
	accountingKafka "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/infra/kafka"
	accountingPostgres "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/infra/postgres"
	accountingUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/accounting/usecase"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/config"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/handler"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/kafka"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/mailer"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/infra/postgres"
	tenantMiddleware "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/middleware"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/migrations"
	approvalUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/approval"
	authUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/auth"
	customerUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/customer"
	dashboardUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/dashboard"
	grUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/gr"
	invoiceUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/invoice"
	matchUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/match"
	paymentUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/payment"
	poUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/po"
	posUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pos"
	posPaymentUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/pospayment"
	productUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/product"
	salesInvoiceUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/sales_invoice"
	salesOrderUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/sales_order"
	teamUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/team"
	vendorUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/vendor"
	wmsUC "github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/usecase/wms"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, _ *http.Request, status int, message string) {
	respondJSON(w, status, map[string]any{"error": message})
}

func main() {
	// Structured logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	if err := godotenv.Load(); err != nil {
		log.Info().Msg("no .env file found, reading from environment")
	}

	// Config validation — fail fast with clear errors
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("config validation failed")
	}

	// Sentry error tracking (optional — only when SENTRY_DSN is set)
	if cfg.SentryDSN != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              cfg.SentryDSN,
			Environment:      cfg.AppEnv,
			TracesSampleRate: 0.1,
		}); err != nil {
			log.Fatal().Err(err).Msg("sentry init failed")
		}
		defer sentry.Flush(2 * time.Second)
		log.Info().Str("env", cfg.AppEnv).Msg("Sentry error tracking enabled")
	}

	// Database
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to open database connection")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}

	// Limit connection pool to avoid exhausting Postgres max_connections.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	log.Info().Msg("connected to PostgreSQL")

	// Auto-run embedded migrations on startup
	log.Info().Msg("running database migrations...")
	if err := migrations.Run(context.Background(), db); err != nil {
		log.Fatal().Err(err).Msg("database migrations failed")
	}
	log.Info().Msg("database migrations completed successfully")

	// Kafka Producer: Decoupled with graceful fallback to no-op producer
	var producer kafka.Producer
	kafkaConfigured := false

	if cfg.KafkaBrokers != "" {
		brokers := strings.Split(cfg.KafkaBrokers, ",")
		for _, b := range brokers {
			b = strings.TrimSpace(b)
			if b == "" {
				continue
			}
			conn, dialErr := net.DialTimeout("tcp", b, 2*time.Second)
			if dialErr == nil {
				conn.Close()
				kafkaConfigured = true
				break
			}
		}

		if kafkaConfigured {
			producer = kafka.NewEventProducerWithBrokers(cfg.KafkaBrokers)
			log.Info().Str("brokers", cfg.KafkaBrokers).Msg("connected to Kafka broker")
		} else {
			log.Warn().Str("brokers", cfg.KafkaBrokers).Msg("connecting to Kafka failed, falling back to no-op event producer")
			producer = kafka.NewNoOpProducer()
		}
	} else {
		log.Info().Msg("KAFKA_BROKERS not configured, using no-op event producer")
		producer = kafka.NewNoOpProducer()
	}

	defer func() {
		if err := producer.Close(); err != nil {
			log.Error().Err(err).Msg("event producer close error")
		}
	}()

	// Wire layers: repo → usecase → handler
	invoiceRepo := postgres.NewInvoiceRepo(db)
	invoiceFacade := invoiceUC.NewFacade(invoiceRepo, producer)
	invoiceHandler := handler.NewInvoiceHandler(invoiceFacade)

	// 3-way match engine
	matchRepo := postgres.NewMatchRepo(db)
	tolerancePctStr := os.Getenv("MATCH_AMOUNT_TOLERANCE_PCT")
	if tolerancePctStr == "" {
		tolerancePctStr = "2.0"
	}
	tolerancePct, err := decimal.NewFromString(tolerancePctStr)
	if err != nil || tolerancePct.LessThan(decimal.Zero) || tolerancePct.GreaterThan(decimal.NewFromInt(10)) {
		log.Fatal().Str("MATCH_AMOUNT_TOLERANCE_PCT", tolerancePctStr).Msg("invalid tolerance: must be 0.0–10.0")
	}
	matchEngine := matchUC.NewUsecase(invoiceRepo, matchRepo, producer, tolerancePct)

	// PO & GR
	poRepo := postgres.NewPORepo(db)
	poUsecase := poUC.New(poRepo)
	poHandler := handler.NewPOHandler(poUsecase)

	grRepo := postgres.NewGRRepo(db)
	grUsecase := grUC.New(grRepo, poRepo)
	grHandler := handler.NewGRHandler(grUsecase)

	// Payment Orders
	paymentOrderRepo := postgres.NewPaymentOrderRepo(db)
	auditLogRepo := postgres.NewAuditLogRepo(db)
	paymentOrderFacade := paymentUC.NewFacade(paymentOrderRepo, invoiceRepo, producer, auditLogRepo)
	paymentOrderHandler := handler.NewPaymentOrderHandler(paymentOrderFacade)

	// Vendors
	vendorRepo := postgres.NewVendorRepo(db)
	vendorFacade := vendorUC.New(vendorRepo, auditLogRepo)
	vendorHandler := handler.NewVendorHandler(vendorFacade)

	// Products & Inventory
	productRepo := postgres.NewProductRepo(db)
	inventoryRepo := postgres.NewInventoryRepo(db)
	productUsecase := productUC.New(productRepo, inventoryRepo)
	productHandler := handler.NewProductHandler(productUsecase)

	// WMS Module
	wmsRepo := postgres.NewWMSRepo(db)
	wmsUsecase := wmsUC.New(wmsRepo)
	wmsHandler := handler.NewWMSHandler(wmsUsecase)

	// Payment Gateways (ADR-008 per-tenant provider config & transactions)
	paymentGatewayRepo := postgres.NewPaymentGatewayRepo(db)
	paymentGatewayHandler := handler.NewPaymentGatewayHandler(paymentGatewayRepo)
	// Midtrans webhook receiver (public route — signature checked inside).
	// factory is nil: the real SDK client is built per tenant, BYO credentials.
	paymentWebhookHandler := handler.NewPaymentWebhookHandler(paymentGatewayRepo, nil)

	// O2C
	customerRepo := postgres.NewCustomerRepo(db)
	customerUsecase := customerUC.New(customerRepo)
	customerHandler := handler.NewCustomerHandler(customerUsecase)

	salesOrderRepo := postgres.NewSalesOrderRepo(db)
	salesOrderUsecase := salesOrderUC.NewWithDeps(salesOrderRepo, customerRepo)
	salesOrderHandler := handler.NewSalesOrderHandler(salesOrderUsecase)

	salesInvoiceRepo := postgres.NewSalesInvoiceRepo(db)
	salesInvoiceUsecase := salesInvoiceUC.NewWithDeps(salesInvoiceRepo, salesOrderRepo, producer)
	salesInvoiceHandler := handler.NewSalesInvoiceHandler(salesInvoiceUsecase)

	// POS (Point of Sale)
	posRepo := postgres.NewPOSRepo(db)
	posUsecase := posUC.New(posRepo, productRepo, inventoryRepo, wmsRepo, customerRepo, salesOrderRepo, salesInvoiceRepo)
	// Non-cash (QRIS) payments for POS sales: demo mode until a tenant adds
	// their own gateway credentials (ADR-008 Model B).
	posPaymentUsecase := posPaymentUC.New(paymentGatewayRepo, nil, cfg.AppURL)
	posHandler := handler.NewPOSHandlerWithPayments(posUsecase, posPaymentUsecase)

	// Accounting Module
	accountRepo := accountingPostgres.NewAccountRepository(db)
	journalEntryRepo := accountingPostgres.NewJournalEntryRepository(db)
	accountingUsecase := accountingUC.NewJournalEntryUseCase(journalEntryRepo, accountRepo)

	// Dashboard Analytics
	dashboardRepo := postgres.NewDashboardRepo(db)
	dashboardUsecase := dashboardUC.New(dashboardRepo)
	dashboardHandler := handler.NewDashboardHandler(dashboardUsecase)

	// Auth
	userRepo := postgres.NewUserRepo(db)
	appURL := cfg.AppURL
	if appURL == "" {
		appURL = "https://tayooli.my.id"
	}
	emailMailer := mailer.New(
		cfg.BrevoAPIKey,
		cfg.BrevoSenderEmail,
		cfg.BrevoSenderName,
		cfg.SMTPHost,
		cfg.SMTPPort,
		cfg.SMTPUser,
		cfg.SMTPPass,
		cfg.SMTPFrom,
		appURL,
	)
	authUsecase := authUC.New(userRepo, cfg.JWTSecret, time.Hour, nil, emailMailer)
	revokedTokenRepo := postgres.NewRevokedTokenRepo(db)
	tenantMiddleware.SetRevocationChecker(revokedTokenRepo)
	authHandler := handler.NewAuthHandler(authUsecase).WithTokenRevoker(revokedTokenRepo)

	// Team Management
	teamUsecase := teamUC.New(userRepo).WithInviter(userRepo, emailMailer)
	teamHandler := handler.NewTeamHandler(teamUsecase)

	// Approvals
	approvalRepo := postgres.NewApprovalRepository(db)
	approvalUsecase := approvalUC.New(approvalRepo, auditLogRepo)
	approvalHandler := handler.NewApprovalHandler(approvalUsecase)

	// Main context — cancelled on SIGINT/SIGTERM to trigger graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Kafka Consumers: Only started when Kafka is configured and reachable
	if kafkaConfigured {
		kafkaBrokers := strings.Split(cfg.KafkaBrokers, ",")
		validOCRStatuses := map[string]bool{
			"ai_processed": true,
			"ai_failed":    true,
		}

		ocrConsumer := kafka.NewOCRConsumer(kafkaBrokers, func(ctx context.Context, result kafka.OCRResult) error {
			invoiceID, err := uuid.Parse(result.InvoiceID)
			if err != nil {
				return fmt.Errorf("unrecoverable: invalid invoice_id %q: %w", result.InvoiceID, domain.ErrInvalidInput)
			}
			tenantID, err := uuid.Parse(result.TenantID)
			if err != nil {
				return fmt.Errorf("unrecoverable: invalid tenant_id %q: %w", result.TenantID, domain.ErrInvalidInput)
			}

			if !validOCRStatuses[result.Status] {
				return fmt.Errorf("unrecoverable: invalid OCR status %q: %w", result.Status, domain.ErrInvalidInput)
			}

			score := result.ConfidenceScore
			if score < 0.0 {
				score = 0.0
			}
			if score > 1.0 {
				score = 1.0
			}

			if err := invoiceFacade.HandleOCRResult(ctx, invoiceID, tenantID, score, result.ExtractedText, result.Status); err != nil {
				return err
			}
			if result.Status == "ai_processed" {
				if matchErr := matchEngine.Execute(ctx, invoiceID, tenantID); matchErr != nil {
					log.Error().Err(matchErr).Str("invoice_id", result.InvoiceID).Msg("3-way match failed")
				}
			}
			return nil
		})
		defer func() {
			if err := ocrConsumer.Close(); err != nil {
				log.Error().Err(err).Msg("kafka consumer close error")
			}
		}()
		go ocrConsumer.Run(ctx)

		accountingConsumer := accountingKafka.NewAccountingConsumer(kafkaBrokers, accountingUsecase)
		defer func() {
			if err := accountingConsumer.Close(); err != nil {
				log.Error().Err(err).Msg("accounting consumer close error")
			}
		}()
		go accountingConsumer.Run(ctx)
		log.Info().Msg("started Kafka consumers (OCR, accounting)")
	} else {
		log.Info().Msg("Kafka is not configured or reachable; skipping Kafka consumers (OCR, accounting)")
	}

	// Rate limiter for sensitive auth mutations (login, register, password reset):
	// 10 req/min (burst 10) to guard against bcrypt CPU exhaustion and brute-force DoS.
	authRateLimiter := tenantMiddleware.NewTokenBucketLimiter(10.0/60.0, 10)

	// Router
	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(15 * time.Second))
	r.Use(tenantMiddleware.RequestID)
	r.Use(tenantMiddleware.MetricsMiddleware)
	r.Use(tenantMiddleware.CORSEnv())

	// Prometheus metrics endpoint
	r.Handle("/metrics", tenantMiddleware.MetricsHandler())

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]string)

		// Database
		if err := db.PingContext(r.Context()); err != nil {
			checks["database"] = "unhealthy: " + err.Error()
		} else {
			checks["database"] = "healthy"
		}

		// Kafka check
		if cfg.KafkaBrokers != "" {
			kafkaHealthy := false
			for _, broker := range strings.Split(cfg.KafkaBrokers, ",") {
				broker = strings.TrimSpace(broker)
				if broker == "" {
					continue
				}
				conn, dialErr := net.DialTimeout("tcp", broker, 2*time.Second)
				if dialErr == nil {
					conn.Close()
					kafkaHealthy = true
					break
				}
			}
			if kafkaHealthy {
				checks["kafka"] = "healthy"
			} else {
				checks["kafka"] = "unavailable: all brokers unreachable"
			}
		} else {
			checks["kafka"] = "disabled"
		}

		// Redis (optional — only when REDIS_URL is set)
		if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
			redisAddr := redisURL
			redisAddr = strings.TrimPrefix(redisAddr, "redis://")
			if idx := strings.Index(redisAddr, "/"); idx != -1 {
				redisAddr = redisAddr[:idx]
			}
			conn, err := net.DialTimeout("tcp", redisAddr, 2*time.Second)
			if err != nil {
				checks["redis"] = "unavailable: " + err.Error()
			} else {
				conn.Close()
				checks["redis"] = "healthy"
			}
		}

		status := "ok"
		httpStatus := http.StatusOK
		if checks["database"] != "healthy" {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": status,
			"checks": checks,
		})
	})

	// /health/ready — K8s readiness probe. Returns 503 if ANY required service is down.
	r.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		checks := make(map[string]string)
		allHealthy := true

		// Database
		if err := db.PingContext(r.Context()); err != nil {
			checks["database"] = "unhealthy: " + err.Error()
			allHealthy = false
		} else {
			checks["database"] = "healthy"
		}

		// Kafka check (only enforces readiness if Kafka is configured)
		if cfg.KafkaBrokers != "" {
			kafkaHealthy := false
			for _, broker := range strings.Split(cfg.KafkaBrokers, ",") {
				broker = strings.TrimSpace(broker)
				if broker == "" {
					continue
				}
				conn, dialErr := net.DialTimeout("tcp", broker, 2*time.Second)
				if dialErr == nil {
					conn.Close()
					kafkaHealthy = true
					break
				}
			}
			if kafkaHealthy {
				checks["kafka"] = "healthy"
			} else {
				checks["kafka"] = "unavailable: all brokers unreachable"
				allHealthy = false
			}
		} else {
			checks["kafka"] = "disabled"
		}

		// Redis (optional — only when REDIS_URL is set)
		if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
			redisAddr := redisURL
			redisAddr = strings.TrimPrefix(redisAddr, "redis://")
			if idx := strings.Index(redisAddr, "/"); idx != -1 {
				redisAddr = redisAddr[:idx]
			}
			conn, err := net.DialTimeout("tcp", redisAddr, 2*time.Second)
			if err != nil {
				checks["redis"] = "unavailable: " + err.Error()
				allHealthy = false
			} else {
				conn.Close()
				checks["redis"] = "healthy"
			}
		}

		status := "ok"
		httpStatus := http.StatusOK
		if !allHealthy {
			status = "not_ready"
			httpStatus = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": status,
			"checks": checks,
		})
	})

	// Auth routes - public (login, register, verify) and protected (me, logout)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.With(authRateLimiter.Middleware).Post("/login", authHandler.Login)
		r.With(authRateLimiter.Middleware).Post("/register", authHandler.Register)
		r.Get("/verify-email", authHandler.VerifyEmail)
		r.With(authRateLimiter.Middleware).Post("/resend-verification", authHandler.ResendVerification)
		r.With(authRateLimiter.Middleware).Post("/forgot-password", authHandler.ForgotPassword)
		r.With(authRateLimiter.Middleware).Post("/reset-password", authHandler.ResetPassword)
		// Logout is public by design: it must clear a dead/expired cookie
		// that TenantMiddleware would 401. It verifies the token itself
		// before revoking.
		r.With(authRateLimiter.Middleware).Post("/logout", authHandler.Logout)

		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.TenantMiddleware)
			r.Get("/me", authHandler.Me)
		})
	})

	// Payment gateway webhooks - PUBLIC by design.
	//
	// The gateway has no user session, so these routes must stay outside
	// TenantMiddleware (a static sibling of /api/v1 wins over the /api/v1
	// subtree, same as /api/v1/auth above). The tenant is derived from the
	// order_id embedded in the callback and every request is authenticated by
	// its SHA512 signature plus a server-to-server status re-check, so an
	// unauthenticated caller cannot mark anything paid.
	r.Route("/api/v1/webhooks", func(r chi.Router) {
		r.Post("/midtrans", paymentWebhookHandler.HandleMidtransWebhook)
	})

	// Core API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(tenantMiddleware.TenantMiddleware)

		// Onboarding / Workspace setup (owner or admin only)
		r.With(tenantMiddleware.RequireRole("owner", "admin")).Post("/workspaces", authHandler.CreateWorkspace)

		// Dashboard Analytics
		r.Get("/dashboard/summary", dashboardHandler.GetSummary)

		// Invoices (AP)
		r.Get("/invoices", invoiceHandler.ListInvoices)
		r.Get("/invoices/{id}", invoiceHandler.GetInvoice)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "accountant"))
			r.Post("/invoices", invoiceHandler.CreateInvoice)
		})
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "approver"))
			r.Post("/invoices/{id}/approve", invoiceHandler.ApproveInvoice)
			r.Post("/invoices/{id}/reject", invoiceHandler.RejectInvoice)
		})
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin"))
			r.Post("/invoices/{id}/requeue", invoiceHandler.RequeueInvoice)
		})

		// Purchase Orders & Goods Receipts
		r.Get("/purchase-orders", poHandler.ListPOs)
		r.Get("/purchase-orders/{id}", poHandler.GetPO)
		r.Get("/goods-receipts", grHandler.ListGRs)
		r.Get("/goods-receipts/{id}", grHandler.GetGR)

		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "purchaser"))
			r.Post("/purchase-orders", poHandler.CreatePO)
		})
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "warehouse"))
			r.Post("/goods-receipts", grHandler.CreateGR)
		})

		// Payment Orders
		r.Get("/payment-orders", paymentOrderHandler.ListPaymentOrders)
		r.Get("/payment-orders/{id}", paymentOrderHandler.GetPaymentOrder)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "treasury"))
			r.Post("/payment-orders", paymentOrderHandler.CreatePaymentOrder)
		})
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "cfo"))
			r.Post("/payment-orders/{id}/approve", paymentOrderHandler.ApprovePaymentOrder)
		})
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "treasury"))
			r.Post("/payment-orders/{id}/pay", paymentOrderHandler.PayPaymentOrder)
		})
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "cfo", "treasury"))
			r.Post("/payment-orders/{id}/reject", paymentOrderHandler.RejectPaymentOrder)
		})

		// Vendors
		r.Get("/vendors", vendorHandler.ListVendors)
		r.Get("/vendors/{id}", vendorHandler.GetVendor)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "accountant"))
			r.Post("/vendors", vendorHandler.CreateVendor)
			r.Put("/vendors/{id}", vendorHandler.UpdateVendor)
		})
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin"))
			r.Delete("/vendors/{id}", vendorHandler.DeleteVendor)
		})
		r.Post("/vendors/{id}/rate", vendorHandler.RateVendor)

		// O2C (Customers, Sales Orders, Sales Invoices)
		r.Get("/customers", customerHandler.List)
		r.Get("/customers/{id}", customerHandler.Get)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "accountant"))
			r.Post("/customers", customerHandler.Create)
		})

		r.Get("/sales-orders", salesOrderHandler.List)
		r.Get("/sales-orders/{id}", salesOrderHandler.Get)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "accountant"))
			r.Post("/sales-orders", salesOrderHandler.Create)
		})

		r.Get("/sales-invoices", salesInvoiceHandler.List)
		r.Get("/sales-invoices/{id}", salesInvoiceHandler.Get)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "accountant"))
			r.Post("/sales-invoices", salesInvoiceHandler.Create)
		})

		// Products & Inventory
		r.Get("/products", productHandler.ListProducts)
		r.Get("/products/{id}", productHandler.GetProduct)
		r.Group(func(r chi.Router) {
			// "owner" is the role every self-registered tenant gets
			// (usecase/auth Register); without it new tenants get 403 on products.
			r.Use(tenantMiddleware.RequireRole("owner", "admin", "accountant"))
			r.Post("/products", productHandler.CreateProduct)
			r.Put("/products/{id}", productHandler.UpdateProduct)
			r.Patch("/products/{id}", productHandler.UpdateProduct)
			r.Delete("/products/{id}", productHandler.DeleteProduct)
		})

		r.Get("/inventory", productHandler.ListInventory)
		r.Get("/inventory/{id}", productHandler.GetInventory)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("owner", "admin", "warehouse_manager", "warehouse"))
			r.Post("/inventory", productHandler.CreateInventory)
		})

		// WMS Module (RBAC: kasir diisolasi total dari modul gudang)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.DenyRole("cashier"))
			wmsHandler.RegisterRoutes(r)
		})

		// POS (Point of Sale)
		r.Route("/pos", func(r chi.Router) {
			// RBAC: staf/kepala gudang & auditor tidak boleh transaksi kasir.
			r.Use(tenantMiddleware.DenyRole("warehouse", "warehouse_manager", "regional_manager", "auditor"))
			r.Post("/checkout", posHandler.Checkout)
			r.Get("/orders", posHandler.ListOrders)
			r.Get("/history", posHandler.ListOrders)
			r.Get("/orders/{id}", posHandler.GetOrder)
			r.Get("/items", productHandler.ListProducts)
			// Non-cash (QRIS) payments: create intent, poll, demo simulate.
			// Demo simulate is only allowed while the tenant has no gateway
			// credentials (ADR-008 Model B), so it cannot fake live payments.
			r.Post("/payments", posHandler.CreatePayment)
			r.Get("/payments/{orderId}/status", posHandler.PaymentStatus)
			r.Post("/payments/{orderId}/simulate", posHandler.SimulatePayment)
		})

		// Approvals
		r.Get("/approvals", approvalHandler.ListApprovals)
		r.Get("/approvals/{id}", approvalHandler.GetApproval)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "approver"))
			r.Post("/approvals/{id}/approve", approvalHandler.ApproveApproval)
			r.Post("/approvals/{id}/reject", approvalHandler.RejectApproval)
		})

		// Payment Gateway Configuration (ADR-008)
		r.Get("/payments/configs", paymentGatewayHandler.GetConfig)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin"))
			r.Post("/payments/configs", paymentGatewayHandler.UpsertConfig)
		})

		// Team Management & Profile Settings
		r.Get("/settings/profile", authHandler.GetCompanyProfile)
		r.Get("/settings/team", teamHandler.ListMembers)
		r.Get("/team", teamHandler.ListMembers)
		r.Group(func(r chi.Router) {
			r.Use(tenantMiddleware.RequireRole("admin", "owner"))
			r.Patch("/settings/profile", authHandler.UpdateCompanyProfile)
			r.Post("/settings/team", teamHandler.InviteMember)
			r.Patch("/settings/team/{id}/role", teamHandler.ChangeRole)
			r.Delete("/settings/team/{id}", teamHandler.RemoveMember)
			r.Post("/settings/team/{id}/resend-invite", teamHandler.ResendInvitation)
			r.Post("/team", teamHandler.InviteMember)
			r.Patch("/team/{id}/role", teamHandler.ChangeRole)
			r.Delete("/team/{id}", teamHandler.RemoveMember)
		})

		// Accounting Module: Chart of Accounts & Journal Entries
		listAccountsHandler := func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := tenantMiddleware.GetTenantID(r.Context())
			if !ok {
				respondError(w, r, http.StatusUnauthorized, "unauthorized")
				return
			}
			accounts, err := accountRepo.List(r.Context(), tenantID)
			if err != nil {
				log.Error().Err(err).Msg("list accounts")
				respondError(w, r, http.StatusInternalServerError, "internal server error")
				return
			}
			respondJSON(w, http.StatusOK, accounts)
		}

		getAccountHandler := func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := tenantMiddleware.GetTenantID(r.Context())
			if !ok {
				respondError(w, r, http.StatusUnauthorized, "unauthorized")
				return
			}
			id, err := uuid.Parse(chi.URLParam(r, "id"))
			if err != nil {
				respondError(w, r, http.StatusBadRequest, "invalid account id")
				return
			}
			account, err := accountRepo.GetByID(r.Context(), id, tenantID)
			if err != nil {
				respondError(w, r, http.StatusNotFound, "account not found")
				return
			}
			respondJSON(w, http.StatusOK, account)
		}

		createAccountHandler := func(w http.ResponseWriter, r *http.Request) {
			tenantID, ok := tenantMiddleware.GetTenantID(r.Context())
			if !ok {
				respondError(w, r, http.StatusUnauthorized, "unauthorized")
				return
			}
			var a accountingDomain.Account
			if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
				respondError(w, r, http.StatusBadRequest, "invalid request body")
				return
			}
			a.TenantID = tenantID
			if err := accountRepo.Create(r.Context(), &a); err != nil {
				log.Error().Err(err).Msg("create account")
				respondError(w, r, http.StatusInternalServerError, "internal server error")
				return
			}
			respondJSON(w, http.StatusCreated, a)
		}

		// /accounting/* routes
		r.Route("/accounting", func(r chi.Router) {
			r.Get("/accounts", listAccountsHandler)
			r.Get("/accounts/{id}", getAccountHandler)
			r.Post("/accounts", createAccountHandler)

			r.Get("/journal-entries", func(w http.ResponseWriter, r *http.Request) {
				tenantID, ok := tenantMiddleware.GetTenantID(r.Context())
				if !ok {
					respondError(w, r, http.StatusUnauthorized, "unauthorized")
					return
				}
				entries, err := accountingUsecase.ListJournalEntries(r.Context(), tenantID)
				if err != nil {
					log.Error().Err(err).Msg("list journal entries")
					respondError(w, r, http.StatusInternalServerError, "internal server error")
					return
				}
				if entries == nil {
					entries = []*accountingDomain.JournalEntry{}
				}
				respondJSON(w, http.StatusOK, entries)
			})

			r.Post("/journal-entries", func(w http.ResponseWriter, r *http.Request) {
				tenantID, ok := tenantMiddleware.GetTenantID(r.Context())
				if !ok {
					respondError(w, r, http.StatusUnauthorized, "unauthorized")
					return
				}
				var entry accountingDomain.JournalEntry
				if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
					respondError(w, r, http.StatusBadRequest, "invalid request body")
					return
				}
				entry.TenantID = tenantID
				if err := accountingUsecase.CreateJournalEntry(r.Context(), &entry); err != nil {
					if err == accountingDomain.ErrJournalNotBalanced {
						respondError(w, r, http.StatusBadRequest, err.Error())
						return
					}
					log.Error().Err(err).Msg("create journal entry")
					respondError(w, r, http.StatusInternalServerError, "internal server error")
					return
				}
				respondJSON(w, http.StatusCreated, entry)
			})
		})

		// Backwards-compatible /accounts
		r.Get("/accounts", listAccountsHandler)
		r.Get("/accounts/{id}", getAccountHandler)
		r.Post("/accounts", createAccountHandler)
	})

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.ServerPort).Msg("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exited")
}
