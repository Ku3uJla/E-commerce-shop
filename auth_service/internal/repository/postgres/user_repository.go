package repository

import (
	"context"
	"errors"
	"os"

	"gorm.io/gorm"

	"auth_service/internal/domain/user"
)

type userRepository struct {
	db    *gorm.DB
	table string
}

// NewUserRepo builds a user repository backed by GORM.
func NewUserRepo(db *gorm.DB) *userRepository {
	table := os.Getenv("USERS_TABLE")
	if table == "" {
		table = "users"
	}
	return &userRepository{db: db, table: table}
}

func (r *userRepository) Create(ctx context.Context, u *user.User) error {
	return r.db.WithContext(ctx).Table(r.table).Create(toModel(u)).Error
}

func (r *userRepository) GetByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	var m UserModel
	err := r.db.WithContext(ctx).Table(r.table).Where("email = ?", email.String()).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}

func (r *userRepository) GetByUID(ctx context.Context, uid user.UserID) (*user.User, error) {
	var m UserModel
	err := r.db.WithContext(ctx).Table(r.table).Where("id = ?", uid.String()).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toDomain(&m), nil
}
