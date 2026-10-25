package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type CustomerEntity struct {
	ID              string
	Code            string
	Name            string
	Email           string
	Phone           string
	BillingAddress  string
	ShippingAddress string
	CreditLimit     decimal.Decimal
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CustomerRepository interface {
	CreateCustomer(ctx context.Context, entity *CustomerEntity) error
	UpdateCustomer(ctx context.Context, id string, fields map[string]any) error
	GetCustomer(ctx context.Context, id string) (*CustomerEntity, error)
	GetCustomers(ctx context.Context, query pkg.ListQuery, scope []string) ([]CustomerEntity, int64, error)
}
