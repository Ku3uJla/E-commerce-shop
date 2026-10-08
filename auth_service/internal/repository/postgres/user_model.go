package repository

import (
	"time"

	"github.com/google/uuid"
)

// UserModel is the GORM persistence model for the users table.
type UserModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Name      string     `gorm:"size:100;not null"`
	Email     string     `gorm:"size:100;not null;uniqueIndex"`
	Password  string     `gorm:"size:100;not null"`
	Role      string     `gorm:"size:20;not null;default:user"`
	CreatedAt time.Time  `gorm:"not null"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime"`
}

// TableName pins the persistence table name to `users`.
func (UserModel) TableName() string {
	return "users"
}
