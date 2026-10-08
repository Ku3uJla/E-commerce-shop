package repository

import (
	"time"

	"github.com/google/uuid"
)

// ProductModel is the GORM persistence model for the products table.
type ProductModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Name        string     `gorm:"size:150;not null"`
	Description string     `gorm:"type:text"`
	Price       int64      `gorm:"not null"`
	Stock       int        `gorm:"not null;default:0"`
	CreatedAt   time.Time  `gorm:"not null"`
	UpdatedAt   *time.Time `gorm:"autoUpdateTime"`
}

// TableName pins the persistence table name to `products`.
func (ProductModel) TableName() string {
	return "products"
}
