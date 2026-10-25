package stock

import (
	"context"

	"github.com/shopspring/decimal"

	"mini-erp/internal/inventory/domain"
	"mini-erp/pkg"
)

// Interface

type StockService interface {
	domain.StockService
	GetBalances(ctx context.Context, query pkg.ListQuery) ([]BalanceResponse, int64, error)
	GetMovements(ctx context.Context, query pkg.ListQuery) ([]MovementResponse, int64, error)
}

// Implementation

type stockService struct {
	stockRepo domain.StockRepository
	txManager pkg.TransactionManager
}

func NewStockService(stockRepo domain.StockRepository, txManager pkg.TransactionManager) StockService {
	return &stockService{
		stockRepo: stockRepo,
		txManager: txManager,
	}
}

func (s *stockService) GetBalances(ctx context.Context, query pkg.ListQuery) ([]BalanceResponse, int64, error) {
	actor, err := scope(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	entities, total, err := s.stockRepo.GetBalances(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]BalanceResponse, len(entities))
	for i := range entities {
		response[i] = ToBalanceResponse(entities[i])
	}

	return response, total, nil
}

func (s *stockService) GetMovements(ctx context.Context, query pkg.ListQuery) ([]MovementResponse, int64, error) {
	actor, err := scope(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	entities, total, err := s.stockRepo.GetMovements(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]MovementResponse, len(entities))
	for i := range entities {
		response[i] = ToMovementResponse(entities[i])
	}

	return response, total, nil
}

func (s *stockService) Receive(ctx context.Context, warehouseID, receiptID string, lines []domain.StockLine) error {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("goods_receipts.post"); err != nil {
		return err
	}

	if err := actor.RequireWarehouse(warehouseID); err != nil {
		return err
	}

	for _, line := range lines {
		if !line.Quantity.IsPositive() {
			return pkg.ErrInvalidInput
		}
	}

	return s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.stockRepo.LockBalances(ctx, warehouseID, lines); err != nil {
			return err
		}

		for _, line := range lines {
			if err := s.stockRepo.CreateMovement(ctx, actor.ID, warehouseID, "GOODS_RECEIPT", receiptID, line); err != nil {
				return err
			}

			if err := s.stockRepo.ChangeBalance(ctx, warehouseID, line.ProductID, line.Quantity, decimal.Zero); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *stockService) Reserve(ctx context.Context, warehouseID string, lines []domain.StockLine) error {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("inventory.reserve"); err != nil {
		return err
	}

	if err := actor.RequireWarehouse(warehouseID); err != nil {
		return err
	}

	for _, line := range lines {
		if !line.Quantity.IsPositive() {
			return pkg.ErrInvalidInput
		}
	}

	return s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.stockRepo.LockBalances(ctx, warehouseID, lines); err != nil {
			return err
		}

		for _, line := range lines {
			if err := s.stockRepo.ChangeBalance(ctx, warehouseID, line.ProductID, decimal.Zero, line.Quantity); err != nil {
				return err
			}

			if err := s.stockRepo.CreateReservation(ctx, actor.ID, warehouseID, line); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *stockService) Ship(ctx context.Context, warehouseID, shipmentID string, lines []domain.StockLine) error {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("shipments.post"); err != nil {
		return err
	}

	if err := actor.RequireWarehouse(warehouseID); err != nil {
		return err
	}

	for _, line := range lines {
		if !line.Quantity.IsPositive() {
			return pkg.ErrInvalidInput
		}
	}

	return s.txManager.Within(ctx, func(ctx context.Context) error {
		if err := s.stockRepo.LockBalances(ctx, warehouseID, lines); err != nil {
			return err
		}

		for _, line := range lines {
			if err := s.stockRepo.ConsumeReservations(ctx, line.OrderItemID, line.Quantity); err != nil {
				return err
			}

			if err := s.stockRepo.ChangeBalance(ctx, warehouseID, line.ProductID, line.Quantity.Neg(), line.Quantity.Neg()); err != nil {
				return err
			}

			movement := line
			movement.Quantity = line.Quantity.Neg()
			if err := s.stockRepo.CreateMovement(ctx, actor.ID, warehouseID, "SHIPMENT", shipmentID, movement); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *stockService) Release(ctx context.Context, warehouseID, orderID string) error {
	if err := pkg.ActorFrom(ctx).Require("sales_orders.cancel"); err != nil {
		return err
	}

	return s.txManager.Within(ctx, func(ctx context.Context) error {
		lines, err := s.stockRepo.ReleaseReservations(ctx, orderID)
		if err != nil {
			return err
		}

		if err := s.stockRepo.LockBalances(ctx, warehouseID, lines); err != nil {
			return err
		}

		for _, line := range lines {
			if err := s.stockRepo.ChangeBalance(ctx, warehouseID, line.ProductID, decimal.Zero, line.Quantity.Neg()); err != nil {
				return err
			}
		}

		return nil
	})
}

// Internal Helper

func scope(ctx context.Context, query pkg.ListQuery) (pkg.Actor, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("inventory.read"); err != nil {
		return actor, err
	}

	if warehouse := query.Filters["warehouse_id"]; warehouse != "" && len(actor.WarehouseScope()) > 0 {
		return actor, actor.RequireWarehouse(warehouse)
	}

	return actor, nil
}
