package role

import (
	"context"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/auth-session/domain"
	"mini-erp/pkg"
)

// Interface

type RoleService interface {
	GetRole(ctx context.Context, id string) (*RoleResponse, error)
	GetRoles(ctx context.Context, query pkg.ListQuery) ([]RoleResponse, int64, error)
	SaveRole(ctx context.Context, id string, req RoleRequest) (*RoleResponse, error)
	DeleteRole(ctx context.Context, id string) error
	Lookups(ctx context.Context, resource string, query pkg.ListQuery) ([]LookupResponse, int64, error)
}

// Implementation

type roleService struct {
	roleRepo    domain.RoleRepository
	validate    *validator.Validate
	txManager   pkg.TransactionManager
	auditLogger pkg.AuditLogger
}

func NewRoleService(
	roleRepo    domain.RoleRepository,
	validate    *validator.Validate,
	txManager   pkg.TransactionManager,
	auditLogger pkg.AuditLogger,
) RoleService {
	return &roleService{
		roleRepo:    roleRepo,
		validate:    validate,
		txManager:   txManager,
		auditLogger: auditLogger,
	}
}

func (s *roleService) SaveRole(ctx context.Context, id string, req RoleRequest) (*RoleResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("roles.manage"); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	if !regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`).MatchString(req.Code) {
		return nil, pkg.ErrInvalidInput
	}

	create := id == ""
	if create {
		id = uuid.NewString()
	} else if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	entity := &domain.RoleEntity{
		ID:            id,
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		PermissionIDs: req.PermissionIDs,
	}

	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		var old *RoleResponse
		if !create {
			previous, err := s.roleRepo.GetRole(ctx, id)
			if err != nil {
				return err
			}

			if systemRole(previous.Code) {
				return pkg.NewError("SYSTEM_ROLE", "Seeded roles are managed by the RBAC seed; create a custom role instead", 409)
			}

			old = ToRoleResponse(previous)
		}

		if err := s.roleRepo.SaveRole(ctx, entity, create); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "roles.save", "ROLE", id, old, ToRoleResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToRoleResponse(entity), nil
}

func (s *roleService) GetRole(ctx context.Context, id string) (*RoleResponse, error) {
	if err := pkg.ActorFrom(ctx).Require("users.read"); err != nil {
		return nil, err
	}

	if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	entity, err := s.roleRepo.GetRole(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToRoleResponse(entity), nil
}

func (s *roleService) GetRoles(ctx context.Context, query pkg.ListQuery) ([]RoleResponse, int64, error) {
	if err := pkg.ActorFrom(ctx).Require("users.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.roleRepo.GetRoles(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	response := make([]RoleResponse, len(entities))
	for i := range entities {
		response[i] = *ToRoleResponse(&entities[i])
	}

	return response, total, nil
}

func (s *roleService) DeleteRole(ctx context.Context, id string) error {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("roles.manage"); err != nil {
		return err
	}

	if pkg.ValidateID(id) != nil {
		return pkg.ErrInvalidInput
	}

	return s.txManager.Within(ctx, func(ctx context.Context) error {
		entity, err := s.roleRepo.GetRole(ctx, id)
		if err != nil {
			return err
		}

		if systemRole(entity.Code) {
			return pkg.NewError("SYSTEM_ROLE", "Seeded roles cannot be deleted", 409)
		}

		if err := s.roleRepo.DeleteRole(ctx, id); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "roles.delete", "ROLE", id, ToRoleResponse(entity), nil)
	})
}

func (s *roleService) Lookups(ctx context.Context, resource string, query pkg.ListQuery) ([]LookupResponse, int64, error) {
	if err := pkg.ActorFrom(ctx).Require("users.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.roleRepo.Lookups(ctx, resource, query)
	if err != nil {
		return nil, 0, err
	}

	response := make([]LookupResponse, len(entities))
	for i := range entities {
		response[i] = ToLookupResponse(entities[i])
	}

	return response, total, nil
}

// Internal Helper

func systemRole(code string) bool {
	for _, role := range []string{"SUPER_ADMIN", "SALES_STAFF", "PURCHASING_STAFF", "WAREHOUSE_STAFF", "FINANCE_STAFF", "MANAGER"} {
		if code == role {
			return true
		}
	}

	return false
}
