package midtrans

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseGrossAmount(t *testing.T) {
	cases := map[string]int64{"10000.00": 10000, "10000": 10000, "1500.50": 1500}
	for in, want := range cases {
		got, err := ParseGrossAmount(in)
		if err != nil || got != want {
			t.Fatalf("ParseGrossAmount(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	if _, err := ParseGrossAmount("abc"); err == nil {
		t.Fatal("expected error for invalid amount")
	}
}

func TestIsPaidStatus(t *testing.T) {
	for _, s := range []string{"settlement", "capture"} {
		if !IsPaidStatus(s) {
			t.Fatalf("%s should be paid", s)
		}
	}
	for _, s := range []string{"pending", "expire", "deny", "cancel", ""} {
		if IsPaidStatus(s) {
			t.Fatalf("%s should NOT be paid", s)
		}
	}
}

func TestCreateChargeQRISValidation(t *testing.T) {
	c := NewClient("SB-dummy", "", false)
	if _, err := c.CreateChargeQRIS(context.Background(), CreateChargeQRISRequest{Amount: decimal.NewFromInt(1000)}); err == nil {
		t.Fatal("expected error for empty order id")
	}
	if _, err := c.CreateChargeQRIS(context.Background(), CreateChargeQRISRequest{OrderID: "X", Amount: decimal.Zero}); err == nil {
		t.Fatal("expected error for zero amount")
	}
}
