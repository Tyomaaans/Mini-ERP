package stock

import "mini-erp/internal/inventory/domain"

func ToBalanceResponse(e domain.BalanceEntity) BalanceResponse {
	return BalanceResponse{
		ID:                e.ID,
		ProductID:         e.ProductID,
		WarehouseID:       e.WarehouseID,
		QuantityOnHand:    e.QuantityOnHand.StringFixed(3),
		QuantityReserved:  e.QuantityReserved.StringFixed(3),
		QuantityAvailable: e.QuantityOnHand.Sub(e.QuantityReserved).StringFixed(3),
		Version:           e.Version,
		UpdatedAt:         e.UpdatedAt,
	}
}

func ToMovementResponse(e domain.MovementEntity) MovementResponse {
	return MovementResponse{
		ID:              e.ID,
		ProductID:       e.ProductID,
		WarehouseID:     e.WarehouseID,
		MovementType:    e.MovementType,
		QuantityDelta:   e.QuantityDelta.StringFixed(3),
		ReferenceType:   e.ReferenceType,
		ReferenceID:     e.ReferenceID,
		ReferenceItemID: e.ReferenceItemID,
		PerformedBy:     e.PerformedBy,
		OccurredAt:      e.OccurredAt,
	}
}
