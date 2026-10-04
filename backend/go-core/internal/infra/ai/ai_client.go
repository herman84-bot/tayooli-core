package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the FastAPI AI serving service.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new AI service client with the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// IngestRequest is the payload sent to the inference ingest endpoint.
type IngestRequest struct {
	InvoiceID     string `json:"invoice_id"`
	TenantID      string `json:"tenant_id"`
	Amount        string `json:"amount"`
	VendorID      string `json:"vendor_id"`
	ExtractedText string `json:"extracted_text"`
}

// IngestResponse is returned after a successful ingest call.
type IngestResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

// StatusResponse holds the inference status for an invoice.
type StatusResponse struct {
	InvoiceID          string  `json:"invoice_id"`
	Status             string  `json:"status"`
	AnomalyScore       float64 `json:"anomaly_score,omitempty"`
	SuggestedGLAccount string  `json:"suggested_gl_account,omitempty"`
	Error              string  `json:"error,omitempty"`
}

// Ingest submits an invoice for AI inference (OCR + anomaly detection).
func (c *Client) Ingest(ctx context.Context, req IngestRequest) (*IngestResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("ai ingest marshal: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/inference/ingest", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("ai ingest request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai ingest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("ai ingest: unexpected status %d", resp.StatusCode)
	}

	var result IngestResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("ai ingest decode: %w", err)
	}
	return &result, nil
}

// GetStatus retrieves the inference status for a given invoice.
// The tenantID is passed as a query parameter so the AI service can enforce tenant isolation.
func (c *Client) GetStatus(ctx context.Context, invoiceID, tenantID string) (*StatusResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/inference/status/"+invoiceID+"?tenant_id="+tenantID, nil)
	if err != nil {
		return nil, fmt.Errorf("ai status request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ai status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("ai status: not found for invoice %s", invoiceID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai status: unexpected status %d", resp.StatusCode)
	}

	var result StatusResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("ai status decode: %w", err)
	}
	return &result, nil
}
