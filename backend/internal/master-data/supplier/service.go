package supplier

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

// Interface

type SupplierService interface {
	CreateSupplier(ctx context.Context, req CreateSupplierRequest) (*SupplierResponse, error)
	UpdateSupplier(ctx context.Context, id string, req UpdateSupplierRequest) (*SupplierResponse, error)
	GetSupplier(ctx context.Context, id string) (*SupplierResponse, error)
	GetSuppliers(ctx context.Context, query pkg.ListQuery) ([]SupplierResponse, int64, error)
}

// Implementation

type supplierService struct {
	supplierRepo domain.SupplierRepository
	validate     *validator.Validate
	txManager    pkg.TransactionManager
	auditLogger  pkg.AuditLogger
}

func NewSupplierService(
	supplierRepo domain.SupplierRepository,
	validate     *validator.Validate,
	txManager    pkg.TransactionManager,
	auditLogger  pkg.AuditLogger,
) SupplierService {
	return &supplierService{
		supplierRepo: supplierRepo,
		validate:     validate,
		txManager:    txManager,
		auditLogger:  auditLogger,
	}
}

func (s *supplierService) CreateSupplier(ctx context.Context, req CreateSupplierRequest) (*SupplierResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("suppliers.create"); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	entity := &domain.SupplierEntity{
		ID:       uuid.NewString(),
		IsActive: true,
	}

	entity.Code = req.Code
	entity.Name = req.Name
	entity.Email = req.Email
	entity.Phone = req.Phone
	entity.Address = req.Address
	entity.PaymentTermsDays = req.PaymentTermsDays
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.supplierRepo.CreateSupplier(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "suppliers.create", "SUPPLIER", entity.ID, nil, ToSupplierResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToSupplierResponse(entity), nil
}

func (s *supplierService) UpdateSupplier(ctx context.Context, id string, req UpdateSupplierRequest) (*SupplierResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("suppliers.update"); err != nil {
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

	if req.Email != nil {
		fields["email"] = *req.Email
	}

	if req.Phone != nil {
		fields["phone"] = *req.Phone
	}

	if req.Address != nil {
		fields["address"] = *req.Address
	}

	if req.PaymentTermsDays != nil {
		fields["payment_terms_days"] = *req.PaymentTermsDays
	}

	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if len(fields) == 0 {
		return nil, pkg.ErrInvalidInput
	}

	var response *SupplierResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		old, err := s.supplierRepo.GetSupplier(ctx, id)
		if err != nil {
			return err
		}

		if err := s.supplierRepo.UpdateSupplier(ctx, id, fields); err != nil {
			return err
		}

		entity, err := s.supplierRepo.GetSupplier(ctx, id)
		if err != nil {
			return err
		}

		response = ToSupplierResponse(entity)
		return s.auditLogger.Record(ctx, actor, "suppliers.update", "SUPPLIER", id, ToSupplierResponse(old), response)
	})
	return response, err
}

func (s *supplierService) GetSupplier(ctx context.Context, id string) (*SupplierResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("suppliers.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.supplierRepo.GetSupplier(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToSupplierResponse(entity), nil
}

func (s *supplierService) GetSuppliers(ctx context.Context, query pkg.ListQuery) ([]SupplierResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("suppliers.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.supplierRepo.GetSuppliers(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]SupplierResponse, len(entities))
	for i := range entities {
		response[i] = *ToSupplierResponse(&entities[i])
	}

	return response, total, nil
}
