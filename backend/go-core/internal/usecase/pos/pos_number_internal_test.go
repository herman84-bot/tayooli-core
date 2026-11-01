package pos

import (
	"regexp"
	"testing"
	"time"
)

func TestGeneratePOSNumberFormat(t *testing.T) {
	re := regexp.MustCompile(`^(POS|MV-POS|SO-POS|INV-POS)-\d{8}-[0-9ABCDEFGHJKMNPQRSTVWXYZ]{10}$`)
	today := time.Now().UTC().Format("20060102")
	for _, prefix := range []string{"POS", "MV-POS", "SO-POS", "INV-POS"} {
		n := generatePOSNumber(prefix)
		if !re.MatchString(n) {
			t.Fatalf("unexpected number format %q", n)
		}
		if got := n[len(prefix)+1 : len(prefix)+9]; got != today {
			t.Fatalf("date part = %q, want %q (%s)", got, today, n)
		}
		if len(n) > 50 {
			t.Fatalf("number %q longer than the narrowest column (VARCHAR(50))", n)
		}
	}
}

// The old 4-digit suffix had 9,000 values per day: a busy day collided on
// pos_orders/stock_movements unique keys and checkout returned 500. 200k numbers
// (20x a very busy day) must all be distinct.
func TestGeneratePOSNumberNoCollisions(t *testing.T) {
	const n = 200_000
	seen := make(map[string]struct{}, n)
	for i := 0; i < n; i++ {
		v := generatePOSNumber("POS")
		if _, dup := seen[v]; dup {
			t.Fatalf("duplicate POS number after %d draws: %s", i, v)
		}
		seen[v] = struct{}{}
	}
}

func TestGeneratePOSNumberUsesWholeAlphabet(t *testing.T) {
	counts := map[rune]int{}
	for i := 0; i < 20_000; i++ {
		v := generatePOSNumber("POS")
		for _, r := range v[len(v)-posNumberRandLen:] {
			counts[r]++
		}
	}
	if len(counts) != len(posNumberAlphabet) {
		t.Fatalf("used %d distinct symbols, want %d", len(counts), len(posNumberAlphabet))
	}
	// 200k symbols / 32 = 6250 expected each; allow wide margin for randomness.
	for r, c := range counts {
		if c < 5000 || c > 7500 {
			t.Fatalf("symbol %q drawn %d times, distribution looks biased", r, c)
		}
	}
}
