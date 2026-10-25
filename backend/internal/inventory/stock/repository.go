package stock

import (
	"context"
	"sort"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/inventory/domain"
	"mini-erp/pkg"
)

type stockRepository struct{ db *gorm.DB }

func NewStockRepository(db *gorm.DB) domain.StockRepository {
	return &stockRepository{db: db}
}

func (r *stockRepository) LockBalances(ctx context.Context, warehouseID string, lines []domain.StockLine) error {
	ids := make([]string, len(lines))
	for i := range lines {
		ids[i] = lines[i].ProductID
	}

	sort.Strings(ids)
	db := postgres.DB(ctx, r.db)
	for _, id := range ids {
		if err := db.Exec("INSERT INTO inventory_balances (product_id, warehouse_id) VALUES (?, ?) ON CONFLICT (product_id, warehouse_id) DO NOTHING", id, warehouseID).Error; err != nil {
			return err
		}

		var balance BalanceStorage
		if err := db.Raw("SELECT * FROM inventory_balances WHERE product_id = ? AND warehouse_id = ? FOR UPDATE", id, warehouseID).Scan(&balance).Error; err != nil {
			return err
		}
	}

	return nil
}

func (r *stockRepository) ChangeBalance(ctx context.Context, warehouseID, productID string, onHand, reserved decimal.Decimal) error {
	result := postgres.DB(ctx, r.db).Exec(`UPDATE inventory_balances SET quantity_on_hand = quantity_on_hand + ?, quantity_reserved = quantity_reserved + ?, version = version + 1, updated_at = CURRENT_TIMESTAMP
		WHERE warehouse_id = ? AND product_id = ? AND quantity_on_hand + ? >= 0 AND quantity_reserved + ? >= 0 AND quantity_reserved + ? <= quantity_on_hand + ?`, onHand, reserved, warehouseID, productID, onHand, reserved, reserved, onHand)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		var balance BalanceStorage
		if err := postgres.DB(ctx, r.db).Where("product_id = ? AND warehouse_id = ?", productID, warehouseID).First(&balance).Error; err != nil {
			return postgres.HandleDBError(err)
		}

		return &pkg.AppError{
			Code:    "INSUFFICIENT_STOCK",
			Message: "Available stock is insufficient",
			Status:  409,
			Details: map[string]string{"product_id": productID, "requested": reserved.Abs().StringFixed(3), "available": balance.QuantityOnHand.Sub(balance.QuantityReserved).StringFixed(3)},
		}
	}

	return nil
}

func (r *stockRepository) CreateMovement(ctx context.Context, actorID, warehouseID, movementType, referenceID string, line domain.StockLine) error {
	return postgres.HandleDBError(postgres.DB(ctx, r.db).Exec(`INSERT INTO stock_movements (product_id, warehouse_id, movement_type, quantity_delta, reference_type, reference_id, reference_item_id, performed_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, line.ProductID, warehouseID, movementType, line.Quantity, movementType, referenceID, line.ItemID, actorID).Error)
}

func (r *stockRepository) CreateReservation(ctx context.Context, actorID, warehouseID string, line domain.StockLine) error {
	return postgres.DB(ctx, r.db).Exec("INSERT INTO stock_reservations (sales_order_item_id, warehouse_id, quantity, created_by) VALUES (?, ?, ?, ?)", line.ItemID, warehouseID, line.Quantity, actorID).Error
}

func (r *stockRepository) ConsumeReservations(ctx context.Context, itemID string, quantity decimal.Decimal) error {
	type reservation struct {
		ID          string
		Quantity    decimal.Decimal
		WarehouseID string
		CreatedBy   string
	}

	db := postgres.DB(ctx, r.db)
	var rows []reservation
	if err := db.Raw("SELECT id, quantity, warehouse_id, created_by FROM stock_reservations WHERE sales_order_item_id = ? AND status = 'ACTIVE' ORDER BY created_at, id FOR UPDATE", itemID).Scan(&rows).Error; err != nil {
		return err
	}

	remaining := quantity
	for _, row := range rows {
		if remaining.IsZero() {
			break
		}

		consumed := decimal.Min(remaining, row.Quantity)
		if consumed.Equal(row.Quantity) {
			if err := db.Exec("UPDATE stock_reservations SET status = 'CONSUMED', released_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?", row.ID).Error; err != nil {
				return err
			}

		} else {
			if err := db.Exec("UPDATE stock_reservations SET quantity = quantity - ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", consumed, row.ID).Error; err != nil {
				return err
			}

			if err := db.Exec("INSERT INTO stock_reservations (sales_order_item_id, warehouse_id, quantity, status, created_by, released_at) VALUES (?, ?, ?, 'CONSUMED', ?, CURRENT_TIMESTAMP)", itemID, row.WarehouseID, consumed, row.CreatedBy).Error; err != nil {
				return err
			}
		}

		remaining = remaining.Sub(consumed)
	}

	if !remaining.IsZero() {
		return pkg.NewError("INSUFFICIENT_RESERVATION", "Shipment exceeds the order reservation", 409)
	}

	return nil
}

func (r *stockRepository) ReleaseReservations(ctx context.Context, orderID string) ([]domain.StockLine, error) {
	db := postgres.DB(ctx, r.db)
	var lines []domain.StockLine
	if err := db.Raw(`SELECT r.id AS item_id, i.product_id, r.quantity FROM stock_reservations r JOIN sales_order_items i ON i.id = r.sales_order_item_id
		WHERE i.sales_order_id = ? AND r.status = 'ACTIVE' ORDER BY i.product_id, r.id FOR UPDATE OF r`, orderID).Scan(&lines).Error; err != nil {
		return nil, err
	}

	for _, line := range lines {
		if err := db.Exec("UPDATE stock_reservations SET status = 'RELEASED', released_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?", line.ItemID).Error; err != nil {
			return nil, err
		}
	}

	return lines, nil
}

func (r *stockRepository) GetBalances(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.BalanceEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&BalanceStorage{})
	if len(scope) > 0 {
		db = db.Where("warehouse_id IN ?", scope)
	}

	for _, key := range []string{"warehouse_id", "product_id"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []BalanceStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "updated_at": "updated_at", "quantity_on_hand": "quantity_on_hand"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.BalanceEntity, len(rows))
	for i, row := range rows {
		entities[i] = domain.BalanceEntity(row)
	}

	return entities, total, nil
}

func (r *stockRepository) GetMovements(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.MovementEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&MovementStorage{})
	if len(scope) > 0 {
		db = db.Where("warehouse_id IN ?", scope)
	}

	for _, key := range []string{"warehouse_id", "product_id", "movement_type", "reference_id"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []MovementStorage
	if err := db.Order(query.Order(map[string]string{"occurred_at": "occurred_at", "quantity_delta": "quantity_delta"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.MovementEntity, len(rows))
	for i, row := range rows {
		entities[i] = domain.MovementEntity(row)
	}

	return entities, total, nil
}
