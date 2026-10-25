package supplier

import "mini-erp/internal/master-data/domain"

// Supplier to Storage <-> Entity

func ToSupplierEntity(s *SupplierStorage) *domain.SupplierEntity {
	return &domain.SupplierEntity{
		ID:               s.ID,
		Code:             s.Code,
		Name:             s.Name,
		Email:            s.Email,
		Phone:            s.Phone,
		Address:          s.Address,
		PaymentTermsDays: s.PaymentTermsDays,
		IsActive:         s.IsActive,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

func ToSupplierStorage(s *domain.SupplierEntity) *SupplierStorage {
	return &SupplierStorage{
		ID:               s.ID,
		Code:             s.Code,
		Name:             s.Name,
		Email:            s.Email,
		Phone:            s.Phone,
		Address:          s.Address,
		PaymentTermsDays: s.PaymentTermsDays,
		IsActive:         s.IsActive,
		CreatedAt:        s.CreatedAt,
		UpdatedAt:        s.UpdatedAt,
	}
}

// Entity to Response

func ToSupplierResponse(e *domain.SupplierEntity) *SupplierResponse {
	return &SupplierResponse{
		ID:               e.ID,
		Code:             e.Code,
		Name:             e.Name,
		Email:            e.Email,
		Phone:            e.Phone,
		Address:          e.Address,
		PaymentTermsDays: e.PaymentTermsDays,
		IsActive:         e.IsActive,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}
