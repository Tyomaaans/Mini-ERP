package salesorder

import (
	"time"

	"mini-erp/pkg"
)

// SalesOrder Requests

type CreateSalesOrderRequest struct {
	CustomerID  string `json:"customer_id"  validate:"required,uuid"`
	WarehouseID string `json:"warehouse_id" validate:"required,uuid"`
	OrderDate   string `json:"order_date"   validate:"required"`

	DiscountAmount string          `json:"discount_amount"`
	Notes          *string         `json:"notes"        validate:"omitempty,max=5000"`
	Items          []pkg.LineInput `json:"items"        validate:"required,min=1,max=1000,dive"`
}

type UpdateSalesOrderRequest struct {
	CustomerID     *string            `json:"customer_id"  validate:"omitempty,uuid"`
	WarehouseID    *string            `json:"warehouse_id" validate:"omitempty,uuid"`
	OrderDate      *string            `json:"order_date"`
	DiscountAmount *string            `json:"discount_amount"`
	Notes          pkg.NullableString `json:"notes"`
	Items          *[]pkg.LineInput   `json:"items"        validate:"omitempty,min=1,max=1000,dive"`
}

type DecisionRequest struct {
	Note string `json:"note" validate:"required,max=1000"`
}

// SalesOrder Responses

type SalesOrderResponse struct {
	ID             string              `json:"id"`
	SONumber       string              `json:"so_number"`
	CustomerID     string              `json:"customer_id"`
	WarehouseID    string              `json:"warehouse_id"`
	Status         string              `json:"status"`
	OrderDate      string              `json:"order_date"`
	ConfirmedAt    *time.Time          `json:"confirmed_at"`
	Subtotal       string              `json:"subtotal"`
	DiscountAmount string              `json:"discount_amount"`
	TotalAmount    string              `json:"total_amount"`
	Notes          *string             `json:"notes"`
	CreatedBy      string              `json:"created_by"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	Items          []OrderItemResponse `json:"items"`
}

type OrderItemResponse struct {
	ID             string `json:"id"`
	ProductID      string `json:"product_id"`
	Quantity       string `json:"quantity"`
	UnitPrice      string `json:"unit_price"`
	DiscountAmount string `json:"discount_amount"`
	LineTotal      string `json:"line_total"`
}
