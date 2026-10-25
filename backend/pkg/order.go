package pkg

import "github.com/shopspring/decimal"

type LineInput struct {
	ProductID      string `json:"product_id" validate:"required,uuid"`
	Quantity       string `json:"quantity"   validate:"required"`
	UnitPrice      string `json:"unit_price" validate:"required"`
	DiscountAmount string `json:"discount_amount"`
}

type CalculatedLine struct {
	ProductID      string
	Quantity       decimal.Decimal
	UnitPrice      decimal.Decimal
	DiscountAmount decimal.Decimal
	LineTotal      decimal.Decimal
}

func CalculateOrder(items []LineInput, discount string) ([]CalculatedLine, decimal.Decimal, decimal.Decimal, decimal.Decimal, error) {
	if len(items) == 0 || len(items) > 1000 {
		return nil, decimal.Zero, decimal.Zero, decimal.Zero, ErrInvalidInput
	}
	seen := make(map[string]bool)
	lines := make([]CalculatedLine, len(items))
	subtotal := decimal.Zero
	for i, item := range items {
		if ValidateID(item.ProductID) != nil || seen[item.ProductID] {
			return nil, subtotal, decimal.Zero, decimal.Zero, ErrInvalidInput
		}
		seen[item.ProductID] = true
		quantity, err := ParseDecimal(item.Quantity, 3, true)
		if err != nil {
			return nil, subtotal, decimal.Zero, decimal.Zero, err
		}
		price, err := ParseDecimal(item.UnitPrice, 2, false)
		if err != nil {
			return nil, subtotal, decimal.Zero, decimal.Zero, err
		}
		if item.DiscountAmount == "" {
			item.DiscountAmount = "0"
		}
		lineDiscount, err := ParseDecimal(item.DiscountAmount, 2, false)
		if err != nil {
			return nil, subtotal, decimal.Zero, decimal.Zero, err
		}
		gross := quantity.Mul(price).Round(2)
		if lineDiscount.GreaterThan(gross) {
			return nil, subtotal, decimal.Zero, decimal.Zero, ErrInvalidInput
		}
		total := gross.Sub(lineDiscount)
		if total.GreaterThanOrEqual(decimal.New(1, 16)) {
			return nil, subtotal, decimal.Zero, decimal.Zero, ErrInvalidInput
		}
		lines[i] = CalculatedLine{
			ProductID:      item.ProductID,
			Quantity:       quantity,
			UnitPrice:      price,
			DiscountAmount: lineDiscount,
			LineTotal:      total,
		}
		subtotal = subtotal.Add(total)
	}
	if subtotal.GreaterThanOrEqual(decimal.New(1, 16)) {
		return nil, subtotal, decimal.Zero, decimal.Zero, ErrInvalidInput
	}
	if discount == "" {
		discount = "0"
	}
	orderDiscount, err := ParseDecimal(discount, 2, false)
	if err != nil || orderDiscount.GreaterThan(subtotal) {
		return nil, subtotal, decimal.Zero, decimal.Zero, ErrInvalidInput
	}
	return lines, subtotal, orderDiscount, subtotal.Sub(orderDiscount), nil
}
