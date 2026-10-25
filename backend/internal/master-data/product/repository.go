package product

import (
	"context"

	"gorm.io/gorm"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

type productRepository struct{ db *gorm.DB }

func NewProductRepository(db *gorm.DB) domain.ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) CreateProduct(ctx context.Context, entity *domain.ProductEntity) error {
	storage := ToProductStorage(entity)
	if err := postgres.DB(ctx, r.db).Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToProductEntity(storage)
	return nil
}

func (r *productRepository) UpdateProduct(ctx context.Context, id string, fields map[string]any) error {
	result := postgres.DB(ctx, r.db).Model(&ProductStorage{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}

func (r *productRepository) GetProduct(ctx context.Context, id string) (*domain.ProductEntity, error) {
	var storage ProductStorage
	if err := postgres.DB(ctx, r.db).Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToProductEntity(&storage), nil
}

func (r *productRepository) GetProducts(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.ProductEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&ProductStorage{})
	if query.Search != "" {
		db = db.Where("name ILIKE ? OR sku ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%")
	}

	for _, key := range []string{"is_active", "category_id"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var storages []ProductStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "name": "name", "sku": "sku", "selling_price": "selling_price"})).Limit(query.PerPage).Offset(query.Offset()).Find(&storages).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.ProductEntity, len(storages))
	for i := range storages {
		entities[i] = *ToProductEntity(&storages[i])
	}

	return entities, total, nil
}
