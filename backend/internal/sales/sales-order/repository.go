package salesorder

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mini-erp/internal/infrastructure/postgres"
	inventory "mini-erp/internal/inventory/domain"
	"mini-erp/internal/sales/domain"
	"mini-erp/pkg"
)

type salesOrderRepository struct{ db *gorm.DB }

func NewSalesOrderRepository(db *gorm.DB) domain.SalesOrderRepository {
	return &salesOrderRepository{db: db}
}

func (r *salesOrderRepository) CreateSalesOrder(ctx context.Context, entity *domain.SalesOrderEntity) error {
	db := postgres.DB(ctx, r.db)
	storage := ToSalesOrderStorage(entity)
	if err := db.Omit("Items").Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	if err := db.Create(&storage.Items).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToSalesOrderEntity(storage)
	return nil
}

func (r *salesOrderRepository) UpdateSalesOrder(ctx context.Context, entity *domain.SalesOrderEntity) error {
	db := postgres.DB(ctx, r.db)
	fields := map[string]any{"customer_id": entity.CustomerID, "warehouse_id": entity.WarehouseID, "order_date": entity.OrderDate,
		"subtotal": entity.Subtotal, "discount_amount": entity.DiscountAmount, "total_amount": entity.TotalAmount, "notes": entity.Notes}
	if err := db.Model(&SalesOrderStorage{}).Where("id = ?", entity.ID).Updates(fields).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	if err := db.Where("sales_order_id = ?", entity.ID).Delete(&OrderItemStorage{}).Error; err != nil {
		return err
	}

	storage := ToSalesOrderStorage(entity)
	return postgres.HandleDBError(db.Create(&storage.Items).Error)
}

func (r *salesOrderRepository) GetSalesOrder(ctx context.Context, id string, lock bool) (*domain.SalesOrderEntity, error) {
	db := postgres.DB(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var storage SalesOrderStorage
	if err := db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("product_id, id")
	}).Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToSalesOrderEntity(&storage), nil
}

func (r *salesOrderRepository) GetSalesOrders(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.SalesOrderEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&SalesOrderStorage{})
	if len(scope) > 0 {
		db = db.Where("warehouse_id IN ?", scope)
	}

	if query.Search != "" {
		db = db.Where("so_number ILIKE ?", "%"+query.Search+"%")
	}

	for _, key := range []string{"status", "warehouse_id", "customer_id", "created_by"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	if value := query.Filters["from"]; value != "" {
		db = db.Where("order_date >= ?", value)
	}

	if value := query.Filters["to"]; value != "" {
		db = db.Where("order_date <= ?", value)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []SalesOrderStorage
	if err := db.Preload("Items").Order(query.Order(map[string]string{"created_at": "created_at", "order_date": "order_date", "total_amount": "total_amount", "so_number": "so_number"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.SalesOrderEntity, len(rows))
	for i := range rows {
		entities[i] = *ToSalesOrderEntity(&rows[i])
	}

	return entities, total, nil
}

func (r *salesOrderRepository) ValidateReferences(ctx context.Context, order *domain.SalesOrderEntity) error {
	db := postgres.DB(ctx, r.db)
	for table, id := range map[string]string{"customers": order.CustomerID, "warehouses": order.WarehouseID} {
		var count int64
		if err := db.Table(table).Where("id = ? AND is_active", id).Count(&count).Error; err != nil {
			return err
		}

		if count == 0 {
			return pkg.ErrInvalidInput
		}
	}

	ids := make([]string, len(order.Items))
	for i := range order.Items {
		ids[i] = order.Items[i].ProductID
	}

	var count int64
	if err := db.Table("products").Where("id IN ? AND is_active", ids).Count(&count).Error; err != nil {
		return err
	}

	if count != int64(len(ids)) {
		return pkg.ErrInvalidInput
	}

	return nil
}

func (r *salesOrderRepository) SetStatus(ctx context.Context, id, status, actorID string) error {
	fields := map[string]any{"status": status}
	if status == "CONFIRMED" {
		fields["confirmed_at"] = time.Now().UTC()
	}

	db := postgres.DB(ctx, r.db)
	if err := db.Model(&SalesOrderStorage{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	if status == "CANCELLED" {
		return db.Exec("UPDATE shipments SET status = 'CANCELLED', updated_at = CURRENT_TIMESTAMP WHERE sales_order_id = ? AND status = 'DRAFT'", id).Error
	}

	return nil
}

func (r *salesOrderRepository) HasActivity(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := postgres.DB(ctx, r.db).Raw("SELECT EXISTS (SELECT 1 FROM shipments WHERE sales_order_id = ? AND status = 'SHIPPED') OR EXISTS (SELECT 1 FROM invoices WHERE sales_order_id = ? AND status NOT IN ('DRAFT', 'CANCELLED'))", id, id).Scan(&exists).Error
	return exists, err
}

func (r *salesOrderRepository) CheckCredit(ctx context.Context, order *domain.SalesOrderEntity) error {
	db := postgres.DB(ctx, r.db)
	var limit decimal.Decimal
	if err := db.Raw("SELECT credit_limit FROM customers WHERE id = ? FOR UPDATE", order.CustomerID).Scan(&limit).Error; err != nil {
		return err
	}

	if limit.IsZero() {
		return nil
	}

	var exposure decimal.Decimal
	if err := db.Raw(`SELECT COALESCE(SUM(so.total_amount - COALESCE(i.paid_amount, 0)), 0) FROM sales_orders so
		LEFT JOIN invoices i ON i.sales_order_id = so.id WHERE so.customer_id = ? AND so.id <> ? AND so.status IN ('CONFIRMED', 'PARTIALLY_SHIPPED', 'SHIPPED')`, order.CustomerID, order.ID).Scan(&exposure).Error; err != nil {
		return err
	}

	if exposure.Add(order.TotalAmount).GreaterThan(limit) {
		return pkg.NewError("CREDIT_LIMIT_EXCEEDED", "Customer credit limit would be exceeded", 409)
	}

	return nil
}

func (r *salesOrderRepository) Unreserved(ctx context.Context, id string) ([]inventory.StockLine, error) {
	var rows []struct {
		ID        string
		ProductID string
		Quantity  decimal.Decimal
		Reserved  decimal.Decimal
		Shipped   decimal.Decimal
	}

	err := postgres.DB(ctx, r.db).Raw(`SELECT i.id, i.product_id, i.quantity,
		COALESCE((SELECT SUM(r.quantity) FROM stock_reservations r WHERE r.sales_order_item_id = i.id AND r.status = 'ACTIVE'), 0) AS reserved,
		COALESCE((SELECT SUM(si.quantity_shipped) FROM shipment_items si JOIN shipments s ON s.id = si.shipment_id WHERE si.sales_order_item_id = i.id AND s.status = 'SHIPPED'), 0) AS shipped
		FROM sales_order_items i WHERE i.sales_order_id = ? ORDER BY i.product_id`, id).Scan(&rows).Error
	lines := make([]inventory.StockLine, 0, len(rows))
	for _, row := range rows {
		quantity := row.Quantity.Sub(row.Reserved).Sub(row.Shipped)
		if quantity.IsPositive() {
			lines = append(lines, inventory.StockLine{
				ItemID:    row.ID,
				ProductID: row.ProductID,
				Quantity:  quantity,
			})
		}
	}

	return lines, err
}

func (r *salesOrderRepository) Shipped(ctx context.Context, id string) (map[string]decimal.Decimal, error) {
	var rows []struct {
		ID       string
		Quantity decimal.Decimal
	}

	err := postgres.DB(ctx, r.db).Raw(`SELECT i.id, COALESCE(SUM(CASE WHEN s.status = 'SHIPPED' THEN si.quantity_shipped ELSE 0 END), 0) AS quantity
		FROM sales_order_items i LEFT JOIN shipment_items si ON si.sales_order_item_id = i.id
		LEFT JOIN shipments s ON s.id = si.shipment_id WHERE i.sales_order_id = ? GROUP BY i.id`, id).Scan(&rows).Error
	quantities := make(map[string]decimal.Decimal)
	for _, row := range rows {
		quantities[row.ID] = row.Quantity
	}

	return quantities, err
}

func (r *salesOrderRepository) Reserved(ctx context.Context, id string) (map[string]decimal.Decimal, error) {
	var rows []struct {
		ID       string
		Quantity decimal.Decimal
	}

	err := postgres.DB(ctx, r.db).Raw(`SELECT i.id, COALESCE(SUM(CASE WHEN r.status = 'ACTIVE' THEN r.quantity ELSE 0 END), 0) AS quantity
		FROM sales_order_items i LEFT JOIN stock_reservations r ON r.sales_order_item_id = i.id WHERE i.sales_order_id = ? GROUP BY i.id`, id).Scan(&rows).Error
	quantities := make(map[string]decimal.Decimal)
	for _, row := range rows {
		quantities[row.ID] = row.Quantity
	}

	return quantities, err
}
