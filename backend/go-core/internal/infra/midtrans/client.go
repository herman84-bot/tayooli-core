package midtrans

import (
	"context"
	"fmt"
	"time"

	midtrans "github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/shopspring/decimal"
)

// Client wraps the Midtrans SDK for Core API.
type Client struct {
	serverKey string
	clientKey string
	env       midtrans.EnvironmentType
	core      *coreapi.Client
}

// NewClient creates a Midtrans client with the provided credentials.
// isProduction=true uses production API; false uses sandbox.
func NewClient(serverKey, clientKey string, isProduction bool) *Client {
	env := midtrans.Sandbox
	if isProduction {
		env = midtrans.Production
	}

	c := &Client{
		serverKey: serverKey,
		clientKey: clientKey,
		env:       env,
		core:      &coreapi.Client{},
	}
	c.core.New(serverKey, env)
	return c
}

// CreateChargeQRISRequest wraps the parameters for QRIS charge.
type CreateChargeQRISRequest struct {
	OrderID         string
	Amount          decimal.Decimal
	ExpiryMinutes   int
	CustomerEmail   string
	CustomerPhone   string
	ItemDescription string
}

// CreateChargeQRISResponse contains the result of a QRIS charge.
type CreateChargeQRISResponse struct {
	OrderID   string
	QRString  string
	ExpiresAt time.Time
}

// CreateChargeQRIS creates a dynamic QRIS charge via Midtrans Core API.
// Returns the QR string (for QRCode.toDataURL on client) and expiry time.
func (c *Client) CreateChargeQRIS(ctx context.Context, req CreateChargeQRISRequest) (*CreateChargeQRISResponse, error) {
	if req.OrderID == "" {
		return nil, fmt.Errorf("CreateChargeQRIS: order_id required")
	}
	if req.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, fmt.Errorf("CreateChargeQRIS: amount must be > 0")
	}
	if req.ExpiryMinutes == 0 {
		req.ExpiryMinutes = 5
	}

	grossAmt := req.Amount.IntPart() // int64

	chargeReq := &coreapi.ChargeReq{
		PaymentType: coreapi.PaymentTypeQris,
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  req.OrderID,
			GrossAmt: grossAmt,
		},
		Qris: &coreapi.QrisDetails{
			Acquirer: "", // Let Midtrans choose default acquirer
		},
	}

	// Add customer details if provided (optional)
	if req.CustomerEmail != "" || req.CustomerPhone != "" {
		chargeReq.CustomerDetails = &midtrans.CustomerDetails{
			Email: req.CustomerEmail,
			Phone: req.CustomerPhone,
		}
	}

	// Add item description if provided
	if req.ItemDescription != "" {
		chargeReq.Items = &[]midtrans.ItemDetails{
			{
				Qty:   1,
				Price: grossAmt,
				Name:  req.ItemDescription,
			},
		}
	}

	// Call Core API
	chargeResp, errMidtrans := c.core.ChargeTransaction(chargeReq)
	if errMidtrans != nil {
		return nil, fmt.Errorf("CreateChargeQRIS: %w", errMidtrans)
	}

	// Extract QR string from the response
	// Midtrans Core API QRIS response includes QRString field
	qrString := chargeResp.QRString
	if qrString == "" {
		return nil, fmt.Errorf("CreateChargeQRIS: no QR string in response (order_id=%s)", req.OrderID)
	}

	// Midtrans default QRIS expiry: 15 minutes from now
	// We use the specified expiry or 5 minutes as safety margin
	expiresAt := time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)

	return &CreateChargeQRISResponse{
		OrderID:   req.OrderID,
		QRString:  qrString,
		ExpiresAt: expiresAt,
	}, nil
}

// TransactionStatus holds the result of a status check.
type TransactionStatus struct {
	Status          string
	Amount          int64
	PaymentType     string
	TransactionTime string
}

// VerifyTransactionStatus re-checks the actual status of a transaction server-to-server.
// This is the authoritative source, not the webhook body.
func (c *Client) VerifyTransactionStatus(ctx context.Context, orderID string) (*TransactionStatus, error) {
	if orderID == "" {
		return nil, fmt.Errorf("VerifyTransactionStatus: order_id required")
	}

	// Core API status check
	txStatus, errMidtrans := c.core.CheckTransaction(orderID)
	if errMidtrans != nil {
		return nil, fmt.Errorf("VerifyTransactionStatus: %w", errMidtrans)
	}

	amount, err := ParseGrossAmount(txStatus.GrossAmount)
	if err != nil {
		return nil, fmt.Errorf("VerifyTransactionStatus: %w", err)
	}

	return &TransactionStatus{
		Status:          txStatus.TransactionStatus,
		Amount:          amount,
		PaymentType:     txStatus.PaymentType,
		TransactionTime: txStatus.TransactionTime,
	}, nil
}

// ParseGrossAmount parses Midtrans gross_amount ("10000.00" or "10000") to whole rupiah.
func ParseGrossAmount(s string) (int64, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return 0, fmt.Errorf("invalid gross_amount %q: %w", s, err)
	}
	return d.IntPart(), nil
}

// IsPaidStatus checks if a Midtrans transaction status represents payment received.
func IsPaidStatus(status string) bool {
	return status == "settlement" || status == "capture"
}
