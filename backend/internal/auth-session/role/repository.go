package role

import (
	"context"

	"gorm.io/gorm"

	"mini-erp/internal/auth-session/domain"
	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/pkg"
)

type roleRepository struct{ db *gorm.DB }

func NewRoleRepository(db *gorm.DB) domain.RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) GetRole(ctx context.Context, id string) (*domain.RoleEntity, error) {
	var row RoleStorage
	db := postgres.DB(ctx, r.db)
	if err := db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	entity := &domain.RoleEntity{
		ID:            row.ID,
		Code:          row.Code,
		Name:          row.Name,
		Description:   row.Description,
		PermissionIDs: []string{},
	}

	if err := db.Raw("SELECT permission_id FROM role_permissions WHERE role_id = ? ORDER BY permission_id", id).Scan(&entity.PermissionIDs).Error; err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *roleRepository) GetRoles(ctx context.Context, query pkg.ListQuery) ([]domain.RoleEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&RoleStorage{})
	if query.Search != "" {
		db = db.Where("name ILIKE ? OR code ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []RoleStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "code": "code", "name": "name"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.RoleEntity, len(rows))
	ids := make([]string, len(rows))
	positions := make(map[string]int, len(rows))
	for i, row := range rows {
		entities[i] = domain.RoleEntity{
			ID:            row.ID,
			Code:          row.Code,
			Name:          row.Name,
			Description:   row.Description,
			PermissionIDs: []string{},
		}

		ids[i] = row.ID
		positions[row.ID] = i
	}

	if len(ids) == 0 {
		return entities, total, nil
	}

	var permissions []struct {
		RoleID       string
		PermissionID string
	}

	if err := postgres.DB(ctx, r.db).Raw(`SELECT role_id, permission_id FROM role_permissions
		WHERE role_id IN ? ORDER BY role_id, permission_id`, ids).Scan(&permissions).Error; err != nil {
		return nil, 0, err
	}

	for _, row := range permissions {
		i := positions[row.RoleID]
		entities[i].PermissionIDs = append(entities[i].PermissionIDs, row.PermissionID)
	}

	return entities, total, nil
}

func (r *roleRepository) SaveRole(ctx context.Context, e *domain.RoleEntity, create bool) error {
	db := postgres.DB(ctx, r.db)
	if err := db.Exec("SELECT pg_advisory_xact_lock(73410202)").Error; err != nil {
		return err
	}

	if create {
		if err := db.Create(&RoleStorage{
			ID:          e.ID,
			Code:        e.Code,
			Name:        e.Name,
			Description: e.Description,
		}).Error; err != nil {
			return postgres.HandleDBError(err)
		}

	} else {
		if err := db.Model(&RoleStorage{}).Where("id = ?", e.ID).Updates(map[string]any{"name": e.Name, "description": e.Description, "code": e.Code}).Error; err != nil {
			return postgres.HandleDBError(err)
		}
	}

	if err := db.Exec("DELETE FROM role_permissions WHERE role_id = ?", e.ID).Error; err != nil {
		return err
	}

	for _, permission := range e.PermissionIDs {
		if err := db.Exec("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING", e.ID, permission).Error; err != nil {
			return postgres.HandleDBError(err)
		}
	}

	return nil
}

func (r *roleRepository) DeleteRole(ctx context.Context, id string) error {
	db := postgres.DB(ctx, r.db)
	if err := db.Exec("SELECT pg_advisory_xact_lock(73410202)").Error; err != nil {
		return err
	}

	var count int64
	if err := db.Table("user_roles").Where("role_id = ?", id).Count(&count).Error; err != nil {
		return err
	}

	if count > 0 {
		return pkg.NewError("ROLE_IN_USE", "Assigned roles cannot be deleted", 409)
	}

	return postgres.HandleDBError(db.Where("id = ?", id).Delete(&RoleStorage{}).Error)
}

func (r *roleRepository) Lookups(ctx context.Context, resource string, query pkg.ListQuery) ([]domain.LookupEntity, int64, error) {
	if resource != "departments" && resource != "permissions" {
		return nil, 0, pkg.ErrNotFound
	}

	db := postgres.DB(ctx, r.db).Table(resource)
	if query.Search != "" {
		db = db.Where("code ILIKE ?", "%"+query.Search+"%")
	}

	if resource == "permissions" {
		if value := query.Filters["module"]; value != "" {
			db = db.Where("module = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	rows := []domain.LookupEntity{}
	if err := db.Order(query.Order(map[string]string{"code": "code", "created_at": "created_at"})).Limit(query.PerPage).Offset(query.Offset()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}
