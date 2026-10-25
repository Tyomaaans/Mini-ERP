package purchaseorder

import (
	"time"

	"mini-erp/internal/purchasing/domain"
)

// PurchaseOrder to Storage <-> Entity

func ToPurchaseOrderEntity(s *PurchaseOrderStorage) *domain.PurchaseOrderEntity {
	items := make([]domain.OrderItemEntity, len(s.Items))
	for i, row := range s.Items {
		items[i] = domain.OrderItemEntity(row)
	}

	return &domain.PurchaseOrderEntity{
		ID:             s.ID,
		PONumber:       s.PONumber,
		SupplierID:     s.SupplierID,
		WarehouseID:    s.WarehouseID,
		Status:         s.Status,
		OrderDate:      s.OrderDate,
		Subtotal:       s.Subtotal,
		DiscountAmount: s.DiscountAmount,
		TotalAmount:    s.TotalAmount,
		Notes:          s.Notes,
		CreatedBy:      s.CreatedBy,
		ExpectedDate:   s.ExpectedDate,
		SubmittedAt:    s.SubmittedAt,
		ApprovedBy:     s.ApprovedBy,
		ApprovedAt:     s.ApprovedAt,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		Items:          items,
	}
}

func ToPurchaseOrderStorage(s *domain.PurchaseOrderEntity) *PurchaseOrderStorage {
	items := make([]OrderItemStorage, len(s.Items))
	for i, row := range s.Items {
		items[i] = OrderItemStorage(row)
	}

	return &PurchaseOrderStorage{
		ID:             s.ID,
		PONumber:       s.PONumber,
		SupplierID:     s.SupplierID,
		WarehouseID:    s.WarehouseID,
		Status:         s.Status,
		OrderDate:      s.OrderDate,
		Subtotal:       s.Subtotal,
		DiscountAmount: s.DiscountAmount,
		TotalAmount:    s.TotalAmount,
		Notes:          s.Notes,
		CreatedBy:      s.CreatedBy,
		ExpectedDate:   s.ExpectedDate,
		SubmittedAt:    s.SubmittedAt,
		ApprovedBy:     s.ApprovedBy,
		ApprovedAt:     s.ApprovedAt,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		Items:          items,
	}
}

// Entity to Response

func ToPurchaseOrderResponse(e *domain.PurchaseOrderEntity) *PurchaseOrderResponse {
	var expectedDate *string
	if e.ExpectedDate != nil {
		value := e.ExpectedDate.Format(time.DateOnly)
		expectedDate = &value
	}

	items := make([]OrderItemResponse, len(e.Items))
	for i, row := range e.Items {
		items[i] = OrderItemResponse{
			ID:             row.ID,
			ProductID:      row.ProductID,
			Quantity:       row.Quantity.StringFixed(3),
			UnitPrice:      row.UnitPrice.StringFixed(2),
			DiscountAmount: row.DiscountAmount.StringFixed(2),
			LineTotal:      row.LineTotal.StringFixed(2),
		}
	}

	return &PurchaseOrderResponse{
		ID:             e.ID,
		PONumber:       e.PONumber,
		SupplierID:     e.SupplierID,
		WarehouseID:    e.WarehouseID,
		Status:         e.Status,
		OrderDate:      e.OrderDate.Format(time.DateOnly),
		ExpectedDate:   expectedDate,
		SubmittedAt:    e.SubmittedAt,
		ApprovedBy:     e.ApprovedBy,
		ApprovedAt:     e.ApprovedAt,
		Subtotal:       e.Subtotal.StringFixed(2),
		DiscountAmount: e.DiscountAmount.StringFixed(2),
		TotalAmount:    e.TotalAmount.StringFixed(2),
		Notes:          e.Notes,
		CreatedBy:      e.CreatedBy,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
		Items:          items,
	}
}
