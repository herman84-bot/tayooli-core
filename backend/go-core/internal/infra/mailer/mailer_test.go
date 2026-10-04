package mailer

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMailer_IsConfigured(t *testing.T) {
	t.Run("returns true when Brevo is configured", func(t *testing.T) {
		m := New("test-api-key", "noreply@tayooli.com", "Tayooli ERP", "", "", "", "", "", "")
		if !m.IsConfigured() {
			t.Fatal("expected IsConfigured to return true for Brevo")
		}
	})

	t.Run("returns true when SMTP is configured", func(t *testing.T) {
		m := New("", "", "", "smtp.example.com", "587", "user", "pass", "noreply@tayooli.com", "")
		if !m.IsConfigured() {
			t.Fatal("expected IsConfigured to return true for SMTP")
		}
	})

	t.Run("returns false when neither is configured", func(t *testing.T) {
		m := New("", "", "", "", "", "", "", "", "")
		if m.IsConfigured() {
			t.Fatal("expected IsConfigured to return false when empty")
		}
	})
}

func TestMailer_Defaults(t *testing.T) {
	m := New("", "", "", "", "", "", "", "", "")
	if m.appURL != DefaultAppURL {
		t.Errorf("appURL = %q, want default %q", m.appURL, DefaultAppURL)
	}
	if m.brevoSenderName != DefaultBrevoSenderName {
		t.Errorf("brevoSenderName = %q, want default %q", m.brevoSenderName, DefaultBrevoSenderName)
	}

	custom := NewWithBrevo("k", "e@tayooli.com", "Custom Sender", "h", "p", "u", "pwd", "f", "https://custom.domain.com")
	if custom.appURL != "https://custom.domain.com" {
		t.Errorf("custom appURL = %q, want https://custom.domain.com", custom.appURL)
	}
	if custom.brevoSenderName != "Custom Sender" {
		t.Errorf("custom brevoSenderName = %q, want Custom Sender", custom.brevoSenderName)
	}
}

func TestMailer_Brevo_SendPasswordResetEmail(t *testing.T) {
	var capturedHeader http.Header
	var capturedPayload brevoEmailPayload
	var requestReceived bool

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived = true
		capturedHeader = r.Header.Clone()

		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/smtp/email" {
			t.Errorf("path = %s, want /v3/smtp/email", r.URL.Path)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		if err := json.Unmarshal(body, &capturedPayload); err != nil {
			t.Fatalf("failed to parse JSON payload: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"messageId": "<2026.test@brevo.com>"}`))
	}))
	defer ts.Close()

	m := New("test-brevo-api-key", "noreply@tayooli.com", "Tayooli ERP", "", "", "", "", "", "https://tayooli.my.id")
	m.brevoBaseURL = ts.URL
	m.httpClient = ts.Client()

	err := m.SendPasswordResetEmail("user@example.com", "secret-reset-token-123")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if !requestReceived {
		t.Fatal("expected request to be received by test server")
	}

	if capturedHeader.Get("api-key") != "test-brevo-api-key" {
		t.Errorf("api-key header = %q, want %q", capturedHeader.Get("api-key"), "test-brevo-api-key")
	}
	if !strings.Contains(capturedHeader.Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q, want application/json", capturedHeader.Get("Content-Type"))
	}

	if capturedPayload.Sender.Email != "noreply@tayooli.com" {
		t.Errorf("sender email = %q, want noreply@tayooli.com", capturedPayload.Sender.Email)
	}
	if capturedPayload.Sender.Name != "Tayooli ERP" {
		t.Errorf("sender name = %q, want Tayooli ERP", capturedPayload.Sender.Name)
	}
	if len(capturedPayload.To) != 1 || capturedPayload.To[0].Email != "user@example.com" {
		t.Errorf("to = %+v, want user@example.com", capturedPayload.To)
	}
	if capturedPayload.Subject != "Tayooli ERP - Reset Password" {
		t.Errorf("subject = %q", capturedPayload.Subject)
	}

	expectedLink := "https://tayooli.my.id/reset-password?token=secret-reset-token-123"
	if !strings.Contains(capturedPayload.HTMLContent, expectedLink) {
		t.Errorf("HTMLContent does not contain expected link %s", expectedLink)
	}
	if !strings.Contains(capturedPayload.TextContent, expectedLink) {
		t.Errorf("TextContent does not contain expected link %s", expectedLink)
	}
}

func TestMailer_Brevo_SendVerificationEmail(t *testing.T) {
	var capturedPayload brevoEmailPayload

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedPayload)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"messageId": "<verif@brevo>"}`))
	}))
	defer ts.Close()

	m := New("test-key", "noreply@tayooli.com", "Tayooli ERP", "", "", "", "", "", "https://tayooli.my.id")
	m.brevoBaseURL = ts.URL
	m.httpClient = ts.Client()

	err := m.SendVerificationEmail("verify@example.com", "verif-token-456")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if capturedPayload.Subject != "Tayooli ERP - Verifikasi Email" {
		t.Errorf("subject = %q", capturedPayload.Subject)
	}
	expectedLink := "https://tayooli.my.id/verify-email?token=verif-token-456"
	if !strings.Contains(capturedPayload.HTMLContent, expectedLink) {
		t.Errorf("HTMLContent missing verify link %s", expectedLink)
	}
	if !strings.Contains(capturedPayload.TextContent, expectedLink) {
		t.Errorf("TextContent missing verify link %s", expectedLink)
	}
}

func TestMailer_Brevo_ErrorHandling(t *testing.T) {
	t.Run("handles 401 unauthorized", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message": "Key not found", "code": "unauthorized"}`))
		}))
		defer ts.Close()

		m := New("invalid-key", "noreply@tayooli.com", "Tayooli ERP", "", "", "", "", "", "")
		m.brevoBaseURL = ts.URL
		m.httpClient = ts.Client()

		err := m.SendPasswordResetEmail("user@example.com", "tok")
		if err == nil {
			t.Fatal("expected error on 401, got nil")
		}
		if !strings.Contains(err.Error(), "status 401") || !strings.Contains(err.Error(), "Key not found") {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("handles 500 server error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message": "internal server error"}`))
		}))
		defer ts.Close()

		m := New("key", "noreply@tayooli.com", "Tayooli ERP", "", "", "", "", "", "")
		m.brevoBaseURL = ts.URL
		m.httpClient = ts.Client()

		err := m.SendPasswordResetEmail("user@example.com", "tok")
		if err == nil {
			t.Fatal("expected error on 500, got nil")
		}
		if !strings.Contains(err.Error(), "status 500") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestMailer_NoOpMode(t *testing.T) {
	m := New("", "", "", "", "", "", "", "", "")
	// Should not fail and should log no-op
	if err := m.SendPasswordResetEmail("user@example.com", "tok"); err != nil {
		t.Fatalf("expected nil error in no-op mode, got: %v", err)
	}
	if err := m.SendVerificationEmail("user@example.com", "tok"); err != nil {
		t.Fatalf("expected nil error in no-op mode, got: %v", err)
	}
}

func TestMailer_HeaderInjection(t *testing.T) {
	m := New("test-key", "noreply@tayooli.com", "Tayooli ERP", "", "", "", "", "", "")

	testCases := []struct {
		name  string
		email string
	}{
		{"CRLF injection", "victim@example.com\r\nBcc: evil@attacker.com"},
		{"LF injection", "victim@example.com\nBcc: evil@attacker.com"},
		{"CR injection", "victim@example.com\rBcc: evil@attacker.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := m.SendPasswordResetEmail(tc.email, "token")
			if err == nil {
				t.Fatalf("expected error for %s, got nil", tc.name)
			}
			if !strings.Contains(err.Error(), "header injection") {
				t.Errorf("expected header injection error, got: %v", err)
			}
		})
	}
}

func TestMailer_HTMLSpecialCharactersEscaping(t *testing.T) {
	var capturedPayload brevoEmailPayload

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &capturedPayload)
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	m := New("test-key", "noreply@tayooli.com", "Tayooli ERP", "", "", "", "", "", "https://tayooli.my.id")
	m.brevoBaseURL = ts.URL
	m.httpClient = ts.Client()

	maliciousToken := `test"onclick="alert(1)"<script>`
	err := m.SendPasswordResetEmail("user@example.com", maliciousToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// In HTML content, <script> and quotes in URL should be query-escaped or HTML-escaped
	if strings.Contains(capturedPayload.HTMLContent, "<script>") {
		t.Errorf("HTMLContent contains unescaped <script> tag: %s", capturedPayload.HTMLContent)
	}
}
