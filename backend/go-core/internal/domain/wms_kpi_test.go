package domain

import "testing"

func TestKPIRatioPct_NoDataIsNil(t *testing.T) {
	if got := KPIRatioPct(0, 0); got != nil {
		t.Fatalf("no samples must be nil (shown as 'belum ada data'), got %v", *got)
	}
}

func TestKPIRatioPct_ComputesAndRounds(t *testing.T) {
	cases := []struct {
		num, den float64
		want     float64
	}{
		{1, 1, 100},
		{0, 4, 0},
		{2, 3, 66.7},
		{199, 200, 99.5},
	}
	for _, c := range cases {
		got := KPIRatioPct(c.num, c.den)
		if got == nil || *got != c.want {
			t.Fatalf("KPIRatioPct(%v,%v) = %v, want %v", c.num, c.den, got, c.want)
		}
	}
}

func TestKPIRatioPct_NeverExceeds100OrBelow0(t *testing.T) {
	if got := KPIRatioPct(5, 4); got == nil || *got != 100 {
		t.Fatalf("clamp high: %v", got)
	}
	if got := KPIRatioPct(-1, 4); got == nil || *got != 0 {
		t.Fatalf("clamp low: %v", got)
	}
}

func TestKPIAvg_NoSamplesIsNil(t *testing.T) {
	if got := KPIAvg(0, 0); got != nil {
		t.Fatalf("avg with zero samples must be nil, got %v", *got)
	}
	if got := KPIAvg(37.035, 3); got == nil || *got != 12.3 { // 12.345 -> 12.3
		t.Fatalf("avg rounding: %v", got)
	}
}
