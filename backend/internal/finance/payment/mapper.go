package payment

import "mini-erp/internal/finance/domain"

func ToPaymentEntity(s *PaymentStorage) *domain.PaymentEntity {
	entity := domain.PaymentEntity(*s)
	return &entity
}

func ToPaymentStorage(e *domain.PaymentEntity) *PaymentStorage {
	storage := PaymentStorage(*e)
	return &storage
}

func ToPaymentResponse(e *domain.PaymentEntity) *PaymentResponse {
	return &PaymentResponse{
		ID:              e.ID,
		PaymentNumber:   e.PaymentNumber,
		InvoiceID:       e.InvoiceID,
		Amount:          e.Amount.StringFixed(2),
		PaymentMethod:   e.PaymentMethod,
		ReferenceNumber: e.ReferenceNumber,
		Status:          e.Status,
		PaidAt:          e.PaidAt,
		CreatedBy:       e.CreatedBy,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}
