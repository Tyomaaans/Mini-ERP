package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type SummaryEntity struct {
	ID          string
	Date        string
	OrderCount  int64
	TotalAmount decimal.Decimal
}

type LowStockEntity struct {
	ID                string
	ProductID         string
	WarehouseID       string
	SKU               string
	Name              string
	Warehouse         string
	QuantityOnHand    decimal.Decimal
	QuantityReserved  decimal.Decimal
	QuantityAvailable decimal.Decimal
	ReorderLevel      decimal.Decimal
}

type OutstandingEntity struct {
	ID                string
	InvoiceNumber     string
	PartyID           string
	PartyName         string
	IssueDate         time.Time
	DueDate           time.Time
	TotalAmount       decimal.Decimal
	PaidAmount        decimal.Decimal
	OutstandingAmount decimal.Decimal
}

type AuditEntity struct {
	ID         string
	ActorID    *string
	Action     string
	EntityType string
	EntityID   string
	OldValues  *string
	NewValues  *string
	RequestID  string
	OccurredAt time.Time
}

type ReportRepository interface {
	Summary(ctx context.Context, kind string, query pkg.ListQuery, scope []string) ([]SummaryEntity, int64, error)
	LowStock(ctx context.Context, query pkg.ListQuery, scope []string) ([]LowStockEntity, int64, error)
	Outstanding(ctx context.Context, kind string, query pkg.ListQuery) ([]OutstandingEntity, int64, error)
	AuditLogs(ctx context.Context, query pkg.ListQuery) ([]AuditEntity, int64, error)
}
