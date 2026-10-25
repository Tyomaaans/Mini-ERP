package warehouse

import "time"

// Warehouse Requests

type CreateWarehouseRequest struct {
	Code    string `json:"code" validate:"required,max=30"`
	Name    string `json:"name" validate:"required,max=100"`
	Address string `json:"address"`
}

type UpdateWarehouseRequest struct {
	Code     *string `json:"code" validate:"omitempty,max=30"`
	Name     *string `json:"name" validate:"omitempty,max=100"`
	Address  *string `json:"address"`
	IsActive *bool   `json:"is_active"`
}

// Warehouse Response

type WarehouseResponse struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
