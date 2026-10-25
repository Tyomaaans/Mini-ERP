package goodsreceipt

import (
	"context"
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	inventory "mini-erp/internal/inventory/domain"
	"mini-erp/internal/purchasing/domain"
	"mini-erp/pkg"
)

// Interface

type GoodsReceiptService interface {
	CreateGoodsReceipt(ctx context.Context, req CreateGoodsReceiptRequest) (*GoodsReceiptResponse, error)
	GetGoodsReceipt(ctx context.Context, id string) (*GoodsReceiptResponse, error)
	GetGoodsReceipts(ctx context.Context, query pkg.ListQuery) ([]GoodsReceiptResponse, int64, error)
	Post(ctx context.Context, id, key string) (json.RawMessage, error)
}

// Implementation

type goodsReceiptService struct {
	goodsReceiptRepo    domain.GoodsReceiptRepository
	purchaseOrderRepo   domain.PurchaseOrderRepository
	stockSvc            inventory.StockService
	validate            *validator.Validate
	txManager           pkg.TransactionManager
	auditLogger         pkg.AuditLogger
	documentSequence    pkg.DocumentSequence
	idempotencyExecutor pkg.IdempotencyExecutor
}

func NewGoodsReceiptService(
	goodsReceiptRepo    domain.GoodsReceiptRepository,
	purchaseOrderRepo   domain.PurchaseOrderRepository,
	stockSvc            inventory.StockService,
	validate            *validator.Validate,
	txManager           pkg.TransactionManager,
	auditLogger         pkg.AuditLogger,
	documentSequence    pkg.DocumentSequence,
	idempotencyExecutor pkg.IdempotencyExecutor,
) GoodsReceiptService {
	return &goodsReceiptService{
		goodsReceiptRepo:    goodsReceiptRepo,
		purchaseOrderRepo:   purchaseOrderRepo,
		stockSvc:            stockSvc,
		validate:            validate,
		txManager:           txManager,
		auditLogger:         auditLogger,
		documentSequence:    documentSequence,
		idempotencyExecutor: idempotencyExecutor,
	}
}

func (s *goodsReceiptService) CreateGoodsReceipt(ctx context.Context, req CreateGoodsReceiptRequest) (*GoodsReceiptResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("goods_receipts.create"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateStruct(s.validate, req); err != nil {
		return nil, err
	}

	var response *GoodsReceiptResponse
	err := s.txManager.Within(ctx, func(ctx context.Context) error {
		order, err := s.purchaseOrderRepo.GetPurchaseOrder(ctx, req.OrderID, true)
		if err != nil {
			return err
		}

		if err := actor.RequireWarehouse(order.WarehouseID); err != nil {
			return err
		}

		if order.Status != "APPROVED" {
			return pkg.ErrInvalidState
		}

		if req.WarehouseID != "" && req.WarehouseID != order.WarehouseID {
			return pkg.ErrInvalidInput
		}

		entity := &domain.GoodsReceiptEntity{
			ID:          uuid.NewString(),
			OrderID:     order.ID,
			WarehouseID: order.WarehouseID,
			Status:      "DRAFT",
			ReceivedBy:  actor.ID,
			Notes:       req.Notes,
			Items:       make([]domain.GoodsReceiptItemEntity, len(req.Items)),
		}

		for i, item := range req.Items {
			quantity, err := pkg.ParseDecimal(item.Quantity, 3, true)
			if err != nil {
				return err
			}

			entity.Items[i] = domain.GoodsReceiptItemEntity{
				ID:          uuid.NewString(),
				DocumentID:  entity.ID,
				OrderItemID: item.OrderItemID,
				Quantity:    quantity,
			}
		}

		if _, err := s.lines(ctx, entity, order); err != nil {
			return err
		}

		number, err := s.documentSequence.Next(ctx, "GR")
		if err != nil {
			return err
		}

		entity.GRNumber = number
		if err := s.goodsReceiptRepo.CreateGoodsReceipt(ctx, entity); err != nil {
			return err
		}

		response = ToGoodsReceiptResponse(entity)
		return s.auditLogger.Record(ctx, actor, "goods_receipts.create", "GOODS_RECEIPT", entity.ID, nil, response)
	})
	return response, err
}

func (s *goodsReceiptService) GetGoodsReceipt(ctx context.Context, id string) (*GoodsReceiptResponse, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("goods_receipts.read"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.goodsReceiptRepo.GetGoodsReceipt(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if len(actor.WarehouseScope()) > 0 {
		if err := actor.RequireWarehouse(entity.WarehouseID); err != nil {
			return nil, err
		}
	}

	return ToGoodsReceiptResponse(entity), nil
}

func (s *goodsReceiptService) GetGoodsReceipts(ctx context.Context, query pkg.ListQuery) ([]GoodsReceiptResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("goods_receipts.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.goodsReceiptRepo.GetGoodsReceipts(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]GoodsReceiptResponse, len(entities))
	for i := range entities {
		response[i] = *ToGoodsReceiptResponse(&entities[i])
	}

	return response, total, nil
}

func (s *goodsReceiptService) Post(ctx context.Context, id, key string) (json.RawMessage, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("goods_receipts.post"); err != nil {
		return nil, err
	}

	if err := pkg.ValidateID(id); err != nil {
		return nil, err
	}

	entity, err := s.goodsReceiptRepo.GetGoodsReceipt(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if err := actor.RequireWarehouse(entity.WarehouseID); err != nil {
		return nil, err
	}

	return s.idempotencyExecutor.Execute(ctx, actor, "goods_receipts.post", key, map[string]string{"id": id}, func(ctx context.Context) (any, error) {
		order, err := s.purchaseOrderRepo.GetPurchaseOrder(ctx, entity.OrderID, true)
		if err != nil {
			return nil, err
		}

		entity, err := s.goodsReceiptRepo.GetGoodsReceipt(ctx, id, true)
		if err != nil {
			return nil, err
		}

		if entity.Status != "DRAFT" || (order.Status != "APPROVED") {
			return nil, pkg.ErrInvalidState
		}

		lines, err := s.lines(ctx, entity, order)
		if err != nil {
			return nil, err
		}

		if err := s.stockSvc.Receive(ctx, entity.WarehouseID, entity.ID, lines); err != nil {
			return nil, err
		}

		if err := s.goodsReceiptRepo.Post(ctx, id, actor.ID); err != nil {
			return nil, err
		}

		quantities, err := s.purchaseOrderRepo.Received(ctx, order.ID)
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
			next = "CLOSED"
		}

		if next != "" {
			if err := s.purchaseOrderRepo.SetStatus(ctx, order.ID, next, actor.ID); err != nil {
				return nil, err
			}

			if err := s.auditLogger.Record(ctx, actor, "purchase_orders.status", "PURCHASE_ORDER", order.ID, map[string]string{"status": order.Status}, map[string]string{"status": next}); err != nil {
				return nil, err
			}
		}

		updated, err := s.goodsReceiptRepo.GetGoodsReceipt(ctx, id, false)
		if err != nil {
			return nil, err
		}

		response := ToGoodsReceiptResponse(updated)
		if err := s.auditLogger.Record(ctx, actor, "goods_receipts.post", "GOODS_RECEIPT", id, ToGoodsReceiptResponse(entity), response); err != nil {
			return nil, err
		}

		return response, nil
	})
}

// Internal Helper

func (s *goodsReceiptService) lines(ctx context.Context, entity *domain.GoodsReceiptEntity, order *domain.PurchaseOrderEntity) ([]inventory.StockLine, error) {
	quantities, err := s.purchaseOrderRepo.Received(ctx, order.ID)
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

		lines[i] = inventory.StockLine{
			ItemID:    item.ID,
			ProductID: parent.ProductID,
			Quantity:  quantity,
		}
	}

	return lines, nil
}
