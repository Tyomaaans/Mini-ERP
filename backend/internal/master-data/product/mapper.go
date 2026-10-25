package product

import "mini-erp/internal/master-data/domain"

// Product to Storage <-> Entity

func ToProductEntity(s *ProductStorage) *domain.ProductEntity {
	return &domain.ProductEntity{
		ID:           s.ID,
		SKU:          s.SKU,
		Name:         s.Name,
		CategoryID:   s.CategoryID,
		Unit:         s.Unit,
		SellingPrice: s.SellingPrice,
		ReorderLevel: s.ReorderLevel,
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

func ToProductStorage(s *domain.ProductEntity) *ProductStorage {
	return &ProductStorage{
		ID:           s.ID,
		SKU:          s.SKU,
		Name:         s.Name,
		CategoryID:   s.CategoryID,
		Unit:         s.Unit,
		SellingPrice: s.SellingPrice,
		ReorderLevel: s.ReorderLevel,
		IsActive:     s.IsActive,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// Entity to Response

func ToProductResponse(e *domain.ProductEntity) *ProductResponse {
	return &ProductResponse{
		ID:           e.ID,
		SKU:          e.SKU,
		Name:         e.Name,
		CategoryID:   e.CategoryID,
		Unit:         e.Unit,
		SellingPrice: e.SellingPrice.StringFixed(2),
		ReorderLevel: e.ReorderLevel.StringFixed(3),
		IsActive:     e.IsActive,
		CreatedAt:    e.CreatedAt,
		UpdatedAt:    e.UpdatedAt,
	}
}
