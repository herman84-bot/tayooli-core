package pakasir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://app.pakasir.com"

// Client is an HTTP client for the Pakasir payment gateway.
type Client struct {
	httpClient *http.Client
}

// NewClient creates a Pakasir client with a 15-second timeout.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// CreateTransactionRequest is the payload for creating a transaction.
type CreateTransactionRequest struct {
	Project string `json:"project"`
	OrderID string `json:"order_id"`
	Amount  int    `json:"amount"`
	APIKey  string `json:"api_key"`
}

// CreateTransactionResponse represents a successful transaction creation.
type CreateTransactionResponse struct {
	Payment struct {
		Project       string `json:"project"`
		OrderID       string `json:"order_id"`
		Amount        int    `json:"amount"`
		Fee           int    `json:"fee"`
		TotalPayment  int    `json:"total_payment"`
		PaymentMethod string `json:"payment_method"`
		PaymentNumber string `json:"payment_number"`
		ExpiredAt     string `json:"expired_at"`
	} `json:"payment"`
}

// TransactionDetailResponse represents the status of a transaction.
type TransactionDetailResponse struct {
	Transaction struct {
		Amount        int    `json:"amount"`
		OrderID       string `json:"order_id"`
		Project       string `json:"project"`
		Status        string `json:"status"`
		PaymentMethod string `json:"payment_method"`
		CompletedAt   string `json:"completed_at"`
	} `json:"transaction"`
}

// WebhookPayload is the structure sent by Pakasir on payment completion.
type WebhookPayload struct {
	Amount        int    `json:"amount"`
	OrderID       string `json:"order_id"`
	Project       string `json:"project"`
	Status        string `json:"status"`
	PaymentMethod string `json:"payment_method"`
	CompletedAt   string `json:"completed_at"`
}

// PaymentMethod constants.
const (
	PaymentMethodQRIS = "qris"
	PaymentMethodBRI  = "bri_va"
	PaymentMethodBNI  = "bni_va"
)

// CreateTransaction creates a new payment transaction via Pakasir API.
func (c *Client) CreateTransaction(ctx context.Context, project, orderID, apiKey, method string, amount int) (*CreateTransactionResponse, error) {
	reqBody := CreateTransactionRequest{
		Project: project,
		OrderID: orderID,
		Amount:  amount,
		APIKey:  apiKey,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("pakasir create marshal: %w", err)
	}

	url := fmt.Sprintf("%s/api/transactioncreate/%s", baseURL, method)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("pakasir create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pakasir create execute: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pakasir create read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pakasir create status %d: %s", resp.StatusCode, string(respBody))
	}

	var result CreateTransactionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("pakasir create unmarshal: %w", err)
	}

	return &result, nil
}

// GetTransactionDetail checks the status of a transaction.
func (c *Client) GetTransactionDetail(ctx context.Context, project, orderID, apiKey string, amount int) (*TransactionDetailResponse, error) {
	url := fmt.Sprintf("%s/api/transactiondetail?project=%s&order_id=%s&api_key=%s&amount=%d",
		baseURL, project, orderID, apiKey, amount)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("pakasir detail request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pakasir detail execute: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pakasir detail read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pakasir detail status %d: %s", resp.StatusCode, string(respBody))
	}

	var result TransactionDetailResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("pakasir detail unmarshal: %w", err)
	}

	return &result, nil
}

// CancelTransaction cancels a pending transaction.
func (c *Client) CancelTransaction(ctx context.Context, project, orderID, apiKey string, amount int) error {
	reqBody := CreateTransactionRequest{
		Project: project,
		OrderID: orderID,
		Amount:  amount,
		APIKey:  apiKey,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("pakasir cancel marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/transactioncancel", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pakasir cancel request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("pakasir cancel execute: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pakasir cancel status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SimulatePayment triggers a payment simulation (sandbox only).
func (c *Client) SimulatePayment(ctx context.Context, project, orderID, apiKey string, amount int) error {
	reqBody := CreateTransactionRequest{
		Project: project,
		OrderID: orderID,
		Amount:  amount,
		APIKey:  apiKey,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("pakasir simulate marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/paymentsimulation", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("pakasir simulate request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("pakasir simulate execute: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pakasir simulate status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
