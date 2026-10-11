package domain

import "math"

// KPIWindowDays is the rolling period every WMS KPI is computed over.
const KPIWindowDays = 30

// KPIRatioPct returns num/den as a percentage rounded to 1 decimal, or nil
// when there is no sample (den <= 0). A missing sample must never be reported
// as 100% (or 0%): the dashboard shows it as "belum ada data".
func KPIRatioPct(num, den float64) *float64 {
	if den <= 0 {
		return nil
	}
	v := math.Round(num/den*1000) / 10
	v = math.Max(0, math.Min(100, v))
	return &v
}

// KPIAvg returns the rounded average sum/count, or nil when count is 0.
func KPIAvg(sum float64, count int) *float64 {
	if count <= 0 {
		return nil
	}
	v := math.Round(sum/float64(count)*10) / 10
	return &v
}
