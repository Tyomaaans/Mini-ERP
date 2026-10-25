package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	inventory "mini-erp/internal/inventory/domain"
	"mini-erp/pkg"
)

type SalesOrderEntity struct {
	ID             string
	SONumber       string
	CustomerID     string
	WarehouseID    string
	Status         string
	OrderDate      time.Time
	Subtotal       decimal.Decimal
	DiscountAmount decimal.Decimal
	TotalAmount    decimal.Decimal
	Notes          *string
	CreatedBy      string
	ConfirmedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Items          []OrderItemEntity
}

type OrderItemEntity struct {
	ID             string
	OrderID        string
	ProductID      string
	Quantity       decimal.Decimal
	UnitPrice      decimal.Decimal
	DiscountAmount decimal.Decimal
	LineTotal      decimal.Decimal
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SalesOrderRepository interface {
	CreateSalesOrder(ctx context.Context, order *SalesOrderEntity) error
	UpdateSalesOrder(ctx context.Context, order *SalesOrderEntity) error
	GetSalesOrder(ctx context.Context, id string, lock bool) (*SalesOrderEntity, error)
	GetSalesOrders(ctx context.Context, query pkg.ListQuery, scope []string) ([]SalesOrderEntity, int64, error)
	ValidateReferences(ctx context.Context, order *SalesOrderEntity) error
	SetStatus(ctx context.Context, id, status, actorID string) error
	HasActivity(ctx context.Context, id string) (bool, error)
	Shipped(ctx context.Context, id string) (map[string]decimal.Decimal, error)
	Reserved(ctx context.Context, id string) (map[string]decimal.Decimal, error)
	CheckCredit(ctx context.Context, order *SalesOrderEntity) error
	Unreserved(ctx context.Context, id string) ([]inventory.StockLine, error)
}
