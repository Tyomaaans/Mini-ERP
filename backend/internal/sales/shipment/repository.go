package shipment

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/sales/domain"
	"mini-erp/pkg"
)

type shipmentRepository struct{ db *gorm.DB }

func NewShipmentRepository(db *gorm.DB) domain.ShipmentRepository {
	return &shipmentRepository{db: db}
}

func (r *shipmentRepository) CreateShipment(ctx context.Context, entity *domain.ShipmentEntity) error {
	db := postgres.DB(ctx, r.db)
	storage := ToShipmentStorage(entity)
	if err := db.Omit("Items").Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	if err := db.Create(&storage.Items).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToShipmentEntity(storage)
	return nil
}

func (r *shipmentRepository) GetShipment(ctx context.Context, id string, lock bool) (*domain.ShipmentEntity, error) {
	db := postgres.DB(ctx, r.db)
	if lock {
		db = db.Clauses(clause.Locking{Strength: "UPDATE"})
	}

	var storage ShipmentStorage
	if err := db.Preload("Items").Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToShipmentEntity(&storage), nil
}

func (r *shipmentRepository) GetShipments(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.ShipmentEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&ShipmentStorage{})
	if len(scope) > 0 {
		db = db.Where("warehouse_id IN ?", scope)
	}

	if query.Search != "" {
		db = db.Where("shipment_number ILIKE ?", "%"+query.Search+"%")
	}

	for _, key := range []string{"status", "warehouse_id", "sales_order_id"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []ShipmentStorage
	if err := db.Preload("Items").Order(query.Order(map[string]string{"created_at": "created_at", "shipment_number": "shipment_number"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.ShipmentEntity, len(rows))
	for i := range rows {
		entities[i] = *ToShipmentEntity(&rows[i])
	}

	return entities, total, nil
}

func (r *shipmentRepository) Post(ctx context.Context, id, actorID string) error {
	return postgres.DB(ctx, r.db).Exec("UPDATE shipments SET status = 'SHIPPED', shipped_at = CURRENT_TIMESTAMP, shipped_by = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", actorID, id).Error
}
