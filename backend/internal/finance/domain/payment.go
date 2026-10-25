package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type PaymentEntity struct {
	ID              string
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

type PaymentRepository interface {
	CreatePayment(ctx context.Context, entity *PaymentEntity) error
	GetPayment(ctx context.Context, id string, lock bool) (*PaymentEntity, error)
	GetPayments(ctx context.Context, query pkg.ListQuery) ([]PaymentEntity, int64, error)
	PostedAmount(ctx context.Context, invoiceID string) (decimal.Decimal, error)
	Post(ctx context.Context, id string) error
}
