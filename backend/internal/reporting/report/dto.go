package report

import (
	"encoding/json"
	"time"
)

type SummaryResponse struct {
	Date        string `json:"date"`
	OrderCount  int64  `json:"order_count"`
	TotalAmount string `json:"total_amount"`
}

type LowStockResponse struct {
	ProductID         string `json:"product_id"`
	WarehouseID       string `json:"warehouse_id"`
	SKU               string `json:"sku"`
	Name              string `json:"name"`
	Warehouse         string `json:"warehouse"`
	QuantityOnHand    string `json:"quantity_on_hand"`
	QuantityReserved  string `json:"quantity_reserved"`
	QuantityAvailable string `json:"quantity_available"`
	ReorderLevel      string `json:"reorder_level"`
}

type OutstandingResponse struct {
	ID                string `json:"id"`
	InvoiceNumber     string `json:"invoice_number"`
	PartyID           string `json:"party_id"`
	PartyName         string `json:"party_name"`
	IssueDate         string `json:"issue_date"`
	DueDate           string `json:"due_date"`
	TotalAmount       string `json:"total_amount"`
	PaidAmount        string `json:"paid_amount"`
	OutstandingAmount string `json:"outstanding_amount"`
}

type AuditResponse struct {
	ID         string          `json:"id"`
	ActorID    *string         `json:"actor_id"`
	Action     string          `json:"action"`
	EntityType string          `json:"entity_type"`
	EntityID   string          `json:"entity_id"`
	OldValues  json.RawMessage `json:"old_values"`
	NewValues  json.RawMessage `json:"new_values"`
	RequestID  string          `json:"request_id"`
	OccurredAt time.Time       `json:"occurred_at"`
}
