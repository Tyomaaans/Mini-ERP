package supplier

import "time"

type SupplierStorage struct {
	ID               string `gorm:"type:uuid;primaryKey"`
	Code             string
	Name             string
	Email            string
	Phone            string
	Address          string
	PaymentTermsDays int
	IsActive         bool `gorm:"not null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (SupplierStorage) TableName() string {
	return "suppliers"
}
