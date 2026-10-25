package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type ShipmentEntity struct {
	ID             string
	ShipmentNumber string
	OrderID        string
	WarehouseID    string
	Status         string
	ShippedAt      *time.Time
	ShippedBy      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Items          []ShipmentItemEntity
}

type ShipmentItemEntity struct {
	ID          string
	DocumentID  string
	OrderItemID string
	Quantity    decimal.Decimal
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ShipmentRepository interface {
	CreateShipment(ctx context.Context, entity *ShipmentEntity) error
	GetShipment(ctx context.Context, id string, lock bool) (*ShipmentEntity, error)
	GetShipments(ctx context.Context, query pkg.ListQuery, scope []string) ([]ShipmentEntity, int64, error)
	Post(ctx context.Context, id, actorID string) error
}
