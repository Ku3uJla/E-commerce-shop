package repository

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect opens a PostgreSQL connection with the given DSN.
func Connect(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// AutoMigrate creates or updates the users table schema.
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&UserModel{})
}
