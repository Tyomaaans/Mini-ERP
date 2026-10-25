package invoice

import (
	"time"

	"github.com/shopspring/decimal"
)

type InvoiceStorage struct {
	ID              string `gorm:"type:uuid;primaryKey"`
	InvoiceNumber   string
	InvoiceType     string
	SalesOrderID    *string
	PurchaseOrderID *string
	CustomerID      *string
	SupplierID      *string
	Status          string
	IssueDate       time.Time
	DueDate         time.Time
	TotalAmount     decimal.Decimal
	PaidAmount      decimal.Decimal
	CreatedBy       string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (InvoiceStorage) TableName() string {
	return "invoices"
}
