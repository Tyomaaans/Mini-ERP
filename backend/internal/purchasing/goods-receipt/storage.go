package goodsreceipt

import (
	"time"

	"github.com/shopspring/decimal"
)

type GoodsReceiptStorage struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	GRNumber    string
	OrderID     string `gorm:"column:purchase_order_id"`
	WarehouseID string
	Status      string
	ReceivedAt  *time.Time
	ReceivedBy  string
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Items       []GoodsReceiptItemStorage `gorm:"foreignKey:DocumentID"`
}

func (GoodsReceiptStorage) TableName() string {
	return "goods_receipts"
}

type GoodsReceiptItemStorage struct {
	ID          string          `gorm:"type:uuid;primaryKey"`
	DocumentID  string          `gorm:"column:goods_receipt_id"`
	OrderItemID string          `gorm:"column:purchase_order_item_id"`
	Quantity    decimal.Decimal `gorm:"column:quantity_received"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (GoodsReceiptItemStorage) TableName() string {
	return "goods_receipt_items"
}
