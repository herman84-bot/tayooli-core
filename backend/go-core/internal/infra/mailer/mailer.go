package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// DefaultAppURL is the fallback public URL of Tayooli ERP when APP_URL is not configured.
const DefaultAppURL = "https://tayooli.my.id"

// DefaultBrevoSenderName is the fallback sender name when BREVO_SENDER_NAME is empty.
const DefaultBrevoSenderName = "Tayooli ERP"

// Mailer sends transactional emails via Brevo HTTP API v3, standard SMTP, or no-op logger.
type Mailer struct {
	// Brevo API settings
	brevoAPIKey      string
	brevoSenderEmail string
	brevoSenderName  string

	// SMTP settings
	host     string
	port     string
	username string
	password string
	from     string

	// General
	appURL     string
	httpClient *http.Client

	// Optional base URL override (used for httptest in unit tests)
	brevoBaseURL string
}

// New creates a new Mailer instance.
// Brevo HTTP API is used as the primary transactional email delivery mechanism if
// brevoAPIKey and brevoSenderEmail are provided. If not provided, it falls back to
// standard SMTP if host is set. Otherwise, it operates in no-op logging mode.
func New(brevoAPIKey, brevoSenderEmail, brevoSenderName, host, port, username, password, from, appURL string) *Mailer {
	if appURL == "" {
		appURL = DefaultAppURL
	}
	if brevoSenderName == "" {
		brevoSenderName = DefaultBrevoSenderName
	}
	return &Mailer{
		brevoAPIKey:      brevoAPIKey,
		brevoSenderEmail: brevoSenderEmail,
		brevoSenderName:  brevoSenderName,
		host:             host,
		port:             port,
		username:         username,
		password:         password,
		from:             from,
		appURL:           appURL,
		httpClient:       &http.Client{Timeout: 45 * time.Second},
	}
}

// NewWithBrevo is an alias to New that explicitly conveys Brevo integration support.
func NewWithBrevo(brevoAPIKey, brevoSenderEmail, brevoSenderName, host, port, username, password, from, appURL string) *Mailer {
	return New(brevoAPIKey, brevoSenderEmail, brevoSenderName, host, port, username, password, from, appURL)
}

// NewSMTP creates a Mailer with SMTP settings only.
func NewSMTP(host, port, username, password, from, appURL string) *Mailer {
	return New("", "", "", host, port, username, password, from, appURL)
}

// IsConfigured returns true if either Brevo API or SMTP settings are configured.
func (m *Mailer) IsConfigured() bool {
	return (m.brevoAPIKey != "" && m.brevoSenderEmail != "") || (m.host != "" && m.from != "")
}

// Brevo payload models
type brevoRecipient struct {
	Email string `json:"email"`
}

type brevoSender struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type brevoEmailPayload struct {
	Sender      brevoSender      `json:"sender"`
	To          []brevoRecipient `json:"to"`
	Subject     string           `json:"subject"`
	HTMLContent string           `json:"htmlContent,omitempty"`
	TextContent string           `json:"textContent,omitempty"`
}

// SendPasswordResetEmail sends a password reset link to the user with a responsive HTML template.
func (m *Mailer) SendPasswordResetEmail(toEmail, resetToken string) error {
	baseURL := strings.TrimRight(m.appURL, "/")
	if baseURL == "" {
		baseURL = DefaultAppURL
	}
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", baseURL, url.QueryEscape(resetToken))
	escapedLink := html.EscapeString(resetLink)

	subject := "Tayooli ERP - Reset Password"

	textBody := fmt.Sprintf(`Halo,

Kami menerima permintaan untuk mereset kata sandi akun Tayooli ERP Anda.

Buka tautan berikut untuk membuat kata sandi baru:
%s

Tautan ini berlaku selama 1 jam.

Jika Anda tidak meminta reset kata sandi, Anda dapat mengabaikan email ini dengan aman.

Salam,
Tim Tayooli ERP
%s`, resetLink, baseURL)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Reset Password - Tayooli ERP</title>
</head>
<body style="margin:0;padding:0;background-color:#f1f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#1e293b;-webkit-font-smoothing:antialiased;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f1f5f9;padding:40px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" style="max-width:560px;background-color:#ffffff;border-radius:12px;box-shadow:0 4px 6px -1px rgba(0,0,0,0.05),0 2px 4px -2px rgba(0,0,0,0.05);overflow:hidden;border:1px solid #e2e8f0;">
          <tr>
            <td style="padding:28px 36px 20px;background:linear-gradient(135deg,#0f172a 0%%,#1e293b 100%%);text-align:left;">
              <span style="font-size:22px;font-weight:700;color:#ffffff;letter-spacing:-0.5px;">Tayooli ERP</span>
              <span style="display:block;font-size:13px;color:#94a3b8;margin-top:4px;">Sistem Manajemen Bisnis &amp; Keuangan Terintegrasi</span>
            </td>
          </tr>
          <tr>
            <td style="padding:32px 36px;">
              <h1 style="font-size:20px;font-weight:600;color:#0f172a;margin-top:0;margin-bottom:16px;">Atur Ulang Kata Sandi</h1>
              <p style="font-size:15px;line-height:1.6;color:#475569;margin:0 0 20px;">Halo,</p>
              <p style="font-size:15px;line-height:1.6;color:#475569;margin:0 0 24px;">
                Kami menerima permintaan untuk mereset kata sandi akun Tayooli ERP Anda. Klik tombol di bawah ini untuk membuat kata sandi baru:
              </p>
              <table role="presentation" cellpadding="0" cellspacing="0" style="margin:28px 0;">
                <tr>
                  <td align="center" style="border-radius:8px;background-color:#0284c7;">
                    <a href="%s" target="_blank" style="display:inline-block;padding:14px 28px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;border-radius:8px;background-color:#0284c7;">
                      Reset Password Sekarang &rarr;
                    </a>
                  </td>
                </tr>
              </table>
              <p style="font-size:14px;line-height:1.6;color:#64748b;margin:0 0 16px;">
                Tautan ini berlaku selama <strong>1 jam</strong>.
              </p>
              <div style="background-color:#f8fafc;border-left:3px solid:#0284c7;padding:12px 16px;border-radius:0 6px 6px 0;margin:20px 0;">
                <p style="font-size:13px;color:#475569;margin:0;line-height:1.5;">
                  <strong>Peringatan Keamanan:</strong> Jika Anda tidak merasa melakukan permintaan ini, segera abaikan email ini. Kata sandi lama Anda tetap aman.
                </p>
              </div>
              <p style="font-size:13px;line-height:1.5;color:#94a3b8;margin:24px 0 0;">
                Jika tombol di atas tidak dapat diklik, salin dan tempel tautan berikut di peramban Anda:<br>
                <a href="%s" style="color:#0284c7;word-break:break-all;text-decoration:underline;">%s</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 36px;background-color:#f8fafc;border-top:1px solid #e2e8f0;text-align:center;">
              <p style="font-size:12px;color:#94a3b8;margin:0;line-height:1.5;">
                &copy; 2026 Tayooli ERP &bull; Paper.id-style Enterprise Resource Planning.<br>
                Email ini dikirim secara otomatis, mohon tidak membalas langsung ke alamat ini.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, escapedLink, escapedLink, escapedLink)

	return m.send(toEmail, subject, htmlBody, textBody)
}

// SendVerificationEmail sends an email verification link to the user with a responsive HTML template.
func (m *Mailer) SendVerificationEmail(toEmail, verificationToken string) error {
	baseURL := strings.TrimRight(m.appURL, "/")
	if baseURL == "" {
		baseURL = DefaultAppURL
	}
	verifyLink := fmt.Sprintf("%s/verify-email?token=%s", baseURL, url.QueryEscape(verificationToken))
	escapedLink := html.EscapeString(verifyLink)

	subject := "Tayooli ERP - Verifikasi Email"

	textBody := fmt.Sprintf(`Halo,

Terima kasih telah mendaftar di Tayooli ERP.

Buka tautan berikut untuk memverifikasi alamat email Anda:
%s

Tautan ini berlaku selama 24 jam.

Salam,
Tim Tayooli ERP
%s`, verifyLink, baseURL)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Verifikasi Email - Tayooli ERP</title>
</head>
<body style="margin:0;padding:0;background-color:#f1f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#1e293b;-webkit-font-smoothing:antialiased;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f1f5f9;padding:40px 16px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" style="max-width:560px;background-color:#ffffff;border-radius:12px;box-shadow:0 4px 6px -1px rgba(0,0,0,0.05),0 2px 4px -2px rgba(0,0,0,0.05);overflow:hidden;border:1px solid #e2e8f0;">
          <tr>
            <td style="padding:28px 36px 20px;background:linear-gradient(135deg,#0f172a 0%%,#1e293b 100%%);text-align:left;">
              <span style="font-size:22px;font-weight:700;color:#ffffff;letter-spacing:-0.5px;">Tayooli ERP</span>
              <span style="display:block;font-size:13px;color:#94a3b8;margin-top:4px;">Sistem Manajemen Bisnis &amp; Keuangan Terintegrasi</span>
            </td>
          </tr>
          <tr>
            <td style="padding:32px 36px;">
              <h1 style="font-size:20px;font-weight:600;color:#0f172a;margin-top:0;margin-bottom:16px;">Verifikasi Alamat Email</h1>
              <p style="font-size:15px;line-height:1.6;color:#475569;margin:0 0 20px;">Halo,</p>
              <p style="font-size:15px;line-height:1.6;color:#475569;margin:0 0 24px;">
                Terima kasih telah mendaftar di Tayooli ERP. Klik tombol di bawah ini untuk memverifikasi alamat email akun Anda:
              </p>
              <table role="presentation" cellpadding="0" cellspacing="0" style="margin:28px 0;">
                <tr>
                  <td align="center" style="border-radius:8px;background-color:#0284c7;">
                    <a href="%s" target="_blank" style="display:inline-block;padding:14px 28px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;border-radius:8px;background-color:#0284c7;">
                      Verifikasi Email Sekarang &rarr;
                    </a>
                  </td>
                </tr>
              </table>
              <p style="font-size:14px;line-height:1.6;color:#64748b;margin:0 0 16px;">
                Tautan verifikasi ini berlaku selama <strong>24 jam</strong>.
              </p>
              <p style="font-size:13px;line-height:1.5;color:#94a3b8;margin:24px 0 0;">
                Jika tombol di atas tidak dapat diklik, salin dan tempel tautan berikut di peramban Anda:<br>
                <a href="%s" style="color:#0284c7;word-break:break-all;text-decoration:underline;">%s</a>
              </p>
            </td>
          </tr>
          <tr>
            <td style="padding:20px 36px;background-color:#f8fafc;border-top:1px solid #e2e8f0;text-align:center;">
              <p style="font-size:12px;color:#94a3b8;margin:0;line-height:1.5;">
                &copy; 2026 Tayooli ERP &bull; Paper.id-style Enterprise Resource Planning.<br>
                Email ini dikirim secara otomatis, mohon tidak membalas langsung ke alamat ini.
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, escapedLink, escapedLink, escapedLink)

	return m.send(toEmail, subject, htmlBody, textBody)
}

// SendInvitationEmail sends a team invitation with a link to set the account
// password. The token reuses the password-reset mechanism (/reset-password).
func (m *Mailer) SendInvitationEmail(toEmail, token string) error {
	baseURL := strings.TrimRight(m.appURL, "/")
	if baseURL == "" {
		baseURL = DefaultAppURL
	}
	link := fmt.Sprintf("%s/reset-password?token=%s&invite=1", baseURL, url.QueryEscape(token))
	escapedLink := html.EscapeString(link)
	escapedEmail := html.EscapeString(toEmail)

	subject := "Undangan bergabung ke Tayooli ERP"

	textBody := fmt.Sprintf(`Halo,

Anda diundang bergabung ke workspace Tayooli ERP dengan email %s.

Buka tautan berikut untuk membuat kata sandi dan mengaktifkan akun Anda:
%s

Tautan ini berlaku selama 72 jam. Setelah itu, minta admin mengirim ulang undangan.

Salam,
Tim Tayooli ERP
%s`, toEmail, link, baseURL)

	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>Undangan Tayooli ERP</title></head>
<body style="margin:0;padding:0;background-color:#f1f5f9;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,Helvetica,Arial,sans-serif;color:#1e293b;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" style="background-color:#f1f5f9;padding:40px 16px;">
    <tr><td align="center">
      <table role="presentation" width="100%%" style="max-width:560px;background-color:#ffffff;border-radius:12px;overflow:hidden;border:1px solid #e2e8f0;">
        <tr><td style="padding:28px 36px 20px;background:#0f172a;">
          <span style="font-size:22px;font-weight:700;color:#ffffff;">Tayooli ERP</span>
        </td></tr>
        <tr><td style="padding:32px 36px;">
          <h1 style="font-size:20px;font-weight:600;color:#0f172a;margin:0 0 16px;">Anda diundang bergabung</h1>
          <p style="font-size:15px;line-height:1.6;color:#475569;margin:0 0 24px;">
            Akun untuk <strong>%s</strong> telah dibuat. Klik tombol di bawah untuk membuat kata sandi dan mulai menggunakan Tayooli ERP.
          </p>
          <table role="presentation" cellpadding="0" cellspacing="0" style="margin:28px 0;"><tr>
            <td style="border-radius:8px;background-color:#0284c7;">
              <a href="%s" target="_blank" style="display:inline-block;padding:14px 28px;font-size:15px;font-weight:600;color:#ffffff;text-decoration:none;">Aktifkan Akun &rarr;</a>
            </td>
          </tr></table>
          <p style="font-size:14px;color:#64748b;margin:0 0 16px;">Tautan berlaku selama <strong>72 jam</strong>.</p>
          <p style="font-size:13px;line-height:1.5;color:#94a3b8;margin:24px 0 0;">
            Jika tombol tidak dapat diklik, salin tautan berikut:<br>
            <a href="%s" style="color:#0284c7;word-break:break-all;">%s</a>
          </p>
        </td></tr>
      </table>
    </td></tr>
  </table>
</body>
</html>`, escapedEmail, escapedLink, escapedLink, escapedLink)

	return m.send(toEmail, subject, htmlBody, textBody)
}

// send routes delivery to Brevo HTTP API (primary), standard SMTP (secondary), or no-op log (fallback).
func (m *Mailer) send(to, subject, htmlBody, textBody string) error {
	// Defense-in-depth: guard against SMTP header injection via newline characters
	if strings.ContainsAny(to, "\r\n") || strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("mailer: header injection detected in recipient or subject")
	}

	// 1. Brevo HTTP API v3 (primary when API key and sender email are provided)
	if m.brevoAPIKey != "" && m.brevoSenderEmail != "" {
		return m.sendBrevo(context.Background(), to, subject, htmlBody, textBody)
	}

	// 2. Standard SMTP (secondary when host is provided)
	if m.host != "" {
		return m.sendSMTP(to, subject, htmlBody, textBody)
	}

	// 3. No-op logging (fallback when neither is provided)
	fmt.Printf("[MAILER] no-op mode: would send to=%s subject=%q\n", to, subject)
	return nil
}

// sendBrevo dispatches the email via Brevo REST API v3 (/v3/smtp/email).
func (m *Mailer) sendBrevo(ctx context.Context, to, subject, htmlBody, textBody string) error {
	baseURL := m.brevoBaseURL
	if baseURL == "" {
		baseURL = "https://api.brevo.com"
	}
	endpoint := strings.TrimRight(baseURL, "/") + "/v3/smtp/email"

	payload := brevoEmailPayload{
		Sender: brevoSender{
			Name:  m.brevoSenderName,
			Email: m.brevoSenderEmail,
		},
		To: []brevoRecipient{
			{Email: to},
		},
		Subject:     subject,
		HTMLContent: htmlBody,
		TextContent: textBody,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mailer: marshal brevo payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("mailer: create brevo request: %w", err)
	}

	req.Header.Set("api-key", m.brevoAPIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := m.httpClient
	if client == nil {
		client = &http.Client{Timeout: 45 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("mailer: send brevo request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("brevo API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	log.Printf("[MAILER] brevo email sent to %s (status %d): %s", to, resp.StatusCode, strings.TrimSpace(string(respBody)))
	return nil
}

// sendSMTP dispatches the email via net/smtp.SendMail with MIME multipart/alternative support.
func (m *Mailer) sendSMTP(to, subject, htmlBody, textBody string) error {
	from := m.from
	if from == "" {
		from = "noreply@tayooli.com"
	}

	var msg string
	if htmlBody != "" && textBody != "" {
		boundary := "====TayooliMultipartBoundary===="
		msg = fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n--%s\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s\r\n\r\n--%s\r\nContent-Type: text/html; charset=\"utf-8\"\r\n\r\n%s\r\n\r\n--%s--\r\n",
			from, to, subject, boundary, boundary, textBody, boundary, htmlBody, boundary)
	} else if htmlBody != "" {
		msg = fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"utf-8\"\r\n\r\n%s",
			from, to, subject, htmlBody)
	} else {
		msg = fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n\r\n%s",
			from, to, subject, textBody)
	}

	addr := fmt.Sprintf("%s:%s", m.host, m.port)

	var auth smtp.Auth
	if m.username != "" {
		auth = smtp.PlainAuth("", m.username, m.password, m.host)
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(msg))
}
