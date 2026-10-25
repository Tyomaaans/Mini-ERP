package user

import (
	"context"
	"errors"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"mini-erp/internal/auth-session/domain"
	"mini-erp/internal/auth-session/session"
	"mini-erp/internal/auth-session/token"
	"mini-erp/pkg"
)

// Interface

type UserService interface {
	Login(ctx context.Context, req LoginRequest) (*token.TokenResponse, error)
	Refresh(ctx context.Context, req RefreshRequest) (*token.TokenResponse, error)
	Authenticate(ctx context.Context, access string) (pkg.Actor, string, error)
	Logout(ctx context.Context, sessionID string) error
	GetUser(ctx context.Context, id string) (*UserResponse, error)
	GetUsers(ctx context.Context, query pkg.ListQuery) ([]UserResponse, int64, error)
	CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error)
	UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*UserResponse, error)
	AssignRoles(ctx context.Context, id string, req AssignRolesRequest) (*UserResponse, error)
	AssignWarehouses(ctx context.Context, id string, req AssignWarehousesRequest) (*UserResponse, error)
}

// Implementation

type userService struct {
	userRepo    domain.UserRepository
	tokenSvc    token.JWTService
	sessionSvc  session.SessionService
	validate    *validator.Validate
	txManager   pkg.TransactionManager
	auditLogger pkg.AuditLogger
}

func NewUserService(
	userRepo    domain.UserRepository,
	tokenSvc    token.JWTService,
	sessionSvc  session.SessionService,
	validate    *validator.Validate,
	txManager   pkg.TransactionManager,
	auditLogger pkg.AuditLogger,
) UserService {
	return &userService{
		userRepo:    userRepo,
		tokenSvc:    tokenSvc,
		sessionSvc:  sessionSvc,
		validate:    validate,
		txManager:   txManager,
		auditLogger: auditLogger,
	}
}

func (s *userService) CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("users.create"); err != nil {
		return nil, err
	}

	req.FullName = strings.TrimSpace(req.FullName)
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	if len(req.Password) > 72 {
		return nil, pkg.ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	entity := &domain.UserEntity{
		ID:           uuid.NewString(),
		DepartmentID: req.DepartmentID,
		FullName:     req.FullName,
		Email:        strings.ToLower(req.Email),
		PasswordHash: string(hash),
		IsActive:     true,
	}

	err = s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.userRepo.CreateUser(ctx, entity); err != nil {
			return err
		}

		if err := s.userRepo.AssignRoles(ctx, entity.ID, req.RoleIDs); err != nil {
			return err
		}

		if err := s.userRepo.AssignWarehouses(ctx, entity.ID, req.WarehouseIDs); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "users.create", "USER", entity.ID, nil, map[string]any{"full_name": entity.FullName, "email": entity.Email, "role_ids": req.RoleIDs, "warehouse_ids": req.WarehouseIDs})
	})
	if err != nil {
		return nil, err
	}

	return s.response(ctx, entity.ID)
}

func (s *userService) UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*UserResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("users.update"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	fields := make(map[string]any)
	if req.FullName != nil {
		value := strings.TrimSpace(*req.FullName)
		if value == "" {
			return nil, pkg.ErrInvalidInput
		}

		fields["full_name"] = value
	}

	if req.Email != nil {
		fields["email"] = strings.ToLower(*req.Email)
	}

	if req.DepartmentID != nil {
		fields["department_id"] = *req.DepartmentID
	}

	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if req.Password != nil {
		if len(*req.Password) > 72 {
			return nil, pkg.ErrInvalidInput
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		fields["password_hash"] = string(hash)
	}

	if len(fields) == 0 {
		return nil, pkg.ErrInvalidInput
	}

	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.userRepo.GuardAdministrator(ctx, id, req.IsActive != nil && !*req.IsActive); err != nil {
			return err
		}

		old, err := s.response(ctx, id)
		if err != nil {
			return err
		}

		if err := s.userRepo.UpdateUser(ctx, id, fields); err != nil {
			return err
		}

		updated, err := s.response(ctx, id)
		if err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "users.update", "USER", id, old, updated)
	})
	if err != nil {
		return nil, err
	}

	return s.response(ctx, id)
}

func (s *userService) GetUser(ctx context.Context, id string) (*UserResponse, error) {
	if err := pkg.ActorFrom(ctx).Require("users.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	return s.response(ctx, id)
}

func (s *userService) GetUsers(ctx context.Context, query pkg.ListQuery) ([]UserResponse, int64, error) {
	if err := pkg.ActorFrom(ctx).Require("users.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.userRepo.GetUsers(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	response := make([]UserResponse, len(entities))
	for i := range entities {
		response[i] = *ToUserResponse(&entities[i], pkg.Actor{
			Roles:        entities[i].Roles,
			WarehouseIDs: entities[i].WarehouseIDs,
		})
	}

	return response, total, nil
}

func (s *userService) Login(ctx context.Context, req LoginRequest) (*token.TokenResponse, error) {
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	entity, err := s.userRepo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(req.Email)))
	if errors.Is(err, pkg.ErrNotFound) {
		_, _ = bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		return nil, pkg.ErrInvalidCredentials
	}

	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(entity.PasswordHash), []byte(req.Password)); err != nil || !entity.IsActive {
		return nil, pkg.ErrInvalidCredentials
	}

	sid, refresh, err := s.sessionSvc.Create(ctx, entity.ID, session.CredentialStamp(entity.PasswordHash, entity.UpdatedAt))
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.LoginAt(ctx, entity.ID); err != nil {
		_ = s.sessionSvc.Revoke(ctx, sid)
		return nil, err
	}

	pair, err := s.tokenSvc.Generate(entity.ID, sid, refresh)
	if err != nil {
		return nil, err
	}

	return token.ToTokenResponse(pair), nil
}

func (s *userService) Refresh(ctx context.Context, req RefreshRequest) (*token.TokenResponse, error) {
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	userID, sid, refresh, err := s.sessionSvc.Rotate(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	entity, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	_, stamp, err := s.sessionSvc.Check(ctx, sid)
	if err != nil {
		return nil, err
	}

	if !entity.IsActive || stamp != session.CredentialStamp(entity.PasswordHash, entity.UpdatedAt) {
		_ = s.sessionSvc.Revoke(ctx, sid)
		return nil, pkg.ErrUnauthorized
	}

	pair, err := s.tokenSvc.Generate(userID, sid, refresh)
	if err != nil {
		return nil, err
	}

	return token.ToTokenResponse(pair), nil
}

func (s *userService) Authenticate(ctx context.Context, access string) (pkg.Actor, string, error) {
	claims, err := s.tokenSvc.Parse(access)
	if err != nil {
		return pkg.Actor{}, "", err
	}

	userID, stamp, err := s.sessionSvc.Check(ctx, claims.SessionID)
	if err != nil {
		return pkg.Actor{}, "", err
	}

	if userID != claims.UserID {
		return pkg.Actor{}, "", pkg.ErrUnauthorized
	}

	entity, err := s.userRepo.GetUser(ctx, userID)
	if err != nil {
		return pkg.Actor{}, "", err
	}

	if !entity.IsActive || stamp != session.CredentialStamp(entity.PasswordHash, entity.UpdatedAt) {
		return pkg.Actor{}, "", pkg.ErrUnauthorized
	}

	actor, err := s.userRepo.GetActor(ctx, userID)
	return actor, claims.SessionID, err
}

func (s *userService) Logout(ctx context.Context, sessionID string) error {
	if pkg.ActorFrom(ctx).ID == "" {
		return pkg.ErrUnauthorized
	}

	return s.sessionSvc.Revoke(ctx, sessionID)
}

func (s *userService) AssignRoles(ctx context.Context, id string, req AssignRolesRequest) (*UserResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("users.assign_role"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		if _, err := s.userRepo.GetUser(ctx, id); err != nil {
			return err
		}

		if err := s.userRepo.AssignRoles(ctx, id, req.RoleIDs); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "users.assign_role", "USER", id, nil, req)
	})
	if err != nil {
		return nil, err
	}

	return s.response(ctx, id)
}

func (s *userService) AssignWarehouses(ctx context.Context, id string, req AssignWarehousesRequest) (*UserResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("users.assign_warehouse"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		if _, err := s.userRepo.GetUser(ctx, id); err != nil {
			return err
		}

		if err := s.userRepo.AssignWarehouses(ctx, id, req.WarehouseIDs); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "users.assign_warehouse", "USER", id, nil, req)
	})
	if err != nil {
		return nil, err
	}

	return s.response(ctx, id)
}

// Internal Helper

func (s *userService) response(ctx context.Context, id string) (*UserResponse, error) {
	entity, err := s.userRepo.GetUser(ctx, id)
	if err != nil {
		return nil, err
	}

	actor, err := s.userRepo.GetActor(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToUserResponse(entity, actor), nil
}
