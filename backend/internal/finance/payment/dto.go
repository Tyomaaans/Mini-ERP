package payment

import "time"

type CreatePaymentRequest struct {
	InvoiceID       string     `json:"invoice_id"       validate:"required,uuid"`
	Amount          string     `json:"amount"           validate:"required"`
	PaymentMethod   string     `json:"payment_method"   validate:"required,oneof=CASH BANK_TRANSFER VIRTUAL_ACCOUNT"`
	ReferenceNumber *string    `json:"reference_number" validate:"omitempty,max=100"`
	PaidAt          *time.Time `json:"paid_at"`
}

type PaymentResponse struct {
	ID              string    `json:"id"`
	PaymentNumber   string    `json:"payment_number"`
	InvoiceID       string    `json:"invoice_id"`
	Amount          string    `json:"amount"`
	PaymentMethod   string    `json:"payment_method"`
	ReferenceNumber *string   `json:"reference_number"`
	Status          string    `json:"status"`
	PaidAt          time.Time `json:"paid_at"`
	CreatedBy       string    `json:"created_by"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
