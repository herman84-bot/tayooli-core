package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Client sends messages via Telegram Bot API.
type Client struct {
	token    string
	chatID   string
	httpClient *http.Client
}

// NewClient creates a new Telegram bot client.
func NewClient(token, chatID string) *Client {
	return &Client{
		token:  token,
		chatID: chatID,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendMessage sends a text message to the configured chat.
func (c *Client) SendMessage(text string) error {
	if c.token == "" || c.chatID == "" {
		return fmt.Errorf("telegram not configured: missing token or chat_id")
	}

	chatIDInt, _ := strconv.ParseInt(c.chatID, 10, 64)
	return c.SendMessageToChat(chatIDInt, text)
}

// SendMessageToChat sends a text message to a specific chat.
func (c *Client) SendMessageToChat(chatID int64, text string) error {
	if c.token == "" {
		return fmt.Errorf("telegram not configured: missing token")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SendPhotoToChat sends a photo with caption to a specific chat.
func (c *Client) SendPhotoToChat(chatID int64, photoURL, caption string) error {
	if c.token == "" {
		return fmt.Errorf("telegram not configured: missing token")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendPhoto", c.token)

	payload := map[string]interface{}{
		"chat_id":  chatID,
		"photo":    photoURL,
		"caption":  caption,
		"parse_mode": "HTML",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// ReportIssue sends a support report to Telegram.
func (c *Client) ReportIssue(category, description, userEmail, tenantID string) error {
	now := time.Now().Format("2006-01-02 15:04 WIB")

	text := fmt.Sprintf(`🔔 <b>LAPORAN BARU DARI TAYOOLI</b>

<b>Kategori:</b> %s
<b>User:</b> %s
<b>Tenant:</b> %s
<b>Deskripsi:</b> %s

<b>Waktu:</b> %s`,
		category,
		userEmail,
		tenantID,
		description,
		now,
	)

	return c.SendMessage(text)
}

// NotifyNewSubscription notifies when a new subscription is created.
func (c *Client) NotifyNewSubscription(tenantID, plan, period string) error {
	now := time.Now().Format("2006-01-02 15:04 WIB")

	text := fmt.Sprintf(`🆕 <b>SUBSCRIPTION BARU</b>

<b>Tenant:</b> %s
<b>Paket:</b> %s
<b>Periode:</b> %s
<b>Waktu:</b> %s`,
		tenantID,
		plan,
		period,
		now,
	)

	return c.SendMessage(text)
}

// NotifyUpgrade notifies when a subscription is upgraded.
func (c *Client) NotifyUpgrade(tenantID, fromPlan, toPlan string) error {
	now := time.Now().Format("2006-01-02 15:04 WIB")

	text := fmt.Sprintf(`⬆️ <b>UPGRADE SUBSCRIPTION</b>

<b>Tenant:</b> %s
<b>Dari:</b> %s
<b>Ke:</b> %s
<b>Waktu:</b> %s`,
		tenantID,
		fromPlan,
		toPlan,
		now,
	)

	return c.SendMessage(text)
}

// NotifyCancel notifies when a subscription is cancelled.
func (c *Client) NotifyCancel(tenantID, plan string) error {
	now := time.Now().Format("2006-01-02 15:04 WIB")

	text := fmt.Sprintf(`❌ <b>SUBSCRIPTION DIBATALKAN</b>

<b>Tenant:</b> %s
<b>Paket:</b> %s
<b>Waktu:</b> %s`,
		tenantID,
		plan,
		now,
	)

	return c.SendMessage(text)
}
