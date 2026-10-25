package domain

import (
	"context"
	"time"

	"mini-erp/pkg"
)

type UserEntity struct {
	ID           string
	DepartmentID string
	FullName     string
	Email        string
	PasswordHash string
	Roles        []string
	WarehouseIDs []string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRepository interface {
	GetUser(ctx context.Context, id string) (*UserEntity, error)
	GetByEmail(ctx context.Context, email string) (*UserEntity, error)
	GetActor(ctx context.Context, id string) (pkg.Actor, error)
	GetUsers(ctx context.Context, query pkg.ListQuery) ([]UserEntity, int64, error)
	CreateUser(ctx context.Context, entity *UserEntity) error
	UpdateUser(ctx context.Context, id string, fields map[string]any) error
	AssignRoles(ctx context.Context, id string, roles []string) error
	AssignWarehouses(ctx context.Context, id string, warehouses []string) error
	GuardAdministrator(ctx context.Context, id string, removing bool) error
	LoginAt(ctx context.Context, id string) error
}
