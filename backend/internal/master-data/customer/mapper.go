package customer

import "mini-erp/internal/master-data/domain"

// Customer to Storage <-> Entity

func ToCustomerEntity(s *CustomerStorage) *domain.CustomerEntity {
	return &domain.CustomerEntity{
		ID:              s.ID,
		Code:            s.Code,
		Name:            s.Name,
		Email:           s.Email,
		Phone:           s.Phone,
		BillingAddress:  s.BillingAddress,
		ShippingAddress: s.ShippingAddress,
		CreditLimit:     s.CreditLimit,
		IsActive:        s.IsActive,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

func ToCustomerStorage(s *domain.CustomerEntity) *CustomerStorage {
	return &CustomerStorage{
		ID:              s.ID,
		Code:            s.Code,
		Name:            s.Name,
		Email:           s.Email,
		Phone:           s.Phone,
		BillingAddress:  s.BillingAddress,
		ShippingAddress: s.ShippingAddress,
		CreditLimit:     s.CreditLimit,
		IsActive:        s.IsActive,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
	}
}

// Entity to Response

func ToCustomerResponse(e *domain.CustomerEntity) *CustomerResponse {
	return &CustomerResponse{
		ID:              e.ID,
		Code:            e.Code,
		Name:            e.Name,
		Email:           e.Email,
		Phone:           e.Phone,
		BillingAddress:  e.BillingAddress,
		ShippingAddress: e.ShippingAddress,
		CreditLimit:     e.CreditLimit.StringFixed(2),
		IsActive:        e.IsActive,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}
