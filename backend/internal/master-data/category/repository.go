package category

import (
	"context"

	"gorm.io/gorm"

	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/internal/master-data/domain"
	"mini-erp/pkg"
)

type categoryRepository struct{ db *gorm.DB }

func NewCategoryRepository(db *gorm.DB) domain.CategoryRepository {
	return &categoryRepository{db: db}
}

func (r *categoryRepository) CreateCategory(ctx context.Context, entity *domain.CategoryEntity) error {
	storage := ToCategoryStorage(entity)
	if err := postgres.DB(ctx, r.db).Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = *ToCategoryEntity(storage)
	return nil
}

func (r *categoryRepository) UpdateCategory(ctx context.Context, id string, fields map[string]any) error {
	result := postgres.DB(ctx, r.db).Model(&CategoryStorage{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}

func (r *categoryRepository) GetCategory(ctx context.Context, id string) (*domain.CategoryEntity, error) {
	var storage CategoryStorage
	if err := postgres.DB(ctx, r.db).Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	return ToCategoryEntity(&storage), nil
}

func (r *categoryRepository) GetCategories(ctx context.Context, query pkg.ListQuery, scope []string) ([]domain.CategoryEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&CategoryStorage{})
	if query.Search != "" {
		db = db.Where("name ILIKE ? OR code ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%")
	}

	for _, key := range []string{"parent_id"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var storages []CategoryStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "name": "name", "code": "code"})).Limit(query.PerPage).Offset(query.Offset()).Find(&storages).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.CategoryEntity, len(storages))
	for i := range storages {
		entities[i] = *ToCategoryEntity(&storages[i])
	}

	return entities, total, nil
}

func (r *categoryRepository) ValidateParent(ctx context.Context, id string, parent *string) error {
	db := postgres.DB(ctx, r.db)
	if err := db.Exec("SELECT pg_advisory_xact_lock(73410203)").Error; err != nil {
		return err
	}

	if parent == nil {
		return nil
	}

	if pkg.ValidateID(*parent) != nil || *parent == id {
		return pkg.ErrInvalidInput
	}

	var count int64
	if err := db.Table("categories").Where("id = ?", *parent).Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return pkg.ErrInvalidInput
	}

	var cycle bool
	if err := db.Raw(`WITH RECURSIVE parents AS (SELECT id, parent_id FROM categories WHERE id = ?
		UNION SELECT c.id, c.parent_id FROM categories c JOIN parents p ON c.id = p.parent_id)
		SELECT EXISTS (SELECT 1 FROM parents WHERE id = ?)`, *parent, id).Scan(&cycle).Error; err != nil {
		return err
	}

	if cycle {
		return pkg.NewError("CATEGORY_CYCLE", "Category parent would create a cycle", 409)
	}

	return nil
}

func (r *categoryRepository) DeleteCategory(ctx context.Context, id string) error {
	result := postgres.DB(ctx, r.db).Where("id = ?", id).Delete(&CategoryStorage{})
	if result.Error != nil {
		if postgres.IsForeignKeyViolation(result.Error) {
			return pkg.NewError("RESOURCE_IN_USE", "Referenced categories cannot be deleted", 409)
		}

		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}
