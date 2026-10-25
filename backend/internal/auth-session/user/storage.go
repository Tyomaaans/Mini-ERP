package user

import "time"

type UserStorage struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	DepartmentID string `gorm:"type:uuid"`
	FullName     string
	Email        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (UserStorage) TableName() string {
	return "users"
}
