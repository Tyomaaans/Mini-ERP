package product

import "time"

// Product Requests

type CreateProductRequest struct {
	SKU          string `json:"sku"           validate:"required,max=50"`
	Name         string `json:"name"          validate:"required,max=200"`
	CategoryID   string `json:"category_id"   validate:"required,uuid"`
	Unit         string `json:"unit"          validate:"required,max=20"`
	SellingPrice string `json:"selling_price" validate:"required"`
	ReorderLevel string `json:"reorder_level" validate:"required"`
}

type UpdateProductRequest struct {
	SKU          *string `json:"sku"         validate:"omitempty,max=50"`
	Name         *string `json:"name"        validate:"omitempty,max=200"`
	CategoryID   *string `json:"category_id" validate:"omitempty,uuid"`
	Unit         *string `json:"unit"        validate:"omitempty,max=20"`
	SellingPrice *string `json:"selling_price"`
	ReorderLevel *string `json:"reorder_level"`
	IsActive     *bool   `json:"is_active"`
}

// Product Response

type ProductResponse struct {
	ID           string    `json:"id"`
	SKU          string    `json:"sku"`
	Name         string    `json:"name"`
	CategoryID   string    `json:"category_id"`
	Unit         string    `json:"unit"`
	SellingPrice string    `json:"selling_price"`
	ReorderLevel string    `json:"reorder_level"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
