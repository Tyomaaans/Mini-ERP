package supplier

import "time"

// Supplier Requests

type CreateSupplierRequest struct {
	Code             string `json:"code"               validate:"required,max=30"`
	Name             string `json:"name"               validate:"required,max=150"`
	Email            string `json:"email"              validate:"omitempty,email,max=255"`
	Phone            string `json:"phone"              validate:"max=30"`
	Address          string `json:"address"`
	PaymentTermsDays int    `json:"payment_terms_days" validate:"min=0,max=3650"`
}

type UpdateSupplierRequest struct {
	Code             *string `json:"code"               validate:"omitempty,max=30"`
	Name             *string `json:"name"               validate:"omitempty,max=150"`
	Email            *string `json:"email"              validate:"omitempty,omitempty,email,max=255"`
	Phone            *string `json:"phone"              validate:"omitempty,max=30"`
	Address          *string `json:"address"`
	PaymentTermsDays *int    `json:"payment_terms_days" validate:"omitempty,min=0,max=3650"`
	IsActive         *bool   `json:"is_active"`
}

// Supplier Response

type SupplierResponse struct {
	ID               string    `json:"id"`
	Code             string    `json:"code"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	Phone            string    `json:"phone"`
	Address          string    `json:"address"`
	PaymentTermsDays int       `json:"payment_terms_days"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
