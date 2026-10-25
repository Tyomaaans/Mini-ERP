package shipment

import "time"

// Shipment Requests

type CreateShipmentRequest struct {
	OrderID     string                `json:"sales_order_id" validate:"required,uuid"`
	WarehouseID string                `json:"warehouse_id"   validate:"omitempty,uuid"`
	Items       []ShipmentItemRequest `json:"items"          validate:"required,min=1,max=1000,dive"`
}

type ShipmentItemRequest struct {
	OrderItemID string `json:"sales_order_item_id" validate:"required,uuid"`
	Quantity    string `json:"quantity_shipped"    validate:"required"`
}

// Shipment Responses

type ShipmentResponse struct {
	ID             string                 `json:"id"`
	ShipmentNumber string                 `json:"shipment_number"`
	OrderID        string                 `json:"sales_order_id"`
	WarehouseID    string                 `json:"warehouse_id"`
	Status         string                 `json:"status"`
	ShippedAt      *time.Time             `json:"shipped_at"`
	ShippedBy      *string                `json:"shipped_by"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Items          []ShipmentItemResponse `json:"items"`
}

type ShipmentItemResponse struct {
	ID          string `json:"id"`
	OrderItemID string `json:"sales_order_item_id"`
	Quantity    string `json:"quantity_shipped"`
}
