package product

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

// Interface

type ProductService interface {
	CreateProduct(ctx context.Context, req CreateProductRequest) (*ProductResponse, error)
	UpdateProduct(ctx context.Context, id string, req UpdateProductRequest) (*ProductResponse, error)
	GetProduct(ctx context.Context, id string) (*ProductResponse, error)
	GetProducts(ctx context.Context, query pkg.ListQuery) ([]ProductResponse, int64, error)
}

// Implementation

type productService struct {
	productRepo domain.ProductRepository
	validate    *validator.Validate
	txManager   pkg.TransactionManager
	auditLogger pkg.AuditLogger
}

func NewProductService(
	productRepo domain.ProductRepository,
	validate    *validator.Validate,
	txManager   pkg.TransactionManager,
	auditLogger pkg.AuditLogger,
) ProductService {
	return &productService{
		productRepo: productRepo,
		validate:    validate,
		txManager:   txManager,
		auditLogger: auditLogger,
	}
}

func (s *productService) CreateProduct(ctx context.Context, req CreateProductRequest) (*ProductResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("products.create"); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	entity := &domain.ProductEntity{
		ID:       uuid.NewString(),
		IsActive: true,
	}

	entity.SKU = req.SKU
	entity.Name = req.Name
	entity.CategoryID = req.CategoryID
	entity.Unit = req.Unit
	valueSellingPrice, err := pkg.ParseDecimal(req.SellingPrice, 2, false)
	if err != nil {
		return nil, err
	}

	entity.SellingPrice = valueSellingPrice
	valueReorderLevel, err := pkg.ParseDecimal(req.ReorderLevel, 3, false)
	if err != nil {
		return nil, err
	}

	entity.ReorderLevel = valueReorderLevel
	err = s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.productRepo.CreateProduct(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "products.create", "PRODUCT", entity.ID, nil, ToProductResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToProductResponse(entity), nil
}

func (s *productService) UpdateProduct(ctx context.Context, id string, req UpdateProductRequest) (*ProductResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("products.update"); err != nil {
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
	if req.SKU != nil {
		fields["sku"] = *req.SKU
	}

	if req.Name != nil {
		fields["name"] = *req.Name
	}

	if req.CategoryID != nil {
		fields["category_id"] = *req.CategoryID
	}

	if req.Unit != nil {
		fields["unit"] = *req.Unit
	}

	if req.SellingPrice != nil {
		value, err := pkg.ParseDecimal(*req.SellingPrice, 2, false)
		if err != nil {
			return nil, err
		}

		fields["selling_price"] = value
	}

	if req.ReorderLevel != nil {
		value, err := pkg.ParseDecimal(*req.ReorderLevel, 3, false)
		if err != nil {
			return nil, err
		}

		fields["reorder_level"] = value
	}

	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if len(fields) == 0 {
		return nil, pkg.ErrInvalidInput
	}

	var response *ProductResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		old, err := s.productRepo.GetProduct(ctx, id)
		if err != nil {
			return err
		}

		if err := s.productRepo.UpdateProduct(ctx, id, fields); err != nil {
			return err
		}

		entity, err := s.productRepo.GetProduct(ctx, id)
		if err != nil {
			return err
		}

		response = ToProductResponse(entity)
		return s.auditLogger.Record(ctx, actor, "products.update", "PRODUCT", id, ToProductResponse(old), response)
	})
	return response, err
}

func (s *productService) GetProduct(ctx context.Context, id string) (*ProductResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("products.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.productRepo.GetProduct(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToProductResponse(entity), nil
}

func (s *productService) GetProducts(ctx context.Context, query pkg.ListQuery) ([]ProductResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("products.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.productRepo.GetProducts(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]ProductResponse, len(entities))
	for i := range entities {
		response[i] = *ToProductResponse(&entities[i])
	}

	return response, total, nil
}
