package purchaseorder

import (
	"context"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/purchasing/domain"
	"mini-erp/pkg"
)

type purchaseOrderRepository struct{ db *gorm.DB }

func NewPurchaseOrderRepository(db *gorm.DB) domain.PurchaseOrderRepository {
	return &purchaseOrderRepository{db: db}
}

func (r *purchaseOrderRepository) CreatePurchaseOrder(ctx context.Context, entity *domain.PurchaseOrderEntity) error {
	db := postgres.DB(ctx, r.db)
	storage := ToPurchaseOrderStorage(entity)
	if err := db.Omit("Items").Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	if err := db.Create(&storage.Items).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToPurchaseOrderEntity(storage)
	return nil
}

func (r *purchaseOrderRepository) UpdatePurchaseOrder(ctx context.Context, entity *domain.PurchaseOrderEntity) error {
	db := postgres.DB(ctx, r.db)
	fields := map[string]any{"supplier_id": entity.SupplierID, "warehouse_id": entity.WarehouseID, "order_date": entity.OrderDate,
		"subtotal": entity.Subtotal, "discount_amount": entity.DiscountAmount, "total_amount": entity.TotalAmount, "notes": entity.Notes}
	fields["expected_date"] = entity.ExpectedDate
	if err := db.Model(&PurchaseOrderStorage{}).Where("id = ?", entity.ID).Updates(fields).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	if err := db.Where("purchase_order_id = ?", entity.ID).Delete(&OrderItemStorage{}).Error; err != nil {
		return err
	}

	storage := ToPurchaseOrderStorage(entity)
	return postgres.HandleDBError(db.Create(&storage.Items).Error)
}

func (r *purchaseOrderRepository) GetPurchaseOrder(ctx context.Context, id string, lock bool) (*domain.PurchaseOrderEntity, error) {
	db := postgres.DB(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var storage PurchaseOrderStorage
	if err := db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("product_id, id")
	}).Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToPurchaseOrderEntity(&storage), nil
}

func (r *purchaseOrderRepository) GetPurchaseOrders(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.PurchaseOrderEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&PurchaseOrderStorage{})
	if len(scope) > 0 {
		db = db.Where("warehouse_id IN ?", scope)
	}

	if query.Search != "" {
		db = db.Where("po_number ILIKE ?", "%"+query.Search+"%")
	}

	for _, key := range []string{"status", "warehouse_id", "supplier_id", "created_by"} {
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

	var rows []PurchaseOrderStorage
	if err := db.Preload("Items").Order(query.Order(map[string]string{"created_at": "created_at", "order_date": "order_date", "total_amount": "total_amount", "po_number": "po_number"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.PurchaseOrderEntity, len(rows))
	for i := range rows {
		entities[i] = *ToPurchaseOrderEntity(&rows[i])
	}

	return entities, total, nil
}

func (r *purchaseOrderRepository) ValidateReferences(ctx context.Context, order *domain.PurchaseOrderEntity) error {
	db := postgres.DB(ctx, r.db)
	for table, id := range map[string]string{"suppliers": order.SupplierID, "warehouses": order.WarehouseID} {
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

func (r *purchaseOrderRepository) SetStatus(ctx context.Context, id, status, actorID string) error {
	fields := map[string]any{"status": status}
	if status == "APPROVED" {
		fields["approved_by"] = actorID
		fields["approved_at"] = time.Now().UTC()
	}

	if status == "SUBMITTED" {
		fields["submitted_at"] = time.Now().UTC()
	}

	if status == "DRAFT" {
		fields["submitted_at"] = nil
		fields["approved_by"] = nil
		fields["approved_at"] = nil
	}

	db := postgres.DB(ctx, r.db)
	if err := db.Model(&PurchaseOrderStorage{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	if status == "CANCELLED" {
		if err := db.Exec("UPDATE approval_requests SET status = 'CANCELLED', decided_at = CURRENT_TIMESTAMP, decision_note = 'Order cancelled', updated_at = CURRENT_TIMESTAMP WHERE document_type = 'PURCHASE_ORDER' AND document_id = ? AND status = 'PENDING'", id).Error; err != nil {
			return err
		}

		return db.Exec("UPDATE goods_receipts SET status = 'CANCELLED', updated_at = CURRENT_TIMESTAMP WHERE purchase_order_id = ? AND status = 'DRAFT'", id).Error
	}

	return nil
}

func (r *purchaseOrderRepository) HasActivity(ctx context.Context, id string) (bool, error) {
	var exists bool
	err := postgres.DB(ctx, r.db).Raw("SELECT EXISTS (SELECT 1 FROM goods_receipts WHERE purchase_order_id = ? AND status = 'POSTED') OR EXISTS (SELECT 1 FROM invoices WHERE purchase_order_id = ? AND status NOT IN ('DRAFT', 'CANCELLED'))", id, id).Scan(&exists).Error
	return exists, err
}

func (r *purchaseOrderRepository) RequestApproval(ctx context.Context, id, userID string) error {
	return postgres.DB(ctx, r.db).Exec("INSERT INTO approval_requests (document_type, document_id, requested_by) VALUES ('PURCHASE_ORDER', ?, ?)", id, userID).Error
}

func (r *purchaseOrderRepository) DecideApproval(ctx context.Context, id, userID, status, note string) error {
	result := postgres.DB(ctx, r.db).Exec(`UPDATE approval_requests SET status = ?, approver_id = ?, decided_at = CURRENT_TIMESTAMP, decision_note = ?, updated_at = CURRENT_TIMESTAMP
		WHERE document_type = 'PURCHASE_ORDER' AND document_id = ? AND status = 'PENDING'`, status, userID, note, id)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected != 1 {
		return pkg.ErrInvalidState
	}

	return nil
}

func (r *purchaseOrderRepository) Received(ctx context.Context, id string) (map[string]decimal.Decimal, error) {
	var rows []struct {
		ID       string
		Quantity decimal.Decimal
	}

	err := postgres.DB(ctx, r.db).Raw(`SELECT i.id, COALESCE(SUM(CASE WHEN gr.status = 'POSTED' THEN gi.quantity_received ELSE 0 END), 0) AS quantity
		FROM purchase_order_items i LEFT JOIN goods_receipt_items gi ON gi.purchase_order_item_id = i.id
		LEFT JOIN goods_receipts gr ON gr.id = gi.goods_receipt_id WHERE i.purchase_order_id = ? GROUP BY i.id`, id).Scan(&rows).Error
	quantities := make(map[string]decimal.Decimal)
	for _, row := range rows {
		quantities[row.ID] = row.Quantity
	}

	return quantities, err
}
