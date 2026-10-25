package warehouse

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

// Interface

type WarehouseService interface {
	CreateWarehouse(ctx context.Context, req CreateWarehouseRequest) (*WarehouseResponse, error)
	UpdateWarehouse(ctx context.Context, id string, req UpdateWarehouseRequest) (*WarehouseResponse, error)
	GetWarehouse(ctx context.Context, id string) (*WarehouseResponse, error)
	GetWarehouses(ctx context.Context, query pkg.ListQuery) ([]WarehouseResponse, int64, error)
}

// Implementation

type warehouseService struct {
	warehouseRepo domain.WarehouseRepository
	validate      *validator.Validate
	txManager     pkg.TransactionManager
	auditLogger   pkg.AuditLogger
}

func NewWarehouseService(
	warehouseRepo domain.WarehouseRepository,
	validate      *validator.Validate,
	txManager     pkg.TransactionManager,
	auditLogger   pkg.AuditLogger,
) WarehouseService {
	return &warehouseService{
		warehouseRepo: warehouseRepo,
		validate:      validate,
		txManager:     txManager,
		auditLogger:   auditLogger,
	}
}

func (s *warehouseService) CreateWarehouse(ctx context.Context, req CreateWarehouseRequest) (*WarehouseResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("warehouses.create"); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	entity := &domain.WarehouseEntity{
		ID:       uuid.NewString(),
		IsActive: true,
	}

	entity.Code = req.Code
	entity.Name = req.Name
	entity.Address = req.Address
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.warehouseRepo.CreateWarehouse(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "warehouses.create", "WAREHOUSE", entity.ID, nil, ToWarehouseResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToWarehouseResponse(entity), nil
}

func (s *warehouseService) UpdateWarehouse(ctx context.Context, id string, req UpdateWarehouseRequest) (*WarehouseResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("warehouses.update"); err != nil {
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

	if req.Address != nil {
		fields["address"] = *req.Address
	}

	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if len(fields) == 0 {
		return nil, pkg.ErrInvalidInput
	}

	var response *WarehouseResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		old, err := s.warehouseRepo.GetWarehouse(ctx, id)
		if err != nil {
			return err
		}

		if err := s.warehouseRepo.UpdateWarehouse(ctx, id, fields); err != nil {
			return err
		}

		entity, err := s.warehouseRepo.GetWarehouse(ctx, id)
		if err != nil {
			return err
		}

		response = ToWarehouseResponse(entity)
		return s.auditLogger.Record(ctx, actor, "warehouses.update", "WAREHOUSE", id, ToWarehouseResponse(old), response)
	})
	return response, err
}

func (s *warehouseService) GetWarehouse(ctx context.Context, id string) (*WarehouseResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("warehouses.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	if len(actor.WarehouseScope()) > 0 {
		if err := actor.RequireWarehouse(id); err != nil {
			return nil, err
		}
	}

	entity, err := s.warehouseRepo.GetWarehouse(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToWarehouseResponse(entity), nil
}

func (s *warehouseService) GetWarehouses(ctx context.Context, query pkg.ListQuery) ([]WarehouseResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("warehouses.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.warehouseRepo.GetWarehouses(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]WarehouseResponse, len(entities))
	for i := range entities {
		response[i] = *ToWarehouseResponse(&entities[i])
	}

	return response, total, nil
}
