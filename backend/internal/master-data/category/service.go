package category

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

// Interface

type CategoryService interface {
	CreateCategory(ctx context.Context, req CreateCategoryRequest) (*CategoryResponse, error)
	UpdateCategory(ctx context.Context, id string, req UpdateCategoryRequest) (*CategoryResponse, error)
	GetCategory(ctx context.Context, id string) (*CategoryResponse, error)
	GetCategories(ctx context.Context, query pkg.ListQuery) ([]CategoryResponse, int64, error)
	DeleteCategory(ctx context.Context, id string) error
}

// Implementation

type categoryService struct {
	categoryRepo domain.CategoryRepository
	validate     *validator.Validate
	txManager    pkg.TransactionManager
	auditLogger  pkg.AuditLogger
}

func NewCategoryService(
	categoryRepo domain.CategoryRepository,
	validate     *validator.Validate,
	txManager    pkg.TransactionManager,
	auditLogger  pkg.AuditLogger,
) CategoryService {
	return &categoryService{
		categoryRepo: categoryRepo,
		validate:     validate,
		txManager:    txManager,
		auditLogger:  auditLogger,
	}
}

func (s *categoryService) CreateCategory(ctx context.Context, req CreateCategoryRequest) (*CategoryResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("categories.create"); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	entity := &domain.CategoryEntity{ID: uuid.NewString()}
	entity.Code = req.Code
	entity.Name = req.Name
	entity.ParentID = req.ParentID
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.categoryRepo.ValidateParent(ctx, entity.ID, entity.ParentID); err != nil {
			return err
		}

		if err := s.categoryRepo.CreateCategory(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "categories.create", "CATEGORY", entity.ID, nil, ToCategoryResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToCategoryResponse(entity), nil
}

func (s *categoryService) UpdateCategory(ctx context.Context, id string, req UpdateCategoryRequest) (*CategoryResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("categories.update"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			return nil, pkg.ErrInvalidInput
		}

		req.Name = &value
	}

	fields := make(map[string]any)
	if req.Code != nil {
		fields["code"] = *req.Code
	}

	if req.Name != nil {
		fields["name"] = *req.Name
	}

	if req.ParentID.Set {
		fields["parent_id"] = req.ParentID.Value
	}

	if len(fields) == 0 {
		return nil, pkg.ErrInvalidInput
	}

	var response *CategoryResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		if req.ParentID.Set {
			if err := s.categoryRepo.ValidateParent(ctx, id, req.ParentID.Value); err != nil {
				return err
			}
		}

		old, err := s.categoryRepo.GetCategory(ctx, id)
		if err != nil {
			return err
		}

		if err := s.categoryRepo.UpdateCategory(ctx, id, fields); err != nil {
			return err
		}

		entity, err := s.categoryRepo.GetCategory(ctx, id)
		if err != nil {
			return err
		}

		response = ToCategoryResponse(entity)
		return s.auditLogger.Record(ctx, actor, "categories.update", "CATEGORY", id, ToCategoryResponse(old), response)
	})
	return response, err
}

func (s *categoryService) GetCategory(ctx context.Context, id string) (*CategoryResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("categories.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.categoryRepo.GetCategory(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToCategoryResponse(entity), nil
}

func (s *categoryService) GetCategories(ctx context.Context, query pkg.ListQuery) ([]CategoryResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("categories.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.categoryRepo.GetCategories(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]CategoryResponse, len(entities))
	for i := range entities {
		response[i] = *ToCategoryResponse(&entities[i])
	}

	return response, total, nil
}

func (s *categoryService) DeleteCategory(ctx context.Context, id string) error {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("categories.delete"); err != nil {
		return err
	}

	if err := pkg.ValidateID(id); err != nil {
		return err
	}

	return s.txManager.Within(ctx, func(ctx context.Context) error {
		old, err := s.categoryRepo.GetCategory(ctx, id)
		if err != nil {
			return err
		}

		if err := s.categoryRepo.DeleteCategory(ctx, id); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "categories.delete", "CATEGORY", id, ToCategoryResponse(old), nil)
	})
}
