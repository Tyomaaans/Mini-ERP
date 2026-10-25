package supplier

import (
	"context"

	"gorm.io/gorm"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

type supplierRepository struct{ db *gorm.DB }

func NewSupplierRepository(db *gorm.DB) domain.SupplierRepository {
	return &supplierRepository{db: db}
}

func (r *supplierRepository) CreateSupplier(ctx context.Context, entity *domain.SupplierEntity) error {
	storage := ToSupplierStorage(entity)
	if err := postgres.DB(ctx, r.db).Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToSupplierEntity(storage)
	return nil
}

func (r *supplierRepository) UpdateSupplier(ctx context.Context, id string, fields map[string]any) error {
	result := postgres.DB(ctx, r.db).Model(&SupplierStorage{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}

func (r *supplierRepository) GetSupplier(ctx context.Context, id string) (*domain.SupplierEntity, error) {
	var storage SupplierStorage
	if err := postgres.DB(ctx, r.db).Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToSupplierEntity(&storage), nil
}

func (r *supplierRepository) GetSuppliers(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.SupplierEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&SupplierStorage{})
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

	var storages []SupplierStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "name": "name", "code": "code"})).Limit(query.PerPage).Offset(query.Offset()).Find(&storages).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.SupplierEntity, len(storages))
	for i := range storages {
		entities[i] = *ToSupplierEntity(&storages[i])
	}

	return entities, total, nil
}
