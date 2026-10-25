package salesorder

import (
	"time"

	"github.com/shopspring/decimal"
)

type SalesOrderStorage struct {
	ID             string `gorm:"type:uuid;primaryKey"`
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
	Items          []OrderItemStorage `gorm:"foreignKey:OrderID"`
}

func (SalesOrderStorage) TableName() string {
	return "sales_orders"
}

type OrderItemStorage struct {
	ID             string `gorm:"type:uuid;primaryKey"`
	OrderID        string `gorm:"column:sales_order_id"`
	ProductID      string
	Quantity       decimal.Decimal
	UnitPrice      decimal.Decimal
	DiscountAmount decimal.Decimal
	LineTotal      decimal.Decimal
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (OrderItemStorage) TableName() string {
	return "sales_order_items"
}
