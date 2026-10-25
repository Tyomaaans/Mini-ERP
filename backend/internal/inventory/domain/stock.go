package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type StockLine struct {
	ItemID      string
	OrderItemID string
	ProductID   string
	Quantity    decimal.Decimal
}

type BalanceEntity struct {
	ID               string
	ProductID        string
	WarehouseID      string
	QuantityOnHand   decimal.Decimal
	QuantityReserved decimal.Decimal
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MovementEntity struct {
	ID              string
	ProductID       string
	WarehouseID     string
	MovementType    string
	QuantityDelta   decimal.Decimal
	ReferenceType   string
	ReferenceID     string
	ReferenceItemID string
	PerformedBy     string
	OccurredAt      time.Time
}

type StockRepository interface {
	LockBalances(ctx context.Context, warehouseID string, lines []StockLine) error
	ChangeBalance(ctx context.Context, warehouseID, productID string, onHand, reserved decimal.Decimal) error
	CreateMovement(ctx context.Context, actorID, warehouseID, movementType, referenceID string, line StockLine) error
	CreateReservation(ctx context.Context, actorID, warehouseID string, line StockLine) error
	ConsumeReservations(ctx context.Context, itemID string, quantity decimal.Decimal) error
	ReleaseReservations(ctx context.Context, orderID string) ([]StockLine, error)
	GetBalances(ctx context.Context, query pkg.ListQuery, scope []string) ([]BalanceEntity, int64, error)
	GetMovements(ctx context.Context, query pkg.ListQuery, scope []string) ([]MovementEntity, int64, error)
}

type StockService interface {
	Receive(ctx context.Context, warehouseID, receiptID string, lines []StockLine) error
	Reserve(ctx context.Context, warehouseID string, lines []StockLine) error
	Ship(ctx context.Context, warehouseID, shipmentID string, lines []StockLine) error
	Release(ctx context.Context, warehouseID, orderID string) error
}
