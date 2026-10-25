package category

import "time"

type CategoryStorage struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	Code      string
	Name      string
	ParentID  *string `gorm:"type:uuid"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (CategoryStorage) TableName() string {
	return "categories"
}
