package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func d(n int64) decimal.Decimal { return decimal.NewFromInt(n) }
func sp(s string) *string       { return &s }

func TestEvaluateQC(t *testing.T) {
	b1, b2 := uuid.New(), uuid.New()
	exp := []QCExpectedLine{{ProductID: uuid.New(), BatchID: b1, ExpectedQty: d(10)}, {ProductID: uuid.New(), BatchID: b2, ExpectedQty: d(5)}}
	base := func(items ...QCLineInput) QCInspectionInput {
		return QCInspectionInput{InspectionMode: QCModeFull, GrossCartons: 3, Items: items}
	}

	t.Run("all good = QC_PASSED with discrepancy report", func(t *testing.T) {
		c, err := EvaluateQC(base(QCLineInput{BatchID: b1, CheckedQty: d(8)}, QCLineInput{BatchID: b2, CheckedQty: d(6)}), exp)
		require.NoError(t, err)
		assert.Equal(t, QCStatusPassed, c.Status)
		assert.True(t, c.Shortage.Equal(d(2)))
		assert.True(t, c.Overage.Equal(d(1)))
		assert.True(t, c.Passed.Equal(d(14)))
	})
	t.Run("partial damage = QUARANTINED, needs BAK driver", func(t *testing.T) {
		in := base(QCLineInput{BatchID: b1, CheckedQty: d(10), DamagedQty: d(3), DamageReason: sp("penyok")}, QCLineInput{BatchID: b2, CheckedQty: d(5)})
		_, err := EvaluateQC(in, exp)
		assert.ErrorIs(t, err, ErrQCBAKDriverRequired)
		in.DriverName, in.DriverSigned = sp("Budi"), true
		c, err := EvaluateQC(in, exp)
		require.NoError(t, err)
		assert.Equal(t, QCStatusQuarantined, c.Status)
		assert.True(t, c.Damaged.Equal(d(3)))
		assert.True(t, c.Items[0].PassedQty.Equal(d(7)))
	})
	t.Run("everything damaged = QC_REJECTED", func(t *testing.T) {
		in := base(QCLineInput{BatchID: b1, CheckedQty: d(10), DamagedQty: d(10), DamageReason: sp("basah")}, QCLineInput{BatchID: b2, CheckedQty: d(5), DamagedQty: d(5), DamageReason: sp("basah")})
		in.DriverName, in.DriverSigned = sp("Budi"), true
		c, err := EvaluateQC(in, exp)
		require.NoError(t, err)
		assert.Equal(t, QCStatusRejected, c.Status)
	})
	t.Run("sampling with damage forces FULL", func(t *testing.T) {
		in := base(QCLineInput{BatchID: b1, CheckedQty: d(2), DamagedQty: d(1), DamageReason: sp("x")}, QCLineInput{BatchID: b2, CheckedQty: d(1)})
		in.InspectionMode, in.SampleQty = QCModeSampling, ptrDec(d(3))
		in.DriverName, in.DriverSigned = sp("Budi"), true
		_, err := EvaluateQC(in, exp)
		assert.ErrorIs(t, err, ErrQCSamplingFailed)
	})
	t.Run("invalid inputs", func(t *testing.T) {
		cases := map[string]QCInspectionInput{
			"missing line":       base(QCLineInput{BatchID: b1, CheckedQty: d(10)}),
			"unknown batch":      base(QCLineInput{BatchID: b1, CheckedQty: d(10)}, QCLineInput{BatchID: uuid.New(), CheckedQty: d(5)}),
			"duplicate batch":    base(QCLineInput{BatchID: b1, CheckedQty: d(1)}, QCLineInput{BatchID: b1, CheckedQty: d(1)}),
			"damaged > checked":  base(QCLineInput{BatchID: b1, CheckedQty: d(1), DamagedQty: d(2), DamageReason: sp("x")}, QCLineInput{BatchID: b2, CheckedQty: d(5)}),
			"negative checked":   base(QCLineInput{BatchID: b1, CheckedQty: d(-1)}, QCLineInput{BatchID: b2, CheckedQty: d(5)}),
			"damage w/o reason":  base(QCLineInput{BatchID: b1, CheckedQty: d(5), DamagedQty: d(1), DamageReason: sp("  ")}, QCLineInput{BatchID: b2, CheckedQty: d(5)}),
			"bad mode":           {InspectionMode: "X", Items: []QCLineInput{{BatchID: b1}}},
			"sampling no sample": {InspectionMode: QCModeSampling, Items: []QCLineInput{{BatchID: b1}, {BatchID: b2}}},
			"negative cartons":   {InspectionMode: QCModeFull, GrossCartons: -1, Items: []QCLineInput{{BatchID: b1}, {BatchID: b2}}},
		}
		for name, in := range cases {
			_, err := EvaluateQC(in, exp)
			assert.ErrorIs(t, err, ErrInvalidInput, name)
		}
	})
	t.Run("damaged above staged qty rejected", func(t *testing.T) {
		in := base(QCLineInput{BatchID: b1, CheckedQty: d(12), DamagedQty: d(11), DamageReason: sp("x")}, QCLineInput{BatchID: b2, CheckedQty: d(5)})
		in.DriverName, in.DriverSigned = sp("Budi"), true
		_, err := EvaluateQC(in, exp)
		assert.ErrorIs(t, err, ErrQCStagedQtyChanged)
	})
	assert.Equal(t, "BAK-GR-001", BAKNumber("GR-001"))
}

func ptrDec(v decimal.Decimal) *decimal.Decimal { return &v }
