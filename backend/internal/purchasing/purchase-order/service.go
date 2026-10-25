package purchaseorder

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	"mini-erp/internal/purchasing/domain"
	"mini-erp/pkg"
)

// Interface

type PurchaseOrderService interface {
	CreatePurchaseOrder(ctx context.Context, req CreatePurchaseOrderRequest) (*PurchaseOrderResponse, error)
	UpdatePurchaseOrder(ctx context.Context, id string, req UpdatePurchaseOrderRequest) (*PurchaseOrderResponse, error)
	GetPurchaseOrder(ctx context.Context, id string) (*PurchaseOrderResponse, error)
	GetPurchaseOrders(ctx context.Context, query pkg.ListQuery) ([]PurchaseOrderResponse, int64, error)
	Submit(ctx context.Context, id, key, note string) (json.RawMessage, error)
	Approve(ctx context.Context, id, key, note string) (json.RawMessage, error)
	Reject(ctx context.Context, id, key, note string) (json.RawMessage, error)
	Revise(ctx context.Context, id, key, note string) (json.RawMessage, error)
	Cancel(ctx context.Context, id, key, note string) (json.RawMessage, error)
}

// Implementation

type purchaseOrderService struct {
	purchaseOrderRepo   domain.PurchaseOrderRepository
	validate            *validator.Validate
	txManager           pkg.TransactionManager
	auditLogger         pkg.AuditLogger
	documentSequence    pkg.DocumentSequence
	idempotencyExecutor pkg.IdempotencyExecutor
}

func NewPurchaseOrderService(
	purchaseOrderRepo   domain.PurchaseOrderRepository,
	validate            *validator.Validate,
	txManager           pkg.TransactionManager,
	auditLogger         pkg.AuditLogger,
	documentSequence    pkg.DocumentSequence,
	idempotencyExecutor pkg.IdempotencyExecutor,
) PurchaseOrderService {
	return &purchaseOrderService{
		purchaseOrderRepo:   purchaseOrderRepo,
		validate:            validate,
		txManager:           txManager,
		auditLogger:         auditLogger,
		documentSequence:    documentSequence,
		idempotencyExecutor: idempotencyExecutor,
	}
}

func (s *purchaseOrderService) CreatePurchaseOrder(ctx context.Context, req CreatePurchaseOrderRequest) (*PurchaseOrderResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("purchase_orders.create"); err != nil {
		return nil, err
	}

	entity, err := s.build(req)
	if err != nil {
		return nil, err
	}

	entity.CreatedBy = actor.ID
	err = s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.purchaseOrderRepo.ValidateReferences(ctx, entity); err != nil {
			return err
		}

		number, err := s.documentSequence.Next(ctx, "PO")
		if err != nil {
			return err
		}

		entity.PONumber = number
		if err := s.purchaseOrderRepo.CreatePurchaseOrder(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "purchase_orders.create", "PURCHASE_ORDER", entity.ID, nil, ToPurchaseOrderResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToPurchaseOrderResponse(entity), nil
}

func (s *purchaseOrderService) UpdatePurchaseOrder(ctx context.Context, id string, req UpdatePurchaseOrderRequest) (*PurchaseOrderResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("purchase_orders.update"); err != nil {
		return nil, err
	}

	if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	if !(req.SupplierID != nil || req.WarehouseID != nil || req.OrderDate != nil || req.DiscountAmount != nil || req.Notes.Set || req.Items != nil || req.ExpectedDate.Set) {
		return nil, pkg.ErrInvalidInput
	}

	var response *PurchaseOrderResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		old, err := s.purchaseOrderRepo.GetPurchaseOrder(ctx, id, true)
		if err != nil {
			return err
		}

		if old.Status != "DRAFT" {
			return pkg.ErrInvalidState
		}

		merged := CreatePurchaseOrderRequest{
			SupplierID:     old.SupplierID,
			WarehouseID:    old.WarehouseID,
			OrderDate:      old.OrderDate.Format(time.DateOnly),
			DiscountAmount: old.DiscountAmount.StringFixed(2),
			Notes:          old.Notes,
			Items:          make([]pkg.LineInput, len(old.Items)),
		}

		for i, item := range old.Items {
			merged.Items[i] = pkg.LineInput{
				ProductID:      item.ProductID,
				Quantity:       item.Quantity.StringFixed(3),
				UnitPrice:      item.UnitPrice.StringFixed(2),
				DiscountAmount: item.DiscountAmount.StringFixed(2),
			}
		}

		if req.SupplierID != nil {
			merged.SupplierID = *req.SupplierID
		}

		if req.WarehouseID != nil {
			merged.WarehouseID = *req.WarehouseID
		}

		if req.OrderDate != nil {
			merged.OrderDate = *req.OrderDate
		}

		if req.DiscountAmount != nil {
			merged.DiscountAmount = *req.DiscountAmount
		}

		if req.Notes.Set {
			merged.Notes = req.Notes.Value
		}

		if req.Items != nil {
			merged.Items = *req.Items
		}

		if old.ExpectedDate != nil {
			value := old.ExpectedDate.Format(time.DateOnly)
			merged.ExpectedDate = &value
		}

		if req.ExpectedDate.Set {
			merged.ExpectedDate = req.ExpectedDate.Value
		}

		entity, err := s.build(merged)
		if err != nil {
			return err
		}

		entity.ID = id
		for i := range entity.Items {
			entity.Items[i].OrderID = id
			for _, previous := range old.Items {
				if previous.ProductID == entity.Items[i].ProductID {
					entity.Items[i].ID = previous.ID
					entity.Items[i].CreatedAt = previous.CreatedAt
				}
			}
		}

		if err := s.purchaseOrderRepo.ValidateReferences(ctx, entity); err != nil {
			return err
		}

		if err := s.purchaseOrderRepo.UpdatePurchaseOrder(ctx, entity); err != nil {
			return err
		}

		updated, err := s.purchaseOrderRepo.GetPurchaseOrder(ctx, id, false)
		if err != nil {
			return err
		}

		response = ToPurchaseOrderResponse(updated)
		return s.auditLogger.Record(ctx, actor, "purchase_orders.update", "PURCHASE_ORDER", id, ToPurchaseOrderResponse(old), response)
	})
	return response, err
}

func (s *purchaseOrderService) GetPurchaseOrder(ctx context.Context, id string) (*PurchaseOrderResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("purchase_orders.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.purchaseOrderRepo.GetPurchaseOrder(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if len(actor.WarehouseScope()) > 0 {
		if err := actor.RequireWarehouse(entity.WarehouseID); err != nil {
			return nil, err
		}
	}

	return ToPurchaseOrderResponse(entity), nil
}

func (s *purchaseOrderService) GetPurchaseOrders(ctx context.Context, query pkg.ListQuery) ([]PurchaseOrderResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("purchase_orders.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.purchaseOrderRepo.GetPurchaseOrders(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]PurchaseOrderResponse, len(entities))
	for i := range entities {
		response[i] = *ToPurchaseOrderResponse(&entities[i])
	}

	return response, total, nil
}

func (s *purchaseOrderService) Submit(ctx context.Context, id, key, note string) (json.RawMessage, error) {
	return s.transition(ctx, id, key, "submit", note)
}

func (s *purchaseOrderService) Approve(ctx context.Context, id, key, note string) (json.RawMessage, error) {
	return s.transition(ctx, id, key, "approve", note)
}

func (s *purchaseOrderService) Reject(ctx context.Context, id, key, note string) (json.RawMessage, error) {
	return s.transition(ctx, id, key, "reject", note)
}

func (s *purchaseOrderService) Revise(ctx context.Context, id, key, note string) (json.RawMessage, error) {
	return s.transition(ctx, id, key, "revise", note)
}

func (s *purchaseOrderService) Cancel(ctx context.Context, id, key, note string) (json.RawMessage, error) {
	return s.transition(ctx, id, key, "cancel", note)
}

// Internal Helper

func (s *purchaseOrderService) build(req CreatePurchaseOrderRequest) (*domain.PurchaseOrderEntity, error) {
	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	date, err := pkg.ParseDate(req.OrderDate)
	if err != nil {
		return nil, err
	}

	lines, subtotal, discount, total, err := pkg.CalculateOrder(req.Items, req.DiscountAmount)
	if err != nil {
		return nil, err
	}

	entity := &domain.PurchaseOrderEntity{
		ID:             uuid.NewString(),
		SupplierID:     req.SupplierID,
		WarehouseID:    req.WarehouseID,
		Status:         "DRAFT",
		OrderDate:      date,
		Subtotal:       subtotal,
		DiscountAmount: discount,
		TotalAmount:    total,
		Notes:          req.Notes,
		Items:          make([]domain.OrderItemEntity, len(lines)),
	}

	if req.ExpectedDate != nil {
		value, err := pkg.ParseDate(*req.ExpectedDate)
		if err != nil || value.Before(date) {
			return nil, pkg.ErrInvalidInput
		}

		entity.ExpectedDate = &value
	}

	for i, line := range lines {
		entity.Items[i] = domain.OrderItemEntity{
			ID:             uuid.NewString(),
			OrderID:        entity.ID,
			ProductID:      line.ProductID,
			Quantity:       line.Quantity,
			UnitPrice:      line.UnitPrice,
			DiscountAmount: line.DiscountAmount,
			LineTotal:      line.LineTotal,
		}
	}

	return entity, nil
}

func (s *purchaseOrderService) transition(ctx context.Context, id, key, action, note string) (json.RawMessage, error) {
	actor := pkg.ActorFrom(ctx)
	permission := map[string]string{"submit": "purchase_orders.submit", "approve": "purchase_orders.approve", "reject": "purchase_orders.approve", "revise": "purchase_orders.update", "cancel": "purchase_orders.cancel"}[action]
	if permission == "" {
		return nil, pkg.ErrForbidden
	}

	if err := actor.Require(permission); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	if action == "reject" || action == "revise" || action == "cancel" {
		if strings.TrimSpace(note) == "" || len(note) > 1000 {
			return nil, pkg.ErrInvalidInput
		}
	}

	return s.idempotencyExecutor.Execute(ctx, actor, "purchase_orders."+action, key, map[string]string{"id": id, "note": note}, func(ctx context.Context) (any, error) {
		order, err := s.purchaseOrderRepo.GetPurchaseOrder(ctx, id, true)
		if err != nil {
			return nil, err
		}

		old := ToPurchaseOrderResponse(order)
		next := ""
		switch action {
		case "submit":
			if order.Status != "DRAFT" {
				return nil, pkg.ErrInvalidState
			}

			if err := s.purchaseOrderRepo.ValidateReferences(ctx, order); err != nil {
				return nil, err
			}

			if err := s.purchaseOrderRepo.RequestApproval(ctx, id, actor.ID); err != nil {
				return nil, err
			}

			next = "SUBMITTED"
		case "approve", "reject":
			if order.CreatedBy == actor.ID {
				return nil, pkg.NewError("SEPARATION_OF_DUTIES", "Purchase order creator cannot decide its approval", 403)
			}

			if order.Status != "SUBMITTED" {
				return nil, pkg.ErrInvalidState
			}

			next = "APPROVED"
			if action == "reject" {
				next = "REJECTED"
			}

			if err := s.purchaseOrderRepo.DecideApproval(ctx, id, actor.ID, next, note); err != nil {
				return nil, err
			}

		case "revise":
			if order.Status != "REJECTED" {
				return nil, pkg.ErrInvalidState
			}

			next = "DRAFT"
		case "cancel":
			if order.Status == "CANCELLED" || order.Status == "CLOSED" {
				return nil, pkg.ErrInvalidState
			}

			active, err := s.purchaseOrderRepo.HasActivity(ctx, id)
			if err != nil {
				return nil, err
			}

			if active {
				return nil, pkg.ErrInvalidState
			}

			next = "CANCELLED"
		}

		if err := s.purchaseOrderRepo.SetStatus(ctx, id, next, actor.ID); err != nil {
			return nil, err
		}

		updated, err := s.purchaseOrderRepo.GetPurchaseOrder(ctx, id, false)
		if err != nil {
			return nil, err
		}

		response := ToPurchaseOrderResponse(updated)
		if err := s.auditLogger.Record(ctx, actor, "purchase_orders."+action, "PURCHASE_ORDER", id, old, map[string]any{"document": response, "decision_note": note}); err != nil {
			return nil, err
		}

		return response, nil
	})
}
