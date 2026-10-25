package payment

import (
	"time"

	"github.com/shopspring/decimal"
)

type PaymentStorage struct {
	ID              string `gorm:"type:uuid;primaryKey"`
	PaymentNumber   string
	InvoiceID       string
	Amount          decimal.Decimal
	PaymentMethod   string
	ReferenceNumber *string
	Status          string
	PaidAt          time.Time
	CreatedBy       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (PaymentStorage) TableName() string {
	return "payments"
}
