package domain

import (
	"context"
	"time"

	"mini-erp/pkg"
)

type SupplierEntity struct {
	ID               string
	Code             string
	Name             string
	Email            string
	Phone            string
	Address          string
	PaymentTermsDays int
	IsActive         bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type SupplierRepository interface {
	CreateSupplier(ctx context.Context, entity *SupplierEntity) error
	UpdateSupplier(ctx context.Context, id string, fields map[string]any) error
	GetSupplier(ctx context.Context, id string) (*SupplierEntity, error)
	GetSuppliers(ctx context.Context, query pkg.ListQuery, scope []string) ([]SupplierEntity, int64, error)
}
