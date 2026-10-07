package midtrans

import (
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// BuildOrderID embeds the tenant id in the Midtrans order_id so that the
// public webhook (which carries no auth/tenant context) can resolve the
// owning tenant without bypassing RLS. Format: T<32 hex tenant>-<8 hex rand>
// (42 chars; Midtrans max is 50).
func BuildOrderID(tenantID uuid.UUID) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return "T" + strings.ReplaceAll(tenantID.String(), "-", "") + "-" + hex.EncodeToString(b)
}

// TenantFromOrderID extracts the tenant id from an order id built by BuildOrderID.
func TenantFromOrderID(orderID string) (uuid.UUID, error) {
	if len(orderID) != 42 || orderID[0] != 'T' || orderID[33] != '-' {
		return uuid.Nil, fmt.Errorf("unrecognized order_id format")
	}
	return uuid.Parse(orderID[1:33])
}

// VerifySignature checks Midtrans notification signature:
// SHA512(order_id + status_code + gross_amount + server_key), constant-time compare.
func VerifySignature(orderID, statusCode, grossAmount, serverKey, signature string) bool {
	if serverKey == "" || signature == "" {
		return false
	}
	sum := sha512.Sum512([]byte(orderID + statusCode + grossAmount + serverKey))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(signature))) == 1
}
