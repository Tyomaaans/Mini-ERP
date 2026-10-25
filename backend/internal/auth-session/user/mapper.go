package user

import (
	"mini-erp/internal/auth-session/domain"
	"mini-erp/pkg"
)

// User to Storage <-> Entity

func ToUserEntity(s UserStorage) domain.UserEntity {
	return domain.UserEntity{
		ID:           s.ID,
		DepartmentID: s.DepartmentID,
		FullName:     s.FullName,
		Email:        s.Email,
		PasswordHash: s.PasswordHash,
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func ToUserStorage(e *domain.UserEntity) *UserStorage {
	return &UserStorage{
		ID:           e.ID,
		DepartmentID: e.DepartmentID,
		FullName:     e.FullName,
		Email:        e.Email,
		PasswordHash: e.PasswordHash,
		IsActive:     e.IsActive,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}

// Entity to Response

func ToUserResponse(e *domain.UserEntity, actor pkg.Actor) *UserResponse {
	return &UserResponse{
		ID:           e.ID,
		DepartmentID: e.DepartmentID,
		FullName:     e.FullName,
		Email:        e.Email,
		IsActive:     e.IsActive,
		Roles:        actor.Roles,
		WarehouseIDs: actor.WarehouseIDs,
		CreatedAt:    e.CreatedAt,
	}
}

func ToMeResponse(actor pkg.Actor) *MeResponse {
	return &MeResponse{
		ID:           actor.ID,
		Name:         actor.Name,
		Department:   actor.Department,
		Roles:        actor.Roles,
		Permissions:  actor.Permissions,
		WarehouseIDs: actor.WarehouseIDs,
	}
}
