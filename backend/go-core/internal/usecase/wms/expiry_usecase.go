package wms

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
	"github.com/shopspring/decimal"
)

// Expiry report window bounds (days ahead of today, WIB).
const (
	DefaultExpiryWindowDays = 30
	MaxExpiryWindowDays     = 365
)

// GetExpiryReport lists on-hand batches that are already expired or expire
// within `days` days, per batch and location (OCA wms §1.3 FEFO / expiry
// control). warehouseID nil means every warehouse the user may read.
func (u *Usecase) GetExpiryReport(ctx context.Context, tenantID, userID uuid.UUID, role string, warehouseID *uuid.UUID, days int) (*domain.ExpiryReport, error) {
	if days <= 0 {
		days = DefaultExpiryWindowDays
	}
	if days > MaxExpiryWindowDays {
		days = MaxExpiryWindowDays
	}

	var whIDs []uuid.UUID
	if warehouseID != nil && *warehouseID != uuid.Nil {
		if err := u.ValidateWarehouseReadAccess(ctx, tenantID, userID, role, *warehouseID); err != nil {
			return nil, err
		}
		whIDs = []uuid.UUID{*warehouseID}
	} else {
		whs, err := u.FilterAccessibleWarehouses(ctx, tenantID, userID, role)
		if err != nil {
			return nil, fmt.Errorf("GetExpiryReport: warehouses: %w", err)
		}
		for _, w := range whs {
			whIDs = append(whIDs, w.ID)
		}
	}

	now := domain.ExpiryClock()
	today := domain.BusinessToday(now)
	until := today.AddDate(0, 0, days)

	report := &domain.ExpiryReport{
		Days: days,
		AsOf: today.Format("2006-01-02"),
		Summary: domain.ExpiryReportSummary{
			ExpiredQty:  decimal.Zero,
			CriticalQty: decimal.Zero,
			WarningQty:  decimal.Zero,
		},
		Lines: []domain.ExpiringBatchLine{},
	}
	if len(whIDs) == 0 {
		return report, nil
	}

	lines, err := u.repo.ListExpiringBatchBalances(ctx, tenantID, whIDs, until)
	if err != nil {
		return nil, fmt.Errorf("GetExpiryReport: %w", err)
	}

	s := &report.Summary
	for _, l := range lines {
		l.DaysLeft = domain.DaysUntilExpiry(l.ExpiryDate, now)
		if l.DaysLeft > days {
			continue // defensive: repo already filters on until
		}
		l.Severity = domain.ExpirySeverity(l.DaysLeft)
		switch l.Severity {
		case domain.ExpirySeverityExpired:
			s.ExpiredBatches++
			s.ExpiredQty = s.ExpiredQty.Add(l.Quantity)
		case domain.ExpirySeverityCritical:
			s.CriticalBatches++
			s.CriticalQty = s.CriticalQty.Add(l.Quantity)
		default:
			s.WarningBatches++
			s.WarningQty = s.WarningQty.Add(l.Quantity)
		}
		report.Lines = append(report.Lines, l)
	}
	return report, nil
}
