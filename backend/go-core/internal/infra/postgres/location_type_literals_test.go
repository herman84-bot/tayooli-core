package postgres

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/herman84-bot/Erp-Like-PAPER-ID/backend/go-core/internal/domain"
)

// Regression for live /wms/stock 500: ListStockSummary compared
// location_type to 'STAGING', which is not a value of the Postgres enum, so
// every call failed with 22P02. Unit tests use mocks and never run the SQL,
// so this checks every literal compared against a location type column.
func TestSQLLocationTypeLiteralsAreValidEnumValues(t *testing.T) {
	valid := map[string]bool{}
	for _, v := range []domain.LocationType{
		domain.LocationTypeInternal, domain.LocationTypeVendor, domain.LocationTypeCustomer,
		domain.LocationTypeTransit, domain.LocationTypeLoss, domain.LocationTypeScrap,
		domain.LocationTypeProduction, domain.LocationTypeStagingInbound,
		domain.LocationTypeStagingOutbound, domain.LocationTypeQuarantine,
	} {
		valid[string(v)] = true
	}

	// e.g. "loc.type IN ('A','B')", "location_type = 'A'", "type != 'A'"
	cmp := regexp.MustCompile(`(?i)\b(?:\w+\.)?(?:location_)?type\s*(?:NOT\s+IN|IN|=|!=|<>)\s*(\([^)]*\)|'[^']*')`)
	lit := regexp.MustCompile(`'([^']*)'`)

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range cmp.FindAllStringSubmatch(string(src), -1) {
			for _, l := range lit.FindAllStringSubmatch(m[1], -1) {
				v := l[1]
				// Only uppercase enum-looking literals; other "type" columns
				// (mapping_type, receipt_type...) are excluded by the regex
				// word boundary but keep the guard narrow anyway.
				if v == "" || v != strings.ToUpper(v) {
					continue
				}
				checked++
				if !valid[v] && looksLikeLocationContext(m[0]) {
					t.Errorf("%s: %q is not a location_type enum value in %q", f, v, m[0])
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no location type literals found; regex is broken")
	}
}

func looksLikeLocationContext(expr string) bool {
	e := strings.ToLower(expr)
	return strings.HasPrefix(e, "loc.") || strings.HasPrefix(e, "location_type") ||
		strings.HasPrefix(e, "ms.location_type") || strings.Contains(e, "location")
}
