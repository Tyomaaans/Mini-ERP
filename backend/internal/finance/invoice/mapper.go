package invoice

import (
	"time"

	"mini-erp/internal/finance/domain"
)

func ToInvoiceEntity(s *InvoiceStorage) *domain.InvoiceEntity {
	entity := domain.InvoiceEntity(*s)
	return &entity
}

func ToInvoiceStorage(e *domain.InvoiceEntity) *InvoiceStorage {
	storage := InvoiceStorage(*e)
	return &storage
}

func ToInvoiceResponse(e *domain.InvoiceEntity) *InvoiceResponse {
	return &InvoiceResponse{
		ID:                e.ID,
		InvoiceNumber:     e.InvoiceNumber,
		InvoiceType:       e.InvoiceType,
		SalesOrderID:      e.SalesOrderID,
		PurchaseOrderID:   e.PurchaseOrderID,
		CustomerID:        e.CustomerID,
		SupplierID:        e.SupplierID,
		Status:            e.Status,
		IssueDate:         e.IssueDate.Format(time.DateOnly),
		DueDate:           e.DueDate.Format(time.DateOnly),
		TotalAmount:       e.TotalAmount.StringFixed(2),
		PaidAmount:        e.PaidAmount.StringFixed(2),
		OutstandingAmount: e.TotalAmount.Sub(e.PaidAmount).StringFixed(2),
		CreatedBy:         e.CreatedBy,
		CreatedAt:         e.CreatedAt,
		UpdatedAt:         e.UpdatedAt,
	}
}
