package customer

import "time"

// Customer Requests

type CreateCustomerRequest struct {
	Code            string `json:"code"         validate:"required,max=30"`
	Name            string `json:"name"         validate:"required,max=150"`
	Email           string `json:"email"        validate:"omitempty,email,max=255"`
	Phone           string `json:"phone"        validate:"max=30"`
	BillingAddress  string `json:"billing_address"`
	ShippingAddress string `json:"shipping_address"`
	CreditLimit     string `json:"credit_limit" validate:"required"`
}

type UpdateCustomerRequest struct {
	Code            *string `json:"code"  validate:"omitempty,max=30"`
	Name            *string `json:"name"  validate:"omitempty,max=150"`
	Email           *string `json:"email" validate:"omitempty,omitempty,email,max=255"`
	Phone           *string `json:"phone" validate:"omitempty,max=30"`
	BillingAddress  *string `json:"billing_address"`
	ShippingAddress *string `json:"shipping_address"`
	CreditLimit     *string `json:"credit_limit"`
	IsActive        *bool   `json:"is_active"`
}

// Customer Response

type CustomerResponse struct {
	ID              string    `json:"id"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	Email           string    `json:"email"`
	Phone           string    `json:"phone"`
	BillingAddress  string    `json:"billing_address"`
	ShippingAddress string    `json:"shipping_address"`
	CreditLimit     string    `json:"credit_limit"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
