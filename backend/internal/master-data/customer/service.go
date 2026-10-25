package customer

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

// Interface

type CustomerService interface {
	CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*CustomerResponse, error)
	UpdateCustomer(ctx context.Context, id string, req UpdateCustomerRequest) (*CustomerResponse, error)
	GetCustomer(ctx context.Context, id string) (*CustomerResponse, error)
	GetCustomers(ctx context.Context, query pkg.ListQuery) ([]CustomerResponse, int64, error)
}

// Implementation

type customerService struct {
	customerRepo domain.CustomerRepository
	validate     *validator.Validate
	txManager    pkg.TransactionManager
	auditLogger  pkg.AuditLogger
}

func NewCustomerService(
	customerRepo domain.CustomerRepository,
	validate     *validator.Validate,
	txManager    pkg.TransactionManager,
	auditLogger  pkg.AuditLogger,
) CustomerService {
	return &customerService{
		customerRepo: customerRepo,
		validate:     validate,
		txManager:    txManager,
		auditLogger:  auditLogger,
	}
}

func (s *customerService) CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*CustomerResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("customers.create"); err != nil {
		return nil, err
	}

	req.Name = strings.TrimSpace(req.Name)
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	entity := &domain.CustomerEntity{
		ID:       uuid.NewString(),
		IsActive: true,
	}

	entity.Code = req.Code
	entity.Name = req.Name
	entity.Email = req.Email
	entity.Phone = req.Phone
	entity.BillingAddress = req.BillingAddress
	entity.ShippingAddress = req.ShippingAddress
	valueCreditLimit, err := pkg.ParseDecimal(req.CreditLimit, 2, false)
	if err != nil {
		return nil, err
	}

	entity.CreditLimit = valueCreditLimit
	err = s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.customerRepo.CreateCustomer(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "customers.create", "CUSTOMER", entity.ID, nil, ToCustomerResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToCustomerResponse(entity), nil
}

func (s *customerService) UpdateCustomer(ctx context.Context, id string, req UpdateCustomerRequest) (*CustomerResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("customers.update"); err != nil {
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

	if req.BillingAddress != nil {
		fields["billing_address"] = *req.BillingAddress
	}

	if req.ShippingAddress != nil {
		fields["shipping_address"] = *req.ShippingAddress
	}

	if req.CreditLimit != nil {
		value, err := pkg.ParseDecimal(*req.CreditLimit, 2, false)
		if err != nil {
			return nil, err
		}

		fields["credit_limit"] = value
	}

	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if len(fields) == 0 {
		return nil, pkg.ErrInvalidInput
	}

	var response *CustomerResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		old, err := s.customerRepo.GetCustomer(ctx, id)
		if err != nil {
			return err
		}

		if err := s.customerRepo.UpdateCustomer(ctx, id, fields); err != nil {
			return err
		}

		entity, err := s.customerRepo.GetCustomer(ctx, id)
		if err != nil {
			return err
		}

		response = ToCustomerResponse(entity)
		return s.auditLogger.Record(ctx, actor, "customers.update", "CUSTOMER", id, ToCustomerResponse(old), response)
	})
	return response, err
}

func (s *customerService) GetCustomer(ctx context.Context, id string) (*CustomerResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("customers.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.customerRepo.GetCustomer(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToCustomerResponse(entity), nil
}

func (s *customerService) GetCustomers(ctx context.Context, query pkg.ListQuery) ([]CustomerResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("customers.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.customerRepo.GetCustomers(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]CustomerResponse, len(entities))
	for i := range entities {
		response[i] = *ToCustomerResponse(&entities[i])
	}

	return response, total, nil
}
