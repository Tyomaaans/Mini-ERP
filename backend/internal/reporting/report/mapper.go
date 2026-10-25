package report

import (
	"encoding/json"
	"time"

	"mini-erp/internal/reporting/domain"
)

func ToSummaryResponse(e domain.SummaryEntity) SummaryResponse {
	return SummaryResponse{
		Date:        e.Date,
		OrderCount:  e.OrderCount,
		TotalAmount: e.TotalAmount.StringFixed(2),
	}
}

func ToLowStockResponse(e domain.LowStockEntity) LowStockResponse {
	return LowStockResponse{
		ProductID:         e.ProductID,
		WarehouseID:       e.WarehouseID,
		SKU:               e.SKU,
		Name:              e.Name,
		Warehouse:         e.Warehouse,
		QuantityOnHand:    e.QuantityOnHand.StringFixed(3),
		QuantityReserved:  e.QuantityReserved.StringFixed(3),
		QuantityAvailable: e.QuantityAvailable.StringFixed(3),
		ReorderLevel:      e.ReorderLevel.StringFixed(3),
	}
}

func ToOutstandingResponse(e domain.OutstandingEntity) OutstandingResponse {
	return OutstandingResponse{
		ID:                e.ID,
		InvoiceNumber:     e.InvoiceNumber,
		PartyID:           e.PartyID,
		PartyName:         e.PartyName,
		IssueDate:         e.IssueDate.Format(time.DateOnly),
		DueDate:           e.DueDate.Format(time.DateOnly),
		TotalAmount:       e.TotalAmount.StringFixed(2),
		PaidAmount:        e.PaidAmount.StringFixed(2),
		OutstandingAmount: e.OutstandingAmount.StringFixed(2),
	}
}

func rawJSON(value *string) json.RawMessage {
	if value == nil {
		return json.RawMessage("null")
	}

	return json.RawMessage(*value)
}

func ToAuditResponse(e domain.AuditEntity) AuditResponse {
	return AuditResponse{
		ID:         e.ID,
		ActorID:    e.ActorID,
		Action:     e.Action,
		EntityType: e.EntityType,
		EntityID:   e.EntityID,
		OldValues:  rawJSON(e.OldValues),
		NewValues:  rawJSON(e.NewValues),
		RequestID:  e.RequestID,
		OccurredAt: e.OccurredAt,
	}
}
