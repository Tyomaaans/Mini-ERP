package stock

import "time"

type BalanceResponse struct {
	ID                string    `json:"id"`
	ProductID         string    `json:"product_id"`
	WarehouseID       string    `json:"warehouse_id"`
	QuantityOnHand    string    `json:"quantity_on_hand"`
	QuantityReserved  string    `json:"quantity_reserved"`
	QuantityAvailable string    `json:"quantity_available"`
	Version           int64     `json:"version"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type MovementResponse struct {
	ID              string    `json:"id"`
	ProductID       string    `json:"product_id"`
	WarehouseID     string    `json:"warehouse_id"`
	MovementType    string    `json:"movement_type"`
	QuantityDelta   string    `json:"quantity_delta"`
	ReferenceType   string    `json:"reference_type"`
	ReferenceID     string    `json:"reference_id"`
	ReferenceItemID string    `json:"reference_item_id"`
	PerformedBy     string    `json:"performed_by"`
	OccurredAt      time.Time `json:"occurred_at"`
}
