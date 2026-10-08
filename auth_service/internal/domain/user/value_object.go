package user

import (
	"errors"
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

type UserID struct {
	value string
}

func GenerateUserID() UserID {
	return UserID{value: uuid.NewString()}
}

func NewUserID(value string) (UserID, error) {
	if _, err := uuid.Parse(value); err != nil {
		return UserID{}, err
	}
	return UserID{value: value}, nil
}

func (id UserID) String() string {
	return id.value
}

type Name struct {
	value string
}

func NewName(value string) (Name, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Name{}, errors.New("name must not be empty")
	}
	return Name{value: value}, nil
}

func (name Name) String() string {
	return name.value
}

type Email struct {
	value string
}

func NewEmail(value string) (Email, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	addr, err := mail.ParseAddress(value)
	if err != nil {
		return Email{}, err
	}
	return Email{value: addr.Address}, nil
}

func (email Email) String() string {
	return email.value
}

type HashedPassword struct {
	value string
}

func NewHashedPassword(pass string) (HashedPassword, error) {
	if pass == "" {
		return HashedPassword{}, errors.New("password hash is empty")
	}
	return HashedPassword{value: pass}, nil
}

func (hash HashedPassword) String() string {
	return hash.value
}
