package shipment

import (
	"time"

	"github.com/shopspring/decimal"
)

type ShipmentStorage struct {
	ID             string `gorm:"type:uuid;primaryKey"`
	ShipmentNumber string
	OrderID        string `gorm:"column:sales_order_id"`
	WarehouseID    string
	Status         string
	ShippedAt      *time.Time
	ShippedBy      *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Items          []ShipmentItemStorage `gorm:"foreignKey:DocumentID"`
}

func (ShipmentStorage) TableName() string {
	return "shipments"
}

type ShipmentItemStorage struct {
	ID          string          `gorm:"type:uuid;primaryKey"`
	DocumentID  string          `gorm:"column:shipment_id"`
	OrderItemID string          `gorm:"column:sales_order_item_id"`
	Quantity    decimal.Decimal `gorm:"column:quantity_shipped"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (ShipmentItemStorage) TableName() string {
	return "shipment_items"
}
