package warehouse

import "mini-erp/internal/master-data/domain"

// Warehouse to Storage <-> Entity

func ToWarehouseEntity(s *WarehouseStorage) *domain.WarehouseEntity {
	return &domain.WarehouseEntity{
		ID:        s.ID,
		Code:      s.Code,
		Name:      s.Name,
		Address:   s.Address,
		IsActive:  s.IsActive,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func ToWarehouseStorage(s *domain.WarehouseEntity) *WarehouseStorage {
	return &WarehouseStorage{
		ID:        s.ID,
		Code:      s.Code,
		Name:      s.Name,
		Address:   s.Address,
		IsActive:  s.IsActive,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// Entity to Response

func ToWarehouseResponse(e *domain.WarehouseEntity) *WarehouseResponse {
	return &WarehouseResponse{
		ID:        e.ID,
		Code:      e.Code,
		Name:      e.Name,
		Address:   e.Address,
		IsActive:  e.IsActive,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
