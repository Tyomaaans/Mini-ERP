package salesorder

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	inventory "mini-erp/internal/inventory/domain"
	"mini-erp/internal/sales/domain"
	"mini-erp/pkg"
)

// Interface

type SalesOrderService interface {
	CreateSalesOrder(ctx context.Context, req CreateSalesOrderRequest) (*SalesOrderResponse, error)
	UpdateSalesOrder(ctx context.Context, id string, req UpdateSalesOrderRequest) (*SalesOrderResponse, error)
	GetSalesOrder(ctx context.Context, id string) (*SalesOrderResponse, error)
	GetSalesOrders(ctx context.Context, query pkg.ListQuery) ([]SalesOrderResponse, int64, error)
	Confirm(ctx context.Context, id, key, note string) (json.RawMessage, error)
	Reserve(ctx context.Context, id, key, note string) (json.RawMessage, error)
	Cancel(ctx context.Context, id, key, note string) (json.RawMessage, error)
}

// Implementation

type salesOrderService struct {
	salesOrderRepo      domain.SalesOrderRepository
	validate            *validator.Validate
	txManager           pkg.TransactionManager
	auditLogger         pkg.AuditLogger
	documentSequence    pkg.DocumentSequence
	idempotencyExecutor pkg.IdempotencyExecutor
	stockSvc            inventory.StockService
}

func NewSalesOrderService(
	salesOrderRepo      domain.SalesOrderRepository,
	validate            *validator.Validate,
	txManager           pkg.TransactionManager,
	auditLogger         pkg.AuditLogger,
	documentSequence    pkg.DocumentSequence,
	idempotencyExecutor pkg.IdempotencyExecutor,
	stockSvc            inventory.StockService,
) SalesOrderService {
	return &salesOrderService{
		salesOrderRepo:      salesOrderRepo,
		validate:            validate,
		txManager:           txManager,
		auditLogger:         auditLogger,
		documentSequence:    documentSequence,
		idempotencyExecutor: idempotencyExecutor,
		stockSvc:            stockSvc,
	}
}

func (s *salesOrderService) CreateSalesOrder(ctx context.Context, req CreateSalesOrderRequest) (*SalesOrderResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("sales_orders.create"); err != nil {
		return nil, err
	}

	entity, err := s.build(req)
	if err != nil {
		return nil, err
	}

	entity.CreatedBy = actor.ID
	err = s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.salesOrderRepo.ValidateReferences(ctx, entity); err != nil {
			return err
		}

		number, err := s.documentSequence.Next(ctx, "SO")
		if err != nil {
			return err
		}

		entity.SONumber = number
		if err := s.salesOrderRepo.CreateSalesOrder(ctx, entity); err != nil {
			return err
		}

		return s.auditLogger.Record(ctx, actor, "sales_orders.create", "SALES_ORDER", entity.ID, nil, ToSalesOrderResponse(entity))
	})
	if err != nil {
		return nil, err
	}

	return ToSalesOrderResponse(entity), nil
}

func (s *salesOrderService) UpdateSalesOrder(ctx context.Context, id string, req UpdateSalesOrderRequest) (*SalesOrderResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("sales_orders.update"); err != nil {
		return nil, err
	}

	if pkg.ValidateID(id) != nil {
		return nil, pkg.ErrInvalidInput
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	if !(req.CustomerID != nil || req.WarehouseID != nil || req.OrderDate != nil || req.DiscountAmount != nil || req.Notes.Set || req.Items != nil) {
		return nil, pkg.ErrInvalidInput
	}

	var response *SalesOrderResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		old, err := s.salesOrderRepo.GetSalesOrder(ctx, id, true)
		if err != nil {
			return err
		}

		if old.CreatedBy != actor.ID {
			return pkg.ErrForbidden
		}

		if old.Status != "DRAFT" {
			return pkg.ErrInvalidState
		}

		merged := CreateSalesOrderRequest{
			CustomerID:     old.CustomerID,
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

		if req.CustomerID != nil {
			merged.CustomerID = *req.CustomerID
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

		if err := s.salesOrderRepo.ValidateReferences(ctx, entity); err != nil {
			return err
		}

		if err := s.salesOrderRepo.UpdateSalesOrder(ctx, entity); err != nil {
			return err
		}

		updated, err := s.salesOrderRepo.GetSalesOrder(ctx, id, false)
		if err != nil {
			return err
		}

		response = ToSalesOrderResponse(updated)
		return s.auditLogger.Record(ctx, actor, "sales_orders.update", "SALES_ORDER", id, ToSalesOrderResponse(old), response)
	})
	return response, err
}

func (s *salesOrderService) GetSalesOrder(ctx context.Context, id string) (*SalesOrderResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("sales_orders.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.salesOrderRepo.GetSalesOrder(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if len(actor.WarehouseScope()) > 0 {
		if err := actor.RequireWarehouse(entity.WarehouseID); err != nil {
			return nil, err
		}
	}

	return ToSalesOrderResponse(entity), nil
}

func (s *salesOrderService) GetSalesOrders(ctx context.Context, query pkg.ListQuery) ([]SalesOrderResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("sales_orders.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.salesOrderRepo.GetSalesOrders(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]SalesOrderResponse, len(entities))
	for i := range entities {
		response[i] = *ToSalesOrderResponse(&entities[i])
	}

	return response, total, nil
}

func (s *salesOrderService) Confirm(ctx context.Context, id, key, note string) (json.RawMessage, error) {
	return s.transition(ctx, id, key, "confirm", note)
}

func (s *salesOrderService) Reserve(ctx context.Context, id, key, note string) (json.RawMessage, error) {
	return s.transition(ctx, id, key, "reserve", note)
}

func (s *salesOrderService) Cancel(ctx context.Context, id, key, note string) (json.RawMessage, error) {
	return s.transition(ctx, id, key, "cancel", note)
}

// Internal Helper

func (s *salesOrderService) build(req CreateSalesOrderRequest) (*domain.SalesOrderEntity, error) {
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

	entity := &domain.SalesOrderEntity{
		ID:             uuid.NewString(),
		CustomerID:     req.CustomerID,
		WarehouseID:    req.WarehouseID,
		Status:         "DRAFT",
		OrderDate:      date,
		Subtotal:       subtotal,
		DiscountAmount: discount,
		TotalAmount:    total,
		Notes:          req.Notes,
		Items:          make([]domain.OrderItemEntity, len(lines)),
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

func (s *salesOrderService) transition(ctx context.Context, id, key, action, note string) (json.RawMessage, error) {
	actor := pkg.ActorFrom(ctx)
	permission := map[string]string{"confirm": "sales_orders.confirm", "reserve": "inventory.reserve", "cancel": "sales_orders.cancel"}[action]
	if permission == "" {
		return nil, pkg.ErrForbidden
	}

	if err := actor.Require(permission); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	if action == "cancel" && (strings.TrimSpace(note) == "" || len(note) > 1000) {
		return nil, pkg.ErrInvalidInput
	}

	order, err := s.salesOrderRepo.GetSalesOrder(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if action == "reserve" {
		if err := actor.RequireWarehouse(order.WarehouseID); err != nil {
			return nil, err
		}

	} else if order.CreatedBy != actor.ID {
		return nil, pkg.ErrForbidden
	}

	return s.idempotencyExecutor.Execute(ctx, actor, "sales_orders."+action, key, map[string]string{"id": id, "note": note}, func(ctx context.Context) (any, error) {
		order, err := s.salesOrderRepo.GetSalesOrder(ctx, id, true)
		if err != nil {
			return nil, err
		}

		old := ToSalesOrderResponse(order)
		switch action {
		case "confirm":
			if order.Status != "DRAFT" {
				return nil, pkg.ErrInvalidState
			}

			if err := s.salesOrderRepo.ValidateReferences(ctx, order); err != nil {
				return nil, err
			}

			if err := s.salesOrderRepo.CheckCredit(ctx, order); err != nil {
				return nil, err
			}

			if err := s.salesOrderRepo.SetStatus(ctx, id, "CONFIRMED", actor.ID); err != nil {
				return nil, err
			}

		case "reserve":
			if order.Status != "CONFIRMED" && order.Status != "PARTIALLY_SHIPPED" {
				return nil, pkg.ErrInvalidState
			}

			lines, err := s.salesOrderRepo.Unreserved(ctx, id)
			if err != nil {
				return nil, err
			}

			if len(lines) == 0 {
				return nil, pkg.NewError("ALREADY_RESERVED", "All remaining order quantities are reserved", 409)
			}

			if err := s.stockSvc.Reserve(ctx, order.WarehouseID, lines); err != nil {
				return nil, err
			}

		case "cancel":
			if order.Status != "DRAFT" && order.Status != "CONFIRMED" {
				return nil, pkg.ErrInvalidState
			}

			active, err := s.salesOrderRepo.HasActivity(ctx, id)
			if err != nil {
				return nil, err
			}

			if active {
				return nil, pkg.ErrInvalidState
			}

			if err := s.stockSvc.Release(ctx, order.WarehouseID, id); err != nil {
				return nil, err
			}

			if err := s.salesOrderRepo.SetStatus(ctx, id, "CANCELLED", actor.ID); err != nil {
				return nil, err
			}
		}

		updated, err := s.salesOrderRepo.GetSalesOrder(ctx, id, false)
		if err != nil {
			return nil, err
		}

		response := ToSalesOrderResponse(updated)
		if err := s.auditLogger.Record(ctx, actor, "sales_orders."+action, "SALES_ORDER", id, old, map[string]any{"document": response, "note": note}); err != nil {
			return nil, err
		}

		return response, nil
	})
}
