package stock

import (
	"time"

	"github.com/shopspring/decimal"
)

type BalanceStorage struct {
	ID               string `gorm:"type:uuid;primaryKey"`
	ProductID        string
	WarehouseID      string
	QuantityOnHand   decimal.Decimal
	QuantityReserved decimal.Decimal
	Version          int64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (BalanceStorage) TableName() string {
	return "inventory_balances"
}

type MovementStorage struct {
	ID              string `gorm:"type:uuid;primaryKey"`
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

func (MovementStorage) TableName() string {
	return "stock_movements"
}
