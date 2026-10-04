package config

import (
	"fmt"
	"os"
)

// MinJWTSecretLength is the minimum length enforced for JWT_SECRET (HS256
// signing key). Values below this are rejected at startup by Load() and at
// request time by the auth middleware.
const MinJWTSecretLength = 32

// Config holds all application configuration loaded from environment variables.
type Config struct {
	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Auth
	JWTSecret string

	// Kafka
	KafkaBrokers string

	// Optional services
	RedisURL       string
	AIServingURL   string
	FrontendOrigin string

	// Server
	ServerPort string

	// Observability
	SentryDSN string
	AppEnv    string

	// SMTP Email (optional — no-op mode if not configured)
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string
	AppURL   string

	// Brevo API Email (primary transactional email provider when configured)
	BrevoAPIKey      string
	BrevoSenderEmail string
	BrevoSenderName  string
}

// Load reads environment variables and validates required ones.
// Returns an error with a clear message if any required variable is missing.
func Load() (*Config, error) {
	cfg := &Config{
		DBHost:         os.Getenv("DB_HOST"),
		DBPort:         os.Getenv("DB_PORT"),
		DBUser:         os.Getenv("DB_USER"),
		DBPassword:     os.Getenv("DB_PASSWORD"),
		DBName:         os.Getenv("DB_NAME"),
		DBSSLMode:      os.Getenv("DB_SSLMODE"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		KafkaBrokers:   os.Getenv("KAFKA_BROKERS"),
		RedisURL:       os.Getenv("REDIS_URL"),
		AIServingURL:   os.Getenv("AI_SERVING_URL"),
		FrontendOrigin: os.Getenv("FRONTEND_ORIGIN"),
		ServerPort:     os.Getenv("SERVER_PORT"),
		SentryDSN:      os.Getenv("SENTRY_DSN"),
		AppEnv:         os.Getenv("APP_ENV"),
		SMTPHost:       os.Getenv("SMTP_HOST"),
		SMTPPort:       os.Getenv("SMTP_PORT"),
		SMTPUser:       os.Getenv("SMTP_USER"),
		SMTPPass:       os.Getenv("SMTP_PASS"),
		SMTPFrom:       os.Getenv("SMTP_FROM"),
		AppURL:         os.Getenv("APP_URL"),
		BrevoAPIKey:      os.Getenv("BREVO_API_KEY"),
		BrevoSenderEmail: os.Getenv("BREVO_SENDER_EMAIL"),
		BrevoSenderName:  os.Getenv("BREVO_SENDER_NAME"),
	}

	// Defaults for optional fields
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8081"
	}
	if cfg.AppEnv == "" {
		cfg.AppEnv = "production"
	}
	if cfg.DBSSLMode == "" {
		cfg.DBSSLMode = "require"
	}
	if cfg.SMTPPort == "" {
		cfg.SMTPPort = "587"
	}
	if cfg.SMTPFrom == "" {
		cfg.SMTPFrom = "noreply@tayooli.com"
	}
	if cfg.AppURL == "" {
		cfg.AppURL = "http://localhost:3000"
	}
	if cfg.BrevoSenderName == "" {
		cfg.BrevoSenderName = "Tayooli ERP"
	}

	// Validate required env vars
	required := map[string]string{
		"DB_HOST":     cfg.DBHost,
		"DB_PORT":     cfg.DBPort,
		"DB_USER":     cfg.DBUser,
		"DB_PASSWORD": cfg.DBPassword,
		"DB_NAME":     cfg.DBName,
		"JWT_SECRET":  cfg.JWTSecret,
	}

	for key, val := range required {
		if val == "" {
			return nil, fmt.Errorf("required env var %s is missing", key)
		}
	}

	// JWT_SECRET is used to sign tokens — enforce a minimum length so a
	// placeholder like "CHANGE_ME" (or any short value) fails fast at startup
	// instead of shipping weak HS256 keys to production.
	if len(cfg.JWTSecret) < MinJWTSecretLength {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d) — replace the CHANGE_ME placeholder before starting", len(cfg.JWTSecret))
	}

	return cfg, nil
}

// DSN builds the PostgreSQL connection string from config fields.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
	)
}
