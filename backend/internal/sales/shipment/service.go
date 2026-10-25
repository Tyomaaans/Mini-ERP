package shipment

import (
	"context"
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	inventory "mini-erp/internal/inventory/domain"
	"mini-erp/internal/sales/domain"
	"mini-erp/pkg"
)

// Interface

type ShipmentService interface {
	CreateShipment(ctx context.Context, req CreateShipmentRequest) (*ShipmentResponse, error)
	GetShipment(ctx context.Context, id string) (*ShipmentResponse, error)
	GetShipments(ctx context.Context, query pkg.ListQuery) ([]ShipmentResponse, int64, error)
	Post(ctx context.Context, id, key string) (json.RawMessage, error)
}

// Implementation

type shipmentService struct {
	shipmentRepo        domain.ShipmentRepository
	salesOrderRepo      domain.SalesOrderRepository
	stockSvc            inventory.StockService
	validate            *validator.Validate
	txManager           pkg.TransactionManager
	auditLogger         pkg.AuditLogger
	documentSequence    pkg.DocumentSequence
	idempotencyExecutor pkg.IdempotencyExecutor
}

func NewShipmentService(
	shipmentRepo        domain.ShipmentRepository,
	salesOrderRepo      domain.SalesOrderRepository,
	stockSvc            inventory.StockService,
	validate            *validator.Validate,
	txManager           pkg.TransactionManager,
	auditLogger         pkg.AuditLogger,
	documentSequence    pkg.DocumentSequence,
	idempotencyExecutor pkg.IdempotencyExecutor,
) ShipmentService {
	return &shipmentService{
		shipmentRepo:        shipmentRepo,
		salesOrderRepo:      salesOrderRepo,
		stockSvc:            stockSvc,
		validate:            validate,
		txManager:           txManager,
		auditLogger:         auditLogger,
		documentSequence:    documentSequence,
		idempotencyExecutor: idempotencyExecutor,
	}
}

func (s *shipmentService) CreateShipment(ctx context.Context, req CreateShipmentRequest) (*ShipmentResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("shipments.create"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	var response *ShipmentResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		order, err := s.salesOrderRepo.GetSalesOrder(ctx, req.OrderID, true)
		if err != nil {
			return err
		}

		if err := actor.RequireWarehouse(order.WarehouseID); err != nil {
			return err
		}

		if order.Status != "CONFIRMED" && order.Status != "PARTIALLY_SHIPPED" {
			return pkg.ErrInvalidState
		}

		if req.WarehouseID != "" && req.WarehouseID != order.WarehouseID {
			return pkg.ErrInvalidInput
		}

		entity := &domain.ShipmentEntity{
			ID:          uuid.NewString(),
			OrderID:     order.ID,
			WarehouseID: order.WarehouseID,
			Status:      "DRAFT",
			Items:       make([]domain.ShipmentItemEntity, len(req.Items)),
		}

		for i, item := range req.Items {
			quantity, err := pkg.ParseDecimal(item.Quantity, 3, true)
			if err != nil {
				return err
			}

			entity.Items[i] = domain.ShipmentItemEntity{
				ID:          uuid.NewString(),
				DocumentID:  entity.ID,
				OrderItemID: item.OrderItemID,
				Quantity:    quantity,
			}
		}

		if _, err := s.lines(ctx, entity, order); err != nil {
			return err
		}

		number, err := s.documentSequence.Next(ctx, "SHP")
		if err != nil {
			return err
		}

		entity.ShipmentNumber = number
		if err := s.shipmentRepo.CreateShipment(ctx, entity); err != nil {
			return err
		}

		response = ToShipmentResponse(entity)
		return s.auditLogger.Record(ctx, actor, "shipments.create", "SHIPMENT", entity.ID, nil, response)
	})
	return response, err
}

func (s *shipmentService) GetShipment(ctx context.Context, id string) (*ShipmentResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("shipments.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.shipmentRepo.GetShipment(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if len(actor.WarehouseScope()) > 0 {
		if err := actor.RequireWarehouse(entity.WarehouseID); err != nil {
			return nil, err
		}
	}

	return ToShipmentResponse(entity), nil
}

func (s *shipmentService) GetShipments(ctx context.Context, query pkg.ListQuery) ([]ShipmentResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("shipments.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.shipmentRepo.GetShipments(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]ShipmentResponse, len(entities))
	for i := range entities {
		response[i] = *ToShipmentResponse(&entities[i])
	}

	return response, total, nil
}

func (s *shipmentService) Post(ctx context.Context, id, key string) (json.RawMessage, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("shipments.post"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.shipmentRepo.GetShipment(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if err := actor.RequireWarehouse(entity.WarehouseID); err != nil {
		return nil, err
	}

	return s.idempotencyExecutor.Execute(ctx, actor, "shipments.post", key, map[string]string{"id": id}, func(ctx context.Context) (any, error) {
		order, err := s.salesOrderRepo.GetSalesOrder(ctx, entity.OrderID, true)
		if err != nil {
			return nil, err
		}

		entity, err := s.shipmentRepo.GetShipment(ctx, id, true)
		if err != nil {
			return nil, err
		}

		if entity.Status != "DRAFT" || (order.Status != "CONFIRMED" && order.Status != "PARTIALLY_SHIPPED") {
			return nil, pkg.ErrInvalidState
		}

		lines, err := s.lines(ctx, entity, order)
		if err != nil {
			return nil, err
		}

		if err := s.stockSvc.Ship(ctx, entity.WarehouseID, entity.ID, lines); err != nil {
			return nil, err
		}

		if err := s.shipmentRepo.Post(ctx, id, actor.ID); err != nil {
			return nil, err
		}

		quantities, err := s.salesOrderRepo.Shipped(ctx, order.ID)
		if err != nil {
			return nil, err
		}

		complete := true
		for _, item := range order.Items {
			if !quantities[item.ID].Equal(item.Quantity) {
				complete = false
			}
		}

		next := ""
		if complete {
			next = "SHIPPED"
		}

		if !complete {
			next = "PARTIALLY_SHIPPED"
		}

		if next != "" {
			if err := s.salesOrderRepo.SetStatus(ctx, order.ID, next, actor.ID); err != nil {
				return nil, err
			}

			if err := s.auditLogger.Record(ctx, actor, "sales_orders.status", "SALES_ORDER", order.ID, map[string]string{"status": order.Status}, map[string]string{"status": next}); err != nil {
				return nil, err
			}
		}

		updated, err := s.shipmentRepo.GetShipment(ctx, id, false)
		if err != nil {
			return nil, err
		}

		response := ToShipmentResponse(updated)
		if err := s.auditLogger.Record(ctx, actor, "shipments.post", "SHIPMENT", id, ToShipmentResponse(entity), response); err != nil {
			return nil, err
		}

		return response, nil
	})
}

// Internal Helper

func (s *shipmentService) lines(ctx context.Context, entity *domain.ShipmentEntity, order *domain.SalesOrderEntity) ([]inventory.StockLine, error) {
	quantities, err := s.salesOrderRepo.Shipped(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	reserved, err := s.salesOrderRepo.Reserved(ctx, order.ID)
	if err != nil {
		return nil, err
	}

	ordered := make(map[string]domain.OrderItemEntity)
	for _, item := range order.Items {
		ordered[item.ID] = item
	}

	seen := make(map[string]bool)
	lines := make([]inventory.StockLine, len(entity.Items))
	for i, item := range entity.Items {
		parent, ok := ordered[item.OrderItemID]
		quantity := item.Quantity
		if !ok || seen[item.OrderItemID] || !quantity.IsPositive() {
			return nil, pkg.ErrInvalidInput
		}

		seen[item.OrderItemID] = true
		if quantities[item.OrderItemID].Add(quantity).GreaterThan(parent.Quantity) {
			return nil, pkg.NewError("QUANTITY_EXCEEDED", "Quantity exceeds the remaining ordered quantity", 409)
		}

		if quantity.GreaterThan(reserved[item.OrderItemID]) {
			return nil, pkg.NewError("INSUFFICIENT_RESERVATION", "Shipment exceeds the order reservation", 409)
		}

		lines[i] = inventory.StockLine{
			ItemID:      item.ID,
			ProductID:   parent.ProductID,
			Quantity:    quantity,
			OrderItemID: item.OrderItemID,
		}
	}

	return lines, nil
}
