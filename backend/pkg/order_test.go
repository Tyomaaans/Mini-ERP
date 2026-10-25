package pkg

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimalValidationPreservesDatabasePrecision(t *testing.T) {
	tests := []struct {
		raw      string
		scale    int32
		positive bool
		valid    bool
	}{
		{"0.00", 2, false, true}, {"0", 3, true, false}, {"0.001", 3, true, true},
		{"9999999999999999.99", 2, false, true}, {"10000000000000000", 2, false, false},
		{"999999999999999.999", 3, true, true}, {"1000000000000000", 3, true, false},
		{"NaN", 2, false, false}, {"Infinity", 2, false, false}, {"-1", 2, false, false},
		{"1e3", 2, false, false}, {"1.001", 2, false, false}, {" 1.00", 2, false, false}, {"+1", 2, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			_, err := ParseDecimal(tt.raw, tt.scale, tt.positive)
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", tt.valid, err)
			}
		})
	}
}

func TestOrderTotalsRoundLinesBeforeApplyingOrderDiscount(t *testing.T) {
	items := []LineInput{{
		ProductID:      "00000000-0000-0000-0000-000000000001",
		Quantity:       "0.125",
		UnitPrice:      "123.45",
		DiscountAmount: "1.00",
	}}
	lines, subtotal, discount, total, err := CalculateOrder(items, "0.43")
	if err != nil {
		t.Fatal(err)
	}
	if lines[0].LineTotal.StringFixed(2) != "14.43" || subtotal.StringFixed(2) != "14.43" || discount.StringFixed(2) != "0.43" || total.StringFixed(2) != "14.00" {
		t.Fatalf("incorrect totals: %v %s %s %s", lines, subtotal, discount, total)
	}
	for _, bad := range [][]LineInput{nil, {items[0], items[0]}, {{
		ProductID:      items[0].ProductID,
		Quantity:       "1",
		UnitPrice:      "1",
		DiscountAmount: "2",
	}}, {{
		ProductID: items[0].ProductID,
		Quantity:  "999999999999999.999",
		UnitPrice: "9999999999999999.99",
	}}} {
		if _, _, _, _, err := CalculateOrder(bad, "0"); err == nil {
			t.Fatalf("invalid order accepted: %v", bad)
		}
	}
}

func TestGeneratedOrderAmountInvariants(t *testing.T) {
	for seed := uint64(0); seed < 200; seed++ {
		checkOrderAmountInvariants(t, seed)
	}
}

func FuzzOrderAmountInvariants(f *testing.F) {
	for _, seed := range []uint64{0, 1, 42, 999, ^uint64(0)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed uint64) {
		checkOrderAmountInvariants(t, seed)
	})
}

func TestOrderAmountMaximumAcrossMultipleLines(t *testing.T) {
	items := []LineInput{
		{
			ProductID: "00000000-0000-0000-0000-000000000001",
			Quantity:  "1",
			UnitPrice: "9999999999999999.98",
		},
		{
			ProductID: "00000000-0000-0000-0000-000000000002",
			Quantity:  "1",
			UnitPrice: "0.01",
		},
	}
	_, subtotal, _, total, err := CalculateOrder(items, "0")
	if err != nil || subtotal.StringFixed(2) != "9999999999999999.99" || !total.Equal(subtotal) {
		t.Fatalf("maximum supported amount: subtotal=%s total=%s error=%v", subtotal, total, err)
	}
	items[1].UnitPrice = "0.02"
	if _, _, _, _, err := CalculateOrder(items, "0"); err == nil {
		t.Fatal("multi-line subtotal overflow was accepted")
	}
	items = items[:1]
	items[0].Quantity = "999999999999999.999"
	items[0].UnitPrice = "0.01"
	_, _, _, total, err = CalculateOrder(items, "0")
	if err != nil || total.StringFixed(2) != "10000000000000.00" {
		t.Fatalf("maximum quantity rounding: total=%s error=%v", total, err)
	}
}

// Internal Helper

func checkOrderAmountInvariants(t testing.TB, seed uint64) {
	t.Helper()
	random := rand.New(rand.NewSource(int64(seed)))
	items := make([]LineInput, 1+random.Intn(8))
	expectedLines := make([]int64, len(items))
	var subtotalCents int64
	for i := range items {
		quantityMilli := 1 + random.Int63n(1000000)
		priceCents := random.Int63n(1000000000)
		// Input bounds keep intermediate products within int64.
		grossCents := (quantityMilli*priceCents + 500) / 1000
		discountCents := random.Int63n(grossCents + 1)
		expectedLines[i] = grossCents - discountCents
		subtotalCents += expectedLines[i]
		items[i] = LineInput{
			ProductID:      fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1),
			Quantity:       fmt.Sprintf("%d.%03d", quantityMilli/1000, quantityMilli%1000),
			UnitPrice:      fmt.Sprintf("%d.%02d", priceCents/100, priceCents%100),
			DiscountAmount: fmt.Sprintf("%d.%02d", discountCents/100, discountCents%100),
		}
	}
	orderDiscountCents := random.Int63n(subtotalCents + 1)
	discountInput := fmt.Sprintf("%d.%02d", orderDiscountCents/100, orderDiscountCents%100)
	lines, subtotal, discount, total, err := CalculateOrder(items, discountInput)
	if err != nil {
		t.Fatalf("seed=%d: %v", seed, err)
	}
	if !subtotal.Equal(decimal.New(subtotalCents, -2)) || !discount.Equal(decimal.New(orderDiscountCents, -2)) || !total.Equal(decimal.New(subtotalCents-orderDiscountCents, -2)) {
		t.Fatalf("seed=%d: subtotal=%s discount=%s total=%s", seed, subtotal, discount, total)
	}
	for i, line := range lines {
		if !line.LineTotal.Equal(decimal.New(expectedLines[i], -2)) || line.LineTotal.IsNegative() {
			t.Fatalf("seed=%d line=%d: total=%s", seed, i, line.LineTotal)
		}
	}
}
