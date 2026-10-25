package domain

import (
	"context"
	"time"

	"mini-erp/pkg"
)

type CategoryEntity struct {
	ID        string
	Code      string
	Name      string
	ParentID  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CategoryRepository interface {
	CreateCategory(ctx context.Context, entity *CategoryEntity) error
	UpdateCategory(ctx context.Context, id string, fields map[string]any) error
	GetCategory(ctx context.Context, id string) (*CategoryEntity, error)
	GetCategories(ctx context.Context, query pkg.ListQuery, scope []string) ([]CategoryEntity, int64, error)
	ValidateParent(ctx context.Context, id string, parent *string) error
	DeleteCategory(ctx context.Context, id string) error
}
