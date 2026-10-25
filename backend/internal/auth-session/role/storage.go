package role

type RoleStorage struct {
	ID          string `gorm:"type:uuid;primaryKey"`
	Code        string
	Name        string
	Description *string
}

func (RoleStorage) TableName() string {
	return "roles"
}
