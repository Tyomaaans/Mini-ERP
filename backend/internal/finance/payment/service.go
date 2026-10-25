package payment

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/finance/domain"
	"mini-erp/internal/finance/invoice"
	"mini-erp/pkg"
)

// Interface

type PaymentService interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*PaymentResponse, error)
	GetPayment(ctx context.Context, id string) (*PaymentResponse, error)
	GetPayments(ctx context.Context, query pkg.ListQuery) ([]PaymentResponse, int64, error)
	Post(ctx context.Context, id, key string) (json.RawMessage, error)
}

// Implementation

type paymentService struct {
	paymentRepo         domain.PaymentRepository
	invoiceRepo         domain.InvoiceRepository
	validate            *validator.Validate
	txManager           pkg.TransactionManager
	auditLogger         pkg.AuditLogger
	documentSequence    pkg.DocumentSequence
	idempotencyExecutor pkg.IdempotencyExecutor
}

func NewPaymentService(
	paymentRepo         domain.PaymentRepository,
	invoiceRepo         domain.InvoiceRepository,
	validate            *validator.Validate,
	txManager           pkg.TransactionManager,
	auditLogger         pkg.AuditLogger,
	documentSequence    pkg.DocumentSequence,
	idempotencyExecutor pkg.IdempotencyExecutor,
) PaymentService {
	return &paymentService{
		paymentRepo:         paymentRepo,
		invoiceRepo:         invoiceRepo,
		validate:            validate,
		txManager:           txManager,
		auditLogger:         auditLogger,
		documentSequence:    documentSequence,
		idempotencyExecutor: idempotencyExecutor,
	}
}

func (s *paymentService) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*PaymentResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("payments.create"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	amount, err := pkg.ParseDecimal(req.Amount, 2, true)
	if err != nil {
		return nil, err
	}

	paidAt := time.Now().UTC()
	if req.PaidAt != nil {
		paidAt = req.PaidAt.UTC()
		if paidAt.After(time.Now().Add(5 * time.Minute)) {
			return nil, pkg.ErrInvalidInput
		}
	}

	entity := &domain.PaymentEntity{
		ID:              uuid.NewString(),
		InvoiceID:       req.InvoiceID,
		Amount:          amount,
		PaymentMethod:   req.PaymentMethod,
		ReferenceNumber: req.ReferenceNumber,
		Status:          "DRAFT",
		PaidAt:          paidAt,
		CreatedBy:       actor.ID,
	}

	err = s.txManager.Within(ctx, func(ctx context.Context) error {
		inv, err := s.invoiceRepo.GetInvoice(ctx, req.InvoiceID, true)
		if err != nil {
			return err
		}

		if inv.Status != "ISSUED" && inv.Status != "PARTIALLY_PAID" {
			return pkg.ErrInvalidState
		}

		if amount.GreaterThan(inv.TotalAmount.Sub(inv.PaidAmount)) {
			return pkg.NewError("OVERPAYMENT", "Payment exceeds the outstanding invoice amount", 409)
		}

		number, err := s.documentSequence.Next(ctx, "PAY")
		if err != nil {
			return err
		}

		entity.PaymentNumber = number
		if err := s.paymentRepo.CreatePayment(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "payments.create", "PAYMENT", entity.ID, nil, ToPaymentResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToPaymentResponse(entity), nil
}

func (s *paymentService) GetPayment(ctx context.Context, id string) (*PaymentResponse, error) {
	if err := pkg.ActorFrom(ctx).Require("payments.read"); err != nil {
		return nil, err
	}

	if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	entity, err := s.paymentRepo.GetPayment(ctx, id, false)
	if err != nil {
		return nil, err
	}

	return ToPaymentResponse(entity), nil
}

func (s *paymentService) GetPayments(ctx context.Context, query pkg.ListQuery) ([]PaymentResponse, int64, error) {
	if err := pkg.ActorFrom(ctx).Require("payments.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.paymentRepo.GetPayments(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	response := make([]PaymentResponse, len(entities))
	for i := range entities {
		response[i] = *ToPaymentResponse(&entities[i])
	}

	return response, total, nil
}

func (s *paymentService) Post(ctx context.Context, id, key string) (json.RawMessage, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("payments.post"); err != nil {
		return nil, err
	}

	if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	preflight, err := s.paymentRepo.GetPayment(ctx, id, false)
	if err != nil {
		return nil, err
	}

	return s.idempotencyExecutor.Execute(ctx, actor, "payments.post", key, map[string]string{"id": id}, func(ctx context.Context) (any, error) {
		inv, err := s.invoiceRepo.GetInvoice(ctx, preflight.InvoiceID, true)
		if err != nil {
			return nil, err
		}

		entity, err := s.paymentRepo.GetPayment(ctx, id, true)
		if err != nil {
			return nil, err
		}

		if entity.Status != "DRAFT" || (inv.Status != "ISSUED" && inv.Status != "PARTIALLY_PAID") {
			return nil, pkg.ErrInvalidState
		}

		posted, err := s.paymentRepo.PostedAmount(ctx, inv.ID)
		if err != nil {
			return nil, err
		}

		paid := posted.Add(entity.Amount)
		if paid.GreaterThan(inv.TotalAmount) {
			return nil, pkg.NewError("OVERPAYMENT", "Payment exceeds the outstanding invoice amount", 409)
		}

		if err := s.paymentRepo.Post(ctx, id); err != nil {
			return nil, err
		}

		status := "PARTIALLY_PAID"
		if paid.Equal(inv.TotalAmount) {
			status = "PAID"
		}

		if err := s.invoiceRepo.UpdateInvoice(ctx, inv.ID, map[string]any{"paid_amount": paid, "status": status}); err != nil {
			return nil, err
		}

		updatedInvoice, err := s.invoiceRepo.GetInvoice(ctx, inv.ID, false)
		if err != nil {
			return nil, err
		}

		if err := s.auditLogger.Record(ctx, actor, "invoices.payment", "INVOICE", inv.ID, invoice.ToInvoiceResponse(inv), invoice.ToInvoiceResponse(updatedInvoice)); err != nil {
			return nil, err
		}

		updated, err := s.paymentRepo.GetPayment(ctx, id, false)
		if err != nil {
			return nil, err
		}

		response := ToPaymentResponse(updated)
		if err := s.auditLogger.Record(ctx, actor, "payments.post", "PAYMENT", id, ToPaymentResponse(entity), response); err != nil {
			return nil, err
		}

		return response, nil
	})
}
