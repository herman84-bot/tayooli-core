package config

import (
	"strings"
	"testing"
)

// setRequiredEnv sets every env var Load() requires, so only the JWT_SECRET
// length gate is exercised. Load() performs no DB connection — it only reads
// environment variables.
func setRequiredEnv(t *testing.T, jwtSecret string) {
	t.Helper()
	t.Setenv("DB_HOST", "localhost")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "tayooli")
	t.Setenv("DB_PASSWORD", "testpass")
	t.Setenv("DB_NAME", "tayooli_test")
	t.Setenv("KAFKA_BROKERS", "localhost:9092")
	t.Setenv("JWT_SECRET", jwtSecret)
}

func TestLoad_JWTSecretLengthGate(t *testing.T) {
	t.Run("rejects secret shorter than 32 chars", func(t *testing.T) {
		setRequiredEnv(t, "short-secret")

		_, err := Load()

		if err == nil {
			t.Fatal("expected error for short JWT_SECRET, got nil")
		}
		if !strings.Contains(err.Error(), "JWT_SECRET must be at least 32 characters") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("rejects 31-char secret", func(t *testing.T) {
		// Boundary check: exactly one char below the 32-char minimum.
		setRequiredEnv(t, strings.Repeat("a", MinJWTSecretLength-1))

		_, err := Load()

		if err == nil {
			t.Fatal("expected error for 31-char JWT_SECRET, got nil")
		}
	})

	t.Run("accepts 32-char secret", func(t *testing.T) {
		setRequiredEnv(t, strings.Repeat("b", MinJWTSecretLength))

		_, err := Load()

		if err != nil {
			t.Fatalf("expected no error for 32-char JWT_SECRET, got: %v", err)
		}
	})

	t.Run("accepts secret longer than 32 chars", func(t *testing.T) {
		setRequiredEnv(t, strings.Repeat("c", MinJWTSecretLength+10))

		_, err := Load()

		if err != nil {
			t.Fatalf("expected no error for 42-char JWT_SECRET, got: %v", err)
		}
	})
}

func TestMinJWTSecretLength(t *testing.T) {
	if MinJWTSecretLength != 32 {
		t.Fatalf("MinJWTSecretLength = %d, want 32", MinJWTSecretLength)
	}
}

func TestLoad_BrevoConfig(t *testing.T) {
	validSecret := strings.Repeat("s", 32)

	t.Run("loads brevo env vars and uses default sender name", func(t *testing.T) {
		setRequiredEnv(t, validSecret)
		t.Setenv("BREVO_API_KEY", "xkeysib-test-key")
		t.Setenv("BREVO_SENDER_EMAIL", "noreply@tayooli.com")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.BrevoAPIKey != "xkeysib-test-key" {
			t.Errorf("BrevoAPIKey = %q, want %q", cfg.BrevoAPIKey, "xkeysib-test-key")
		}
		if cfg.BrevoSenderEmail != "noreply@tayooli.com" {
			t.Errorf("BrevoSenderEmail = %q, want %q", cfg.BrevoSenderEmail, "noreply@tayooli.com")
		}
		if cfg.BrevoSenderName != "Tayooli ERP" {
			t.Errorf("BrevoSenderName = %q, want default %q", cfg.BrevoSenderName, "Tayooli ERP")
		}
	})

	t.Run("respects custom brevo sender name", func(t *testing.T) {
		setRequiredEnv(t, validSecret)
		t.Setenv("BREVO_API_KEY", "xkeysib-test-key")
		t.Setenv("BREVO_SENDER_EMAIL", "noreply@tayooli.com")
		t.Setenv("BREVO_SENDER_NAME", "Tayooli Sales")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.BrevoSenderName != "Tayooli Sales" {
			t.Errorf("BrevoSenderName = %q, want %q", cfg.BrevoSenderName, "Tayooli Sales")
		}
	})
}

func TestLoad_KafkaBrokersOptional(t *testing.T) {
	validSecret := strings.Repeat("s", 32)
	setRequiredEnv(t, validSecret)
	t.Setenv("KAFKA_BROKERS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected Load() to succeed without KAFKA_BROKERS, got: %v", err)
	}

	if cfg.KafkaBrokers != "" {
		t.Errorf("KafkaBrokers = %q, want empty", cfg.KafkaBrokers)
	}
}

func TestLoad_DatabaseURLFallback(t *testing.T) {
	validSecret := strings.Repeat("s", 32)
	t.Setenv("JWT_SECRET", validSecret)
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("DATABASE_URL", "postgres://zeabur_user:zeabur_pass@zeabur-db.internal:5432/zeabur_db?sslmode=disable")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DBHost != "zeabur-db.internal" {
		t.Errorf("DBHost = %q, want %q", cfg.DBHost, "zeabur-db.internal")
	}
	if cfg.DBPort != "5432" {
		t.Errorf("DBPort = %q, want %q", cfg.DBPort, "5432")
	}
	if cfg.DBUser != "zeabur_user" {
		t.Errorf("DBUser = %q, want %q", cfg.DBUser, "zeabur_user")
	}
	if cfg.DBPassword != "zeabur_pass" {
		t.Errorf("DBPassword = %q, want %q", cfg.DBPassword, "zeabur_pass")
	}
	if cfg.DBName != "zeabur_db" {
		t.Errorf("DBName = %q, want %q", cfg.DBName, "zeabur_db")
	}
	if cfg.DBSSLMode != "disable" {
		t.Errorf("DBSSLMode = %q, want %q", cfg.DBSSLMode, "disable")
	}
}

