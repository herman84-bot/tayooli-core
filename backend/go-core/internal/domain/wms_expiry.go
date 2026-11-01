package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Batch expiry control (OCA wms §1.3 FEFO: a lot past its expiry date is not
// pickable; ADR-014: every stock line is tied to a batch, so expiry is judged
// per batch, never per product).

// ErrExpiredStockBlocked is returned when the only stock that could satisfy an
// outbound request (delivery order, POS, transfer, marketplace, manifest) is in
// batches whose expiry date has passed. Expired stock can still leave the rack
// through Scrap, Opname or Quarantine.
var ErrExpiredStockBlocked = errors.New("stok sudah kedaluwarsa (expired) dan tidak boleh dijual, dikirim, atau ditransfer")

// IsOutboundStockRef reports whether a movement reference takes stock out of the
// warehouse to a customer or another warehouse. Expired batches are blocked for
// these; scrap, opname, putaway, QC and quarantine moves stay allowed.
func IsOutboundStockRef(refType string) bool {
	switch refType {
	case StockRefDeliveryOrder, StockRefPOS, StockRefTransfer, StockRefMarketplace:
		return true
	}
	return false
}

// ExpiryClock returns the current time. Tests override it; production uses time.Now.
var ExpiryClock = time.Now

// expiryLocation is the business calendar used to decide "today". Batch expiry
// dates are calendar dates printed on Indonesian packaging, so a lot expiring on
// 15 Oct must be blocked from 00:00 WIB on 16 Oct, not 07:00 WIB (00:00 UTC).
var expiryLocation = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Jakarta"); err == nil {
		return loc
	}
	return time.FixedZone("WIB", 7*60*60)
}()

// expiryDay normalises a time to its calendar day at UTC midnight so two days
// can be compared regardless of their original location.
func expiryDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// BusinessToday is today's calendar date in the business time zone (WIB).
func BusinessToday(now time.Time) time.Time {
	return expiryDay(now.In(expiryLocation).Date())
}

// DaysUntilExpiry returns how many whole days remain until expiry relative to
// now (0 = expires today, negative = already expired). The expiry value is a
// calendar date (Postgres DATE), so its own wall-clock date is used.
func DaysUntilExpiry(expiry time.Time, now time.Time) int {
	e := expiryDay(expiry.Date())
	return int(e.Sub(BusinessToday(now)).Hours() / 24)
}

// IsBatchExpired reports whether a batch with this expiry date is past it.
// A batch is still usable on its expiry date itself and blocked from the next
// day, matching the release rule `expiry_date >= CURRENT_DATE`. Batches without
// an expiry date never expire.
func IsBatchExpired(expiry *time.Time, now time.Time) bool {
	if expiry == nil {
		return false
	}
	return DaysUntilExpiry(*expiry, now) < 0
}

// Expiry severity buckets for the expiry control report.
const (
	ExpirySeverityExpired  = "EXPIRED"  // past expiry date: blocked from outbound
	ExpirySeverityCritical = "CRITICAL" // expires within ExpiryCriticalDays
	ExpirySeverityWarning  = "WARNING"  // expires within the requested window
)

// ExpiryCriticalDays is the window flagged as critical in the report.
const ExpiryCriticalDays = 7

// ExpirySeverity classifies days-until-expiry into a report bucket.
func ExpirySeverity(daysLeft int) string {
	switch {
	case daysLeft < 0:
		return ExpirySeverityExpired
	case daysLeft <= ExpiryCriticalDays:
		return ExpirySeverityCritical
	default:
		return ExpirySeverityWarning
	}
}

// ExpiringBatchLine is one batch at one location in the expiry control report.
type ExpiringBatchLine struct {
	BatchID       uuid.UUID        `json:"batch_id"`
	BatchNumber   string           `json:"batch_number"`
	ExpiryDate    time.Time        `json:"expiry_date"`
	DaysLeft      int              `json:"days_left"`
	Severity      string           `json:"severity"`
	BatchStatus   StockBatchStatus `json:"batch_status"`
	ProductID     uuid.UUID        `json:"product_id"`
	ProductName   string           `json:"product_name"`
	ProductSKU    string           `json:"product_sku"`
	WarehouseID   uuid.UUID        `json:"warehouse_id"`
	WarehouseName string           `json:"warehouse_name"`
	LocationID    uuid.UUID        `json:"location_id"`
	LocationCode  string           `json:"location_code"`
	LocationType  LocationType     `json:"location_type"`
	Quantity      decimal.Decimal  `json:"quantity"`
}

// ExpiryReportSummary totals the report per severity.
type ExpiryReportSummary struct {
	ExpiredBatches  int             `json:"expired_batches"`
	ExpiredQty      decimal.Decimal `json:"expired_qty"`
	CriticalBatches int             `json:"critical_batches"`
	CriticalQty     decimal.Decimal `json:"critical_qty"`
	WarningBatches  int             `json:"warning_batches"`
	WarningQty      decimal.Decimal `json:"warning_qty"`
}

// ExpiryReport lists on-hand batches that are expired or expire within Days.
type ExpiryReport struct {
	Days    int                 `json:"days"`
	AsOf    string              `json:"as_of"` // business date (YYYY-MM-DD, WIB)
	Summary ExpiryReportSummary `json:"summary"`
	Lines   []ExpiringBatchLine `json:"lines"`
}
