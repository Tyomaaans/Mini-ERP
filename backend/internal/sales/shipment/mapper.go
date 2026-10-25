package shipment

import "mini-erp/internal/sales/domain"

func ToShipmentEntity(s *ShipmentStorage) *domain.ShipmentEntity {
	items := make([]domain.ShipmentItemEntity, len(s.Items))
	for i, row := range s.Items {
		items[i] = domain.ShipmentItemEntity(row)
	}

	return &domain.ShipmentEntity{
		ID:             s.ID,
		ShipmentNumber: s.ShipmentNumber,
		OrderID:        s.OrderID,
		WarehouseID:    s.WarehouseID,
		Status:         s.Status,
		ShippedAt:      s.ShippedAt,
		ShippedBy:      s.ShippedBy,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		Items:          items,
	}
}

func ToShipmentStorage(s *domain.ShipmentEntity) *ShipmentStorage {
	items := make([]ShipmentItemStorage, len(s.Items))
	for i, row := range s.Items {
		items[i] = ShipmentItemStorage(row)
	}

	return &ShipmentStorage{
		ID:             s.ID,
		ShipmentNumber: s.ShipmentNumber,
		OrderID:        s.OrderID,
		WarehouseID:    s.WarehouseID,
		Status:         s.Status,
		ShippedAt:      s.ShippedAt,
		ShippedBy:      s.ShippedBy,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		Items:          items,
	}
}

func ToShipmentResponse(e *domain.ShipmentEntity) *ShipmentResponse {
	items := make([]ShipmentItemResponse, len(e.Items))
	for i, row := range e.Items {
		items[i] = ShipmentItemResponse{
			ID:          row.ID,
			OrderItemID: row.OrderItemID,
			Quantity:    row.Quantity.StringFixed(3),
		}
	}

	return &ShipmentResponse{
		ID:             e.ID,
		ShipmentNumber: e.ShipmentNumber,
		OrderID:        e.OrderID,
		WarehouseID:    e.WarehouseID,
		Status:         e.Status,
		ShippedAt:      e.ShippedAt,
		ShippedBy:      e.ShippedBy,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
		Items:          items,
	}
}
