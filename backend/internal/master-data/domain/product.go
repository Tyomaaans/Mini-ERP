package domain

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"mini-erp/pkg"
)

type ProductEntity struct {
	ID           string
	SKU          string
	Name         string
	CategoryID   string
	Unit         string
	SellingPrice decimal.Decimal
	ReorderLevel decimal.Decimal
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ProductRepository interface {
	CreateProduct(ctx context.Context, entity *ProductEntity) error
	UpdateProduct(ctx context.Context, id string, fields map[string]any) error
	GetProduct(ctx context.Context, id string) (*ProductEntity, error)
	GetProducts(ctx context.Context, query pkg.ListQuery, scope []string) ([]ProductEntity, int64, error)
}
