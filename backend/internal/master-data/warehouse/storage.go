package warehouse

import "time"

type WarehouseStorage struct {
	ID        string `gorm:"type:uuid;primaryKey"`
	Code      string
	Name      string
	Address   string
	IsActive  bool `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (WarehouseStorage) TableName() string {
	return "warehouses"
}
