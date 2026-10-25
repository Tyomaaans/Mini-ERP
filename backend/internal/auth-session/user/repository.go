package user

import (
	"context"

	"gorm.io/gorm"

	"mini-erp/internal/auth-session/domain"
	"mini-erp/internal/infrastructure/postgres"
	"mini-erp/pkg"
)

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetUser(ctx context.Context, id string) (*domain.UserEntity, error) {
	var storage UserStorage
	if err := postgres.DB(ctx, r.db).Where("id = ?", id).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	entity := ToUserEntity(storage)
	return &entity, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.UserEntity, error) {
	var storage UserStorage
	if err := postgres.DB(ctx, r.db).Where("LOWER(email) = ?", email).First(&storage).Error; err != nil {
		return nil, postgres.HandleDBError(err)
	}

	entity := ToUserEntity(storage)
	return &entity, nil
}

func (r *userRepository) GetActor(ctx context.Context, id string) (pkg.Actor, error) {
	actor := pkg.Actor{
		ID:           id,
		Roles:        []string{},
		Permissions:  []string{},
		WarehouseIDs: []string{},
	}

	db := postgres.DB(ctx, r.db)
	var profile struct {
		Name       string
		Department string
	}

	if err := db.Raw("SELECT u.full_name AS name, d.name AS department FROM users u JOIN departments d ON d.id = u.department_id WHERE u.id = ?", id).Scan(&profile).Error; err != nil {
		return actor, err
	}

	actor.Name, actor.Department = profile.Name, profile.Department
	if err := db.Raw("SELECT r.code FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = ? ORDER BY r.code", id).Scan(&actor.Roles).Error; err != nil {
		return actor, err
	}

	if err := db.Raw("SELECT DISTINCT p.code FROM permissions p JOIN role_permissions rp ON rp.permission_id = p.id JOIN user_roles ur ON ur.role_id = rp.role_id WHERE ur.user_id = ? ORDER BY p.code", id).Scan(&actor.Permissions).Error; err != nil {
		return actor, err
	}

	if err := db.Raw("SELECT uw.warehouse_id FROM user_warehouses uw JOIN warehouses w ON w.id = uw.warehouse_id AND w.is_active WHERE uw.user_id = ? ORDER BY uw.warehouse_id", id).Scan(&actor.WarehouseIDs).Error; err != nil {
		return actor, err
	}

	return actor, nil
}

func (r *userRepository) GetUsers(ctx context.Context, query pkg.ListQuery) ([]domain.UserEntity, int64, error) {
	db := postgres.DB(ctx, r.db).Model(&UserStorage{})
	if query.Search != "" {
		db = db.Where("full_name ILIKE ? OR email ILIKE ?", "%"+query.Search+"%", "%"+query.Search+"%")
	}

	for _, key := range []string{"department_id", "is_active"} {
		if value := query.Filters[key]; value != "" {
			db = db.Where(key+" = ?", value)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var storages []UserStorage
	if err := db.Order(query.Order(map[string]string{"created_at": "created_at", "full_name": "full_name", "email": "email"})).Limit(query.PerPage).Offset(query.Offset()).Find(&storages).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]domain.UserEntity, len(storages))
	ids := make([]string, len(storages))
	positions := make(map[string]int, len(storages))
	for i := range storages {
		entities[i] = ToUserEntity(storages[i])
		entities[i].Roles = []string{}
		entities[i].WarehouseIDs = []string{}
		ids[i] = storages[i].ID
		positions[ids[i]] = i
	}

	if len(ids) == 0 {
		return entities, total, nil
	}

	var roles []struct {
		UserID string
		Code   string
	}

	if err := postgres.DB(ctx, r.db).Raw(`SELECT ur.user_id, r.code FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id WHERE ur.user_id IN ? ORDER BY ur.user_id, r.code`, ids).Scan(&roles).Error; err != nil {
		return nil, 0, err
	}

	for _, row := range roles {
		i := positions[row.UserID]
		entities[i].Roles = append(entities[i].Roles, row.Code)
	}

	var warehouses []struct {
		UserID      string
		WarehouseID string
	}

	if err := postgres.DB(ctx, r.db).Raw(`SELECT uw.user_id, uw.warehouse_id FROM user_warehouses uw
		JOIN warehouses w ON w.id = uw.warehouse_id AND w.is_active
		WHERE uw.user_id IN ? ORDER BY uw.user_id, uw.warehouse_id`, ids).Scan(&warehouses).Error; err != nil {
		return nil, 0, err
	}

	for _, row := range warehouses {
		i := positions[row.UserID]
		entities[i].WarehouseIDs = append(entities[i].WarehouseIDs, row.WarehouseID)
	}

	return entities, total, nil
}

func (r *userRepository) CreateUser(ctx context.Context, entity *domain.UserEntity) error {
	storage := ToUserStorage(entity)
	if err := postgres.DB(ctx, r.db).Create(storage).Error; err != nil {
		return postgres.HandleDBError(err)
	}

	*entity = ToUserEntity(*storage)
	return nil
}

func (r *userRepository) UpdateUser(ctx context.Context, id string, fields map[string]any) error {
	result := postgres.DB(ctx, r.db).Model(&UserStorage{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return postgres.HandleDBError(result.Error)
	}

	if result.RowsAffected == 0 {
		return pkg.ErrNotFound
	}

	return nil
}

func (r *userRepository) AssignRoles(ctx context.Context, id string, roles []string) error {
	db := postgres.DB(ctx, r.db)
	if err := db.Exec("SELECT pg_advisory_xact_lock(73410202)").Error; err != nil {
		return err
	}

	if err := db.Exec("DELETE FROM user_roles WHERE user_id = ?", id).Error; err != nil {
		return err
	}

	for _, role := range roles {
		if err := db.Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING", id, role).Error; err != nil {
			return postgres.HandleDBError(err)
		}
	}

	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM users u JOIN user_roles ur ON ur.user_id = u.id JOIN roles r ON r.id = ur.role_id WHERE u.is_active AND r.code = 'SUPER_ADMIN'").Scan(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return pkg.NewError("LAST_ADMINISTRATOR", "At least one active administrator is required", 409)
	}

	return nil
}

func (r *userRepository) AssignWarehouses(ctx context.Context, id string, warehouses []string) error {
	db := postgres.DB(ctx, r.db)
	if err := db.Exec("DELETE FROM user_warehouses WHERE user_id = ?", id).Error; err != nil {
		return err
	}

	for _, warehouse := range warehouses {
		if err := db.Exec("INSERT INTO user_warehouses (user_id, warehouse_id) SELECT ?, id FROM warehouses WHERE id = ? AND is_active", id, warehouse).Error; err != nil {
			return postgres.HandleDBError(err)
		}

		var count int64
		if err := db.Table("user_warehouses").Where("user_id = ? AND warehouse_id = ?", id, warehouse).Count(&count).Error; err != nil {
			return err
		}

		if count == 0 {
			return pkg.ErrInvalidInput
		}
	}

	return nil
}

func (r *userRepository) GuardAdministrator(ctx context.Context, id string, removing bool) error {
	db := postgres.DB(ctx, r.db)
	if err := db.Exec("SELECT pg_advisory_xact_lock(73410202)").Error; err != nil {
		return err
	}

	if !removing {
		return nil
	}

	var target, total int64
	if err := db.Raw("SELECT COUNT(*) FROM users u JOIN user_roles ur ON ur.user_id = u.id JOIN roles r ON r.id = ur.role_id WHERE u.is_active AND r.code = 'SUPER_ADMIN' AND u.id = ?", id).Scan(&target).Error; err != nil {
		return err
	}

	if target == 0 {
		return nil
	}

	if err := db.Raw("SELECT COUNT(*) FROM users u JOIN user_roles ur ON ur.user_id = u.id JOIN roles r ON r.id = ur.role_id WHERE u.is_active AND r.code = 'SUPER_ADMIN'").Scan(&total).Error; err != nil {
		return err
	}

	if total <= 1 {
		return pkg.NewError("LAST_ADMINISTRATOR", "At least one active administrator is required", 409)
	}

	return nil
}

func (r *userRepository) LoginAt(ctx context.Context, id string) error {
	return postgres.DB(ctx, r.db).Exec("UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?", id).Error
}
