package domain

import (
	"context"

	"mini-erp/pkg"
)

type RoleEntity struct {
	ID            string
	Code          string
	Name          string
	Description   *string
	PermissionIDs []string
}

type LookupEntity struct {
	ID     string
	Code   string
	Name   string
	Module string
	Action string
}

type RoleRepository interface {
	GetRole(ctx context.Context, id string) (*RoleEntity, error)
	GetRoles(ctx context.Context, query pkg.ListQuery) ([]RoleEntity, int64, error)
	SaveRole(ctx context.Context, entity *RoleEntity, create bool) error
	DeleteRole(ctx context.Context, id string) error
	Lookups(ctx context.Context, resource string, query pkg.ListQuery) ([]LookupEntity, int64, error)
}
