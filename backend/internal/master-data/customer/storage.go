package customer

import (
	"time"

	"github.com/shopspring/decimal"
)

type CustomerStorage struct {
	ID              string `gorm:"type:uuid;primaryKey"`
	Code            string
	Name            string
	Email           string
	Phone           string
	BillingAddress  string
	ShippingAddress string
	CreditLimit     decimal.Decimal
	IsActive        bool `gorm:"not null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (CustomerStorage) TableName() string {
	return "customers"
}
