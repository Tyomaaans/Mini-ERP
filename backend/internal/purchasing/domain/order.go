package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type PurchaseOrderEntity struct {
	ID             string
	PONumber       string
	SupplierID     string
	WarehouseID    string
	Status         string
	OrderDate      time.Time
	Subtotal       decimal.Decimal
	DiscountAmount decimal.Decimal
	TotalAmount    decimal.Decimal
	Notes          *string
	CreatedBy      string
	ExpectedDate   *time.Time
	SubmittedAt    *time.Time
	ApprovedBy     *string
	ApprovedAt     *time.Time
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

type PurchaseOrderRepository interface {
	CreatePurchaseOrder(ctx context.Context, order *PurchaseOrderEntity) error
	UpdatePurchaseOrder(ctx context.Context, order *PurchaseOrderEntity) error
	GetPurchaseOrder(ctx context.Context, id string, lock bool) (*PurchaseOrderEntity, error)
	GetPurchaseOrders(ctx context.Context, query pkg.ListQuery, scope []string) ([]PurchaseOrderEntity, int64, error)
	ValidateReferences(ctx context.Context, order *PurchaseOrderEntity) error
	SetStatus(ctx context.Context, id, status, actorID string) error
	HasActivity(ctx context.Context, id string) (bool, error)
	RequestApproval(ctx context.Context, id, userID string) error
	DecideApproval(ctx context.Context, id, userID, status, note string) error
	Received(ctx context.Context, id string) (map[string]decimal.Decimal, error)
}
