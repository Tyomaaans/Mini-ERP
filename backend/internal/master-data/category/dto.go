package category

import (
	"time"

	"mini-erp/pkg"
)

// Category Requests

type CreateCategoryRequest struct {
	Code     string  `json:"code"      validate:"required,max=30"`
	Name     string  `json:"name"      validate:"required,max=100"`
	ParentID *string `json:"parent_id" validate:"omitempty,uuid"`
}

type UpdateCategoryRequest struct {
	Code     *string            `json:"code" validate:"omitempty,max=30"`
	Name     *string            `json:"name" validate:"omitempty,max=100"`
	ParentID pkg.NullableString `json:"parent_id"`
}

// Category Response

type CategoryResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	ParentID  *string   `json:"parent_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
