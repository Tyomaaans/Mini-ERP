package goodsreceipt

import "mini-erp/internal/purchasing/domain"

func ToGoodsReceiptEntity(s *GoodsReceiptStorage) *domain.GoodsReceiptEntity {
	items := make([]domain.GoodsReceiptItemEntity, len(s.Items))
	for i, row := range s.Items {
		items[i] = domain.GoodsReceiptItemEntity(row)
	}

	return &domain.GoodsReceiptEntity{
		ID:          s.ID,
		GRNumber:    s.GRNumber,
		OrderID:     s.OrderID,
		WarehouseID: s.WarehouseID,
		Status:      s.Status,
		ReceivedAt:  s.ReceivedAt,
		ReceivedBy:  s.ReceivedBy,
		Notes:       s.Notes,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		Items:       items,
	}
}

func ToGoodsReceiptStorage(s *domain.GoodsReceiptEntity) *GoodsReceiptStorage {
	items := make([]GoodsReceiptItemStorage, len(s.Items))
	for i, row := range s.Items {
		items[i] = GoodsReceiptItemStorage(row)
	}

	return &GoodsReceiptStorage{
		ID:          s.ID,
		GRNumber:    s.GRNumber,
		OrderID:     s.OrderID,
		WarehouseID: s.WarehouseID,
		Status:      s.Status,
		ReceivedAt:  s.ReceivedAt,
		ReceivedBy:  s.ReceivedBy,
		Notes:       s.Notes,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		Items:       items,
	}
}

func ToGoodsReceiptResponse(e *domain.GoodsReceiptEntity) *GoodsReceiptResponse {
	items := make([]GoodsReceiptItemResponse, len(e.Items))
	for i, row := range e.Items {
		items[i] = GoodsReceiptItemResponse{
			ID:          row.ID,
			OrderItemID: row.OrderItemID,
			Quantity:    row.Quantity.StringFixed(3),
		}
	}

	return &GoodsReceiptResponse{
		ID:          e.ID,
		GRNumber:    e.GRNumber,
		OrderID:     e.OrderID,
		WarehouseID: e.WarehouseID,
		Status:      e.Status,
		ReceivedAt:  e.ReceivedAt,
		ReceivedBy:  e.ReceivedBy,
		Notes:       e.Notes,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
		Items:       items,
	}
}
