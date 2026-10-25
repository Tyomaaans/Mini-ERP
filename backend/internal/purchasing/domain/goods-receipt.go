package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type GoodsReceiptEntity struct {
	ID          string
	GRNumber    string
	OrderID     string
	WarehouseID string
	Status      string
	ReceivedAt  *time.Time
	ReceivedBy  string
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Items       []GoodsReceiptItemEntity
}

type GoodsReceiptItemEntity struct {
	ID          string
	DocumentID  string
	OrderItemID string
	Quantity    decimal.Decimal
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GoodsReceiptRepository interface {
	CreateGoodsReceipt(ctx context.Context, entity *GoodsReceiptEntity) error
	GetGoodsReceipt(ctx context.Context, id string, lock bool) (*GoodsReceiptEntity, error)
	GetGoodsReceipts(ctx context.Context, query pkg.ListQuery, scope []string) ([]GoodsReceiptEntity, int64, error)
	Post(ctx context.Context, id, actorID string) error
}
