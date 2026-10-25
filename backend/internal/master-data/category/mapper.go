package category

import "mini-erp/internal/master-data/domain"

// Category to Storage <-> Entity

func ToCategoryEntity(s *CategoryStorage) *domain.CategoryEntity {
	return &domain.CategoryEntity{
		ID:        s.ID,
		Code:      s.Code,
		Name:      s.Name,
		ParentID:  s.ParentID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func ToCategoryStorage(s *domain.CategoryEntity) *CategoryStorage {
	return &CategoryStorage{
		ID:        s.ID,
		Code:      s.Code,
		Name:      s.Name,
		ParentID:  s.ParentID,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

// Entity to Response

func ToCategoryResponse(e *domain.CategoryEntity) *CategoryResponse {
	return &CategoryResponse{
		ID:        e.ID,
		Code:      e.Code,
		Name:      e.Name,
		ParentID:  e.ParentID,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
