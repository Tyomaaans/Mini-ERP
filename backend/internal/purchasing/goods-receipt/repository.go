package goodsreceipt

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/purchasing/domain"
	"mini-erp/pkg"
)

type goodsReceiptRepository struct{ db *gorm.DB }

func NewGoodsReceiptRepository(db *gorm.DB) domain.GoodsReceiptRepository {
	return &goodsReceiptRepository{db: db}
}

func (r *goodsReceiptRepository) CreateGoodsReceipt(ctx context.Context, entity *domain.GoodsReceiptEntity) error {
	db := postgres.DB(ctx, r.db)
	storage := ToGoodsReceiptStorage(entity)
	if err := db.Omit("Items").Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	if err := db.Create(&storage.Items).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToGoodsReceiptEntity(storage)
	return nil
}

func (r *goodsReceiptRepository) GetGoodsReceipt(ctx context.Context, id string, lock bool) (*domain.GoodsReceiptEntity, error) {
	db := postgres.DB(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var storage GoodsReceiptStorage
	if err := db.Preload("Items").Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToGoodsReceiptEntity(&storage), nil
}

func (r *goodsReceiptRepository) GetGoodsReceipts(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.GoodsReceiptEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&GoodsReceiptStorage{})
	if len(scope) > 0 {
		db = db.Where("warehouse_id IN ?", scope)
	}

	if query.Search != "" {
		db = db.Where("gr_number ILIKE ?", "%"+query.Search+"%")
	}

	for _, key := range []string{"status", "warehouse_id", "purchase_order_id"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []GoodsReceiptStorage
	if err := db.Preload("Items").Order(query.Order(map[string]string{"created_at": "created_at", "gr_number": "gr_number"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.GoodsReceiptEntity, len(rows))
	for i := range rows {
		entities[i] = *ToGoodsReceiptEntity(&rows[i])
	}

	return entities, total, nil
}

func (r *goodsReceiptRepository) Post(ctx context.Context, id, actorID string) error {
	return postgres.DB(ctx, r.db).Exec("UPDATE goods_receipts SET status = 'POSTED', received_at = CURRENT_TIMESTAMP, received_by = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", actorID, id).Error
}
