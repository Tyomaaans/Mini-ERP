package domain

import (
	"context"
	"time"

	"mini-erp/pkg"
)

type WarehouseEntity struct {
	ID        string
	Code      string
	Name      string
	Address   string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type WarehouseRepository interface {
	CreateWarehouse(ctx context.Context, entity *WarehouseEntity) error
	UpdateWarehouse(ctx context.Context, id string, fields map[string]any) error
	GetWarehouse(ctx context.Context, id string) (*WarehouseEntity, error)
	GetWarehouses(ctx context.Context, query pkg.ListQuery, scope []string) ([]WarehouseEntity, int64, error)
}
