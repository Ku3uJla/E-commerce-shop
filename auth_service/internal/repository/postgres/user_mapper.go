package repository

import (
	"github.com/google/uuid"

	"auth_service/internal/domain/user"
)

// toModel converts a domain user into its persistence representation.
func toModel(u *user.User) *UserModel {
	return &UserModel{
		ID:        uuid.MustParse(u.ID().String()),
		Name:      u.Name().String(),
		Email:     u.Email().String(),
		Password:  u.Password().String(),
		Role:      "user",
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
	}
}

// toDomain rehydrates a domain user from its persistence representation.
func toDomain(m *UserModel) *user.User {
	id, err := user.NewUserID(m.ID.String())
	if err != nil {
		id = user.GenerateUserID()
	}
	name, err := user.NewName(m.Name)
	if err != nil {
		name, _ = user.NewName(m.Email)
	}
	email, err := user.NewEmail(m.Email)
	if err != nil {
		return nil
	}
	password, err := user.NewHashedPassword(m.Password)
	if err != nil {
		return nil
	}
	return user.Reconstitute(id, name, email, password, m.CreatedAt, m.UpdatedAt)
}
