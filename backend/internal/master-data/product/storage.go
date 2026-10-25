package product

import (
	"time"

	"github.com/shopspring/decimal"
)

type ProductStorage struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	SKU          string
	Name         string
	CategoryID   string `gorm:"type:uuid"`
	Unit         string
	SellingPrice decimal.Decimal
	ReorderLevel decimal.Decimal
	IsActive     bool `gorm:"not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (ProductStorage) TableName() string {
	return "products"
}
