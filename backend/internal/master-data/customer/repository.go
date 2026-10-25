package customer

import (
	"context"

	"gorm.io/gorm"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

type customerRepository struct{ db *gorm.DB }

func NewCustomerRepository(db *gorm.DB) domain.CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) CreateCustomer(ctx context.Context, entity *domain.CustomerEntity) error {
	storage := ToCustomerStorage(entity)
	if err := postgres.DB(ctx, r.db).Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToCustomerEntity(storage)
	return nil
}

func (r *customerRepository) UpdateCustomer(ctx context.Context, id string, fields map[string]any) error {
	result := postgres.DB(ctx, r.db).Model(&CustomerStorage{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}

func (r *customerRepository) GetCustomer(ctx context.Context, id string) (*domain.CustomerEntity, error) {
	var storage CustomerStorage
	if err := postgres.DB(ctx, r.db).Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToCustomerEntity(&storage), nil
}

func (r *customerRepository) GetCustomers(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.CustomerEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&CustomerStorage{})
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

	var storages []CustomerStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "name": "name", "code": "code"})).Limit(query.PerPage).Offset(query.Offset()).Find(&storages).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.CustomerEntity, len(storages))
	for i := range storages {
		entities[i] = *ToCustomerEntity(&storages[i])
	}

	return entities, total, nil
}
