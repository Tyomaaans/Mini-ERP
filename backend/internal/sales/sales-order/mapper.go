package salesorder

import (
	"time"

	"mini-erp/internal/sales/domain"
)

// SalesOrder to Storage <-> Entity

func ToSalesOrderEntity(s *SalesOrderStorage) *domain.SalesOrderEntity {
	items := make([]domain.OrderItemEntity, len(s.Items))
	for i, row := range s.Items {
		items[i] = domain.OrderItemEntity(row)
	}

	return &domain.SalesOrderEntity{
		ID:             s.ID,
		SONumber:       s.SONumber,
		CustomerID:     s.CustomerID,
		WarehouseID:    s.WarehouseID,
		Status:         s.Status,
		OrderDate:      s.OrderDate,
		Subtotal:       s.Subtotal,
		DiscountAmount: s.DiscountAmount,
		TotalAmount:    s.TotalAmount,
		Notes:          s.Notes,
		CreatedBy:      s.CreatedBy,
		ConfirmedAt:    s.ConfirmedAt,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		Items:          items,
	}
}

func ToSalesOrderStorage(s *domain.SalesOrderEntity) *SalesOrderStorage {
	items := make([]OrderItemStorage, len(s.Items))
	for i, row := range s.Items {
		items[i] = OrderItemStorage(row)
	}

	return &SalesOrderStorage{
		ID:             s.ID,
		SONumber:       s.SONumber,
		CustomerID:     s.CustomerID,
		WarehouseID:    s.WarehouseID,
		Status:         s.Status,
		OrderDate:      s.OrderDate,
		Subtotal:       s.Subtotal,
		DiscountAmount: s.DiscountAmount,
		TotalAmount:    s.TotalAmount,
		Notes:          s.Notes,
		CreatedBy:      s.CreatedBy,
		ConfirmedAt:    s.ConfirmedAt,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
		Items:          items,
	}
}

// Entity to Response

func ToSalesOrderResponse(e *domain.SalesOrderEntity) *SalesOrderResponse {

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

	return &SalesOrderResponse{
		ID:             e.ID,
		SONumber:       e.SONumber,
		CustomerID:     e.CustomerID,
		WarehouseID:    e.WarehouseID,
		Status:         e.Status,
		OrderDate:      e.OrderDate.Format(time.DateOnly),
		ConfirmedAt:    e.ConfirmedAt,
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
