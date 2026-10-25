package goodsreceipt

import "time"

// GoodsReceipt Requests

type CreateGoodsReceiptRequest struct {
	OrderID     string                    `json:"purchase_order_id" validate:"required,uuid"`
	WarehouseID string                    `json:"warehouse_id"      validate:"omitempty,uuid"`
	Notes       *string                   `json:"notes"             validate:"omitempty,max=5000"`
	Items       []GoodsReceiptItemRequest `json:"items"             validate:"required,min=1,max=1000,dive"`
}

type GoodsReceiptItemRequest struct {
	OrderItemID string `json:"purchase_order_item_id" validate:"required,uuid"`
	Quantity    string `json:"quantity_received"      validate:"required"`
}

// GoodsReceipt Responses

type GoodsReceiptResponse struct {
	ID          string                     `json:"id"`
	GRNumber    string                     `json:"gr_number"`
	OrderID     string                     `json:"purchase_order_id"`
	WarehouseID string                     `json:"warehouse_id"`
	Status      string                     `json:"status"`
	ReceivedAt  *time.Time                 `json:"received_at"`
	ReceivedBy  string                     `json:"received_by"`
	Notes       *string                    `json:"notes"`
	CreatedAt   time.Time                  `json:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at"`
	Items       []GoodsReceiptItemResponse `json:"items"`
}

type GoodsReceiptItemResponse struct {
	ID          string `json:"id"`
	OrderItemID string `json:"purchase_order_item_id"`
	Quantity    string `json:"quantity_received"`
}
