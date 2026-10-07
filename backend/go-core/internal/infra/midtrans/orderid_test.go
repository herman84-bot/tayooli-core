package midtrans

import (
	"crypto/sha512"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
)

func TestOrderIDRoundTrip(t *testing.T) {
	tid := uuid.New()
	oid := BuildOrderID(tid)
	if len(oid) > 50 {
		t.Fatalf("order id too long: %d", len(oid))
	}
	got, err := TenantFromOrderID(oid)
	if err != nil || got != tid {
		t.Fatalf("roundtrip failed: %v %v", got, err)
	}
	if BuildOrderID(tid) == oid {
		t.Fatal("order ids must be unique")
	}
	for _, bad := range []string{"", "POS-123", "X" + oid[1:], oid + "x"} {
		if _, err := TenantFromOrderID(bad); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestVerifySignature(t *testing.T) {
	sum := sha512.Sum512([]byte("ORD1" + "200" + "10000.00" + "SK"))
	sig := hex.EncodeToString(sum[:])
	if !VerifySignature("ORD1", "200", "10000.00", "SK", sig) {
		t.Fatal("valid signature rejected")
	}
	if VerifySignature("ORD1", "200", "99999.00", "SK", sig) {
		t.Fatal("tampered amount accepted")
	}
	if VerifySignature("ORD1", "200", "10000.00", "OTHER", sig) {
		t.Fatal("wrong key accepted")
	}
	if VerifySignature("ORD1", "200", "10000.00", "", sig) {
		t.Fatal("empty key accepted")
	}
}
