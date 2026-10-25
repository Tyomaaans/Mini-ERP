package warehouse

import (
	"context"

	"gorm.io/gorm"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

type warehouseRepository struct{ db *gorm.DB }

func NewWarehouseRepository(db *gorm.DB) domain.WarehouseRepository {
	return &warehouseRepository{db: db}
}

func (r *warehouseRepository) CreateWarehouse(ctx context.Context, entity *domain.WarehouseEntity) error {
	storage := ToWarehouseStorage(entity)
	if err := postgres.DB(ctx, r.db).Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToWarehouseEntity(storage)
	return nil
}

func (r *warehouseRepository) UpdateWarehouse(ctx context.Context, id string, fields map[string]any) error {
	result := postgres.DB(ctx, r.db).Model(&WarehouseStorage{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}

func (r *warehouseRepository) GetWarehouse(ctx context.Context, id string) (*domain.WarehouseEntity, error) {
	var storage WarehouseStorage
	if err := postgres.DB(ctx, r.db).Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToWarehouseEntity(&storage), nil
}

func (r *warehouseRepository) GetWarehouses(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.WarehouseEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&WarehouseStorage{})
	if len(scope) > 0 {
		db = db.Where("id IN ?", scope)
	}

	if query.Search != "" {
		db = db.Where("name ILIKE ? OR code ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%")
	}

	for _, key := range []string{"is_active"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var storages []WarehouseStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "name": "name", "code": "code"})).Limit(query.PerPage).Offset(query.Offset()).Find(&storages).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.WarehouseEntity, len(storages))
	for i := range storages {
		entities[i] = *ToWarehouseEntity(&storages[i])
	}

	return entities, total, nil
}
