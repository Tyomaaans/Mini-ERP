package invoice

import (
	"context"
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/finance/domain"
	"mini-erp/pkg"
)

// Interface

type InvoiceService interface {
	CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*InvoiceResponse, error)
	UpdateInvoice(ctx context.Context, id string, req UpdateInvoiceRequest) (*InvoiceResponse, error)
	GetInvoice(ctx context.Context, id string) (*InvoiceResponse, error)
	GetInvoices(ctx context.Context, query pkg.ListQuery) ([]InvoiceResponse, int64, error)
	Issue(ctx context.Context, id, key string) (json.RawMessage, error)
}

// Implementation

type invoiceService struct {
	invoiceRepo         domain.InvoiceRepository
	validate            *validator.Validate
	txManager           pkg.TransactionManager
	auditLogger         pkg.AuditLogger
	documentSequence    pkg.DocumentSequence
	idempotencyExecutor pkg.IdempotencyExecutor
}

func NewInvoiceService(
	invoiceRepo         domain.InvoiceRepository,
	validate            *validator.Validate,
	txManager           pkg.TransactionManager,
	auditLogger         pkg.AuditLogger,
	documentSequence    pkg.DocumentSequence,
	idempotencyExecutor pkg.IdempotencyExecutor,
) InvoiceService {
	return &invoiceService{
		invoiceRepo:         invoiceRepo,
		validate:            validate,
		txManager:           txManager,
		auditLogger:         auditLogger,
		documentSequence:    documentSequence,
		idempotencyExecutor: idempotencyExecutor,
	}
}

func (s *invoiceService) CreateInvoice(ctx context.Context, req CreateInvoiceRequest) (*InvoiceResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("invoices.create"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	if (req.InvoiceType == "AR" && (req.SalesOrderID == nil || req.PurchaseOrderID != nil)) || (req.InvoiceType == "AP" && (req.PurchaseOrderID == nil || req.SalesOrderID != nil)) {
		return nil, pkg.ErrInvalidInput
	}

	issue, err := pkg.ParseDate(req.IssueDate)
	if err != nil {
		return nil, err
	}

	due, err := pkg.ParseDate(req.DueDate)
	if err != nil || due.Before(issue) {
		return nil, pkg.ErrInvalidInput
	}

	entity := &domain.InvoiceEntity{
		ID:              uuid.NewString(),
		InvoiceType:     req.InvoiceType,
		SalesOrderID:    req.SalesOrderID,
		PurchaseOrderID: req.PurchaseOrderID,
		IssueDate:       issue,
		DueDate:         due,
		Status:          "DRAFT",
		CreatedBy:       actor.ID,
	}

	err = s.txManager.Within(ctx, func(ctx context.Context) error {
		source, err := s.invoiceRepo.LoadSource(ctx, entity.InvoiceType, sourceID(entity), true)
		if err != nil {
			return err
		}

		if err := sourceReady(source, entity.InvoiceType); err != nil {
			return err
		}

		if entity.InvoiceType == "AR" {
			entity.CustomerID = &source.PartyID
		} else {
			entity.SupplierID = &source.PartyID
		}

		entity.TotalAmount = source.TotalAmount
		number, err := s.documentSequence.Next(ctx, "INV-"+entity.InvoiceType)
		if err != nil {
			return err
		}

		entity.InvoiceNumber = number
		if err := s.invoiceRepo.CreateInvoice(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "invoices.create", "INVOICE", entity.ID, nil, ToInvoiceResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToInvoiceResponse(entity), nil
}

func (s *invoiceService) UpdateInvoice(ctx context.Context, id string, req UpdateInvoiceRequest) (*InvoiceResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("invoices.update"); err != nil {
		return nil, err
	}

	if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	issue, err := pkg.ParseDate(req.IssueDate)
	if err != nil {
		return nil, err
	}

	due, err := pkg.ParseDate(req.DueDate)
	if err != nil || due.Before(issue) {
		return nil, pkg.ErrInvalidInput
	}

	var response *InvoiceResponse
	err = s.txManager.Within(ctx, func(ctx context.Context) error {
		old, err := s.invoiceRepo.GetInvoice(ctx, id, true)
		if err != nil {
			return err
		}

		if old.Status != "DRAFT" {
			return pkg.ErrInvalidState
		}

		if err := s.invoiceRepo.UpdateInvoice(ctx, id, map[string]any{"issue_date": issue, "due_date": due}); err != nil {
			return err
		}

		entity, err := s.invoiceRepo.GetInvoice(ctx, id, false)
		if err != nil {
			return err
		}

		response = ToInvoiceResponse(entity)
		return s.auditLogger.Record(ctx, actor, "invoices.update", "INVOICE", id, ToInvoiceResponse(old), response)
	})
	return response, err
}

func (s *invoiceService) GetInvoice(ctx context.Context, id string) (*InvoiceResponse, error) {
	if err := pkg.ActorFrom(ctx).Require("invoices.read"); err != nil {
		return nil, err
	}

	if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	entity, err := s.invoiceRepo.GetInvoice(ctx, id, false)
	if err != nil {
		return nil, err
	}

	return ToInvoiceResponse(entity), nil
}

func (s *invoiceService) GetInvoices(ctx context.Context, query pkg.ListQuery) ([]InvoiceResponse, int64, error) {
	if err := pkg.ActorFrom(ctx).Require("invoices.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.invoiceRepo.GetInvoices(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	response := make([]InvoiceResponse, len(entities))
	for i := range entities {
		response[i] = *ToInvoiceResponse(&entities[i])
	}

	return response, total, nil
}

func (s *invoiceService) Issue(ctx context.Context, id, key string) (json.RawMessage, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("invoices.issue"); err != nil {
		return nil, err
	}

	if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	preflight, err := s.invoiceRepo.GetInvoice(ctx, id, false)
	if err != nil {
		return nil, err
	}

	return s.idempotencyExecutor.Execute(ctx, actor, "invoices.issue", key, map[string]string{"id": id}, func(ctx context.Context) (any, error) {
		source, err := s.invoiceRepo.LoadSource(ctx, preflight.InvoiceType, sourceID(preflight), true)
		if err != nil {
			return nil, err
		}

		entity, err := s.invoiceRepo.GetInvoice(ctx, id, true)
		if err != nil {
			return nil, err
		}

		if entity.Status != "DRAFT" {
			return nil, pkg.ErrInvalidState
		}

		if err := sourceReady(source, entity.InvoiceType); err != nil {
			return nil, err
		}

		if !entity.TotalAmount.Equal(source.TotalAmount) {
			return nil, pkg.ErrConflict
		}

		status := "ISSUED"
		if entity.TotalAmount.IsZero() {
			status = "PAID"
		}

		if err := s.invoiceRepo.UpdateInvoice(ctx, id, map[string]any{"status": status}); err != nil {
			return nil, err
		}

		updated, err := s.invoiceRepo.GetInvoice(ctx, id, false)
		if err != nil {
			return nil, err
		}

		response := ToInvoiceResponse(updated)
		if err := s.auditLogger.Record(ctx, actor, "invoices.issue", "INVOICE", id, ToInvoiceResponse(entity), response); err != nil {
			return nil, err
		}

		return response, nil
	})
}

// Internal Helper

func sourceID(entity *domain.InvoiceEntity) string {
	if entity.InvoiceType == "AR" {
		return *entity.SalesOrderID
	}

	return *entity.PurchaseOrderID
}

func sourceReady(source *domain.InvoiceSource, invoiceType string) error {
	if (invoiceType == "AR" && source.Status != "SHIPPED") || (invoiceType == "AP" && source.Status != "CLOSED") {
		return pkg.NewError("ORDER_NOT_FULFILLED", "The complete order must be shipped or received before invoicing", 409)
	}

	return nil
}
