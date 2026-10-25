package purchaseorder

import (
	"time"

	"github.com/shopspring/decimal"
)

type PurchaseOrderStorage struct {
	ID             string `gorm:"type:uuid;primaryKey"`
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
	Items          []OrderItemStorage `gorm:"foreignKey:OrderID"`
}

func (PurchaseOrderStorage) TableName() string {
	return "purchase_orders"
}

type OrderItemStorage struct {
	ID             string `gorm:"type:uuid;primaryKey"`
	OrderID        string `gorm:"column:purchase_order_id"`
	ProductID      string
	Quantity       decimal.Decimal
	UnitPrice      decimal.Decimal
	DiscountAmount decimal.Decimal
	LineTotal      decimal.Decimal
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (OrderItemStorage) TableName() string {
	return "purchase_order_items"
}
