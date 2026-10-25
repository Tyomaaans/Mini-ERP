package user

import "time"

// User Requests

type CreateUserRequest struct {
	DepartmentID string   `json:"department_id" validate:"required,uuid"`
	FullName     string   `json:"full_name"     validate:"required,max=150"`
	Email        string   `json:"email"         validate:"required,email,max=255"`
	Password     string   `json:"password"      validate:"required,min=12,max=72"`
	RoleIDs      []string `json:"role_ids"      validate:"required,min=1,dive,uuid"`
	WarehouseIDs []string `json:"warehouse_ids" validate:"dive,uuid"`
}

type UpdateUserRequest struct {
	DepartmentID *string `json:"department_id" validate:"omitempty,uuid"`
	FullName     *string `json:"full_name"     validate:"omitempty,min=1,max=150"`
	Email        *string `json:"email"         validate:"omitempty,email,max=255"`
	Password     *string `json:"password"      validate:"omitempty,min=12,max=72"`
	IsActive     *bool   `json:"is_active"`
}

type AssignRolesRequest struct {
	RoleIDs []string `json:"role_ids" validate:"required,min=1,dive,uuid"`
}

type AssignWarehousesRequest struct {
	WarehouseIDs []string `json:"warehouse_ids" validate:"required,dive,uuid"`
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,max=72"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required,max=200"`
}

// User Responses

type UserResponse struct {
	ID           string    `json:"id"`
	DepartmentID string    `json:"department_id"`
	FullName     string    `json:"full_name"`
	Email        string    `json:"email"`
	IsActive     bool      `json:"is_active"`
	Roles        []string  `json:"roles"`
	WarehouseIDs []string  `json:"warehouse_ids"`
	CreatedAt    time.Time `json:"created_at"`
}

type MeResponse struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Department   string   `json:"department"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
	WarehouseIDs []string `json:"warehouse_ids"`
}
