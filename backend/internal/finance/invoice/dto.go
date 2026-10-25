package invoice

import "time"

type CreateInvoiceRequest struct {
	InvoiceType     string  `json:"invoice_type"      validate:"required,oneof=AR AP"`
	SalesOrderID    *string `json:"sales_order_id"    validate:"omitempty,uuid"`
	PurchaseOrderID *string `json:"purchase_order_id" validate:"omitempty,uuid"`
	IssueDate       string  `json:"issue_date"        validate:"required"`
	DueDate         string  `json:"due_date"          validate:"required"`
}

type UpdateInvoiceRequest struct {
	IssueDate string `json:"issue_date" validate:"required"`
	DueDate   string `json:"due_date"   validate:"required"`
}

type InvoiceResponse struct {
	ID                string    `json:"id"`
	InvoiceNumber     string    `json:"invoice_number"`
	InvoiceType       string    `json:"invoice_type"`
	SalesOrderID      *string   `json:"sales_order_id"`
	PurchaseOrderID   *string   `json:"purchase_order_id"`
	CustomerID        *string   `json:"customer_id"`
	SupplierID        *string   `json:"supplier_id"`
	Status            string    `json:"status"`
	IssueDate         string    `json:"issue_date"`
	DueDate           string    `json:"due_date"`
	TotalAmount       string    `json:"total_amount"`
	PaidAmount        string    `json:"paid_amount"`
	OutstandingAmount string    `json:"outstanding_amount"`
	CreatedBy         string    `json:"created_by"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
