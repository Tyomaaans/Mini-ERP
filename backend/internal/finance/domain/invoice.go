package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type InvoiceEntity struct {
	ID              string
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

type InvoiceSource struct {
	ID          string
	PartyID     string
	Status      string
	TotalAmount decimal.Decimal
}

type InvoiceRepository interface {
	LoadSource(ctx context.Context, invoiceType, id string, lock bool) (*InvoiceSource, error)
	CreateInvoice(ctx context.Context, entity *InvoiceEntity) error
	GetInvoice(ctx context.Context, id string, lock bool) (*InvoiceEntity, error)
	GetInvoices(ctx context.Context, query pkg.ListQuery) ([]InvoiceEntity, int64, error)
	UpdateInvoice(ctx context.Context, id string, fields map[string]any) error
}
